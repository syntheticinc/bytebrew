package create_team

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockTeamCreator struct {
	createFn func(ctx context.Context, team *domain.Team) (*domain.Team, error)
}

func (m *mockTeamCreator) CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error) {
	if m.createFn != nil {
		return m.createFn(ctx, team)
	}
	created := *team
	created.ID = "team-id-1"
	return &created, nil
}

type mockMemberAdder struct {
	addFn  func(ctx context.Context, member *domain.TeamMember) (*domain.TeamMember, error)
	called bool
	member *domain.TeamMember
}

func (m *mockMemberAdder) AddMember(ctx context.Context, member *domain.TeamMember) (*domain.TeamMember, error) {
	m.called = true
	m.member = member
	if m.addFn != nil {
		return m.addFn(ctx, member)
	}
	added := *member
	added.ID = "member-id-1"
	return &added, nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		input       Input
		teamErr     error
		memberErr   error
		wantCode    string
		wantTeamID  string
		wantMember  bool
		wantMemberRole domain.TeamRole
	}{
		{
			name:           "success: creates team and adds owner as admin",
			input:          Input{OwnerID: "owner-1", Name: "My Team"},
			wantTeamID:     "team-id-1",
			wantMember:     true,
			wantMemberRole: domain.TeamRoleAdmin,
		},
		{
			name:     "error: empty name",
			input:    Input{OwnerID: "owner-1", Name: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: empty owner ID",
			input:    Input{OwnerID: "", Name: "My Team"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: team creator fails",
			input:    Input{OwnerID: "owner-1", Name: "My Team"},
			teamErr:  fmt.Errorf("db connection lost"),
			wantCode: errors.CodeInternal,
		},
		{
			name:      "error: member adder fails",
			input:     Input{OwnerID: "owner-1", Name: "My Team"},
			memberErr: fmt.Errorf("db insert failed"),
			wantCode:  errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &mockTeamCreator{}
			if tt.teamErr != nil {
				creator.createFn = func(_ context.Context, _ *domain.Team) (*domain.Team, error) {
					return nil, tt.teamErr
				}
			}

			adder := &mockMemberAdder{}
			if tt.memberErr != nil {
				adder.addFn = func(_ context.Context, _ *domain.TeamMember) (*domain.TeamMember, error) {
					return nil, tt.memberErr
				}
			}

			uc := New(creator, adder)
			got, err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantCode), "expected error code %s, got: %v", tt.wantCode, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantTeamID, got.Team.ID)
			assert.Equal(t, tt.input.Name, got.Team.Name)
			assert.Equal(t, tt.input.OwnerID, got.Team.OwnerID)

			if tt.wantMember {
				require.True(t, adder.called, "expected AddMember to be called")
				assert.Equal(t, tt.wantTeamID, adder.member.TeamID)
				assert.Equal(t, tt.input.OwnerID, adder.member.UserID)
				assert.Equal(t, tt.wantMemberRole, adder.member.Role)
			}
		})
	}
}
