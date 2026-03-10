package tdt

import (
	"reflect"
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
	want := []string{"query", "prometheus", "metrics", "alerts", "and", "recording", "rules"}
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
	want := []string{"prom", "query", "alerts"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestTokenize_Empty(t *testing.T) {
	got := tokenize("")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}
