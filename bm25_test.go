package tdt

import (
	"reflect"
	"strings"
	"testing"
)

func TestTokenize_Basic(t *testing.T) {
	got := tokenize("hello world")
	want := []string{"hello", "world"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_Underscores(t *testing.T) {
	got := tokenize("brave_web_search")
	want := []string{"brave", "web", "search"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_CamelCase(t *testing.T) {
	got := tokenize("getWeather")
	want := []string{"get", "weather"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_Punctuation(t *testing.T) {
	got := tokenize("Query Prometheus metrics, alerts, and recording rules.")
	// Stemmed: query→queri, metrics→metric, alerts→alert, recording→record, rules→rule
	want := []string{"queri", "prometheus", "metric", "alert", "and", "record", "rule"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_ShortTokensDropped(t *testing.T) {
	got := tokenize("I a am do it")
	// "I", "a" dropped (< 2 chars); "am", "do", "it" kept
	want := []string{"am", "do", "it"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_MixedUnderscoreAndCamelCase(t *testing.T) {
	got := tokenize("prom_queryAlerts")
	// Stemmed: query→queri, alerts→alert
	want := []string{"prom", "queri", "alert"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_Stemming(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		// "networking" and "network" should produce the same stem
		{"networking", []string{"network"}},
		{"network", []string{"network"}},
		// "deployment" and "deploy" should produce the same stem
		{"deployment", []string{"deploy"}},
		{"deploy", []string{"deploy"}},
		// "configuration" and "configure" should produce the same stem
		{"configuration", []string{"configur"}},
		{"configure", []string{"configur"}},
		// "monitoring" and "monitor" should produce the same stem
		{"monitoring", []string{"monitor"}},
		{"monitor", []string{"monitor"}},
	}
	for _, tc := range cases {
		got := tokenize(tc.input)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestTokenize_Empty(t *testing.T) {
	got := tokenize("")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestBuildCompositeText(t *testing.T) {
	sm := ServerMetadata{
		ServerName: "prom",
		Category:   "observability",
		Tags:       map[string]string{"department": "platform", "env": "prod"},
		Tools: []ToolInfo{
			{Name: "prom_query", Description: "Run a PromQL query"},
		},
	}
	got := buildCompositeText(sm, sm.Tools[0])
	for _, want := range []string{"prom_query", "Run a PromQL query", "observability", "department", "platform", "env", "prod"} {
		if !strings.Contains(got, want) {
			t.Fatalf("composite text %q missing %q", got, want)
		}
	}
}

func TestBM25Corpus_Score(t *testing.T) {
	servers := testServers()
	corpus := newBM25Corpus(servers)

	scores := corpus.score("prometheus metrics query")

	if len(scores) == 0 {
		t.Fatal("expected scores")
	}

	topIdx := 0
	for i, s := range scores {
		if s.score > scores[topIdx].score {
			topIdx = i
		}
	}
	if scores[topIdx].toolName != "prom_query" {
		t.Fatalf("expected prom_query as top result, got %s", scores[topIdx].toolName)
	}
}

func TestBM25Corpus_ScoreNoMatch(t *testing.T) {
	servers := testServers()
	corpus := newBM25Corpus(servers)

	scores := corpus.score("xyznonexistent")
	for _, s := range scores {
		if s.score != 0 {
			t.Fatalf("expected 0 score for non-matching query, got %f for %s", s.score, s.toolName)
		}
	}
}
