package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Storage handles all database operations for the flow package
type Storage struct {
	db *bun.DB
}

// NewStorage creates a new flow storage instance
func NewStorage(db *bun.DB) *Storage {
	return &Storage{db: db}
}

// ==================== EVENTS ====================

// CreateEvent persists an event to the database
func (s *Storage) CreateEvent(ctx context.Context, event *Event) error {
	_, err := s.db.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.CreateEvent: %w", err)
	}
	return nil
}

// GetEvent retrieves an event by ID
func (s *Storage) GetEvent(ctx context.Context, id string) (*Event, error) {
	event := &Event{}
	err := s.db.NewSelect().Model(event).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetEvent: %w", err)
	}
	return event, nil
}

// ListEvents retrieves events for a workspace with optional filtering
func (s *Storage) ListEvents(ctx context.Context, workspaceID string, types []EventType, limit, offset int) ([]Event, error) {
	var events []Event
	q := s.db.NewSelect().Model(&events).Where("workspace_id = ?", workspaceID).Order("occurred_at DESC")
	if len(types) > 0 {
		q = q.Where("type IN (?)", bun.In(types))
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.ListEvents: %w", err)
	}
	return events, nil
}

// MarkEventProcessed marks an event as processed
func (s *Storage) MarkEventProcessed(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.NewUpdate().Model((*Event)(nil)).Set("processed_at = ?", now).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.MarkEventProcessed: %w", err)
	}
	return nil
}

// GetUnprocessedEvents returns unprocessed events for a given type
func (s *Storage) GetUnprocessedEvents(ctx context.Context, eventType EventType, limit int) ([]Event, error) {
	var events []Event
	q := s.db.NewSelect().Model(&events).Where("type = ?", eventType).Where("processed_at IS NULL").Order("occurred_at ASC").Limit(limit)
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetUnprocessedEvents: %w", err)
	}
	return events, nil
}

// ==================== BLUEPRINTS ====================

// CreateBlueprint persists a new flow blueprint
func (s *Storage) CreateBlueprint(ctx context.Context, bp *FlowBlueprint) error {
	_, err := s.db.NewInsert().Model(bp).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.CreateBlueprint: %w", err)
	}
	return nil
}

// GetBlueprint retrieves a blueprint by ID
func (s *Storage) GetBlueprint(ctx context.Context, id string) (*FlowBlueprint, error) {
	bp := &FlowBlueprint{}
	err := s.db.NewSelect().Model(bp).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetBlueprint: %w", err)
	}
	return bp, nil
}

// GetBlueprintsForTrigger returns blueprints that match a trigger type
func (s *Storage) GetBlueprintsForTrigger(ctx context.Context, workspaceID string, triggerType EventType) ([]FlowBlueprint, error) {
	var bps []FlowBlueprint
	err := s.db.NewSelect().
		Model(&bps).
		Where("workspace_id = ?", workspaceID).
		Where("trigger_type = ?", triggerType).
		Where("is_active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetBlueprintsForTrigger: %w", err)
	}
	return bps, nil
}

// ListBlueprints returns all blueprints for a workspace
func (s *Storage) ListBlueprints(ctx context.Context, workspaceID string, limit, offset int) ([]FlowBlueprint, error) {
	var bps []FlowBlueprint
	q := s.db.NewSelect().Model(&bps).Where("workspace_id = ?", workspaceID).Order("created_at DESC").Limit(limit).Offset(offset)
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.ListBlueprints: %w", err)
	}
	return bps, nil
}

// UpdateBlueprint updates a blueprint
func (s *Storage) UpdateBlueprint(ctx context.Context, bp *FlowBlueprint) error {
	bp.UpdatedAt = time.Now()
	_, err := s.db.NewUpdate().Model(bp).Where("id = ?", bp.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.UpdateBlueprint: %w", err)
	}
	return nil
}

// DeleteBlueprint deletes a blueprint
func (s *Storage) DeleteBlueprint(ctx context.Context, id string) error {
	_, err := s.db.NewDelete().Model((*FlowBlueprint)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.DeleteBlueprint: %w", err)
	}
	return nil
}

// IncrementRunCount increments the run count and sets last run time
func (s *Storage) IncrementRunCount(ctx context.Context, id string) error {
	now := time.Now()
	_, err := s.db.NewUpdate().Model((*FlowBlueprint)(nil)).
		Set("run_count = run_count + 1").
		Set("last_run_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.IncrementRunCount: %w", err)
	}
	return nil
}

// ==================== EXECUTIONS ====================

// CreateExecution persists a blueprint execution
func (s *Storage) CreateExecution(ctx context.Context, exec *BlueprintExecution) error {
	_, err := s.db.NewInsert().Model(exec).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.CreateExecution: %w", err)
	}
	return nil
}

// GetExecution retrieves an execution by ID
func (s *Storage) GetExecution(ctx context.Context, id string) (*BlueprintExecution, error) {
	exec := &BlueprintExecution{}
	err := s.db.NewSelect().Model(exec).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetExecution: %w", err)
	}
	return exec, nil
}

// ListExecutions returns executions for a blueprint
func (s *Storage) ListExecutions(ctx context.Context, blueprintID string, limit, offset int) ([]BlueprintExecution, error) {
	var execs []BlueprintExecution
	q := s.db.NewSelect().Model(&execs).Where("blueprint_id = ?", blueprintID).Order("started_at DESC").Limit(limit).Offset(offset)
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.ListExecutions: %w", err)
	}
	return execs, nil
}

// UpdateExecution updates an execution
func (s *Storage) UpdateExecution(ctx context.Context, exec *BlueprintExecution) error {
	_, err := s.db.NewUpdate().Model(exec).Where("id = ?", exec.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.UpdateExecution: %w", err)
	}
	return nil
}

// CompleteExecution marks an execution as completed
func (s *Storage) CompleteExecution(ctx context.Context, id string, status string, actionsRun, actionsFailed int, errMsg string) error {
	now := time.Now()
	exec := s.db.NewUpdate().Model((*BlueprintExecution)(nil)).
		Set("status = ?", status).
		Set("actions_run = ?", actionsRun).
		Set("actions_failed = ?", actionsFailed).
		Set("completed_at = ?", now)

	if errMsg != "" {
		exec = exec.Set("error = ?", errMsg)
	}

	_, err := exec.Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.CompleteExecution: %w", err)
	}
	return nil
}

// ==================== AI CONVERSATIONS ====================

// CreateConversation persists a new AI conversation
func (s *Storage) CreateConversation(ctx context.Context, conv *AIConversation) error {
	_, err := s.db.NewInsert().Model(conv).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.CreateConversation: %w", err)
	}
	return nil
}

// GetConversation retrieves a conversation by ID
func (s *Storage) GetConversation(ctx context.Context, id string) (*AIConversation, error) {
	conv := &AIConversation{}
	err := s.db.NewSelect().Model(conv).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetConversation: %w", err)
	}
	return conv, nil
}

// ListConversations returns conversations for a workspace
func (s *Storage) ListConversations(ctx context.Context, workspaceID, userID string, limit, offset int) ([]AIConversation, error) {
	var convs []AIConversation
	q := s.db.NewSelect().Model(&convs).Where("workspace_id = ?", workspaceID).Where("user_id = ?", userID).Order("updated_at DESC").Limit(limit).Offset(offset)
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.ListConversations: %w", err)
	}
	return convs, nil
}

// UpdateConversation updates a conversation title and message count
func (s *Storage) UpdateConversation(ctx context.Context, conv *AIConversation) error {
	conv.UpdatedAt = time.Now()
	_, err := s.db.NewUpdate().Model(conv).Where("id = ?", conv.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.UpdateConversation: %w", err)
	}
	return nil
}

// DeleteConversation deletes a conversation
func (s *Storage) DeleteConversation(ctx context.Context, id string) error {
	_, err := s.db.NewDelete().Model((*AIConversation)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.DeleteConversation: %w", err)
	}
	return nil
}

// CreateMessage persists an AI message
func (s *Storage) CreateMessage(ctx context.Context, msg *AIMessage) error {
	_, err := s.db.NewInsert().Model(msg).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Storage.CreateMessage: %w", err)
	}
	return nil
}

// GetMessages returns messages for a conversation
func (s *Storage) GetMessages(ctx context.Context, conversationID string, limit int, offset int) ([]AIMessage, error) {
	var messages []AIMessage
	q := s.db.NewSelect().Model(&messages).Where("conversation_id = ?", conversationID).Order("created_at ASC").Limit(limit).Offset(offset)
	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Storage.GetMessages: %w", err)
	}
	return messages, nil
}
