package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type anthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropicProvider(apiKey, model string, timeout time.Duration) (LLMProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required for the Anthropic provider")
	}
	if model == "" {
		model = "claude-sonnet-5"
	}
	return &anthropicProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (a *anthropicProvider) Generate(ctx context.Context, prompt string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	payload := map[string]interface{}{
		"model":       a.model,
		"max_tokens":  4096,
		"system":      "You are a formal methods compiler. You output ONLY valid, raw JSON without markdown formatting or code blocks.",
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"temperature": 0.0,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic api returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode anthropic response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("anthropic returned an empty response")
	}

	return result.Content[0].Text, nil
}
