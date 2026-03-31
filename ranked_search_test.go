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

func TestRankedSearch_CommaSplitQuery_MaxScore(t *testing.T) {
	idx := NewIndex()
	idx.Update(goldenServers())

	// Query with two unrelated intents separated by a comma.
	// "send a message" should match slack tools; "DNS records" should match dns tools.
	// Each tool should get its max score across sub-queries, not a diluted combined score.
	results := idx.RankedSearch(Query{Text: "send a message, DNS records"}, SearchOptions{TopK: 5})
	if len(results) == 0 {
		t.Fatal("expected results")
	}

	// Both slack and dns tools should appear in results.
	found := map[string]bool{}
	for _, r := range results {
		found[r.ToolName] = true
	}
	for _, want := range []string{"slack_send_message", "dns_lookup"} {
		if !found[want] {
			names := make([]string, len(results))
			for i, r := range results {
				names[i] = r.ToolName
			}
			t.Errorf("expected %q in results, got %v", want, names)
		}
	}
}

func TestRankedSearch_CommaSplitQuery_SingleSegment(t *testing.T) {
	idx := NewIndex()
	idx.Update(goldenServers())

	// A query without commas should behave identically to before.
	withoutComma := idx.RankedSearch(Query{Text: "prometheus metrics"}, SearchOptions{TopK: 3})
	if len(withoutComma) == 0 {
		t.Fatal("expected results")
	}
	if withoutComma[0].ToolName != "prom_query" {
		t.Fatalf("expected prom_query first, got %s", withoutComma[0].ToolName)
	}
}

func TestRankedSearch_DefaultMinScore(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	// With default MinScore (0 → DefaultMinScore), weak matches should be filtered.
	results := idx.RankedSearch(Query{Text: "prometheus metrics"}, SearchOptions{})
	for _, r := range results {
		if r.Score < DefaultMinScore {
			t.Fatalf("result %s has score %f below DefaultMinScore %f", r.ToolName, r.Score, DefaultMinScore)
		}
	}
}

func TestRankedSearch_CommaQuery_FiltersIrrelevantUtilityTools(t *testing.T) {
	// Regression test: a multi-intent query like "book a restaurant, check availability,
	// send message to contacts" should NOT return generic utility tools like time, add,
	// headers, pi, dozen, slow, etc.
	idx := NewIndex()
	idx.Update(gatewayTestServers())

	results := idx.RankedSearch(Query{Text: "book a restaurant, check availability, send message to contacts"}, SearchOptions{})

	relevant := map[string]bool{
		"restaurants":     true,
		"book_restaurant": true,
		"friends":         true,
		"greet":           true,
		"hello_world":     true,
	}
	irrelevant := map[string]bool{
		"time": true, "add": true, "dozen": true, "pi": true,
		"get_weather": true, "headers": true, "get_headers": true,
		"auth1234": true, "slow": true, "add_tool": true,
	}

	for _, r := range results {
		if irrelevant[r.ToolName] {
			t.Errorf("irrelevant tool %q should not appear in results (score: %f)", r.ToolName, r.Score)
		}
	}

	// At least the core relevant tools should appear.
	found := map[string]bool{}
	for _, r := range results {
		found[r.ToolName] = true
	}
	for _, want := range []string{"restaurants", "book_restaurant", "friends"} {
		if !found[want] {
			t.Errorf("expected relevant tool %q in results", want)
		}
	}
	_ = relevant // used for documentation
}

// gatewayTestServers mimics the mcp-gateway test server corpus.
func gatewayTestServers() []ServerMetadata {
	return []ServerMetadata{
		{
			ServerName: "test1",
			ToolPrefix: "test1_",
			Tools: []ToolInfo{
				{Name: "greet", Description: "Say hi to someone"},
				{Name: "time", Description: "Get current time"},
				{Name: "slow", Description: "Wait N seconds"},
				{Name: "headers", Description: "Get HTTP headers"},
				{Name: "friends", Description: "List contact information of friends"},
				{Name: "add_tool", Description: "Dynamically add a new tool"},
			},
		},
		{
			ServerName: "test2",
			ToolPrefix: "test2_",
			Tools: []ToolInfo{
				{Name: "hello_world", Description: "Say hello to someone"},
				{Name: "time", Description: "Get current time"},
				{Name: "headers", Description: "Get HTTP headers"},
				{Name: "auth1234", Description: "Check authorization header"},
				{Name: "slow", Description: "Wait N seconds"},
			},
		},
		{
			ServerName: "test3",
			ToolPrefix: "test3_",
			Tools: []ToolInfo{
				{Name: "time", Description: "Get current time"},
				{Name: "add", Description: "Add two numbers"},
				{Name: "dozen", Description: "Return 12"},
				{Name: "pi", Description: "Return 3.1415"},
				{Name: "get_weather", Description: "Get current weather for a specific city"},
				{Name: "slow", Description: "Wait N seconds"},
				{Name: "get_headers", Description: "Get HTTP headers"},
				{Name: "restaurants", Description: "List available restaurants in a city"},
				{Name: "book_restaurant", Description: "Book or reserve a table at a restaurant in a city"},
			},
		},
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

	// Use explicit low MinScore since mock embedder produces low RRF scores.
	results := idx.RankedSearch(Query{Text: "prometheus metrics"}, SearchOptions{MinScore: 0.001})
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

	results := idx.RankedSearch(Query{Text: "PromQL query alerts"}, SearchOptions{})
	if len(results) == 0 {
		t.Fatal("expected BM25 fallback results despite embedder failure")
	}
}
