package tdt

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestNewDiscoveryTool_ToolDefinition(t *testing.T) {
	idx := NewIndex()
	tool, _ := NewDiscoveryTool(idx)

	if tool.Name != "discover_tools" {
		t.Fatalf("expected tool name 'discover_tools', got %q", tool.Name)
	}
	if tool.Description == "" {
		t.Fatal("expected non-empty description")
	}
}

func TestNewDiscoveryTool_HandlerReturnsCatalog(t *testing.T) {
	idx := NewIndex()
	idx.Update([]ServerMetadata{
		{
			ServerName: "weather",
			Category:   "data",
			Hint:       "Weather data",
			Tags:       map[string]string{"region": "us"},
			Tools: []ToolInfo{
				{Name: "weather_get", Description: "Get weather"},
			},
		},
	})

	_, handler := NewDiscoveryTool(idx)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected non-error result")
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var catalog CatalogResponse
	if err := json.Unmarshal([]byte(textContent.Text), &catalog); err != nil {
		t.Fatalf("failed to parse catalog JSON: %v", err)
	}

	if len(catalog.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(catalog.Categories))
	}
	if catalog.Categories[0].Name != "data" {
		t.Fatalf("expected category 'data', got %q", catalog.Categories[0].Name)
	}
	if len(catalog.Categories[0].Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(catalog.Categories[0].Servers))
	}
	if catalog.Categories[0].Servers[0].Name != "weather" {
		t.Fatalf("expected server 'weather', got %q", catalog.Categories[0].Servers[0].Name)
	}
}

func TestNewDiscoveryTool_HandlerEmptyIndex(t *testing.T) {
	idx := NewIndex()
	_, handler := NewDiscoveryTool(idx)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var catalog CatalogResponse
	if err := json.Unmarshal([]byte(textContent.Text), &catalog); err != nil {
		t.Fatalf("failed to parse catalog JSON: %v", err)
	}

	if len(catalog.Categories) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(catalog.Categories))
	}
}
