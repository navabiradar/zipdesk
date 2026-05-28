package crm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// Repository handles CRM database operations
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new CRM repository
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// CreateContact inserts a new CRM contact
func (r *Repository) CreateContact(
	ctx context.Context,
	contact *CRMContact,
) error {
	_, err := r.db.NewInsert().Model(contact).Exec(ctx)
	if err != nil {
		return fmt.Errorf("crm.Repository.CreateContact: %w", err)
	}
	return nil
}

// GetContactByEmail finds contact by email
func (r *Repository) GetContactByEmail(
	ctx context.Context,
	workspaceID string,
	email string,
) (*CRMContact, error) {
	contact := new(CRMContact)
	err := r.db.NewSelect().Model(contact).
		Where("workspace_id = ? AND email = ?", workspaceID, strings.ToLower(email)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

// ListContacts returns paginated CRM contacts
func (r *Repository) ListContacts(
	ctx context.Context,
	workspaceID string,
	p ListParams,
) ([]CRMContact, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	q := r.db.NewSelect().Model((*CRMContact)(nil)).Where("workspace_id = ?", workspaceID)

	if p.Search != "" {
		q = q.Where("email ILIKE ? OR first_name ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("crm.Repository.ListContacts: %w", err)
	}

	var items []CRMContact
	err = q.OrderExpr("created_at DESC").Limit(p.PerPage).Offset((p.Page - 1) * p.PerPage).Scan(ctx, &items)
	if err != nil {
		return nil, 0, err
	}

	return items, int64(total), nil
}

// UpdateContact updates CRM contact
func (r *Repository) UpdateContact(
	ctx context.Context,
	contact *CRMContact,
) error {
	contact.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().Model(contact).Where("id = ?", contact.ID).Exec(ctx)
	return err
}

// CreateDeal inserts a new deal
func (r *Repository) CreateDeal(
	ctx context.Context,
	deal *CRMDeal,
) error {
	_, err := r.db.NewInsert().Model(deal).Exec(ctx)
	if err != nil {
		return fmt.Errorf("crm.Repository.CreateDeal: %w", err)
	}
	return nil
}

// ListDeals returns paginated deals
func (r *Repository) ListDeals(
	ctx context.Context,
	workspaceID string,
	p ListParams,
) ([]CRMDeal, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	q := r.db.NewSelect().Model((*CRMDeal)(nil)).Where("workspace_id = ?", workspaceID)

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	var items []CRMDeal
	err = q.OrderExpr("created_at DESC").Limit(p.PerPage).Offset((p.Page - 1) * p.PerPage).Scan(ctx, &items)
	if err != nil {
		return nil, 0, err
	}

	return items, int64(total), nil
}
