package mail

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

// Handler handles HTTP requests for mail
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new mail handler
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// RegisterRoutes registers all mail routes
func (h *Handler) RegisterRoutes(
	app *fiber.App,
	v1 fiber.Router,
) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(jwtSecret, h.log)
	wsMiddleware := middleware.WorkspaceRequired()

	// Public tracking routes
	app.Get("/track/open/:token", h.TrackOpen)
	app.Get("/track/click/:token", h.TrackClick)
	app.Get("/unsubscribe/:token", h.Unsubscribe)

	// Protected API routes
	mail := v1.Group("/mail", authMiddleware, wsMiddleware)

	mail.Post("/contacts", h.CreateContact)
	mail.Get("/contacts", h.ListContacts)
	mail.Delete("/contacts/:id", h.DeleteContact)

	mail.Post("/campaigns", h.CreateCampaign)
	mail.Get("/campaigns", h.ListCampaigns)
	mail.Get("/campaigns/:id/stats", h.GetCampaignStats)
}

// CreateContact handles POST /api/v1/mail/contacts
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
		return handleMailError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    contact,
	})
}

// ListContacts handles GET /api/v1/mail/contacts
func (h *Handler) ListContacts(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
		Search:  c.Query("search"),
	}

	contacts, total, err := h.svc.ListContacts(c.Context(), workspaceID, params)
	if err != nil {
		return handleMailError(c, err)
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

// DeleteContact handles DELETE /mail/contacts/:id
func (h *Handler) DeleteContact(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	if err := h.svc.DeleteContact(c.Context(), id, workspaceID); err != nil {
		return handleMailError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"message": "contact deleted"},
	})
}

// CreateCampaign handles POST /mail/campaigns
func (h *Handler) CreateCampaign(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	var input CreateCampaignInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   fiber.Map{"code": "VALIDATION_ERROR", "message": "invalid request body"},
		})
	}

	campaign, err := h.svc.CreateCampaign(c.Context(), workspaceID, input)
	if err != nil {
		return handleMailError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    campaign,
	})
}

// ListCampaigns handles GET /mail/campaigns
func (h *Handler) ListCampaigns(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
	}

	campaigns, total, err := h.svc.ListCampaigns(c.Context(), workspaceID, params)
	if err != nil {
		return handleMailError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    campaigns,
		"meta": fiber.Map{
			"total":    total,
			"page":     params.Page,
			"per_page": params.PerPage,
		},
	})
}

// GetCampaignStats handles GET /campaigns/:id/stats
func (h *Handler) GetCampaignStats(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	stats, err := h.svc.GetCampaignStats(c.Context(), id, workspaceID)
	if err != nil {
		return handleMailError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// TrackOpen handles GET /track/open/:token
func (h *Handler) TrackOpen(c *fiber.Ctx) error {
	pixel := []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61,
		0x01, 0x00, 0x01, 0x00, 0x00, 0xff,
		0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x00, 0x02,
		0x00, 0x3b,
	}
	c.Set("Content-Type", "image/gif")
	c.Set("Cache-Control", "no-cache")
	return c.Send(pixel)
}

// TrackClick handles GET /track/click/:token
func (h *Handler) TrackClick(c *fiber.Ctx) error {
	return c.Redirect("https://zipdesk.io")
}

// Unsubscribe handles GET /unsubscribe/:token
func (h *Handler) Unsubscribe(c *fiber.Ctx) error {
	return c.SendString("You have been unsubscribed successfully.")
}

// handleMailError converts errors to responses
func handleMailError(c *fiber.Ctx, err error) error {
	if mailErr, ok := err.(*MailError); ok {
		status := fiber.StatusBadRequest
		switch mailErr.Code {
		case "NOT_FOUND":
			status = fiber.StatusNotFound
		case "DUPLICATE_CONTACT":
			status = fiber.StatusConflict
		}
		return c.Status(status).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    mailErr.Code,
				"message": mailErr.Message,
			},
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    "INTERNAL_ERROR",
			"message": "an unexpected error occurred",
		},
	})
}
