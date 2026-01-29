package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_ClaudeCode(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// Choose Claude Code (1, the default), no embeddings
	input := "1\nn\n"
	var out bytes.Buffer

	cfg, err := RunInit(strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	if cfg.LLM.Provider != "claude" {
		t.Errorf("expected claude, got %q", cfg.LLM.Provider)
	}

	savedPath := filepath.Join(dir, ".botmem", "config.yaml")
	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Error("config file not saved")
	}

	output := out.String()
	if !strings.Contains(output, "botmem block set") {
		t.Error("output should mention example commands")
	}
}

func TestRunInit_Anthropic(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// Choose Anthropic (2), default model, enter API key, no embeddings
	input := "2\n\nsk-test-key\nn\n"
	var out bytes.Buffer

	cfg, err := RunInit(strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("expected anthropic, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.APIKey != "sk-test-key" {
		t.Errorf("expected sk-test-key, got %q", cfg.LLM.APIKey)
	}
	if cfg.Embeddings.Enabled {
		t.Error("expected embeddings disabled")
	}
}

func TestRunInit_Ollama(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// Choose Ollama (3), default URL, default model, enable embeddings with defaults
	input := "3\n\n\ny\n\n\n"
	var out bytes.Buffer

	cfg, err := RunInit(strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected ollama, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.BaseURL != "http://localhost:11434" {
		t.Errorf("expected default URL, got %q", cfg.LLM.BaseURL)
	}
	if !cfg.Embeddings.Enabled {
		t.Error("expected embeddings enabled")
	}
	if cfg.Embeddings.Model != "nomic-embed-text" {
		t.Errorf("expected nomic-embed-text, got %q", cfg.Embeddings.Model)
	}
}

func TestRunInit_DefaultsToClaudeCode(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// Just press enter for everything (defaults)
	input := "\nn\n"
	var out bytes.Buffer

	cfg, err := RunInit(strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	if cfg.LLM.Provider != "claude" {
		t.Errorf("expected claude as default, got %q", cfg.LLM.Provider)
	}
}
