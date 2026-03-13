package tdt

import (
	"regexp"
	"sort"
)

// RegexSearch returns tools whose name or description matches the given regex pattern.
// If query.Regex is empty, falls back to exact matching with score 1.0.
// All matching tools receive a score of 1.0. Results are sorted by tool name for stable ordering.
// Returns nil if the regex pattern is invalid.
func (idx *Index) RegexSearch(query Query, opts SearchOptions) []ScoredTool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// If no regex, fall back to exact-match with score 1.0.
	if query.Regex == "" {
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

	re, err := regexp.Compile(query.Regex)
	if err != nil {
		return nil
	}

	// Pre-filter by Category/Tags if set.
	candidates := idx.servers
	if query.Category != "" || len(query.Tags) > 0 {
		candidates = idx.searchLocked(Query{Category: query.Category, Tags: query.Tags})
	}

	var results []ScoredTool
	for _, s := range candidates {
		for _, t := range s.Tools {
			if regexMatchTool(re, t) {
				results = append(results, ScoredTool{
					ToolName:   t.Name,
					ServerName: s.ServerName,
					Score:      1.0,
				})
			}
		}
	}

	// Sort by tool name for stable ordering.
	sort.Slice(results, func(i, j int) bool {
		return results[i].ToolName < results[j].ToolName
	})

	// Apply MinScore filter.
	if opts.MinScore > 0 {
		filtered := results[:0]
		for _, r := range results {
			if r.Score >= opts.MinScore {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	// Apply TopK limit.
	if opts.TopK > 0 && len(results) > opts.TopK {
		results = results[:opts.TopK]
	}

	return results
}

// regexMatchTool returns true if the regex matches the tool's name or description.
func regexMatchTool(re *regexp.Regexp, tool ToolInfo) bool {
	return re.MatchString(tool.Name) || re.MatchString(tool.Description)
}
