package auth

import (
	"time"

	"github.com/uptrace/bun"
)

// User represents a ZipDesk user
type User struct {
	bun.BaseModel `bun:"table:users"`

	ID           string    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	Email        string    `bun:"email,notnull"                   json:"email"`
	Name         string    `bun:"name,notnull,default:''"         json:"name"`
	PasswordHash string    `bun:"password_hash"                   json:"-"`
	GoogleID     string    `bun:"google_id,nullzero"              json:"-"`
	AvatarURL    string    `bun:"avatar_url,nullzero"             json:"avatar_url"`
	IsVerified   bool      `bun:"is_verified,default:false"       json:"is_verified"`
	CreatedAt    time.Time `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt    time.Time `bun:"updated_at,default:now()"        json:"updated_at"`
}

// Workspace represents a ZipDesk workspace
type Workspace struct {
	bun.BaseModel `bun:"table:workspaces"`

	ID        string    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	Name      string    `bun:"name,notnull"                    json:"name"`
	Slug      string    `bun:"slug,notnull"                    json:"slug"`
	LogoURL   string    `bun:"logo_url"                        json:"logo_url"`
	OwnerID   string    `bun:"owner_id,notnull"                json:"owner_id"`
	Plan      string    `bun:"plan,default:'free'"             json:"plan"`
	CreatedAt time.Time `bun:"created_at,default:now()"        json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,default:now()"        json:"updated_at"`
}

// WorkspaceMember links users to workspaces
type WorkspaceMember struct {
	bun.BaseModel `bun:"table:workspace_members"`

	ID          string    `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	WorkspaceID string    `bun:"workspace_id,notnull"            json:"workspace_id"`
	UserID      string    `bun:"user_id,notnull"                 json:"user_id"`
	Role        string    `bun:"role,default:'member'"           json:"role"`
	JoinedAt    time.Time `bun:"joined_at,default:now()"         json:"joined_at"`
}

// EmailVerification holds email verification tokens
type EmailVerification struct {
	bun.BaseModel `bun:"table:email_verifications"`

	ID        string     `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	UserID    string     `bun:"user_id,notnull"                 json:"user_id"`
	Token     string     `bun:"token,notnull"                   json:"token"`
	ExpiresAt time.Time  `bun:"expires_at,notnull"              json:"expires_at"`
	UsedAt    *time.Time `bun:"used_at"                         json:"used_at"`
	CreatedAt time.Time  `bun:"created_at,default:now()"        json:"created_at"`
}

// PasswordReset holds password reset tokens
type PasswordReset struct {
	bun.BaseModel `bun:"table:password_resets"`

	ID        string     `bun:"id,pk,default:gen_random_uuid()" json:"id"`
	UserID    string     `bun:"user_id,notnull"                 json:"user_id"`
	Token     string     `bun:"token,notnull"                   json:"token"`
	ExpiresAt time.Time  `bun:"expires_at,notnull"              json:"expires_at"`
	UsedAt    *time.Time `bun:"used_at"                         json:"used_at"`
	CreatedAt time.Time  `bun:"created_at,default:now()"        json:"created_at"`
}

// RegisterInput holds registration data
type RegisterInput struct {
	Name          string `json:"name"            validate:"required,min=2,max=100"`
	Email         string `json:"email"           validate:"required,email"`
	Password      string `json:"password"        validate:"required,min=8,max=100"`
	WorkspaceName string `json:"workspace_name"  validate:"required,min=2,max=100"`
}

// LoginInput holds login data
type LoginInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ForgotPasswordInput holds forgot password data
type ForgotPasswordInput struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordInput holds reset password data
type ResetPasswordInput struct {
	Token    string `json:"token"    validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

// AuthResponse is returned on successful auth
type AuthResponse struct {
	User         *UserResponse      `json:"user"`
	Workspace    *WorkspaceResponse `json:"workspace"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token,omitempty"`
}

// UserResponse is the public user representation
type UserResponse struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	AvatarURL  string    `json:"avatar_url"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
}

// WorkspaceResponse is the public workspace representation
type WorkspaceResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	LogoURL string `json:"logo_url"`
	Plan    string `json:"plan"`
	Role    string `json:"role"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		AvatarURL:  u.AvatarURL,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt,
	}
}
