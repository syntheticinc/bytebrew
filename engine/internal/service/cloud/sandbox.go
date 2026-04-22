package cloud

import (
	"fmt"
	"os"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// DeploymentMode returns the current deployment mode from BYTEBREW_MODE env var.
// Defaults to "ce" (Community Edition).
func DeploymentMode() string {
	mode := os.Getenv("BYTEBREW_MODE")
	if mode == "" {
		return "ce"
	}
	return mode
}

// Sandbox enforces Cloud security restrictions.
type Sandbox struct {
	isCloud bool
}

// NewSandbox creates a new Sandbox. Pass true for Cloud mode, false for CE.
func NewSandbox(isCloud bool) *Sandbox {
	return &Sandbox{isCloud: isCloud}
}

// ValidateToolAccess checks if a tool is allowed in the current deployment mode.
// Returns a structured error (not silent fail) if blocked.
func (s *Sandbox) ValidateToolAccess(toolName string) error {
	if !s.isCloud {
		return nil // CE mode: everything allowed
	}

	tier := domain.ClassifyToolTier(toolName)
	if tier == domain.ToolTierSelfHosted {
		return &ToolBlockedError{
			ToolName: toolName,
			Reason:   fmt.Sprintf("tool %q is blocked in Cloud deployment (Tier 3: self-hosted only)", toolName),
		}
	}

	return nil
}

// FilterTools returns only the tools that are allowed in the current deployment mode.
func (s *Sandbox) FilterTools(toolNames []string) (allowed []string, blocked []ToolBlockedError) {
	for _, name := range toolNames {
		if err := s.ValidateToolAccess(name); err != nil {
			if tbe, ok := err.(*ToolBlockedError); ok {
				blocked = append(blocked, *tbe)
			}
			continue
		}
		allowed = append(allowed, name)
	}
	return
}

// ToolBlockedError is returned when a tool is blocked by the Cloud sandbox.
type ToolBlockedError struct {
	ToolName string
	Reason   string
}

func (e *ToolBlockedError) Error() string {
	return e.Reason
}
