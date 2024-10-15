package utils

import (
	"ai-cli/config"
	"bytes"
	"container/list"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/agnivade/levenshtein"
	"github.com/bbalet/stopwords"
	"github.com/clipperhouse/jargon"
	"github.com/clipperhouse/jargon/filters/synonyms"
	"github.com/sashabaranov/go-openai"
)

const (
	NumClusters         = 8
	cacheTTL            = 1 * time.Hour
	cacheSizeLimit      = 5000
	similarityThreshold = 0.8
)

type OllamaResponse struct {
	Model      string    `json:"model"`
	CreatedAt  time.Time `json:"created_at"`
	Response   string    `json:"response"`
	Done       bool      `json:"done"`
	DoneReason string    `json:"done_reason,omitempty"`
}

type CacheEntry struct {
	Key      string
	Response string
	Expiry   time.Time
}

type Cluster struct {
	cache     sync.Map
	evictList *list.List
	mutex     sync.RWMutex
	size      int
}

type DistributedCache struct {
	clusters [NumClusters]*Cluster
	// mutex    sync.RWMutex
}

type CommandEntry struct {
	Command     string
	Description string
}

func getPrompt(ostype string, input string) string {
	return fmt.Sprintf(`You are a helpful assistant that converts natural language instructions into command line instructions. 1. Your output should only include the command line instruction, nothing else. 2. Provide the command in the correct format for %s, ensuring it is syntactically correct and executable. 3. Include specific arguments or options as necessary. 4. Do not include unnecessary explanations or additional text. 5. Task: "%s" Return the command without any extra formatting.`, ostype, input)
}

type CommandDataset struct {
	Commands map[string]CommandEntry
}

type AIModel string

const (
	Ollama    AIModel = "ollama"
	OpenAI    AIModel = "openai"
	Anthropic AIModel = "anthropic"
)

var (
	globalCache    *DistributedCache
	commandDataset *CommandDataset
)

func init() {
	globalCache = NewDistributedCache()
	globalCache.StartCleanup(5 * time.Minute)
	commandDataset = loadCommandDatasets()
}

func NewCluster() *Cluster {
	return &Cluster{
		cache:     sync.Map{},
		evictList: list.New(),
	}
}

func NewDistributedCache() *DistributedCache {
	dc := &DistributedCache{}
	for i := 0; i < NumClusters; i++ {
		dc.clusters[i] = NewCluster()
	}
	return dc
}

func (dc *DistributedCache) getCluster(input string) *Cluster {
	h := fnv.New32a()
	h.Write([]byte(input))
	return dc.clusters[h.Sum32()%NumClusters]
}

func (c *Cluster) Set(key string, value CacheEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if ee, ok := c.cache.Load(key); ok {
		c.evictList.MoveToFront(ee.(*list.Element))
		ee.(*list.Element).Value.(*CacheEntry).Response = value.Response
		ee.(*list.Element).Value.(*CacheEntry).Expiry = value.Expiry
	} else {
		ele := c.evictList.PushFront(&CacheEntry{Key: key, Response: value.Response, Expiry: value.Expiry})
		c.cache.Store(key, ele)
		c.size++
	}

	if c.size > cacheSizeLimit {
		c.evict()
	}
}

func (c *Cluster) Get(key string) (CacheEntry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if ele, ok := c.cache.Load(key); ok {
		entry := ele.(*list.Element).Value.(*CacheEntry)
		if time.Now().Before(entry.Expiry) {
			c.evictList.MoveToFront(ele.(*list.Element))
			return *entry, true
		}
		// Remove expired entry
		c.evictList.Remove(ele.(*list.Element))
		c.cache.Delete(key)
		c.size--
	}
	return CacheEntry{}, false
}

func (c *Cluster) evict() {
	ele := c.evictList.Back()
	if ele != nil {
		c.evictList.Remove(ele)
		kv := ele.Value.(*CacheEntry)
		c.cache.Delete(kv.Key)
		c.size--
	}
}

func (c *Cluster) RemoveExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	var next *list.Element
	for e := c.evictList.Back(); e != nil; e = next {
		next = e.Prev()
		kv := e.Value.(*CacheEntry)
		if now.After(kv.Expiry) {
			c.evictList.Remove(e)
			c.cache.Delete(kv.Key)
			c.size--
		}
	}
}

func (dc *DistributedCache) Set(input, response string) {
	normalizedInput := normalizeInput(input)
	cluster := dc.getCluster(normalizedInput)
	cluster.Set(normalizedInput, CacheEntry{
		Key:      normalizedInput,
		Response: response,
		Expiry:   time.Now().Add(cacheTTL),
	})
}

func (dc *DistributedCache) Get(input string) (string, bool) {
	normalizedInput := normalizeInput(input)
	cluster := dc.getCluster(normalizedInput)

	if entry, ok := cluster.Get(normalizedInput); ok {
		return entry.Response, true
	}

	// Fuzzy matching
	var bestMatch string
	var bestSimilarity float64
	cluster.cache.Range(func(key, value interface{}) bool {
		similarity := calculateSimilarity(normalizedInput, key.(string))
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = key.(string)
		}
		return true
	})

	if bestSimilarity > similarityThreshold {
		if entry, ok := cluster.Get(bestMatch); ok {
			return entry.Response, true
		}
	}

	return "", false
}

func (dc *DistributedCache) RemoveExpired() {
	for _, cluster := range dc.clusters {
		cluster.RemoveExpired()
	}
}

func (dc *DistributedCache) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			dc.RemoveExpired()
		}
	}()
}

// func normalizeInput(input string) string {
// 	cleanedInput := stopwords.CleanString(input, "en", true)
// 	words := strings.Fields(cleanedInput)
// 	normalizedWords := make([]string, 0, len(words))

// 	for _, word := range words {
// 		synonym, err := nlp.GetSynonyms(word)
// 		if err == nil && synonym != "" {
// 			word = synonym
// 		}
// 		normalizedWords = append(normalizedWords, word)
// 	}

// 	sort.Strings(normalizedWords)
// 	return strings.Join(normalizedWords, " ")
// }

func normalizeInput(input string) string {
	// Convert to lowercase
	input = strings.ToLower(input)

	// Remove punctuation
	input = strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, input)

	// Remove stopwords
	cleanedInput := stopwords.CleanString(input, "en", true)
	fmt.Println("Cleaned input:", cleanedInput)

	// Tokenize the cleaned input
	tokens := jargon.Tokenize(strings.NewReader(cleanedInput))

	synonym := map[string]string{
		"quick": "fast",
		"smart": "intelligent",
		"large": "big",
	}

	// Create a new synonym filter with custom mappings
	filter := synonyms.NewFilter(synonym, true, nil)
	filtered := filter(tokens)

	// Collect normalized words
	var normalizedWords []string
	for {
		token, err := filtered.Next()
		if err != nil {
			break
		}
		if token == nil {
			continue // skip nil tokens
		}
		normalizedWords = append(normalizedWords, token.String())
	}
	sort.Strings(normalizedWords)

	return strings.Join(normalizedWords, " ")
}

func calculateSimilarity(s1, s2 string) float64 {
	distance := levenshtein.ComputeDistance(s1, s2)
	maxLen := max(len(s1), len(s2))
	return 1 - float64(distance)/float64(maxLen)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func loadCommandDatasets() *CommandDataset {
	dataset := &CommandDataset{
		Commands: make(map[string]CommandEntry),
	}

	datasetFiles := []string{
		"data/cmd_commands.csv",
		"data/linux_commands.csv",
		"data/macos_commands.csv",
		"data/vbscript_commands.csv",
	}

	for _, file := range datasetFiles {
		loadCSV(dataset, file)
	}

	return dataset
}

func loadCSV(dataset *CommandDataset, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", filename, err)
		return
	}
	fmt.Print("Loading commands from ", filename)
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error reading CSV from %s: %v\n", filename, err)
		return
	}

	for _, record := range records {
		if len(record) >= 2 {
			dataset.Commands[record[0]] = CommandEntry{
				Command:     record[0],
				Description: record[1],
			}
		}
	}
}

func CallLLM(model AIModel, prompt string) (string, error) {
	llmPrompt := getPrompt(config.OsType, prompt)
	switch model {
	case Ollama:
		return callOllama(llmPrompt)
	case OpenAI:
		return callOpenAI(llmPrompt)
	case Anthropic:
		return callAnthropic(llmPrompt)
	default:
		return "", fmt.Errorf("unknown model")
	}
}

func callOllama(prompt string) (string, error) {
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

func callOpenAI(prompt string) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	}
	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func callAnthropic(prompt string) (string, error) {
	// Implementation for Anthropic API
	fmt.Print(prompt)
	return "", fmt.Errorf("anthropic API is not implemented")
}

func (dc *DistributedCache) GetOrFetch(input string, model AIModel) (string, error) {
	// Check cache first
	if response, found := dc.Get(input); found {
		return response, nil
	}

	// Check command dataset
	if command, found := findMatchingCommand(input); found {
		return fmt.Sprintf("Command: %s\nDescription: %s", command.Command, command.Description), nil
	}
	fmt.Println("Command not found in dataset", input)

	// If not in cache or command dataset, fetch from LLM
	response, err := CallLLM(model, input)
	if err != nil {
		return "", err
	}

	// Cache the result
	dc.Set(input, response)
	return response, nil
}

func findMatchingCommand(input string) (CommandEntry, bool) {
	normalizedInput := normalizeInput(input)
	var bestMatch CommandEntry
	var bestSimilarity float64

	for _, entry := range commandDataset.Commands {
		similarity := calculateSimilarity(normalizedInput, normalizeInput(entry.Command))
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestMatch = entry
		}
	}

	if bestSimilarity > similarityThreshold {
		return bestMatch, true
	}

	return CommandEntry{}, false
}

func FetchLLMResponse(prompt string, selectedModel AIModel) (string, error) {
	return globalCache.GetOrFetch(prompt, selectedModel)
}
