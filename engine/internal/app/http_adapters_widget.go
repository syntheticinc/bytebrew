package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"gorm.io/gorm"
)

// widgetServiceHTTPAdapter bridges GORMWidgetRepository to the http.WidgetService interface.
type widgetServiceHTTPAdapter struct {
	repo *config_repo.GORMWidgetRepository
}

func (a *widgetServiceHTTPAdapter) ListWidgets(ctx context.Context) ([]deliveryhttp.WidgetInfo, error) {
	tenantID := domain.TenantIDFromContext(ctx)
	records, err := a.repo.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list widgets: %w", err)
	}

	result := make([]deliveryhttp.WidgetInfo, 0, len(records))
	for _, r := range records {
		result = append(result, toWidgetInfo(r))
	}
	return result, nil
}

func (a *widgetServiceHTTPAdapter) GetWidget(ctx context.Context, id string) (*deliveryhttp.WidgetInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("widget not found: %s", id))
		}
		return nil, fmt.Errorf("get widget: %w", err)
	}

	info := toWidgetInfo(*record)
	return &info, nil
}

func (a *widgetServiceHTTPAdapter) CreateWidget(ctx context.Context, req deliveryhttp.CreateWidgetRequest) (*deliveryhttp.WidgetInfo, error) {
	tenantID := domain.TenantIDFromContext(ctx)

	enabled := req.Status != "disabled"

	record := &config_repo.WidgetRecord{
		TenantID:        tenantID,
		Name:            req.Name,
		SchemaID:        req.Schema,
		PrimaryColor:    defaultStr(req.PrimaryColor, "#6366f1"),
		Position:        defaultStr(req.Position, "bottom-right"),
		Size:            defaultStr(req.Size, "standard"),
		WelcomeMessage:  defaultStr(req.WelcomeMessage, "Hi! How can I help?"),
		Placeholder:     defaultStr(req.PlaceholderText, "Type a message..."),
		AvatarURL:       req.AvatarURL,
		DomainWhitelist: splitDomains(req.DomainWhitelist),
		CustomHeaders:   req.CustomHeaders,
		Enabled:         enabled,
	}
	if len(record.DomainWhitelist) == 0 {
		record.DomainWhitelist = []string{"*"}
	}

	if err := a.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create widget: %w", err)
	}

	info := toWidgetInfo(*record)
	return &info, nil
}

func (a *widgetServiceHTTPAdapter) UpdateWidget(ctx context.Context, id string, req deliveryhttp.CreateWidgetRequest) error {
	existing, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("widget not found: %s", id))
		}
		return fmt.Errorf("get widget: %w", err)
	}

	record := &config_repo.WidgetRecord{
		Name:            defaultStr(req.Name, existing.Name),
		SchemaID:        defaultStr(req.Schema, existing.SchemaID),
		PrimaryColor:    defaultStr(req.PrimaryColor, existing.PrimaryColor),
		Position:        defaultStr(req.Position, existing.Position),
		Size:            defaultStr(req.Size, existing.Size),
		WelcomeMessage:  defaultStr(req.WelcomeMessage, existing.WelcomeMessage),
		Placeholder:     defaultStr(req.PlaceholderText, existing.Placeholder),
		AvatarURL:       defaultStr(req.AvatarURL, existing.AvatarURL),
		DomainWhitelist: existing.DomainWhitelist,
		CustomHeaders:   existing.CustomHeaders,
		Enabled:         existing.Enabled,
	}
	if req.DomainWhitelist != "" {
		record.DomainWhitelist = splitDomains(req.DomainWhitelist)
	}
	if req.CustomHeaders != nil {
		record.CustomHeaders = req.CustomHeaders
	}
	if req.Status != "" {
		record.Enabled = req.Status != "disabled"
	}

	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("widget not found: %s", id))
		}
		return fmt.Errorf("update widget: %w", err)
	}
	return nil
}

func (a *widgetServiceHTTPAdapter) DeleteWidget(ctx context.Context, id string) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("widget not found: %s", id))
		}
		return fmt.Errorf("delete widget: %w", err)
	}
	return nil
}

func toWidgetInfo(r config_repo.WidgetRecord) deliveryhttp.WidgetInfo {
	status := "active"
	if !r.Enabled {
		status = "disabled"
	}

	var createdAt string
	if !r.CreatedAt.IsZero() {
		createdAt = r.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return deliveryhttp.WidgetInfo{
		ID:              r.ID,
		Name:            r.Name,
		Schema:          r.SchemaID,
		Status:          status,
		PrimaryColor:    r.PrimaryColor,
		Position:        r.Position,
		Size:            r.Size,
		WelcomeMessage:  r.WelcomeMessage,
		PlaceholderText: r.Placeholder,
		AvatarURL:       r.AvatarURL,
		DomainWhitelist: joinDomains(r.DomainWhitelist),
		CustomHeaders:   r.CustomHeaders,
		CreatedAt:       createdAt,
	}
}

// splitDomains splits a comma-separated domain whitelist string into a slice.
func splitDomains(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// joinDomains joins a domain whitelist slice into a comma-separated string.
func joinDomains(domains []string) string {
	// Filter out the default wildcard for cleaner UI display.
	if len(domains) == 1 && domains[0] == "*" {
		return ""
	}
	return strings.Join(domains, ", ")
}

// engineTaskManagerAdapter implements tools.EngineTaskManager using GORMTaskRepository.
type engineTaskManagerAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *engineTaskManagerAdapter) CreateTask(ctx context.Context, params tools.CreateEngineTaskParams) (string, error) {
	task := &domain.EngineTask{
		Title:       params.Title,
		Description: params.Description,
		AgentName:   params.AgentName,
		SessionID:   params.SessionID,
		Source:      domain.TaskSource(params.Source),
		UserID:      params.UserID,
		Status:      domain.EngineTaskStatusPending,
		Mode:        domain.TaskModeInteractive,
	}
	if err := a.repo.Create(ctx, task); err != nil {
		return "", err
	}
	return task.ID, nil
}

func (a *engineTaskManagerAdapter) UpdateTask(ctx context.Context, id string, title, description string) error {
	task, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if title != "" {
		task.Title = title
	}
	if description != "" {
		task.Description = description
	}
	return a.repo.Update(ctx, task)
}

func (a *engineTaskManagerAdapter) SetTaskStatus(ctx context.Context, id string, status string, result string) error {
	return a.repo.UpdateStatus(ctx, id, domain.EngineTaskStatus(status), result)
}

func (a *engineTaskManagerAdapter) ListTasks(ctx context.Context, sessionID string) ([]tools.EngineTaskSummary, error) {
	tasks, err := a.repo.List(ctx, config_repo.TaskFilter{SessionID: &sessionID})
	if err != nil {
		return nil, err
	}
	result := make([]tools.EngineTaskSummary, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, tools.EngineTaskSummary{
			ID:        t.ID,
			Title:     t.Title,
			Status:    string(t.Status),
			AgentName: t.AgentName,
			ParentID:  t.ParentTaskID,
		})
	}
	return result, nil
}

func (a *engineTaskManagerAdapter) CreateSubTask(ctx context.Context, parentID string, params tools.CreateEngineTaskParams) (string, error) {
	task := &domain.EngineTask{
		Title:        params.Title,
		Description:  params.Description,
		AgentName:    params.AgentName,
		SessionID:    params.SessionID,
		Source:       domain.TaskSource(params.Source),
		UserID:       params.UserID,
		ParentTaskID: &parentID,
		Status:       domain.EngineTaskStatusPending,
		Mode:         domain.TaskModeInteractive,
	}
	if err := a.repo.Create(ctx, task); err != nil {
		return "", err
	}
	return task.ID, nil
}

// schemaAgentResolverAdapter resolves schema UUID → agent names via schema repo.
type schemaAgentResolverAdapter struct {
	schemaRepo *config_repo.GORMSchemaRepository
}

func (a *schemaAgentResolverAdapter) ResolveAgents(ctx context.Context, schemaID string) ([]string, error) {
	return a.schemaRepo.ListAgents(ctx, schemaID)
}

func defaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

