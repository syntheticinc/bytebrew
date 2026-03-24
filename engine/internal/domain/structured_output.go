package domain

// StructuredOutput represents a structured data block displayed to the user.
// Used for summary tables, action buttons, and other rich output formats.
type StructuredOutput struct {
	OutputType  string             `json:"output_type"`            // "summary_table"
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Rows        []StructuredRow    `json:"rows,omitempty"`
	Actions     []StructuredAction `json:"actions,omitempty"`
}

// StructuredRow represents a single label-value row in a structured output block.
type StructuredRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// StructuredAction represents an interactive action button in a structured output block.
type StructuredAction struct {
	Label string `json:"label"`
	Type  string `json:"type"`  // "primary", "secondary"
	Value string `json:"value"` // machine-readable value sent back
}
