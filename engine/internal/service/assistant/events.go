package assistant

import (
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AdminEventType defines the type of admin SSE event for live animation.
type AdminEventType string

const (
	AdminEventNodeCreate   AdminEventType = "admin.node_create"
	AdminEventNodeUpdate   AdminEventType = "admin.node_update"
	AdminEventEdgeCreate   AdminEventType = "admin.edge_create"
	AdminEventPageNavigate AdminEventType = "admin.page_navigate"
	AdminEventFieldUpdate  AdminEventType = "admin.field_update"
)

// AnimationType defines the visual animation for the admin UI.
type AnimationType string

const (
	AnimationFadeIn  AnimationType = "fade_in"
	AnimationPulse   AnimationType = "pulse"
	AnimationDraw    AnimationType = "draw"
	AnimationTyping  AnimationType = "typing"
	AnimationSlideDown AnimationType = "slide_down"
)

// AdminEvent represents an SSE event for the admin UI live animation.
type AdminEvent struct {
	Type      AdminEventType         `json:"type"`
	Target    string                 `json:"target,omitempty"`    // page/component/field
	Action    string                 `json:"action,omitempty"`    // create/update/delete
	Value     interface{}            `json:"value,omitempty"`     // the data
	Animation AnimationType          `json:"animation,omitempty"` // visual effect
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewNodeCreateEvent creates an admin.node_create event.
func NewNodeCreateEvent(agentName string, position map[string]float64) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          domain.AgentEventType(AdminEventNodeCreate),
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		Content:       agentName,
		Metadata: map[string]interface{}{
			"agent_name": agentName,
			"position":   position,
			"animation":  string(AnimationFadeIn),
		},
	}
}

// NewNodeUpdateEvent creates an admin.node_update event.
func NewNodeUpdateEvent(agentName, field string, value interface{}) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          domain.AgentEventType(AdminEventNodeUpdate),
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		Content:       agentName,
		Metadata: map[string]interface{}{
			"agent_name": agentName,
			"field":      field,
			"value":      value,
			"animation":  string(AnimationPulse),
		},
	}
}

// NewEdgeCreateEvent creates an admin.edge_create event.
func NewEdgeCreateEvent(source, target, edgeType string) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          domain.AgentEventType(AdminEventEdgeCreate),
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"source":    source,
			"target":    target,
			"edge_type": edgeType,
			"animation": string(AnimationDraw),
		},
	}
}

// NewPageNavigateEvent creates an admin.page_navigate event.
func NewPageNavigateEvent(page string, params map[string]interface{}) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          domain.AgentEventType(AdminEventPageNavigate),
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"page":   page,
			"params": params,
		},
	}
}

// NewFieldUpdateEvent creates an admin.field_update event.
func NewFieldUpdateEvent(page, component, field string, value interface{}) *domain.AgentEvent {
	return &domain.AgentEvent{
		Type:          domain.AgentEventType(AdminEventFieldUpdate),
		SchemaVersion: domain.EventSchemaVersion,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"page":      page,
			"component": component,
			"field":     field,
			"value":     value,
			"animation": string(AnimationTyping),
		},
	}
}
