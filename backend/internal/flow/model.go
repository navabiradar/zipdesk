package flow

import (
	"time"

	"github.com/uptrace/bun"
)

// ═══════════════════════════════════════
// EVENT TYPES
// ═══════════════════════════════════════

// EventType defines all system events
type EventType = string

const (
	// Form events
	EventFormSubmitted EventType = "form.submitted"
	EventFormPublished EventType = "form.published"

	// Mail events
	EventMailContactAdded   EventType = "mail.contact_added"
	EventMailContactUpdated EventType = "mail.contact_updated"
	EventMailCampaignSent   EventType = "mail.campaign_sent"
	EventMailOpened         EventType = "mail.opened"
	EventMailClicked        EventType = "mail.clicked"
	EventMailUnsubscribed   EventType = "mail.unsubscribed"
	EventMailBounced        EventType = "mail.bounced"

	// Link events
	EventLinkClicked EventType = "link.clicked"
	EventLinkCreated EventType = "link.created"

	// Doc events
	EventDocViewed    EventType = "doc.viewed"
	EventDocPublished EventType = "doc.published"

	// CRM events
	EventCRMContactCreated EventType = "crm.contact_created"
	EventCRMDealCreated    EventType = "crm.deal_created"
	EventCRMDealWon        EventType = "crm.deal_won"
	EventCRMDealLost       EventType = "crm.deal_lost"

	// System events
	EventHealthCheck   EventType = "system.health_check"
	EventQuotaWarning  EventType = "system.quota_warning"
	EventQuotaExceeded EventType = "system.quota_exceeded"
)

// ActionType defines automation actions
type ActionType = string

const (
	ActionAddMailContact   ActionType = "mail.add_contact"
	ActionSendEmail        ActionType = "mail.send_email"
	ActionSendNotification ActionType = "system.notify"
	ActionWebhook          ActionType = "system.webhook"
	ActionUpdateContact    ActionType = "mail.update_contact"
	ActionWait             ActionType = "system.wait"
	ActionCreateCRMContact ActionType = "crm.create_contact"
)

// ═══════════════════════════════════════
// BASE EVENT
// ═══════════════════════════════════════

// BaseEvent is embedded in all events
type BaseEvent struct {
	ID          string    `json:"id"`
	Type        EventType `json:"type"`
	WorkspaceID string    `json:"workspace_id"`
	Source      string    `json:"source"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// ═══════════════════════════════════════
// SPECIFIC EVENT PAYLOADS
// ═══════════════════════════════════════

// FormSubmittedEvent fires when form submitted
type FormSubmittedEvent struct {
	BaseEvent
	FormID     string         `json:"form_id"`
	FormName   string         `json:"form_name"`
	ResponseID string         `json:"response_id"`
	Data       map[string]any `json:"data"`
	Email      string         `json:"email,omitempty"`
	Name       string         `json:"name,omitempty"`
}

// FormPublishedEvent fires when form published
type FormPublishedEvent struct {
	BaseEvent
	FormID   string `json:"form_id"`
	FormName string `json:"form_name"`
	FormSlug string `json:"form_slug"`
}

// MailContactAddedEvent fires on new contact
type MailContactAddedEvent struct {
	BaseEvent
	ContactID string `json:"contact_id"`
	Email     string `json:"email"`
	Source    string `json:"contact_source"`
}

// MailUnsubscribedEvent fires on unsubscribe
type MailUnsubscribedEvent struct {
	BaseEvent
	ContactID string `json:"contact_id"`
	Email     string `json:"email"`
}

// LinkClickedEvent fires on link click
type LinkClickedEvent struct {
	BaseEvent
	LinkID       string `json:"link_id"`
	ShortCode    string `json:"short_code"`
	Country      string `json:"country"`
	Device       string `json:"device"`
	ContactEmail string `json:"contact_email,omitempty"`
}

// LinkCreatedEvent fires on link creation
type LinkCreatedEvent struct {
	BaseEvent
	LinkID    string `json:"link_id"`
	ShortCode string `json:"short_code"`
}

// DocViewedEvent fires when doc is viewed
type DocViewedEvent struct {
	BaseEvent
	DocID       string `json:"doc_id"`
	ViewerEmail string `json:"viewer_email,omitempty"`
}

// DocPublishedEvent fires when doc published
type DocPublishedEvent struct {
	BaseEvent
	DocID    string `json:"doc_id"`
	DocTitle string `json:"doc_title"`
	DocSlug  string `json:"doc_slug"`
}

// CRMContactCreatedEvent fires on new CRM contact
type CRMContactCreatedEvent struct {
	BaseEvent
	ContactID string `json:"contact_id"`
	Email     string `json:"email"`
	Source    string `json:"crm_source"`
}

// CRMDealCreatedEvent fires on new deal
type CRMDealCreatedEvent struct {
	BaseEvent
	DealID string `json:"deal_id"`
	Title  string `json:"title"`
	Value  int    `json:"value,omitempty"`
}

// CRMDealWonEvent fires when deal is won
type CRMDealWonEvent struct {
	BaseEvent
	DealID string `json:"deal_id"`
	Title  string `json:"title"`
	Value  int    `json:"value,omitempty"`
}

// CRMDealLostEvent fires when deal is lost
type CRMDealLostEvent struct {
	BaseEvent
	DealID string `json:"deal_id"`
	Title  string `json:"title"`
	Reason string `json:"reason,omitempty"`
}

// HealthCheckEvent fires on health check
type HealthCheckEvent struct {
	BaseEvent
	Service   string `json:"service"`
	Status    string `json:"status"`
	Quota     int64  `json:"quota_used"`
	QuotaMax  int64  `json:"quota_max"`
	LatencyMs int64  `json:"latency_ms"`
}

// QuotaWarningEvent fires when quota threshold is hit
type QuotaWarningEvent struct {
	BaseEvent
	Service  string  `json:"service"`
	Quota    int64   `json:"quota_used"`
	QuotaMax int64   `json:"quota_max"`
	QuotaPct float64 `json:"quota_pct"`
}

// CampaignSentEvent fires when a campaign is sent
type CampaignSentEvent struct {
	BaseEvent
	CampaignID string `json:"campaign_id"`
	Recipients int    `json:"recipients"`
}

// CampaignOpenedEvent fires on email open
type CampaignOpenedEvent struct {
	BaseEvent
	CampaignID string `json:"campaign_id"`
	ContactID  string `json:"contact_id"`
	Email      string `json:"email"`
}

// CampaignClickedEvent fires on email link click
type CampaignClickedEvent struct {
	BaseEvent
	CampaignID string `json:"campaign_id"`
	ContactID  string `json:"contact_id"`
	Link       string `json:"link"`
}

// ═══════════════════════════════════════
// DATABASE MODELS
// ═══════════════════════════════════════

// Event is persisted to PostgreSQL
type Event struct {
	bun.BaseModel `bun:"table:events"`

	ID          string         `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID string         `bun:"workspace_id,notnull"            json:"workspace_id"`
	Type        EventType      `bun:"type,notnull"                    json:"type"`
	Source      string         `bun:"source,notnull"                  json:"source"`
	Payload     map[string]any `bun:"payload,type:jsonb"              json:"payload"`
	OccurredAt  time.Time      `bun:"occurred_at,default:now()"       json:"occurred_at"`
	ProcessedAt *time.Time     `bun:"processed_at"                    json:"processed_at,omitempty"`
	Error       string         `bun:"error"                           json:"error,omitempty"`
}

// FlowBlueprint defines an automation
type FlowBlueprint struct {
	bun.BaseModel `bun:"table:flow_blueprints"`

	ID            string         `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID   string         `bun:"workspace_id,notnull"            json:"workspace_id"`
	Name          string         `bun:"name,notnull"                    json:"name"`
	Description   string         `bun:"description,default:''"          json:"description"`
	TriggerType   EventType      `bun:"trigger_type,notnull"            json:"trigger_type"`
	TriggerConfig map[string]any `bun:"trigger_config,type:jsonb"       json:"trigger_config"`
	Actions       []FlowAction   `bun:"actions,type:jsonb"              json:"actions"`
	IsActive      bool           `bun:"is_active,default:true"          json:"is_active"`
	RunCount      int            `bun:"run_count,default:0"             json:"run_count"`
	LastRunAt     *time.Time     `bun:"last_run_at"                     json:"last_run_at,omitempty"`
	CreatedAt     time.Time      `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt     time.Time      `bun:"updated_at,default:now()"        json:"updated_at"`
}

// FlowAction defines one step in automation
type FlowAction struct {
	ID     string         `json:"id"`
	Type   ActionType     `json:"type"`
	Config map[string]any `json:"config"`
	Order  int            `json:"order"`
}

// BlueprintExecution logs a run
type BlueprintExecution struct {
	bun.BaseModel `bun:"table:blueprint_executions"`

	ID            string     `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	BlueprintID   string     `bun:"blueprint_id,notnull"            json:"blueprint_id"`
	WorkspaceID   string     `bun:"workspace_id,notnull"            json:"workspace_id"`
	EventID       string     `bun:"event_id"                        json:"event_id"`
	Status        string     `bun:"status,default:'running'"        json:"status"`
	ActionsRun    int        `bun:"actions_run,default:0"           json:"actions_run"`
	ActionsFailed int        `bun:"actions_failed,default:0"        json:"actions_failed"`
	Error         string     `bun:"error"                           json:"error,omitempty"`
	StartedAt     time.Time  `bun:"started_at,default:now()"        json:"started_at"`
	CompletedAt   *time.Time `bun:"completed_at"                    json:"completed_at,omitempty"`
}

// AIConversation holds chat history
type AIConversation struct {
	bun.BaseModel `bun:"table:ai_conversations"`

	ID           string    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID  string    `bun:"workspace_id,notnull"            json:"workspace_id"`
	UserID       string    `bun:"user_id,notnull"                 json:"user_id"`
	Title        string    `bun:"title,default:'New conversation'" json:"title"`
	MessageCount int       `bun:"message_count,default:0"         json:"message_count"`
	CreatedAt    time.Time `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt    time.Time `bun:"updated_at,default:now()"        json:"updated_at"`
}

// AIMessage holds a single chat message
type AIMessage struct {
	bun.BaseModel `bun:"table:ai_messages"`

	ID             string       `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	ConversationID string       `bun:"conversation_id,notnull"         json:"conversation_id"`
	Role           string       `bun:"role,notnull"                    json:"role"`
	Content        string       `bun:"content,notnull"                 json:"content"`
	ToolCalls      []ToolCall   `bun:"tool_calls,type:jsonb"           json:"tool_calls,omitempty"`
	ToolResults    []ToolResult `bun:"tool_results,type:jsonb"         json:"tool_results,omitempty"`
	TokensUsed     int          `bun:"tokens_used,default:0"           json:"tokens_used"`
	CreatedAt      time.Time    `bun:"created_at,default:now()"        json:"created_at"`
}

// ToolCall represents an AI tool invocation
type ToolCall struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// ToolResult holds result of a tool call
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// ServiceHealth holds service status
type ServiceHealth struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	LatencyMs int64   `json:"latency_ms"`
	QuotaUsed int64   `json:"quota_used,omitempty"`
	QuotaMax  int64   `json:"quota_max,omitempty"`
	QuotaPct  float64 `json:"quota_pct,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// HealthReport holds all service statuses
type HealthReport struct {
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]ServiceHealth `json:"services"`
	Overall   string                   `json:"overall"`
}

// CreateBlueprintInput holds creation data
type CreateBlueprintInput struct {
	Name          string         `json:"name"         validate:"required"`
	Description   string         `json:"description"`
	TriggerType   EventType      `json:"trigger_type" validate:"required"`
	TriggerConfig map[string]any `json:"trigger_config"`
	Actions       []FlowAction   `json:"actions"`
}

// ChatInput holds a chat message
type ChatInput struct {
	Message        string      `json:"message" validate:"required"`
	ConversationID string      `json:"conversation_id"`
	History        []AIMessage `json:"history"`
}
