# Servex

**AI Memory Server (MCP Compatible)** Persistent memory and knowledge graph for AI agents.

A single Go binary that gives any MCP-compatible AI tool (Claude Code, Cursor, Continue, etc.) the ability to remember, search, and reason over persistent knowledge. Zero model downloads. Zero external services. Works fully offline.

## Why Servex?

| Problem | Solution |
|---------|----------|
| AI agents forget everything between sessions | Persistent SQLite memory store with soft-delete |
| Keyword search misses semantic matches | Hybrid search: FTS5 keyword + TF-IDF vector + RRF fusion |
| No way to model relationships between knowledge | Built-in knowledge graph (entities, relations, observations) |
| Existing memory servers need Python/Node/external deps | Single Go binary, zero runtime dependencies |

## Features

- **12 MCP Tools** — Memory CRUD (create, search, get, update, delete, list) + Knowledge Graph (entities, relations, graph read/export) + Stats
- **Hybrid Search** — FTS5 BM25 keyword + 256-dim TF-IDF vector + Reciprocal Rank Fusion (RRF). Three search modes: `hybrid`, `keyword`, `semantic`
- **Knowledge Graph** — Entities with observations, directed relations. Export as JSON or Markdown
- **SQLite Backend** — WAL mode, foreign keys, ACID transactions, single-file database
- **Zero Model Downloads** — Pure Go TF-IDF feature hashing (256-dim vectors), no API keys, no network calls
- **Dual Transport** — MCP stdio (Claude Code/Cursor) + HTTP+SSE (web apps)
- **Single Binary** — `go build` → ~15MB static binary, zero runtime dependencies

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

```bash
claude mcp add servex /path/to/servex -- --stdio
```

Or add to `.claude/settings.json`:

```json
{
  "mcpServers": {
    "servex": {
      "command": "/path/to/servex",
      "args": ["--stdio"]
    }
  }
}
```

### Cursor / Continue / Other MCP Clients

Use stdio transport with the same command pattern. See your client's MCP documentation.

## Tools

### Memory (7 tools)

| Tool | Description |
|------|-------------|
| `memory_create` | Store a new memory with optional tags, type, category, importance |
| `memory_search` | Hybrid search: FTS5 keyword + TF-IDF vector + RRF fusion. Modes: `hybrid`, `keyword`, `semantic` |
| `memory_get` | Get a specific memory by ID |
| `memory_update` | Patch a memory's content and/or tags (recomputes embedding) |
| `memory_delete` | Soft-delete a memory |
| `memory_list` | List memories with pagination and filters (type, tag) |
| `memory_stats` | Store statistics: counts, top tags, types, DB size |

### Knowledge Graph (5 tools)

| Tool | Description |
|------|-------------|
| `entity_create` | Create an entity with observations (atomic facts) |
| `entity_search` | Search entities by name |
| `relation_create` | Directed edge between entities (e.g. `depends_on`, `uses`, `part_of`) |
| `graph_read` | Full graph dump (entities + relations) |
| `graph_export` | Export graph as JSON or Markdown |

## Architecture

```
servex/
├── main.go                    # Entry point: stdio or HTTP mode
├── Makefile                   # Build with CGO flags for FTS5
├── internal/
│   ├── server/
│   │   └── mcp.go             # MCP tool registration + handlers (12 tools)
│   ├── memory/
│   │   ├── store.go           # SQLite CRUD, FTS5 migration, memory lifecycle
│   │   ├── search.go          # Hybrid search: FTS5 + vector + RRF fusion
│   │   └── graph.go           # Knowledge graph: entity/relation/observation CRUD
│   ├── embed/
│   │   └── embed.go           # TF-IDF feature hashing (256-dim), cosine similarity
│   └── types/
│       └── types.go           # Shared types: Memory, Entity, Relation, Graph, Stats
├── migrations/
│   └── 001_init.sql           # Schema: memories, FTS5, entities, observations, relations
└── web/
    └── index.html             # (TODO) Dashboard UI
```

### Search Pipeline

```
Query → Tokenize → FTS5 BM25 search  ─┐
                                      ├─ Reciprocal Rank Fusion → Top-N results
        Tokenize → TF-IDF 256-dim   ─┘   (k=60, combined score)
               → Cosine similarity against stored vectors
```

## Roadmap

- [x] Phase 1: Memory CRUD + FTS5 search + MCP server
- [x] Phase 2: Knowledge graph (entities, relations, observations)
- [x] Phase 3: Hybrid search (FTS5 + TF-IDF vector + RRF fusion)
- [ ] Phase 4: Dashboard UI (web/index.html)

## License

MIT
