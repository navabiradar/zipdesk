package flow

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Executor runs blueprint actions in response to events
type Executor struct {
	repo     *Repository
	mailSvc  interface{}
	bus      *EventBus
	registry *ActionRegistry
	logger   *zap.Logger
}

// NewExecutor creates a new blueprint executor
func NewExecutor(repo *Repository, mailSvc interface{}, bus *EventBus, logger *zap.Logger) *Executor {
	reg := NewActionRegistry()
	actions := NewActions(logger, mailSvc)
	actions.RegisterBuiltin(reg)

	return &Executor{
		repo:     repo,
		mailSvc:  mailSvc,
		bus:      bus,
		registry: reg,
		logger:   logger,
	}
}

// RunBlueprintsForEvent finds active blueprints matching the trigger and executes them
func (e *Executor) RunBlueprintsForEvent(
	ctx context.Context,
	workspaceID string,
	triggerType EventType,
	eventID string,
	payload map[string]any,
) error {
	blueprints, err := e.repo.GetActiveBlueprintsForTrigger(ctx, workspaceID, triggerType)
	if err != nil {
		return fmt.Errorf("executor: lookup blueprints: %w", err)
	}

	for _, bp := range blueprints {
		e.logger.Info("executing blueprint",
			zap.String("blueprint_id", bp.ID),
			zap.String("trigger", string(triggerType)),
		)

		exec := &BlueprintExecution{
			ID:          GenerateExecutionID(),
			BlueprintID: bp.ID,
			WorkspaceID: workspaceID,
			EventID:     eventID,
			Status:      "running",
			ActionsRun:  0,
		}

		if err := e.repo.CreateExecution(ctx, exec); err != nil {
			e.logger.Error("failed to create execution", zap.Error(err))
			continue
		}

		err := e.runActions(ctx, exec, bp.Actions, payload)
		if err != nil {
			e.logger.Error("blueprint execution failed", zap.String("execution_id", exec.ID), zap.Error(err))
			exec.Status = "failed"
			e.repo.UpdateExecution(ctx, exec)
		} else {
			exec.Status = "completed"
			exec.ActionsRun = len(bp.Actions)
			e.repo.UpdateExecution(ctx, exec)
		}

		e.repo.UpdateBlueprintRunCount(ctx, bp.ID)
	}

	return nil
}

func (e *Executor) runActions(
	ctx context.Context,
	exec *BlueprintExecution,
	actions []FlowAction,
	payload map[string]any,
) error {
	for _, action := range actions {
		e.logger.Info("running action",
			zap.String("action_id", action.ID),
			zap.String("type", action.Type),
		)

		fn, err := e.registry.Get(action.Type)
		if err != nil {
			e.logger.Error("action not found", zap.String("type", string(action.Type)), zap.Error(err))
			continue
		}

		_, err = fn(nil, ctx, payload, action.Config)
		if err != nil {
			e.logger.Error("action failed",
				zap.String("action_id", action.ID),
				zap.String("type", string(action.Type)),
				zap.Error(err),
			)
			return fmt.Errorf("executor.runActions: action %s: %w", action.Type, err)
		}
	}
	return nil
}
