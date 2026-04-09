package app

import (
	"context"
	"errors"
	"fmt"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
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

func (a *widgetServiceHTTPAdapter) GetWidget(ctx context.Context, id uint) (*deliveryhttp.WidgetInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("widget not found: %d", id))
		}
		return nil, fmt.Errorf("get widget: %w", err)
	}

	info := toWidgetInfo(*record)
	return &info, nil
}

func (a *widgetServiceHTTPAdapter) CreateWidget(ctx context.Context, req deliveryhttp.CreateWidgetRequest) (*deliveryhttp.WidgetInfo, error) {
	tenantID := domain.TenantIDFromContext(ctx)

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	record := &config_repo.WidgetRecord{
		TenantID:        tenantID,
		Name:            req.Name,
		SchemaID:        req.SchemaID,
		PrimaryColor:    defaultStr(req.PrimaryColor, "#6366f1"),
		Position:        defaultStr(req.Position, "bottom-right"),
		Size:            defaultStr(req.Size, "standard"),
		WelcomeMessage:  defaultStr(req.WelcomeMessage, "Hi! How can I help?"),
		Placeholder:     defaultStr(req.Placeholder, "Type a message..."),
		AvatarURL:       req.AvatarURL,
		DomainWhitelist: req.DomainWhitelist,
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

func (a *widgetServiceHTTPAdapter) UpdateWidget(ctx context.Context, id uint, req deliveryhttp.CreateWidgetRequest) error {
	existing, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("widget not found: %d", id))
		}
		return fmt.Errorf("get widget: %w", err)
	}

	record := &config_repo.WidgetRecord{
		Name:            defaultStr(req.Name, existing.Name),
		SchemaID:        defaultUint(req.SchemaID, existing.SchemaID),
		PrimaryColor:    defaultStr(req.PrimaryColor, existing.PrimaryColor),
		Position:        defaultStr(req.Position, existing.Position),
		Size:            defaultStr(req.Size, existing.Size),
		WelcomeMessage:  defaultStr(req.WelcomeMessage, existing.WelcomeMessage),
		Placeholder:     defaultStr(req.Placeholder, existing.Placeholder),
		AvatarURL:       defaultStr(req.AvatarURL, existing.AvatarURL),
		DomainWhitelist: existing.DomainWhitelist,
		CustomHeaders:   existing.CustomHeaders,
		Enabled:         existing.Enabled,
	}
	if len(req.DomainWhitelist) > 0 {
		record.DomainWhitelist = req.DomainWhitelist
	}
	if req.CustomHeaders != nil {
		record.CustomHeaders = req.CustomHeaders
	}
	if req.Enabled != nil {
		record.Enabled = *req.Enabled
	}

	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("widget not found: %d", id))
		}
		return fmt.Errorf("update widget: %w", err)
	}
	return nil
}

func (a *widgetServiceHTTPAdapter) DeleteWidget(ctx context.Context, id uint) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("widget not found: %d", id))
		}
		return fmt.Errorf("delete widget: %w", err)
	}
	return nil
}

func toWidgetInfo(r config_repo.WidgetRecord) deliveryhttp.WidgetInfo {
	return deliveryhttp.WidgetInfo{
		ID:              r.ID,
		Name:            r.Name,
		SchemaID:        r.SchemaID,
		PrimaryColor:    r.PrimaryColor,
		Position:        r.Position,
		Size:            r.Size,
		WelcomeMessage:  r.WelcomeMessage,
		Placeholder:     r.Placeholder,
		AvatarURL:       r.AvatarURL,
		DomainWhitelist: r.DomainWhitelist,
		CustomHeaders:   r.CustomHeaders,
		Enabled:         r.Enabled,
	}
}

func defaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func defaultUint(val, def uint) uint {
	if val == 0 {
		return def
	}
	return val
}
