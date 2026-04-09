package domain

import (
	"fmt"
	"strings"
	"time"
)

// WidgetPosition represents the position of the widget on the page.
type WidgetPosition string

const (
	WidgetPositionBottomRight WidgetPosition = "bottom-right"
	WidgetPositionBottomLeft  WidgetPosition = "bottom-left"
)

// WidgetSize represents the size of the widget.
type WidgetSize string

const (
	WidgetSizeCompact  WidgetSize = "compact"
	WidgetSizeStandard WidgetSize = "standard"
	WidgetSizeFull     WidgetSize = "full"
)

// Widget represents an embeddable chat widget bound to a schema.
type Widget struct {
	ID              string
	TenantID        string
	Name            string
	SchemaID        string
	PrimaryColor    string
	Position        WidgetPosition
	Size            WidgetSize
	WelcomeMessage  string
	Placeholder     string
	AvatarURL       string
	DomainWhitelist []string          // allowed origins, ["*"] = all
	CustomHeaders   map[string]string // headers forwarded with every chat request
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewWidget creates a new Widget with defaults and validation.
func NewWidget(name, schemaID string) (*Widget, error) {
	w := &Widget{
		Name:            name,
		SchemaID:        schemaID,
		PrimaryColor:    "#6366f1",
		Position:        WidgetPositionBottomRight,
		Size:            WidgetSizeStandard,
		WelcomeMessage:  "Hi! How can I help?",
		Placeholder:     "Type a message...",
		DomainWhitelist: []string{"*"},
		Enabled:         true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	return w, nil
}

// Validate validates the Widget.
func (w *Widget) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("widget name is required")
	}
	if w.SchemaID == "" {
		return fmt.Errorf("widget schema_id is required")
	}
	return nil
}

// IsOriginAllowed checks if the given origin is in the domain whitelist.
func (w *Widget) IsOriginAllowed(origin string) bool {
	if len(w.DomainWhitelist) == 0 {
		return true
	}
	for _, allowed := range w.DomainWhitelist {
		if allowed == "*" {
			return true
		}
		// Match domain (case-insensitive)
		if strings.EqualFold(allowed, origin) {
			return true
		}
		// Match if origin ends with the allowed domain (subdomain support)
		if strings.HasSuffix(strings.ToLower(origin), strings.ToLower(allowed)) {
			return true
		}
	}
	return false
}
