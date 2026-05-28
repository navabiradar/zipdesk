package flow

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
)

// MailServiceInterface defines mail operations used by system triggers
type MailServiceInterface interface {
	UpsertContact(
		ctx context.Context,
		workspaceID string,
		email string,
		data map[string]any,
	) (string, error)
	UnsubscribeContact(
		ctx context.Context,
		workspaceID string,
		email string,
	) error
}

// CRMServiceInterface defines CRM operations
type CRMServiceInterface interface {
	CreateContactFromEvent(
		ctx context.Context,
		workspaceID string,
		email string,
		name string,
		source string,
	) (string, error)
}

// SystemTriggers handles built-in integrations.
// These run on EVERY matching event, regardless of user blueprints.
type SystemTriggers struct {
	mailSvc MailServiceInterface
	crmSvc  CRMServiceInterface
	logger  *zap.Logger
}

// NewSystemTriggers creates system triggers
func NewSystemTriggers(
	mailSvc interface{},
	crmSvc interface{},
	logger *zap.Logger,
) *SystemTriggers {
	var msi MailServiceInterface
	var crm CRMServiceInterface
	if mailSvc != nil {
		if cast, ok := mailSvc.(MailServiceInterface); ok {
			msi = cast
		}
	}
	if crmSvc != nil {
		if cast, ok := crmSvc.(CRMServiceInterface); ok {
			crm = cast
		}
	}
	return &SystemTriggers{
		mailSvc: msi,
		crmSvc:  crm,
		logger:  logger,
	}
}

// Register wires all system triggers to bus
func (t *SystemTriggers) Register(bus *EventBus) {
	// Form submitted → add mail contact
	bus.Subscribe(EventFormSubmitted, t.onFormSubmitted)

	// Mail unsubscribed → update contact
	bus.Subscribe(EventMailUnsubscribed, t.onMailUnsubscribed)

	// Log all events for debugging
	allEvents := []EventType{
		EventFormSubmitted,
		EventFormPublished,
		EventMailContactAdded,
		EventLinkClicked,
		EventLinkCreated,
		EventDocViewed,
		EventDocPublished,
		EventCRMContactCreated,
		EventHealthCheck,
	}

	for _, eventType := range allEvents {
		et := eventType
		bus.Subscribe(et, t.logEvent(et))
	}
}

// onFormSubmitted handles form.submitted event.
// CRITICAL: This is the core integration.
// Form submit → Mail contact upserted.
func (t *SystemTriggers) onFormSubmitted(
	ctx context.Context,
	event *Event,
) error {
	t.logger.Debug("onFormSubmitted",
		zap.String("workspace_id", event.WorkspaceID),
	)

	var fs FormSubmittedEvent
	payload, _ := json.Marshal(event.Payload)
	if err := json.Unmarshal(payload, &fs); err != nil {
		return err
	}

	if fs.Email == "" {
		t.logger.Debug("onFormSubmitted: email empty, skipping")
		return nil
	}

	// Upsert mail contact
	if t.mailSvc != nil {
		contactData := map[string]any{
			"source":    "form",
			"form_id":   fs.FormID,
			"form_name": fs.FormName,
			"name":      fs.Name,
		}

		contactID, err := t.mailSvc.UpsertContact(
			ctx,
			event.WorkspaceID,
			fs.Email,
			contactData,
		)
		if err != nil {
			t.logger.Warn(
				"failed to upsert mail contact from form",
				zap.String("form_id", fs.FormID),
				zap.String("email", fs.Email),
				zap.Error(err),
			)
		} else {
			t.logger.Debug(
				"mail contact upserted from form",
				zap.String("contact_id", contactID),
				zap.String("email", fs.Email),
				zap.String("form_id", fs.FormID),
			)
		}
	}

	// Create CRM contact if service available
	if t.crmSvc != nil {
		_, err := t.crmSvc.CreateContactFromEvent(
			ctx,
			event.WorkspaceID,
			fs.Email,
			fs.Name,
			"form:"+fs.FormID,
		)
		if err != nil {
			t.logger.Warn(
				"failed to create CRM contact from form",
				zap.String("email", fs.Email),
				zap.Error(err),
			)
		}
	}

	return nil
}

// onMailUnsubscribed handles unsubscribe event
func (t *SystemTriggers) onMailUnsubscribed(
	ctx context.Context,
	event *Event,
) error {
	var mu MailUnsubscribedEvent
	payload, _ := json.Marshal(event.Payload)
	if err := json.Unmarshal(payload, &mu); err != nil {
		return err
	}

	if t.mailSvc != nil && mu.Email != "" {
		if err := t.mailSvc.UnsubscribeContact(
			ctx,
			event.WorkspaceID,
			mu.Email,
		); err != nil {
			t.logger.Warn(
				"failed to unsubscribe contact",
				zap.String("email", mu.Email),
				zap.Error(err),
			)
		}
	}

	return nil
}

// logEvent returns a handler that logs the event
func (t *SystemTriggers) logEvent(
	eventType EventType,
) EventHandler {
	return func(
		ctx context.Context,
		event *Event,
	) error {
		t.logger.Debug("event received",
			zap.String("type", eventType),
		)
		return nil
	}
}