package chat_session_create

import (
	"context"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// ChatSessionRepository defines interface for chat session persistence operations
type ChatSessionRepository interface {
	Create(ctx context.Context, session *domain.ChatSession) error
}

// Input represents input for chat session creation
type Input struct {
	UserID    string
	ProjectID *string
}

// Output represents output from chat session creation
type Output struct {
	SessionID string
}

// Usecase handles chat session creation
type Usecase struct {
	sessionRepo ChatSessionRepository
}

// New creates a new Chat Session Create use case
func New(sessionRepo ChatSessionRepository) (*Usecase, error) {
	if sessionRepo == nil {
		return nil, errors.New(errors.CodeInvalidInput, "session repository is required")
	}

	return &Usecase{
		sessionRepo: sessionRepo,
	}, nil
}

// Execute creates a new chat session
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	slog.InfoContext(ctx, "creating chat session", "user_id", input.UserID)

	if input.UserID == "" {
		return nil, errors.New(errors.CodeInvalidInput, "user_id is required")
	}

	// Create domain entity
	session, err := domain.NewChatSession(input.UserID, input.ProjectID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create chat session entity", "error", err)
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid chat session data")
	}

	// Save session
	if err := u.sessionRepo.Create(ctx, session); err != nil {
		slog.ErrorContext(ctx, "failed to save chat session", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to save chat session")
	}

	slog.InfoContext(ctx, "chat session created successfully", "session_id", session.ID)

	return &Output{
		SessionID: session.ID,
	}, nil
}
