package tdt

import "testing"

func TestQuery_IsEmpty_WithTextOnly(t *testing.T) {
	q := Query{Text: "something"}
	if q.IsEmpty() {
		t.Fatal("query with Text should not be empty")
	}
}

func TestSearchOptions_Defaults(t *testing.T) {
	opts := SearchOptions{}
	if opts.TopK != 0 {
		t.Fatalf("expected zero-value TopK, got %d", opts.TopK)
	}
	if opts.MinScore != 0.0 {
		t.Fatalf("expected zero-value MinScore, got %f", opts.MinScore)
	}
}
