package memory

import (
	"database/sql"
	"fmt"
	"time"
)

type Block struct {
	ID        int64     `json:"id"`
	Label     string    `json:"label"`
	BlockType string    `json:"block_type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BlockStore struct {
	db *sql.DB
}

func NewBlockStore(db *sql.DB) *BlockStore {
	return &BlockStore{db: db}
}

func (s *BlockStore) Create(label, blockType, content string) (*Block, error) {
	if blockType == "" {
		blockType = "core"
	}
	res, err := s.db.Exec(
		`INSERT INTO memory_blocks (label, block_type, content) VALUES (?, ?, ?)`,
		label, blockType, content,
	)
	if err != nil {
		return nil, fmt.Errorf("create block: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *BlockStore) GetByLabel(label string) (*Block, error) {
	b := &Block{}
	err := s.db.QueryRow(
		`SELECT id, label, block_type, content, created_at, updated_at FROM memory_blocks WHERE label = ?`, label,
	).Scan(&b.ID, &b.Label, &b.BlockType, &b.Content, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get block %q: %w", label, err)
	}
	return b, nil
}

func (s *BlockStore) GetByID(id int64) (*Block, error) {
	b := &Block{}
	err := s.db.QueryRow(
		`SELECT id, label, block_type, content, created_at, updated_at FROM memory_blocks WHERE id = ?`, id,
	).Scan(&b.ID, &b.Label, &b.BlockType, &b.Content, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get block %d: %w", id, err)
	}
	return b, nil
}

func (s *BlockStore) Update(label, content string) (*Block, error) {
	_, err := s.db.Exec(
		`UPDATE memory_blocks SET content = ?, updated_at = CURRENT_TIMESTAMP WHERE label = ?`,
		content, label,
	)
	if err != nil {
		return nil, fmt.Errorf("update block %q: %w", label, err)
	}
	return s.GetByLabel(label)
}

func (s *BlockStore) Delete(label string) error {
	_, err := s.db.Exec(`DELETE FROM memory_blocks WHERE label = ?`, label)
	return err
}

func (s *BlockStore) List(blockType string) ([]*Block, error) {
	query := `SELECT id, label, block_type, content, created_at, updated_at FROM memory_blocks`
	var args []any
	if blockType != "" {
		query += ` WHERE block_type = ?`
		args = append(args, blockType)
	}
	query += ` ORDER BY label`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list blocks: %w", err)
	}
	defer rows.Close()

	var blocks []*Block
	for rows.Next() {
		b := &Block{}
		if err := rows.Scan(&b.ID, &b.Label, &b.BlockType, &b.Content, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}
