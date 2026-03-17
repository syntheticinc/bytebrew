package remove_member

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// TeamByUserReader reads the team a user belongs to.
type TeamByUserReader interface {
	GetTeamByUserID(ctx context.Context, userID string) (*domain.Team, error)
}

// MemberRemover removes a member from a team.
type MemberRemover interface {
	RemoveMember(ctx context.Context, teamID, userID string) error
}

// SubscriptionReader reads a user's subscription.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// SeatUpdater updates the seat count on a Stripe subscription.
type SeatUpdater interface {
	UpdateSubscriptionQuantity(ctx context.Context, subscriptionID string, quantity int64) error
}

// MemberCounter counts members in a team.
type MemberCounter interface {
	CountMembers(ctx context.Context, teamID string) (int, error)
}

// TeamSeatsUpdater updates the max seats for a team.
type TeamSeatsUpdater interface {
	UpdateTeamMaxSeats(ctx context.Context, teamID string, maxSeats int) error
}

// Input is the remove member request.
type Input struct {
	UserID    string
	RequestBy string
}

// Usecase handles removing a member from a team.
type Usecase struct {
	teamByUserReader TeamByUserReader
	memberRemover    MemberRemover
	subReader        SubscriptionReader
	seatUpdater      SeatUpdater
	memberCounter    MemberCounter
	teamSeatsUpdater TeamSeatsUpdater
}

// New creates a new RemoveMember usecase.
func New(
	teamByUserReader TeamByUserReader,
	memberRemover MemberRemover,
	subReader SubscriptionReader,
	seatUpdater SeatUpdater,
	memberCounter MemberCounter,
	teamSeatsUpdater TeamSeatsUpdater,
) *Usecase {
	return &Usecase{
		teamByUserReader: teamByUserReader,
		memberRemover:    memberRemover,
		subReader:        subReader,
		seatUpdater:      seatUpdater,
		memberCounter:    memberCounter,
		teamSeatsUpdater: teamSeatsUpdater,
	}
}

// Execute removes a member from a team.
// Only the team owner can remove members, and the owner cannot be removed.
// The team is resolved from the requester's membership.
func (u *Usecase) Execute(ctx context.Context, input Input) error {
	if input.UserID == "" {
		return errors.InvalidInput("user ID is required")
	}
	if input.RequestBy == "" {
		return errors.InvalidInput("requester ID is required")
	}

	team, err := u.teamByUserReader.GetTeamByUserID(ctx, input.RequestBy)
	if err != nil {
		return errors.Internal("get team", err)
	}
	if team == nil {
		return errors.NotFound("you are not a member of any team")
	}

	if team.OwnerID != input.RequestBy {
		return errors.Forbidden("only the team owner can remove members")
	}

	if input.UserID == team.OwnerID {
		return errors.Forbidden("cannot remove the team owner")
	}

	if err := u.memberRemover.RemoveMember(ctx, team.ID, input.UserID); err != nil {
		return errors.Internal("remove member", err)
	}

	// Decrement seat in Stripe (prorated credit)
	newCount, err := u.memberCounter.CountMembers(ctx, team.ID)
	if err != nil {
		return errors.Internal("count remaining members", err)
	}

	sub, err := u.subReader.GetByUserID(ctx, team.OwnerID)
	if err != nil {
		return errors.Internal("get owner subscription", err)
	}

	if sub != nil && sub.StripeSubscriptionID != nil {
		quantity := int64(newCount)
		if quantity < 1 {
			quantity = 1 // minimum 1 seat (owner)
		}
		if err := u.seatUpdater.UpdateSubscriptionQuantity(ctx, *sub.StripeSubscriptionID, quantity); err != nil {
			return errors.Internal("update stripe seat count", err)
		}
		if err := u.teamSeatsUpdater.UpdateTeamMaxSeats(ctx, team.ID, int(quantity)); err != nil {
			return errors.Internal("update team max seats", err)
		}
	}

	return nil
}
