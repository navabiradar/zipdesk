package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository handles auth database operations
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser inserts a new user
func (r *Repository) CreateUser(
	ctx context.Context,
	user *User,
) error {
	_, err := r.db.NewInsert().
		Model(user).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("auth.Repository.CreateUser: %w", err)
	}
	return nil
}

// GetUserByEmail finds a user by email
func (r *Repository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	user := new(User)
	err := r.db.NewSelect().
		Model(user).
		Where("email = ?", email).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.GetUserByEmail: %w", err)
	}
	return user, nil
}

// GetUserByID finds a user by ID
func (r *Repository) GetUserByID(
	ctx context.Context,
	id string,
) (*User, error) {
	user := new(User)
	err := r.db.NewSelect().
		Model(user).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.GetUserByID: %w", err)
	}
	return user, nil
}

// UpdateUser updates user fields
func (r *Repository) UpdateUser(
	ctx context.Context,
	user *User,
) error {
	user.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(user).
		Where("id = ?", user.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("auth.Repository.UpdateUser: %w", err)
	}
	return nil
}

// CreateWorkspace inserts a new workspace
func (r *Repository) CreateWorkspace(
	ctx context.Context,
	workspace *Workspace,
) error {
	_, err := r.db.NewInsert().
		Model(workspace).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("auth.Repository.CreateWorkspace: %w", err)
	}
	return nil
}

// GetWorkspaceByID finds a workspace by ID
func (r *Repository) GetWorkspaceByID(
	ctx context.Context,
	id string,
) (*Workspace, error) {
	workspace := new(Workspace)
	err := r.db.NewSelect().
		Model(workspace).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.GetWorkspaceByID: %w", err)
	}
	return workspace, nil
}

// GetWorkspaceBySlug finds a workspace by slug
func (r *Repository) GetWorkspaceBySlug(
	ctx context.Context,
	slug string,
) (*Workspace, error) {
	workspace := new(Workspace)
	err := r.db.NewSelect().
		Model(workspace).
		Where("slug = ?", slug).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.Repository.GetWorkspaceBySlug: %w", err)
	}
	return workspace, nil
}

// CreateWorkspaceMember adds a user to workspace
func (r *Repository) CreateWorkspaceMember(
	ctx context.Context,
	member *WorkspaceMember,
) error {
	_, err := r.db.NewInsert().
		Model(member).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("auth.Repository.CreateWorkspaceMember: %w", err)
	}
	return nil
}

// GetWorkspaceMember gets a member record
func (r *Repository) GetWorkspaceMember(
	ctx context.Context,
	workspaceID string,
	userID string,
) (*WorkspaceMember, error) {
	member := new(WorkspaceMember)
	err := r.db.NewSelect().
		Model(member).
		Where("workspace_id = ? AND user_id = ?",
			workspaceID, userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"auth.Repository.GetWorkspaceMember: %w", err,
		)
	}
	return member, nil
}

// GetUserWorkspaces returns all workspaces for a user
func (r *Repository) GetUserWorkspaces(
	ctx context.Context,
	userID string,
) ([]Workspace, error) {
	var workspaces []Workspace
	err := r.db.NewSelect().
		TableExpr("workspaces w").
		ColumnExpr("w.*").
		Join("JOIN workspace_members wm ON wm.workspace_id = w.id").
		Where("wm.user_id = ?", userID).
		OrderExpr("w.created_at ASC").
		Scan(ctx, &workspaces)
	if err != nil {
		return nil, fmt.Errorf(
			"auth.Repository.GetUserWorkspaces: %w", err,
		)
	}
	return workspaces, nil
}

// CreateEmailVerification stores a verification token
func (r *Repository) CreateEmailVerification(
	ctx context.Context,
	ev *EmailVerification,
) error {
	_, err := r.db.NewInsert().
		Model(ev).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"auth.Repository.CreateEmailVerification: %w", err,
		)
	}
	return nil
}

// GetEmailVerification finds a verification by token
func (r *Repository) GetEmailVerification(
	ctx context.Context,
	token string,
) (*EmailVerification, error) {
	ev := new(EmailVerification)
	err := r.db.NewSelect().
		Model(ev).
		Where("token = ? AND used_at IS NULL", token).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"auth.Repository.GetEmailVerification: %w", err,
		)
	}
	return ev, nil
}

// MarkEmailVerificationUsed marks token as used
func (r *Repository) MarkEmailVerificationUsed(
	ctx context.Context,
	id string,
) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		TableExpr("email_verifications").
		Set("used_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"auth.Repository.MarkEmailVerificationUsed: %w", err,
		)
	}
	return nil
}

// CreatePasswordReset stores a reset token
func (r *Repository) CreatePasswordReset(
	ctx context.Context,
	pr *PasswordReset,
) error {
	_, err := r.db.NewInsert().
		Model(pr).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"auth.Repository.CreatePasswordReset: %w", err,
		)
	}
	return nil
}

// GetPasswordReset finds a reset by token
func (r *Repository) GetPasswordReset(
	ctx context.Context,
	token string,
) (*PasswordReset, error) {
	pr := new(PasswordReset)
	err := r.db.NewSelect().
		Model(pr).
		Where("token = ? AND used_at IS NULL", token).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"auth.Repository.GetPasswordReset: %w", err,
		)
	}
	return pr, nil
}

// MarkPasswordResetUsed marks token as used
func (r *Repository) MarkPasswordResetUsed(
	ctx context.Context,
	id string,
) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		TableExpr("password_resets").
		Set("used_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"auth.Repository.MarkPasswordResetUsed: %w", err,
		)
	}
	return nil
}
