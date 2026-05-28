package crm

import (
	"time"

	"github.com/uptrace/bun"
)

// CRMContact represents a CRM contact
type CRMContact struct {
	bun.BaseModel `bun:"table:crm_contacts"`

	ID             string         `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID    string         `bun:"workspace_id,notnull"            json:"workspace_id"`
	FirstName      string         `bun:"first_name,default:''"           json:"first_name"`
	LastName       string         `bun:"last_name,default:''"            json:"last_name"`
	Email          string         `bun:"email,default:''"                json:"email"`
	Phone          string         `bun:"phone,default:''"                json:"phone"`
	JobTitle       string         `bun:"job_title,default:''"            json:"job_title"`
	CompanyID      *string         `bun:"company_id"                      json:"company_id,omitempty"`
	LeadSource     string          `bun:"lead_source,default:''"          json:"lead_source"`
	LeadStatus     string          `bun:"lead_status,default:'new'"       json:"lead_status"`
	LeadScore      int             `bun:"lead_score,default:0"            json:"lead_score"`
	OwnerID        *string         `bun:"owner_id"                        json:"owner_id,omitempty"`
	Tags           []string       `bun:"tags,type:jsonb"                 json:"tags"`
	CustomFields   map[string]any `bun:"custom_fields,type:jsonb"        json:"custom_fields"`
	LastActivityAt *time.Time     `bun:"last_activity_at"                json:"last_activity_at,omitempty"`
	CreatedAt      time.Time      `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt      time.Time      `bun:"updated_at,default:now()"        json:"updated_at"`
}

// CRMDeal represents a sales deal
type CRMDeal struct {
	bun.BaseModel `bun:"table:crm_deals"`

	ID          string     `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID string     `bun:"workspace_id,notnull"            json:"workspace_id"`
	Title       string     `bun:"title,notnull"                   json:"title"`
	ContactID   string     `bun:"contact_id"                      json:"contact_id,omitempty"`
	CompanyID   string     `bun:"company_id"                      json:"company_id,omitempty"`
	PipelineID  string     `bun:"pipeline_id"                     json:"pipeline_id,omitempty"`
	StageID     string     `bun:"stage_id"                        json:"stage_id,omitempty"`
	Value       float64    `bun:"value,default:0"                 json:"value"`
	Currency    string     `bun:"currency,default:'USD'"          json:"currency"`
	Probability int        `bun:"probability,default:0"           json:"probability"`
	OwnerID     string     `bun:"owner_id"                        json:"owner_id,omitempty"`
	WonAt       *time.Time `bun:"won_at"                          json:"won_at,omitempty"`
	LostAt      *time.Time `bun:"lost_at"                         json:"lost_at,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,default:now()"        json:"updated_at"`
}

// CreateContactInput holds CRM contact data
type CreateContactInput struct {
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	JobTitle   string   `json:"job_title"`
	LeadSource string   `json:"lead_source"`
	Tags       []string `json:"tags"`
}

// ListParams holds pagination params
type ListParams struct {
	Page    int    `query:"page"`
	PerPage int    `query:"per_page"`
	Search  string `query:"search"`
}

// CRMError is a domain error
type CRMError struct {
	Code    string
	Message string
}

func (e *CRMError) Error() string {
	return e.Code + ": " + e.Message
}
