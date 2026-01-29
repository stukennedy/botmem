# botmem

Persistent structured memory for LLM agents. Four memory types, one SQLite file, zero cloud dependencies.

**[botmem.dev](https://botmem.dev)**

## Install

```bash
go install github.com/stukennedy/botmem@latest
botmem init
```

## Memory Types

| Type | Purpose | Example |
|------|---------|---------|
| **Blocks** | Always-on working memory | User preferences, persona, current context |
| **Archival** | Long-term facts with FTS5 search | "Stu has a PhD in Mathematics from Strathclyde" |
| **Knowledge Graph** | Entity-relationship triplets | `Stu -[co-founded]-> Fluxwise AI` |
| **Summaries** | Hierarchical conversation compression | Multi-level overviews of past conversations |

## Quick Start

```bash
# Set up
botmem init

# Store some memories
botmem block set human "Stu Kennedy — senior AI engineer from Scotland"
botmem archive add "Prefers Outside IR35 contracts" --tags "work,preferences"
botmem graph add "Stu" "built" "Layercode voice platform"

# Query
botmem graph query "Stu"
botmem archive search "preferences"

# Export full context for LLM injection
botmem context
```

## LLM-Powered Ingestion

Feed conversation text and botmem automatically extracts structured memories:

```bash
botmem ingest "Had a meeting with Chris about the Fluxwise roadmap. 
Decided to focus on voice AI products first, targeting healthcare sector."
```

This extracts:
- **Block updates** — updates working memory with current context
- **Facts** — tagged archival entries
- **Triplets** — knowledge graph relationships
- **Summary** — conversation overview

## Providers

| Provider | Setup | Notes |
|----------|-------|-------|
| **Claude Code** | `claude /login` | Uses `claude -p` — no API key in config |
| **Anthropic API** | Set `ANTHROPIC_API_KEY` | Direct API access |
| **Ollama** | Local models | Fully offline, supports embeddings |

## Agent Integration

### Moltbot / Clawdbot Skill

A packaged skill is available in `skills/botmem/` and `dist/botmem.skill` for Moltbot/Clawdbot agents.

### Claude Code

Add to your `CLAUDE.md`:
```markdown
## Memory
Use botmem for persistent memory:
- `botmem context` — load full memory at session start
- `botmem ingest "<text>"` — store memories after conversations
- `botmem graph query "<entity>"` — recall relationships
- `botmem archive search "<term>"` — search facts
```

### Any LLM Agent

```bash
# Inject memory into system prompt
MEMORY=$(botmem context)
# Pass $MEMORY as part of your system prompt
```

## Storage

Everything lives in `~/.botmem/`:
- `config.yaml` — provider settings
- `botmem.db` — SQLite database (FTS5 enabled)

Use `--db <path>` for per-project memory stores.

## License

MIT
