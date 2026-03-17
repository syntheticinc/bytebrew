package list_members

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockTeamByUserReader struct {
	team *domain.Team
	err  error
}

func (m *mockTeamByUserReader) GetTeamByUserID(_ context.Context, _ string) (*domain.Team, error) {
	return m.team, m.err
}

type mockMemberLister struct {
	members []domain.TeamMemberWithEmail
	err     error
}

func (m *mockMemberLister) ListMembers(_ context.Context, _ string) ([]domain.TeamMemberWithEmail, error) {
	return m.members, m.err
}

type mockInviteLister struct {
	invites []*domain.TeamInvite
	err     error
}

func (m *mockInviteLister) ListPendingInvites(_ context.Context, _ string) ([]*domain.TeamInvite, error) {
	return m.invites, m.err
}

// --- Tests ---

func TestExecute(t *testing.T) {
	defaultTeam := &domain.Team{
		ID:       "team-1",
		Name:     "My Team",
		OwnerID:  "owner-1",
		MaxSeats: 5,
	}

	defaultMembers := []domain.TeamMemberWithEmail{
		{
			TeamMember: domain.TeamMember{
				ID:     "m-1",
				TeamID: "team-1",
				UserID: "owner-1",
				Role:   domain.TeamRoleAdmin,
			},
			Email: "owner@example.com",
		},
		{
			TeamMember: domain.TeamMember{
				ID:     "m-2",
				TeamID: "team-1",
				UserID: "member-1",
				Role:   domain.TeamRoleMember,
			},
			Email: "member@example.com",
		},
	}

	defaultInvites := []*domain.TeamInvite{
		{
			ID:        "inv-1",
			TeamID:    "team-1",
			Email:     "pending@example.com",
			Status:    domain.InviteStatusPending,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
	}

	tests := []struct {
		name        string
		input       Input
		team        *domain.Team
		teamErr     error
		members     []domain.TeamMemberWithEmail
		membersErr  error
		invites     []*domain.TeamInvite
		invitesErr  error
		wantCode    string
		wantMembers int
		wantInvites int
	}{
		{
			name:        "success: returns members and pending invites",
			input:       Input{UserID: "owner-1"},
			team:        defaultTeam,
			members:     defaultMembers,
			invites:     defaultInvites,
			wantMembers: 2,
			wantInvites: 1,
		},
		{
			name:     "error: empty user ID",
			input:    Input{UserID: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: user not in any team",
			input:    Input{UserID: "stranger-1"},
			team:     nil,
			wantCode: errors.CodeNotFound,
		},
		{
			name:     "error: get team fails",
			input:    Input{UserID: "owner-1"},
			teamErr:  fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:       "error: list members fails",
			input:      Input{UserID: "owner-1"},
			team:       defaultTeam,
			membersErr: fmt.Errorf("db error"),
			wantCode:   errors.CodeInternal,
		},
		{
			name:       "error: list invites fails",
			input:      Input{UserID: "owner-1"},
			team:       defaultTeam,
			members:    defaultMembers,
			invitesErr: fmt.Errorf("db error"),
			wantCode:   errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(
				&mockTeamByUserReader{team: tt.team, err: tt.teamErr},
				&mockMemberLister{members: tt.members, err: tt.membersErr},
				&mockInviteLister{invites: tt.invites, err: tt.invitesErr},
			)

			got, err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantCode), "expected error code %s, got: %v", tt.wantCode, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, defaultTeam.ID, got.TeamID)
			assert.Equal(t, defaultTeam.Name, got.TeamName)
			assert.Equal(t, defaultTeam.MaxSeats, got.MaxSeats)
			assert.Len(t, got.Members, tt.wantMembers)
			assert.Len(t, got.Invites, tt.wantInvites)
		})
	}
}
