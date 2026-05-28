package flow

import (
	"bufio"
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

// Handler handles HTTP requests for flow
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new flow handler
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes registers all flow routes
func (h *Handler) RegisterRoutes(
	app *fiber.App,
	v1 fiber.Router,
) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(jwtSecret, h.logger)
	wsMiddleware := middleware.WorkspaceRequired()

	flow := v1.Group("/flow", authMiddleware, wsMiddleware)

	flow.Get("/health", h.GetHealth)
	flow.Get("/events", h.ListEvents)

	flow.Get("/blueprints", h.ListBlueprints)
	flow.Post("/blueprints", h.CreateBlueprint)
	flow.Delete("/blueprints/:id", h.DeleteBlueprint)

	flow.Post("/chat", h.Chat)
	flow.Get("/conversations/:id/messages", h.GetMessages)
}

// GetHealth handles GET /api/v1/flow/health
func (h *Handler) GetHealth(c *fiber.Ctx) error {
	report := h.svc.GetHealthReport(c.Context())
	return c.JSON(fiber.Map{"success": true, "data": report})
}

// ListEvents handles GET /api/v1/flow/events
func (h *Handler) ListEvents(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	limit := c.QueryInt("limit", 50)

	events, err := h.svc.ListEvents(c.Context(), workspaceID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": events})
}

// ListBlueprints handles GET /api/v1/flow/blueprints
func (h *Handler) ListBlueprints(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	blueprints, err := h.svc.ListBlueprints(c.Context(), workspaceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": blueprints})
}

// CreateBlueprint handles POST /api/v1/flow/blueprints
func (h *Handler) CreateBlueprint(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	var input CreateBlueprintInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "VALIDATION_ERROR", "message": "invalid request body"},
		})
	}

	bp, err := h.svc.CreateBlueprint(c.Context(), workspaceID, input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": bp})
}

// DeleteBlueprint handles DELETE /api/v1/flow/blueprints/:id
func (h *Handler) DeleteBlueprint(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	if err := h.svc.DeleteBlueprint(c.Context(), id, workspaceID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"message": "blueprint deleted"}})
}

// Chat handles POST /api/v1/flow/chat
func (h *Handler) Chat(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	userID := middleware.GetUserID(c)

	var input ChatInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "message is required",
			},
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	bw := bufio.NewWriter(c)
	h.svc.streamChat(c.Context(), bw, workspaceID, userID, input)
	return nil
}

// GetMessages handles GET /api/v1/flow/conversations/:id/messages
func (h *Handler) GetMessages(c *fiber.Ctx) error {
	id := c.Params("id")

	messages, err := h.svc.GetConversationMessages(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "NOT_FOUND", "message": "conversation not found"},
		})
	}

	return c.JSON(fiber.Map{"success": true, "data": messages})
}
