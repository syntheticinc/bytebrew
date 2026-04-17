package tools

import "context"

// ClientOperationsProxy defines the interface used by tools that delegate
// interactive operations to the client side of the gRPC bidirectional
// stream. Only ask_user needs this pattern today.
type ClientOperationsProxy interface {
	AskUserQuestionnaire(ctx context.Context, sessionID, questionsJSON string) (string, error)
}
