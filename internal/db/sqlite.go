package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Open opens or creates the botmem database at the default location (~/.botmem/botmem.db).
// If dbPath is empty, the default path is used.
func Open(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dbPath = filepath.Join(home, ".botmem", "botmem.db")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS memory_blocks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT NOT NULL UNIQUE,
			block_type TEXT NOT NULL DEFAULT 'core',
			content TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS archival (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '',
			embedding BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// FTS5 virtual table for full-text search on archival content
		`CREATE VIRTUAL TABLE IF NOT EXISTS archival_fts USING fts5(
			content,
			tags,
			content='archival',
			content_rowid='id'
		)`,

		// Triggers to keep FTS in sync
		`CREATE TRIGGER IF NOT EXISTS archival_ai AFTER INSERT ON archival BEGIN
			INSERT INTO archival_fts(rowid, content, tags) VALUES (new.id, new.content, new.tags);
		END`,

		`CREATE TRIGGER IF NOT EXISTS archival_ad AFTER DELETE ON archival BEGIN
			INSERT INTO archival_fts(archival_fts, rowid, content, tags) VALUES('delete', old.id, old.content, old.tags);
		END`,

		`CREATE TRIGGER IF NOT EXISTS archival_au AFTER UPDATE ON archival BEGIN
			INSERT INTO archival_fts(archival_fts, rowid, content, tags) VALUES('delete', old.id, old.content, old.tags);
			INSERT INTO archival_fts(rowid, content, tags) VALUES (new.id, new.content, new.tags);
		END`,

		// Knowledge graph
		`CREATE TABLE IF NOT EXISTS entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			entity_type TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			subject_id INTEGER NOT NULL REFERENCES entities(id),
			predicate TEXT NOT NULL,
			object_id INTEGER NOT NULL REFERENCES entities(id),
			metadata TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(subject_id, predicate, object_id)
		)`,

		// Conversation summaries
		`CREATE TABLE IF NOT EXISTS conversation_summaries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level INTEGER NOT NULL DEFAULT 0,
			content TEXT NOT NULL,
			source_ids TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, m)
		}
	}
	return nil
}
