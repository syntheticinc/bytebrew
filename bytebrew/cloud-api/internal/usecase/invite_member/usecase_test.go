package invite_member

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
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

type mockMemberCounter struct {
	count int
	err   error
}

func (m *mockMemberCounter) CountMembers(_ context.Context, _ string) (int, error) {
	return m.count, m.err
}

type mockInviteCreator struct {
	createFn func(ctx context.Context, invite *domain.TeamInvite) (*domain.TeamInvite, error)
}

func (m *mockInviteCreator) CreateInvite(ctx context.Context, invite *domain.TeamInvite) (*domain.TeamInvite, error) {
	if m.createFn != nil {
		return m.createFn(ctx, invite)
	}
	created := *invite
	created.ID = "invite-id-1"
	return &created, nil
}

type mockEmailSender struct {
	err    error
	called bool
}

func (m *mockEmailSender) SendTeamInvite(_ context.Context, _, _, _ string) error {
	m.called = true
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
	err            error
	called         bool
	lastQuantity   int64
}

func (m *mockSeatUpdater) UpdateSubscriptionQuantity(_ context.Context, _ string, quantity int64) error {
	m.called = true
	m.lastQuantity = quantity
	return m.err
}

type mockTeamSeatsUpdater struct {
	err      error
	called   bool
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
		name       string
		input      Input
		team       *domain.Team
		teamErr    error
		count      int
		countErr   error
		sub        *domain.Subscription
		subErr     error
		seatErr    error
		teamSeatsErr error
		inviteErr  error
		emailErr   error
		wantCode   string
		wantInvite bool
		wantEmail  bool
		wantSeatQty int64
	}{
		{
			name:        "success: creates invite, increments seat and sends email",
			input:       Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:        defaultTeam,
			count:       2,
			sub:         defaultSub,
			wantInvite:  true,
			wantEmail:   true,
			wantSeatQty: 3, // count(2) + 1
		},
		{
			name:     "error: empty email",
			input:    Input{Email: "", InvitedBy: "owner-1"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: empty inviter ID",
			input:    Input{Email: "new@example.com", InvitedBy: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "error: inviter not in any team",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     nil,
			wantCode: errors.CodeNotFound,
		},
		{
			name:     "error: inviter is not the owner",
			input:    Input{Email: "new@example.com", InvitedBy: "member-1"},
			team:     defaultTeam,
			wantCode: errors.CodeForbidden,
		},
		{
			name:     "error: owner has no subscription",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     defaultTeam,
			count:    2,
			sub:      nil,
			wantCode: errors.CodeForbidden,
		},
		{
			name:     "error: subscription has no stripe ID",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     defaultTeam,
			count:    2,
			sub:      &domain.Subscription{UserID: "owner-1", StripeSubscriptionID: nil},
			wantCode: errors.CodeForbidden,
		},
		{
			name:     "error: stripe seat update fails",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     defaultTeam,
			count:    2,
			sub:      defaultSub,
			seatErr:  fmt.Errorf("stripe error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:         "error: team seats update fails",
			input:        Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:         defaultTeam,
			count:        2,
			sub:          defaultSub,
			teamSeatsErr: fmt.Errorf("db error"),
			wantCode:     errors.CodeInternal,
		},
		{
			name:      "error: invite repository fails",
			input:     Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:      defaultTeam,
			count:     2,
			sub:       defaultSub,
			inviteErr: fmt.Errorf("db error"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:     "error: email sending fails",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     defaultTeam,
			count:    2,
			sub:      defaultSub,
			emailErr: fmt.Errorf("smtp error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:     "error: get team fails",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			teamErr:  fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:     "error: count members fails",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     defaultTeam,
			countErr: fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:     "error: get subscription fails",
			input:    Input{Email: "new@example.com", InvitedBy: "owner-1"},
			team:     defaultTeam,
			count:    2,
			subErr:   fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teamReader := &mockTeamByUserReader{team: tt.team, err: tt.teamErr}
			counter := &mockMemberCounter{count: tt.count, err: tt.countErr}

			inviteCreator := &mockInviteCreator{}
			if tt.inviteErr != nil {
				inviteCreator.createFn = func(_ context.Context, _ *domain.TeamInvite) (*domain.TeamInvite, error) {
					return nil, tt.inviteErr
				}
			}

			emailSender := &mockEmailSender{err: tt.emailErr}
			subReader := &mockSubscriptionReader{sub: tt.sub, err: tt.subErr}
			seatUpdater := &mockSeatUpdater{err: tt.seatErr}
			teamSeatsUpdater := &mockTeamSeatsUpdater{err: tt.teamSeatsErr}

			uc := New(teamReader, counter, inviteCreator, emailSender, subReader, seatUpdater, teamSeatsUpdater)
			got, err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantCode), "expected error code %s, got: %v", tt.wantCode, err)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.Invite)
			assert.Equal(t, "invite-id-1", got.Invite.ID)
			assert.Equal(t, tt.input.Email, got.Invite.Email)
			assert.Equal(t, tt.input.InvitedBy, got.Invite.InvitedBy)
			assert.Equal(t, domain.InviteStatusPending, got.Invite.Status)
			assert.NotEmpty(t, got.Invite.Token, "token must be generated")

			if tt.wantSeatQty > 0 {
				assert.True(t, seatUpdater.called, "expected Stripe seat update")
				assert.Equal(t, tt.wantSeatQty, seatUpdater.lastQuantity)
				assert.True(t, teamSeatsUpdater.called, "expected team seats update")
				assert.Equal(t, int(tt.wantSeatQty), teamSeatsUpdater.lastSeats)
			}

			if tt.wantEmail {
				assert.True(t, emailSender.called, "expected email to be sent")
			}
		})
	}
}
