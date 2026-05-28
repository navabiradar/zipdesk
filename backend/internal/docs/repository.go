package docs

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository handles docs database operations
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new docs repository
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new document
func (r *Repository) Create(
	ctx context.Context,
	doc *Document,
) error {
	_, err := r.db.NewInsert().
		Model(doc).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"docs.Repository.Create: %w", err,
		)
	}
	return nil
}

// GetByID finds doc by ID and workspace
func (r *Repository) GetByID(
	ctx context.Context,
	id string,
	workspaceID string,
) (*Document, error) {
	doc := new(Document)
	err := r.db.NewSelect().
		Model(doc).
		Where("id = ? AND workspace_id = ?",
			id, workspaceID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"docs.Repository.GetByID: %w", err,
		)
	}
	return doc, nil
}

// GetBySlug finds public doc by slug
func (r *Repository) GetBySlug(
	ctx context.Context,
	slug string,
) (*Document, error) {
	doc := new(Document)
	err := r.db.NewSelect().
		Model(doc).
		Where(
			"slug = ? AND is_published = true",
			slug,
		).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"docs.Repository.GetBySlug: %w", err,
		)
	}
	return doc, nil
}

// Update modifies an existing document
func (r *Repository) Update(
	ctx context.Context,
	doc *Document,
) error {
	doc.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(doc).
		Where("id = ?", doc.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"docs.Repository.Update: %w", err,
		)
	}
	return nil
}

// Delete removes a document
func (r *Repository) Delete(
	ctx context.Context,
	id string,
	workspaceID string,
) error {
	_, err := r.db.NewDelete().
		TableExpr("documents").
		Where("id = ? AND workspace_id = ?",
			id, workspaceID).
		Exec(ctx)
	return err
}

// List returns paginated documents
func (r *Repository) List(
	ctx context.Context,
	workspaceID string,
	p ListParams,
) ([]Document, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	q := r.db.NewSelect().
		Model((*Document)(nil)).
		Where("workspace_id = ?", workspaceID)

	if p.Search != "" {
		q = q.Where(
			"title ILIKE ?",
			"%"+p.Search+"%",
		)
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	var items []Document
	err = q.
		OrderExpr("created_at DESC").
		Limit(p.PerPage).
		Offset((p.Page - 1) * p.PerPage).
		Scan(ctx, &items)
	if err != nil {
		return nil, 0, err
	}

	return items, int64(total), nil
}

// SlugExists checks if slug is taken
func (r *Repository) SlugExists(
	ctx context.Context,
	slug string,
) (bool, error) {
	count, err := r.db.NewSelect().
		TableExpr("documents").
		Where("slug = ?", slug).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
