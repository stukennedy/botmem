package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// RunInit walks the user through setting up botmem.
// It accepts an io.Reader and io.Writer so it can be tested.
func RunInit(in io.Reader, out io.Writer) (*Config, error) {
	scanner := bufio.NewScanner(in)
	prompt := func(question, defaultVal string) string {
		if defaultVal != "" {
			fmt.Fprintf(out, "%s [%s]: ", question, defaultVal)
		} else {
			fmt.Fprintf(out, "%s: ", question)
		}
		if scanner.Scan() {
			val := strings.TrimSpace(scanner.Text())
			if val != "" {
				return val
			}
		}
		return defaultVal
	}

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  botmem — Local LLM Memory System")
	fmt.Fprintln(out, "  ================================")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  This will set up your memory configuration.")
	fmt.Fprintln(out, "")

	// LLM Provider
	fmt.Fprintln(out, "  Which LLM provider do you want to use for memory extraction?")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "    1) Claude Code (uses claude -p — recommended)")
	fmt.Fprintln(out, "    2) Anthropic   (Claude API — requires API key)")
	fmt.Fprintln(out, "    3) Ollama      (local models — requires Ollama running)")
	fmt.Fprintln(out, "")

	choice := prompt("  Choose (1, 2, or 3)", "1")

	cfg := &Config{}

	switch choice {
	case "3":
		cfg.LLM.Provider = "ollama"
		cfg.LLM.BaseURL = prompt("  Ollama URL", "http://localhost:11434")
		cfg.LLM.Model = prompt("  Ollama model for extraction", "llama3.2")
	case "2":
		cfg.LLM.Provider = "anthropic"
		cfg.LLM.Model = prompt("  Anthropic model", "claude-sonnet-4-20250514")

		// Check env first
		envKey := os.Getenv("ANTHROPIC_API_KEY")
		if envKey != "" {
			fmt.Fprintln(out, "")
			fmt.Fprintln(out, "  Found ANTHROPIC_API_KEY in environment.")
			useEnv := prompt("  Use environment variable? (y/n)", "y")
			if strings.ToLower(useEnv) == "y" {
				cfg.LLM.APIKey = "" // will read from env at runtime
				fmt.Fprintln(out, "  Will use ANTHROPIC_API_KEY from environment.")
			} else {
				cfg.LLM.APIKey = prompt("  Anthropic API key", "")
			}
		} else {
			fmt.Fprintln(out, "")
			fmt.Fprintln(out, "  You can provide your API key here, or set ANTHROPIC_API_KEY env var.")
			cfg.LLM.APIKey = prompt("  Anthropic API key (or leave empty to use env var)", "")
		}
	default:
		cfg.LLM.Provider = "claude"
		cfg.LLM.Model = "claude"
		fmt.Fprintln(out, "  Using Claude Code (claude -p). No API key needed.")
	}

	// Embeddings
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  Embeddings enable semantic search (finding memories by meaning).")
	fmt.Fprintln(out, "  This is optional — keyword search (FTS5) works without it.")
	fmt.Fprintln(out, "  Embeddings require Ollama running locally.")
	fmt.Fprintln(out, "")

	enableEmb := prompt("  Enable embeddings? (y/n)", "n")
	if strings.ToLower(enableEmb) == "y" {
		cfg.Embeddings.Enabled = true
		cfg.Embeddings.BaseURL = prompt("  Ollama URL for embeddings", "http://localhost:11434")
		cfg.Embeddings.Model = prompt("  Embedding model", "nomic-embed-text")
	}

	// Save
	fmt.Fprintln(out, "")
	path, _ := DefaultPath()
	if err := Save(cfg, ""); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(out, "  Config saved to %s\n", path)
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  You're all set! Try:")
	fmt.Fprintln(out, "    botmem block set human \"Your name and info here\"")
	fmt.Fprintln(out, "    botmem context")
	fmt.Fprintln(out, "")

	return cfg, nil
}
