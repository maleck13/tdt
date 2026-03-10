package tdt

import (
	"context"
	"fmt"
	"math"
	"testing"

	chromem "github.com/philippgille/chromem-go"
)

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

// mockEmbeddingFunc returns a deterministic embedding based on text length.
// Not semantically meaningful but lets us verify the hybrid path runs.
func mockEmbeddingFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		v := make([]float32, 3)
		n := float32(len(text) % 10)
		v[0] = n / 10.0
		v[1] = (10 - n) / 10.0
		v[2] = 0.5
		mag := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
		for i := range v {
			v[i] /= mag
		}
		return v, nil
	}
}

func TestRankedSearch_Hybrid_UsesEmbedder(t *testing.T) {
	idx := NewIndexWithEmbedder(mockEmbeddingFunc())
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Text: "prometheus metrics"}, SearchOptions{})
	if len(results) == 0 {
		t.Fatal("expected results from hybrid search")
	}
	for _, r := range results {
		if r.Score <= 0 {
			t.Fatalf("expected positive score, got %f for %s", r.Score, r.ToolName)
		}
	}
}

func TestRankedSearch_Hybrid_FallsBackOnEmbedderError(t *testing.T) {
	failingEmbedder := chromem.EmbeddingFunc(func(ctx context.Context, text string) ([]float32, error) {
		return nil, fmt.Errorf("embedding service unavailable")
	})
	idx := NewIndexWithEmbedder(failingEmbedder)
	idx.Update(testServers())

	results := idx.RankedSearch(Query{Text: "prometheus metrics"}, SearchOptions{})
	if len(results) == 0 {
		t.Fatal("expected BM25 fallback results despite embedder failure")
	}
}
