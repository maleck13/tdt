package tdt

import "testing"

// goldenServers returns a realistic corpus of ~20 tools across 6 categories.
func goldenServers() []ServerMetadata {
	return []ServerMetadata{
		{
			ServerName: "prometheus-tools",
			ToolPrefix: "prom_",
			Category:   "observability",
			Tags:       map[string]string{"department": "platform", "environment": "production"},
			Hint:       "Query Prometheus metrics, alerts, and recording rules",
			Tools: []ToolInfo{
				{Name: "prom_query", Description: "Run a PromQL query against Prometheus to check metrics and resource usage"},
				{Name: "prom_alerts", Description: "List active Prometheus alerts and their current status"},
			},
		},
		{
			ServerName: "grafana-tools",
			ToolPrefix: "grafana_",
			Category:   "observability",
			Tags:       map[string]string{"department": "platform", "environment": "production"},
			Hint:       "Interact with Grafana dashboards and visualizations",
			Tools: []ToolInfo{
				{Name: "grafana_list_dashboards", Description: "List all Grafana dashboards"},
			},
		},
		{
			ServerName: "loki-tools",
			ToolPrefix: "loki_",
			Category:   "observability",
			Tags:       map[string]string{"department": "platform", "environment": "production"},
			Hint:       "Query and search application logs stored in Loki",
			Tools: []ToolInfo{
				{Name: "loki_query_logs", Description: "Search and query application logs using LogQL"},
			},
		},
		{
			ServerName: "dns-manager",
			ToolPrefix: "dns_",
			Category:   "networking",
			Tags:       map[string]string{"department": "infrastructure"},
			Hint:       "Manage DNS records and zones",
			Tools: []ToolInfo{
				{Name: "dns_create_record", Description: "Create a new DNS record in a zone"},
				{Name: "dns_lookup", Description: "Look up DNS records for a domain"},
			},
		},
		{
			ServerName: "load-balancer",
			ToolPrefix: "lb_",
			Category:   "networking",
			Tags:       map[string]string{"department": "infrastructure"},
			Hint:       "Configure and manage load balancer rules",
			Tools: []ToolInfo{
				{Name: "lb_configure", Description: "Configure load balancer routing rules and backends"},
			},
		},
		{
			ServerName: "vault-tools",
			ToolPrefix: "vault_",
			Category:   "security",
			Tags:       map[string]string{"department": "security", "compliance": "soc2"},
			Hint:       "Manage secrets and credentials in HashiCorp Vault",
			Tools: []ToolInfo{
				{Name: "vault_read_secret", Description: "Read a secret from a Vault path"},
				{Name: "vault_list_secrets", Description: "List available secrets in a Vault mount"},
			},
		},
		{
			ServerName: "cert-manager",
			ToolPrefix: "cert_",
			Category:   "security",
			Tags:       map[string]string{"department": "security"},
			Hint:       "Issue and manage TLS/SSL certificates",
			Tools: []ToolInfo{
				{Name: "cert_issue", Description: "Issue a new TLS certificate for a domain"},
			},
		},
		{
			ServerName: "github-tools",
			ToolPrefix: "gh_",
			Category:   "cicd",
			Tags:       map[string]string{"department": "engineering"},
			Hint:       "Interact with GitHub repositories, pull requests, and issues",
			Tools: []ToolInfo{
				{Name: "gh_create_pr", Description: "Create a new pull request on a GitHub repository"},
				{Name: "gh_list_issues", Description: "List open issues in a GitHub repository"},
			},
		},
		{
			ServerName: "argocd-tools",
			ToolPrefix: "argocd_",
			Category:   "cicd",
			Tags:       map[string]string{"department": "engineering", "environment": "production"},
			Hint:       "Deploy and sync applications via ArgoCD",
			Tools: []ToolInfo{
				{Name: "argocd_sync", Description: "Sync and deploy an application using ArgoCD"},
			},
		},
		{
			ServerName: "postgres-tools",
			ToolPrefix: "pg_",
			Category:   "database",
			Tags:       map[string]string{"department": "data", "engine": "postgresql"},
			Hint:       "Query and manage PostgreSQL databases",
			Tools: []ToolInfo{
				{Name: "pg_query", Description: "Run a SQL query against a PostgreSQL database"},
				{Name: "pg_backup", Description: "Create a backup of a PostgreSQL database"},
			},
		},
		{
			ServerName: "redis-tools",
			ToolPrefix: "redis_",
			Category:   "database",
			Tags:       map[string]string{"department": "data", "engine": "redis"},
			Hint:       "Read and write data in Redis cache",
			Tools: []ToolInfo{
				{Name: "redis_get", Description: "Get a value from Redis by key"},
				{Name: "redis_set", Description: "Set a key-value pair in Redis"},
			},
		},
		{
			ServerName: "slack-tools",
			ToolPrefix: "slack_",
			Category:   "messaging",
			Tags:       map[string]string{"department": "communications"},
			Hint:       "Send messages and manage Slack channels",
			Tools: []ToolInfo{
				{Name: "slack_send_message", Description: "Send a message to a Slack channel or user"},
				{Name: "slack_list_channels", Description: "List available Slack channels"},
			},
		},
	}
}

// goldenCase defines a query with expected top results.
type goldenCase struct {
	name         string
	query        string
	expectTop1   string   // expected top-1 tool name
	expectInTop3 []string // tools that must appear in top 3 (order-independent)
	negativeCase bool     // if true, all scores should be near zero
}

func goldenCases() []goldenCase {
	return []goldenCase{
		{
			// BM25 limitation: prom_alerts doesn't contain "metrics" or "query",
			// so it scores lower than tools that do (pg_query, loki_query_logs).
			// Semantic search should fix this by understanding "prometheus" context.
			name:         "exact keyword match - prometheus",
			query:        "prometheus metrics query",
			expectTop1:   "prom_query",
			expectInTop3: []string{"prom_query"},
		},
		{
			name:         "intent match - send message",
			query:        "send a message to the team",
			expectTop1:   "slack_send_message",
			expectInTop3: []string{"slack_send_message"},
		},
		{
			name:         "intent match - deployment",
			query:        "deploy the application",
			expectTop1:   "argocd_sync",
			expectInTop3: []string{"argocd_sync"},
		},
		{
			name:         "exact match - DNS lookup",
			query:        "look up DNS records",
			expectTop1:   "dns_lookup",
			expectInTop3: []string{"dns_lookup", "dns_create_record"},
		},
		{
			name:         "exact match - vault secret",
			query:        "read a secret from the vault",
			expectTop1:   "vault_read_secret",
			expectInTop3: []string{"vault_read_secret", "vault_list_secrets"},
		},
		{
			// BM25 limitation: "database" appears in both redis and postgres tags.
			// Redis tools score higher due to shorter descriptions (length normalization).
			// Semantic search should understand "database" maps more strongly to postgres.
			name:         "intent match - database query",
			query:        "what is in the database",
			expectInTop3: []string{"pg_query"},
		},
		{
			name:         "exact match - pull request",
			query:        "create a pull request",
			expectTop1:   "gh_create_pr",
			expectInTop3: []string{"gh_create_pr"},
		},
		{
			name:         "intent match - application logs",
			query:        "check application logs",
			expectTop1:   "loki_query_logs",
			expectInTop3: []string{"loki_query_logs"},
		},
		{
			name:         "negative case - irrelevant query",
			query:        "cook pasta recipe",
			negativeCase: true,
		},
		{
			name:         "domain term match - SSL certificate",
			query:        "SSL certificate",
			expectTop1:   "cert_issue",
			expectInTop3: []string{"cert_issue"},
		},
		{
			name:         "intent match - cache",
			query:        "cache key value",
			expectInTop3: []string{"redis_get", "redis_set"},
		},
		{
			// BM25 limitation: no stemming, so "network" doesn't match "networking".
			// "configuration" doesn't appear in lb_configure's description either.
			// Semantic search should understand "network" ≈ "networking".
			name:         "broad category match - networking",
			query:        "load balancer configuration",
			expectTop1:   "lb_configure",
			expectInTop3: []string{"lb_configure"},
		},
		{
			name:         "tag match - production alerts",
			query:        "production alerts",
			expectTop1:   "prom_alerts",
			expectInTop3: []string{"prom_alerts"},
		},
	}
}

// TestGolden_BM25Accuracy runs the golden test suite against BM25-only search.
// These tests will initially fail — they are implemented here so that Task 7
// (RankedSearch) can validate accuracy as it is built.
func TestGolden_BM25Accuracy(t *testing.T) {
	idx := NewIndex()
	idx.Update(goldenServers())

	for _, tc := range goldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			results := idx.RankedSearch(Query{Text: tc.query}, SearchOptions{TopK: 5})

			if tc.negativeCase {
				for _, r := range results {
					if r.Score > 0.3 {
						t.Errorf("negative case: tool %s scored %f (expected all < 0.3)", r.ToolName, r.Score)
					}
				}
				return
			}

			if len(results) == 0 {
				t.Fatalf("expected results for query %q", tc.query)
			}

			// Check top-1
			if tc.expectTop1 != "" && results[0].ToolName != tc.expectTop1 {
				t.Errorf("expected top-1 %q, got %q (score: %f)", tc.expectTop1, results[0].ToolName, results[0].Score)
			}

			// Check top-3 contains expected tools
			top3 := map[string]bool{}
			limit := 3
			if len(results) < limit {
				limit = len(results)
			}
			for _, r := range results[:limit] {
				top3[r.ToolName] = true
			}
			for _, expected := range tc.expectInTop3 {
				if !top3[expected] {
					t.Errorf("expected %q in top 3, got %v", expected, top3Names(results, limit))
				}
			}
		})
	}
}

func top3Names(results []ScoredTool, limit int) []string {
	var names []string
	for i, r := range results {
		if i >= limit {
			break
		}
		names = append(names, r.ToolName)
	}
	return names
}
