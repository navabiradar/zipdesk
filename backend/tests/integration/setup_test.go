package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/internal/auth"
	"github.com/zipdesk/backend/internal/crm"
	"github.com/zipdesk/backend/internal/docs"
	"github.com/zipdesk/backend/internal/flow"
	"github.com/zipdesk/backend/internal/forms"
	"github.com/zipdesk/backend/internal/links"
	"github.com/zipdesk/backend/internal/mail"
	"github.com/zipdesk/backend/pkg/cache"
	"github.com/zipdesk/backend/pkg/database"
)

// ZipDeskSuite is the base test suite
type ZipDeskSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	db          *bun.DB
	redis       *cache.Client
	app         *fiber.App
	workspaceID string
	userID      string
	userEmail   string
	authToken   string

	// Services (accessible in subtests)
	authSvc  *auth.Service
	linksSvc *links.Service
	formsSvc *forms.Service
	mailSvc  *mail.Service
	crmSvc   *crm.Service
	docsSvc  *docs.Service
	flowSvc  *flow.Service
	eventBus *flow.EventBus
}

// SetupSuite initializes test infrastructure
func (s *ZipDeskSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	os.Setenv("APP_ENV", "test")
	os.Setenv("JWT_SECRET", "test-jwt-secret-256bit")
	os.Setenv("APP_SECRET", "test-256-bit-secret-here")

	// ---- Database ----
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:test@localhost:5432/zipdesk_test?sslmode=disable"
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dbURL)))
	sqldb.SetMaxOpenConns(10)
	sqldb.SetMaxIdleConns(5)
	s.db = bun.NewDB(sqldb, pgdialect.New())
	s.Require().NoError(s.db.PingContext(s.ctx), "database must be available for tests")

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = filepath.Join("..", "..", "migrations")
	}
	migrationsDir = filepath.ToSlash(migrationsDir)
	err := database.Migrate(dbURL, migrationsDir)
	s.Require().NoError(err, "migrations must pass")

	// ---- Redis ----
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	s.redis, err = cache.New(cache.Config{URL: redisURL})
	s.Require().NoError(err, "redis must be available for tests")

	// ---- Test user & workspace (direct DB to avoid slug collision) ----
	s.createTestWorkspace()

	// ---- Full app ----
	s.buildApp()
}

// TearDownSuite cleans up after all tests
func (s *ZipDeskSuite) TearDownSuite() {
	s.cancel()
	s.cleanTestData()
	if s.db != nil {
		s.db.Close()
	}
}

// SetupTest runs before each test
func (s *ZipDeskSuite) SetupTest() {
	s.cleanEventData()
}

// buildApp creates the Fiber test app with all services
func (s *ZipDeskSuite) buildApp() {
	log, _ := zap.NewDevelopment()

	s.app = fiber.New(fiber.Config{
		AppName: "ZipDesk Test",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
		},
	})

	v1 := s.app.Group("/api/v1")

	// Event bus (repo variant – persists to DB)
	flowRepo := flow.NewRepository(s.db)
	s.eventBus = flow.NewEventBusFromRepo(nil, flowRepo, log)

	// Auth handler
	authRepo := auth.NewRepository(s.db)
	s.authSvc = auth.NewService(authRepo, s.redis, log)
	authHandler := auth.NewHandler(s.authSvc, log)
	authHandler.RegisterRoutes(s.app, v1)

	// Links
	linksRepo := links.NewRepository(s.db, nil)
	s.linksSvc = links.NewService(linksRepo, s.redis, s.eventBus, log)
	linksHandler := links.NewHandler(s.linksSvc, log)
	linksHandler.RegisterRoutes(s.app, v1)

	// Forms
	formsRepo := forms.NewRepository(s.db)
	s.formsSvc = forms.NewService(formsRepo, nil, s.eventBus, nil, log)
	formsHandler := forms.NewHandler(s.formsSvc, log)
	formsHandler.RegisterRoutes(s.app, v1)

	// Mail
	mailRepo := mail.NewRepository(s.db, nil)
	s.mailSvc = mail.NewService(mailRepo, nil, s.eventBus, log)
	mailHandler := mail.NewHandler(s.mailSvc, log)
	mailHandler.RegisterRoutes(s.app, v1)

	// CRM
	crmRepo := crm.NewRepository(s.db)
	s.crmSvc = crm.NewService(crmRepo, s.eventBus, log)
	crmHandler := crm.NewHandler(s.crmSvc, log)
	crmHandler.RegisterRoutes(s.app, v1)

	// Docs
	docsRepo := docs.NewRepository(s.db)
	s.docsSvc = docs.NewService(docsRepo, nil, nil, s.eventBus, log)
	docsHandler := docs.NewHandler(s.docsSvc, log)
	docsHandler.RegisterRoutes(s.app, v1)

	// Flow (needs mailSvc for triggers)
	s.flowSvc = flow.NewService(flowRepo, s.eventBus, s.mailSvc, s.redis, log)
	flowHandler := flow.NewHandler(s.flowSvc, log)
	flowHandler.RegisterRoutes(s.app, v1)

	// System triggers (mail + crm event listeners)
	triggers := flow.NewSystemTriggers(s.mailSvc, s.crmSvc, log)
	triggers.Register(s.eventBus)

	// Init default blueprints for the test workspace
	_ = s.flowSvc.InitWorkspace(s.ctx, s.workspaceID)
}

// createTestWorkspace creates test user + workspace directly in the DB
// and generates a JWT.  Uses a unique slug every time to avoid collisions
// from partial / aborted test runs on a shared database.
func (s *ZipDeskSuite) createTestWorkspace() {
	now := time.Now()
	slugSuffix := fmt.Sprintf("test-%d", now.UnixMilli())

	// Insert user
	userID := uuid.New().String()
	email := "testuser+" + slugSuffix + "@zipdesk-test.com"
	_, err := s.db.NewInsert().Model(&struct {
		bun.BaseModel `bun:"table:users"`
		ID            string    `bun:"id,pk"`
		Email         string    `bun:"email"`
		Name          string    `bun:"name"`
		PasswordHash  string    `bun:"password_hash"`
		IsVerified    bool      `bun:"is_verified"`
		CreatedAt     time.Time `bun:"created_at"`
		UpdatedAt     time.Time `bun:"updated_at"`
	}{
		ID:           userID,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "$2a$10$dummyhashnotreallyvalidbutokayfornow",
		IsVerified:   true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Exec(s.ctx)
	s.Require().NoError(err, "insert test user")

	// Insert workspace
	workspaceID := uuid.New().String()
	_, err = s.db.NewInsert().Model(&struct {
		bun.BaseModel `bun:"table:workspaces"`
		ID            string    `bun:"id,pk"`
		Name          string    `bun:"name"`
		Slug          string    `bun:"slug"`
		OwnerID       string    `bun:"owner_id"`
		Plan          string    `bun:"plan"`
		CreatedAt     time.Time `bun:"created_at"`
		UpdatedAt     time.Time `bun:"updated_at"`
	}{
		ID:        workspaceID,
		Name:      "Test Workspace",
		Slug:      slugSuffix,
		OwnerID:   userID,
		Plan:      "free",
		CreatedAt: now,
		UpdatedAt: now,
	}).Exec(s.ctx)
	s.Require().NoError(err, "insert test workspace")

	// Insert workspace member
	_, err = s.db.NewInsert().Model(&struct {
		bun.BaseModel `bun:"table:workspace_members"`
		ID            string    `bun:"id,pk"`
		WorkspaceID   string    `bun:"workspace_id"`
		UserID        string    `bun:"user_id"`
		Role          string    `bun:"role"`
		JoinedAt      time.Time `bun:"joined_at"`
	}{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        "owner",
		JoinedAt:    now,
	}).Exec(s.ctx)
	s.Require().NoError(err, "insert workspace member")

	// Generate JWT
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-jwt-secret-256bit"
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":          userID,
		"email":        email,
		"workspace_id": workspaceID,
		"exp":          time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	s.Require().NoError(err, "generate jwt")

	s.workspaceID = workspaceID
	s.userID = userID
	s.userEmail = email
	s.authToken = tokenString
}

// cleanTestData removes all test workspace data
func (s *ZipDeskSuite) cleanTestData() {
	if s.workspaceID == "" {
		return
	}

	// Workspace CASCADE will clean most child rows
	_, _ = s.db.NewDelete().TableExpr("workspaces").
		Where("id = ?", s.workspaceID).Exec(s.ctx)

	// User may exist independently (separate from workspace CASCADE)
	_, _ = s.db.NewDelete().TableExpr("users").
		Where("id = ?", s.userID).Exec(s.ctx)
}

// cleanEventData clears ephemeral data between subtests
func (s *ZipDeskSuite) cleanEventData() {
	// Delete all test artifacts regardless of workspace — each test run
	// creates a new unique workspace anyway.  bun's NewDelete requires at
	// least one Where clause as a safety measure, so use "1=1".
	whereAll := "1=1"
	_, _ = s.db.NewDelete().TableExpr("events").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("forms").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("documents").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("links").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("mail_contacts").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("mail_campaigns").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("crm_contacts").Where(whereAll).Exec(s.ctx)
	_, _ = s.db.NewDelete().TableExpr("crm_deals").Where(whereAll).Exec(s.ctx)
}

// authHeader returns the bearer token header value
func (s *ZipDeskSuite) authHeader() string {
	return "Bearer " + s.authToken
}

// waitForAsync yields briefly so async event handlers complete
func (s *ZipDeskSuite) waitForAsync() {
	time.Sleep(300 * time.Millisecond)
}

// getEvents returns events of the given type for the test workspace
func (s *ZipDeskSuite) getEvents(eventType string) []flow.Event {
	var events []flow.Event
	_ = s.db.NewSelect().Model(&events).
		Where("workspace_id = ? AND type = ?", s.workspaceID, eventType).
		OrderExpr("occurred_at DESC").Scan(s.ctx, &events)
	return events
}

// getMailContact looks up a contact by email
func (s *ZipDeskSuite) getMailContact(email string) *mail.Contact {
	contact := new(mail.Contact)
	err := s.db.NewSelect().Model(contact).
		Where("workspace_id = ? AND email = ?", s.workspaceID, email).
		Scan(s.ctx)
	if err != nil {
		return nil
	}
	return contact
}

// getCRMContact looks up a CRM contact by email
func (s *ZipDeskSuite) getCRMContact(email string) *crm.CRMContact {
	contact := new(crm.CRMContact)
	err := s.db.NewSelect().Model(contact).
		Where("workspace_id = ? AND email = ?", s.workspaceID, email).
		Scan(s.ctx)
	if err != nil {
		return nil
	}
	return contact
}

// mustJSON marshals v to JSON or fails the test
func mustJSON(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustJSON: %v", err)
	}
	return b
}

// TestZipDeskSuite runs the full integration suite
func TestZipDeskSuite(t *testing.T) {
	suite.Run(t, new(ZipDeskSuite))
}
