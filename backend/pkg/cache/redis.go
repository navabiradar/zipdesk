package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Client wraps redis.Client
type Client struct {
    *redis.Client
}

// Config holds Redis configuration
type Config struct {
    URL          string
    PoolSize     int
    MinIdleConns int
}

// New creates a new Redis client
func New(cfg Config) (*Client, error) {
    opts, err := redis.ParseURL(cfg.URL)
    if err != nil {
        return nil, fmt.Errorf("cache.New: parse url: %w", err)
    }

    if cfg.PoolSize > 0 {
        opts.PoolSize = cfg.PoolSize
    }
    if cfg.MinIdleConns > 0 {
        opts.MinIdleConns = cfg.MinIdleConns
    }

    client := redis.NewClient(opts)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("cache.New: ping failed: %w", err)
    }

    return &Client{client}, nil
}

// HealthCheck verifies Redis connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
    if err := c.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("cache.HealthCheck: %w", err)
    }
    return nil
}

// SetJSON stores a value as JSON with TTL
func (c *Client) SetJSON(
    ctx context.Context,
    key string,
    value interface{},
    ttl time.Duration,
) error {
    data, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("cache.SetJSON: marshal: %w", err)
    }
    return c.Set(ctx, key, data, ttl).Err()
}

// GetJSON retrieves and unmarshals a JSON value
func (c *Client) GetJSON(
    ctx context.Context,
    key string,
    dest interface{},
) error {
    data, err := c.Get(ctx, key).Bytes()
    if err != nil {
        return fmt.Errorf("cache.GetJSON: get: %w", err)
    }
    return json.Unmarshal(data, dest)
}

// Delete removes a key
func (c *Client) Delete(ctx context.Context, key string) error {
    return c.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
    n, err := c.Client.Exists(ctx, key).Result()
    return n > 0, err
}

// IncrBy increments a key by amount
func (c *Client) IncrBy(
    ctx context.Context,
    key string,
    amount int64,
    ttl time.Duration,
) (int64, error) {
    pipe := c.Pipeline()
    incr := pipe.IncrBy(ctx, key, amount)
    pipe.Expire(ctx, key, ttl)
    _, err := pipe.Exec(ctx)
    if err != nil {
        return 0, fmt.Errorf("cache.IncrBy: %w", err)
    }
    return incr.Val(), nil
}
