# tdt — Tool Discovery Tool

A Go library for discovering and filtering MCP (Model Context Protocol) tools. Designed for use with the [mcp-gateway](https://github.com/Kuadrant/mcp-gateway) broker.

## Features

- **Exact-match filtering** by category and tags (`Search`, `MatchingToolNames`)
- **Relevance-ranked search** using BM25 keyword scoring (`RankedSearch`)
- **Optional semantic search** via embeddings (chromem-go) combined with BM25 using Reciprocal Rank Fusion
- **Discovery tool** (`discover_tools`) that exposes the catalog as an MCP tool

## Usage

```go
idx := tdt.NewIndex()
idx.Update(servers) // rebuild index from server metadata

// Exact match
results := idx.Search(tdt.Query{Category: "observability"})

// Relevance search (BM25)
scored := idx.RankedSearch(
    tdt.Query{Text: "check CPU metrics"},
    tdt.SearchOptions{TopK: 5, MinScore: 0.1},
)

// Hybrid search (BM25 + semantic) — requires an embedding provider
idx := tdt.NewIndexWithEmbedder(chromem.NewEmbeddingFuncOllama("nomic-embed-text", "http://localhost:11434"))
idx.Update(servers)
scored := idx.RankedSearch(tdt.Query{Text: "check CPU metrics"}, tdt.SearchOptions{TopK: 5})
```

## BM25 Limitations

BM25 is a keyword-based scoring algorithm. It works well for queries that share exact terms with tool descriptions but has known limitations:

- **No stemming:** "network" does not match "networking", "deploy" does not match "deployment". Queries must use the same word forms as the tool descriptions.
- **No semantic understanding:** "check CPU usage" matches tools containing those words, but cannot infer that a Prometheus query tool is relevant when the description says "metrics" instead of "CPU".
- **Length normalization bias:** shorter tool descriptions can score higher than longer ones when keyword overlap is equal, because BM25 normalizes for document length.
- **Tag ambiguity:** when the same tag value (e.g., "database") appears across multiple servers, BM25 cannot distinguish which server is more relevant without additional keyword signal.

These limitations are addressed by enabling semantic search via an embedding provider, which combines cosine similarity with BM25 using Reciprocal Rank Fusion (RRF).
