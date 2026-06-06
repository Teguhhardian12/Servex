# Servex

**AI Memory Server (MCP Compatible)** — Persistent memory and knowledge graph for AI agents.

Build with Go, zero runtime dependencies. Single binary that works with Claude Code, Cursor, Continue, and any MCP-compatible AI tool.

## Features

- **12 MCP Tools** — Memory CRUD (create, search, get, update, delete, list) + Knowledge Graph (entities, relations, graph read/export) + Stats
- **FTS5 Full-Text Search** — Fast, built into SQLite, no external search service needed
- **Knowledge Graph** — Entities with observations, directed relations between entities
- **SQLite Backend** — WAL mode, foreign keys, ACID transactions
- **Dual Transport** — MCP stdio mode (for Claude Code/Cursor) + HTTP+SSE mode (for web apps)
- **Single Binary** — `go build` → one static binary, zero runtime dependencies

## Quick Start

### Build

```bash
# Requires: Go 1.20+, GCC (for CGO/SQLite)
make build
```

### Run (stdio mode, for Claude Code / Cursor)

```bash
./servex --stdio
```

### Run (HTTP+SSE mode, for web apps)

```bash
./servex --http :8080
```

### Custom database path

```bash
./servex --stdio --db ~/.servex/memory.db
```

## MCP Configuration

### Claude Code

Add to your `.claude/settings.json`:

```json
{
  "mcpServers": {
    "servex": {
      "command": "/path/to/servex",
      "args": ["--stdio"],
      "env": {}
    }
  }
}
```

### Cursor / Continue / Other MCP Clients

Use stdio transport with the same command pattern.

## Tools

### Memory CRUD

| Tool | Description |
|------|-------------|
| `memory_create` | Store a new memory with optional tags, type, and category |
| `memory_search` | Full-text search (FTS5) with type and tag filters |
| `memory_get` | Get a specific memory by ID |
| `memory_update` | Patch a memory's content and/or tags |
| `memory_delete` | Soft-delete a memory |
| `memory_list` | List memories with pagination and filters |
| `memory_stats` | Store statistics: counts, top tags, DB size |

### Knowledge Graph

| Tool | Description |
|------|-------------|
| `entity_create` | Create an entity with observations (atomic facts) |
| `entity_search` | Search entities by name |
| `relation_create` | Directed edge between entities (e.g. "depends_on", "uses") |
| `graph_read` | Full graph dump (entities + relations) |
| `graph_export` | Export graph as JSON or Markdown |

## Architecture

```
servex/
├── main.go                    # Entry point (stdio / HTTP flag)
├── internal/
│   ├── server/
│   │   └── mcp.go             # MCP protocol handler (12 tools)
│   ├── memory/
│   │   ├── store.go           # SQLite CRUD + FTS5 search
│   │   └── graph.go           # Knowledge graph operations
│   ├── embed/
│   │   └── embed.go           # (TODO) Local embedding via ONNX
│   └── types/
│       └── types.go           # Shared types
└── migrations/
    └── 001_init.sql           # Schema
```

## Roadmap

- [x] Phase 1: Memory CRUD + FTS5 search + MCP server
- [x] Phase 2: Knowledge graph (entities, relations, observations)
- [ ] Phase 3: Local embedding + semantic search (all-MiniLM-L6-v2 via ONNX)
- [ ] Phase 4: Dashboard UI

## License

MIT
