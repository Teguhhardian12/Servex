// Package memory — knowledge graph operations (entity/relation/observation).
package memory

import (
	"fmt"
	"time"

	"github.com/rxyz/servex/internal/types"
)

// CreateEntity creates an entity with optional observations.
func (s *Store) CreateEntity(name, entityType string, observations []string) (*types.Entity, error) {
	now := time.Now().UnixMilli()
	id := nanoid()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO entities (id, name, type, created_at) VALUES (?, ?, ?, ?)`, id, name, entityType, now)
	if err != nil {
		return nil, fmt.Errorf("insert entity: %w", err)
	}

	for _, obs := range observations {
		obsID := nanoid()
		_, err = tx.Exec(`INSERT INTO observations (id, entity_id, content, created_at) VALUES (?, ?, ?, ?)`, obsID, id, obs, now)
		if err != nil {
			return nil, fmt.Errorf("insert observation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &types.Entity{
		ID:           id,
		Name:         name,
		Type:         entityType,
		Observations: observations,
		CreatedAt:    now,
	}, nil
}

// SearchEntities searches entities by name substring.
func (s *Store) SearchEntities(query string, limit int) ([]types.Entity, error) {
	rows, err := s.db.Query(
		`SELECT id, name, type, created_at FROM entities WHERE name LIKE ? ORDER BY created_at DESC LIMIT ?`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Entity
	for rows.Next() {
		var e types.Entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Observations, _ = s.getObservations(e.ID)
		out = append(out, e)
	}
	return out, nil
}

// GetEntity retrieves an entity by ID with its observations.
func (s *Store) GetEntity(id string) (*types.Entity, error) {
	var e types.Entity
	err := s.db.QueryRow(`SELECT id, name, type, created_at FROM entities WHERE id = ?`, id).Scan(&e.ID, &e.Name, &e.Type, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	e.Observations, _ = s.getObservations(e.ID)
	return &e, nil
}

func (s *Store) getObservations(entityID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT content FROM observations WHERE entity_id = ? ORDER BY created_at`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		out = append(out, c)
	}
	return out, nil
}

// CreateRelation creates a directed relation between two entities.
func (s *Store) CreateRelation(fromID, toID, relationType string) (*types.Relation, error) {
	now := time.Now().UnixMilli()
	id := nanoid()

	_, err := s.db.Exec(
		`INSERT INTO relations (id, from_entity, to_entity, relation_type, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, fromID, toID, relationType, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert relation: %w", err)
	}

	return &types.Relation{
		ID:           id,
		FromEntity:   fromID,
		ToEntity:     toID,
		RelationType: relationType,
		CreatedAt:    now,
	}, nil
}

// ReadGraph returns the full knowledge graph.
func (s *Store) ReadGraph() (*types.Graph, error) {
	// Entities
	eRows, err := s.db.Query(`SELECT id, name, type, created_at FROM entities ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()

	var entities []types.Entity
	for eRows.Next() {
		var e types.Entity
		if err := eRows.Scan(&e.ID, &e.Name, &e.Type, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Observations, _ = s.getObservations(e.ID)
		entities = append(entities, e)
	}

	// Relations
	rRows, err := s.db.Query(`SELECT id, from_entity, to_entity, relation_type, created_at FROM relations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rRows.Close()

	var relations []types.Relation
	for rRows.Next() {
		var r types.Relation
		if err := rRows.Scan(&r.ID, &r.FromEntity, &r.ToEntity, &r.RelationType, &r.CreatedAt); err != nil {
			return nil, err
		}
		relations = append(relations, r)
	}

	return &types.Graph{Entities: entities, Relations: relations}, nil
}
