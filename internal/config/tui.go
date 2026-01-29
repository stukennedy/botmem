package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/stukennedy/tooey/app"
	"github.com/stukennedy/tooey/component"
	"github.com/stukennedy/tooey/input"
	"github.com/stukennedy/tooey/node"
	"golang.org/x/term"
)

// TUI screen states
const (
	screenWelcome = iota
	screenProvider
	screenAnthropicModel
	screenAnthropicKey
	screenOllamaURL
	screenOllamaModel
	screenEmbeddings
	screenEmbeddingsURL
	screenEmbeddingsModel
	screenConfirm
	screenDone
)

type tuiModel struct {
	screen   int
	cfg      Config
	selected int // for list selections

	// text inputs
	apiKeyInput     component.TextInput
	modelInput      component.TextInput
	urlInput        component.TextInput
	embURLInput     component.TextInput
	embModelInput   component.TextInput

	envKeyFound bool
	useEnvKey   bool
	err         error
}

// RunInitTUI runs the interactive TUI setup wizard.
// Returns the saved config or an error.
func RunInitTUI() (*Config, error) {
	// Check if we're in a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return RunInit(os.Stdin, os.Stdout)
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fall back to plain text
		return RunInit(os.Stdin, os.Stdout)
	}

	var result *Config
	var resultErr error

	a := &app.App{
		Init: func() interface{} {
			envKey := os.Getenv("ANTHROPIC_API_KEY")
			return &tuiModel{
				screen:        screenWelcome,
				apiKeyInput:   component.NewTextInput("sk-ant-..."),
				modelInput:    component.NewTextInput("claude-sonnet-4-20250514"),
				urlInput:      component.NewTextInput("http://localhost:11434"),
				embURLInput:   component.NewTextInput("http://localhost:11434"),
				embModelInput: component.NewTextInput("nomic-embed-text"),
				envKeyFound:   envKey != "",
			}
		},
		Update: func(m interface{}, msg app.Msg) app.UpdateResult {
			mdl := m.(*tuiModel)

			km, ok := msg.(app.KeyMsg)
			if !ok {
				return app.NoCmd(mdl)
			}

			// Global quit
			if km.Key.Type == input.CtrlC {
				resultErr = fmt.Errorf("cancelled")
				return app.UpdateResult{Model: nil}
			}

			switch mdl.screen {
			case screenWelcome:
				if km.Key.Type == input.Enter {
					mdl.screen = screenProvider
				}

			case screenProvider:
				switch km.Key.Type {
				case input.Up:
					if mdl.selected > 0 {
						mdl.selected--
					}
				case input.Down:
					if mdl.selected < 2 {
						mdl.selected++
					}
				case input.Enter:
					switch mdl.selected {
					case 0:
						mdl.cfg.LLM.Provider = "claude"
						mdl.cfg.LLM.Model = "claude" // uses whatever model claude code is configured with
						mdl.screen = screenEmbeddings
					case 1:
						mdl.cfg.LLM.Provider = "anthropic"
						mdl.screen = screenAnthropicModel
					case 2:
						mdl.cfg.LLM.Provider = "ollama"
						mdl.screen = screenOllamaURL
					}
					mdl.selected = 0
				}

			case screenAnthropicModel:
				if km.Key.Type == input.Enter {
					val := strings.TrimSpace(mdl.modelInput.Value)
					if val == "" {
						val = "claude-sonnet-4-20250514"
					}
					mdl.cfg.LLM.Model = val
					if mdl.envKeyFound {
						mdl.useEnvKey = true
						mdl.screen = screenEmbeddings
					} else {
						mdl.screen = screenAnthropicKey
					}
				} else {
					mdl.modelInput = mdl.modelInput.Update(km.Key)
				}

			case screenAnthropicKey:
				if km.Key.Type == input.Enter {
					val := strings.TrimSpace(mdl.apiKeyInput.Value)
					mdl.cfg.LLM.APIKey = val
					mdl.screen = screenEmbeddings
				} else {
					mdl.apiKeyInput = mdl.apiKeyInput.Update(km.Key)
				}

			case screenOllamaURL:
				if km.Key.Type == input.Enter {
					val := strings.TrimSpace(mdl.urlInput.Value)
					if val == "" {
						val = "http://localhost:11434"
					}
					mdl.cfg.LLM.BaseURL = val
					mdl.screen = screenOllamaModel
				} else {
					mdl.urlInput = mdl.urlInput.Update(km.Key)
				}

			case screenOllamaModel:
				if km.Key.Type == input.Enter {
					val := strings.TrimSpace(mdl.modelInput.Value)
					if val == "" {
						val = "llama3.2"
					}
					mdl.cfg.LLM.Model = val
					mdl.screen = screenEmbeddings
				} else {
					mdl.modelInput = mdl.modelInput.Update(km.Key)
				}

			case screenEmbeddings:
				switch km.Key.Type {
				case input.Up:
					if mdl.selected > 0 {
						mdl.selected--
					}
				case input.Down:
					if mdl.selected < 1 {
						mdl.selected++
					}
				case input.Enter:
					if mdl.selected == 0 {
						// No embeddings
						mdl.screen = screenConfirm
					} else {
						mdl.cfg.Embeddings.Enabled = true
						mdl.screen = screenEmbeddingsURL
					}
					mdl.selected = 0
				}

			case screenEmbeddingsURL:
				if km.Key.Type == input.Enter {
					val := strings.TrimSpace(mdl.embURLInput.Value)
					if val == "" {
						val = "http://localhost:11434"
					}
					mdl.cfg.Embeddings.BaseURL = val
					mdl.screen = screenEmbeddingsModel
				} else {
					mdl.embURLInput = mdl.embURLInput.Update(km.Key)
				}

			case screenEmbeddingsModel:
				if km.Key.Type == input.Enter {
					val := strings.TrimSpace(mdl.embModelInput.Value)
					if val == "" {
						val = "nomic-embed-text"
					}
					mdl.cfg.Embeddings.Model = val
					mdl.screen = screenConfirm
				} else {
					mdl.embModelInput = mdl.embModelInput.Update(km.Key)
				}

			case screenConfirm:
				if km.Key.Type == input.Enter {
					if err := Save(&mdl.cfg, ""); err != nil {
						mdl.err = err
					}
					result = &mdl.cfg
					mdl.screen = screenDone
				}
				if km.Key.Type == input.Escape {
					mdl.screen = screenProvider
				}

			case screenDone:
				if km.Key.Type == input.Enter || km.Key.Type == input.Escape {
					return app.UpdateResult{Model: nil}
				}
			}

			return app.NoCmd(mdl)
		},
		View: func(m interface{}, focused string) node.Node {
			mdl := m.(*tuiModel)
			return renderTUI(mdl, focused)
		},
	}

	err = a.Run(context.Background())
	term.Restore(int(os.Stdin.Fd()), oldState)

	if err != nil && resultErr == nil {
		resultErr = err
	}
	if resultErr != nil {
		return nil, resultErr
	}
	return result, nil
}

func renderTUI(mdl *tuiModel, focused string) node.Node {
	title := node.TextStyled("  ◆ botmem — Memory System Setup", node.Color(6), 0, node.Bold)
	divider := node.TextStyled("  "+strings.Repeat("─", 40), node.Color(8), 0, 0)

	var content node.Node

	switch mdl.screen {
	case screenWelcome:
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Welcome to botmem!", node.Color(2), 0, node.Bold),
			node.Text(""),
			node.TextStyled("  botmem gives your LLM persistent memory —", node.Color(7), 0, 0),
			node.TextStyled("  conversations, facts, relationships, and context", node.Color(7), 0, 0),
			node.TextStyled("  that survive between sessions.", node.Color(7), 0, 0),
			node.Text(""),
			node.TextStyled("  Let's get you set up.", node.Color(8), 0, node.Italic),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Press Enter to begin →", node.Color(3), 0, 0),
			node.Text(""),
		)

	case screenProvider:
		items := component.List{
			Key:        "provider",
			Items:      []string{"Claude Code (uses claude -p — recommended)", "Anthropic   (Claude API — requires key)", "Ollama      (local models — private)"},
			Selected:   mdl.selected,
			FG:         node.Color(7),
			SelectedFG: node.Color(0),
			SelectedBG: node.Color(6),
		}
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Which LLM provider for memory extraction?", node.Color(2), 0, node.Bold),
			node.Text(""),
			items.Render(focused),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  ↑/↓ to select, Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenAnthropicModel:
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Anthropic Model", node.Color(2), 0, node.Bold),
			node.Text(""),
			node.TextStyled("  Which model should be used for memory extraction?", node.Color(7), 0, 0),
			node.TextStyled("  (Press Enter for default)", node.Color(8), 0, node.Italic),
			node.Text(""),
			mdl.modelInput.Render("  Model: ", node.Color(7), 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenAnthropicKey:
		var hint node.Node
		if mdl.envKeyFound {
			hint = node.TextStyled("  ✓ ANTHROPIC_API_KEY found in environment", node.Color(2), 0, 0)
		} else {
			hint = node.Column(
				node.TextStyled("  Enter your API key, or leave blank to use", node.Color(7), 0, 0),
				node.TextStyled("  the ANTHROPIC_API_KEY environment variable.", node.Color(7), 0, 0),
			)
		}
		content = node.Column(
			node.Text(""),
			node.TextStyled("  API Key", node.Color(2), 0, node.Bold),
			node.Text(""),
			hint,
			node.Text(""),
			mdl.apiKeyInput.Render("  Key: ", node.Color(7), 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenOllamaURL:
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Ollama URL", node.Color(2), 0, node.Bold),
			node.Text(""),
			node.TextStyled("  Where is Ollama running?", node.Color(7), 0, 0),
			node.TextStyled("  (Press Enter for default)", node.Color(8), 0, node.Italic),
			node.Text(""),
			mdl.urlInput.Render("  URL: ", node.Color(7), 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenOllamaModel:
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Ollama Model", node.Color(2), 0, node.Bold),
			node.Text(""),
			node.TextStyled("  Which model for memory extraction?", node.Color(7), 0, 0),
			node.TextStyled("  (Press Enter for default)", node.Color(8), 0, node.Italic),
			node.Text(""),
			mdl.modelInput.Render("  Model: ", node.Color(7), 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenEmbeddings:
		items := component.List{
			Key:        "embeddings",
			Items:      []string{"Skip  (keyword search only — simpler)", "Enable (semantic search — requires Ollama)"},
			Selected:   mdl.selected,
			FG:         node.Color(7),
			SelectedFG: node.Color(0),
			SelectedBG: node.Color(6),
		}
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Embeddings (Semantic Search)", node.Color(2), 0, node.Bold),
			node.Text(""),
			node.TextStyled("  Embeddings let you search memories by meaning,", node.Color(7), 0, 0),
			node.TextStyled("  not just keywords. Requires Ollama locally.", node.Color(7), 0, 0),
			node.TextStyled("  Keyword search (FTS5) works without this.", node.Color(8), 0, node.Italic),
			node.Text(""),
			items.Render(focused),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  ↑/↓ to select, Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenEmbeddingsURL:
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Embeddings — Ollama URL", node.Color(2), 0, node.Bold),
			node.Text(""),
			mdl.embURLInput.Render("  URL: ", node.Color(7), 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenEmbeddingsModel:
		content = node.Column(
			node.Text(""),
			node.TextStyled("  Embeddings — Model", node.Color(2), 0, node.Bold),
			node.Text(""),
			mdl.embModelInput.Render("  Model: ", node.Color(7), 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to confirm", node.Color(8), 0, 0),
			node.Text(""),
		)

	case screenConfirm:
		lines := []node.Node{
			node.Text(""),
			node.TextStyled("  Configuration Summary", node.Color(2), 0, node.Bold),
			node.Text(""),
			node.TextStyled(fmt.Sprintf("  Provider:    %s", mdl.cfg.LLM.Provider), node.Color(7), 0, 0),
			node.TextStyled(fmt.Sprintf("  Model:       %s", mdl.cfg.LLM.Model), node.Color(7), 0, 0),
		}
		if mdl.cfg.LLM.Provider == "anthropic" {
			keyStatus := "from environment"
			if mdl.cfg.LLM.APIKey != "" {
				keyStatus = "configured"
			} else if !mdl.envKeyFound {
				keyStatus = "not set (use ANTHROPIC_API_KEY env var)"
			}
			lines = append(lines, node.TextStyled(fmt.Sprintf("  API Key:     %s", keyStatus), node.Color(7), 0, 0))
		}
		if mdl.cfg.LLM.Provider == "ollama" {
			lines = append(lines, node.TextStyled(fmt.Sprintf("  Ollama URL:  %s", mdl.cfg.LLM.BaseURL), node.Color(7), 0, 0))
		}
		embStatus := "disabled"
		if mdl.cfg.Embeddings.Enabled {
			embStatus = fmt.Sprintf("enabled (%s)", mdl.cfg.Embeddings.Model)
		}
		lines = append(lines,
			node.TextStyled(fmt.Sprintf("  Embeddings:  %s", embStatus), node.Color(7), 0, 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Enter to save • Esc to go back", node.Color(3), 0, 0),
			node.Text(""),
		)
		content = node.Column(lines...)

	case screenDone:
		path, _ := DefaultPath()
		var statusLine node.Node
		if mdl.err != nil {
			statusLine = node.TextStyled(fmt.Sprintf("  ✗ Error: %v", mdl.err), node.Color(1), 0, 0)
		} else {
			statusLine = node.TextStyled(fmt.Sprintf("  ✓ Saved to %s", path), node.Color(2), 0, 0)
		}
		content = node.Column(
			node.Text(""),
			node.TextStyled("  All Set!", node.Color(2), 0, node.Bold),
			node.Text(""),
			statusLine,
			node.Text(""),
			node.TextStyled("  Get started:", node.Color(7), 0, 0),
			node.Text(""),
			node.TextStyled("    botmem block set human \"Your name here\"", node.Color(6), 0, 0),
			node.TextStyled("    botmem context", node.Color(6), 0, 0),
			node.TextStyled("    botmem ingest \"conversation text...\"", node.Color(6), 0, 0),
			node.Text(""),
			node.Spacer(),
			node.TextStyled("  Press Enter to exit", node.Color(8), 0, 0),
			node.Text(""),
		)

	default:
		content = node.Text("Unknown screen")
	}

	return node.Column(
		node.Text(""),
		title,
		divider,
		content,
	).WithFlex(1)
}
