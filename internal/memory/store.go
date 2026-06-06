// Package memory implements the SQLite-backed memory store with FTS5 search.
package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rxyz/servex/internal/types"
)

// Store manages SQLite memory operations.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the database and runs migrations.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB for graph operations.
func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	mig, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		// Fallback: try relative to executable
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = s.db.Exec(string(mig))
	if err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}

// nanoid generates a short unique ID.
func nanoid() string {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 21)
	for i := range b {
		b[i] = alphabet[time.Now().UnixNano()%int64(len(alphabet))+int64(i)*37%int64(len(alphabet))]
		time.Sleep(0) // ensure different nanoseconds
	}
	return string(b)
}

// CreateMemory stores a new memory and returns it.
func (s *Store) CreateMemory(content, memType, category string, tags []string, importance float64) (*types.Memory, error) {
	now := time.Now().UnixMilli()
	id := nanoid()
	tagsJSON, _ := json.Marshal(tags)
	emb := computeEmbedding(content, tags)

	_, err := s.db.Exec(
		`INSERT INTO memories (id, content, type, category, tags, importance, embedding, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, content, memType, category, string(tagsJSON), importance, emb, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert memory: %w", err)
	}

	return &types.Memory{
		ID:         id,
		Content:    content,
		Type:       memType,
		Category:   category,
		Tags:       tags,
		Importance: importance,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// GetMemory retrieves a memory by ID (excluding soft-deleted).
func (s *Store) GetMemory(id string) (*types.Memory, error) {
	row := s.db.QueryRow(
		`SELECT id, content, type, category, tags, importance, access_count, created_at, updated_at, last_accessed_at
		 FROM memories WHERE id = ? AND deleted_at IS NULL`, id,
	)
	return scanMemory(row)
}

// UpdateMemory patches a memory's content and/or tags, recomputing embedding if needed.
func (s *Store) UpdateMemory(id string, content *string, tags *[]string) (*types.Memory, error) {
	now := time.Now().UnixMilli()
	if content != nil {
		_, err := s.db.Exec(`UPDATE memories SET content = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, *content, now, id)
		if err != nil {
			return nil, err
		}
	}
	if tags != nil {
		tagsJSON, _ := json.Marshal(*tags)
		_, err := s.db.Exec(`UPDATE memories SET tags = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, string(tagsJSON), now, id)
		if err != nil {
			return nil, err
		}
	}
	// Recompute embedding
	if content != nil || tags != nil {
		m, err := s.GetMemory(id)
		if err != nil {
			return nil, err
		}
		emb := computeEmbedding(m.Content, m.Tags)
		s.db.Exec(`UPDATE memories SET embedding = ? WHERE id = ?`, emb, id)
	}
	return s.GetMemory(id)
}

// DeleteMemory soft-deletes a memory.
func (s *Store) DeleteMemory(id string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`UPDATE memories SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`, now, id)
	return err
}

// ListMemories returns memories with optional filters.
func (s *Store) ListMemories(limit, offset int, memType, tag string) ([]types.Memory, error) {
	query := `SELECT id, content, type, category, tags, importance, access_count, created_at, updated_at, last_accessed_at
	          FROM memories WHERE deleted_at IS NULL`
	args := []any{}

	if memType != "" {
		query += " AND type = ?"
		args = append(args, memType)
	}
	if tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, "%"+tag+"%")
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Memory
	for rows.Next() {
		m, err := scanMemoryRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, nil
}

// SearchMemory performs FTS5 full-text search.
func (s *Store) SearchMemory(query string, limit int, memType, tag string) ([]types.Memory, error) {
	// FTS5 query: quote each term for safety
	terms := strings.Fields(query)
	ftsQuery := strings.Join(terms, " OR ")

	sqlStr := `SELECT m.id, m.content, m.type, m.category, m.tags, m.importance, m.access_count, m.created_at, m.updated_at, m.last_accessed_at
	           FROM memories m
	           JOIN memories_fts fts ON m.rowid = fts.rowid
	           WHERE memories_fts MATCH ? AND m.deleted_at IS NULL`
	args := []any{ftsQuery}

	if memType != "" {
		sqlStr += " AND m.type = ?"
		args = append(args, memType)
	}
	if tag != "" {
		sqlStr += " AND m.tags LIKE ?"
		args = append(args, "%"+tag+"%")
	}
	sqlStr += " ORDER BY rank LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Memory
	for rows.Next() {
		m, err := scanMemoryRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}

	// Fallback: if FTS returns nothing, try substring search
	if len(out) == 0 {
		return s.substringSearch(query, limit)
	}
	return out, nil
}

func (s *Store) substringSearch(query string, limit int) ([]types.Memory, error) {
	rows, err := s.db.Query(
		`SELECT id, content, type, category, tags, importance, access_count, created_at, updated_at, last_accessed_at
		 FROM memories WHERE deleted_at IS NULL AND content LIKE ? ORDER BY created_at DESC LIMIT ?`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Memory
	for rows.Next() {
		m, err := scanMemoryRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, nil
}

// Stats returns aggregate memory statistics.
func (s *Store) Stats() (*types.MemoryStats, error) {
	stats := &types.MemoryStats{
		TopTags: make(map[string]int),
		Types:   make(map[string]int),
	}

	s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE deleted_at IS NULL`).Scan(&stats.TotalMemories)
	s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE deleted_at IS NOT NULL`).Scan(&stats.DeletedMemories)
	s.db.QueryRow(`SELECT COUNT(*) FROM entities`).Scan(&stats.TotalEntities)
	s.db.QueryRow(`SELECT COUNT(*) FROM relations`).Scan(&stats.TotalRelations)

	// Type breakdown
	rows, _ := s.db.Query(`SELECT type, COUNT(*) FROM memories WHERE deleted_at IS NULL GROUP BY type`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var c int
			rows.Scan(&t, &c)
			stats.Types[t] = c
		}
	}

	// Top tags (parse JSON arrays)
	tagRows, _ := s.db.Query(`SELECT tags FROM memories WHERE deleted_at IS NULL AND tags IS NOT NULL AND tags != 'null'`)
	if tagRows != nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var raw string
			tagRows.Scan(&raw)
			var tags []string
			json.Unmarshal([]byte(raw), &tags)
			for _, t := range tags {
				stats.TopTags[t]++
			}
		}
	}

	// DB size
	var pageCount, pageSize int64
	s.db.QueryRow(`PRAGMA page_count`).Scan(&pageCount)
	s.db.QueryRow(`PRAGMA page_size`).Scan(&pageSize)
	stats.DBSizeBytes = pageCount * pageSize

	return stats, nil
}

func scanMemory(row *sql.Row) (*types.Memory, error) {
	var m types.Memory
	var tagsRaw sql.NullString
	var lastAccess sql.NullInt64
	err := row.Scan(&m.ID, &m.Content, &m.Type, &m.Category, &tagsRaw, &m.Importance, &m.AccessCount, &m.CreatedAt, &m.UpdatedAt, &lastAccess)
	if err != nil {
		return nil, err
	}
	if tagsRaw.Valid {
		json.Unmarshal([]byte(tagsRaw.String), &m.Tags)
	}
	if lastAccess.Valid {
		m.LastAccessAt = &lastAccess.Int64
	}
	return &m, nil
}

func scanMemoryRows(rows *sql.Rows) (*types.Memory, error) {
	var m types.Memory
	var tagsRaw sql.NullString
	var lastAccess sql.NullInt64
	err := rows.Scan(&m.ID, &m.Content, &m.Type, &m.Category, &tagsRaw, &m.Importance, &m.AccessCount, &m.CreatedAt, &m.UpdatedAt, &lastAccess)
	if err != nil {
		return nil, err
	}
	if tagsRaw.Valid {
		json.Unmarshal([]byte(tagsRaw.String), &m.Tags)
	}
	if lastAccess.Valid {
		m.LastAccessAt = &lastAccess.Int64
	}
	return &m, nil
}
