package docs

import (
	"time"

	"github.com/uptrace/bun"
)

// DocType defines document types
type DocType = string

const (
	DocTypeProposal DocType = "proposal"
	DocTypeContract DocType = "contract"
	DocTypeInvoice  DocType = "invoice"
	DocTypeReport   DocType = "report"
	DocTypeOther    DocType = "other"
)

// DocStatus defines document states
type DocStatus = string

const (
	DocStatusDraft     DocStatus = "draft"
	DocStatusPublished DocStatus = "published"
	DocStatusExpired   DocStatus = "expired"
)

// ContentBlock is a document block
type ContentBlock struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Content string         `json:"content"`
	Props   map[string]any `json:"props,omitempty"`
}

// DocumentContent holds doc content
type DocumentContent struct {
	Blocks    []ContentBlock    `json:"blocks"`
	Variables map[string]string `json:"variables"`
}

// DocSettings holds document settings
type DocSettings struct {
	Password      string `json:"password"`
	AllowDownload bool   `json:"allow_download"`
	RequireEmail  bool   `json:"require_email"`
	Watermark     string `json:"watermark"`
}

// Document represents a ZipDesk document
type Document struct {
	bun.BaseModel `bun:"table:documents"`

	ID             string          `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID    string          `bun:"workspace_id,notnull"            json:"workspace_id"`
	Title          string          `bun:"title,notnull"                   json:"title"`
	Slug           string          `bun:"slug,notnull"                    json:"slug"`
	Type           DocType         `bun:"type,default:'other'"            json:"type"`
	Status         DocStatus       `bun:"status,default:'draft'"          json:"status"`
	Content        DocumentContent `bun:"content,type:jsonb"              json:"content"`
	TemplateID     string     `bun:"template_id,scanonly" json:"template_id,omitempty"`
	PDFUrl         string          `bun:"pdf_url"                         json:"pdf_url,omitempty"`
	PDFGeneratedAt *time.Time      `bun:"pdf_generated_at"                json:"pdf_generated_at,omitempty"`
	Settings       DocSettings     `bun:"settings,type:jsonb"             json:"settings"`
	IsPublished    bool            `bun:"is_published,default:false"      json:"is_published"`
	ExpiresAt      *time.Time      `bun:"expires_at"                      json:"expires_at,omitempty"`
	CreatedBy      string          `bun:"created_by"                      json:"created_by"`
	CreatedAt      time.Time       `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt      time.Time       `bun:"updated_at,default:now()"        json:"updated_at"`
}

// CreateDocInput holds doc creation data
type CreateDocInput struct {
	Title    string          `json:"title" validate:"required"`
	Type     DocType         `json:"type"`
	Content  DocumentContent `json:"content"`
	Settings DocSettings     `json:"settings"`
}

// UpdateDocInput holds doc update data
type UpdateDocInput struct {
	Title    string          `json:"title"`
	Content  DocumentContent `json:"content"`
	Settings DocSettings     `json:"settings"`
}

// ListParams holds pagination params
type ListParams struct {
	Page    int    `query:"page"`
	PerPage int    `query:"per_page"`
	Search  string `query:"search"`
}

// DocsError is a domain error
type DocsError struct {
	Code    string
	Message string
}

func (e *DocsError) Error() string {
	return e.Code + ": " + e.Message
}
