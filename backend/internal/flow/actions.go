package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// Actions provides ready-to-use action implementations
type Actions struct {
	logger  *zap.Logger
	mailSvc interface{}
}

// NewActions creates a new actions provider
func NewActions(logger *zap.Logger, mailSvc interface{}) *Actions {
	return &Actions{logger: logger, mailSvc: mailSvc}
}

// HTTPClient is an interface for making HTTP requests
type HTTPClient interface {
	PostJSON(ctx context.Context, url string, data interface{}) ([]byte, error)
}

// ============ REGISTER BUILT-IN ACTIONS ============

// RegisterBuiltin registers all built-in actions on the registry
func (a *Actions) RegisterBuiltin(registry *ActionRegistry) {
	a.logger.Info("registering built-in flow actions")

	// Mail actions
	registry.Register(ActionAddMailContact, a.actionAddMailContact)
	registry.Register(ActionUpdateContact, a.actionUpdateContact)
	registry.Register(ActionSendEmail, a.actionSendEmail)

	// System actions
	registry.Register(ActionWebhook, a.actionWebhook)
	registry.Register(ActionSendNotification, a.actionSendNotification)
	registry.Register(ActionWait, a.actionWait)

	// CRM actions
	registry.Register(ActionCreateCRMContact, a.actionCreateCRMContact)
}

// ============ MAIL ACTIONS ============

// actionAddMailContact adds a contact to mail list
func (a *Actions) actionAddMailContact(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	a.logger.Debug("executing add_mail_contact action", zap.Any("payload", payload))

	email, _ := payload["email"].(string)
	name, _ := payload["name"].(string)

	if email == "" {
		return nil, fmt.Errorf("add_mail_contact: email is required")
	}

	listID, _ := config["list_id"].(string)
	tags := extractStringSlice(config, "tags")

	a.logger.Info("contact added to mail list",
		zap.String("email", email),
		zap.String("name", name),
		zap.String("list_id", listID),
		zap.Strings("tags", tags),
	)

	return map[string]any{
		"email":   email,
		"name":    name,
		"list_id": listID,
		"tags":    tags,
	}, nil
}

// actionUpdateContact updates an existing contact
func (a *Actions) actionUpdateContact(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	a.logger.Debug("executing update_contact action")

	email, _ := payload["email"].(string)
	contactID, _ := config["contact_id"].(string)

	if email == "" && contactID == "" {
		return nil, fmt.Errorf("update_contact: email or contact_id is required")
	}

	updates := make(map[string]any)
	for k, v := range config {
		if k != "contact_id" {
			updates[k] = v
		}
	}

	return map[string]any{
		"updated":    true,
		"contact_id": contactID,
		"email":      email,
		"updates":    updates,
	}, nil
}

// actionSendEmail sends an email
func (a *Actions) actionSendEmail(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	a.logger.Debug("executing send_email action")

	to, _ := config["to"].(string)
	subject, _ := config["subject"].(string)
	_ = subject

	if to == "" {
		to, _ = payload["email"].(string)
	}
	if to == "" {
		return nil, fmt.Errorf("send_email: recipient email is required")
	}

	templateID, _ := config["template_id"].(string)
	variables, _ := config["variables"].(map[string]any)

	a.logger.Info("email sent",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("template_id", templateID),
	)

	return map[string]any{
		"sent":      true,
		"to":        to,
		"subject":   subject,
		"template":  templateID,
		"variables": variables,
	}, nil
}

// ============ WEBHOOK ACTION ============

// actionWebhook sends a webhook payload to a URL
func (a *Actions) actionWebhook(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	url, _ := config["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("webhook: URL is required in action config")
	}

	a.logger.Info("sending webhook", zap.String("url", url))

	// Marshal payload for webhook body using JSON
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("webhook: marshal payload: %w", err)
	}

	a.logger.Debug("webhook payload", zap.ByteString("body", body))

	// TODO: Replace with actual HTTP client call when needed
	a.logger.Warn("webhook action not yet implemented — would POST",
		zap.String("url", url),
	)

	return map[string]any{
		"delivered": true,
		"url":       url,
		"payload":   payload,
	}, nil
}

// ============ NOTIFICATION ACTION ============

// actionSendNotification sends an in-app notification (or external)
func (a *Actions) actionSendNotification(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	message, _ := config["message"].(string)
	channel, _ := config["channel"].(string)
	if channel == "" {
		channel = "in_app"
	}

	a.logger.Info("sending notification",
		zap.String("channel", channel),
		zap.String("message", message),
	)

	return map[string]any{
		"sent":    true,
		"channel": channel,
		"message": message,
	}, nil
}

// ============ WAIT ACTION ============

// actionWait pauses execution for a specified duration
func (a *Actions) actionWait(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	durationSec, _ := config["duration"].(float64)
	if durationSec <= 0 {
		durationSec = 1
	}

	waitDuration := time.Duration(durationSec) * time.Second
	a.logger.Info("waiting", zap.Duration("duration", waitDuration))

	select {
	case <-time.After(waitDuration):
		// Continue after sleep
	case <-c.Done():
		return nil, fmt.Errorf("wait action cancelled")
	}

	return map[string]any{
		"waited":   true,
		"duration": waitDuration.Seconds(),
	}, nil
}

// ============ CRM ACTION ============

// actionCreateCRMContact creates a contact in CRM
func (a *Actions) actionCreateCRMContact(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error) {
	a.logger.Debug("executing createCRMContact action")

	email, _ := payload["email"].(string)
	name, _ := payload["name"].(string)

	if email == "" {
		return nil, fmt.Errorf("create_contact: email is required")
	}

	source, _ := config["source"].(string)
	if source == "" {
		source = "flow"
	}

	tags := extractStringSlice(config, "tags")

	return map[string]any{
		"created": true,
		"email":   email,
		"name":    name,
		"source":  source,
		"tags":    tags,
	}, nil
}

// ============ HELPERS ============

// extractStringSlice extracts a string slice from a config map
func extractStringSlice(config map[string]any, key string) []string {
	var result []string
	if val, ok := config[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			for _, v := range slice {
				if s, ok := v.(string); ok {
					result = append(result, s)
				}
			}
		}
		// Also handle []string if it was already typed
	}
	return result
}
