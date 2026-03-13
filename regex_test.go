package tdt

import "testing"

func TestRegexSearch_MatchesToolName(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: "prom_query"}, SearchOptions{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToolName != "prom_query" {
		t.Fatalf("expected prom_query, got %s", results[0].ToolName)
	}
	if results[0].Score != 1.0 {
		t.Fatalf("expected score 1.0, got %f", results[0].Score)
	}
}

func TestRegexSearch_MatchesDescription(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: "PromQL"}, SearchOptions{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToolName != "prom_query" {
		t.Fatalf("expected prom_query, got %s", results[0].ToolName)
	}
}

func TestRegexSearch_CaseInsensitive(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: "(?i)promql"}, SearchOptions{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToolName != "prom_query" {
		t.Fatalf("expected prom_query, got %s", results[0].ToolName)
	}
}

func TestRegexSearch_WildcardPattern(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: "prom_.*"}, SearchOptions{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	names := map[string]bool{}
	for _, r := range results {
		names[r.ToolName] = true
	}
	if !names["prom_query"] || !names["prom_alerts"] {
		t.Fatalf("expected prom_query and prom_alerts, got %v", names)
	}
}

func TestRegexSearch_PreFilterByCategory(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{
		Category: "networking",
		Regex:    "dns_.*",
	}, SearchOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToolName != "dns_create_record" {
		t.Fatalf("expected dns_create_record, got %s", results[0].ToolName)
	}
}

func TestRegexSearch_InvalidRegex(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: "[invalid"}, SearchOptions{})
	if results != nil {
		t.Fatalf("expected nil for invalid regex, got %d results", len(results))
	}
}

func TestRegexSearch_TopK(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: ".*"}, SearchOptions{TopK: 2})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestRegexSearch_MinScore(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	// All matches score 1.0, so MinScore 0.5 should keep all matches.
	results := idx.RegexSearch(Query{Regex: "prom_.*"}, SearchOptions{MinScore: 0.5})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// MinScore above 1.0 should return nothing.
	results = idx.RegexSearch(Query{Regex: "prom_.*"}, SearchOptions{MinScore: 1.1})
	if len(results) != 0 {
		t.Fatalf("expected 0 results with MinScore 1.1, got %d", len(results))
	}
}

func TestRegexSearch_EmptyRegex(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	// Empty regex with category filter should fall back to exact match.
	results := idx.RegexSearch(Query{Category: "observability"}, SearchOptions{})
	if len(results) != 3 {
		t.Fatalf("expected 3 results (2 prom + 1 grafana), got %d", len(results))
	}
	for _, r := range results {
		if r.Score != 1.0 {
			t.Fatalf("expected score 1.0 for exact match fallback, got %f", r.Score)
		}
	}
}

func TestRegexSearch_NoMatch(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RegexSearch(Query{Regex: "xyznonexistent"}, SearchOptions{})
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
