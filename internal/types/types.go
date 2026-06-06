// Package types defines shared types for Servex.
package types

// Memory represents a stored memory entry.
type Memory struct {
	ID            string   `json:"id"`
	Content       string   `json:"content"`
	Type          string   `json:"type"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Importance    float64  `json:"importance"`
	AccessCount   int      `json:"access_count"`
	CreatedAt     int64    `json:"created_at"`
	UpdatedAt     int64    `json:"updated_at"`
	LastAccessAt  *int64   `json:"last_accessed_at,omitempty"`
}

// Entity represents a node in the knowledge graph.
type Entity struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type,omitempty"`
	Observations []string `json:"observations,omitempty"`
	CreatedAt int64    `json:"created_at"`
}

// Relation represents a directed edge between two entities.
type Relation struct {
	ID           string `json:"id"`
	FromEntity   string `json:"from_entity"`
	ToEntity     string `json:"to_entity"`
	RelationType string `json:"relation_type"`
	CreatedAt    int64  `json:"created_at"`
}

// Graph represents the full knowledge graph.
type Graph struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// MemoryStats holds aggregate statistics.
type MemoryStats struct {
	TotalMemories  int               `json:"total_memories"`
	DeletedMemories int              `json:"deleted_memories"`
	TotalEntities  int               `json:"total_entities"`
	TotalRelations int               `json:"total_relations"`
	TopTags        map[string]int    `json:"top_tags"`
	Types          map[string]int    `json:"types"`
	DBSizeBytes    int64             `json:"db_size_bytes"`
}
