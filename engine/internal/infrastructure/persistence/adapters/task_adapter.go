package adapters

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// SubtaskToRuntimeModel converts domain Subtask to RuntimeSubtaskModel
func SubtaskToRuntimeModel(subtask *domain.Subtask) *models.RuntimeSubtaskModel {
	if subtask == nil {
		return nil
	}

	id := subtask.ID
	if id == "" {
		id = uuid.New().String()
	}

	blockedBy := ""
	if len(subtask.BlockedBy) > 0 {
		data, _ := json.Marshal(subtask.BlockedBy)
		blockedBy = string(data)
	}

	filesInvolved := ""
	if len(subtask.FilesInvolved) > 0 {
		data, _ := json.Marshal(subtask.FilesInvolved)
		filesInvolved = string(data)
	}

	ctx := ""
	if len(subtask.Context) > 0 {
		data, _ := json.Marshal(subtask.Context)
		ctx = string(data)
	}

	return &models.RuntimeSubtaskModel{
		ID:              id,
		SessionID:       subtask.SessionID,
		TaskID:          subtask.TaskID,
		Title:           subtask.Title,
		Description:     subtask.Description,
		Status:          string(subtask.Status),
		AssignedAgentID: subtask.AssignedAgentID,
		BlockedBy:       blockedBy,
		FilesInvolved:   filesInvolved,
		Result:          subtask.Result,
		Context:         ctx,
		CreatedAt:       subtask.CreatedAt,
		UpdatedAt:       subtask.UpdatedAt,
		CompletedAt:     subtask.CompletedAt,
	}
}

// RuntimeModelToSubtask converts RuntimeSubtaskModel to domain Subtask
func RuntimeModelToSubtask(model *models.RuntimeSubtaskModel) (*domain.Subtask, error) {
	if model == nil {
		return nil, nil
	}

	subtask := &domain.Subtask{
		ID:              model.ID,
		SessionID:       model.SessionID,
		TaskID:          model.TaskID,
		Title:           model.Title,
		Description:     model.Description,
		Status:          domain.SubtaskStatus(model.Status),
		AssignedAgentID: model.AssignedAgentID,
		Result:          model.Result,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
		CompletedAt:     model.CompletedAt,
	}

	if model.BlockedBy != "" {
		_ = json.Unmarshal([]byte(model.BlockedBy), &subtask.BlockedBy)
	}
	if model.FilesInvolved != "" {
		_ = json.Unmarshal([]byte(model.FilesInvolved), &subtask.FilesInvolved)
	}
	if model.Context != "" {
		_ = json.Unmarshal([]byte(model.Context), &subtask.Context)
	}

	return subtask, nil
}

// SubtaskToTaskModel is a legacy alias — use SubtaskToRuntimeModel instead.
// Kept temporarily for backward compatibility during migration.
func SubtaskToTaskModel(subtask *domain.Subtask) *models.RuntimeSubtaskModel {
	return SubtaskToRuntimeModel(subtask)
}

// TaskModelToSubtask is a legacy alias — use RuntimeModelToSubtask instead.
func TaskModelToSubtask(model *models.RuntimeSubtaskModel) (*domain.Subtask, error) {
	return RuntimeModelToSubtask(model)
}
