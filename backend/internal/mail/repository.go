package mail

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository handles mail database operations
type Repository struct {
	db *bun.DB
	ch interface{}
}

// NewRepository creates a new mail repository
func NewRepository(db *bun.DB, ch interface{}) *Repository {
	return &Repository{db: db, ch: ch}
}

// Ping verifies DB connectivity and returns contact count
func (r *Repository) Ping(ctx context.Context) (int, error) {
	var count int
	err := r.db.NewRaw("SELECT count(*) FROM mail_contacts").Scan(ctx, &count)
	if err != nil {
		return 0, fmt.Errorf("mail.Repository.Ping: %w", err)
	}
	return count, nil
}

// ==================== CONTACTS ====================

// CreateContact inserts a contact
func (r *Repository) CreateContact(ctx context.Context, c *Contact) error {
	_, err := r.db.NewInsert().Model(c).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.CreateContact: %w", err)
	}
	return nil
}

// GetContact retrieves a contact by ID
func (r *Repository) GetContact(ctx context.Context, id string) (*Contact, error) {
	c := &Contact{}
	err := r.db.NewSelect().Model(c).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.GetContact: %w", err)
	}
	return c, nil
}

// GetContactByEmail finds a contact by email within a workspace
func (r *Repository) GetContactByEmail(ctx context.Context, workspaceID, email string) (*Contact, error) {
	c := &Contact{}
	err := r.db.NewSelect().Model(c).
		Where("workspace_id = ? AND email = ?", workspaceID, email).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.GetContactByEmail: %w", err)
	}
	return c, nil
}

// ListContacts returns contacts for a workspace with pagination
func (r *Repository) ListContacts(ctx context.Context, workspaceID string, params ListParams) ([]Contact, int64, error) {
	var contacts []Contact
	q := r.db.NewSelect().Model(&contacts).Where("workspace_id = ?", workspaceID)

	if params.Search != "" {
		q = q.Where("email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			"%"+params.Search+"%", "%"+params.Search+"%", "%"+params.Search+"%")
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("mail.Repository.ListContacts: %w", err)
	}

	if params.PerPage <= 0 {
		params.PerPage = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	err = q.OrderExpr("created_at DESC").Limit(params.PerPage).Offset((params.Page - 1) * params.PerPage).Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("mail.Repository.ListContacts: %w", err)
	}

	return contacts, int64(total), nil
}

// UpdateContact updates a contact
func (r *Repository) UpdateContact(ctx context.Context, c *Contact) error {
	c.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(c).Where("id = ?", c.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.UpdateContact: %w", err)
	}
	return nil
}

// UpsertContact creates or updates a contact by email
func (r *Repository) UpsertContact(ctx context.Context, c *Contact) error {
	_, err := r.db.NewRaw(`
		INSERT INTO mail_contacts (
			id, workspace_id, email, first_name, last_name, company, phone,
			tags, custom_fields, status, source, subscribed_at
		) VALUES (
			?, ?, ?, ?, DEFAULT, DEFAULT, DEFAULT,
			?, ?, ?, ?, NOW()
		)
		ON CONFLICT ON CONSTRAINT mail_contacts_workspace_id_email_key
		DO UPDATE SET
			first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), mail_contacts.first_name),
			tags = CASE
				WHEN jsonb_typeof(EXCLUDED.tags) = 'array' AND jsonb_array_length(EXCLUDED.tags) > 0
				THEN (mail_contacts.tags || EXCLUDED.tags)::jsonb
				ELSE mail_contacts.tags
			END,
			source = COALESCE(NULLIF(EXCLUDED.source, ''), mail_contacts.source),
			status = COALESCE(NULLIF(EXCLUDED.status, ''), mail_contacts.status),
			custom_fields = mail_contacts.custom_fields || EXCLUDED.custom_fields,
			updated_at = NOW()
		RETURNING id
	`,
		c.ID, c.WorkspaceID, c.Email, c.FirstName,
		c.Tags, c.CustomFields, c.Status, c.Source,
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.UpsertContact: %w", err)
	}
	return nil
}

// UnsubscribeContact marks a contact as unsubscribed
func (r *Repository) UnsubscribeContact(ctx context.Context, workspaceID, email string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*Contact)(nil)).
		Set("status = ?", ContactStatusUnsubscribed).
		Set("unsubscribed_at = ?", now).
		Set("updated_at = ?", now).
		Where("workspace_id = ? AND email = ?", workspaceID, email).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.UnsubscribeContact: %w", err)
	}
	return nil
}

// DeleteContact deletes a contact scoped to workspace
func (r *Repository) DeleteContact(ctx context.Context, id, workspaceID string) error {
	_, err := r.db.NewDelete().Model((*Contact)(nil)).Where("id = ? AND workspace_id = ?", id, workspaceID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.DeleteContact: %w", err)
	}
	return nil
}

// ==================== LISTS ====================

// CreateList creates a mail list
func (r *Repository) CreateList(ctx context.Context, l *MailList) error {
	_, err := r.db.NewInsert().Model(l).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.CreateList: %w", err)
	}
	return nil
}

// GetList retrieves a list by ID
func (r *Repository) GetList(ctx context.Context, id string) (*MailList, error) {
	l := &MailList{}
	err := r.db.NewSelect().Model(l).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.GetList: %w", err)
	}
	return l, nil
}

// ListLists returns all lists for a workspace
func (r *Repository) ListLists(ctx context.Context, workspaceID string) ([]MailList, error) {
	var lists []MailList
	err := r.db.NewSelect().Model(&lists).Where("workspace_id = ?", workspaceID).Order("created_at DESC").Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.ListLists: %w", err)
	}
	return lists, nil
}

// DeleteList deletes a list
func (r *Repository) DeleteList(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*MailList)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.DeleteList: %w", err)
	}
	return nil
}

// ==================== CAMPAIGNS ====================

// CreateCampaign creates a campaign
func (r *Repository) CreateCampaign(ctx context.Context, c *Campaign) error {
	_, err := r.db.NewInsert().Model(c).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.CreateCampaign: %w", err)
	}
	return nil
}

// GetCampaign retrieves a campaign by ID
func (r *Repository) GetCampaign(ctx context.Context, id string) (*Campaign, error) {
	c := &Campaign{}
	err := r.db.NewSelect().Model(c).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.GetCampaign: %w", err)
	}
	return c, nil
}

// GetCampaignByID retrieves a campaign scoped to workspace
func (r *Repository) GetCampaignByID(ctx context.Context, id, workspaceID string) (*Campaign, error) {
	c := &Campaign{}
	err := r.db.NewSelect().Model(c).Where("id = ? AND workspace_id = ?", id, workspaceID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.GetCampaignByID: %w", err)
	}
	return c, nil
}

// ListCampaigns returns paginated campaigns for a workspace
func (r *Repository) ListCampaigns(ctx context.Context, workspaceID string, params ListParams) ([]Campaign, int64, error) {
	var campaigns []Campaign
	q := r.db.NewSelect().Model(&campaigns).Where("workspace_id = ?", workspaceID)

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("mail.Repository.ListCampaigns: %w", err)
	}

	if params.PerPage <= 0 {
		params.PerPage = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	err = q.OrderExpr("created_at DESC").Limit(params.PerPage).Offset((params.Page - 1) * params.PerPage).Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("mail.Repository.ListCampaigns: %w", err)
	}

	return campaigns, int64(total), nil
}

// UpdateCampaign updates a campaign
func (r *Repository) UpdateCampaign(ctx context.Context, c *Campaign) error {
	c.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(c).Where("id = ?", c.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.UpdateCampaign: %w", err)
	}
	return nil
}

// DeleteCampaign deletes a campaign
func (r *Repository) DeleteCampaign(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*Campaign)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.DeleteCampaign: %w", err)
	}
	return nil
}

// ==================== STATS ====================

// CreateCampaignStats initializes a campaign stats row
func (r *Repository) CreateCampaignStats(ctx context.Context, s *CampaignStats) error {
	_, err := r.db.NewInsert().Model(s).Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.CreateCampaignStats: %w", err)
	}
	return nil
}

// GetCampaignStats retrieves campaign stats
func (r *Repository) GetCampaignStats(ctx context.Context, campaignID string) (*CampaignStats, error) {
	s := &CampaignStats{}
	err := r.db.NewSelect().Model(s).Where("campaign_id = ?", campaignID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("mail.Repository.GetCampaignStats: %w", err)
	}
	return s, nil
}

// UpsertCampaignStats creates or updates campaign stats
func (r *Repository) UpsertCampaignStats(ctx context.Context, s *CampaignStats) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.NewInsert().Model(s).On("CONFLICT (campaign_id) DO UPDATE").
		Set("sent = EXCLUDED.sent").
		Set("delivered = EXCLUDED.delivered").
		Set("opened = EXCLUDED.opened").
		Set("clicked = EXCLUDED.clicked").
		Set("bounced = EXCLUDED.bounced").
		Set("unsubscribed = EXCLUDED.unsubscribed").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mail.Repository.UpsertCampaignStats: %w", err)
	}
	return nil
}
