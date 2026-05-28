package links

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/zipdesk/backend/pkg/clickhouse"
)

// Repository handles links database operations
type Repository struct {
	db *bun.DB
	ch *clickhouse.Client
}

// NewRepository creates a new links repository
func NewRepository(db *bun.DB, ch *clickhouse.Client) *Repository {
	return &Repository{db: db, ch: ch}
}

// Create inserts a new link
func (r *Repository) Create(
	ctx context.Context,
	link *Link,
) error {
	_, err := r.db.NewInsert().
		Model(link).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("links.Repository.Create: %w", err)
	}
	return nil
}

// GetByID finds a link by ID and workspace
func (r *Repository) GetByID(
	ctx context.Context,
	id string,
	workspaceID string,
) (*Link, error) {
	link := new(Link)
	err := r.db.NewSelect().
		Model(link).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf(
			"links.Repository.GetByID: %w", err,
		)
	}
	return link, nil
}

// GetBySlug finds a link by short code or custom slug
func (r *Repository) GetBySlug(
	ctx context.Context,
	slug string,
) (*Link, error) {
	link := new(Link)
	err := r.db.NewSelect().
		Model(link).
		Where("short_code = ? OR custom_slug = ?", slug, slug).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf(
			"links.Repository.GetBySlug: %w", err,
		)
	}
	return link, nil
}

// ShortCodeExists checks whether a short code or custom slug is taken.
func (r *Repository) ShortCodeExists(ctx context.Context, slug string) (bool, error) {
	return slugExists(ctx, r.db, slug)
}

// Update modifies an existing link
func (r *Repository) Update(
	ctx context.Context,
	link *Link,
) error {
	link.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(link).
		Where("id = ?", link.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"links.Repository.Update: %w", err,
		)
	}
	return nil
}

// Delete removes a link
func (r *Repository) Delete(
	ctx context.Context,
	id string,
	workspaceID string,
) error {
	_, err := r.db.NewDelete().
		TableExpr("links").
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"links.Repository.Delete: %w", err,
		)
	}
	return nil
}

// List returns paginated links for a workspace
func (r *Repository) List(
	ctx context.Context,
	workspaceID string,
	p ListParams,
) ([]Link, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	q := r.db.NewSelect().
		Model((*Link)(nil)).
		Where("workspace_id = ?", workspaceID)

	if p.Search != "" {
		q = q.Where(
			"title ILIKE ? OR original_url ILIKE ?",
			"%"+p.Search+"%",
			"%"+p.Search+"%",
		)
	}

	if p.FolderID != "" {
		q = q.Where("folder_id = ?", p.FolderID)
	}

	// Count total
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"links.Repository.List count: %w", err,
		)
	}

	// Apply sorting
	sortCol := "created_at"
	if p.Sort == "clicks" {
		sortCol = "total_clicks"
	}
	sortDir := "DESC"
	if strings.ToUpper(p.Order) == "ASC" {
		sortDir = "ASC"
	}

	var items []Link
	err = q.
		OrderExpr(fmt.Sprintf("%s %s", sortCol, sortDir)).
		Limit(p.PerPage).
		Offset((p.Page-1)*p.PerPage).
		Scan(ctx, &items)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"links.Repository.List scan: %w", err,
		)
	}

	return items, int64(total), nil
}

// IncrementClicks atomically increments click count
func (r *Repository) IncrementClicks(
	ctx context.Context,
	id string,
	unique bool,
) error {
	q := r.db.NewUpdate().
		TableExpr("links").
		Set("total_clicks = total_clicks + 1").
		Where("id = ?", id)

	if unique {
		q = q.Set("unique_clicks = unique_clicks + 1")
	}

	_, err := q.Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"links.Repository.IncrementClicks: %w", err,
		)
	}
	return nil
}

// RecordClick writes a click event to ClickHouse
func (r *Repository) RecordClick(
	ctx context.Context,
	click LinkClick,
) error {
	if r.ch == nil {
		return nil // ClickHouse not available
	}

	query := `INSERT INTO link_clicks (
        id, link_id, workspace_id,
        session_hash, ip_hash,
        country_code, country_name, city,
        latitude, longitude,
        device_type, browser, os,
        referrer_domain,
        utm_source, utm_medium, utm_campaign,
        clicked_at
    ) VALUES`

	return r.ch.Insert(ctx, query, []interface{}{&click})
}

// GetAnalytics queries ClickHouse for analytics
func (r *Repository) GetAnalytics(
	ctx context.Context,
	linkID string,
	from time.Time,
	to time.Time,
) (*Analytics, error) {
	analytics := &Analytics{
		ClicksByDay:     []DayCount{},
		ClicksByCountry: []CountryCount{},
		ClicksByDevice:  []DeviceCount{},
		ClicksByBrowser: []BrowserCount{},
		TopReferrers:    []ReferrerCount{},
	}

	if r.ch == nil {
		return analytics, nil
	}

	// Total clicks
	rows, err := r.ch.Query(ctx,
		`SELECT count() as total,
                countDistinct(session_hash) as unique_count
         FROM link_clicks
         WHERE link_id = ? AND clicked_at BETWEEN ? AND ?`,
		linkID, from, to,
	)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			_ = rows.Scan(
				&analytics.TotalClicks,
				&analytics.UniqueClicks,
			)
		}
	}

	// Clicks by day
	dayRows, err := r.ch.Query(ctx,
		`SELECT toString(toDate(clicked_at)) as date,
                count() as clicks
         FROM link_clicks
         WHERE link_id = ? AND clicked_at BETWEEN ? AND ?
         GROUP BY date ORDER BY date ASC`,
		linkID, from, to,
	)
	if err == nil {
		defer dayRows.Close()
		for dayRows.Next() {
			var d DayCount
			if err := dayRows.Scan(&d.Date, &d.Clicks); err == nil {
				analytics.ClicksByDay = append(
					analytics.ClicksByDay, d,
				)
			}
		}
	}

	// Clicks by country
	countryRows, err := r.ch.Query(ctx,
		`SELECT country_name, country_code, count() as clicks
         FROM link_clicks
         WHERE link_id = ? AND clicked_at BETWEEN ? AND ?
         GROUP BY country_name, country_code
         ORDER BY clicks DESC LIMIT 10`,
		linkID, from, to,
	)
	if err == nil {
		defer countryRows.Close()
		for countryRows.Next() {
			var c CountryCount
			if err := countryRows.Scan(
				&c.Country, &c.Code, &c.Clicks,
			); err == nil {
				analytics.ClicksByCountry = append(
					analytics.ClicksByCountry, c,
				)
			}
		}
	}

	// Clicks by device
	deviceRows, err := r.ch.Query(ctx,
		`SELECT device_type, count() as clicks
         FROM link_clicks
         WHERE link_id = ? AND clicked_at BETWEEN ? AND ?
         GROUP BY device_type ORDER BY clicks DESC`,
		linkID, from, to,
	)
	if err == nil {
		defer deviceRows.Close()
		for deviceRows.Next() {
			var d DeviceCount
			if err := deviceRows.Scan(
				&d.Device, &d.Clicks,
			); err == nil {
				analytics.ClicksByDevice = append(
					analytics.ClicksByDevice, d,
				)
			}
		}
	}

	return analytics, nil
}

// CreateFolder inserts a new folder
func (r *Repository) CreateFolder(ctx context.Context, folder *LinkFolder) error {
	_, err := r.db.NewInsert().
		Model(folder).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("links.Repository.CreateFolder: %w", err)
	}
	return nil
}

// ListFolders returns folders for a workspace
func (r *Repository) ListFolders(ctx context.Context, workspaceID string) ([]LinkFolder, error) {
	var folders []LinkFolder
	err := r.db.NewSelect().
		Model(&folders).
		Where("workspace_id = ?", workspaceID).
		Order("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("links.Repository.ListFolders: %w", err)
	}
	return folders, nil
}

// hashIP anonymizes an IP address
func hashIP(ip string) string {
	h := sha256.New()
	h.Write([]byte(ip + "zipdesk-salt"))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// hashSession creates a session identifier
func hashSession(ip, ua string) string {
	h := sha256.New()
	h.Write([]byte(ip + ua + "zipdesk-session"))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// newClickID generates a UUID for click
func newClickID() string {
	return uuid.New().String()
}
