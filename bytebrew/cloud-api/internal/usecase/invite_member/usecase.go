package invite_member

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// TeamByUserReader reads the team a user belongs to.
type TeamByUserReader interface {
	GetTeamByUserID(ctx context.Context, userID string) (*domain.Team, error)
}

// MemberCounter counts members in a team.
type MemberCounter interface {
	CountMembers(ctx context.Context, teamID string) (int, error)
}

// InviteCreator persists a new invite.
type InviteCreator interface {
	CreateInvite(ctx context.Context, invite *domain.TeamInvite) (*domain.TeamInvite, error)
}

// EmailSender sends invite emails.
type EmailSender interface {
	SendTeamInvite(ctx context.Context, email, teamName, inviteToken string) error
}

// SubscriptionReader reads a user's subscription.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// SeatUpdater updates the seat count on a Stripe subscription.
type SeatUpdater interface {
	UpdateSubscriptionQuantity(ctx context.Context, subscriptionID string, quantity int64) error
}

// TeamSeatsUpdater updates the max seats for a team.
type TeamSeatsUpdater interface {
	UpdateTeamMaxSeats(ctx context.Context, teamID string, maxSeats int) error
}

// Input is the invite member request.
type Input struct {
	Email     string
	InvitedBy string
}

// Output is the invite member response.
type Output struct {
	Invite *domain.TeamInvite
}

// Usecase handles team member invitation.
type Usecase struct {
	teamByUserReader TeamByUserReader
	memberCounter    MemberCounter
	inviteCreator    InviteCreator
	emailSender      EmailSender
	subReader        SubscriptionReader
	seatUpdater      SeatUpdater
	teamSeatsUpdater TeamSeatsUpdater
}

// New creates a new InviteMember usecase.
func New(
	teamByUserReader TeamByUserReader,
	memberCounter MemberCounter,
	inviteCreator InviteCreator,
	emailSender EmailSender,
	subReader SubscriptionReader,
	seatUpdater SeatUpdater,
	teamSeatsUpdater TeamSeatsUpdater,
) *Usecase {
	return &Usecase{
		teamByUserReader: teamByUserReader,
		memberCounter:    memberCounter,
		inviteCreator:    inviteCreator,
		emailSender:      emailSender,
		subReader:        subReader,
		seatUpdater:      seatUpdater,
		teamSeatsUpdater: teamSeatsUpdater,
	}
}

// Execute creates an invitation and sends an email to the invitee.
// The team is resolved from the inviter's membership.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.Email == "" {
		return nil, errors.InvalidInput("email is required")
	}
	if input.InvitedBy == "" {
		return nil, errors.InvalidInput("inviter ID is required")
	}

	team, err := u.teamByUserReader.GetTeamByUserID(ctx, input.InvitedBy)
	if err != nil {
		return nil, errors.Internal("get team", err)
	}
	if team == nil {
		return nil, errors.NotFound("you are not a member of any team")
	}

	if team.OwnerID != input.InvitedBy {
		return nil, errors.Forbidden("only the team owner can invite members")
	}

	count, err := u.memberCounter.CountMembers(ctx, team.ID)
	if err != nil {
		return nil, errors.Internal("count members", err)
	}

	// Increment seat in Stripe (prorated charge)
	sub, err := u.subReader.GetByUserID(ctx, team.OwnerID)
	if err != nil {
		return nil, errors.Internal("get owner subscription", err)
	}
	if sub == nil || sub.StripeSubscriptionID == nil {
		return nil, errors.Forbidden("team owner has no active subscription")
	}

	newQuantity := int64(count + 1)
	if err := u.seatUpdater.UpdateSubscriptionQuantity(ctx, *sub.StripeSubscriptionID, newQuantity); err != nil {
		return nil, errors.Internal("update stripe seat count", err)
	}

	if err := u.teamSeatsUpdater.UpdateTeamMaxSeats(ctx, team.ID, int(newQuantity)); err != nil {
		return nil, errors.Internal("update team max seats", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, errors.Internal("generate invite token", err)
	}

	invite, err := u.inviteCreator.CreateInvite(ctx, &domain.TeamInvite{
		TeamID:    team.ID,
		Email:     input.Email,
		InvitedBy: input.InvitedBy,
		Token:     token,
		Status:    domain.InviteStatusPending,
	})
	if err != nil {
		return nil, errors.Internal("create invite", err)
	}

	if err := u.emailSender.SendTeamInvite(ctx, input.Email, team.Name, token); err != nil {
		return nil, errors.Internal("send invite email", err)
	}

	return &Output{Invite: invite}, nil
}

// generateToken creates a cryptographically random hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
