package database

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/pgdialect"
    "github.com/uptrace/bun/driver/pgdriver"
    "github.com/uptrace/bun/extra/bundebug"
    "go.uber.org/zap"
)

// DB wraps bun.DB with helpers
type DB struct {
    *bun.DB
}

// Config holds database configuration
type Config struct {
    DSN             string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    Debug           bool
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
    sqldb := sql.OpenDB(
        pgdriver.NewConnector(pgdriver.WithDSN(cfg.DSN)),
    )

    sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
    sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
    sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := sqldb.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("database.New: ping failed: %w", err)
    }

    db := bun.NewDB(sqldb, pgdialect.New())

    if cfg.Debug {
        db.AddQueryHook(bundebug.NewQueryHook(
            bundebug.WithVerbose(true),
        ))
    }

    return &DB{db}, nil
}

// Migrate runs database migrations
func Migrate(databaseURL, migrationsPath string) error {
    m, err := migrate.New(
        "file://"+migrationsPath,
        databaseURL,
    )
    if err != nil {
        return fmt.Errorf("database.Migrate: create migrator: %w", err)
    }
    defer m.Close()

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("database.Migrate: run migrations: %w", err)
    }

    return nil
}

// HealthCheck verifies database connectivity
func (db *DB) HealthCheck(ctx context.Context) error {
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("database.HealthCheck: %w", err)
    }
    return nil
}

// BunDB returns the underlying *bun.DB
// Use this when passing to repositories
func (db *DB) BunDB() *bun.DB {
    return db.DB
}

// WithTx runs a function within a transaction
func (db *DB) WithTx(
    ctx context.Context,
    fn func(ctx context.Context, tx bun.Tx) error,
) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("database.WithTx: begin: %w", err)
    }

    if err := fn(ctx, tx); err != nil {
        _ = tx.Rollback()
        return err
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("database.WithTx: commit: %w", err)
    }

    return nil
}

// Logger for database queries
func NewLogger(log *zap.Logger) bun.QueryHook {
    return &queryLogger{log: log}
}

type queryLogger struct {
    log *zap.Logger
}

func (l *queryLogger) BeforeQuery(
    ctx context.Context,
    event *bun.QueryEvent,
) context.Context {
    return ctx
}

func (l *queryLogger) AfterQuery(
    ctx context.Context,
    event *bun.QueryEvent,
) {
    if event.Err != nil {
        duration := time.Since(event.StartTime)
        l.log.Error("database query failed",
            zap.String("query", event.Query),
            zap.Error(event.Err),
            zap.Duration("duration", duration),
        )
    }
}
