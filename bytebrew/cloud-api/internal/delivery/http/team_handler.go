package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/accept_invite"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_team"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/invite_member"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/list_members"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/remove_member"
)

type createTeamUsecase interface {
	Execute(ctx context.Context, input create_team.Input) (*create_team.Output, error)
}

type inviteMemberUsecase interface {
	Execute(ctx context.Context, input invite_member.Input) (*invite_member.Output, error)
}

type acceptInviteUsecase interface {
	Execute(ctx context.Context, input accept_invite.Input) (*accept_invite.Output, error)
}

type removeMemberUsecase interface {
	Execute(ctx context.Context, input remove_member.Input) error
}

type listMembersUsecase interface {
	Execute(ctx context.Context, input list_members.Input) (*list_members.Output, error)
}

// TeamHandler handles team management endpoints.
type TeamHandler struct {
	createTeamUC   createTeamUsecase
	inviteMemberUC inviteMemberUsecase
	acceptInviteUC acceptInviteUsecase
	removeMemberUC removeMemberUsecase
	listMembersUC  listMembersUsecase
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(
	createTeamUC createTeamUsecase,
	inviteMemberUC inviteMemberUsecase,
	acceptInviteUC acceptInviteUsecase,
	removeMemberUC removeMemberUsecase,
	listMembersUC listMembersUsecase,
) *TeamHandler {
	return &TeamHandler{
		createTeamUC:   createTeamUC,
		inviteMemberUC: inviteMemberUC,
		acceptInviteUC: acceptInviteUC,
		removeMemberUC: removeMemberUC,
		listMembersUC:  listMembersUC,
	}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type createTeamResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MaxSeats int    `json:"max_seats"`
}

// CreateTeam handles POST /api/v1/teams.
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.createTeamUC.Execute(r.Context(), create_team.Input{
		OwnerID: middleware.GetUserID(r.Context()),
		Name:    req.Name,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusCreated, createTeamResponse{
		ID:       out.Team.ID,
		Name:     out.Team.Name,
		MaxSeats: out.Team.MaxSeats,
	})
}

type inviteMemberRequest struct {
	Email string `json:"email"`
}

type inviteMemberResponse struct {
	InviteID string `json:"invite_id"`
	Token    string `json:"token"`
}

// InviteMember handles POST /api/v1/teams/invite.
func (h *TeamHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	var req inviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.inviteMemberUC.Execute(r.Context(), invite_member.Input{
		Email:     req.Email,
		InvitedBy: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusCreated, inviteMemberResponse{
		InviteID: out.Invite.ID,
		Token:    out.Invite.Token,
	})
}

type acceptInviteRequest struct {
	Token string `json:"token"`
}

type acceptInviteResponse struct {
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
}

// AcceptInvite handles POST /api/v1/teams/accept.
func (h *TeamHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req acceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.acceptInviteUC.Execute(r.Context(), accept_invite.Input{
		Token:       req.Token,
		CallerEmail: middleware.GetEmail(r.Context()),
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, acceptInviteResponse{
		TeamID: out.TeamID,
		UserID: out.UserID,
	})
}

// RemoveMember handles DELETE /api/v1/teams/members/{userID}.
func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "userID")

	err := h.removeMemberUC.Execute(r.Context(), remove_member.Input{
		UserID:    targetUserID,
		RequestBy: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type memberResponse struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type inviteResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type listMembersResponse struct {
	TeamID   string           `json:"team_id"`
	TeamName string           `json:"team_name"`
	MaxSeats int              `json:"max_seats"`
	Members  []memberResponse `json:"members"`
	Invites  []inviteResponse `json:"invites"`
}

// ListMembers handles GET /api/v1/teams/members.
func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	out, err := h.listMembersUC.Execute(r.Context(), list_members.Input{
		UserID: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		Error(w, err)
		return
	}

	members := make([]memberResponse, 0, len(out.Members))
	for _, m := range out.Members {
		members = append(members, memberResponse{
			ID:       m.ID,
			UserID:   m.UserID,
			Email:    m.Email,
			Role:     string(m.Role),
			JoinedAt: m.JoinedAt,
		})
	}

	invites := make([]inviteResponse, 0, len(out.Invites))
	for _, inv := range out.Invites {
		invites = append(invites, inviteResponse{
			ID:        inv.ID,
			Email:     inv.Email,
			Status:    string(inv.Status),
			CreatedAt: inv.CreatedAt,
			ExpiresAt: inv.ExpiresAt,
		})
	}

	JSON(w, http.StatusOK, listMembersResponse{
		TeamID:   out.TeamID,
		TeamName: out.TeamName,
		MaxSeats: out.MaxSeats,
		Members:  members,
		Invites:  invites,
	})
}
