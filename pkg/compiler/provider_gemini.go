package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type geminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiProvider(apiKey, model string, timeout time.Duration) (LLMProvider, error) {
	if apiKey == "" {
		return nil, errors.New("LLM_API_KEY is required for the Gemini provider")
	}
	if model == "" {
		model = "gemini-3.6-flash"
	}
	return &geminiProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (g *geminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	payload := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"temperature":      0.0,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini api returned HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode gemini response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini returned an empty response")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
