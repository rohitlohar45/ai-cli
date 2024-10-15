package main

import (
	"ai-cli/config"
	"ai-cli/internal/history"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func getOllamaStatus() (bool, error) {
	cmd := exec.Command("docker", "ps", "--filter", "publish=11434", "--format", "{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check docker status: %v", err)
	}

	if strings.TrimSpace(string(output)) == "" {
		return false, nil
	}

	return true, nil
}

func main() {
	config.InitConfig()
	if len(os.Args) < 2 {
		fmt.Println("Usage: ai-cli <subcommand> [options] or ai-cli \"<input command>\"")
		os.Exit(1)
	}

	var modelError error

	if config.ApiKey == "" {
		status, err := getOllamaStatus()

		if err != nil {
			log.Fatalf("Error checking Ollama status: %v", err)
			modelError = fmt.Errorf("Error checking Ollama status: %v", err)
		}

		if status {
			config.UseOllama = true
		} else {
			modelError = fmt.Errorf("OpenAI API key is not set and Ollama is not running on the port 11434")
		}
	}
	switch os.Args[1] {
	case "set-api-key":
		handleSetAPIKey(os.Args[2:])
	case "set-model":
		handleSetModel(os.Args[2:])
	case "set-temperature":
		handleSetTemperature(os.Args[2:])
	case "set-max-tokens":
		handleSetMaxTokens(os.Args[2:])
	case "set-ollama":
		handleSetOllama(os.Args[2:])
	case "show-history":
		history.ShowHistory()
	case "clear-history":
		history.ClearHistory()
	case "help":
		printHelp()
	default:

		if modelError != nil {
			log.Fatal(modelError)
		}

		input := strings.Join(os.Args[1:], " ")
		command, err := processInput(input)
		if err != nil {
			log.Fatalf("Error processing input: %v", err)
		}
		fmt.Printf("Generated command: %s\n", command)
		handleUserOptions(command)
	}
}
