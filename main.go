package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/stukennedy/botmem/internal/config"
	botmemctx "github.com/stukennedy/botmem/internal/context"
	"github.com/stukennedy/botmem/internal/db"
	"github.com/stukennedy/botmem/internal/embeddings"
	"github.com/stukennedy/botmem/internal/ingest"
	"github.com/stukennedy/botmem/internal/memory"

	"github.com/spf13/cobra"
)

var dbPath string

func main() {
	root := &cobra.Command{
		Use:   "botmem",
		Short: "Local LLM memory system",
	}
	root.PersistentFlags().StringVar(&dbPath, "db", "", "database path (default: ~/.botmem/botmem.db)")

	root.AddCommand(initCmd(), blockCmd(), archiveCmd(), graphCmd(), summaryCmd(), contextCmd(), ingestCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func blockCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "block", Short: "Manage memory blocks"}

	cmd.AddCommand(&cobra.Command{
		Use:   "list [type]",
		Short: "List memory blocks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			blockType := ""
			if len(args) > 0 {
				blockType = args[0]
			}
			blocks, err := memory.NewBlockStore(database).List(blockType)
			if err != nil {
				return err
			}
			for _, b := range blocks {
				fmt.Printf("[%s] %s (%s)\n", b.BlockType, b.Label, b.UpdatedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <label>",
		Short: "Get a memory block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			b, err := memory.NewBlockStore(database).GetByLabel(args[0])
			if err != nil {
				return err
			}
			fmt.Println(b.Content)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <label> <content>",
		Short: "Set/update a memory block",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			store := memory.NewBlockStore(database)
			// Try update first, create if not exists
			if _, err := store.GetByLabel(args[0]); err != nil {
				_, err = store.Create(args[0], "core", args[1])
			} else {
				_, err = store.Update(args[0], args[1])
			}
			if err != nil {
				return err
			}
			fmt.Printf("Block %q updated.\n", args[0])
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create <label> <type> [content]",
		Short: "Create a new memory block",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			content := ""
			if len(args) > 2 {
				content = args[2]
			}
			b, err := memory.NewBlockStore(database).Create(args[0], args[1], content)
			if err != nil {
				return err
			}
			fmt.Printf("Created block %q (id=%d)\n", b.Label, b.ID)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <label>",
		Short: "Delete a memory block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()
			return memory.NewBlockStore(database).Delete(args[0])
		},
	})

	return cmd
}

func archiveCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "archive", Short: "Manage archival memory"}

	cmd.AddCommand(&cobra.Command{
		Use:   "add <text> [--tags tag1,tag2]",
		Short: "Add an archival entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			tagsFlag, _ := cmd.Flags().GetString("tags")
			var tags []string
			if tagsFlag != "" {
				tags = strings.Split(tagsFlag, ",")
			}

			e, err := memory.NewArchivalStore(database).Add(args[0], tags, nil)
			if err != nil {
				return err
			}
			fmt.Printf("Added archival entry (id=%d)\n", e.ID)
			return nil
		},
	})
	cmd.Commands()[0].Flags().String("tags", "", "comma-separated tags")

	cmd.AddCommand(&cobra.Command{
		Use:   "search <query>",
		Short: "Search archival memory (full-text)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			entries, err := memory.NewArchivalStore(database).Search(args[0], 10)
			if err != nil {
				return err
			}
			for _, e := range entries {
				fmt.Printf("[%d] %s (tags: %s)\n", e.ID, e.Content, e.Tags)
			}
			if len(entries) == 0 {
				fmt.Println("No results.")
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List archival entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			tag, _ := cmd.Flags().GetString("tag")
			entries, err := memory.NewArchivalStore(database).List(tag, 50)
			if err != nil {
				return err
			}
			for _, e := range entries {
				fmt.Printf("[%d] %s (tags: %s)\n", e.ID, truncate(e.Content, 80), e.Tags)
			}
			return nil
		},
	})
	cmd.Commands()[1].Flags().String("tag", "", "filter by tag")

	return cmd
}

func graphCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "graph", Short: "Knowledge graph operations"}

	cmd.AddCommand(&cobra.Command{
		Use:   "add <subject> <predicate> <object>",
		Short: "Add a relationship triplet",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			if err := memory.NewGraphStore(database).AddRelation(args[0], args[1], args[2], ""); err != nil {
				return err
			}
			fmt.Printf("Added: %s -[%s]-> %s\n", args[0], args[1], args[2])
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "query <entity>",
		Short: "Query relations for an entity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			rels, err := memory.NewGraphStore(database).QueryEntity(args[0])
			if err != nil {
				return err
			}
			for _, r := range rels {
				fmt.Printf("%s -[%s]-> %s\n", r.Subject, r.Predicate, r.Object)
			}
			if len(rels) == 0 {
				fmt.Println("No relations found.")
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "search <predicate>",
		Short: "Search relations by predicate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			rels, err := memory.NewGraphStore(database).SearchRelations(args[0])
			if err != nil {
				return err
			}
			for _, r := range rels {
				fmt.Printf("%s -[%s]-> %s\n", r.Subject, r.Predicate, r.Object)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "entities [type]",
		Short: "List entities",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			entityType := ""
			if len(args) > 0 {
				entityType = args[0]
			}
			entities, err := memory.NewGraphStore(database).ListEntities(entityType)
			if err != nil {
				return err
			}
			for _, e := range entities {
				fmt.Printf("%s (%s)\n", e.Name, e.EntityType)
			}
			return nil
		},
	})

	return cmd
}

func summaryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "summary", Short: "Conversation summaries"}

	cmd.AddCommand(&cobra.Command{
		Use:   "add <text>",
		Short: "Add a conversation summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			level, _ := cmd.Flags().GetInt("level")
			s, err := memory.NewSummaryStore(database).Add(level, args[0], "")
			if err != nil {
				return err
			}
			fmt.Printf("Added summary (id=%d, level=%d)\n", s.ID, s.Level)
			return nil
		},
	})
	cmd.Commands()[0].Flags().Int("level", 0, "summary level")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List summaries",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			level, _ := cmd.Flags().GetInt("level")
			summaries, err := memory.NewSummaryStore(database).List(level, 20)
			if err != nil {
				return err
			}
			for _, s := range summaries {
				fmt.Printf("[L%d #%d] %s\n", s.Level, s.ID, truncate(s.Content, 100))
			}
			return nil
		},
	})
	cmd.Commands()[1].Flags().Int("level", 0, "summary level")

	return cmd
}

func contextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "context",
		Short: "Dump full context payload for LLM injection",
		RunE: func(cmd *cobra.Command, args []string) error {
			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			payload, err := botmemctx.Build(database)
			if err != nil {
				return err
			}
			out, err := payload.JSON()
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up botmem — configure LLM provider, API keys, and embeddings",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.RunInitTUI()
			return err
		},
	}
}

func loadIngestConfig() (*ingest.Config, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}

	var embedProv embeddings.Provider
	if cfg.Embeddings.Enabled {
		embedProv = embeddings.NewOllamaProvider(cfg.Embeddings.BaseURL, cfg.Embeddings.Model)
	}

	return ingest.ConfigFromAppConfig(
		cfg.LLM.Provider,
		cfg.LLM.Model,
		cfg.LLM.APIKey,
		cfg.LLM.BaseURL,
		embedProv,
	), nil
}

func ingestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest <text>",
		Short: "Ingest conversation text — auto-extracts memories, facts, and relations",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadIngestConfig()
			if err != nil {
				return err
			}

			database, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			defer database.Close()

			var text string
			if len(args) > 0 {
				text = args[0]
			} else {
				buf, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				text = string(buf)
			}

			if text == "" {
				return fmt.Errorf("no text provided")
			}

			result, err := ingest.Run(database, text, cfg)
			if err != nil {
				return err
			}

			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
	return cmd
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

