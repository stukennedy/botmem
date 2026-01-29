package memory

import (
	"path/filepath"
	"testing"

	"github.com/stukennedy/botmem/internal/db"
)

func testArchivalStore(t *testing.T) *ArchivalStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewArchivalStore(database)
}

func TestArchivalAdd(t *testing.T) {
	store := testArchivalStore(t)
	e, err := store.Add("Go is great for CLIs", []string{"tech", "opinion"}, nil)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if e.Content != "Go is great for CLIs" {
		t.Errorf("unexpected content: %q", e.Content)
	}
	if e.Tags != "tech,opinion" {
		t.Errorf("unexpected tags: %q", e.Tags)
	}
}

func TestArchivalAdd_EmptyTags(t *testing.T) {
	store := testArchivalStore(t)
	e, err := store.Add("some fact", nil, nil)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if e.Tags != "" {
		t.Errorf("expected empty tags, got %q", e.Tags)
	}
}

func TestArchivalAdd_WithEmbedding(t *testing.T) {
	store := testArchivalStore(t)
	emb := []byte{1, 2, 3, 4}
	e, err := store.Add("embedded fact", nil, emb)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if len(e.Embedding) != 4 {
		t.Errorf("expected 4-byte embedding, got %d", len(e.Embedding))
	}
}

func TestArchivalSearch_FTS(t *testing.T) {
	store := testArchivalStore(t)
	store.Add("Stuart prefers Go for CLI tools", []string{"preference"}, nil)
	store.Add("Python is good for ML", []string{"tech"}, nil)
	store.Add("Rust has great memory safety", []string{"tech"}, nil)

	results, err := store.Search("Go CLI", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Content != "Stuart prefers Go for CLI tools" {
		t.Errorf("unexpected first result: %q", results[0].Content)
	}
}

func TestArchivalSearch_NoResults(t *testing.T) {
	store := testArchivalStore(t)
	store.Add("something unrelated", nil, nil)

	results, err := store.Search("quantum computing", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestArchivalSearch_DefaultLimit(t *testing.T) {
	store := testArchivalStore(t)
	for i := 0; i < 20; i++ {
		store.Add("matching content about Go programming", nil, nil)
	}

	results, err := store.Search("Go programming", 0) // 0 should default to 10
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) > 10 {
		t.Errorf("expected max 10 results with default limit, got %d", len(results))
	}
}

func TestArchivalList(t *testing.T) {
	store := testArchivalStore(t)
	store.Add("fact 1", []string{"a"}, nil)
	store.Add("fact 2", []string{"b"}, nil)
	store.Add("fact 3", []string{"a", "b"}, nil)

	all, err := store.List("", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}

	tagged, err := store.List("a", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged) != 2 {
		t.Errorf("expected 2 with tag 'a', got %d", len(tagged))
	}
}

func TestArchivalDelete(t *testing.T) {
	store := testArchivalStore(t)
	e, _ := store.Add("to delete", nil, nil)

	if err := store.Delete(e.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := store.GetByID(e.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestArchivalAllWithEmbeddings(t *testing.T) {
	store := testArchivalStore(t)
	store.Add("no embedding", nil, nil)
	store.Add("has embedding", nil, []byte{1, 2, 3, 4})

	entries, err := store.AllWithEmbeddings()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry with embedding, got %d", len(entries))
	}
}
