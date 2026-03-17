package domain

// KitSession holds session-level context for a kit.
type KitSession struct {
	SessionID   string
	ProjectRoot string
	ProjectKey  string
}

// Enrichment is additional context appended to a tool result by a kit.
type Enrichment struct {
	Content string
}
