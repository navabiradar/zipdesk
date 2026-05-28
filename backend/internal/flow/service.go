package flow

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/cache"
)

// Service handles flow business logic
type Service struct {
	repo     *Repository
	eventBus *EventBus
	executor *Executor
	monitor  *HealthMonitor
	mailSvc  interface{}
	redis    *cache.Client
	logger   *zap.Logger
}

// NewService creates a new flow service
func NewService(
	repo *Repository,
	eventBus *EventBus,
	mailSvc interface{},
	redis *cache.Client,
	logger *zap.Logger,
) *Service {
	executor := NewExecutor(
		repo, mailSvc, eventBus, logger,
	)

	monitor := NewHealthMonitor(redis, nil, eventBus, logger)

	allTriggers := []EventType{
		EventFormSubmitted,
		EventFormPublished,
		EventMailContactAdded,
		EventLinkClicked,
		EventDocViewed,
		EventCRMContactCreated,
	}

	for _, triggerType := range allTriggers {
		tt := triggerType
		eventBus.Subscribe(tt, func(ctx context.Context, event *Event) error {
			return executor.RunBlueprintsForEvent(
				ctx,
				event.WorkspaceID,
				tt,
				event.ID,
				event.Payload,
			)
		})
	}

	return &Service{
		repo:     repo,
		eventBus: eventBus,
		executor: executor,
		monitor:  monitor,
		mailSvc:  mailSvc,
		redis:    redis,
		logger:   logger,
	}
}

// CreateBlueprint creates a new automation
func (s *Service) CreateBlueprint(
	ctx context.Context,
	workspaceID string,
	input CreateBlueprintInput,
) (*FlowBlueprint, error) {
	bp := &FlowBlueprint{
		WorkspaceID:   workspaceID,
		Name:          input.Name,
		Description:   input.Description,
		TriggerType:   input.TriggerType,
		TriggerConfig: input.TriggerConfig,
		Actions:       input.Actions,
		IsActive:      true,
	}

	if bp.Actions == nil {
		bp.Actions = []FlowAction{}
	}
	if bp.TriggerConfig == nil {
		bp.TriggerConfig = map[string]any{}
	}

	if err := s.repo.CreateBlueprint(ctx, bp); err != nil {
		return nil, fmt.Errorf(
			"flow.Service.CreateBlueprint: %w", err,
		)
	}

	return bp, nil
}

// GetHealthReport returns current health status
func (s *Service) GetHealthReport(
	ctx context.Context,
) *HealthReport {
	if s.monitor != nil {
		return s.monitor.CheckAll(ctx)
	}

	return &HealthReport{
		Timestamp: time.Now(),
		Services:  map[string]ServiceHealth{},
		Overall:   "unknown",
	}
}

// InitWorkspace sets up default blueprints for a new workspace
func (s *Service) InitWorkspace(
	ctx context.Context,
	workspaceID string,
) error {
	return s.repo.InitDefaultBlueprints(ctx, workspaceID)
}

// ChatStream streams an AI chat response to the client via SSE
func (s *Service) ChatStream(
	ctx context.Context,
	w *bufio.Writer,
	workspaceID string,
	userID string,
	input ChatInput,
) {
	s.streamChat(ctx, w, workspaceID, userID, input)
}

// ListEvents returns recent events for a workspace
func (s *Service) ListEvents(ctx context.Context, workspaceID string, limit int) ([]Event, error) {
	return s.repo.ListEvents(ctx, workspaceID, limit)
}

// ListBlueprints returns all blueprints for a workspace
func (s *Service) ListBlueprints(ctx context.Context, workspaceID string) ([]FlowBlueprint, error) {
	return s.repo.ListBlueprints(ctx, workspaceID)
}

// DeleteBlueprint removes a blueprint scoped to workspace
func (s *Service) DeleteBlueprint(ctx context.Context, id string, workspaceID string) error {
	return s.repo.DeleteBlueprint(ctx, id, workspaceID)
}

// GetConversationMessages returns messages for a conversation
func (s *Service) GetConversationMessages(ctx context.Context, conversationID string) ([]AIMessage, error) {
	return s.repo.GetConversationMessages(ctx, conversationID)
}

// SetMonitor sets the health monitor on the service
func (s *Service) SetMonitor(m *HealthMonitor) {
	s.monitor = m
}

// ==================== HELPERS ====================

func unmarshalBase(payload []byte, base *BaseEvent) error {
	return json.Unmarshal(payload, base)
}

func unmarshalMap(payload []byte, m *map[string]any) error {
	return json.Unmarshal(payload, m)
}
