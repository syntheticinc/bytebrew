package accept_invite

import (
	"context"
	"strings"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// InviteReader reads invite data by token.
type InviteReader interface {
	GetInviteByToken(ctx context.Context, token string) (*domain.TeamInvite, error)
}

// InviteUpdater updates the status of an invite.
type InviteUpdater interface {
	UpdateInviteStatus(ctx context.Context, inviteID string, status domain.InviteStatus) error
}

// UserFinder finds a user by email.
type UserFinder interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// MemberAdder adds a member to a team.
type MemberAdder interface {
	AddMember(ctx context.Context, member *domain.TeamMember) (*domain.TeamMember, error)
}

// Input is the accept invite request.
type Input struct {
	Token       string
	CallerEmail string // Email of the authenticated user making the request
}

// Output is the accept invite response.
type Output struct {
	TeamID string
	UserID string
}

// Usecase handles accepting a team invitation.
type Usecase struct {
	inviteReader  InviteReader
	inviteUpdater InviteUpdater
	userFinder    UserFinder
	memberAdder   MemberAdder
	nowFunc       func() time.Time
}

// New creates a new AcceptInvite usecase.
func New(inviteReader InviteReader, inviteUpdater InviteUpdater, userFinder UserFinder, memberAdder MemberAdder) *Usecase {
	return &Usecase{
		inviteReader:  inviteReader,
		inviteUpdater: inviteUpdater,
		userFinder:    userFinder,
		memberAdder:   memberAdder,
		nowFunc:       time.Now,
	}
}

// Execute accepts a team invitation by token.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.Token == "" {
		return nil, errors.InvalidInput("invite token is required")
	}
	if input.CallerEmail == "" {
		return nil, errors.InvalidInput("caller email is required")
	}

	invite, err := u.inviteReader.GetInviteByToken(ctx, input.Token)
	if err != nil {
		return nil, errors.Internal("get invite", err)
	}
	if invite == nil {
		return nil, errors.NotFound("invite not found")
	}

	if invite.Status != domain.InviteStatusPending {
		return nil, errors.InvalidInput("invite is no longer pending")
	}

	if u.nowFunc().After(invite.ExpiresAt) {
		return nil, errors.Forbidden("invite has expired")
	}

	if !strings.EqualFold(invite.Email, input.CallerEmail) {
		return nil, errors.Forbidden("invite is for a different email")
	}

	user, err := u.userFinder.GetByEmail(ctx, invite.Email)
	if err != nil {
		return nil, errors.Internal("find user by email", err)
	}
	if user == nil {
		return nil, errors.NotFound("no user account found for this email, please register first")
	}

	_, err = u.memberAdder.AddMember(ctx, &domain.TeamMember{
		TeamID: invite.TeamID,
		UserID: user.ID,
		Role:   domain.TeamRoleMember,
	})
	if err != nil {
		return nil, errors.Internal("add team member", err)
	}

	if err := u.inviteUpdater.UpdateInviteStatus(ctx, invite.ID, domain.InviteStatusAccepted); err != nil {
		return nil, errors.Internal("update invite status", err)
	}

	return &Output{
		TeamID: invite.TeamID,
		UserID: user.ID,
	}, nil
}
