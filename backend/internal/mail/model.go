package mail

import (
	"time"

	"github.com/uptrace/bun"
)

// ContactStatus defines contact states
type ContactStatus = string

const (
	ContactStatusSubscribed   ContactStatus = "subscribed"
	ContactStatusUnsubscribed ContactStatus = "unsubscribed"
	ContactStatusBounced      ContactStatus = "bounced"
)

// CampaignStatus defines campaign states
type CampaignStatus = string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusSending   CampaignStatus = "sending"
	CampaignStatusSent      CampaignStatus = "sent"
)

// Contact represents an email contact
type Contact struct {
	bun.BaseModel `bun:"table:mail_contacts"`

	ID             string         `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID    string         `bun:"workspace_id,notnull"            json:"workspace_id"`
	Email          string         `bun:"email,notnull"                   json:"email"`
	FirstName      string         `bun:"first_name,default:''"           json:"first_name"`
	LastName       string         `bun:"last_name,default:''"            json:"last_name"`
	Company        string         `bun:"company,default:''"              json:"company"`
	Phone          string         `bun:"phone,default:''"                json:"phone"`
	Tags           []string       `bun:"tags,type:jsonb"                 json:"tags"`
	CustomFields   map[string]any `bun:"custom_fields,type:jsonb"        json:"custom_fields"`
	Status         ContactStatus  `bun:"status,default:'subscribed'"     json:"status"`
	Source         string         `bun:"source,default:'manual'"         json:"source"`
	SubscribedAt   time.Time      `bun:"subscribed_at,default:now()"     json:"subscribed_at"`
	UnsubscribedAt *time.Time     `bun:"unsubscribed_at"                 json:"unsubscribed_at,omitempty"`
	CreatedAt      time.Time      `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt      time.Time      `bun:"updated_at,default:now()"        json:"updated_at"`
}

// MailList represents an email list
type MailList struct {
	bun.BaseModel `bun:"table:mail_lists"`

	ID           UUID      `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID  string    `bun:"workspace_id,notnull"            json:"workspace_id"`
	Name         string    `bun:"name,notnull"                    json:"name"`
	Description  string    `bun:"description,default:''"          json:"description"`
	ContactCount int       `bun:"contact_count,default:0"         json:"contact_count"`
	CreatedAt    time.Time `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt    time.Time `bun:"updated_at,default:now()"        json:"updated_at"`
}

// CampaignContent holds email content
type CampaignContent struct {
	HTML    string `json:"html"`
	Text    string `json:"text"`
	Subject string `json:"subject"`
}

// Campaign represents an email campaign
type Campaign struct {
	bun.BaseModel `bun:"table:mail_campaigns"`

	ID          string          `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID string          `bun:"workspace_id,notnull"            json:"workspace_id"`
	Name        string          `bun:"name,notnull"                    json:"name"`
	Subject     string          `bun:"subject,notnull"                 json:"subject"`
	PreviewText string          `bun:"preview_text,default:''"         json:"preview_text"`
	FromName    string          `bun:"from_name,notnull"               json:"from_name"`
	FromEmail   string          `bun:"from_email,notnull"              json:"from_email"`
	Content     CampaignContent `bun:"content,type:jsonb"              json:"content"`
	ListID      string          `bun:"list_id,nullzero"                json:"list_id"`
	Status      CampaignStatus  `bun:"status,default:'draft'"          json:"status"`
	ScheduledAt *time.Time      `bun:"scheduled_at"                    json:"scheduled_at,omitempty"`
	SentAt      *time.Time      `bun:"sent_at"                         json:"sent_at,omitempty"`
	CreatedAt   time.Time       `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt   time.Time       `bun:"updated_at,default:now()"        json:"updated_at"`
}

// CampaignStats holds campaign metrics
type CampaignStats struct {
	bun.BaseModel `bun:"table:mail_campaign_stats"`

	CampaignID   string    `bun:"campaign_id,pk"          json:"campaign_id"`
	Sent         int       `bun:"sent,default:0"          json:"sent"`
	Delivered    int       `bun:"delivered,default:0"     json:"delivered"`
	Opened       int       `bun:"opened,default:0"        json:"opened"`
	Clicked      int       `bun:"clicked,default:0"       json:"clicked"`
	Bounced      int       `bun:"bounced,default:0"       json:"bounced"`
	Unsubscribed int       `bun:"unsubscribed,default:0"  json:"unsubscribed"`
	UpdatedAt    time.Time `bun:"updated_at,default:now()" json:"updated_at"`
}

// CreateContactInput holds contact creation data
type CreateContactInput struct {
	Email        string         `json:"email"      validate:"required,email"`
	FirstName    string         `json:"first_name"`
	LastName     string         `json:"last_name"`
	Company      string         `json:"company"`
	Phone        string         `json:"phone"`
	Tags         []string       `json:"tags"`
	CustomFields map[string]any `json:"custom_fields"`
	Source       string         `json:"source"`
}

// UpsertContactData holds upsert data
type UpsertContactData struct {
	Email     string
	FirstName string
	LastName  string
	Source    string
	Tags      []string
	Extra     map[string]any
}

// CreateCampaignInput holds campaign creation
type CreateCampaignInput struct {
	Name        string          `json:"name"       validate:"required"`
	Subject     string          `json:"subject"    validate:"required"`
	PreviewText string          `json:"preview_text"`
	FromName    string          `json:"from_name"  validate:"required"`
	FromEmail   string          `json:"from_email" validate:"required,email"`
	Content     CampaignContent `json:"content"`
	ListID      string          `json:"list_id"`
}

// ListParams holds pagination params
type ListParams struct {
	Page    int    `query:"page"`
	PerPage int    `query:"per_page"`
	Search  string `query:"search"`
}

// MailError is a domain error
type MailError struct {
	Code    string
	Message string
}

func (e *MailError) Error() string {
	return e.Code + ": " + e.Message
}

// UUID type alias for clarity
type UUID = string
