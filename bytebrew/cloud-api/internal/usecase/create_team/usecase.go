package create_team

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// TeamCreator persists a new team.
type TeamCreator interface {
	CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error)
}

// MemberAdder adds a member to a team.
type MemberAdder interface {
	AddMember(ctx context.Context, member *domain.TeamMember) (*domain.TeamMember, error)
}

// Input is the create team request.
type Input struct {
	OwnerID string
	Name    string
}

// Output is the create team response.
type Output struct {
	Team *domain.Team
}

// DefaultMaxSeats is the default maximum number of seats for a new team.
const DefaultMaxSeats = 1

// Usecase handles team creation.
type Usecase struct {
	teamCreator    TeamCreator
	memberAdder    MemberAdder
	defaultMaxSeats int
}

// New creates a new CreateTeam usecase with DefaultMaxSeats.
func New(teamCreator TeamCreator, memberAdder MemberAdder) *Usecase {
	return NewWithMaxSeats(teamCreator, memberAdder, DefaultMaxSeats)
}

// NewWithMaxSeats creates a new CreateTeam usecase with a custom max seats limit.
func NewWithMaxSeats(teamCreator TeamCreator, memberAdder MemberAdder, defaultMaxSeats int) *Usecase {
	return &Usecase{
		teamCreator:    teamCreator,
		memberAdder:    memberAdder,
		defaultMaxSeats: defaultMaxSeats,
	}
}

// Execute creates a new team and adds the owner as an admin member.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.OwnerID == "" {
		return nil, errors.InvalidInput("owner ID is required")
	}
	if input.Name == "" {
		return nil, errors.InvalidInput("team name is required")
	}

	team, err := domain.NewTeam(input.Name, input.OwnerID, u.defaultMaxSeats)
	if err != nil {
		return nil, errors.InvalidInput(err.Error())
	}

	created, err := u.teamCreator.CreateTeam(ctx, team)
	if err != nil {
		return nil, errors.Internal("create team", err)
	}

	_, err = u.memberAdder.AddMember(ctx, &domain.TeamMember{
		TeamID: created.ID,
		UserID: input.OwnerID,
		Role:   domain.TeamRoleAdmin,
	})
	if err != nil {
		return nil, errors.Internal("add owner as admin", err)
	}

	return &Output{Team: created}, nil
}
