package tdt

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewDiscoveryTool returns an mcp-go tool definition and handler that renders
// the catalog from the index. The tool has no input parameters.
func NewDiscoveryTool(idx *Index) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.Tool{
		Name:        "discover_tools",
		Description: "Returns a catalog of available tool categories, tags, and hints. Use this to understand what tools are available before filtering with tools/list.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]any{},
		},
		Annotations: mcp.ToolAnnotation{
			ReadOnlyHint: mcp.ToBoolPtr(true),
		},
	}

	handler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		catalog := idx.Catalog()
		data, err := json.Marshal(catalog)
		if err != nil {
			return mcp.NewToolResultError("failed to marshal catalog: " + err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}

	return tool, handler
}
