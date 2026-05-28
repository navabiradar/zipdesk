package queue

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/hibiken/asynq"
)

// Task type constants
const (
    TypeSendEmail      = "email:send"
    TypeGeneratePDF    = "pdf:generate"
    TypeRecordClick    = "analytics:click"
    TypeProcessWebhook = "webhook:process"
    TypeSendCampaign   = "campaign:send"
    TypeRunBlueprint   = "flow:blueprint"
)

// Client wraps asynq client
type Client struct {
    client    *asynq.Client
    inspector *asynq.Inspector
}

// Config holds queue configuration
type Config struct {
    RedisURL string
}

// New creates a new queue client
func New(cfg Config) (*Client, error) {
    opt, err := asynq.ParseRedisURI(cfg.RedisURL)
    if err != nil {
        return nil, fmt.Errorf("queue.New: parse redis url: %w", err)
    }

    client := asynq.NewClient(opt)
    inspector := asynq.NewInspector(opt)

    return &Client{
        client:    client,
        inspector: inspector,
    }, nil
}

// Enqueue adds a task to the queue
func (c *Client) Enqueue(
    ctx context.Context,
    taskType string,
    payload interface{},
    opts ...asynq.Option,
) (*asynq.TaskInfo, error) {
    data, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("queue.Enqueue: marshal: %w", err)
    }

    task := asynq.NewTask(taskType, data)
    info, err := c.client.EnqueueContext(ctx, task, opts...)
    if err != nil {
        return nil, fmt.Errorf("queue.Enqueue: %w", err)
    }

    return info, nil
}

// EnqueueCritical adds a high priority task
func (c *Client) EnqueueCritical(
    ctx context.Context,
    taskType string,
    payload interface{},
) (*asynq.TaskInfo, error) {
    return c.Enqueue(
        ctx, taskType, payload,
        asynq.Queue("critical"),
        asynq.MaxRetry(3),
    )
}

// EnqueueDefault adds a default priority task
func (c *Client) EnqueueDefault(
    ctx context.Context,
    taskType string,
    payload interface{},
) (*asynq.TaskInfo, error) {
    return c.Enqueue(
        ctx, taskType, payload,
        asynq.Queue("default"),
        asynq.MaxRetry(3),
    )
}

// Close closes the queue client
func (c *Client) Close() error {
    return c.client.Close()
}
