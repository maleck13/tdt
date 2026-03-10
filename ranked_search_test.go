package tdt

import "testing"

func TestRankedSearch_BM25Only_RanksRelevantToolsFirst(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Text: "prometheus metrics query"}, SearchOptions{})
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].ToolName != "prom_query" {
		t.Fatalf("expected prom_query first, got %s", results[0].ToolName)
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive score, got %f", results[0].Score)
	}
}

func TestRankedSearch_BM25Only_TopK(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Text: "query"}, SearchOptions{TopK: 2})
	if len(results) > 2 {
		t.Fatalf("expected at most 2 results, got %d", len(results))
	}
}

func TestRankedSearch_BM25Only_MinScore(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Text: "prometheus metrics query"}, SearchOptions{MinScore: 0.5})
	for _, r := range results {
		if r.Score < 0.5 {
			t.Fatalf("result %s has score %f below MinScore 0.5", r.ToolName, r.Score)
		}
	}
}

func TestRankedSearch_BM25Only_PreFilterByCategory(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RankedSearch(Query{
		Category: "networking",
		Text:     "create record",
	}, SearchOptions{})

	for _, r := range results {
		if r.ServerName != "dns-manager" {
			t.Fatalf("expected only dns-manager results, got %s", r.ServerName)
		}
	}
}

func TestRankedSearch_EmptyText_FallsBackToExactMatch(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Category: "observability"}, SearchOptions{})
	if len(results) != 3 {
		t.Fatalf("expected 3 results (2 prom + 1 grafana), got %d", len(results))
	}
	for _, r := range results {
		if r.Score != 1.0 {
			t.Fatalf("expected score 1.0 for exact match fallback, got %f", r.Score)
		}
	}
}

func TestRankedSearch_NoMatch(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Text: "xyznonexistent"}, SearchOptions{MinScore: 0.01})
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
