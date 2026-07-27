package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type openAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(apiKey, model, baseURL string, timeout time.Duration) (LLMProvider, error) {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &openAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

func (o *openAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	url := o.baseURL + "/chat/completions"

	payload := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a formal methods compiler. You output ONLY valid, raw JSON without markdown formatting or code blocks."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
	}

	if strings.Contains(o.baseURL, "api.openai.com") {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if o.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm api returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode llm response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", errors.New("llm returned an empty choice list")
	}

	return result.Choices[0].Message.Content, nil
}
