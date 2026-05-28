package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/zipdesk/backend/pkg/cache"
)

const (
	// Token validity periods
	EmailVerificationTokenTTL = 24 * time.Hour
	PasswordResetTokenTTL     = 1 * time.Hour
	AccessTokenTTL            = 7 * 24 * time.Hour // 7 days
	RefreshTokenTTL           = 30 * 24 * time.Hour

	// Cache keys
	accessTokenPrefix  = "auth:token:"
	refreshTokenPrefix = "auth:refresh:"
	verificationPrefix = "auth:verify:"
)

// TokenData holds cached token info
type TokenData struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
}

// Service handles authentication logic
type Service struct {
	repo  *Repository
	redis *cache.Client
	log   *zap.Logger
}

// NewService creates a new auth service
func NewService(repo *Repository, redis *cache.Client, log *zap.Logger) *Service {
	return &Service{
		repo:  repo,
		redis: redis,
		log:   log,
	}
}

// Register creates a new user and workspace
func (s *Service) Register(ctx context.Context, input *RegisterInput) (*AuthResponse, error) {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: hash password: %w", err)
	}

	// Create user
	user := &User{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: string(hash),
		IsVerified:   false,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Insert user
	if _, err := s.repo.db.NewInsert().Model(user).Exec(ctx); err != nil {
		s.log.Error("failed to create user", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: create user: %w", err)
	}

	// Create workspace
	workspace := &Workspace{
		Name:      input.WorkspaceName,
		Slug:      generateSlug(input.WorkspaceName),
		OwnerID:   user.ID,
		Plan:      "free",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if _, err := s.repo.db.NewInsert().Model(workspace).Exec(ctx); err != nil {
		s.log.Error("failed to create workspace", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: create workspace: %w", err)
	}

	// Add user to workspace
	member := &WorkspaceMember{
		WorkspaceID: workspace.ID,
		UserID:      user.ID,
		Role:        "owner",
		JoinedAt:    time.Now().UTC(),
	}

	if _, err := s.repo.db.NewInsert().Model(member).Exec(ctx); err != nil {
		s.log.Error("failed to add user to workspace", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: add member: %w", err)
	}

	// Generate access token and refresh token
	accessToken, err := generateAccessToken(user.ID, workspace.ID, "owner", workspace.Plan)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: generate token: %w", err)
	}
	refreshToken := generateToken()
	if err := s.cacheAccessToken(ctx, user.ID, workspace.ID, accessToken); err != nil {
		s.log.Error("failed to cache access token", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: cache token: %w", err)
	}
	if err := s.cacheRefreshToken(ctx, user.ID, workspace.ID, refreshToken); err != nil {
		s.log.Error("failed to cache refresh token", zap.Error(err))
		return nil, fmt.Errorf("auth.Register: cache refresh token: %w", err)
	}

	// Send verification email (async)
	go s.sendVerificationEmail(ctx, user)

	return &AuthResponse{
		User: user.ToResponse(),
		Workspace: &WorkspaceResponse{
			ID:      workspace.ID,
			Name:    workspace.Name,
			Slug:    workspace.Slug,
			LogoURL: workspace.LogoURL,
			Plan:    workspace.Plan,
			Role:    "owner",
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Login authenticates a user
func (s *Service) Login(ctx context.Context, input *LoginInput) (*AuthResponse, error) {
	user := &User{}
	err := s.repo.db.NewSelect().Model(user).
		Where("email = ?", input.Email).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			s.log.Warn("login attempt with non-existent user", zap.String("email", input.Email))
			return nil, fmt.Errorf("invalid credentials")
		}
		s.log.Error("failed to fetch user", zap.Error(err))
		return nil, fmt.Errorf("auth.Login: fetch user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		s.log.Warn("failed password check", zap.String("user_id", user.ID))
		return nil, fmt.Errorf("invalid credentials")
	}

	// Fetch workspace member
	member := &WorkspaceMember{}
	err = s.repo.db.NewSelect().Model(member).
		Where("user_id = ?", user.ID).
		Order("joined_at ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		s.log.Error("failed to fetch workspace member", zap.Error(err))
		return nil, fmt.Errorf("auth.Login: fetch member: %w", err)
	}

	// Fetch workspace
	workspace := &Workspace{}
	err = s.repo.db.NewSelect().Model(workspace).
		Where("id = ?", member.WorkspaceID).
		Scan(ctx)
	if err != nil {
		s.log.Error("failed to fetch workspace", zap.Error(err))
		return nil, fmt.Errorf("auth.Login: fetch workspace: %w", err)
	}

	// Generate access token and refresh token
	accessToken, err := generateAccessToken(user.ID, workspace.ID, member.Role, workspace.Plan)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		return nil, fmt.Errorf("auth.Login: generate token: %w", err)
	}
	refreshToken := generateToken()
	if err := s.cacheAccessToken(ctx, user.ID, workspace.ID, accessToken); err != nil {
		s.log.Error("failed to cache access token", zap.Error(err))
		return nil, fmt.Errorf("auth.Login: cache token: %w", err)
	}
	if err := s.cacheRefreshToken(ctx, user.ID, workspace.ID, refreshToken); err != nil {
		s.log.Error("failed to cache refresh token", zap.Error(err))
		return nil, fmt.Errorf("auth.Login: cache refresh token: %w", err)
	}

	return &AuthResponse{
		User: user.ToResponse(),
		Workspace: &WorkspaceResponse{
			ID:      workspace.ID,
			Name:    workspace.Name,
			Slug:    workspace.Slug,
			LogoURL: workspace.LogoURL,
			Plan:    workspace.Plan,
			Role:    member.Role,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// VerifyEmail marks an email as verified
func (s *Service) VerifyEmail(ctx context.Context, token string) (*User, error) {
	emailVerif := &EmailVerification{}
	err := s.repo.db.NewSelect().Model(emailVerif).
		Where("token = ? AND expires_at > ?", token, time.Now().UTC()).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired token")
		}
		return nil, fmt.Errorf("auth.VerifyEmail: fetch token: %w", err)
	}

	if emailVerif.UsedAt != nil {
		return nil, fmt.Errorf("token already used")
	}

	// Mark token as used
	now := time.Now().UTC()
	if _, err := s.repo.db.NewUpdate().Model(emailVerif).
		Set("used_at = ?", now).
		Where("id = ?", emailVerif.ID).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("auth.VerifyEmail: update token: %w", err)
	}

	// Mark user as verified
	if _, err := s.repo.db.NewUpdate().Model(&User{}).
		Set("is_verified = true").
		Where("id = ?", emailVerif.UserID).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("auth.VerifyEmail: update user: %w", err)
	}

	// Fetch updated user
	user := &User{}
	if err := s.repo.db.NewSelect().Model(user).
		Where("id = ?", emailVerif.UserID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("auth.VerifyEmail: fetch user: %w", err)
	}

	return user, nil
}

// ForgotPassword sends a password reset email
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user := &User{}
	err := s.repo.db.NewSelect().Model(user).
		Where("email = ?", email).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if user exists
			return nil
		}
		return fmt.Errorf("auth.ForgotPassword: fetch user: %w", err)
	}

	token := generateToken()
	reset := &PasswordReset{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(PasswordResetTokenTTL),
		CreatedAt: time.Now().UTC(),
	}

	if _, err := s.repo.db.NewInsert().Model(reset).Exec(ctx); err != nil {
		return fmt.Errorf("auth.ForgotPassword: create reset token: %w", err)
	}

	// Send reset email (async)
	go s.sendPasswordResetEmail(ctx, user, token)

	return nil
}

// ResetPassword resets a user's password
func (s *Service) ResetPassword(ctx context.Context, input *ResetPasswordInput) error {
	reset := &PasswordReset{}
	err := s.repo.db.NewSelect().Model(reset).
		Where("token = ? AND expires_at > ?", input.Token, time.Now().UTC()).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid or expired token")
		}
		return fmt.Errorf("auth.ResetPassword: fetch token: %w", err)
	}

	if reset.UsedAt != nil {
		return fmt.Errorf("token already used")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth.ResetPassword: hash password: %w", err)
	}

	// Update password
	if _, err := s.repo.db.NewUpdate().Model(&User{}).
		Set("password_hash = ?", string(hash)).
		Where("id = ?", reset.UserID).
		Exec(ctx); err != nil {
		return fmt.Errorf("auth.ResetPassword: update password: %w", err)
	}

	// Mark token as used
	now := time.Now().UTC()
	if _, err := s.repo.db.NewUpdate().Model(reset).
		Set("used_at = ?", now).
		Where("id = ?", reset.ID).
		Exec(ctx); err != nil {
		return fmt.Errorf("auth.ResetPassword: update token: %w", err)
	}

	return nil
}

// ValidateToken checks if a token is valid
func (s *Service) ValidateToken(ctx context.Context, token string) (string, string, error) {
	td := &TokenData{}
	if err := s.redis.GetJSON(ctx, accessTokenPrefix+token, td); err != nil {
		return "", "", fmt.Errorf("invalid token")
	}

	return td.UserID, td.WorkspaceID, nil
}

// GetUserWorkspaces fetches all workspaces for a user
func (s *Service) GetUserWorkspaces(ctx context.Context, userID string) ([]*WorkspaceResponse, error) {
	members := []*WorkspaceMember{}
	err := s.repo.db.NewSelect().Model(&members).
		Where("user_id = ?", userID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth.GetUserWorkspaces: fetch members: %w", err)
	}

	var response []*WorkspaceResponse
	for _, member := range members {
		workspace := &Workspace{}
		if err := s.repo.db.NewSelect().Model(workspace).
			Where("id = ?", member.WorkspaceID).
			Scan(ctx); err != nil {
			s.log.Error("failed to fetch workspace", zap.Error(err))
			continue
		}

		response = append(response, &WorkspaceResponse{
			ID:      workspace.ID,
			Name:    workspace.Name,
			Slug:    workspace.Slug,
			LogoURL: workspace.LogoURL,
			Plan:    workspace.Plan,
			Role:    member.Role,
		})
	}

	return response, nil
}

// Helper functions

func (s *Service) cacheAccessToken(ctx context.Context, userID, workspaceID, token string) error {
	data := &TokenData{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}
	return s.redis.SetJSON(ctx, accessTokenPrefix+token, data, AccessTokenTTL)
}

func (s *Service) cacheRefreshToken(ctx context.Context, userID, workspaceID, token string) error {
	data := &TokenData{
		UserID:      userID,
		WorkspaceID: workspaceID,
	}
	return s.redis.SetJSON(ctx, refreshTokenPrefix+token, data, RefreshTokenTTL)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.redis.Delete(ctx, refreshTokenPrefix+refreshToken)
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	td := &TokenData{}
	if err := s.redis.GetJSON(ctx, refreshTokenPrefix+refreshToken, td); err != nil {
		return "", fmt.Errorf("invalid refresh token")
	}

	workspace, err := s.repo.GetWorkspaceByID(ctx, td.WorkspaceID)
	if err != nil {
		return "", fmt.Errorf("auth.RefreshToken: fetch workspace: %w", err)
	}
	member, err := s.repo.GetWorkspaceMember(ctx, td.WorkspaceID, td.UserID)
	if err != nil {
		return "", fmt.Errorf("auth.RefreshToken: fetch membership: %w", err)
	}

	accessToken, err := generateAccessToken(td.UserID, td.WorkspaceID, member.Role, workspace.Plan)
	if err != nil {
		return "", fmt.Errorf("auth.RefreshToken: generate token: %w", err)
	}
	if err := s.cacheAccessToken(ctx, td.UserID, td.WorkspaceID, accessToken); err != nil {
		return "", fmt.Errorf("auth.RefreshToken: cache token: %w", err)
	}

	return accessToken, nil
}

func (s *Service) GetMe(ctx context.Context, userID, workspaceID string) (*AuthResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetMe: fetch user: %w", err)
	}

	workspace, err := s.repo.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetMe: fetch workspace: %w", err)
	}

	member, err := s.repo.GetWorkspaceMember(ctx, workspaceID, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetMe: fetch membership: %w", err)
	}

	return &AuthResponse{
		User: user.ToResponse(),
		Workspace: &WorkspaceResponse{
			ID:      workspace.ID,
			Name:    workspace.Name,
			Slug:    workspace.Slug,
			LogoURL: workspace.LogoURL,
			Plan:    workspace.Plan,
			Role:    member.Role,
		},
	}, nil
}

func (s *Service) sendVerificationEmail(ctx context.Context, user *User) {
	// TODO: Implement Resend email service integration
	s.log.Info("verification email would be sent", zap.String("email", user.Email))
}

func (s *Service) sendPasswordResetEmail(ctx context.Context, user *User, token string) {
	// TODO: Implement Resend email service integration
	s.log.Info("password reset email would be sent", zap.String("email", user.Email))
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func generateAccessToken(userID, workspaceID, role, plan string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = os.Getenv("APP_SECRET")
	}
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is required")
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":          userID,
		"workspace_id": workspaceID,
		"role":         role,
		"plan":         plan,
		"iat":          now.Unix(),
		"exp":          now.Add(AccessTokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateSlug(name string) string {
	// Simple slug generation - in production use a proper library
	slug := name
	// Remove spaces and lowercase
	for i, c := range slug {
		if c == ' ' {
			slug = slug[:i] + "-" + slug[i+1:]
		}
	}
	return slug
}
