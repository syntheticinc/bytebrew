package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTaskService struct {
	createdID   uint
	tasks       []TaskResponse
	taskDetail  *TaskDetailResponse
	cancelledID uint
	inputID     uint
	inputText   string
	err         error
}

func (m *mockTaskService) CreateTask(_ context.Context, params CreateTaskRequest) (uint, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.createdID, nil
}

func (m *mockTaskService) ListTasks(_ context.Context, _ TaskListFilter) ([]TaskResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}

func (m *mockTaskService) GetTask(_ context.Context, id uint) (*TaskDetailResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.taskDetail != nil && m.taskDetail.ID == id {
		return m.taskDetail, nil
	}
	return nil, nil
}

func (m *mockTaskService) CancelTask(_ context.Context, id uint) error {
	if m.err != nil {
		return m.err
	}
	m.cancelledID = id
	return nil
}

func (m *mockTaskService) ProvideInput(_ context.Context, id uint, input string) error {
	if m.err != nil {
		return m.err
	}
	m.inputID = id
	m.inputText = input
	return nil
}

func newTaskRouter(handler *TaskHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Mount("/tasks", handler.Routes())
	return r
}

func TestTaskHandler_Create(t *testing.T) {
	mock := &mockTaskService{createdID: 42}
	handler := NewTaskHandler(mock)
	router := newTaskRouter(handler)

	body, _ := json.Marshal(CreateTaskRequest{
		Title:     "Deploy v2",
		AgentName: "devops",
	})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, float64(42), resp["task_id"])
	assert.Equal(t, "pending", resp["status"])
}

func TestTaskHandler_Create_MissingTitle(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{})
	router := newTaskRouter(handler)

	body, _ := json.Marshal(CreateTaskRequest{AgentName: "devops"})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskHandler_Create_MissingAgent(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{})
	router := newTaskRouter(handler)

	body, _ := json.Marshal(CreateTaskRequest{Title: "test"})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskHandler_List(t *testing.T) {
	tasks := []TaskResponse{
		{ID: 1, Title: "Task 1", AgentName: "sales", Status: "completed"},
		{ID: 2, Title: "Task 2", AgentName: "devops", Status: "pending"},
	}
	handler := NewTaskHandler(&mockTaskService{tasks: tasks})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result []TaskResponse
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestTaskHandler_List_WithFilters(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{tasks: []TaskResponse{}})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/tasks?source=api&agent_name=sales&status=pending", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskHandler_Get(t *testing.T) {
	detail := &TaskDetailResponse{
		TaskResponse: TaskResponse{ID: 5, Title: "Build feature", AgentName: "coder", Status: "in_progress"},
		Mode:         "interactive",
	}
	handler := NewTaskHandler(&mockTaskService{taskDetail: detail})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/tasks/5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result TaskDetailResponse
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, uint(5), result.ID)
	assert.Equal(t, "Build feature", result.Title)
}

func TestTaskHandler_Get_NotFound(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/tasks/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskHandler_Get_InvalidID(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/tasks/abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskHandler_Cancel(t *testing.T) {
	mock := &mockTaskService{}
	handler := NewTaskHandler(mock)
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(7), mock.cancelledID)
}

func TestTaskHandler_Cancel_Error(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{err: fmt.Errorf("not found")})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodDelete, "/tasks/7", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestTaskHandler_ProvideInput(t *testing.T) {
	mock := &mockTaskService{}
	handler := NewTaskHandler(mock)
	router := newTaskRouter(handler)

	body, _ := json.Marshal(ProvideInputRequest{Input: "yes, proceed"})
	req := httptest.NewRequest(http.MethodPost, "/tasks/10/input", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(10), mock.inputID)
	assert.Equal(t, "yes, proceed", mock.inputText)
}

func TestTaskHandler_ProvideInput_EmptyInput(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{})
	router := newTaskRouter(handler)

	body, _ := json.Marshal(ProvideInputRequest{Input: ""})
	req := httptest.NewRequest(http.MethodPost, "/tasks/10/input", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskHandler_ProvideInput_InvalidBody(t *testing.T) {
	handler := NewTaskHandler(&mockTaskService{})
	router := newTaskRouter(handler)

	req := httptest.NewRequest(http.MethodPost, "/tasks/10/input", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
