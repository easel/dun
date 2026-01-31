//go:build ignore

// Option D: Direct Anthropic API calls
//
// Pros:
// - Full programmatic control
// - No CLI dependency
// - Can manage context window precisely
// - Structured responses via tool_use
//
// Cons:
// - Most complex implementation
// - Need to handle API errors, retries, rate limits
// - Token counting and context management
// - No access to Claude Code's tools (Edit, Bash, etc.)

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func callAnthropic(prompt string) (*Response, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	req := Request{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 256,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", anthropicAPI, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

const prompt = `Here is a list of 10 words. Pick your favorite word, make it **bold**, and move it to OUTPUT.

INPUT: apple banana cherry date elderberry fig grape honeydew kiwi lemon
OUTPUT:

Respond with ONLY the updated INPUT/OUTPUT block, nothing else.`

func main() {
	fmt.Println("=== Option D: Direct API ===")
	fmt.Println("Calling Anthropic API...")
	fmt.Println()

	resp, err := callAnthropic(prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(resp.Content) > 0 {
		fmt.Println("Response:")
		fmt.Println(resp.Content[0].Text)
		fmt.Println()
		fmt.Printf("Tokens: %d input, %d output\n",
			resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}
}
