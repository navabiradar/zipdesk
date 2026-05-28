package forms

import (
	"time"

	"github.com/uptrace/bun"
)

// FieldType defines supported field types
type FieldType string

const (
	FieldTypeText      FieldType = "text"
	FieldTypeEmail     FieldType = "email"
	FieldTypePhone     FieldType = "phone"
	FieldTypeNumber    FieldType = "number"
	FieldTypeDate      FieldType = "date"
	FieldTypeDropdown  FieldType = "dropdown"
	FieldTypeMCQ       FieldType = "mcq"
	FieldTypeCheckbox  FieldType = "checkbox"
	FieldTypeRating    FieldType = "rating"
	FieldTypeYesNo     FieldType = "yes_no"
	FieldTypeFile      FieldType = "file"
	FieldTypeLongText  FieldType = "long_text"
	FieldTypeSignature FieldType = "signature"
)

// FormSettings holds form configuration
type FormSettings struct {
	SubmitMessage string  `json:"submit_message"`
	RedirectURL   string  `json:"redirect_url"`
	ResponseLimit *int    `json:"response_limit"`
	CloseDate     *string `json:"close_date"`
	Password      string  `json:"password"`
	AllowEdit     bool    `json:"allow_edit"`
	OnePerEmail   bool    `json:"one_per_email"`
	NotifyEmail   string  `json:"notify_email"`
	ShowProgress  bool    `json:"show_progress"`
}

// FieldOption represents a choice option
type FieldOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// FieldValidation holds validation rules
type FieldValidation struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	MinLen  *int     `json:"min_length,omitempty"`
	MaxLen  *int     `json:"max_length,omitempty"`
	Pattern string   `json:"pattern,omitempty"`
}

// FieldLogic holds conditional logic rules
type FieldLogic struct {
	Condition      LogicCondition  `json:"condition"`
	ConditionGroup *ConditionGroup `json:"condition_group,omitempty"`
	Action         string          `json:"action"`
	Target         string          `json:"target"`
}

// LogicCondition defines when logic triggers
type LogicCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// ConditionGroup groups conditions with AND/OR logic
type ConditionGroup struct {
	Match      string           `json:"match"` // "all" or "any"
	Conditions []LogicCondition `json:"conditions"`
}

// Form represents a ZipDesk form
type Form struct {
	bun.BaseModel `bun:"table:forms"`

	ID          string       `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID string       `bun:"workspace_id,notnull"            json:"workspace_id"`
	Title       string       `bun:"title,notnull"                   json:"title"`
	Description string       `bun:"description,default:''"          json:"description"`
	Slug        string       `bun:"slug,notnull"                    json:"slug"`
	Settings    FormSettings `bun:"settings,type:jsonb"             json:"settings"`
	IsPublished bool         `bun:"is_published,default:false"      json:"is_published"`
	PublishedAt *time.Time   `bun:"published_at"                    json:"published_at,omitempty"`
	CreatedBy   string       `bun:"created_by"                      json:"created_by"`
	CreatedAt   time.Time    `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt   time.Time    `bun:"updated_at,default:now()"        json:"updated_at"`

	// Relations
	Fields []FormField `bun:"rel:has-many,join:id=form_id" json:"fields,omitempty"`
}

// FormField represents a single form field
type FormField struct {
	bun.BaseModel `bun:"table:form_fields"`

	ID          string          `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	FormID      string          `bun:"form_id,notnull"                 json:"form_id"`
	Type        FieldType       `bun:"type,notnull"                    json:"type"`
	Label       string          `bun:"label,notnull"                   json:"label"`
	Placeholder string          `bun:"placeholder,default:''"          json:"placeholder"`
	HelperText  string          `bun:"helper_text,default:''"          json:"helper_text"`
	Required    bool            `bun:"required,default:false"          json:"required"`
	Options     []FieldOption   `bun:"options,type:jsonb"              json:"options"`
	Validation  FieldValidation `bun:"validation,type:jsonb"           json:"validation"`
	Logic       []FieldLogic    `bun:"logic,type:jsonb"                json:"logic"`
	FieldOrder  int             `bun:"field_order,default:0"           json:"order"`
	CreatedAt   time.Time       `bun:"created_at,default:now()"        json:"created_at"`
}

// FormResponse stores a form submission
type FormResponse struct {
	bun.BaseModel `bun:"table:form_responses"`

	ID             string         `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	FormID         string         `bun:"form_id,notnull"                 json:"form_id"`
	WorkspaceID    string         `bun:"workspace_id,notnull"            json:"workspace_id"`
	Data           map[string]any `bun:"data,type:jsonb"                 json:"data"`
	Score          *int           `bun:"score"                           json:"score,omitempty"`
	IPAddress      string         `bun:"ip_address"                      json:"-"`
	UserAgent      string         `bun:"user_agent"                      json:"-"`
	Referrer       string         `bun:"referrer"                        json:"-"`
	CompletionTime int            `bun:"completion_time,default:0"       json:"completion_time"`
	IsComplete     bool           `bun:"is_complete,default:true"        json:"is_complete"`
	SubmittedAt    time.Time      `bun:"submitted_at,default:now()"      json:"submitted_at"`
}

// FormView tracks form views for analytics
type FormView struct {
	bun.BaseModel `bun:"table:form_views"`

	ID        string    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	FormID    string    `bun:"form_id,notnull"                 json:"form_id"`
	IPAddress string    `bun:"ip_address"                      json:"-"`
	Device    string    `bun:"device"                          json:"device"`
	Referrer  string    `bun:"referrer"                        json:"referrer"`
	ViewedAt  time.Time `bun:"viewed_at,default:now()"         json:"viewed_at"`
}

// CreateFormInput holds form creation data
type CreateFormInput struct {
	Title       string       `json:"title"       validate:"required,min=1,max=200"`
	Description string       `json:"description"`
	Settings    FormSettings `json:"settings"`
	Fields      []FormField  `json:"fields"`
}

// UpdateFormInput holds form update data
type UpdateFormInput struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Settings    FormSettings `json:"settings"`
	Fields      []FormField  `json:"fields"`
}

// SubmitInput holds form submission data
type SubmitInput struct {
	Data           map[string]any `json:"data"            validate:"required"`
	CompletionTime int            `json:"completion_time"`
}

// ResponseMeta holds submission metadata
type ResponseMeta struct {
	IP        string
	UserAgent string
	Referrer  string
}

// FormPublicView is the public form representation
type FormPublicView struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Slug        string       `json:"slug"`
	Settings    FormSettings `json:"settings"`
	Fields      []FormField  `json:"fields"`
}

// FormAnalytics holds form performance data
type FormAnalytics struct {
	TotalViews     int64          `json:"total_views"`
	TotalResponses int64          `json:"total_responses"`
	CompletionRate float64        `json:"completion_rate"`
	AverageTime    float64        `json:"average_time_seconds"`
	ResponsesByDay []DayCount     `json:"responses_by_day"`
	FieldDropoff   []FieldDropoff `json:"field_dropoff"`
}

// DayCount holds daily counts
type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// FieldDropoff holds field analytics
type FieldDropoff struct {
	FieldID    string  `json:"field_id"`
	FieldLabel string  `json:"field_label"`
	Dropoff    float64 `json:"dropoff_rate"`
}

// ListParams holds pagination params
type ListParams struct {
	Page    int    `query:"page"`
	PerPage int    `query:"per_page"`
	Search  string `query:"search"`
}

// ListResponse wraps paginated results
type ListResponse struct {
	Items   []Form `json:"items"`
	Total   int64  `json:"total"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

// ResponseListResponse wraps responses
type ResponseListResponse struct {
	Items   []FormResponse `json:"items"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}

// FormError is a domain error
type FormError struct {
	Code    string
	Message string
	Field   string
}

func (e *FormError) Error() string {
	return e.Code + ": " + e.Message
}
