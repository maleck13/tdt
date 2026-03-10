package tdt

import (
	"sort"
	"sync"
)

// Index holds the searchable metadata for all registered servers.
type Index struct {
	mu      sync.RWMutex
	servers []ServerMetadata
	corpus  *bm25Corpus
}

// NewIndex creates a new empty Index.
func NewIndex() *Index {
	return &Index{}
}

// Update replaces the entire index with new metadata.
func (idx *Index) Update(servers []ServerMetadata) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.servers = make([]ServerMetadata, len(servers))
	copy(idx.servers, servers)
	idx.corpus = newBM25Corpus(idx.servers)
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
	if len(candidates) != len(idx.servers) {
		corpus = newBM25Corpus(candidates)
	}

	// Score with BM25.
	bm25Scores := corpus.score(query.Text)

	// Normalize BM25 scores to 0-1.
	normalized := normalizeBM25(bm25Scores)

	// Sort descending.
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].score > normalized[j].score
	})

	// Apply MinScore and TopK.
	var results []ScoredTool
	for _, s := range normalized {
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
