package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/internal/auth"
	"github.com/zipdesk/backend/internal/links"
	"github.com/zipdesk/backend/internal/forms"
	"github.com/zipdesk/backend/internal/docs"
	"github.com/zipdesk/backend/internal/mail"
	"github.com/zipdesk/backend/internal/crm"
	"github.com/zipdesk/backend/internal/flow"
	"github.com/zipdesk/backend/pkg/cache"
	"github.com/zipdesk/backend/pkg/clickhouse"
	"github.com/zipdesk/backend/pkg/database"
	"github.com/zipdesk/backend/pkg/queue"
	"github.com/zipdesk/backend/pkg/storage"
)

func main() {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}

	log, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	ctx := context.Background()

	db, err := database.New(database.Config{
		DSN:             os.Getenv("DATABASE_URL"),
		MaxOpenConns:    20,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := database.Migrate(os.Getenv("DATABASE_URL"), "./migrations"); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}

	redis, err := cache.New(cache.Config{URL: os.Getenv("REDIS_URL")})
	if err != nil {
		log.Fatal("failed to connect to redis", zap.Error(err))
	}

	ch, err := clickhouse.New(clickhouse.Config{DSN: os.Getenv("CLICKHOUSE_DSN")})
	if err != nil {
		log.Warn("clickhouse not available", zap.Error(err))
	}

	store, err := storage.New(storage.Config{
		AccountID:  os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		AccessKey:  os.Getenv("CLOUDFLARE_R2_ACCESS_KEY"),
		SecretKey:  os.Getenv("CLOUDFLARE_R2_SECRET_KEY"),
		BucketName: os.Getenv("CLOUDFLARE_R2_BUCKET"),
		PublicURL:  os.Getenv("CLOUDFLARE_R2_PUBLIC_URL"),
	})
	if err != nil {
		log.Warn("storage not available", zap.Error(err))
	}

	q, err := queue.New(queue.Config{RedisURL: os.Getenv("ASYNQ_REDIS_URL")})
	if err != nil {
		log.Warn("queue not available", zap.Error(err))
	}

	app := fiber.New(fiber.Config{
		AppName:                 "ZipDesk API v1",
		ReadTimeout:             30 * time.Second,
		WriteTimeout:            30 * time.Second,
		IdleTimeout:             120 * time.Second,
		EnableTrustedProxyCheck: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"success": false, "error": fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()}})
		},
	})

	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{Format: "${time} ${method} ${path} ${status} ${latency}\n"}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     os.Getenv("FRONTEND_URL") + ",http://localhost:3000",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "zipdesk-api", "time": time.Now().UTC()})
	})

	deps := Deps{
		DB:      db,
		BunDB:   db.BunDB(),
		Redis:   redis,
		CH:      ch,
		Storage: store,
		Queue:   q,
		Logger:  log,
	}

	v1 := app.Group("/api/v1")
	registerRoutes(ctx, app, v1, deps)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("starting zipdesk api", zap.String("port", port))
		if err := app.Listen(":" + port); err != nil {
			log.Error("server error", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down gracefully...")

	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}

	log.Info("zipdesk api stopped")
}

type Deps struct {
	DB      *database.DB
	BunDB   *bun.DB
	Redis   *cache.Client
	CH      *clickhouse.Client
	Storage *storage.Client
	Queue   *queue.Client
	Logger  *zap.Logger
}

func registerRoutes(
	ctx context.Context,
	app *fiber.App,
	v1 fiber.Router,
	deps Deps,
) {
	// Auth
	authRepo := auth.NewRepository(deps.BunDB)
	authSvc := auth.NewService(authRepo, deps.Redis, deps.Logger)
	authHandler := auth.NewHandler(authSvc, deps.Logger)
	authHandler.RegisterRoutes(app, v1)

	// Flow — Wire up the full event bus system
	flowRepo := flow.NewRepository(deps.BunDB)
	flowStorage := flow.NewStorage(deps.BunDB)
	flowActions := flow.NewActions(deps.Logger, nil)
	actionRegistry := flow.NewActionRegistry()
	flowActions.RegisterBuiltin(actionRegistry)
	engine := flow.NewEngine(actionRegistry, flowStorage, deps.Logger)
	eventBus := flow.NewEventBus(deps.Queue, flowStorage, engine, deps.Logger)

	// Links
	linksRepo := links.NewRepository(deps.BunDB, deps.CH)
	linksSvc := links.NewService(linksRepo, deps.Redis, eventBus, deps.Logger)
	linksHandler := links.NewHandler(linksSvc, deps.Logger)
	linksHandler.RegisterRoutes(app, v1)

	// Forms
	formsRepo := forms.NewRepository(deps.BunDB)
	formsSvc := forms.NewService(
		formsRepo, deps.Queue, eventBus,
		deps.Storage, deps.Logger,
	)
	formsHandler := forms.NewHandler(formsSvc, deps.Logger)
	formsHandler.RegisterRoutes(app, v1)

	// Docs
	docsRepo := docs.NewRepository(deps.BunDB)
	docsSvc := docs.NewService(
		docsRepo, deps.Storage,
		deps.Queue, eventBus, deps.Logger,
	)
	docsHandler := docs.NewHandler(docsSvc, deps.Logger)
	docsHandler.RegisterRoutes(app, v1)

	// Mail
	mailRepo := mail.NewRepository(deps.BunDB, deps.CH)
	mailSvc := mail.NewService(
		mailRepo, deps.Queue, eventBus, deps.Logger,
	)
	mailHandler := mail.NewHandler(mailSvc, deps.Logger)
	mailHandler.RegisterRoutes(app, v1)

	// CRM
	crmRepo := crm.NewRepository(deps.BunDB)
	crmSvc := crm.NewService(crmRepo, eventBus, deps.Logger)
	crmHandler := crm.NewHandler(crmSvc, deps.Logger)
	crmHandler.RegisterRoutes(app, v1)

	// Flow service wiring
	flowSvc := flow.NewService(
		flowRepo, eventBus, mailSvc, deps.Redis, deps.Logger,
	)
	flowHandler := flow.NewHandler(flowSvc, deps.Logger)
	flowHandler.RegisterRoutes(app, v1)

	// System triggers
	triggers := flow.NewSystemTriggers(
		mailSvc, crmSvc, deps.Logger,
	)
	triggers.Register(eventBus)

	// Health monitor
	monitor := flow.NewHealthMonitor(
		deps.Redis, deps.BunDB,
		eventBus, deps.Logger,
	)
	flowSvc.SetMonitor(monitor)
	monitor.StartCron(ctx)
}
