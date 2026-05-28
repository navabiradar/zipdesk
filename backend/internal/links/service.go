package links

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/zipdesk/backend/internal/flow"
	"github.com/zipdesk/backend/pkg/cache"
)

const (
	linkCacheTTL = 24 * time.Hour
	linkCacheKey = "link:"
)

var (
	ErrNotFound       = errors.New("link not found")
	ErrInvalidInput   = errors.New("invalid link input")
	ErrInactive       = errors.New("link is inactive")
	ErrExpired        = errors.New("link has expired")
	ErrClickLimit     = errors.New("link click limit reached")
	ErrPasswordNeeded = errors.New("link password required")
)

// Service handles links business logic
type Service struct {
	repo     *Repository
	redis    *cache.Client
	eventBus *flow.EventBus
	log      *zap.Logger
}

// NewService creates a new links service
func NewService(
	repo *Repository,
	redis *cache.Client,
	eventBus *flow.EventBus,
	log *zap.Logger,
) *Service {
	return &Service{
		repo:     repo,
		redis:    redis,
		eventBus: eventBus,
		log:      log,
	}
}

// CreateLink creates a new short link
func (s *Service) CreateLink(
	ctx context.Context,
	workspaceID string,
	userID string,
	input CreateLinkInput,
) (*Link, error) {
	// Normalize URL
	normalURL, err := NormalizeURL(input.OriginalURL)
	if err != nil {
		return nil, err
	}

	// Validate custom slug if provided
	if input.CustomSlug != "" {
		if err := ValidateCustomSlug(input.CustomSlug); err != nil {
			return nil, err
		}
		exists, err := s.repo.ShortCodeExists(ctx, input.CustomSlug)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, &ValidationError{
				Field:   "custom_slug",
				Message: "slug already exists",
			}
		}
	}

	// Generate short code
	shortCode, err := GenerateUniqueCode(
		ctx, s.repo.db, codeLength,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"links.Service.CreateLink: generate code: %w", err,
		)
	}

	// Hash password if provided
	var passwordHash string
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword(
			[]byte(input.Password), 10,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"links.Service.CreateLink: hash password: %w", err,
			)
		}
		passwordHash = string(hash)
	}

	link := &Link{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		OriginalURL: normalURL,
		ShortCode:   shortCode,
		CustomSlug:  input.CustomSlug,
		Title:       input.Title,
		Password:    passwordHash,
		ExpiresAt:   input.ExpiresAt,
		ClickLimit:  input.ClickLimit,
		Tags:        input.Tags,
		FolderID:    input.FolderID,
		UTMParams:   input.UTMParams,
		Settings:    map[string]any{},
		IsActive:    true,
		CreatedBy:   userID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if link.Tags == nil {
		link.Tags = []string{}
	}
	if link.UTMParams == nil {
		link.UTMParams = map[string]any{}
	}

	if err := s.repo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf(
			"links.Service.CreateLink: %w", err,
		)
	}

	// Cache the link
	_ = s.cacheLinkTargets(ctx, link)

	// Fire event
	if s.eventBus != nil {
		s.eventBus.PublishAsync(flow.LinkCreatedEvent{
			BaseEvent: flow.BaseEvent{
				Type:        flow.EventLinkCreated,
				WorkspaceID: workspaceID,
				Source:      "links",
				OccurredAt:  time.Now(),
			},
			LinkID:    link.ID,
			ShortCode: shortCode,
		})
	}

	s.log.Info("link created",
		zap.String("link_id", link.ID),
		zap.String("workspace_id", workspaceID),
		zap.String("short_code", shortCode),
	)

	return link, nil
}

// HandleRedirect processes a redirect request
// This is the CRITICAL hot path - must be < 10ms
func (s *Service) HandleRedirect(
	ctx context.Context,
	slug string,
	req ClickRequest,
) (string, error) {
	// Step 1: Check Redis cache first
	cacheKey := linkCacheKey + slug
	cachedURL, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && cachedURL != "" {
		// Cache hit - record async and return
		go s.recordClickBackground(slug, req)
		return cachedURL, nil
	}

	// Step 2: Cache miss - query database
	link, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return "", &LinkError{
			Code:    "NOT_FOUND",
			Message: "link not found",
		}
	}

	// Step 3: Validate link
	if err := s.validateLink(link); err != nil {
		return "", err
	}

	target := BuildUTMURL(link.OriginalURL, link.UTMParams)

	// Step 4: Cache for future redirects
	_ = s.redis.Set(
		ctx, cacheKey, target, linkCacheTTL,
	).Err()

	// Step 5: Record click asynchronously
	go s.recordClickBackground(slug, req)

	return target, nil
}

// validateLink checks if link can be redirected
func (s *Service) validateLink(link *Link) error {
	if !link.IsActive {
		return &LinkError{
			Code:    "LINK_INACTIVE",
			Message: "this link has been disabled",
		}
	}

	if link.ExpiresAt != nil && time.Now().After(*link.ExpiresAt) {
		return &LinkError{
			Code:    "EXPIRED",
			Message: "this link has expired",
		}
	}

	if link.ClickLimit != nil &&
		link.TotalClicks >= *link.ClickLimit {
		return &LinkError{
			Code:    "CLICK_LIMIT_REACHED",
			Message: "this link has reached its click limit",
		}
	}

	return nil
}

// recordClickBackground records click without blocking
func (s *Service) recordClickBackground(
	slug string,
	req ClickRequest,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(), 5*time.Second,
	)
	defer cancel()

	link, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return
	}

	// Check if unique click (Redis based)
	sessionHash := hashSession(req.IP, req.UserAgent)
	uniqueKey := fmt.Sprintf(
		"unique:%s:%s", link.ID, sessionHash,
	)
	isNew, _ := s.redis.SetNX(
		ctx, uniqueKey, "1", 24*time.Hour,
	).Result()

	// Increment click count
	_ = s.repo.IncrementClicks(ctx, link.ID, isNew)

	// Build click record
	click := LinkClick{
		ID:          newClickID(),
		LinkID:      link.ID,
		WorkspaceID: link.WorkspaceID,
		SessionHash: sessionHash,
		IPHash:      hashIP(req.IP),
		DeviceType:  parseDevice(req.UserAgent),
		Browser:     parseBrowser(req.UserAgent),
		OS:          parseOS(req.UserAgent),
		ClickedAt:   time.Now(),
	}

	// Parse referrer
	if req.Referrer != "" {
		click.ReferrerDomain = parseDomain(req.Referrer)
	}

	// Parse UTM params from URL
	if link.UTMParams != nil {
		if src, ok := link.UTMParams["utm_source"].(string); ok {
			click.UTMSource = src
		}
		if med, ok := link.UTMParams["utm_medium"].(string); ok {
			click.UTMMedium = med
		}
		if cam, ok := link.UTMParams["utm_campaign"].(string); ok {
			click.UTMCampaign = cam
		}
	}

	// Record in ClickHouse
	_ = s.repo.RecordClick(ctx, click)

	// Fire event
	if s.eventBus != nil {
		s.eventBus.PublishAsync(flow.LinkClickedEvent{
			BaseEvent: flow.BaseEvent{
				Type:        flow.EventLinkClicked,
				WorkspaceID: link.WorkspaceID,
				Source:      "links",
				OccurredAt:  time.Now(),
			},
			LinkID:    link.ID,
			ShortCode: slug,
			Country:   click.CountryCode,
			Device:    click.DeviceType,
		})
	}
}

// GetByID returns a link by ID
func (s *Service) GetByID(
	ctx context.Context,
	id string,
	workspaceID string,
) (*Link, error) {
	link, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &LinkError{
			Code:    "NOT_FOUND",
			Message: "link not found",
		}
	}
	return link, nil
}

// UpdateLink modifies an existing link
func (s *Service) UpdateLink(
	ctx context.Context,
	id string,
	workspaceID string,
	input UpdateLinkInput,
) (*Link, error) {
	link, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &LinkError{
			Code:    "NOT_FOUND",
			Message: "link not found",
		}
	}

	oldSlug := preferredSlug(link)

	if input.Title != "" {
		link.Title = input.Title
	}
	if input.OriginalURL != "" {
		normalized, err := NormalizeURL(input.OriginalURL)
		if err != nil {
			return nil, err
		}
		link.OriginalURL = normalized
	}
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword(
			[]byte(input.Password), 10,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"links.Service.UpdateLink: hash password: %w", err,
			)
		}
		link.Password = string(hash)
	}
	if input.IsActive != nil {
		link.IsActive = *input.IsActive
	}
	if input.Tags != nil {
		link.Tags = input.Tags
	}
	if input.ExpiresAt != nil {
		link.ExpiresAt = input.ExpiresAt
	}
	if input.ClickLimit != nil {
		link.ClickLimit = input.ClickLimit
	}
	if input.FolderID != "" {
		link.FolderID = input.FolderID
	}
	if input.UTMParams != nil {
		link.UTMParams = input.UTMParams
	}

	if err := s.repo.Update(ctx, link); err != nil {
		return nil, fmt.Errorf(
			"links.Service.UpdateLink: %w", err,
		)
	}

	// Invalidate cache
	_ = s.redis.Del(ctx, linkCacheKey+oldSlug).Err()
	_ = s.redis.Del(ctx, linkCacheKey+link.ShortCode).Err()
	if link.CustomSlug != "" {
		_ = s.redis.Del(ctx, linkCacheKey+link.CustomSlug).Err()
	}

	return link, nil
}

// DeleteLink removes a link
func (s *Service) DeleteLink(
	ctx context.Context,
	id string,
	workspaceID string,
) error {
	link, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return &LinkError{
			Code:    "NOT_FOUND",
			Message: "link not found",
		}
	}

	if err := s.repo.Delete(ctx, id, workspaceID); err != nil {
		return fmt.Errorf(
			"links.Service.DeleteLink: %w", err,
		)
	}

	// Invalidate cache
	_ = s.redis.Del(ctx, linkCacheKey+link.ShortCode).Err()
	if link.CustomSlug != "" {
		_ = s.redis.Del(ctx, linkCacheKey+link.CustomSlug).Err()
	}

	return nil
}

// ListLinks returns paginated links
func (s *Service) ListLinks(
	ctx context.Context,
	workspaceID string,
	params ListParams,
) (*ListResponse, error) {
	items, total, err := s.repo.List(
		ctx, workspaceID, params,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"links.Service.ListLinks: %w", err,
		)
	}

	page, perPage := normalizeListPagination(params.Page, params.PerPage)
	return &ListResponse{
		Items:   items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

// GetAnalytics returns click analytics for a link
func (s *Service) GetAnalytics(
	ctx context.Context,
	id string,
	workspaceID string,
	params AnalyticsParams,
) (*Analytics, error) {
	// Verify ownership
	link, err := s.repo.GetByID(ctx, id, workspaceID)
	if err != nil {
		return nil, &LinkError{
			Code:    "NOT_FOUND",
			Message: "link not found",
		}
	}

	// Default date range: last 30 days
	to := time.Now()
	from := to.AddDate(0, 0, -30)

	if params.From != "" {
		if t, err := time.Parse("2006-01-02", params.From); err == nil {
			from = t
		}
	}
	if params.To != "" {
		if t, err := time.Parse("2006-01-02", params.To); err == nil {
			to = t
		}
	}

	analytics, err := s.repo.GetAnalytics(ctx, id, from, to)
	if err != nil {
		return nil, err
	}
	if s.repo.ch == nil {
		analytics.TotalClicks = int64(link.TotalClicks)
		analytics.UniqueClicks = int64(link.UniqueClicks)
	}
	return analytics, nil
}

func (s *Service) CreateFolder(ctx context.Context, workspaceID string, input *CreateFolderInput) (*LinkFolder, error) {
	name := strings.TrimSpace(input.Name)
	if workspaceID == "" || name == "" {
		return nil, fmt.Errorf("%w: folder name is required", ErrInvalidInput)
	}
	folder := &LinkFolder{
		WorkspaceID: workspaceID,
		Name:        name,
		ParentID:    input.ParentID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repo.CreateFolder(ctx, folder); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *Service) ListFolders(ctx context.Context, workspaceID string) ([]LinkFolder, error) {
	return s.repo.ListFolders(ctx, workspaceID)
}

// Compatibility wrappers used by the current HTTP handler.
func (s *Service) Create(ctx context.Context, workspaceID, userID string, input *CreateLinkInput) (*Link, error) {
	return s.CreateLink(ctx, workspaceID, userID, *input)
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (*Link, error) {
	return s.GetByID(ctx, id, workspaceID)
}

func (s *Service) Update(ctx context.Context, workspaceID, id string, input *UpdateLinkInput) (*Link, error) {
	return s.UpdateLink(ctx, id, workspaceID, *input)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	return s.DeleteLink(ctx, id, workspaceID)
}

func (s *Service) List(ctx context.Context, workspaceID string, params ListParams) (*ListResponse, error) {
	return s.ListLinks(ctx, workspaceID, params)
}

func (s *Service) Analytics(ctx context.Context, workspaceID, id string, params AnalyticsParams) (*Analytics, error) {
	return s.GetAnalytics(ctx, id, workspaceID, params)
}

func (s *Service) Resolve(ctx context.Context, slug, password string, req ClickRequest) (*Link, error) {
	link, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, &LinkError{Code: "NOT_FOUND", Message: "link not found"}
	}
	if link.Password != "" {
		if password == "" {
			return nil, ErrPasswordNeeded
		}
		if err := bcrypt.CompareHashAndPassword([]byte(link.Password), []byte(password)); err != nil {
			return nil, ErrPasswordNeeded
		}
	}
	if err := s.validateLink(link); err != nil {
		return nil, err
	}
	target, err := s.HandleRedirect(ctx, slug, req)
	if err != nil {
		return nil, err
	}
	link.OriginalURL = target
	return link, nil
}

func (s *Service) cacheLinkTargets(ctx context.Context, link *Link) error {
	target := BuildUTMURL(link.OriginalURL, link.UTMParams)
	if err := s.redis.Set(ctx, linkCacheKey+link.ShortCode, target, linkCacheTTL).Err(); err != nil {
		return err
	}
	if link.CustomSlug != "" {
		return s.redis.Set(ctx, linkCacheKey+link.CustomSlug, target, linkCacheTTL).Err()
	}
	return nil
}

func preferredSlug(link *Link) string {
	if link.CustomSlug != "" {
		return link.CustomSlug
	}
	return link.ShortCode
}

func normalizeListPagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}

// User agent parsing helpers
func parseDevice(ua string) string {
	if containsAny(ua, "Mobile", "Android", "iPhone", "iPad") {
		if containsAny(ua, "iPad", "Tablet") {
			return "tablet"
		}
		return "mobile"
	}
	return "desktop"
}

func parseBrowser(ua string) string {
	switch {
	case contains(ua, "Chrome") && !contains(ua, "Edge"):
		return "Chrome"
	case contains(ua, "Firefox"):
		return "Firefox"
	case contains(ua, "Safari") && !contains(ua, "Chrome"):
		return "Safari"
	case contains(ua, "Edge"):
		return "Edge"
	default:
		return "Other"
	}
}

func parseOS(ua string) string {
	switch {
	case contains(ua, "Windows"):
		return "Windows"
	case contains(ua, "Mac OS"):
		return "macOS"
	case contains(ua, "Android"):
		return "Android"
	case contains(ua, "Linux"):
		return "Linux"
	case contains(ua, "iOS") || contains(ua, "iPhone"):
		return "iOS"
	default:
		return "Other"
	}
}

func parseDomain(referrer string) string {
	if referrer == "" {
		return "direct"
	}
	parsed, err := url.Parse(referrer)
	if err != nil || parsed.Hostname() == "" {
		return referrer
	}
	return parsed.Hostname()
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func clientIP(ip string) string {
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		return host
	}
	return ip
}

// LinkError is a domain error
type LinkError struct {
	Code    string
	Message string
}

func (e *LinkError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
