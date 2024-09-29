package history

import (
	"log"
	"os"
	"strings"
)

const historyFile = "history.log"

var historyLog *log.Logger

func init() {
	file, err := os.OpenFile(historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	historyLog = log.New(file, "cli-history: ", log.LstdFlags)
}

// ShowHistory displays the command history to the console.
func ShowHistory() {
	data, err := os.ReadFile(historyFile)
	if err != nil {
		log.Fatal(err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.Split(line, "|")
			if len(parts) == 2 {
				log.Printf("Command: %s | Output: %s\n", parts[0], parts[1])
			}
		}
	}
}

// ClearHistory deletes the history file to clear all recorded commands.
func ClearHistory() {
	if err := os.Remove(historyFile); err != nil {
		log.Printf("Error clearing history: %v\n", err)
	} else {
		log.Println("History cleared successfully.")
	}
}

// LogCommand logs a command and its output to the history file.
func LogCommand(command, output string) {
	historyLog.Printf("%s|%s\n", command, output)
}
