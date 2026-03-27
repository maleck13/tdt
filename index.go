package tdt

import (
	"context"
	"log"
	"math"
	"sort"
	"strings"
	"sync"

	chromem "github.com/philippgille/chromem-go"
)

// Index holds the searchable metadata for all registered servers.
type Index struct {
	mu            sync.RWMutex
	servers       []ServerMetadata
	corpus        *bm25Corpus
	embeddingFunc chromem.EmbeddingFunc
	embeddings    [][]float32
}

// NewIndex creates a new empty Index.
func NewIndex() *Index {
	return &Index{}
}

// NewIndexWithEmbedder creates an index with BM25 + semantic search.
func NewIndexWithEmbedder(embeddingFunc chromem.EmbeddingFunc) *Index {
	return &Index{embeddingFunc: embeddingFunc}
}

// Update replaces the entire index with new metadata.
// Embedding computation happens outside the lock to avoid blocking readers.
func (idx *Index) Update(servers []ServerMetadata) {
	// Copy servers outside the lock.
	serversCopy := make([]ServerMetadata, len(servers))
	copy(serversCopy, servers)

	// Build corpus outside the lock.
	corpus := newBM25Corpus(serversCopy)

	// Compute embeddings outside the lock (potentially slow network calls).
	var embeddings [][]float32
	if idx.embeddingFunc != nil {
		embeddings = make([][]float32, len(corpus.docs))
		for i, doc := range corpus.docs {
			text := buildCompositeText(
				findServer(serversCopy, doc.serverName),
				findTool(serversCopy, doc.serverName, doc.toolName),
			)
			emb, err := idx.embeddingFunc(context.Background(), text)
			if err != nil {
				log.Printf("tdt: failed to embed tool %s: %v, semantic search disabled", doc.toolName, err)
				embeddings = nil
				break
			}
			embeddings[i] = emb
		}
	}

	// Take lock only for final assignment.
	idx.mu.Lock()
	idx.servers = serversCopy
	idx.corpus = corpus
	idx.embeddings = embeddings
	idx.mu.Unlock()
}

// Search returns servers matching the query. An empty query returns all servers.
func (idx *Index) Search(query Query) []ServerMetadata {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.searchLocked(query)
}

func (idx *Index) searchLocked(query Query) []ServerMetadata {
	if query.IsEmpty() {
		result := make([]ServerMetadata, len(idx.servers))
		copy(result, idx.servers)
		return result
	}

	var results []ServerMetadata
	for _, s := range idx.servers {
		if matchesQuery(s, query) {
			results = append(results, s)
		}
	}
	return results
}

// Catalog returns the full catalog grouped by category.
func (idx *Index) Catalog() CatalogResponse {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	catMap := map[string][]CatalogServer{}
	for _, s := range idx.servers {
		cs := CatalogServer{
			Name: s.ServerName,
			Hint: s.Hint,
			Tags: s.Tags,
		}
		catMap[s.Category] = append(catMap[s.Category], cs)
	}

	categories := make([]CatalogCategory, 0, len(catMap))
	for name, servers := range catMap {
		categories = append(categories, CatalogCategory{
			Name:    name,
			Servers: servers,
		})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	return CatalogResponse{Categories: categories}
}

// MatchingToolNames returns the names of all tools from servers matching the query.
// An empty query returns all tool names.
func (idx *Index) MatchingToolNames(query Query) []string {
	servers := idx.Search(query)
	var names []string
	for _, s := range servers {
		for _, t := range s.Tools {
			names = append(names, t.Name)
		}
	}
	return names
}

func matchesQuery(s ServerMetadata, q Query) bool {
	if q.Category != "" && s.Category != q.Category {
		return false
	}
	for k, v := range q.Tags {
		sv, ok := s.Tags[k]
		if !ok || sv != v {
			return false
		}
	}
	return true
}

// RankedSearch performs relevance-based search using BM25 scoring.
// If query.Text is empty, falls back to exact matching with score 1.0.
// Supports pre-filtering by Category and Tags before BM25 scoring.
func (idx *Index) RankedSearch(query Query, opts SearchOptions) []ScoredTool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Apply default minimum score threshold when none specified.
	if opts.MinScore == 0 {
		opts.MinScore = DefaultMinScore
	}

	// If no text query, fall back to exact-match and return all matches with score 1.0.
	if query.Text == "" {
		matches := idx.searchLocked(query)
		var results []ScoredTool
		for _, s := range matches {
			for _, t := range s.Tools {
				results = append(results, ScoredTool{
					ToolName:   t.Name,
					ServerName: s.ServerName,
					Score:      1.0,
				})
			}
		}
		return results
	}

	// Pre-filter: if category/tags set, restrict to matching servers.
	candidates := idx.servers
	if query.Category != "" || len(query.Tags) > 0 {
		candidates = idx.searchLocked(Query{Category: query.Category, Tags: query.Tags})
	}

	// Build a temporary corpus from candidates if pre-filtered.
	corpus := idx.corpus
	if corpus == nil {
		return nil // Index not initialized yet
	}
	if len(candidates) != len(idx.servers) {
		corpus = newBM25Corpus(candidates)
	}

	// Split query on commas into sub-queries. Score each independently
	// and take the max score per tool across sub-queries.
	subQueries := splitQuerySegments(query.Text)

	// Score each sub-query and merge with max-score strategy.
	merged := idx.scoreSubQueries(corpus, subQueries)

	// Sort descending by score.
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].score > merged[j].score
	})

	// Apply MinScore and TopK filters.
	var results []ScoredTool
	for _, s := range merged {
		if opts.MinScore > 0 && s.score < opts.MinScore {
			continue
		}
		results = append(results, ScoredTool{
			ToolName:   s.toolName,
			ServerName: s.serverName,
			Score:      s.score,
		})
		if opts.TopK > 0 && len(results) >= opts.TopK {
			break
		}
	}
	return results
}

// splitQuerySegments splits a query on commas into trimmed, non-empty segments.
func splitQuerySegments(text string) []string {
	parts := strings.Split(text, ",")
	var segments []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			segments = append(segments, p)
		}
	}
	return segments
}

// scoreSubQueries scores each sub-query independently and returns the max
// score per tool across all sub-queries.
func (idx *Index) scoreSubQueries(corpus *bm25Corpus, subQueries []string) []toolScore {
	type toolKey struct {
		toolName   string
		serverName string
	}

	best := map[toolKey]float64{}

	for _, sq := range subQueries {
		bm25Scores := corpus.score(sq)

		// If embedder available and embeddings exist, run hybrid for this sub-query.
		if idx.embeddingFunc != nil && idx.embeddings != nil {
			queryEmb, err := idx.embeddingFunc(context.Background(), sq)
			if err != nil {
				log.Printf("tdt: query embedding failed: %v, falling back to BM25", err)
			} else {
				semanticScores := make([]toolScore, len(bm25Scores))
				for i, doc := range corpus.docs {
					embIdx := idx.findEmbeddingIndex(doc.toolName, doc.serverName)
					sim := float64(0)
					if embIdx >= 0 {
						sim = cosineSimilarity(queryEmb, idx.embeddings[embIdx])
					}
					semanticScores[i] = toolScore{
						toolName:   doc.toolName,
						serverName: doc.serverName,
						score:      sim,
					}
				}
				combined := combineRRF(bm25Scores, semanticScores, 60)
				for _, s := range combined {
					k := toolKey{s.toolName, s.serverName}
					if s.score > best[k] {
						best[k] = s.score
					}
				}
				continue
			}
		}

		// BM25-only path.
		normalized := normalizeBM25(bm25Scores)
		for _, s := range normalized {
			k := toolKey{s.toolName, s.serverName}
			if s.score > best[k] {
				best[k] = s.score
			}
		}
	}

	merged := make([]toolScore, 0, len(best))
	for k, score := range best {
		merged = append(merged, toolScore{
			toolName:   k.toolName,
			serverName: k.serverName,
			score:      score,
		})
	}
	return merged
}

func findServer(servers []ServerMetadata, name string) ServerMetadata {
	for _, s := range servers {
		if s.ServerName == name {
			return s
		}
	}
	return ServerMetadata{}
}

func findTool(servers []ServerMetadata, serverName, toolName string) ToolInfo {
	for _, s := range servers {
		if s.ServerName == serverName {
			for _, t := range s.Tools {
				if t.Name == toolName {
					return t
				}
			}
		}
	}
	return ToolInfo{Name: toolName}
}

func (idx *Index) findEmbeddingIndex(toolName, serverName string) int {
	for i, doc := range idx.corpus.docs {
		if doc.toolName == toolName && doc.serverName == serverName {
			return i
		}
	}
	return -1
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
