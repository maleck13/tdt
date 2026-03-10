package tdt

import (
	"testing"
)

func TestRRF_CombinesTwoRankedLists(t *testing.T) {
	bm25Scores := []toolScore{
		{toolName: "a", serverName: "s1", score: 10.0},
		{toolName: "b", serverName: "s1", score: 5.0},
		{toolName: "c", serverName: "s2", score: 1.0},
	}
	semanticScores := []toolScore{
		{toolName: "c", serverName: "s2", score: 0.9},
		{toolName: "a", serverName: "s1", score: 0.5},
		{toolName: "b", serverName: "s1", score: 0.1},
	}

	result := combineRRF(bm25Scores, semanticScores, 60)

	// "a" is rank 1 in BM25, rank 2 in semantic -> 1/61 + 1/62
	// "c" is rank 3 in BM25, rank 1 in semantic -> 1/63 + 1/61
	// "a" should beat "c" slightly because 1/61 > 1/63
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	if result[0].toolName != "a" {
		t.Fatalf("expected 'a' first, got %q", result[0].toolName)
	}
}

func TestRRF_HandlesDisjointLists(t *testing.T) {
	bm25Scores := []toolScore{
		{toolName: "a", serverName: "s1", score: 10.0},
	}
	semanticScores := []toolScore{
		{toolName: "b", serverName: "s1", score: 0.9},
	}

	result := combineRRF(bm25Scores, semanticScores, 60)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
}

func TestNormalizeBM25Scores(t *testing.T) {
	scores := []toolScore{
		{toolName: "a", score: 10.0},
		{toolName: "b", score: 5.0},
		{toolName: "c", score: 0.0},
	}
	normalized := normalizeBM25(scores)
	if normalized[0].score != 1.0 {
		t.Fatalf("expected top score 1.0, got %f", normalized[0].score)
	}
	if normalized[1].score != 0.5 {
		t.Fatalf("expected second score 0.5, got %f", normalized[1].score)
	}
	if normalized[2].score != 0.0 {
		t.Fatalf("expected third score 0.0, got %f", normalized[2].score)
	}
}

func TestNormalizeBM25Scores_AllZero(t *testing.T) {
	scores := []toolScore{
		{toolName: "a", score: 0.0},
		{toolName: "b", score: 0.0},
	}
	normalized := normalizeBM25(scores)
	for _, s := range normalized {
		if s.score != 0.0 {
			t.Fatalf("expected 0.0, got %f", s.score)
		}
	}
}
