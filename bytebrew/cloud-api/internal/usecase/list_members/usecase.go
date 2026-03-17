package list_members

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// MemberLister lists team members with their emails.
type MemberLister interface {
	ListMembers(ctx context.Context, teamID string) ([]domain.TeamMemberWithEmail, error)
}

// InviteLister lists pending invites for a team.
type InviteLister interface {
	ListPendingInvites(ctx context.Context, teamID string) ([]*domain.TeamInvite, error)
}

// TeamByUserReader reads the team a user belongs to.
type TeamByUserReader interface {
	GetTeamByUserID(ctx context.Context, userID string) (*domain.Team, error)
}

// Input is the list members request.
type Input struct {
	UserID string
}

// Output is the list members response.
type Output struct {
	TeamID   string
	TeamName string
	MaxSeats int
	Members  []domain.TeamMemberWithEmail
	Invites  []*domain.TeamInvite
}

// Usecase handles listing team members and pending invites.
type Usecase struct {
	teamByUserReader TeamByUserReader
	memberLister     MemberLister
	inviteLister     InviteLister
}

// New creates a new ListMembers usecase.
func New(teamByUserReader TeamByUserReader, memberLister MemberLister, inviteLister InviteLister) *Usecase {
	return &Usecase{
		teamByUserReader: teamByUserReader,
		memberLister:     memberLister,
		inviteLister:     inviteLister,
	}
}

// Execute lists all members and pending invites for the user's team.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.UserID == "" {
		return nil, errors.InvalidInput("user ID is required")
	}

	team, err := u.teamByUserReader.GetTeamByUserID(ctx, input.UserID)
	if err != nil {
		return nil, errors.Internal("get team by user", err)
	}
	if team == nil {
		return nil, errors.NotFound("you are not a member of any team")
	}

	members, err := u.memberLister.ListMembers(ctx, team.ID)
	if err != nil {
		return nil, errors.Internal("list members", err)
	}

	invites, err := u.inviteLister.ListPendingInvites(ctx, team.ID)
	if err != nil {
		return nil, errors.Internal("list pending invites", err)
	}

	return &Output{
		TeamID:   team.ID,
		TeamName: team.Name,
		MaxSeats: team.MaxSeats,
		Members:  members,
		Invites:  invites,
	}, nil
}
