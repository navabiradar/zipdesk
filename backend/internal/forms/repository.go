package forms

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository handles forms database operations
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new forms repository
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// CreateForm inserts a new form
func (r *Repository) CreateForm(
	ctx context.Context,
	form *Form,
) error {
	_, err := r.db.NewInsert().
		Model(form).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"forms.Repository.CreateForm: %w", err,
		)
	}
	return nil
}

// GetFormByID finds form by ID and workspace
func (r *Repository) GetFormByID(
	ctx context.Context,
	id string,
	workspaceID string,
) (*Form, error) {
	form := new(Form)
	err := r.db.NewSelect().
		Model(form).
		Where("form.id = ? AND form.workspace_id = ?",
			id, workspaceID).
		Relation("Fields", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("field_order ASC")
		}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"forms.Repository.GetFormByID: %w", err,
		)
	}
	return form, nil
}

// GetFormBySlug finds public form by slug
func (r *Repository) GetFormBySlug(
	ctx context.Context,
	slug string,
) (*Form, error) {
	form := new(Form)
	err := r.db.NewSelect().
		Model(form).
		Where("form.slug = ? AND form.is_published = true", slug).
		Relation("Fields", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("field_order ASC")
		}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"forms.Repository.GetFormBySlug: %w", err,
		)
	}
	return form, nil
}

// UpdateForm updates form metadata
func (r *Repository) UpdateForm(
	ctx context.Context,
	form *Form,
) error {
	form.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(form).
		Where("id = ?", form.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"forms.Repository.UpdateForm: %w", err,
		)
	}
	return nil
}

// DeleteForm removes a form and its fields
func (r *Repository) DeleteForm(
	ctx context.Context,
	id string,
	workspaceID string,
) error {
	_, err := r.db.NewDelete().
		TableExpr("forms").
		Where("id = ? AND workspace_id = ?",
			id, workspaceID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"forms.Repository.DeleteForm: %w", err,
		)
	}
	return nil
}

// ListForms returns paginated forms
func (r *Repository) ListForms(
	ctx context.Context,
	workspaceID string,
	p ListParams,
) ([]Form, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	q := r.db.NewSelect().
		Model((*Form)(nil)).
		Where("workspace_id = ?", workspaceID)

	if p.Search != "" {
		q = q.Where(
			"title ILIKE ?",
			"%"+p.Search+"%",
		)
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"forms.Repository.ListForms count: %w", err,
		)
	}

	var items []Form
	err = q.
		OrderExpr("created_at DESC").
		Limit(p.PerPage).
		Offset((p.Page-1)*p.PerPage).
		Scan(ctx, &items)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"forms.Repository.ListForms scan: %w", err,
		)
	}

	return items, int64(total), nil
}

// ReplaceFields deletes and re-inserts all fields
func (r *Repository) ReplaceFields(
	ctx context.Context,
	formID string,
	fields []FormField,
) error {
	return r.db.RunInTx(ctx, nil, func(
		ctx context.Context, tx bun.Tx,
	) error {
		// Delete existing fields
		_, err := tx.NewDelete().
			TableExpr("form_fields").
			Where("form_id = ?", formID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf(
				"forms.Repository.ReplaceFields delete: %w",
				err,
			)
		}

		if len(fields) == 0 {
			return nil
		}

		// Insert new fields
		for i := range fields {
			fields[i].FormID = formID
			fields[i].FieldOrder = i
			if fields[i].Options == nil {
				fields[i].Options = []FieldOption{}
			}
			if fields[i].Logic == nil {
				fields[i].Logic = []FieldLogic{}
			}
		}

		_, err = tx.NewInsert().
			Model(&fields).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf(
				"forms.Repository.ReplaceFields insert: %w",
				err,
			)
		}

		return nil
	})
}

// CreateResponse saves a form submission
func (r *Repository) CreateResponse(
	ctx context.Context,
	response *FormResponse,
) error {
	_, err := r.db.NewInsert().
		Model(response).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"forms.Repository.CreateResponse: %w", err,
		)
	}
	return nil
}

// ListResponses returns paginated responses
func (r *Repository) ListResponses(
	ctx context.Context,
	formID string,
	workspaceID string,
	p ListParams,
) ([]FormResponse, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	q := r.db.NewSelect().
		Model((*FormResponse)(nil)).
		Where("form_id = ? AND workspace_id = ?",
			formID, workspaceID)

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"forms.Repository.ListResponses count: %w", err,
		)
	}

	var items []FormResponse
	err = q.
		OrderExpr("submitted_at DESC").
		Limit(p.PerPage).
		Offset((p.Page-1)*p.PerPage).
		Scan(ctx, &items)
	if err != nil {
		return nil, 0, fmt.Errorf(
			"forms.Repository.ListResponses scan: %w", err,
		)
	}

	return items, int64(total), nil
}

// GetResponseCount returns total response count
func (r *Repository) GetResponseCount(
	ctx context.Context,
	formID string,
) (int64, error) {
	count, err := r.db.NewSelect().
		TableExpr("form_responses").
		Where("form_id = ?", formID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"forms.Repository.GetResponseCount: %w", err,
		)
	}
	return int64(count), nil
}

// RecordView saves a form view
func (r *Repository) RecordView(
	ctx context.Context,
	view *FormView,
) error {
	_, err := r.db.NewInsert().
		Model(view).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"forms.Repository.RecordView: %w", err,
		)
	}
	return nil
}

// GetViewCount returns total view count
func (r *Repository) GetViewCount(
	ctx context.Context,
	formID string,
) (int64, error) {
	count, err := r.db.NewSelect().
		TableExpr("form_views").
		Where("form_id = ?", formID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf(
			"forms.Repository.GetViewCount: %w", err,
		)
	}
	return int64(count), nil
}

// SlugExists checks if slug is taken
func (r *Repository) SlugExists(
	ctx context.Context,
	slug string,
) (bool, error) {
	count, err := r.db.NewSelect().
		TableExpr("forms").
		Where("slug = ?", slug).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf(
			"forms.Repository.SlugExists: %w", err,
		)
	}
	return count > 0, nil
}

// ExportResponses returns all responses as CSV
func (r *Repository) ExportResponses(
	ctx context.Context,
	formID string,
	workspaceID string,
) ([]FormResponse, error) {
	var responses []FormResponse
	err := r.db.NewSelect().
		Model(&responses).
		Where("form_id = ? AND workspace_id = ?",
			formID, workspaceID).
		OrderExpr("submitted_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"forms.Repository.ExportResponses: %w", err,
		)
	}
	return responses, nil
}
