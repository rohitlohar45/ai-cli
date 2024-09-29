package main

import (
	"ai-cli/internal/history"
	internal "ai-cli/internal/spinner"
	"ai-cli/utils"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

func processInput(input string) (string, error) {
	llmPrompt := fmt.Sprintf(`You are a helpful assistant that converts natural language instructions into command line instructions.
1. Your output should only include the command line instruction, nothing else.
2. Provide the command in the correct format for %s, ensuring it is syntactically correct and executable.
3. Include specific arguments or options as necessary.
4. Do not include unnecessary explanations or additional text.
5. Task: "%s"
Return the command without any extra formatting.`, osType, input)

	internal.StartSpinner()
	defer internal.StopSpinner()

	if useOllama {
		return sendOllamaRequest(llmPrompt)
	}
	return sendOpenAIRequest(llmPrompt)
}

func sendOllamaRequest(prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", ollamaBaseURL)
	data := map[string]string{"model": model, "prompt": prompt}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var fullResponse string
	decoder := json.NewDecoder(resp.Body)
	for {
		var ollamaResp utils.OllamaResponse
		if err := decoder.Decode(&ollamaResp); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to decode response: %w", err)
		}
		fullResponse += ollamaResp.Response
		if ollamaResp.Done {
			break
		}
	}

	return strings.TrimSpace(fullResponse), nil
}

func sendOpenAIRequest(input string) (string, error) {
	client := openai.NewClient(apiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       model,
			Temperature: float32(temperature),
			MaxTokens:   maxTokens,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: input},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("OpenAI error: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

func handleUserOptions(command string) {
	fmt.Println("Do you want to:")
	fmt.Println("1. Execute the command")
	fmt.Println("2. Revise the command")
	fmt.Println("3. Cancel")
	fmt.Print("Enter your choice (1/2/3): ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		choice := scanner.Text()
		switch choice {
		case "1":
			executeCommand(command)
			return
		case "2":
			revisedCommand, err := reviseCommand(command)
			if err != nil {
				fmt.Printf("Error revising command: %v\n", err)
				return
			}
			fmt.Printf("Revised command: %s\n", revisedCommand)
			handleUserOptions(revisedCommand)
			return
		case "3":
			fmt.Println("Operation cancelled.")
			return
		default:
			fmt.Print("Invalid choice. Please enter 1, 2, or 3: ")
		}
	}
}

func reviseCommand(originalCommand string) (string, error) {
	revisionPrompt := fmt.Sprintf(`The following command was generated, but the user wants a revision:
Original command: %s
Please provide a revised, possibly simpler or more efficient version of this command.
Only return the command, nothing else.`, originalCommand)

	return processInput(revisionPrompt)
}

func executeCommand(command string) {
	command = strings.Trim(command, "`'\"")

	if strings.Contains(command, "~/") {
		usr, err := user.Current()
		if err == nil {
			command = strings.ReplaceAll(command, "~/", usr.HomeDir+"/")
		}
	}

	fmt.Printf("Executing command: %s\n", command)
	internal.StartSpinner()
	defer internal.StopSpinner()
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		fmt.Println("No command to execute.")
		return
	}

	var expandedParts []string
	for _, part := range cmdParts {
		if strings.Contains(part, "*") {
			matches, err := filepath.Glob(part)
			if err == nil && len(matches) > 0 {
				expandedParts = append(expandedParts, matches...)
			} else {
				expandedParts = append(expandedParts, part)
			}
		} else {
			expandedParts = append(expandedParts, part)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, expandedParts[0], expandedParts[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing command: %v\n", err)
		fmt.Printf("Command output: %s\n", out.String())
		return
	}

	fmt.Printf("Command output:\n%s\n", out.String())
	history.LogCommand(command, out.String())
}

func handleSetAPIKey(args []string) {
	fs := flag.NewFlagSet("set-api-key", flag.ExitOnError)
	var newAPIKey string
	fs.StringVar(&newAPIKey, "key", "", "Set OpenAI API Key")
	fs.Parse(args)
	if newAPIKey != "" {
		apiKey = newAPIKey
		useOllama = false
		fmt.Printf("API Key set. Using OpenAI model: %s\n", model)
	} else {
		fmt.Println("Please provide a valid API key.")
	}
}

func handleSetModel(args []string) {
	fs := flag.NewFlagSet("set-model", flag.ExitOnError)
	var newModel string
	fs.StringVar(&newModel, "model", "", "Set model (gpt-3.5-turbo, gpt-4, codellama)")
	fs.Parse(args)
	if newModel != "" {
		model = newModel
		useOllama = (newModel == "codellama")
		fmt.Printf("Model set to: %s\n", model)
		if !useOllama && apiKey == "" {
			fmt.Println("Warning: API Key is required for OpenAI models. Use 'set-api-key' command to set it.")
		}
	} else {
		fmt.Println("Please provide a valid model name.")
	}
}

func handleSetTemperature(args []string) {
	fs := flag.NewFlagSet("set-temperature", flag.ExitOnError)
	fs.Float64Var(&temperature, "temp", 0.7, "Set temperature (between 0.0 and 1.0)")
	fs.Parse(args)
	fmt.Printf("Temperature set to: %f\n", temperature)
}

func handleSetMaxTokens(args []string) {
	fs := flag.NewFlagSet("set-max-tokens", flag.ExitOnError)
	fs.IntVar(&maxTokens, "max-tokens", 1000, "Set maximum number of tokens")
	fs.Parse(args)
	fmt.Printf("Max tokens set to: %d\n", maxTokens)
}

func handleSetOllama(args []string) {
	fs := flag.NewFlagSet("set-ollama", flag.ExitOnError)
	var newOllamaURL string
	fs.StringVar(&newOllamaURL, "url", "", "Set Ollama base URL")
	fs.Parse(args)
	if newOllamaURL != "" {
		ollamaBaseURL = newOllamaURL
		useOllama = true
		fmt.Printf("Using Ollama URL: %s\n", ollamaBaseURL)
	} else {
		fmt.Println("Please provide a valid Ollama URL.")
	}
}

func printHelp() {
	fmt.Println("Usage: ai-cli <subcommand> [options] or ai-cli \"<input command>\"")
	fmt.Println("\nSubcommands:")
	fmt.Println("  set-api-key -key <API Key>: Set your OpenAI API key")
	fmt.Println("  set-model -model <model>: Set the model (gpt-3.5-turbo, gpt-4, codellama)")
	fmt.Println("  set-temperature -temp <value>: Set temperature (0.0 - 1.0)")
	fmt.Println("  set-max-tokens -max-tokens <number>: Set maximum number of tokens")
	fmt.Println("  set-ollama -url <Ollama URL>: Set Ollama base URL")
	fmt.Println("  show-history: Show command history")
	fmt.Println("  clear-history: Clear command history")
	fmt.Println("  help: Show this help message")
	fmt.Println("\nFor natural language commands, simply type: ai-cli \"your command here\"")
}
