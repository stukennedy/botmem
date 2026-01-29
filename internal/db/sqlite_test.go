package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_CreatesDBAndDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "botmem.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("db file not created")
	}
}

func TestOpen_DefaultPath(t *testing.T) {
	// Just ensure it doesn't panic with empty path — it'll use ~/.botmem/botmem.db
	// We skip actual creation to avoid writing to real home dir in CI
	t.Skip("skipped: would write to real home directory")
}

func TestMigrations_Idempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Open twice — migrations should be idempotent (CREATE IF NOT EXISTS)
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db2.Close()
}

func TestMigrations_AllTablesExist(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	tables := []string{"memory_blocks", "archival", "entities", "relations", "conversation_summaries"}
	for _, table := range tables {
		var name string
		err := database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// Check FTS virtual table
	var name string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='archival_fts'").Scan(&name)
	if err != nil {
		t.Error("FTS table archival_fts not found")
	}
}

func TestWALMode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	var mode string
	if err := database.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected WAL mode, got %q", mode)
	}
}

func TestForeignKeysEnabled(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	var fk int
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fk)
	}
}
