package main

import (
	"ai-cli/internal/history"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

var (
	apiKey        string
	model         string
	temperature   float64
	maxTokens     int
	ollamaBaseURL string
	osType        string
	useOllama     bool
)

func init() {
	apiKey = os.Getenv("OPENAI_API_KEY")
	model = "codellama"
	temperature = 0.7
	maxTokens = 1000
	ollamaBaseURL = "http://localhost:11434"
	osType = runtime.GOOS
	useOllama = true
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ai-cli <subcommand> [options] or ai-cli \"<input command>\"")
		os.Exit(1)
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
		input := strings.Join(os.Args[1:], " ")
		command, err := processInput(input)
		if err != nil {
			log.Fatalf("Error processing input: %v", err)
		}
		fmt.Printf("Generated command: %s\n", command)
		handleUserOptions(command)
	}
}
