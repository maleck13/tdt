# Improvements and Open Questions

## Open Questions

### Is chromem-go the right embedding dependency?

We chose chromem-go because it is pure Go, has no CGo/native dependencies, and provides ready-made `EmbeddingFunc` factories for multiple providers (OpenAI, Ollama, Cohere, etc.). However:

- **Maintenance status:** chromem-go is a relatively small project. If it becomes unmaintained, we inherit that risk. We should monitor its release cadence and issue activity.
- **We only use a fraction of it:** We import chromem-go for `EmbeddingFunc` type and the provider factories. We don't use its vector store, persistence, or document management. This means we pull in more code than we need.
- **Alternative: define our own interface.** We could define `type EmbeddingFunc func(ctx context.Context, text string) ([]float32, error)` directly in tdt and write thin adapter packages for Ollama/OpenAI. This removes the dependency entirely but requires us to maintain HTTP client code. Worth considering if chromem-go proves problematic.

### Could we use all-MiniLM-L6-v2 locally instead of an external service?

The tool corpus is small (typically 10-200 short text blobs). A local embedding model could eliminate the Ollama/API dependency entirely:

- [clems4ever/all-minilm-l6-v2-go](https://pkg.go.dev/github.com/clems4ever/all-minilm-l6-v2-go) embeds the all-MiniLM-L6-v2 model directly in the Go binary via ONNX Runtime. 384-dimensional embeddings, good quality for short text similarity.
- **Trade-off:** adds ~90MB to the binary size and requires ONNX Runtime as a system dependency. This complicates the mcp-gateway container image.
- [kelindar/search](https://github.com/kelindar/search) runs GGUF BERT models via llama.cpp (no CGo, uses purego). Smaller model files (~23MB) but requires precompiled llama.cpp binaries.
- **For our corpus size, either would work.** Embedding 200 short texts takes milliseconds locally. The question is whether the deployment complexity is worth avoiding an external Ollama/API call that only happens on config changes.

### How can we improve and benchmark search accuracy?

**Current measurement:** The golden test set (`golden_test.go`) has 13 query/expected-result pairs against a 20-tool corpus. It validates top-1 and top-3 results. 3 of the original expectations were adjusted down due to BM25 limitations (no stemming, no semantic understanding).

**Improvements to explore:**

- **Stemming:** Add a simple stemmer (e.g., Porter stemmer) to the tokenizer so "network" matches "networking" and "deploy" matches "deployment". This is the lowest-effort improvement to BM25 accuracy.
- **Server hint in composite text:** Currently `buildCompositeText` uses tool name + description + category + tags. Adding the server-level `Hint` field would give BM25 more text to match against.
- **Weighted fields:** Not all text is equally important. A match on the tool name could be weighted higher than a match in a tag value. BM25F (field-aware BM25) supports this but adds complexity.

**Stemmer language configuration:**

- The stemmer currently hardcodes `"english"` as the Snowball language. This should be configurable so that non-English tool descriptions are stemmed correctly. The `snowball` library supports many languages (e.g., `"spanish"`, `"french"`, `"german"`). Consider adding a `StemLanguage` option to `Index` or `SearchOptions`.

**Benchmarking approach:**

- **Expand the golden test set:** Add more queries, especially edge cases and queries that real agents would send. Target 30-50 test cases.
- **Add quantitative metrics:** Implement a benchmark function that computes precision@k, recall@k, and Mean Reciprocal Rank (MRR) across the golden set. Report these as `go test -bench` output so regressions are visible.
- **Compare BM25-only vs hybrid:** Once semantic search is integrated with a real embedding model, run the same golden set against both modes and compare metrics. This tells us concretely how much value embeddings add.
- **Real-world corpus:** The golden corpus is synthetic. Testing against actual MCP server registrations from a production mcp-gateway deployment would give more realistic accuracy data.
