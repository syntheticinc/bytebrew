package domain

// ToolType represents the classification of a tool by execution location
type ToolType string

const (
	// ToolTypeProxied indicates tools executed on the client side
	ToolTypeProxied ToolType = "proxied"
	// ToolTypeServerSide indicates tools executed on the server side
	ToolTypeServerSide ToolType = "server_side"
)

// ToolClassifier defines the interface for classifying tools by their type
type ToolClassifier interface {
	ClassifyTool(toolName string) ToolType
}
