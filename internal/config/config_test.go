package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		LLM: LLMConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			APIKey:   "sk-test-123",
		},
		Embeddings: EmbeddingsConfig{
			Enabled: true,
			Model:   "nomic-embed-text",
			BaseURL: "http://localhost:11434",
		},
	}

	if err := Save(cfg, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.LLM.Provider != "anthropic" {
		t.Errorf("expected anthropic, got %q", loaded.LLM.Provider)
	}
	if loaded.LLM.APIKey != "sk-test-123" {
		t.Errorf("expected sk-test-123, got %q", loaded.LLM.APIKey)
	}
	if !loaded.Embeddings.Enabled {
		t.Error("expected embeddings enabled")
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "botmem init") {
		t.Errorf("error should mention 'botmem init', got: %s", err)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{LLM: LLMConfig{APIKey: "secret"}}
	if err := Save(cfg, path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected 0600 permissions, got %o", perm)
	}
}

func TestLoad_EmbeddingsBool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("llm:\n  provider: anthropic\nembeddings: false\n"), 0600)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Embeddings.Enabled {
		t.Error("expected embeddings disabled")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if Exists(path) {
		t.Error("should not exist yet")
	}

	Save(&Config{}, path)

	if !Exists(path) {
		t.Error("should exist after save")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	if err := Save(&Config{}, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file not created")
	}
}
