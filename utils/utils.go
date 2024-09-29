package utils

import (
	internal "ai-cli/internal/spinner"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

type OllamaResponse struct {
	Model      string    `json:"model"`
	CreatedAt  time.Time `json:"created_at"`
	Response   string    `json:"response"`
	Done       bool      `json:"done"`
	DoneReason string    `json:"done_reason,omitempty"`
}

var (
	historyLog *log.Logger
)

func CheckError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func SendOllamaRequest(prompt, baseURL, model string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", baseURL)
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
		var ollamaResp OllamaResponse
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

// SendOpenAIRequest sends a request to the OpenAI API.
func SendOpenAIRequest(input, apiKey, model string, temperature float64, maxTokens int) (string, error) {
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
	historyLog.Printf("Executed: %s | Output: %s", command, out.String())
}
