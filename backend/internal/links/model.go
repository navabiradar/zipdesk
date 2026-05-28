package links

import (
	"time"

	"github.com/uptrace/bun"
)

// Link represents a shortened URL
type Link struct {
	bun.BaseModel `bun:"table:links"`

	ID           string         `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID  string         `bun:"workspace_id,notnull"            json:"workspace_id"`
	OriginalURL  string         `bun:"original_url,notnull"            json:"original_url"`
	ShortCode    string         `bun:"short_code,notnull"              json:"short_code"`
	CustomSlug   string         `bun:"custom_slug,nullzero"            json:"custom_slug,omitempty"`
	CustomDomain string         `bun:"custom_domain"                   json:"custom_domain,omitempty"`
	Title        string         `bun:"title,default:''"                json:"title"`
	Description  string         `bun:"description,default:''"          json:"description"`
	Tags         []string       `bun:"tags,type:jsonb"                 json:"tags"`
	FolderID     string         `bun:"folder_id,nullzero"              json:"folder_id,omitempty"`
	Password     string         `bun:"password"                        json:"-"`
	ExpiresAt    *time.Time     `bun:"expires_at"                      json:"expires_at,omitempty"`
	ClickLimit   *int           `bun:"click_limit"                     json:"click_limit,omitempty"`
	TotalClicks  int            `bun:"total_clicks,default:0"          json:"total_clicks"`
	UniqueClicks int            `bun:"unique_clicks,default:0"         json:"unique_clicks"`
	IsActive     bool           `bun:"is_active,default:true"          json:"is_active"`
	UTMParams    map[string]any `bun:"utm_params,type:jsonb"           json:"utm_params,omitempty"`
	Settings     map[string]any `bun:"settings,type:jsonb"             json:"settings,omitempty"`
	CreatedBy    string         `bun:"created_by,nullzero"             json:"created_by"`
	CreatedAt    time.Time      `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt    time.Time      `bun:"updated_at,default:now()"        json:"updated_at"`
}

// LinkFolder organizes links
type LinkFolder struct {
	bun.BaseModel `bun:"table:link_folders"`

	ID          string    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID string    `bun:"workspace_id,notnull"            json:"workspace_id"`
	Name        string    `bun:"name,notnull"                    json:"name"`
	ParentID    string    `bun:"parent_id,nullzero"              json:"parent_id,omitempty"`
	CreatedAt   time.Time `bun:"created_at,default:now()"        json:"created_at"`
}

// LinkClick is stored in ClickHouse
type LinkClick struct {
	ID             string    `ch:"id"`
	LinkID         string    `ch:"link_id"`
	WorkspaceID    string    `ch:"workspace_id"`
	SessionHash    string    `ch:"session_hash"`
	IPHash         string    `ch:"ip_hash"`
	CountryCode    string    `ch:"country_code"`
	CountryName    string    `ch:"country_name"`
	City           string    `ch:"city"`
	Latitude       float64   `ch:"latitude"`
	Longitude      float64   `ch:"longitude"`
	DeviceType     string    `ch:"device_type"`
	Browser        string    `ch:"browser"`
	OS             string    `ch:"os"`
	ReferrerDomain string    `ch:"referrer"`
	UTMSource      string    `ch:"utm_source"`
	UTMMedium      string    `ch:"utm_medium"`
	UTMCampaign    string    `ch:"utm_campaign"`
	ClickedAt      time.Time `ch:"clicked_at"`
}

// Analytics holds link performance data
type Analytics struct {
	TotalClicks     int64           `json:"total_clicks"`
	UniqueClicks    int64           `json:"unique_clicks"`
	ClicksByDay     []DayCount      `json:"clicks_by_day"`
	ClicksByCountry []CountryCount  `json:"clicks_by_country"`
	ClicksByDevice  []DeviceCount   `json:"clicks_by_device"`
	ClicksByBrowser []BrowserCount  `json:"clicks_by_browser"`
	TopReferrers    []ReferrerCount `json:"top_referrers"`
}

type DayCount struct {
	Date   string `json:"date"`
	Clicks int64  `json:"clicks"`
}

type CountryCount struct {
	Country string `json:"country"`
	Code    string `json:"code"`
	Clicks  int64  `json:"clicks"`
}

type DeviceCount struct {
	Device string `json:"device"`
	Clicks int64  `json:"clicks"`
}

type BrowserCount struct {
	Browser string `json:"browser"`
	Clicks  int64  `json:"clicks"`
}

type ReferrerCount struct {
	Referrer string `json:"referrer"`
	Clicks   int64  `json:"clicks"`
}

// CreateLinkInput holds link creation data
type CreateLinkInput struct {
	OriginalURL string         `json:"original_url"  validate:"required,url"`
	CustomSlug  string         `json:"custom_slug"`
	Title       string         `json:"title"`
	Password    string         `json:"password"`
	ExpiresAt   *time.Time     `json:"expires_at"`
	ClickLimit  *int           `json:"click_limit"`
	Tags        []string       `json:"tags"`
	FolderID    string         `json:"folder_id"`
	UTMParams   map[string]any `json:"utm_params"`
}

// UpdateLinkInput holds link update data
type UpdateLinkInput struct {
	Title       string         `json:"title"`
	OriginalURL string         `json:"original_url"`
	Password    string         `json:"password"`
	ExpiresAt   *time.Time     `json:"expires_at"`
	ClickLimit  *int           `json:"click_limit"`
	Tags        []string       `json:"tags"`
	FolderID    string         `json:"folder_id"`
	IsActive    *bool          `json:"is_active"`
	UTMParams   map[string]any `json:"utm_params"`
}

// ListParams holds pagination parameters
type ListParams struct {
	Page     int    `query:"page"`
	PerPage  int    `query:"per_page"`
	Search   string `query:"search"`
	FolderID string `query:"folder_id"`
	Tag      string `query:"tag"`
	Sort     string `query:"sort"`
	Order    string `query:"order"`
}

// AnalyticsParams holds analytics query params
type AnalyticsParams struct {
	From string `query:"from"`
	To   string `query:"to"`
}

// ClickRequest holds click metadata
type ClickRequest struct {
	IP        string
	UserAgent string
	Referrer  string
}

// ListResponse wraps paginated results
type ListResponse struct {
	Items   []Link `json:"items"`
	Total   int64  `json:"total"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
}

type CreateFolderInput struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}
