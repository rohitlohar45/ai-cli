package main

import (
	"ai-cli/config"
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
	internal.StartSpinner()
	defer internal.StopSpinner()

	return utils.FetchLLMResponse(input, utils.AIModel(config.Model))

	// if useOllama {
	// 	return sendOllamaRequest(llmPrompt)
	// }
	// return sendOpenAIRequest(llmPrompt)
}

func SendOllamaRequest(prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", config.OllamaBaseURL)
	data := map[string]string{"model": config.Model, "prompt": prompt}
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

func SendOpenAIRequest(input string) (string, error) {
	client := openai.NewClient(config.ApiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       config.Model,
			Temperature: float32(config.Temperature),
			MaxTokens:   config.MaxTokens,
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

	fmt.Println("Setting API Key...", args)

	if newAPIKey != "" {
		config.ApiKey = newAPIKey
		os.Setenv("OPENAI_API_KEY", newAPIKey)

		err := persistAPIKey(newAPIKey)
		if err != nil {
			fmt.Printf("Failed to save API Key persistently: %v\n", err)
			return
		}

		fmt.Println("API Key has been set and saved persistently.")
	} else {
		fmt.Println("Please provide a valid API key. in the format: ai-cli set-api-key --key <API_KEY>")
	}
}

func persistAPIKey(apiKey string) error {
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to get current user: %v", err)
	}

	shell := os.Getenv("SHELL")
	var profileFile string
	if strings.Contains(shell, "bash") {
		profileFile = ".bashrc"
	} else if strings.Contains(shell, "zsh") {
		profileFile = ".zshrc"
	} else {
		profileFile = ".profile"
	}

	profilePath := filepath.Join(usr.HomeDir, profileFile)

	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open shell profile: %v", err)
	}
	defer f.Close()

	exportCommand := fmt.Sprintf("\nexport OPENAI_API_KEY=\"%s\"\n", apiKey)
	if _, err = f.WriteString(exportCommand); err != nil {
		return fmt.Errorf("unable to write API key to profile: %v", err)
	}

	return nil
}

func handleSetModel(args []string) {
	fs := flag.NewFlagSet("set-model", flag.ExitOnError)
	var newModel string
	fs.StringVar(&newModel, "model", "", "Set model (gpt-3.5-turbo, gpt-4, codellama)")
	fs.Parse(args)
	if newModel != "" {
		config.Model = newModel
		config.UseOllama = (newModel == "codellama")
		fmt.Printf("Model set to: %s\n", config.Model)
		if !config.UseOllama && config.ApiKey == "" {
			fmt.Println("Warning: API Key is required for OpenAI models. Use 'set-api-key' command to set it.")
		}
	} else {
		fmt.Println("Please provide a valid model name.")
	}
}

func handleSetTemperature(args []string) {
	fs := flag.NewFlagSet("set-temperature", flag.ExitOnError)
	fs.Float64Var(&config.Temperature, "temp", 0.7, "Set temperature (between 0.0 and 1.0)")
	fs.Parse(args)
	fmt.Printf("Temperature set to: %f\n", config.Temperature)
}

func handleSetMaxTokens(args []string) {
	fs := flag.NewFlagSet("set-max-tokens", flag.ExitOnError)
	fs.IntVar(&config.MaxTokens, "max-tokens", 1000, "Set maximum number of tokens")
	fs.Parse(args)
	fmt.Printf("Max tokens set to: %d\n", config.MaxTokens)
}

func handleSetOllama(args []string) {
	fs := flag.NewFlagSet("set-ollama", flag.ExitOnError)
	var newOllamaURL string
	fs.StringVar(&newOllamaURL, "url", "", "Set Ollama base URL")
	fs.Parse(args)
	if newOllamaURL != "" {
		config.OllamaBaseURL = newOllamaURL
		config.UseOllama = true
		fmt.Printf("Using Ollama URL: %s\n", config.OllamaBaseURL)
	} else {
		fmt.Println("Please provide a valid Ollama URL.")
	}
}

func printHelp() {
	fmt.Println("Usage: ai-cli <subcommand> [options] or ai-cli \"<input command>\"")
	fmt.Println("\nSubcommands:")
	fmt.Println("  set-api-key --key <API Key>: Set your OpenAI API key")
	fmt.Println("  set-model --model <model>: Set the model (gpt-3.5-turbo, gpt-4, codellama)")
	fmt.Println("  set-temperature --temp <value>: Set temperature (0.0 - 1.0)")
	fmt.Println("  set-max-tokens --max-tokens <number>: Set maximum number of tokens")
	fmt.Println("  set-ollama --url <Ollama URL>: Set Ollama base URL")
	fmt.Println("  show-history: Show command history")
	fmt.Println("  clear-history: Clear command history")
	fmt.Println("  help: Show this help message")
	fmt.Println("\nFor natural language commands, simply type: ai-cli \"your command here\"")
}
