package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// ExecutionContext holds the state for a single blueprint run
type ExecutionContext struct {
	ExecutionID  string         `json:"execution_id"`
	BlueprintID  string         `json:"blueprint_id"`
	WorkspaceID  string         `json:"workspace_id"`
	EventID      string         `json:"event_id"`
	EventPayload map[string]any `json:"event_payload"`
	Variables    map[string]any `json:"variables"`
	Results      []ActionResult `json:"results"`
	StartedAt    time.Time      `json:"started_at"`
}

// ActionResult holds the result of a single action execution
type ActionResult struct {
	ActionID string         `json:"action_id"`
	Type     ActionType     `json:"type"`
	Status   string         `json:"status"`
	Output   map[string]any `json:"output,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// Engine is the core blueprint execution engine
type Engine struct {
	actions *ActionRegistry
	storage *Storage
	logger  *zap.Logger
}

// NewEngine creates a new execution engine
func NewEngine(actions *ActionRegistry, storage *Storage, logger *zap.Logger) *Engine {
	return &Engine{
		actions: actions,
		storage: storage,
		logger:  logger,
	}
}

// ExecuteBlueprint runs a blueprint against an event
func (e *Engine) ExecuteBlueprint(ctx *fiber.Ctx, c context.Context, bp *FlowBlueprint, event *Event) error {
	e.logger.Info("executing blueprint",
		zap.String("blueprint_id", bp.ID),
		zap.String("blueprint_name", bp.Name),
		zap.String("event_id", event.ID),
	)

	// Create execution record
	exec := &BlueprintExecution{
		BlueprintID: bp.ID,
		WorkspaceID: bp.WorkspaceID,
		EventID:     event.ID,
		Status:      "running",
		StartedAt:   time.Now(),
	}

	if err := e.storage.CreateExecution(c, exec); err != nil {
		e.logger.Error("failed to create execution record", zap.Error(err))
		return fmt.Errorf("engine.ExecuteBlueprint: create execution: %w", err)
	}

	// Prepare execution context
	execCtx := &ExecutionContext{
		ExecutionID:  exec.ID,
		BlueprintID:  bp.ID,
		WorkspaceID:  bp.WorkspaceID,
		EventID:      event.ID,
		EventPayload: event.Payload,
		Variables:    make(map[string]any),
		Results:      make([]ActionResult, 0, len(bp.Actions)),
		StartedAt:    time.Now(),
	}

	// Seed variables from event payload
	for k, v := range event.Payload {
		execCtx.Variables[k] = v
	}

	// Execute each action in order
	actionsRun := 0
	actionsFailed := 0

	for i, action := range bp.Actions {
		e.logger.Info("executing action",
			zap.String("action_id", action.ID),
			zap.String("action_type", string(action.Type)),
			zap.Int("step", i+1),
		)

		result := e.executeAction(ctx, c, action, execCtx)
		execCtx.Results = append(execCtx.Results, result)

		if result.Status == "success" {
			actionsRun++
			// Make action output available to subsequent actions
			if result.Output != nil {
				for k, v := range result.Output {
					execCtx.Variables[action.ID+"_"+k] = v
				}
			}
		} else {
			actionsFailed++
			if action.Type == ActionWait {
				// Wait actions failing is ok, continue
				continue
			}
			// Stop on first non-recoverable error
			e.logger.Error("action failed, stopping execution",
				zap.String("action_id", action.ID),
				zap.String("error", result.Error),
			)
			break
		}
	}

	// Determine final status
	status := "completed"
	if actionsFailed > 0 {
		status = "partial"
	}

	// Update execution record
	var errMsg string
	if actionsFailed > 0 {
		// Build error message from last failed action
		for _, r := range execCtx.Results {
			if r.Status == "failed" {
				errMsg = fmt.Sprintf("action %s failed: %s", r.ActionID, r.Error)
				break
			}
		}
	}

	err := e.storage.CompleteExecution(c, exec.ID, status, actionsRun, actionsFailed, errMsg)
	if err != nil {
		e.logger.Error("failed to complete execution", zap.Error(err))
	}

	// Increment blueprint run count
	if err := e.storage.IncrementRunCount(c, bp.ID); err != nil {
		e.logger.Error("failed to increment run count", zap.Error(err))
	}

	e.logger.Info("blueprint execution finished",
		zap.String("blueprint_id", bp.ID),
		zap.String("execution_id", exec.ID),
		zap.String("status", status),
		zap.Int("actions_run", actionsRun),
		zap.Int("actions_failed", actionsFailed),
	)

	return nil
}

// executeAction runs a single action
func (e *Engine) executeAction(ctx *fiber.Ctx, c context.Context, action FlowAction, execCtx *ExecutionContext) ActionResult {
	result := ActionResult{
		ActionID: action.ID,
		Type:     action.Type,
		Status:   "running",
	}

	// Merge action config with execution variables (for templating)
	config := make(map[string]any)
	for k, v := range action.Config {
		config[k] = v
	}

	// Resolve variable references in config: ${varName} -> actual value
	for k, v := range config {
		if s, ok := v.(string); ok {
			resolved := e.resolveVariables(s, execCtx.Variables)
			config[k] = resolved
		}
	}

	// Get action function from registry
	actionFn, err := e.actions.Get(action.Type)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("action not registered: %s", action.Type)
		return result
	}

	// Call action function
	output, err := actionFn(ctx, c, execCtx.EventPayload, config)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result
	}

	result.Status = "success"
	result.Output = output
	return result
}

// resolveVariables replaces ${varName} placeholders with actual values
func (e *Engine) resolveVariables(input string, vars map[string]any) string {
	if len(input) < 3 {
		return input
	}

	result := input
	for varName, value := range vars {
		placeholder := fmt.Sprintf("${%s}", varName)
		if valStr, ok := value.(string); ok {
			result = ReplaceAll(result, placeholder, valStr)
		} else {
			result = ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
		}
	}
	return result
}

// ReplaceAll replaces all occurrences with basic string replacement (avoids strings import)
func ReplaceAll(s, old, new string) string {
	result := ""
	for {
		idx := -1
		for i := 0; i <= len(s)-len(old); i++ {
			if i+len(old) <= len(s) && s[i:i+len(old)] == old {
				idx = i
				break
			}
		}
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}
