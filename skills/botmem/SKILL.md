---
name: botmem
description: Persistent structured memory for LLM agents using botmem CLI. Use when storing, retrieving, or managing agent memory — facts, knowledge graph relationships, conversation summaries, and context blocks. Use for memory recall, ingestion of conversation content, knowledge graph queries, and exporting full context for prompt injection.
---

# botmem — Persistent LLM Memory

CLI tool providing four memory types: blocks (working memory), archival (long-term facts), knowledge graph (entity relationships), and conversation summaries.

## Prerequisites

- `botmem` binary on PATH (`go install github.com/stukennedy/botmem@latest`)
- Configured via `botmem init` (supports Claude Code CLI, Anthropic API, or Ollama)
- Config at `~/.botmem/config.yaml`, DB at `~/.botmem/botmem.db`

## Commands

### Memory Blocks (working memory — always-on context)
```bash
botmem block set <label> <content>    # Set/update a block (human, persona, context)
botmem block get <label>              # Read a block
botmem block list [type]              # List blocks
botmem block delete <label>           # Delete a block
```

### Archival Memory (long-term facts with FTS5 search)
```bash
botmem archive add <text> --tags tag1,tag2   # Store a fact
botmem archive search <query>                 # Full-text search
botmem archive list [--tag tag]               # List entries
```

### Knowledge Graph (entity-relationship triplets)
```bash
botmem graph add <subject> <predicate> <object>   # Add relationship
botmem graph query <entity>                        # All relations for entity
botmem graph search <predicate>                    # Search by relationship type
botmem graph entities [type]                       # List entities
```

### Conversation Summaries (hierarchical)
```bash
botmem summary add <text> [--level N]   # Add summary (level 0 = most detailed)
botmem summary list [--level N]         # List summaries
```

### Context Export (full memory dump for prompt injection)
```bash
botmem context   # Returns JSON: { core_blocks, key_relations, ... }
```

### Ingest (LLM-powered extraction from conversation text)
```bash
botmem ingest <text>       # Extract facts, triplets, block updates, summary
echo <text> | botmem ingest   # Pipe from stdin
```
Ingest requires a configured LLM provider. It automatically:
- Updates memory blocks (human, persona, context)
- Extracts tagged facts → archival
- Extracts entity-relationship triplets → knowledge graph
- Generates conversation summary

## Integration Patterns

### Session Start — Load Context
Run `botmem context` and include the JSON in system prompt for full memory recall.

### After Important Conversations — Ingest
Summarise the conversation and pipe to `botmem ingest` to automatically extract and store structured memories.

### Ad-hoc Recall — Query
Use `botmem graph query <entity>` or `botmem archive search <term>` for targeted recall.

### Periodic Maintenance
Use `botmem block set context <current situation>` to keep working memory current.

## Custom DB Path
All commands accept `--db <path>` to use a different database file. Useful for per-agent or per-project memory stores.
