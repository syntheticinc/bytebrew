package mobile_command

import (
	"context"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// FlowReader defines interface for reading active flows
type FlowReader interface {
	Get(sessionID string) (*domain.ActiveFlow, bool)
}

// FlowCanceller defines interface for cancelling active flows
type FlowCanceller interface {
	CancelFlow(sessionID string) error
}

// MessageInjector defines interface for injecting messages into active flows
type MessageInjector interface {
	InjectMessage(sessionID string, task string) error
	InjectAskUserReply(sessionID string, question, answer string) error
}

// Usecase handles routing commands from mobile to active sessions
type Usecase struct {
	flowReader FlowReader
	canceller  FlowCanceller
	injector   MessageInjector
}

// New creates a new Mobile Command use case
func New(flowReader FlowReader, canceller FlowCanceller, injector MessageInjector) (*Usecase, error) {
	if flowReader == nil {
		return nil, errors.New(errors.CodeInvalidInput, "flow reader is required")
	}
	if canceller == nil {
		return nil, errors.New(errors.CodeInvalidInput, "flow canceller is required")
	}
	if injector == nil {
		return nil, errors.New(errors.CodeInvalidInput, "message injector is required")
	}

	return &Usecase{
		flowReader: flowReader,
		canceller:  canceller,
		injector:   injector,
	}, nil
}

// SendNewTask injects a new task into an active session
func (u *Usecase) SendNewTask(ctx context.Context, sessionID, task string) error {
	slog.InfoContext(ctx, "sending new task", "session_id", sessionID)

	if sessionID == "" {
		return errors.New(errors.CodeInvalidInput, "session_id is required")
	}
	if task == "" {
		return errors.New(errors.CodeInvalidInput, "task is required")
	}

	if _, exists := u.flowReader.Get(sessionID); !exists {
		return errors.New(errors.CodeNotFound, "session not found")
	}

	if err := u.injector.InjectMessage(sessionID, task); err != nil {
		slog.ErrorContext(ctx, "failed to inject message", "error", err, "session_id", sessionID)
		return errors.Wrap(err, errors.CodeInternal, "failed to send task")
	}

	slog.InfoContext(ctx, "task sent successfully", "session_id", sessionID)

	return nil
}

// SendAskUserReply sends a reply to an ask_user question in an active session
func (u *Usecase) SendAskUserReply(ctx context.Context, sessionID, question, answer string) error {
	slog.InfoContext(ctx, "sending ask_user reply", "session_id", sessionID)

	if sessionID == "" {
		return errors.New(errors.CodeInvalidInput, "session_id is required")
	}
	if question == "" {
		return errors.New(errors.CodeInvalidInput, "question is required")
	}
	if answer == "" {
		return errors.New(errors.CodeInvalidInput, "answer is required")
	}

	if _, exists := u.flowReader.Get(sessionID); !exists {
		return errors.New(errors.CodeNotFound, "session not found")
	}

	if err := u.injector.InjectAskUserReply(sessionID, question, answer); err != nil {
		slog.ErrorContext(ctx, "failed to inject ask_user reply", "error", err, "session_id", sessionID)
		return errors.Wrap(err, errors.CodeInternal, "failed to send ask_user reply")
	}

	slog.InfoContext(ctx, "ask_user reply sent successfully", "session_id", sessionID)

	return nil
}

// CancelSession cancels an active session
func (u *Usecase) CancelSession(ctx context.Context, sessionID string) error {
	slog.InfoContext(ctx, "cancelling session", "session_id", sessionID)

	if sessionID == "" {
		return errors.New(errors.CodeInvalidInput, "session_id is required")
	}

	if _, exists := u.flowReader.Get(sessionID); !exists {
		return errors.New(errors.CodeNotFound, "session not found")
	}

	if err := u.canceller.CancelFlow(sessionID); err != nil {
		slog.ErrorContext(ctx, "failed to cancel flow", "error", err, "session_id", sessionID)
		return errors.Wrap(err, errors.CodeInternal, "failed to cancel session")
	}

	slog.InfoContext(ctx, "session cancelled successfully", "session_id", sessionID)

	return nil
}
