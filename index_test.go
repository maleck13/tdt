package tdt

import (
	"testing"
)

func testServers() []ServerMetadata {
	return []ServerMetadata{
		{
			ServerName: "prometheus-tools",
			ToolPrefix: "prom_",
			Category:   "observability",
			Tags:       map[string]string{"department": "platform", "environment": "production"},
			Hint:       "Query Prometheus metrics and alerts",
			Tools: []ToolInfo{
				{Name: "prom_query", Description: "Run a PromQL query"},
				{Name: "prom_alerts", Description: "List active alerts"},
			},
		},
		{
			ServerName: "dns-manager",
			ToolPrefix: "dns_",
			Category:   "networking",
			Tags:       map[string]string{"department": "infrastructure"},
			Hint:       "Manage DNS records and zones",
			Tools: []ToolInfo{
				{Name: "dns_create_record", Description: "Create a DNS record"},
			},
		},
		{
			ServerName: "grafana-tools",
			ToolPrefix: "grafana_",
			Category:   "observability",
			Tags:       map[string]string{"department": "platform", "environment": "staging"},
			Hint:       "Query Grafana dashboards",
			Tools: []ToolInfo{
				{Name: "grafana_list_dashboards", Description: "List dashboards"},
			},
		},
		{
			ServerName: "uncategorized-server",
			ToolPrefix: "uc_",
			Category:   "",
			Tags:       nil,
			Hint:       "A server with no category or tags",
			Tools: []ToolInfo{
				{Name: "uc_do_thing", Description: "Do a thing"},
			},
		},
	}
}

func TestIndex_SearchByCategory(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.Search(Query{Category: "observability"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	names := map[string]bool{}
	for _, r := range results {
		names[r.ServerName] = true
	}
	if !names["prometheus-tools"] || !names["grafana-tools"] {
		t.Fatalf("expected prometheus-tools and grafana-tools, got %v", names)
	}
}

func TestIndex_SearchByTags(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.Search(Query{Tags: map[string]string{"department": "platform"}})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestIndex_SearchByTagsAND(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.Search(Query{Tags: map[string]string{"department": "platform", "environment": "production"}})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ServerName != "prometheus-tools" {
		t.Fatalf("expected prometheus-tools, got %s", results[0].ServerName)
	}
}

func TestIndex_SearchByCategoryAndTags(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.Search(Query{
		Category: "observability",
		Tags:     map[string]string{"environment": "staging"},
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ServerName != "grafana-tools" {
		t.Fatalf("expected grafana-tools, got %s", results[0].ServerName)
	}
}

func TestIndex_SearchEmptyQuery(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.Search(Query{})
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
}

func TestIndex_SearchNoMatch(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	results := idx.Search(Query{Category: "nonexistent"})
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestIndex_UpdateReplacesData(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	idx.Update([]ServerMetadata{
		{ServerName: "new-server", Category: "new-cat"},
	})

	results := idx.Search(Query{Category: "observability"})
	if len(results) != 0 {
		t.Fatalf("expected 0 results after update, got %d", len(results))
	}
	results = idx.Search(Query{Category: "new-cat"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestIndex_Catalog(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	catalog := idx.Catalog()

	if len(catalog.Categories) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(catalog.Categories))
	}

	catMap := map[string]CatalogCategory{}
	for _, c := range catalog.Categories {
		catMap[c.Name] = c
	}

	obs, ok := catMap["observability"]
	if !ok {
		t.Fatal("expected observability category")
	}
	if len(obs.Servers) != 2 {
		t.Fatalf("expected 2 servers in observability, got %d", len(obs.Servers))
	}

	net, ok := catMap["networking"]
	if !ok {
		t.Fatal("expected networking category")
	}
	if len(net.Servers) != 1 {
		t.Fatalf("expected 1 server in networking, got %d", len(net.Servers))
	}
}

func TestIndex_MatchingToolNames(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	names := idx.MatchingToolNames(Query{Category: "observability"})
	// prometheus-tools has 2 tools, grafana-tools has 1 = 3 total
	if len(names) != 3 {
		t.Fatalf("expected 3 tool names, got %d", len(names))
	}

	expected := map[string]bool{"prom_query": true, "prom_alerts": true, "grafana_list_dashboards": true}
	for _, n := range names {
		if !expected[n] {
			t.Fatalf("unexpected tool name %q", n)
		}
	}
}

func TestIndex_MatchingToolNamesEmptyQuery(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	names := idx.MatchingToolNames(Query{})
	// All servers: 2 + 1 + 1 + 1 = 5 tools
	if len(names) != 5 {
		t.Fatalf("expected 5 tool names, got %d", len(names))
	}
}

func TestIndex_MatchingToolNamesNoMatch(t *testing.T) {
	idx := NewIndex()
	idx.Update(testServers())

	names := idx.MatchingToolNames(Query{Category: "nonexistent"})
	if len(names) != 0 {
		t.Fatalf("expected 0 tool names, got %d", len(names))
	}
}
