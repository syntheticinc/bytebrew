package domain

import (
	"fmt"
	"time"
)

// TeamRole represents a member's role within a team.
type TeamRole string

const (
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

// InviteStatus represents the current state of a team invite.
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// Team represents a team that groups users under a Teams subscription.
type Team struct {
	ID        string
	Name      string
	OwnerID   string
	MaxSeats  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TeamMember represents a user's membership in a team.
type TeamMember struct {
	ID       string
	TeamID   string
	UserID   string
	Role     TeamRole
	JoinedAt time.Time
}

// TeamInvite represents a pending invitation to join a team.
type TeamInvite struct {
	ID        string
	TeamID    string
	Email     string
	InvitedBy string
	Token     string
	Status    InviteStatus
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewTeam creates a new Team with validation.
func NewTeam(name, ownerID string, maxSeats int) (*Team, error) {
	if name == "" {
		return nil, fmt.Errorf("team name is required")
	}
	if ownerID == "" {
		return nil, fmt.Errorf("owner ID is required")
	}
	if maxSeats <= 0 {
		return nil, fmt.Errorf("max seats must be positive")
	}

	now := time.Now()
	return &Team{
		Name:      name,
		OwnerID:   ownerID,
		MaxSeats:  maxSeats,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// TeamMemberWithEmail holds a team member with their email address.
type TeamMemberWithEmail struct {
	TeamMember
	Email string
}

// IsExpired returns true if the invite has passed its expiration time.
func (i *TeamInvite) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}
