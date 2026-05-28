package crm

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

// Handler handles HTTP requests for CRM
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new CRM handler
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// RegisterRoutes registers all CRM routes
func (h *Handler) RegisterRoutes(
	app *fiber.App,
	v1 fiber.Router,
) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(jwtSecret, h.log)
	wsMiddleware := middleware.WorkspaceRequired()

	crm := v1.Group("/crm", authMiddleware, wsMiddleware)

	crm.Post("/contacts", h.CreateContact)
	crm.Get("/contacts", h.ListContacts)
	crm.Get("/deals", h.ListDeals)
}

// CreateContact handles POST /api/v1/crm/contacts
func (h *Handler) CreateContact(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	var input CreateContactInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "VALIDATION_ERROR", "message": "invalid request body"},
		})
	}

	contact, err := h.svc.CreateContact(c.Context(), workspaceID, input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    contact,
	})
}

// ListContacts handles GET /api/v1/crm/contacts
func (h *Handler) ListContacts(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
		Search:  c.Query("search"),
	}

	contacts, total, err := h.svc.ListContacts(c.Context(), workspaceID, params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    contacts,
		"meta": fiber.Map{
			"total":    total,
			"page":     params.Page,
			"per_page": params.PerPage,
		},
	})
}

// ListDeals handles GET /api/v1/crm/deals
func (h *Handler) ListDeals(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
	}

	deals, total, err := h.svc.ListDeals(c.Context(), workspaceID, params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    deals,
		"meta": fiber.Map{
			"total":    total,
			"page":     params.Page,
			"per_page": params.PerPage,
		},
	})
}
