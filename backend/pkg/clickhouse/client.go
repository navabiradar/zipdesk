package clickhouse

import (
    "context"
    "fmt"
    "time"

    "github.com/ClickHouse/clickhouse-go/v2"
    "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Client wraps ClickHouse connection
type Client struct {
    conn driver.Conn
}

// Config holds ClickHouse configuration
type Config struct {
    DSN      string
    Database string
    Debug    bool
}

// New creates a new ClickHouse client
func New(cfg Config) (*Client, error) {
    opts, err := clickhouse.ParseDSN(cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("clickhouse.New: parse dsn: %w", err)
    }

    opts.Debug = cfg.Debug
    opts.Compression = &clickhouse.Compression{
        Method: clickhouse.CompressionLZ4,
    }
    opts.DialTimeout = 10 * time.Second
    opts.ReadTimeout = 30 * time.Second

    conn, err := clickhouse.Open(opts)
    if err != nil {
        return nil, fmt.Errorf("clickhouse.New: open: %w", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := conn.Ping(ctx); err != nil {
        return nil, fmt.Errorf("clickhouse.New: ping: %w", err)
    }

    return &Client{conn: conn}, nil
}

// Insert inserts a batch of rows
func (c *Client) Insert(
    ctx context.Context,
    query string,
    rows []interface{},
) error {
    batch, err := c.conn.PrepareBatch(ctx, query)
    if err != nil {
        return fmt.Errorf("clickhouse.Insert: prepare: %w", err)
    }

    for _, row := range rows {
        if err := batch.AppendStruct(row); err != nil {
            return fmt.Errorf("clickhouse.Insert: append: %w", err)
        }
    }

    if err := batch.Send(); err != nil {
        return fmt.Errorf("clickhouse.Insert: send: %w", err)
    }

    return nil
}

// Query executes a query and scans results
func (c *Client) Query(
    ctx context.Context,
    query string,
    args ...interface{},
) (driver.Rows, error) {
    rows, err := c.conn.Query(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("clickhouse.Query: %w", err)
    }
    return rows, nil
}

// Exec executes a statement
func (c *Client) Exec(
    ctx context.Context,
    query string,
    args ...interface{},
) error {
    if err := c.conn.Exec(ctx, query, args...); err != nil {
        return fmt.Errorf("clickhouse.Exec: %w", err)
    }
    return nil
}

// HealthCheck verifies ClickHouse connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
    if err := c.conn.Ping(ctx); err != nil {
        return fmt.Errorf("clickhouse.HealthCheck: %w", err)
    }
    return nil
}

// InitSchema creates required tables
func (c *Client) InitSchema(ctx context.Context) error {
    queries := []string{
        createLinkClicksTable,
        createEmailEventsTable,
    }

    for _, q := range queries {
        if err := c.Exec(ctx, q); err != nil {
            return fmt.Errorf("clickhouse.InitSchema: %w", err)
        }
    }

    return nil
}

const createLinkClicksTable = `
CREATE TABLE IF NOT EXISTS link_clicks (
    id           UUID,
    link_id      String,
    workspace_id String,
    session_hash String,
    ip_hash      String,
    country_code String,
    country_name String,
    city         String,
    latitude     Float64,
    longitude    Float64,
    device_type  String,
    browser      String,
    os           String,
    referrer     String,
    utm_source   String,
    utm_medium   String,
    utm_campaign String,
    clicked_at   DateTime
) ENGINE = MergeTree()
ORDER BY (workspace_id, link_id, clicked_at)
PARTITION BY toYYYYMM(clicked_at)
`

const createEmailEventsTable = `
CREATE TABLE IF NOT EXISTS email_events (
    id           UUID,
    campaign_id  String,
    contact_id   String,
    workspace_id String,
    event_type   String,
    link_url     String,
    ip_hash      String,
    device       String,
    event_at     DateTime
) ENGINE = MergeTree()
ORDER BY (workspace_id, campaign_id, event_at)
PARTITION BY toYYYYMM(event_at)
`
