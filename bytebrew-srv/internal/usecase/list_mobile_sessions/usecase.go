package list_mobile_sessions

import (
	"context"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// FlowReader defines interface for reading active flows
type FlowReader interface {
	ListActiveFlows() []*domain.ActiveFlow
}

// MobileSession represents an active session for the mobile app
type MobileSession struct {
	SessionID      string
	ProjectKey     string
	ProjectRoot    string
	Status         domain.FlowStatus
	CurrentTask    string
	StartedAt      time.Time
	LastActivityAt time.Time
	HasAskUser     bool
	Platform       string
}

// Usecase handles listing active sessions for the mobile app
type Usecase struct {
	flowReader FlowReader
}

// New creates a new List Mobile Sessions use case
func New(flowReader FlowReader) (*Usecase, error) {
	if flowReader == nil {
		return nil, errors.New(errors.CodeInvalidInput, "flow reader is required")
	}

	return &Usecase{
		flowReader: flowReader,
	}, nil
}

// Execute returns all active sessions as MobileSession list
func (u *Usecase) Execute(ctx context.Context) ([]MobileSession, error) {
	slog.InfoContext(ctx, "listing mobile sessions")

	flows := u.flowReader.ListActiveFlows()

	sessions := make([]MobileSession, 0, len(flows))
	for _, flow := range flows {
		sessions = append(sessions, MobileSession{
			SessionID:      flow.SessionID,
			ProjectKey:     flow.ProjectKey,
			Status:         flow.Status,
			CurrentTask:    flow.Task,
			StartedAt:      flow.StartedAt,
			LastActivityAt: flow.StartedAt, // ActiveFlow does not track last activity yet
		})
	}

	slog.InfoContext(ctx, "mobile sessions listed", "count", len(sessions))

	return sessions, nil
}
