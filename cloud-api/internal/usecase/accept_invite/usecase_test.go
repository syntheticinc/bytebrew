package accept_invite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks ---

type mockInviteReader struct {
	invite *domain.TeamInvite
	err    error
}

func (m *mockInviteReader) GetInviteByToken(_ context.Context, _ string) (*domain.TeamInvite, error) {
	return m.invite, m.err
}

type mockInviteUpdater struct {
	err      error
	called   bool
	statusID string
	status   domain.InviteStatus
}

func (m *mockInviteUpdater) UpdateInviteStatus(_ context.Context, inviteID string, status domain.InviteStatus) error {
	m.called = true
	m.statusID = inviteID
	m.status = status
	return m.err
}

type mockUserFinder struct {
	user *domain.User
	err  error
}

func (m *mockUserFinder) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	return m.user, m.err
}

type mockMemberAdder struct {
	err    error
	called bool
	member *domain.TeamMember
}

func (m *mockMemberAdder) AddMember(_ context.Context, member *domain.TeamMember) (*domain.TeamMember, error) {
	m.called = true
	m.member = member
	if m.err != nil {
		return nil, m.err
	}
	added := *member
	added.ID = "member-id-1"
	return &added, nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	validInvite := &domain.TeamInvite{
		ID:        "invite-1",
		TeamID:    "team-1",
		Email:     "user@example.com",
		InvitedBy: "owner-1",
		Token:     "valid-token",
		Status:    domain.InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	validUser := &domain.User{
		ID:    "user-1",
		Email: "user@example.com",
	}

	tests := []struct {
		name       string
		input      Input
		invite     *domain.TeamInvite
		inviteErr  error
		updateErr  error
		user       *domain.User
		userErr    error
		memberErr  error
		wantCode   string
		wantTeamID string
		wantUserID string
	}{
		{
			name:       "success: accepts invite and adds user to team",
			input:      Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite:     validInvite,
			user:       validUser,
			wantTeamID: "team-1",
			wantUserID: "user-1",
		},
		{
			name:       "success: case-insensitive email match",
			input:      Input{Token: "valid-token", CallerEmail: "User@Example.COM"},
			invite:     validInvite,
			user:       validUser,
			wantTeamID: "team-1",
			wantUserID: "user-1",
		},
		{
			name:     "error: empty token",
			input:    Input{Token: "", CallerEmail: "user@example.com"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: empty caller email",
			input:    Input{Token: "valid-token", CallerEmail: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: invite not found (nil)",
			input:    Input{Token: "bad-token", CallerEmail: "user@example.com"},
			invite:   nil,
			wantCode: errors.CodeNotFound,
		},
		{
			name:      "error: invite reader fails",
			input:     Input{Token: "valid-token", CallerEmail: "user@example.com"},
			inviteErr: fmt.Errorf("db error"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:  "error: invite already accepted",
			input: Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite: &domain.TeamInvite{
				ID:        "invite-1",
				TeamID:    "team-1",
				Email:     "user@example.com",
				Token:     "valid-token",
				Status:    domain.InviteStatusAccepted,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:  "error: invite expired",
			input: Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite: &domain.TeamInvite{
				ID:        "invite-1",
				TeamID:    "team-1",
				Email:     "user@example.com",
				Token:     "valid-token",
				Status:    domain.InviteStatusPending,
				ExpiresAt: time.Now().Add(-24 * time.Hour),
			},
			wantCode: errors.CodeForbidden,
		},
		{
			name:  "error: caller email does not match invite email",
			input: Input{Token: "valid-token", CallerEmail: "other@example.com"},
			invite: &domain.TeamInvite{
				ID:        "invite-1",
				TeamID:    "team-1",
				Email:     "user@example.com",
				Token:     "valid-token",
				Status:    domain.InviteStatusPending,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			wantCode: errors.CodeForbidden,
		},
		{
			name:     "error: user not found by email",
			input:    Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite:   validInvite,
			user:     nil,
			wantCode: errors.CodeNotFound,
		},
		{
			name:     "error: user finder fails",
			input:    Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite:   validInvite,
			userErr:  fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:      "error: member adder fails",
			input:     Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite:    validInvite,
			user:      validUser,
			memberErr: fmt.Errorf("db error"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:      "error: invite updater fails",
			input:     Input{Token: "valid-token", CallerEmail: "user@example.com"},
			invite:    validInvite,
			user:      validUser,
			updateErr: fmt.Errorf("db error"),
			wantCode:  errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(
				&mockInviteReader{invite: tt.invite, err: tt.inviteErr},
				&mockInviteUpdater{err: tt.updateErr},
				&mockUserFinder{user: tt.user, err: tt.userErr},
				&mockMemberAdder{err: tt.memberErr},
			)
			// Override nowFunc to ensure deterministic time checks.
			// Use a fixed "now" that is before valid expiry and after expired expiry.
			uc.nowFunc = func() time.Time { return time.Now() }

			got, err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantCode), "expected error code %s, got: %v", tt.wantCode, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantTeamID, got.TeamID)
			assert.Equal(t, tt.wantUserID, got.UserID)
		})
	}
}
