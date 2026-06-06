// Package memory — hybrid search combining FTS5 BM25 + vector cosine similarity.
package memory

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/rxyz/servex/internal/embed"
	"github.com/rxyz/servex/internal/types"
)

// hybridResult pairs a memory with its combined score.
type hybridResult struct {
	memory types.Memory
	score  float64
}

// SearchMemoryHybrid combines FTS5 keyword search with vector cosine similarity
// using Reciprocal Rank Fusion (RRF).
//
// mode: "hybrid" (default), "keyword" (FTS5 only), "semantic" (vector only)
func (s *Store) SearchMemoryHybrid(query string, limit int, mode, memType, tag string) ([]types.Memory, error) {
	switch mode {
	case "keyword":
		return s.SearchMemory(query, limit, memType, tag)
	case "semantic":
		return s.vectorSearch(query, limit)
	default: // "hybrid"
		return s.hybridSearch(query, limit)
	}
}

// hybridSearch runs FTS5 and vector search in parallel, then fuses results with RRF.
func (s *Store) hybridSearch(query string, limit int) ([]types.Memory, error) {
	// FTS5 results (keyword precision)
	ftsResults, _ := s.SearchMemory(query, limit*2, "", "")
	// Vector results (semantic similarity)
	vecResults, _ := s.vectorSearch(query, limit*2)

	// RRF fusion: score = sum(1 / (k + rank)) for each method
	const k = 60 // RRF constant
	scores := make(map[string]float64)
	memLookup := make(map[string]types.Memory)

	// Rank FTS5 results
	for rank, m := range ftsResults {
		scores[m.ID] += 1.0 / (k + float64(rank+1))
		memLookup[m.ID] = m
	}

	// Rank vector results
	for rank, m := range vecResults {
		scores[m.ID] += 1.0 / (k + float64(rank+1))
		memLookup[m.ID] = m
	}

	// Sort by combined score
	combined := make([]hybridResult, 0, len(scores))
	for id, score := range scores {
		combined = append(combined, hybridResult{memory: memLookup[id], score: score})
	}
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].score > combined[j].score
	})

	// Take top N
	if len(combined) > limit {
		combined = combined[:limit]
	}

	out := make([]types.Memory, len(combined))
	for i, r := range combined {
		out[i] = r.memory
	}
	return out, nil
}

// vectorSearch computes cosine similarity between query and all stored embeddings.
func (s *Store) vectorSearch(query string, limit int) ([]types.Memory, error) {
	queryTokens := embed.Tokenize(query)
	queryVec := embed.Embed(queryTokens, nil)

	rows, err := s.db.Query(
		`SELECT id, content, type, category, tags, importance, access_count, created_at, updated_at, last_accessed_at, embedding
		 FROM memories WHERE deleted_at IS NULL AND embedding IS NOT NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		memory types.Memory
		sim    float32
	}
	var results []scored

	for rows.Next() {
		var m types.Memory
		var tagsRaw, embRaw []byte
		var lastAccess sqlNullInt64

		err := rows.Scan(
			&m.ID, &m.Content, &m.Type, &m.Category, &tagsRaw,
			&m.Importance, &m.AccessCount, &m.CreatedAt, &m.UpdatedAt, &lastAccess, &embRaw,
		)
		if err != nil {
			continue
		}
		if len(embRaw) == 0 {
			continue
		}

		memVec := embed.DeserializeVector(embRaw)
		sim := embed.CosineSimilarity(queryVec, memVec)
		if sim <= 0 {
			continue
		}

		if tagsRaw != nil {
			json.Unmarshal(tagsRaw, &m.Tags)
		}
		if lastAccess.Valid {
			m.LastAccessAt = &lastAccess.Int64
		}

		results = append(results, scored{memory: m, sim: sim})
	}

	// Sort by similarity
	sort.Slice(results, func(i, j int) bool {
		return results[i].sim > results[j].sim
	})

	if len(results) > limit {
		results = results[:limit]
	}

	out := make([]types.Memory, len(results))
	for i, r := range results {
		out[i] = r.memory
	}
	return out, nil
}

// computeEmbedding computes and returns the serialized vector for content.
func computeEmbedding(content string, tags []string) []byte {
	text := content
	for _, t := range tags {
		text += " " + t
	}
	tokens := embed.Tokenize(text)
	vec := embed.Embed(tokens, nil)
	return vec.Serialize()
}

// sqlNullInt64 is a local helper for nullable int64 scanning.
type sqlNullInt64 struct {
	Int64  int64
	Valid  bool
}

func (n *sqlNullInt64) Scan(value any) error {
	if value == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case int64:
		n.Int64 = v
	case float64:
		n.Int64 = int64(math.Round(v))
	}
	return nil
}
