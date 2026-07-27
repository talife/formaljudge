package compiler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// DefaultHTTPTimeout prevents worker goroutine hangs during LLM generation.
const DefaultHTTPTimeout = 60 * time.Second

// LLMProvider defines the contract for formal methods code generation.
type LLMProvider interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// Config holds runtime configuration for the neuro-symbolic compiler.
type Config struct {
	Provider string // "gemini", "anthropic", "openai", "ollama", "vllm"
	Model    string
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
}

// LoadConfigFromEnv inspects environment variables and assigns up-to-date production defaults.
func LoadConfigFromEnv() Config {
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = "gemini"
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		switch provider {
		case "anthropic", "claude":
			model = "claude-sonnet-5"
		case "openai":
			model = "gpt-5.6-sol"
		case "ollama", "vllm":
			model = "deepseek-r1"
		default: // gemini
			model = "gemini-3.6-flash"
		}
	}

	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" && (provider == "ollama" || provider == "vllm") {
		baseURL = "http://localhost:11434/v1"
	}

	return Config{
		Provider: provider,
		Model:    model,
		APIKey:   os.Getenv("LLM_API_KEY"),
		BaseURL:  baseURL,
		Timeout:  DefaultHTTPTimeout,
	}
}

// NewProvider instantiates the correct provider adapter.
func NewProvider(cfg Config) (LLMProvider, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultHTTPTimeout
	}

	switch strings.ToLower(cfg.Provider) {
	case "gemini":
		return NewGeminiProvider(cfg.APIKey, cfg.Model, cfg.Timeout)
	case "anthropic", "claude":
		return NewAnthropicProvider(cfg.APIKey, cfg.Model, cfg.Timeout)
	case "openai", "ollama", "vllm":
		return NewOpenAIProvider(cfg.APIKey, cfg.Model, cfg.BaseURL, cfg.Timeout)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
