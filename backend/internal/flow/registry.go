package flow

import (
	"context"
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// ActionFn is a function that executes an action
// ctx: fiber context for HTTP-related operations
// c: application context for background operations
// payload: event payload data
// config: action-specific configuration
// Returns: partial result data and error

type ActionFn func(ctx *fiber.Ctx, c context.Context, payload map[string]any, config map[string]any) (map[string]any, error)

// ActionRegistry maps action types to implementation functions
type ActionRegistry struct {
	mu      sync.RWMutex
	actions map[ActionType]ActionFn
}

// NewActionRegistry creates a new, empty action registry
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		actions: make(map[ActionType]ActionFn),
	}
}

// Register adds an action to the registry (thread-safe)
func (r *ActionRegistry) Register(actionType ActionType, fn ActionFn) {
	r.mu.Lock()
	r.actions[actionType] = fn
	r.mu.Unlock()
}

// Get retrieves an action from the registry (thread-safe)
func (r *ActionRegistry) Get(actionType ActionType) (ActionFn, error) {
	r.mu.RLock()
	fn, ok := r.actions[actionType]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("action not found: %s", actionType)
	}
	return fn, nil
}

// Has checks if an action type is registered (thread-safe)
func (r *ActionRegistry) Has(actionType ActionType) bool {
	r.mu.RLock()
	_, ok := r.actions[actionType]
	r.mu.RUnlock()
	return ok
}

// Remove unregisters an action (thread-safe)
func (r *ActionRegistry) Remove(actionType ActionType) {
	r.mu.Lock()
	delete(r.actions, actionType)
	r.mu.Unlock()
}

// List returns all registered action types (thread-safe)
func (r *ActionRegistry) List() []ActionType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ActionType, 0, len(r.actions))
	for at := range r.actions {
		result = append(result, at)
	}
	return result
}
