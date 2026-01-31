//go:build ignore

// measure-context.go - Measure context overhead for different approaches
//
// This tool sends a known-size prompt and measures:
// 1. Input tokens consumed (overhead indicator)
// 2. Response latency
// 3. Effective context ratio
//
// Usage:
//   go run measure-context.go --provider anthropic
//   go run measure-context.go --provider openai
//   go run measure-context.go --provider google

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var provider = flag.String("provider", "anthropic", "API provider: anthropic, openai, google")

// Minimal test prompt - known size
const testPrompt = `Pick one word from this list and make it bold:
apple banana cherry
Return ONLY the bold word.`

// Rough token estimate (words * 1.3)
func estimateTokens(text string) int {
	words := len(strings.Fields(text))
	return int(float64(words) * 1.3)
}

type Result struct {
	Provider      string
	Model         string
	InputTokens   int
	OutputTokens  int
	Latency       time.Duration
	PromptEstimate int
	Overhead      int  // InputTokens - PromptEstimate
	Response      string
}

func callAnthropic(prompt string) (*Result, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	start := time.Now()

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 64,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var data struct {
		Content []struct{ Text string } `json:"content"`
		Usage   struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(respBody, &data)

	promptEst := estimateTokens(prompt)
	return &Result{
		Provider:       "anthropic",
		Model:          "claude-sonnet-4-20250514",
		InputTokens:    data.Usage.InputTokens,
		OutputTokens:   data.Usage.OutputTokens,
		Latency:        latency,
		PromptEstimate: promptEst,
		Overhead:       data.Usage.InputTokens - promptEst,
		Response:       data.Content[0].Text,
	}, nil
}

func callOpenAI(prompt string) (*Result, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	start := time.Now()

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "gpt-4o",
		"max_tokens": 64,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var data struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(respBody, &data)

	promptEst := estimateTokens(prompt)
	return &Result{
		Provider:       "openai",
		Model:          "gpt-4o",
		InputTokens:    data.Usage.PromptTokens,
		OutputTokens:   data.Usage.CompletionTokens,
		Latency:        latency,
		PromptEstimate: promptEst,
		Overhead:       data.Usage.PromptTokens - promptEst,
		Response:       data.Choices[0].Message.Content,
	}, nil
}

func main() {
	flag.Parse()

	fmt.Printf("=== Context Overhead Measurement ===\n")
	fmt.Printf("Provider: %s\n", *provider)
	fmt.Printf("Test prompt: %d chars, ~%d tokens estimated\n\n",
		len(testPrompt), estimateTokens(testPrompt))

	var result *Result
	var err error

	switch *provider {
	case "anthropic":
		result, err = callAnthropic(testPrompt)
	case "openai":
		result, err = callOpenAI(testPrompt)
	case "google":
		fmt.Println("Google/Gemini: Use option-e-gemini.sh for now")
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", *provider)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Results:\n")
	fmt.Printf("  Model: %s\n", result.Model)
	fmt.Printf("  Latency: %v\n", result.Latency)
	fmt.Printf("  Input tokens: %d\n", result.InputTokens)
	fmt.Printf("  Output tokens: %d\n", result.OutputTokens)
	fmt.Printf("  Prompt estimate: %d tokens\n", result.PromptEstimate)
	fmt.Printf("  Overhead: %d tokens\n", result.Overhead)
	fmt.Printf("  Response: %s\n", result.Response)

	// Context window comparison
	fmt.Printf("\nContext Efficiency (200K window):\n")
	fmt.Printf("  Available: %d tokens (%.2f%%)\n",
		200000-result.InputTokens,
		float64(200000-result.InputTokens)/2000.0)
}
