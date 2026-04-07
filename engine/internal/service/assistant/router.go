package assistant

import (
	"strings"
)

// RequestIntent represents the classified intent of a user message.
type RequestIntent string

const (
	IntentInterview RequestIntent = "interview"  // needs clarifying questions
	IntentDirect    RequestIntent = "direct"     // execute directly
	IntentAnswer    RequestIntent = "answer"     // answer a question
)

// Router classifies user messages to determine the appropriate action.
type Router struct {
	hasSchemas bool
	isFirstVisit bool
}

// NewRouter creates a new Router.
func NewRouter(hasSchemas, isFirstVisit bool) *Router {
	return &Router{hasSchemas: hasSchemas, isFirstVisit: isFirstVisit}
}

// Classify determines the intent of a user message.
func (r *Router) Classify(message string) RequestIntent {
	lower := strings.ToLower(strings.TrimSpace(message))

	// Check for escape phrases first (overrides interview)
	if containsEscape(lower) {
		return IntentDirect
	}

	// No schemas + first visit → interview
	if !r.hasSchemas && r.isFirstVisit {
		return IntentInterview
	}

	// Simple question detection
	if isQuestion(lower) {
		return IntentAnswer
	}

	// Modification/config requests → direct execution
	if isModification(lower) {
		return IntentDirect
	}

	// Check if request is vague
	if isVague(lower) {
		return IntentInterview
	}

	// Default: direct execution for clear requests
	return IntentDirect
}

// isQuestion checks if the message is a question.
func isQuestion(msg string) bool {
	questionPrefixes := []string{
		"как ", "что ", "почему ", "зачем ", "какой ", "какая ", "какие ",
		"how ", "what ", "why ", "when ", "where ", "which ", "can ",
		"is it ", "does ", "do ", "will ",
	}
	for _, prefix := range questionPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return strings.HasSuffix(msg, "?")
}

// containsEscape checks for phrases that skip interview.
func containsEscape(msg string) bool {
	escapes := []string{
		"просто сделай", "just do", "just make", "сделай это", "go ahead",
	}
	for _, esc := range escapes {
		if strings.Contains(msg, esc) {
			return true
		}
	}
	return false
}

// isModification checks if the message is a modification request.
func isModification(msg string) bool {
	modifiers := []string{
		"добавь", "удали", "измени", "обнови", "подключи", "настрой",
		"add ", "remove ", "delete ", "update ", "change ", "modify ", "connect ",
		"set ", "configure ", "enable ", "disable ",
	}
	for _, mod := range modifiers {
		if strings.Contains(msg, mod) {
			return true
		}
	}
	return false
}

// isVague checks if the message is too vague for direct execution.
func isVague(msg string) bool {
	words := strings.Fields(msg)

	// Short messages (≤15 words) without specific names are vague
	if len(words) <= 15 {
		if !hasSpecificNames(msg) {
			return true
		}
	}

	return false
}

// hasSpecificNames checks if the message contains specific agent/tool/schema names.
func hasSpecificNames(msg string) bool {
	// Common indicators of specificity
	specifics := []string{
		"agent", "trigger", "schema", "mcp", "flow", "edge", "gate",
		"агент", "триггер", "схем",
		// Tool names
		"memory", "knowledge", "escalat", "guardrail",
	}
	lower := strings.ToLower(msg)
	for _, s := range specifics {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
