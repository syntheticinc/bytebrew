package domain

// StructuredOutput represents a structured data block displayed to the user.
// Used for summary tables, action buttons, and non-blocking user-input forms.
type StructuredOutput struct {
	OutputType  string             `json:"output_type"` // "summary_table", "form", "info"
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Rows        []StructuredRow    `json:"rows,omitempty"`
	Actions     []StructuredAction `json:"actions,omitempty"`
	Questions   []Question         `json:"questions,omitempty"` // form-mode input fields
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

// Question represents a single form input field. Used when StructuredOutput
// is in form mode (output_type="form"). The tool that emits a form-mode
// StructuredOutput is non-blocking: the agent's turn ends after emission and
// the user's reply arrives as the next chat message.
type Question struct {
	ID      string           `json:"id"`                // stable identifier returned with the answer
	Label   string           `json:"label"`             // human-readable prompt text
	Type    string           `json:"type"`              // "text", "select", "multiselect"
	Options []QuestionOption `json:"options,omitempty"` // required for select/multiselect (2-5)
	Default string           `json:"default,omitempty"` // optional default value
}

// QuestionOption represents a selectable option for a select/multiselect Question.
type QuestionOption struct {
	Label string `json:"label"`           // display label
	Value string `json:"value,omitempty"` // machine-readable value (defaults to Label if empty)
}
