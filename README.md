# tdt — Tool Discovery Tool

A Go library for discovering and filtering MCP (Model Context Protocol) tools. Designed for use with the [mcp-gateway](https://github.com/Kuadrant/mcp-gateway) broker.

## Why Tool Discovery?

MCP gateways aggregate tools from many upstream servers. A gateway with 10 servers might expose 50+ tools, each with a name, description, and full input schema. Every one of those tool definitions is included in the AI agent's context on **every turn** of the conversation — not just the first message.

This creates two problems:

1. **Token cost.** Dozens of tool schemas consume thousands of tokens per turn. Over a multi-turn session this adds up significantly.
2. **Tool selection accuracy.** The more tools in context, the more likely the agent picks the wrong one — especially when tools have similar names or overlapping descriptions.

tdt solves this by letting agents **search for relevant tools first**, then scoping the session to only the tools they need. Instead of 50 tool schemas in context per turn, the agent sees 2-3 discovery tools initially, runs one search, and then works with a focused set of 5-6 tools for the rest of the session.

### How It Works With Agents

The gateway can be configured to initially return only the tdt tools (`discover_tools`, `search_tools`) in `tools/list`. The agent flow is:

1. **Connect** — agent initializes a session and receives `tools/list` with only the discovery tools (2-3 schemas instead of 50+)
2. **Search** — agent calls `discover_tools` with keywords from its task (e.g. `"kubernetes deploy"`, not a full sentence) to find relevant tools
3. **Work** — subsequent `tools/list` calls return only the matched tools, keeping context small for the rest of the session
4. **Re-search** — agent can call the search tool again to change or expand the scoped tool set, or list all tools if needed

The key insight is that agents form **keyword queries**, not natural language sentences. A query like `"prometheus alerts metrics"` is more effective than `"I want to check if there are any alerting issues in my monitoring system"`. The tool descriptions guide this.

## Features

- **Exact-match filtering** by category and tags (`Search`, `MatchingToolNames`)
- **Relevance-ranked search** using porter stemming and BM25 keyword scoring (`RankedSearch`)
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

## Using Ollama for Semantic Search

[Ollama](https://ollama.com) runs embedding models locally, so you don't need an external API key. This example shows how to wire it into the mcp-gateway broker.

### Prerequisites

1. Install Ollama: https://ollama.com/download
2. Pull an embedding model:
   ```bash
   ollama pull nomic-embed-text
   ```
3. Ollama runs on `http://localhost:11434` by default.

### Broker Integration

```go
package main

import (
	"fmt"

	chromem "github.com/philippgille/chromem-go"
	"github.com/maleck13/tdt"
)

func main() {
	// Create an index with Ollama embeddings.
	// nomic-embed-text is a good general-purpose embedding model (768 dimensions).
	embeddingFunc := chromem.NewEmbeddingFuncOllama(
		"nomic-embed-text",
		"http://localhost:11434/api",
	)
	idx := tdt.NewIndexWithEmbedder(embeddingFunc)

	// Register your MCP servers (normally from broker config / CRD).
	idx.Update([]tdt.ServerMetadata{
		{
			ServerName: "prometheus-tools",
			Category:   "observability",
			Tags:       map[string]string{"environment": "production"},
			Hint:       "Query Prometheus metrics, alerts, and recording rules",
			Tools: []tdt.ToolInfo{
				{Name: "prom_query", Description: "Run a PromQL query against Prometheus"},
				{Name: "prom_alerts", Description: "List active Prometheus alerts"},
			},
		},
		{
			ServerName: "github-tools",
			Category:   "cicd",
			Tags:       map[string]string{"department": "engineering"},
			Hint:       "Interact with GitHub repositories and pull requests",
			Tools: []tdt.ToolInfo{
				{Name: "gh_create_pr", Description: "Create a new pull request"},
				{Name: "gh_list_issues", Description: "List open issues in a repository"},
			},
		},
	})

	// Search using natural language — hybrid BM25 + semantic scoring.
	results := idx.RankedSearch(
		tdt.Query{Text: "check CPU usage"},
		tdt.SearchOptions{TopK: 3, MinScore: 0.01},
	)

	for _, r := range results {
		fmt.Printf("%-20s (server: %s, score: %.4f)\n", r.ToolName, r.ServerName, r.Score)
	}
}
```

### How it works

1. On `idx.Update()`, each tool's composite text (name + description + category + tags) is sent to Ollama to generate an embedding vector. This happens once when the broker config changes.
2. On `idx.RankedSearch()`, the query text is embedded and compared against all tool embeddings using cosine similarity. This score is combined with the BM25 keyword score using Reciprocal Rank Fusion (RRF).
3. If Ollama is unavailable at query time, the search falls back to BM25-only — it degrades gracefully.

### Alternative models

Any Ollama model that supports embeddings works. Some options:

| Model | Dimensions | Notes |
|---|---|---|
| `nomic-embed-text` | 768 | Good general-purpose, recommended |
| `all-minilm` | 384 | Smaller, faster |
| `mxbai-embed-large` | 1024 | Higher quality, slower |

```go
// Use a different model
embeddingFunc := chromem.NewEmbeddingFuncOllama("all-minilm", "http://localhost:11434/api")
```

### Without Ollama (BM25 only)

If you don't want to run an embedding service, `NewIndex()` works with BM25 keyword search alone:

```go
idx := tdt.NewIndex() // no embedder — BM25 only
idx.Update(servers)
results := idx.RankedSearch(tdt.Query{Text: "check CPU usage"}, tdt.SearchOptions{TopK: 5})
```

## BM25 Limitations

BM25 is a keyword-based scoring algorithm. It works well for queries that share exact terms with tool descriptions but has known limitations:

- **No stemming:** "network" does not match "networking", "deploy" does not match "deployment". Queries must use the same word forms as the tool descriptions.
- **No semantic understanding:** "check CPU usage" matches tools containing those words, but cannot infer that a Prometheus query tool is relevant when the description says "metrics" instead of "CPU".
- **Length normalization bias:** shorter tool descriptions can score higher than longer ones when keyword overlap is equal, because BM25 normalizes for document length.
- **Tag ambiguity:** when the same tag value (e.g., "database") appears across multiple servers, BM25 cannot distinguish which server is more relevant without additional keyword signal.

These limitations are addressed by enabling semantic search via an embedding provider, which combines cosine similarity with BM25 using Reciprocal Rank Fusion (RRF).
