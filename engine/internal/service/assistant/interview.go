package assistant

import (
	"fmt"
	"strings"
)

// InterviewState tracks the state of a clarifying interview.
type InterviewState struct {
	Channels     []string // chat, email, telegram, etc.
	Queries      []string // typical user queries
	Integrations []string // CRM, Google Sheets, etc.
	DataSources  []string // databases, APIs, etc.
	AgentCount   int      // desired number of agents
	SchemaName   string   // name for the schema
	Answered     map[string]bool
}

// NewInterviewState creates a new interview state.
func NewInterviewState() *InterviewState {
	return &InterviewState{
		Answered: make(map[string]bool),
	}
}

// NextQuestion returns the next clarifying question, or empty if interview is complete.
func (s *InterviewState) NextQuestion() string {
	if !s.Answered["channels"] {
		return "What channels will users interact through? (e.g., website chat, email, telegram, API)"
	}
	if !s.Answered["queries"] {
		return "What are the typical questions or tasks your users will have?"
	}
	if !s.Answered["integrations"] {
		return "Do you need any external integrations? (e.g., CRM, Google Sheets, databases, APIs)"
	}
	if !s.Answered["name"] {
		return "What would you like to name this workflow?"
	}
	return "" // interview complete
}

// ProcessAnswer processes a user's answer and updates the state.
func (s *InterviewState) ProcessAnswer(question, answer string) {
	lower := strings.ToLower(question)

	switch {
	case strings.Contains(lower, "channel"):
		s.Channels = parseListAnswer(answer)
		s.Answered["channels"] = true
	case strings.Contains(lower, "question") || strings.Contains(lower, "task"):
		s.Queries = parseListAnswer(answer)
		s.Answered["queries"] = true
	case strings.Contains(lower, "integration"):
		s.Integrations = parseListAnswer(answer)
		s.Answered["integrations"] = true
	case strings.Contains(lower, "name"):
		s.SchemaName = strings.TrimSpace(answer)
		s.Answered["name"] = true
	}
}

// IsComplete returns true if enough information has been gathered.
func (s *InterviewState) IsComplete() bool {
	return s.Answered["channels"] && s.Answered["queries"]
}

// Summary returns a summary of gathered information.
func (s *InterviewState) Summary() string {
	var parts []string
	if len(s.Channels) > 0 {
		parts = append(parts, fmt.Sprintf("Channels: %s", strings.Join(s.Channels, ", ")))
	}
	if len(s.Queries) > 0 {
		parts = append(parts, fmt.Sprintf("Typical queries: %s", strings.Join(s.Queries, ", ")))
	}
	if len(s.Integrations) > 0 {
		parts = append(parts, fmt.Sprintf("Integrations: %s", strings.Join(s.Integrations, ", ")))
	}
	if s.SchemaName != "" {
		parts = append(parts, fmt.Sprintf("Schema name: %s", s.SchemaName))
	}
	return strings.Join(parts, "\n")
}

func parseListAnswer(answer string) []string {
	// Split by comma, semicolon, or "and"
	answer = strings.ReplaceAll(answer, " and ", ",")
	answer = strings.ReplaceAll(answer, ";", ",")
	parts := strings.Split(answer, ",")

	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
