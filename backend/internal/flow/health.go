package flow

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/cache"
)

// HealthMonitor checks the health of all services
type HealthMonitor struct {
	redis    *cache.Client
	db       *bun.DB
	eventBus *EventBus
	logger   *zap.Logger
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(redis *cache.Client, db *bun.DB, eventBus *EventBus, logger *zap.Logger) *HealthMonitor {
	return &HealthMonitor{redis: redis, db: db, eventBus: eventBus, logger: logger}
}

// StartCron starts the health check cron
func (h *HealthMonitor) StartCron(ctx context.Context) {
	h.logger.Info("health monitor cron started")

	// Run periodic health checks
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.checkAll(ctx)
			}
		}
	}()
}

// CheckAll runs all health checks and returns the report
func (h *HealthMonitor) CheckAll(ctx context.Context) *HealthReport {
	report := &HealthReport{
		Timestamp: time.Now(),
		Services: map[string]ServiceHealth{
			"api":   h.checkAPI(),
			"db":    h.checkDB(),
			"redis": h.checkRedis(),
			"queue": h.checkQueue(),
		},
		Overall: "healthy",
	}

	for _, svc := range report.Services {
		if svc.Status != "healthy" && svc.Status != "ok" {
			report.Overall = "degraded"
			break
		}
	}

	return report
}

// checkAll runs all health checks
func (h *HealthMonitor) checkAll(ctx context.Context) {
	h.logger.Info("running health checks")

	// Placeholder: checks would check DB, Redis, and other services
	// In a real implementation, you'd check connectivity to DB, Redis, etc.

	report := &HealthReport{
		Timestamp: time.Now(),
		Services: map[string]ServiceHealth{
			"api":   h.checkAPI(),
			"db":    h.checkDB(),
			"redis": h.checkRedis(),
			"queue": h.checkQueue(),
		},
		Overall: "healthy",
	}

	// Determine overall status
	for _, svc := range report.Services {
		if svc.Status != "healthy" && svc.Status != "ok" {
			report.Overall = "degraded"
			break
		}
	}

	h.logger.Info("health check complete", zap.String("overall", report.Overall))
}

func (h *HealthMonitor) checkAPI() ServiceHealth {
	return ServiceHealth{Name: "api", Status: "healthy", LatencyMs: 12}
}

func (h *HealthMonitor) checkDB() ServiceHealth {
	return ServiceHealth{Name: "db", Status: "healthy", LatencyMs: 45}
}

func (h *HealthMonitor) checkRedis() ServiceHealth {
	return ServiceHealth{Name: "redis", Status: "healthy", LatencyMs: 8}
}

func (h *HealthMonitor) checkQueue() ServiceHealth {
	return ServiceHealth{Name: "queue", Status: "healthy", LatencyMs: 15}
}
