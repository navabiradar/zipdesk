package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/zipdesk/backend/pkg/queue"
	"go.uber.org/zap"
)

const defaultTimeout = 5 * time.Minute

// EventHandler handles an event (primary signature)
type EventHandler func(ctx context.Context, event *Event) error

// ByteHandler handles a raw byte payload (backward-compat signature)
type ByteHandler func(ctx context.Context, payload []byte) error

// EventBus is the core event bus that connects all services
type EventBus struct {
	queue   *queue.Client
	storage *Storage
	repo    *Repository
	engine  *Engine
	logger  *zap.Logger

	mu          sync.RWMutex
	subscribers map[EventType][]EventHandler
}

// NewEventBus creates a new event bus (storage + engine variant)
func NewEventBus(q *queue.Client, storage *Storage, engine *Engine, logger *zap.Logger) *EventBus {
	return &EventBus{
		queue:       q,
		storage:     storage,
		engine:      engine,
		logger:      logger,
		subscribers: make(map[EventType][]EventHandler),
	}
}

// NewEventBusFromRepo creates a new event bus (repo variant)
func NewEventBusFromRepo(
	q *queue.Client,
	repo *Repository,
	logger *zap.Logger,
) *EventBus {
	return &EventBus{
		queue:       q,
		repo:        repo,
		logger:      logger,
		subscribers: make(map[EventType][]EventHandler),
	}
}

// Publish publishes an event synchronously (persists to DB and processes)
func (b *EventBus) Publish(ctx context.Context, event *Event) error {
	b.logger.Info("publishing event",
		zap.String("type", event.Type),
		zap.String("workspace_id", event.WorkspaceID),
		zap.String("source", event.Source),
	)

	// Persist event to database via storage
	if b.storage != nil {
		if err := b.storage.CreateEvent(ctx, event); err != nil {
			b.logger.Error("failed to persist event via storage", zap.Error(err))
			return fmt.Errorf("eventbus.Publish: store event: %w", err)
		}
	} else if b.repo != nil {
		// Fallback: persist via repository
		if err := b.repo.InsertEvent(ctx, event); err != nil {
			b.logger.Warn("failed to persist event via repo", zap.Error(err))
		}
	}

	// Enqueue for background processing if queue is available
	if b.queue != nil {
		go func() {
			c, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()
			_, err := b.queue.Enqueue(c, "flow:event", event)
			if err != nil {
				b.logger.Error("failed to enqueue event", zap.Error(err))
			}
		}()
	}

	// Trigger any matching blueprints (requires engine+storage)
	if b.engine != nil && b.storage != nil {
		go b.triggerBlueprints(ctx, event)
	}

	// Notify subscribers (Synchronously for testing)
	b.notifySubscribers(ctx, event)

	return nil
}

// PublishAsync publishes any event-like object asynchronously via the event bus
func (b *EventBus) PublishAsync(event interface{}) {
	b.logger.Debug("publishing async event")

	// Best-effort: try to cast to Event directly
	if evt, ok := event.(*Event); ok {
		c, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		if err := b.Publish(c, evt); err != nil {
			b.logger.Error("async publish failed", zap.Error(err))
		}
		return
	}

	// Fallback for typed event structs: build a generic event
	evt := eventToGenericEvent(event)
	if evt != nil {
		c, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		if err := b.Publish(c, evt); err != nil {
			b.logger.Error("async publish failed", zap.Error(err))
		}
	}
}

// eventToGenericEvent best-effort converts typed events to generic Event
func eventToGenericEvent(event interface{}) *Event {
	switch e := event.(type) {
	case *Event:
		return e
	case Event:
		return &e
	case *FormSubmittedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"form_id": e.FormID, "form_name": e.FormName, "response_id": e.ResponseID, "email": e.Email, "name": e.Name, "data": e.Data}}
	case FormSubmittedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"form_id": e.FormID, "form_name": e.FormName, "response_id": e.ResponseID, "email": e.Email, "name": e.Name, "data": e.Data}}
	case *LinkCreatedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"link_id": e.LinkID, "short_code": e.ShortCode}}
	case LinkCreatedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"link_id": e.LinkID, "short_code": e.ShortCode}}
	case *LinkClickedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"link_id": e.LinkID, "short_code": e.ShortCode, "country": e.Country, "device": e.Device}}
	case LinkClickedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"link_id": e.LinkID, "short_code": e.ShortCode, "country": e.Country, "device": e.Device}}
	case *DocViewedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"doc_id": e.DocID}}
	case DocViewedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"doc_id": e.DocID}}
	case *DocPublishedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"doc_id": e.DocID, "doc_title": e.DocTitle, "doc_slug": e.DocSlug}}
	case DocPublishedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"doc_id": e.DocID, "doc_title": e.DocTitle, "doc_slug": e.DocSlug}}
	case *MailContactAddedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.BaseEvent.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"contact_id": e.ContactID, "email": e.Email, "contact_source": e.Source}}
	case MailContactAddedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.BaseEvent.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"contact_id": e.ContactID, "email": e.Email, "contact_source": e.Source}}
	case *CRMContactCreatedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"contact_id": e.ContactID, "email": e.Email}}
	case CRMContactCreatedEvent:
		return &Event{ID: e.ID, WorkspaceID: e.WorkspaceID, Type: e.Type, Source: e.Source, OccurredAt: e.OccurredAt, Payload: map[string]any{"contact_id": e.ContactID, "email": e.Email}}
	default:
		return nil
	}
}

// triggerBlueprints finds and triggers blueprints that match the event type
func (b *EventBus) triggerBlueprints(ctx context.Context, event *Event) {
	if event == nil || event.WorkspaceID == "" {
		b.logger.Warn("triggerBlueprints: invalid event or missing workspace_id")
		return
	}

	// Find matching blueprints
	blueprints, err := b.storage.GetBlueprintsForTrigger(ctx, event.WorkspaceID, event.Type)
	if err != nil {
		b.logger.Error("failed to get blueprints for trigger", zap.Error(err))
		return
	}

	if len(blueprints) == 0 {
		b.logger.Debug("no blueprints found for event",
			zap.String("type", event.Type),
			zap.String("workspace_id", event.WorkspaceID),
		)
		return
	}

	b.logger.Info("triggering blueprints",
		zap.String("event_type", event.Type),
		zap.Int("blueprint_count", len(blueprints)),
	)

	for _, bp := range blueprints {
		go func(blueprint FlowBlueprint) {
			c, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			defer cancel()

			// Run blueprint asynchronously
			if err := b.engine.ExecuteBlueprint(nil, c, &blueprint, event); err != nil {
				b.logger.Error("blueprint execution failed",
					zap.String("blueprint_id", blueprint.ID),
					zap.Error(err),
				)
			}
		}(bp)
	}
}

// notifySubscribers calls all registered handlers for an event type
func (b *EventBus) notifySubscribers(ctx context.Context, event *Event) {
	b.mu.RLock()
	handlers := b.subscribers[EventType(event.Type)]
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	b.logger.Debug("notifying subscribers", zap.String("type", event.Type), zap.Int("count", len(handlers)))

	for _, handler := range handlers {
		c, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		if err := handler(c, event); err != nil {
			b.logger.Error("event handler error", zap.Error(err))
		}
		cancel()
	}
}

// Subscribe adds a handler for an event type
func (b *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
	b.mu.Unlock()

	b.logger.Info("subscribed to event type", zap.String("type", eventType))
}

// SubscribeHandler registers a byte-payload-based handler for an event type
func (b *EventBus) SubscribeHandler(eventType EventType, handler ByteHandler) {
	wrapped := func(ctx context.Context, event *Event) error {
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("SubscribeHandler: marshal payload: %w", err)
		}
		return handler(ctx, payload)
	}
	b.Subscribe(eventType, wrapped)
}

// Unsubscribe removes all handlers for an event type (for cleanup)
func (b *EventBus) Unsubscribe(eventType EventType) {
	b.mu.Lock()
	delete(b.subscribers, eventType)
	b.mu.Unlock()

	b.logger.Info("unsubscribed from event type", zap.String("type", eventType))
}
