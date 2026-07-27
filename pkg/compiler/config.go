package compiler

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3" // Or use standard encoding/json if preferring zero external dependencies
)

// ProviderDefaults maps vendor names to their default models and endpoints.
type ProviderDefaults struct {
	DefaultModel string `yaml:"default_model" json:"default_model"`
	BaseURL      string `yaml:"base_url" json:"base_url"`
}

// ConfigFile Schema
type ConfigFile struct {
	DefaultProvider string                      `yaml:"default_provider" json:"default_provider"`
	Providers       map[string]ProviderDefaults `yaml:"providers" json:"providers"`
}

// LoadConfig resolves parameters using the hierarchy: Env Vars > Config File > Hardcoded Safety Net
func LoadConfig() Config {
	// 1. Embedded fallback safety net (in case no config file or env vars exist)
	fileCfg := ConfigFile{
		DefaultProvider: "gemini",
		Providers: map[string]ProviderDefaults{
			"gemini":    {DefaultModel: "gemini-2.5-flash"},
			"anthropic": {DefaultModel: "claude-3-7-sonnet"},
			"openai":    {DefaultModel: "gpt-4o"},
			"ollama":    {DefaultModel: "deepseek-r1", BaseURL: "http://localhost:11434/v1"},
		},
	}

	// 2. Try loading from file (checks current dir, ~/.formaljudge.yaml, or /etc/formaljudge/)
	if loaded, err := loadConfigFile(); err == nil {
		fileCfg = *loaded
	}

	// 3. Resolve Active Provider (ENV overrides Config File)
	provider := strings.ToLower(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = fileCfg.DefaultProvider
	}

	// 4. Resolve Active Model & Base URL for chosen provider
	defaults := fileCfg.Providers[provider]

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = defaults.DefaultModel
	}

	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = defaults.BaseURL
	}

	return Config{
		Provider: provider,
		Model:    model,
		APIKey:   os.Getenv("LLM_API_KEY"),
		BaseURL:  baseURL,
		Timeout:  DefaultHTTPTimeout,
	}
}

// loadConfigFile searches standard locations for a config file
func loadConfigFile() (*ConfigFile, error) {
	searchPaths := []string{
		"./config.yaml",
		"./formaljudge.yaml",
		"./formaljudge.json",
		filepath.Join(os.Getenv("HOME"), ".formaljudge.yaml"),
		"/etc/formaljudge/formaljudge.yaml",
	}

	for _, path := range searchPaths {
		cleanPath := filepath.Clean(path)

		//nolint:gosec // G703: searchPaths are predefined, trusted configuration locations.
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			continue // File not found, try next
		}

		var cfg ConfigFile
		if strings.HasSuffix(cleanPath, ".json") {
			if err := json.Unmarshal(data, &cfg); err == nil {
				return &cfg, nil
			}
		} else {
			if err := yaml.Unmarshal(data, &cfg); err == nil {
				return &cfg, nil
			}
		}
	}

	return nil, errors.New("no valid config file found in search paths")
}
