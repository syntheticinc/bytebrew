package remove_member

import (
	"context"
	"fmt"
	"testing"

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

type mockMemberRemover struct {
	err    error
	called bool
	teamID string
	userID string
}

func (m *mockMemberRemover) RemoveMember(_ context.Context, teamID, userID string) error {
	m.called = true
	m.teamID = teamID
	m.userID = userID
	return m.err
}

type mockSubscriptionReader struct {
	sub *domain.Subscription
	err error
}

func (m *mockSubscriptionReader) GetByUserID(_ context.Context, _ string) (*domain.Subscription, error) {
	return m.sub, m.err
}

type mockSeatUpdater struct {
	err          error
	called       bool
	lastQuantity int64
}

func (m *mockSeatUpdater) UpdateSubscriptionQuantity(_ context.Context, _ string, quantity int64) error {
	m.called = true
	m.lastQuantity = quantity
	return m.err
}

type mockMemberCounter struct {
	count int
	err   error
}

func (m *mockMemberCounter) CountMembers(_ context.Context, _ string) (int, error) {
	return m.count, m.err
}

type mockTeamSeatsUpdater struct {
	err       error
	called    bool
	lastSeats int
}

func (m *mockTeamSeatsUpdater) UpdateTeamMaxSeats(_ context.Context, _ string, maxSeats int) error {
	m.called = true
	m.lastSeats = maxSeats
	return m.err
}

// --- Tests ---

func TestExecute(t *testing.T) {
	stripeSubID := "sub_stripe_1"
	defaultTeam := &domain.Team{
		ID:       "team-1",
		Name:     "My Team",
		OwnerID:  "owner-1",
		MaxSeats: 5,
	}
	defaultSub := &domain.Subscription{
		UserID:               "owner-1",
		StripeSubscriptionID: &stripeSubID,
	}

	tests := []struct {
		name         string
		input        Input
		team         *domain.Team
		teamErr      error
		removeErr    error
		sub          *domain.Subscription
		subErr       error
		newCount     int
		countErr     error
		seatErr      error
		teamSeatsErr error
		wantCode     string
		wantCall     bool
		wantSeatQty  int64
	}{
		{
			name:        "success: admin removes member and decrements seat",
			input:       Input{UserID: "member-1", RequestBy: "owner-1"},
			team:        defaultTeam,
			sub:         defaultSub,
			newCount:    3,
			wantCall:    true,
			wantSeatQty: 3,
		},
		{
			name:        "success: decrement enforces minimum 1 seat",
			input:       Input{UserID: "member-1", RequestBy: "owner-1"},
			team:        defaultTeam,
			sub:         defaultSub,
			newCount:    0,
			wantCall:    true,
			wantSeatQty: 1, // minimum 1 seat (owner)
		},
		{
			name:     "success: no subscription — still removes member, no stripe call",
			input:    Input{UserID: "member-1", RequestBy: "owner-1"},
			team:     defaultTeam,
			sub:      nil,
			newCount: 2,
			wantCall: true,
		},
		{
			name:     "error: empty user ID",
			input:    Input{UserID: "", RequestBy: "owner-1"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: empty requester ID",
			input:    Input{UserID: "member-1", RequestBy: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: requester not in any team",
			input:    Input{UserID: "member-1", RequestBy: "owner-1"},
			team:     nil,
			wantCode: errors.CodeNotFound,
		},
		{
			name:     "error: get team fails",
			input:    Input{UserID: "member-1", RequestBy: "owner-1"},
			teamErr:  fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:     "error: non-admin tries to remove",
			input:    Input{UserID: "member-2", RequestBy: "member-1"},
			team:     defaultTeam,
			wantCode: errors.CodeForbidden,
		},
		{
			name:     "error: cannot remove team owner",
			input:    Input{UserID: "owner-1", RequestBy: "owner-1"},
			team:     defaultTeam,
			wantCode: errors.CodeForbidden,
		},
		{
			name:      "error: member remover fails",
			input:     Input{UserID: "member-1", RequestBy: "owner-1"},
			team:      defaultTeam,
			removeErr: fmt.Errorf("db error"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:     "error: count members after remove fails",
			input:    Input{UserID: "member-1", RequestBy: "owner-1"},
			team:     defaultTeam,
			countErr: fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:     "error: get subscription fails",
			input:    Input{UserID: "member-1", RequestBy: "owner-1"},
			team:     defaultTeam,
			newCount: 2,
			subErr:   fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:     "error: stripe seat update fails",
			input:    Input{UserID: "member-1", RequestBy: "owner-1"},
			team:     defaultTeam,
			sub:      defaultSub,
			newCount: 2,
			seatErr:  fmt.Errorf("stripe error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:         "error: team seats update fails",
			input:        Input{UserID: "member-1", RequestBy: "owner-1"},
			team:         defaultTeam,
			sub:          defaultSub,
			newCount:     2,
			teamSeatsErr: fmt.Errorf("db error"),
			wantCode:     errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamReader := &mockTeamByUserReader{team: tt.team, err: tt.teamErr}
			remover := &mockMemberRemover{err: tt.removeErr}
			subReader := &mockSubscriptionReader{sub: tt.sub, err: tt.subErr}
			seatUpdater := &mockSeatUpdater{err: tt.seatErr}
			counter := &mockMemberCounter{count: tt.newCount, err: tt.countErr}
			teamSeatsUpdater := &mockTeamSeatsUpdater{err: tt.teamSeatsErr}

			uc := New(teamReader, remover, subReader, seatUpdater, counter, teamSeatsUpdater)
			err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantCode), "expected error code %s, got: %v", tt.wantCode, err)
				return
			}

			require.NoError(t, err)

			if tt.wantCall {
				require.True(t, remover.called, "expected RemoveMember to be called")
				assert.Equal(t, defaultTeam.ID, remover.teamID)
				assert.Equal(t, tt.input.UserID, remover.userID)
			}

			if tt.wantSeatQty > 0 {
				assert.True(t, seatUpdater.called, "expected Stripe seat update")
				assert.Equal(t, tt.wantSeatQty, seatUpdater.lastQuantity)
				assert.True(t, teamSeatsUpdater.called, "expected team seats update")
				assert.Equal(t, int(tt.wantSeatQty), teamSeatsUpdater.lastSeats)
			}
		})
	}
}
