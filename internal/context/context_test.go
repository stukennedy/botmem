package context

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stukennedy/botmem/internal/db"
	"github.com/stukennedy/botmem/internal/memory"
)

func testSetup(t *testing.T) (*memory.BlockStore, *memory.GraphStore, *memory.SummaryStore, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return memory.NewBlockStore(database),
		memory.NewGraphStore(database),
		memory.NewSummaryStore(database),
		dbPath
}

func TestBuild_Empty(t *testing.T) {
	_, _, _, dbPath := testSetup(t)
	database, _ := db.Open(dbPath)
	defer database.Close()

	payload, err := Build(database)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(payload.CoreBlocks) != 0 {
		t.Errorf("expected 0 core blocks, got %d", len(payload.CoreBlocks))
	}
}

func TestBuild_WithData(t *testing.T) {
	blocks, graph, summaries, dbPath := testSetup(t)

	blocks.Create("human", "core", "Stuart")
	blocks.Create("persona", "core", "Moltbot")
	blocks.Create("notes", "archival", "should not appear in core")
	graph.AddRelation("Stuart", "works_on", "Moltbot", "")
	summaries.Add(0, "Test conversation", "")

	database, _ := db.Open(dbPath)
	defer database.Close()

	payload, err := Build(database)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if len(payload.CoreBlocks) != 2 {
		t.Errorf("expected 2 core blocks (not archival), got %d", len(payload.CoreBlocks))
	}
	if len(payload.Graph) != 1 {
		t.Errorf("expected 1 relation, got %d", len(payload.Graph))
	}
	if len(payload.Summaries) != 1 {
		t.Errorf("expected 1 summary, got %d", len(payload.Summaries))
	}
}

func TestPayload_JSON(t *testing.T) {
	p := &Payload{
		CoreBlocks: []*memory.Block{{Label: "test", Content: "hello"}},
	}
	out, err := p.JSON()
	if err != nil {
		t.Fatalf("json: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := parsed["core_blocks"]; !ok {
		t.Error("missing core_blocks key")
	}
}
