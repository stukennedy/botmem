package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM        LLMConfig        `yaml:"llm"`
	Embeddings EmbeddingsConfig `yaml:"embeddings"`
}

type LLMConfig struct {
	Provider string `yaml:"provider"` // "anthropic", "ollama"
	Model    string `yaml:"model"`    // e.g. "claude-sonnet-4-20250514", "llama3.2"
	APIKey   string `yaml:"api_key"`  // for anthropic
	BaseURL  string `yaml:"base_url"` // for ollama, default http://localhost:11434
}

type EmbeddingsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Model   string `yaml:"model"`   // e.g. "nomic-embed-text"
	BaseURL string `yaml:"base_url"`
}

// DefaultPath returns ~/.botmem/config.yaml
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".botmem", "config.yaml"), nil
}

// Load reads the config file. Returns an error if it doesn't exist.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no config found — run 'botmem init' to set up")
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config, path string) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600) // 0600 — may contain API keys
}

// Exists checks if a config file exists.
func Exists(path string) bool {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return false
		}
	}
	_, err := os.Stat(path)
	return err == nil
}
