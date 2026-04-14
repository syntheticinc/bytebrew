package configrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// WidgetRecord is an intermediate struct for DB <-> domain mapping.
type WidgetRecord struct {
	ID              string
	TenantID        string
	Name            string
	SchemaID        string
	PrimaryColor    string
	Position        string
	Size            string
	WelcomeMessage  string
	Placeholder     string
	AvatarURL       string
	DomainWhitelist []string
	CustomHeaders   map[string]string
	Enabled         bool
	CreatedAt       time.Time
}

// GORMWidgetRepository implements widget CRUD using GORM.
type GORMWidgetRepository struct {
	db *gorm.DB
}

// NewGORMWidgetRepository creates a new GORMWidgetRepository.
func NewGORMWidgetRepository(db *gorm.DB) *GORMWidgetRepository {
	return &GORMWidgetRepository{db: db}
}

// List returns all widgets, optionally scoped by tenant.
func (r *GORMWidgetRepository) List(ctx context.Context, tenantID string) ([]WidgetRecord, error) {
	query := r.db.WithContext(ctx)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}

	var widgets []models.WidgetModel
	if err := query.Order("created_at ASC").Find(&widgets).Error; err != nil {
		return nil, fmt.Errorf("list widgets: %w", err)
	}

	records := make([]WidgetRecord, 0, len(widgets))
	for _, w := range widgets {
		records = append(records, toWidgetRecord(w))
	}
	return records, nil
}

// GetByID returns a single widget by ID.
func (r *GORMWidgetRepository) GetByID(ctx context.Context, id string) (*WidgetRecord, error) {
	var widget models.WidgetModel
	if err := r.db.WithContext(ctx).First(&widget, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("get widget %s: %w", id, err)
	}
	rec := toWidgetRecord(widget)
	return &rec, nil
}

// Create inserts a new widget.
func (r *GORMWidgetRepository) Create(ctx context.Context, record *WidgetRecord) error {
	model := toWidgetModel(record)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create widget: %w", err)
	}
	record.ID = model.ID
	return nil
}

// Update updates an existing widget by ID.
func (r *GORMWidgetRepository) Update(ctx context.Context, id string, record *WidgetRecord) error {
	customHeadersJSON := ""
	if len(record.CustomHeaders) > 0 {
		b, err := json.Marshal(record.CustomHeaders)
		if err != nil {
			return fmt.Errorf("marshal custom headers: %w", err)
		}
		customHeadersJSON = string(b)
	}

	result := r.db.WithContext(ctx).Model(&models.WidgetModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":             record.Name,
		"schema_id":        record.SchemaID,
		"primary_color":    record.PrimaryColor,
		"position":         record.Position,
		"size":             record.Size,
		"welcome_message":  record.WelcomeMessage,
		"placeholder":      record.Placeholder,
		"avatar_url":       record.AvatarURL,
		"domain_whitelist": strings.Join(record.DomainWhitelist, ","),
		"custom_headers":   customHeadersJSON,
		"enabled":          record.Enabled,
	})
	if result.Error != nil {
		return fmt.Errorf("update widget %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a widget by ID.
func (r *GORMWidgetRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.WidgetModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete widget %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func toWidgetRecord(w models.WidgetModel) WidgetRecord {
	var domains []string
	if w.DomainWhitelist != "" {
		domains = strings.Split(w.DomainWhitelist, ",")
		for i := range domains {
			domains[i] = strings.TrimSpace(domains[i])
		}
	}

	var customHeaders map[string]string
	if w.CustomHeaders != "" {
		_ = json.Unmarshal([]byte(w.CustomHeaders), &customHeaders)
	}

	return WidgetRecord{
		ID:              w.ID,
		TenantID:        w.TenantID,
		Name:            w.Name,
		SchemaID:        w.SchemaID,
		PrimaryColor:    w.PrimaryColor,
		Position:        w.Position,
		Size:            w.Size,
		WelcomeMessage:  w.WelcomeMessage,
		Placeholder:     w.Placeholder,
		AvatarURL:       w.AvatarURL,
		DomainWhitelist: domains,
		CustomHeaders:   customHeaders,
		Enabled:         w.Enabled,
		CreatedAt:       w.CreatedAt,
	}
}

func toWidgetModel(r *WidgetRecord) models.WidgetModel {
	customHeadersJSON := ""
	if len(r.CustomHeaders) > 0 {
		b, _ := json.Marshal(r.CustomHeaders)
		customHeadersJSON = string(b)
	}

	return models.WidgetModel{
		TenantID:        r.TenantID,
		Name:            r.Name,
		SchemaID:        r.SchemaID,
		PrimaryColor:    r.PrimaryColor,
		Position:        r.Position,
		Size:            r.Size,
		WelcomeMessage:  r.WelcomeMessage,
		Placeholder:     r.Placeholder,
		AvatarURL:       r.AvatarURL,
		DomainWhitelist: strings.Join(r.DomainWhitelist, ","),
		CustomHeaders:   customHeadersJSON,
		Enabled:         r.Enabled,
	}
}
