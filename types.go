package tdt

// ServerMetadata holds discovery metadata for a registered MCP server.
type ServerMetadata struct {
	ServerName string
	ToolPrefix string
	Category   string
	Tags       map[string]string
	Hint       string
	Tools      []ToolInfo
}

// ToolInfo holds basic info about a tool.
type ToolInfo struct {
	Name        string
	Description string
}

// Query represents a discovery filter.
type Query struct {
	Category string
	Tags     map[string]string
	Text     string // natural language query for relevance scoring
}

// IsEmpty returns true if the query has no filters set.
func (q Query) IsEmpty() bool {
	return q.Category == "" && len(q.Tags) == 0 && q.Text == ""
}

// CatalogCategory represents a category in the catalog response.
type CatalogCategory struct {
	Name    string          `json:"name"`
	Servers []CatalogServer `json:"servers"`
}

// CatalogServer represents a server entry in the catalog response.
type CatalogServer struct {
	Name string            `json:"name"`
	Hint string            `json:"hint"`
	Tags map[string]string `json:"tags"`
}

// CatalogResponse is the output of the discover_tools tool.
type CatalogResponse struct {
	Categories []CatalogCategory `json:"categories"`
}

// SearchOptions controls relevance search behavior.
type SearchOptions struct {
	TopK     int     // max results to return (0 means no limit)
	MinScore float64 // minimum score threshold (0.0 means no threshold)
}

// ScoredTool is a tool with its relevance score.
type ScoredTool struct {
	ToolName   string
	ServerName string
	Score      float64
}
