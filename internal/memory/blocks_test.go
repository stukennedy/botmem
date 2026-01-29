package memory

import (
	"path/filepath"
	"testing"

	"github.com/stukennedy/botmem/internal/db"
)

func testDB(t *testing.T) *BlockStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewBlockStore(database)
}

func TestBlockCreate(t *testing.T) {
	store := testDB(t)
	b, err := store.Create("human", "core", "Stuart Kennedy")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if b.Label != "human" || b.Content != "Stuart Kennedy" || b.BlockType != "core" {
		t.Errorf("unexpected block: %+v", b)
	}
	if b.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestBlockCreate_DefaultType(t *testing.T) {
	store := testDB(t)
	b, err := store.Create("test", "", "content")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if b.BlockType != "core" {
		t.Errorf("expected default type 'core', got %q", b.BlockType)
	}
}

func TestBlockCreate_DuplicateLabel(t *testing.T) {
	store := testDB(t)
	if _, err := store.Create("human", "core", "v1"); err != nil {
		t.Fatal(err)
	}
	_, err := store.Create("human", "core", "v2")
	if err == nil {
		t.Error("expected error on duplicate label")
	}
}

func TestBlockGetByLabel(t *testing.T) {
	store := testDB(t)
	store.Create("persona", "core", "helpful bot")

	b, err := store.GetByLabel("persona")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if b.Content != "helpful bot" {
		t.Errorf("expected 'helpful bot', got %q", b.Content)
	}
}

func TestBlockGetByLabel_NotFound(t *testing.T) {
	store := testDB(t)
	_, err := store.GetByLabel("nonexistent")
	if err == nil {
		t.Error("expected error for missing label")
	}
}

func TestBlockUpdate(t *testing.T) {
	store := testDB(t)
	store.Create("human", "core", "original")

	b, err := store.Update("human", "updated content")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if b.Content != "updated content" {
		t.Errorf("expected 'updated content', got %q", b.Content)
	}
}

func TestBlockUpdate_EmptyContent(t *testing.T) {
	store := testDB(t)
	store.Create("test", "core", "something")

	b, err := store.Update("test", "")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if b.Content != "" {
		t.Errorf("expected empty content, got %q", b.Content)
	}
}

func TestBlockDelete(t *testing.T) {
	store := testDB(t)
	store.Create("todelete", "core", "bye")

	if err := store.Delete("todelete"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := store.GetByLabel("todelete")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestBlockDelete_NonExistent(t *testing.T) {
	store := testDB(t)
	// Should not error — DELETE WHERE on missing row is fine
	if err := store.Delete("nope"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBlockList(t *testing.T) {
	store := testDB(t)
	store.Create("a", "core", "1")
	store.Create("b", "archival", "2")
	store.Create("c", "core", "3")

	all, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3, got %d", len(all))
	}

	core, err := store.List("core")
	if err != nil {
		t.Fatal(err)
	}
	if len(core) != 2 {
		t.Errorf("expected 2 core blocks, got %d", len(core))
	}
}

func TestBlockList_Empty(t *testing.T) {
	store := testDB(t)
	blocks, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Errorf("expected empty list, got %d", len(blocks))
	}
}

func TestBlockList_OrderedByLabel(t *testing.T) {
	store := testDB(t)
	store.Create("zebra", "core", "")
	store.Create("alpha", "core", "")
	store.Create("mid", "core", "")

	blocks, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if blocks[0].Label != "alpha" || blocks[1].Label != "mid" || blocks[2].Label != "zebra" {
		t.Errorf("not ordered by label: %s, %s, %s", blocks[0].Label, blocks[1].Label, blocks[2].Label)
	}
}
