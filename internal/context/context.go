package context

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/stukennedy/botmem/internal/memory"
)

// Payload is the structured context returned to an LLM.
type Payload struct {
	CoreBlocks []*memory.Block    `json:"core_blocks"`
	Summaries  []*memory.Summary  `json:"recent_summaries,omitempty"`
	Graph      []*memory.Relation `json:"key_relations,omitempty"`
}

// Build assembles the full context payload from all memory stores.
func Build(db *sql.DB) (*Payload, error) {
	blocks := memory.NewBlockStore(db)
	summaries := memory.NewSummaryStore(db)
	graph := memory.NewGraphStore(db)

	coreBlocks, err := blocks.List("core")
	if err != nil {
		return nil, fmt.Errorf("load core blocks: %w", err)
	}

	// Get most recent L0 summaries
	recentSummaries, err := summaries.List(0, 5)
	if err != nil {
		return nil, fmt.Errorf("load summaries: %w", err)
	}

	// Get all relations (for small graphs; paginate later if needed)
	var allRelations []*memory.Relation
	entities, err := graph.ListEntities("")
	if err != nil {
		return nil, fmt.Errorf("load entities: %w", err)
	}
	seen := map[int64]bool{}
	for _, e := range entities {
		rels, err := graph.QueryEntity(e.Name)
		if err != nil {
			continue
		}
		for _, r := range rels {
			if !seen[r.ID] {
				seen[r.ID] = true
				allRelations = append(allRelations, r)
			}
		}
	}

	return &Payload{
		CoreBlocks: coreBlocks,
		Summaries:  recentSummaries,
		Graph:      allRelations,
	}, nil
}

// JSON returns the payload as formatted JSON.
func (p *Payload) JSON() (string, error) {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
