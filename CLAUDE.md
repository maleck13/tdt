# tdt — Tool Discovery Tool

## What This Is

A Go library for relevance-ranked discovery of MCP (Model Context Protocol) tools. Designed for use with the [mcp-gateway](https://github.com/Kuadrant/mcp-gateway) broker to reduce token consumption by filtering tools before sending them to AI agents.

## Architecture

Single Go package `tdt` with no subpackages. All source lives in the project root.

| File | Purpose |
|---|---|
| `types.go` | Core types: `ServerMetadata`, `ToolInfo`, `Query`, `ScoredTool`, `SearchOptions` |
| `index.go` | `Index` — the main entry point. Holds servers, BM25 corpus, and optional embeddings. Provides `Search`, `RankedSearch`, `Catalog`, `MatchingToolNames` |
| `bm25.go` | Tokenizer (camelCase/underscore splitting, Snowball stemming), BM25 corpus builder, BM25 scorer |
| `rrf.go` | Reciprocal Rank Fusion for combining BM25 + semantic scores; BM25 score normalization |
| `tool.go` | `NewDiscoveryTool` — exposes the catalog as an MCP tool via mcp-go |

## Search Modes

1. **Exact match** — `Search(Query{Category: "...", Tags: {...}})` — filters by category/tags
2. **BM25 keyword search** — `RankedSearch(Query{Text: "..."}, opts)` — relevance-ranked with stemming
3. **Hybrid BM25 + semantic** — `NewIndexWithEmbedder(fn)` + `RankedSearch` — combines BM25 with cosine similarity via RRF (k=60)

BM25 parameters: k1=1.2, b=0.75. Tokenizer stems with Snowball English stemmer.

## Key Dependencies

- `github.com/mark3labs/mcp-go` — MCP protocol types and tool handler interface
- `github.com/philippgille/chromem-go` — `EmbeddingFunc` type and provider factories (Ollama, OpenAI). Only used for optional semantic search
- `github.com/kljensen/snowball` — Snowball stemmer for BM25 tokenization

## Testing

```bash
go test ./...
```

- `golden_test.go` — 13 query/expected-result pairs against a 20-tool corpus across 6 categories. This is the primary accuracy validation. When modifying search logic, check that golden tests still pass.
- `bm25_test.go` — Tokenizer and BM25 scoring unit tests
- `rrf_test.go` — RRF combination and normalization
- `ranked_search_test.go` — Integration tests for `RankedSearch` (BM25-only and hybrid paths)
- `index_test.go` — Exact-match search, catalog, tool name listing
- `tool_test.go` — MCP discovery tool definition and handler

## Conventions

- Commits use `git commit -sm "message"` (no Co-Authored-By)
- Test-driven development: write failing test, implement, verify
- `docs/improvements.md` tracks open questions and future work
- `docs/plans/` contains design docs and implementation plans

## Common Patterns

**Adding a tool to the golden corpus:** Add a `ServerMetadata` entry in `goldenServers()` and a corresponding `goldenCase` in `goldenCases()` in `golden_test.go`.

**Modifying tokenization:** Changes to `tokenize()` in `bm25.go` affect both indexing and query processing. Update `TestTokenize_*` expectations in `bm25_test.go` since stemmed forms will change. Run golden tests to check accuracy impact.

**The composite text:** `buildCompositeText()` in `bm25.go` concatenates tool name + description + category + tags. This is what gets tokenized and indexed. It's also the text sent to the embedding function.
