package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// CreateTaskRequest is the body for POST /api/v1/tasks.
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AgentName   string `json:"agent_name"`
	Mode        string `json:"mode,omitempty"` // "interactive" | "background"
	UserID      string `json:"user_id,omitempty"`
}

// TaskListFilter contains query parameters for listing tasks.
type TaskListFilter struct {
	Source    string
	AgentName string
	Status    string
	Limit     int
	Offset    int
}

// TaskResponse is a summary of a task for list responses.
type TaskResponse struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	AgentName string `json:"agent_name"`
	Status    string `json:"status"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

// TaskDetailResponse is the full task representation.
type TaskDetailResponse struct {
	TaskResponse
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode"`
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// PaginatedTaskResponse wraps a page of tasks with pagination metadata.
type PaginatedTaskResponse struct {
	Data       []TaskResponse `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PerPage    int            `json:"per_page"`
	TotalPages int            `json:"total_pages"`
}

// ProvideInputRequest is the body for POST /api/v1/tasks/{id}/input.
type ProvideInputRequest struct {
	Input string `json:"input"`
}

// TaskService provides task CRUD operations.
type TaskService interface {
	CreateTask(ctx context.Context, params CreateTaskRequest) (uint, error)
	ListTasks(ctx context.Context, filter TaskListFilter) ([]TaskResponse, error)
	CountTasks(ctx context.Context, filter TaskListFilter) (int64, error)
	GetTask(ctx context.Context, id uint) (*TaskDetailResponse, error)
	CancelTask(ctx context.Context, id uint) error
	ProvideInput(ctx context.Context, id uint, input string) error
}

// TaskHandler serves /api/v1/tasks endpoints.
type TaskHandler struct {
	service TaskService
}

// NewTaskHandler creates a TaskHandler.
func NewTaskHandler(service TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

// Routes returns a chi router with task endpoints mounted.
func (h *TaskHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Cancel)
	r.Post("/{id}/input", h.ProvideInput)
	return r
}

// Create handles POST /api/v1/tasks.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.AgentName == "" {
		writeJSONError(w, http.StatusBadRequest, "agent_name is required")
		return
	}

	taskID, err := h.service.CreateTask(r.Context(), req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"task_id": taskID,
		"status":  "pending",
	})
}

// List handles GET /api/v1/tasks.
// Supports pagination via ?page=N&per_page=M query parameters.
// Without pagination params, returns all tasks (backward compatible).
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := TaskListFilter{
		Source:    r.URL.Query().Get("source"),
		AgentName: r.URL.Query().Get("agent_name"),
		Status:    r.URL.Query().Get("status"),
	}

	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")

	// No pagination params — return plain array (backward compat)
	if pageStr == "" && perPageStr == "" {
		tasks, err := h.service.ListTasks(r.Context(), filter)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
		return
	}

	page := 1
	perPage := 20
	if pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			page = v
		}
	}
	if perPageStr != "" {
		if v, err := strconv.Atoi(perPageStr); err == nil && v > 0 {
			if v > 100 {
				v = 100
			}
			perPage = v
		}
	}

	filter.Limit = perPage
	filter.Offset = (page - 1) * perPage

	tasks, err := h.service.ListTasks(r.Context(), filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	total, err := h.service.CountTasks(r.Context(), filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, PaginatedTaskResponse{
		Data:       tasks,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// Get handles GET /api/v1/tasks/{id}.
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.service.GetTask(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("task not found: %d", id))
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Cancel handles DELETE /api/v1/tasks/{id}.
func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.CancelTask(r.Context(), id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ProvideInput handles POST /api/v1/tasks/{id}/input.
func (h *TaskHandler) ProvideInput(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDParam(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req ProvideInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if req.Input == "" {
		writeJSONError(w, http.StatusBadRequest, "input is required")
		return
	}

	if err := h.service.ProvideInput(r.Context(), id, req.Input); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseIDParam extracts and validates the "id" URL parameter.
func parseIDParam(r *http.Request) (uint, error) {
	raw := chi.URLParam(r, "id")
	if raw == "" {
		return 0, fmt.Errorf("id parameter is required")
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", raw)
	}
	return uint(id), nil
}
