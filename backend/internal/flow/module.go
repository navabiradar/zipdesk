package flow

import (
	"github.com/gofiber/fiber/v2"
)

// Module wires together all flow components
type Module struct {
	Handler  *Handler
	EventBus *EventBus
	Service  *Service
	Health   *HealthMonitor
	Engine   *Engine
	Registry *ActionRegistry
	Actions  *Actions
	Storage  *Storage
	Repo     *Repository
}

// ProvideModule creates and wires the complete flow module
// This is used in main.go to wire everything together
func ProvideModule(
	repo *Repository,
	storage *Storage,
	eventBus *EventBus,
	engine *Engine,
	registry *ActionRegistry,
	actions *Actions,
	mailSvc interface{},
	svc *Service,
	handler *Handler,
	health *HealthMonitor,
) *Module {
	return &Module{
		Handler:  handler,
		EventBus: eventBus,
		Service:  svc,
		Health:   health,
		Engine:   engine,
		Registry: registry,
		Actions:  actions,
		Storage:  storage,
		Repo:     repo,
	}
}

// RegisterRoutes exposes the module's routes on Fiber
func (m *Module) RegisterRoutes(app *fiber.App, v1 fiber.Router) {
	m.Handler.RegisterRoutes(app, v1)
}
