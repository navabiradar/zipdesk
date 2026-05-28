package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// Repository handles flow database operations
type Repository struct {
	db *bun.DB
}

// NewRepository creates a new flow repository
func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

// InsertEvent persists an event to DB
func (r *Repository) InsertEvent(
	ctx context.Context,
	event *Event,
) error {
	_, err := r.db.NewInsert().
		Model(event).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"flow.Repository.InsertEvent: %w", err,
		)
	}
	return nil
}

// ListEvents returns recent events for workspace
func (r *Repository) ListEvents(
	ctx context.Context,
	workspaceID string,
	limit int,
) ([]Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var events []Event
	err := r.db.NewSelect().
		Model(&events).
		Where("workspace_id = ?", workspaceID).
		OrderExpr("occurred_at DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"flow.Repository.ListEvents: %w", err,
		)
	}
	return events, nil
}

// GetEventsByType returns events filtered by type
func (r *Repository) GetEventsByType(
	ctx context.Context,
	workspaceID string,
	eventType EventType,
	limit int,
) ([]Event, error) {
	var events []Event
	err := r.db.NewSelect().
		Model(&events).
		Where("workspace_id = ? AND type = ?",
			workspaceID, eventType).
		OrderExpr("occurred_at DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"flow.Repository.GetEventsByType: %w", err,
		)
	}
	return events, nil
}

// MarkEventProcessed marks event as processed
func (r *Repository) MarkEventProcessed(
	ctx context.Context,
	id string,
	processErr string,
) error {
	now := time.Now()
	q := r.db.NewUpdate().
		TableExpr("events").
		Set("processed_at = ?", now).
		Where("id = ?", id)

	if processErr != "" {
		q = q.Set("error = ?", processErr)
	}

	_, err := q.Exec(ctx)
	return err
}

// CreateBlueprint inserts a new blueprint
func (r *Repository) CreateBlueprint(
	ctx context.Context,
	bp *FlowBlueprint,
) error {
	_, err := r.db.NewInsert().
		Model(bp).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(
			"flow.Repository.CreateBlueprint: %w", err,
		)
	}
	return nil
}

// GetBlueprint retrieves a blueprint by ID
func (r *Repository) GetBlueprint(
	ctx context.Context,
	id string,
) (*FlowBlueprint, error) {
	bp := &FlowBlueprint{}
	err := r.db.NewSelect().Model(bp).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Repository.GetBlueprint: %w", err)
	}
	return bp, nil
}

// ListBlueprints returns all blueprints for workspace
func (r *Repository) ListBlueprints(
	ctx context.Context,
	workspaceID string,
) ([]FlowBlueprint, error) {
	var blueprints []FlowBlueprint
	err := r.db.NewSelect().
		Model(&blueprints).
		Where("workspace_id = ?", workspaceID).
		OrderExpr("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"flow.Repository.ListBlueprints: %w", err,
		)
	}
	return blueprints, nil
}

// GetActiveBlueprintsForTrigger finds matching blueprints
func (r *Repository) GetActiveBlueprintsForTrigger(
	ctx context.Context,
	workspaceID string,
	triggerType EventType,
) ([]FlowBlueprint, error) {
	var blueprints []FlowBlueprint
	err := r.db.NewSelect().
		Model(&blueprints).
		Where(
			"workspace_id = ? AND trigger_type = ? AND is_active = true",
			workspaceID, triggerType,
		).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"flow.Repository.GetActiveBlueprintsForTrigger: %w",
			err,
		)
	}
	return blueprints, nil
}

// UpdateBlueprintRunCount increments run counter
func (r *Repository) UpdateBlueprintRunCount(
	ctx context.Context,
	id string,
) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		TableExpr("flow_blueprints").
		Set("run_count = run_count + 1").
		Set("last_run_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// UpdateBlueprint updates a blueprint
func (r *Repository) UpdateBlueprint(
	ctx context.Context,
	bp *FlowBlueprint,
) error {
	_, err := r.db.NewUpdate().Model(bp).Where("id = ?", bp.ID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("flow.Repository.UpdateBlueprint: %w", err)
	}
	return nil
}

// DeleteBlueprint removes a blueprint
func (r *Repository) DeleteBlueprint(
	ctx context.Context,
	id string,
	workspaceID string,
) error {
	_, err := r.db.NewDelete().
		TableExpr("flow_blueprints").
		Where("id = ? AND workspace_id = ?",
			id, workspaceID).
		Exec(ctx)
	return err
}

// CreateExecution logs a blueprint execution
func (r *Repository) CreateExecution(
	ctx context.Context,
	exec *BlueprintExecution,
) error {
	_, err := r.db.NewInsert().
		Model(exec).
		Exec(ctx)
	return err
}

// GetExecution retrieves an execution by ID
func (r *Repository) GetExecution(
	ctx context.Context,
	id string,
) (*BlueprintExecution, error) {
	exec := &BlueprintExecution{}
	err := r.db.NewSelect().Model(exec).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("flow.Repository.GetExecution: %w", err)
	}
	return exec, nil
}

// UpdateExecution updates execution status
func (r *Repository) UpdateExecution(
	ctx context.Context,
	exec *BlueprintExecution,
) error {
	now := time.Now()
	exec.CompletedAt = &now
	_, err := r.db.NewUpdate().
		Model(exec).
		Where("id = ?", exec.ID).
		Exec(ctx)
	return err
}

// CreateConversation creates a new AI conversation
func (r *Repository) CreateConversation(
	ctx context.Context,
	conv *AIConversation,
) error {
	_, err := r.db.NewInsert().
		Model(conv).
		Exec(ctx)
	return err
}

// GetConversation finds conversation by ID
func (r *Repository) GetConversation(
	ctx context.Context,
	id string,
	workspaceID string,
) (*AIConversation, error) {
	conv := new(AIConversation)
	err := r.db.NewSelect().
		Model(conv).
		Where("id = ? AND workspace_id = ?",
			id, workspaceID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"flow.Repository.GetConversation: %w", err,
		)
	}
	return conv, nil
}

// SaveMessage saves an AI message
func (r *Repository) SaveMessage(
	ctx context.Context,
	msg *AIMessage,
) error {
	_, err := r.db.NewInsert().
		Model(msg).
		Exec(ctx)
	return err
}

// GetConversationMessages returns messages in order
func (r *Repository) GetConversationMessages(
	ctx context.Context,
	conversationID string,
) ([]AIMessage, error) {
	var messages []AIMessage
	err := r.db.NewSelect().
		Model(&messages).
		Where("conversation_id = ?", conversationID).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"flow.Repository.GetConversationMessages: %w",
			err,
		)
	}
	return messages, nil
}

// InitDefaultBlueprints creates default automations
func (r *Repository) InitDefaultBlueprints(
	ctx context.Context,
	workspaceID string,
) error {
	blueprints := []FlowBlueprint{
		{
			WorkspaceID: workspaceID,
			Name:        "Form → Add to Mail List",
			Description: "Auto-add form submitters to mail contacts",
			TriggerType: EventFormSubmitted,
			TriggerConfig: map[string]any{
				"require_email": true,
			},
			Actions: []FlowAction{
				{
					ID:   "action-1",
					Type: ActionAddMailContact,
					Config: map[string]any{
						"source": "form",
						"tags":   []string{"form-lead"},
					},
					Order: 1,
				},
			},
			IsActive: true,
		},
		{
			WorkspaceID: workspaceID,
			Name:        "Form → Notify Owner",
			Description: "Send notification on new form response",
			TriggerType: EventFormSubmitted,
			Actions: []FlowAction{
				{
					ID:   "action-1",
					Type: ActionSendNotification,
					Config: map[string]any{
						"channel": "email",
						"message": "New form response received",
					},
					Order: 1,
				},
			},
			IsActive: true,
		},
	}

	for i := range blueprints {
		exists, err := r.blueprintExists(
			ctx, workspaceID,
			blueprints[i].Name,
		)
		if err != nil || exists {
			continue
		}
		if err := r.CreateBlueprint(
			ctx, &blueprints[i],
		); err != nil {
			return fmt.Errorf(
				"flow.Repository.InitDefaultBlueprints: %w",
				err,
			)
		}
	}

	return nil
}

// blueprintExists checks if blueprint name is already taken
func (r *Repository) blueprintExists(
	ctx context.Context,
	workspaceID string,
	name string,
) (bool, error) {
	count, err := r.db.NewSelect().
		TableExpr("flow_blueprints").
		Where("workspace_id = ? AND name = ?",
			workspaceID, name).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
