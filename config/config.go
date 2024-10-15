// config/config.go
package config

import (
	"os"
	"runtime"
)

var (
	ApiKey        string
	Model         string
	Temperature   float64
	MaxTokens     int
	OllamaBaseURL string
	OsType        string
	UseOllama     bool
)

func InitConfig() {
	ApiKey = os.Getenv("OPENAI_API_KEY")
	Model = "codellama"
	Temperature = 0.7
	MaxTokens = 1000
	OllamaBaseURL = "http://localhost:11434"
	OsType = runtime.GOOS
	UseOllama = false
}
