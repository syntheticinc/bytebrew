package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/infrastructure/postgres/sqlcgen"
)

// TeamRepository implements team persistence with PostgreSQL.
type TeamRepository struct {
	queries *sqlcgen.Queries
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db sqlcgen.DBTX) *TeamRepository {
	return &TeamRepository{
		queries: sqlcgen.New(db),
	}
}

// CreateTeam inserts a new team and returns the created team.
func (r *TeamRepository) CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error) {
	ownerID, err := parseUUID(team.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("parse owner ID: %w", err)
	}
	row, err := r.queries.CreateTeam(ctx, sqlcgen.CreateTeamParams{
		Name:     team.Name,
		OwnerID:  ownerID,
		MaxSeats: int32(team.MaxSeats),
	})
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}
	return mapTeam(row), nil
}

// GetTeamByID returns a team by ID, or nil if not found.
func (r *TeamRepository) GetTeamByID(ctx context.Context, id string) (*domain.Team, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, fmt.Errorf("parse team ID: %w", err)
	}
	row, err := r.queries.GetTeamByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get team by id: %w", err)
	}
	return mapTeam(row), nil
}

// GetTeamByOwnerID returns a team by owner user ID, or nil if not found.
func (r *TeamRepository) GetTeamByOwnerID(ctx context.Context, ownerID string) (*domain.Team, error) {
	uid, err := parseUUID(ownerID)
	if err != nil {
		return nil, fmt.Errorf("parse owner ID: %w", err)
	}
	row, err := r.queries.GetTeamByOwnerID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get team by owner id: %w", err)
	}
	return mapTeam(row), nil
}

// GetTeamByUserID returns the team a user belongs to, or nil if not found.
func (r *TeamRepository) GetTeamByUserID(ctx context.Context, userID string) (*domain.Team, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user ID: %w", err)
	}
	row, err := r.queries.GetTeamByUserID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get team by user id: %w", err)
	}
	return mapTeam(row), nil
}

// AddMember inserts a new team member and returns the created member.
func (r *TeamRepository) AddMember(ctx context.Context, member *domain.TeamMember) (*domain.TeamMember, error) {
	teamID, err := parseUUID(member.TeamID)
	if err != nil {
		return nil, fmt.Errorf("parse team ID: %w", err)
	}
	userID, err := parseUUID(member.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user ID: %w", err)
	}
	row, err := r.queries.AddTeamMember(ctx, sqlcgen.AddTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
		Role:   string(member.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("add team member: %w", err)
	}
	return mapTeamMember(row), nil
}

// RemoveMember deletes a team member by team ID and user ID.
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID string) error {
	tid, err := parseUUID(teamID)
	if err != nil {
		return fmt.Errorf("parse team ID: %w", err)
	}
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.RemoveTeamMember(ctx, sqlcgen.RemoveTeamMemberParams{
		TeamID: tid,
		UserID: uid,
	})
}

// ListMembers returns all members of a team with their email addresses.
func (r *TeamRepository) ListMembers(ctx context.Context, teamID string) ([]domain.TeamMemberWithEmail, error) {
	tid, err := parseUUID(teamID)
	if err != nil {
		return nil, fmt.Errorf("parse team ID: %w", err)
	}
	rows, err := r.queries.ListTeamMembers(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}

	members := make([]domain.TeamMemberWithEmail, 0, len(rows))
	for _, row := range rows {
		members = append(members, domain.TeamMemberWithEmail{
			TeamMember: domain.TeamMember{
				ID:       uuidToString(row.ID),
				TeamID:   uuidToString(row.TeamID),
				UserID:   uuidToString(row.UserID),
				Role:     domain.TeamRole(row.Role),
				JoinedAt: timestamptzToTimeValue(row.JoinedAt),
			},
			Email: row.Email,
		})
	}
	return members, nil
}

// CountMembers returns the number of members in a team.
func (r *TeamRepository) CountMembers(ctx context.Context, teamID string) (int, error) {
	tid, err := parseUUID(teamID)
	if err != nil {
		return 0, fmt.Errorf("parse team ID: %w", err)
	}
	count, err := r.queries.CountTeamMembers(ctx, tid)
	if err != nil {
		return 0, fmt.Errorf("count team members: %w", err)
	}
	return int(count), nil
}

// CreateInvite inserts a new team invite and returns the created invite.
func (r *TeamRepository) CreateInvite(ctx context.Context, invite *domain.TeamInvite) (*domain.TeamInvite, error) {
	teamID, err := parseUUID(invite.TeamID)
	if err != nil {
		return nil, fmt.Errorf("parse team ID: %w", err)
	}
	invitedBy, err := parseUUID(invite.InvitedBy)
	if err != nil {
		return nil, fmt.Errorf("parse invited by: %w", err)
	}
	row, err := r.queries.CreateTeamInvite(ctx, sqlcgen.CreateTeamInviteParams{
		TeamID:    teamID,
		Email:     invite.Email,
		InvitedBy: invitedBy,
		Token:     invite.Token,
		Status:    string(invite.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("create team invite: %w", err)
	}
	return mapTeamInvite(row), nil
}

// GetInviteByToken returns a team invite by token, or nil if not found.
func (r *TeamRepository) GetInviteByToken(ctx context.Context, token string) (*domain.TeamInvite, error) {
	row, err := r.queries.GetTeamInviteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get invite by token: %w", err)
	}
	return mapTeamInvite(row), nil
}

// UpdateInviteStatus updates the status of a team invite.
func (r *TeamRepository) UpdateInviteStatus(ctx context.Context, inviteID string, status domain.InviteStatus) error {
	uid, err := parseUUID(inviteID)
	if err != nil {
		return fmt.Errorf("parse invite ID: %w", err)
	}
	return r.queries.UpdateInviteStatus(ctx, sqlcgen.UpdateInviteStatusParams{
		Status: string(status),
		ID:     uid,
	})
}

// ListPendingInvites returns all pending invites for a team.
func (r *TeamRepository) ListPendingInvites(ctx context.Context, teamID string) ([]*domain.TeamInvite, error) {
	tid, err := parseUUID(teamID)
	if err != nil {
		return nil, fmt.Errorf("parse team ID: %w", err)
	}
	rows, err := r.queries.ListPendingInvites(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("list pending invites: %w", err)
	}

	invites := make([]*domain.TeamInvite, 0, len(rows))
	for _, row := range rows {
		invites = append(invites, mapTeamInvite(row))
	}
	return invites, nil
}

// UpdateTeamMaxSeats updates the maximum number of seats for a team.
func (r *TeamRepository) UpdateTeamMaxSeats(ctx context.Context, teamID string, maxSeats int) error {
	tid, err := parseUUID(teamID)
	if err != nil {
		return fmt.Errorf("parse team ID: %w", err)
	}
	return r.queries.UpdateTeamMaxSeats(ctx, sqlcgen.UpdateTeamMaxSeatsParams{
		MaxSeats: int32(maxSeats),
		ID:       tid,
	})
}

func mapTeam(row sqlcgen.Team) *domain.Team {
	return &domain.Team{
		ID:        uuidToString(row.ID),
		Name:      row.Name,
		OwnerID:   uuidToString(row.OwnerID),
		MaxSeats:  int(row.MaxSeats),
		CreatedAt: timestamptzToTimeValue(row.CreatedAt),
		UpdatedAt: timestamptzToTimeValue(row.UpdatedAt),
	}
}

func mapTeamMember(row sqlcgen.TeamMember) *domain.TeamMember {
	return &domain.TeamMember{
		ID:       uuidToString(row.ID),
		TeamID:   uuidToString(row.TeamID),
		UserID:   uuidToString(row.UserID),
		Role:     domain.TeamRole(row.Role),
		JoinedAt: timestamptzToTimeValue(row.JoinedAt),
	}
}

func mapTeamInvite(row sqlcgen.TeamInvite) *domain.TeamInvite {
	return &domain.TeamInvite{
		ID:        uuidToString(row.ID),
		TeamID:    uuidToString(row.TeamID),
		Email:     row.Email,
		InvitedBy: uuidToString(row.InvitedBy),
		Token:     row.Token,
		Status:    domain.InviteStatus(row.Status),
		CreatedAt: timestamptzToTimeValue(row.CreatedAt),
		ExpiresAt: timestamptzToTimeValue(row.ExpiresAt),
	}
}
