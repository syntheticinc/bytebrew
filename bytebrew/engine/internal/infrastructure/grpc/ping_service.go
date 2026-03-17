package grpc

import (
	"context"
	"log/slog"
	"sync"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
)

// PingService manages keep-alive ping/pong mechanism for streaming connections
type PingService struct {
	mu           sync.RWMutex
	sessions     map[string]*pingSession
	pingInterval time.Duration
	stopChannels map[string]chan struct{}
}

// pingSession represents an active ping session
type pingSession struct {
	sessionID string
	sendPong  func(*pb.PongResponse) error
	lastPing  time.Time
	pingCount int
}

// NewPingService creates a new PingService
func NewPingService(pingInterval time.Duration) (*PingService, error) {
	if pingInterval <= 0 {
		return nil, errors.New(errors.CodeInvalidInput, "ping interval must be positive")
	}

	return &PingService{
		sessions:     make(map[string]*pingSession),
		stopChannels: make(map[string]chan struct{}),
		pingInterval: pingInterval,
	}, nil
}

// Start begins sending ping messages for a session
func (s *PingService) Start(ctx context.Context, sessionID string, sendPong func(*pb.PongResponse) error) error {
	if sessionID == "" {
		return errors.New(errors.CodeInvalidInput, "session_id is required")
	}
	if sendPong == nil {
		return errors.New(errors.CodeInvalidInput, "sendPong callback is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists
	if _, exists := s.sessions[sessionID]; exists {
		slog.WarnContext(ctx, "ping session already exists", "session_id", sessionID)
		return nil
	}

	// Create ping session
	session := &pingSession{
		sessionID: sessionID,
		sendPong:  sendPong,
		lastPing:  time.Now(),
		pingCount: 0,
	}
	s.sessions[sessionID] = session

	// Create stop channel
	stopChan := make(chan struct{})
	s.stopChannels[sessionID] = stopChan

	// Start ping goroutine
	go s.runPingLoop(ctx, session, stopChan)

	slog.InfoContext(ctx, "ping service started", "session_id", sessionID, "interval", s.pingInterval)
	return nil
}

// Stop stops sending ping messages for a session
func (s *PingService) Stop(sessionID string) {
	if sessionID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get stop channel
	stopChan, exists := s.stopChannels[sessionID]
	if !exists {
		return
	}

	// Signal stop
	close(stopChan)

	// Remove session
	delete(s.sessions, sessionID)
	delete(s.stopChannels, sessionID)

	slog.InfoContext(context.Background(), "ping service stopped", "session_id", sessionID)
}

// runPingLoop runs the ping loop for a session
func (s *PingService) runPingLoop(ctx context.Context, session *pingSession, stopChan chan struct{}) {
	ticker := time.NewTicker(s.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "ping loop stopped by context", "session_id", session.sessionID)
			return
		case <-stopChan:
			slog.InfoContext(ctx, "ping loop stopped by signal", "session_id", session.sessionID)
			return
		case <-ticker.C:
			// Send pong
			pong := &pb.PongResponse{
				Status:    "alive",
				Timestamp: time.Now().Unix(),
			}

			if err := session.sendPong(pong); err != nil {
				slog.ErrorContext(ctx, "failed to send pong", "session_id", session.sessionID, "error", err)
				// Don't stop on error, continue trying
			} else {
				session.lastPing = time.Now()
				session.pingCount++
				slog.DebugContext(ctx, "pong sent", "session_id", session.sessionID, "count", session.pingCount)
			}
		}
	}
}

// GetSessionCount returns the number of active ping sessions
func (s *PingService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// IsSessionActive checks if a session is active
func (s *PingService) IsSessionActive(sessionID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.sessions[sessionID]
	return exists
}
