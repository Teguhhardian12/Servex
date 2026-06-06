// Package server implements the MCP protocol handler for Servex.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxyz/servex/internal/memory"
	"github.com/rxyz/servex/internal/types"
)

// NewMCPServer creates and configures the MCP server with all 12 tools.
func NewMCPServer(store *memory.Store) *server.MCPServer {
	s := server.NewMCPServer(
		"servex",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	registerMemoryTools(s, store)
	registerGraphTools(s, store)

	return s
}

func registerMemoryTools(s *server.MCPServer, store *memory.Store) {
	// memory_create
	s.AddTool(
		mcp.NewTool("memory_create",
			mcp.WithDescription("Store a new memory with optional tags, type, and category"),
			mcp.WithString("content", mcp.Required(), mcp.Description("The memory content to store")),
			mcp.WithString("type", mcp.Description("Memory type: note, fact, preference, context (default: note)")),
			mcp.WithString("category", mcp.Description("Optional category for grouping")),
			mcp.WithArray("tags", mcp.Description("Optional tags for filtering")),
			mcp.WithNumber("importance", mcp.Description("Importance score 0.0-10.0 (default: 1.0)")),
		),
		handleMemoryCreate(store),
	)

	// memory_search
	s.AddTool(
		mcp.NewTool("memory_search",
			mcp.WithDescription("Search memories using full-text search (FTS5)"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithNumber("limit", mcp.Description("Max results (default: 10)")),
			mcp.WithString("type_filter", mcp.Description("Filter by memory type")),
			mcp.WithString("tag_filter", mcp.Description("Filter by tag")),
		),
		handleMemorySearch(store),
	)

	// memory_get
	s.AddTool(
		mcp.NewTool("memory_get",
			mcp.WithDescription("Get a specific memory by ID"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Memory ID")),
		),
		handleMemoryGet(store),
	)

	// memory_update
	s.AddTool(
		mcp.NewTool("memory_update",
			mcp.WithDescription("Update a memory's content and/or tags"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Memory ID")),
			mcp.WithString("content", mcp.Description("New content")),
			mcp.WithArray("tags", mcp.Description("New tags")),
		),
		handleMemoryUpdate(store),
	)

	// memory_delete
	s.AddTool(
		mcp.NewTool("memory_delete",
			mcp.WithDescription("Soft-delete a memory by ID"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Memory ID")),
		),
		handleMemoryDelete(store),
	)

	// memory_list
	s.AddTool(
		mcp.NewTool("memory_list",
			mcp.WithDescription("List all memories with optional filters"),
			mcp.WithNumber("limit", mcp.Description("Max results (default: 20)")),
			mcp.WithNumber("offset", mcp.Description("Pagination offset (default: 0)")),
			mcp.WithString("type_filter", mcp.Description("Filter by memory type")),
			mcp.WithString("tag_filter", mcp.Description("Filter by tag")),
		),
		handleMemoryList(store),
	)

	// memory_stats
	s.AddTool(
		mcp.NewTool("memory_stats",
			mcp.WithDescription("Get memory store statistics: counts, top tags, types, DB size"),
		),
		handleMemoryStats(store),
	)
}

func registerGraphTools(s *server.MCPServer, store *memory.Store) {
	// entity_create
	s.AddTool(
		mcp.NewTool("entity_create",
			mcp.WithDescription("Create an entity (knowledge graph node) with optional observations"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Entity name")),
			mcp.WithString("type", mcp.Description("Entity type (e.g. person, project, concept)")),
			mcp.WithArray("observations", mcp.Description("Atomic facts about this entity")),
		),
		handleEntityCreate(store),
	)

	// entity_search
	s.AddTool(
		mcp.NewTool("entity_search",
			mcp.WithDescription("Search entities by name"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
			mcp.WithNumber("limit", mcp.Description("Max results (default: 10)")),
		),
		handleEntitySearch(store),
	)

	// relation_create
	s.AddTool(
		mcp.NewTool("relation_create",
			mcp.WithDescription("Create a directed relation between two entities"),
			mcp.WithString("from", mcp.Required(), mcp.Description("Source entity ID")),
			mcp.WithString("to", mcp.Required(), mcp.Description("Target entity ID")),
			mcp.WithString("relation_type", mcp.Required(), mcp.Description("Relation type (e.g. 'depends_on', 'uses', 'part_of')")),
		),
		handleRelationCreate(store),
	)

	// graph_read
	s.AddTool(
		mcp.NewTool("graph_read",
			mcp.WithDescription("Read the full knowledge graph (entities + relations)"),
		),
		handleGraphRead(store),
	)

	// graph_export
	s.AddTool(
		mcp.NewTool("graph_export",
			mcp.WithDescription("Export the knowledge graph in JSON or Markdown format"),
			mcp.WithString("format", mcp.Description("Export format: json or markdown (default: json)")),
		),
		handleGraphExport(store),
	)
}

// --- Handlers ---

func handleMemoryCreate(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content, _ := req.RequireString("content")
		memType := req.GetString("type", "note")
		category := req.GetString("category", "")
		importance := req.GetFloat("importance", 1.0)

		tags := req.GetStringSlice("tags", nil)

		m, err := store.CreateMemory(content, memType, category, tags, importance)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(m)
	}
}

func handleMemorySearch(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, _ := req.RequireString("query")
		limit := int(req.GetFloat("limit", 10))
		typeFilter := req.GetString("type_filter", "")
		tagFilter := req.GetString("tag_filter", "")

		results, err := store.SearchMemory(query, limit, typeFilter, tagFilter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(results)
	}
}

func handleMemoryGet(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, _ := req.RequireString("id")
		m, err := store.GetMemory(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("memory not found: %s", id)), nil
		}
		return jsonResult(m)
	}
}

func handleMemoryUpdate(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, _ := req.RequireString("id")

		var content *string
		if c := req.GetString("content", ""); c != "" {
			content = &c
		}

		var tags *[]string
		if args := req.GetArguments(); args != nil {
			if raw, ok := args["tags"]; ok && raw != nil {
				if arr, ok := raw.([]any); ok {
					var t []string
					for _, v := range arr {
						if s, ok := v.(string); ok {
							t = append(t, s)
						}
					}
					tags = &t
				}
			}
		}

		m, err := store.UpdateMemory(id, content, tags)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(m)
	}
}

func handleMemoryDelete(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, _ := req.RequireString("id")
		if err := store.DeleteMemory(id); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return textResult(fmt.Sprintf("Memory %s deleted", id)), nil
	}
}

func handleMemoryList(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := int(req.GetFloat("limit", 20))
		offset := int(req.GetFloat("offset", 0))
		typeFilter := req.GetString("type_filter", "")
		tagFilter := req.GetString("tag_filter", "")

		results, err := store.ListMemories(limit, offset, typeFilter, tagFilter)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(results)
	}
}

func handleMemoryStats(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stats, err := store.Stats()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(stats)
	}
}

func handleEntityCreate(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, _ := req.RequireString("name")
		entityType := req.GetString("type", "")

		observations := req.GetStringSlice("observations", nil)

		e, err := store.CreateEntity(name, entityType, observations)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(e)
	}
}

func handleEntitySearch(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, _ := req.RequireString("query")
		limit := int(req.GetFloat("limit", 10))

		results, err := store.SearchEntities(query, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(results)
	}
}

func handleRelationCreate(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		from, _ := req.RequireString("from")
		to, _ := req.RequireString("to")
		relationType, _ := req.RequireString("relation_type")

		r, err := store.CreateRelation(from, to, relationType)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(r)
	}
}

func handleGraphRead(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		graph, err := store.ReadGraph()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return jsonResult(graph)
	}
}

func handleGraphExport(store *memory.Store) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		format := req.GetString("format", "json")
		graph, err := store.ReadGraph()
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if format == "markdown" {
			return textResult(exportMarkdown(graph)), nil
		}
		return jsonResult(graph)
	}
}

// --- Helpers ---

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, _ := json.MarshalIndent(v, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

func textResult(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

func exportMarkdown(graph *types.Graph) string {
	var b strings.Builder
	b.WriteString("# Knowledge Graph\n\n")

	b.WriteString("## Entities\n\n")
	for _, e := range graph.Entities {
		b.WriteString(fmt.Sprintf("### %s", e.Name))
		if e.Type != "" {
			b.WriteString(fmt.Sprintf(" (%s)", e.Type))
		}
		b.WriteString("\n")
		for _, obs := range e.Observations {
			b.WriteString(fmt.Sprintf("- %s\n", obs))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Relations\n\n")
	for _, r := range graph.Relations {
		b.WriteString(fmt.Sprintf("- **%s** → %s → **%s**\n", r.FromEntity, r.RelationType, r.ToEntity))
	}

	return b.String()
}
