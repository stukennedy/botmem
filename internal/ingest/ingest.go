package ingest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/stukennedy/botmem/internal/embeddings"
	"github.com/stukennedy/botmem/internal/memory"
)

// ExtractionResult is what the LLM returns after analyzing conversation text.
type ExtractionResult struct {
	BlockUpdates []BlockUpdate `json:"block_updates"`
	Facts        []Fact        `json:"facts"`
	Triplets     []Triplet     `json:"triplets"`
	Summary      string        `json:"summary"`
}

type BlockUpdate struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}

type Fact struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type Triplet struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

const systemPrompt = `You are a memory extraction system. Given conversation text, extract:

1. block_updates: Updates to core memory blocks. Labels are: "human" (personal info about the user), "persona" (bot personality), "context" (current project/session context). Only include blocks that need updating. Provide the FULL updated content for each block, not just the diff.

2. facts: Important facts worth remembering long-term. Each fact should be a self-contained statement with relevant tags.

3. triplets: Entity-relationship triplets (subject, predicate, object) for the knowledge graph. Examples: ("Stuart", "works_on", "Moltbot"), ("Moltbot", "is_a", "Discord bot").

4. summary: A concise summary of this conversation.

Return ONLY valid JSON matching this schema:
{
  "block_updates": [{"label": "string", "content": "string"}],
  "facts": [{"content": "string", "tags": ["string"]}],
  "triplets": [{"subject": "string", "predicate": "string", "object": "string"}],
  "summary": "string"
}`

// Config holds settings for the ingest pipeline.
type Config struct {
	Provider  string // "anthropic" or "ollama"
	LLMURL    string // e.g., http://localhost:11434 for Ollama
	LLMModel  string // e.g., llama3.2, claude-sonnet-4-20250514
	APIKey    string // for anthropic
	EmbedProv embeddings.Provider
}

// ConfigFromAppConfig creates an ingest Config from the app-level config.
func ConfigFromAppConfig(provider, model, apiKey, baseURL string, embedProv embeddings.Provider) *Config {
	return &Config{
		Provider:  provider,
		LLMURL:    baseURL,
		LLMModel:  model,
		APIKey:    apiKey,
		EmbedProv: embedProv,
	}
}

// Run processes conversation text through the LLM and stores extracted information.
func Run(db *sql.DB, text string, cfg *Config) (*ExtractionResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("no config provided — run 'botmem init' to set up")
	}

	var result *ExtractionResult
	var err error

	switch cfg.Provider {
	case "claude":
		result, err = extractWithClaude(text)
	case "anthropic":
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("no Anthropic API key — set ANTHROPIC_API_KEY or run 'botmem init'")
		}
		result, err = extractWithAnthropic(text, apiKey)
	case "ollama":
		result, err = extractWithOllama(text, cfg)
	default:
		return nil, fmt.Errorf("unknown provider %q — run 'botmem init' to configure", cfg.Provider)
	}
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	// Apply block updates
	blocks := memory.NewBlockStore(db)
	for _, bu := range result.BlockUpdates {
		existing, err := blocks.GetByLabel(bu.Label)
		if err != nil {
			// Block doesn't exist, create it
			if _, err := blocks.Create(bu.Label, "core", bu.Content); err != nil {
				return nil, fmt.Errorf("create block %q: %w", bu.Label, err)
			}
		} else {
			_ = existing
			if _, err := blocks.Update(bu.Label, bu.Content); err != nil {
				return nil, fmt.Errorf("update block %q: %w", bu.Label, err)
			}
		}
	}

	// Store facts in archival
	archival := memory.NewArchivalStore(db)
	for _, f := range result.Facts {
		var emb []byte
		if cfg.EmbedProv != nil {
			if vec, err := cfg.EmbedProv.Embed(f.Content); err == nil {
				emb = embeddings.SerializeEmbedding(vec)
			}
		}
		if _, err := archival.Add(f.Content, f.Tags, emb); err != nil {
			return nil, fmt.Errorf("add fact: %w", err)
		}
	}

	// Store triplets in graph
	graph := memory.NewGraphStore(db)
	for _, t := range result.Triplets {
		if err := graph.AddRelation(t.Subject, t.Predicate, t.Object, ""); err != nil {
			return nil, fmt.Errorf("add triplet: %w", err)
		}
	}

	// Store summary
	if result.Summary != "" {
		summaries := memory.NewSummaryStore(db)
		if _, err := summaries.Add(0, result.Summary, ""); err != nil {
			return nil, fmt.Errorf("add summary: %w", err)
		}
	}

	return result, nil
}

func extractWithOllama(text string, cfg *Config) (*ExtractionResult, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":  cfg.LLMModel,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": text},
		},
		"format": "json",
	})

	resp, err := http.Post(cfg.LLMURL+"/api/chat", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, body)
	}

	var ollamaResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	var result ExtractionResult
	if err := json.Unmarshal([]byte(ollamaResp.Message.Content), &result); err != nil {
		return nil, fmt.Errorf("decode extraction result: %w\nraw: %s", err, ollamaResp.Message.Content)
	}
	return &result, nil
}

func extractWithAnthropic(text, apiKey string) (*ExtractionResult, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": text},
		},
	})

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, body)
	}

	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}
	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("empty anthropic response")
	}

	var result ExtractionResult
	if err := json.Unmarshal([]byte(anthropicResp.Content[0].Text), &result); err != nil {
		return nil, fmt.Errorf("decode extraction result: %w\nraw: %s", err, anthropicResp.Content[0].Text)
	}
	return &result, nil
}

func extractWithClaude(text string) (*ExtractionResult, error) {
	// Build the full prompt: system instructions + user text
	prompt := systemPrompt + "\n\nConversation text to extract from:\n\n" + text

	cmd := exec.Command("claude", "-p", "--output-format", "text", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude -p failed: %w\nstderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return nil, fmt.Errorf("empty response from claude -p")
	}

	// Claude may wrap JSON in markdown code fences — strip them
	output = stripCodeFences(output)

	var result ExtractionResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("decode extraction result: %w\nraw: %s", err, output)
	}
	return &result, nil
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (```json or ```)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}
