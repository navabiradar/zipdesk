package links

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

// Handler handles HTTP requests for links
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new links handler
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// RegisterRoutes registers all link routes
func (h *Handler) RegisterRoutes(
	app *fiber.App,
	v1 fiber.Router,
) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(jwtSecret, h.log)
	wsMiddleware := middleware.WorkspaceRequired()

	// Public redirect route (no auth, must be fast)
	app.Get("/s/:slug", h.Redirect)

	// Protected API routes
	links := v1.Group("/links",
		authMiddleware,
		wsMiddleware,
	)
	links.Post("/", h.CreateLink)
	links.Get("/", h.ListLinks)
	links.Get("/:id", h.GetLink)
	links.Put("/:id", h.UpdateLink)
	links.Delete("/:id", h.DeleteLink)
	links.Get("/:id/analytics", h.GetAnalytics)
}

// Redirect handles GET /s/:slug
// CRITICAL: Must be < 10ms
func (h *Handler) Redirect(c *fiber.Ctx) error {
	slug := c.Params("slug")

	req := ClickRequest{
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Referrer:  c.Get("Referer"),
	}

	destURL, err := h.svc.HandleRedirect(
		c.Context(), slug, req,
	)
	if err != nil {
		if linkErr, ok := err.(*LinkError); ok {
			switch linkErr.Code {
			case "NOT_FOUND":
				return c.Status(fiber.StatusNotFound).
					SendString("Link not found")
			case "EXPIRED":
				return c.Status(fiber.StatusGone).
					SendString("This link has expired")
			case "LINK_INACTIVE":
				return c.Status(fiber.StatusGone).
					SendString("This link is no longer active")
			case "CLICK_LIMIT_REACHED":
				return c.Status(fiber.StatusGone).
					SendString("This link has reached its limit")
			}
		}
		return c.Status(fiber.StatusInternalServerError).
			SendString("Redirect failed")
	}

	return c.Redirect(destURL, fiber.StatusMovedPermanently)
}

// CreateLink handles POST /api/v1/links
func (h *Handler) CreateLink(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	userID := middleware.GetUserID(c)

	var input CreateLinkInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "invalid request body",
			},
		})
	}

	link, err := h.svc.CreateLink(
		c.Context(), workspaceID, userID, input,
	)
	if err != nil {
		return handleLinksError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    link,
	})
}

// ListLinks handles GET /api/v1/links
func (h *Handler) ListLinks(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:     c.QueryInt("page", 1),
		PerPage:  c.QueryInt("per_page", 20),
		Search:   c.Query("search"),
		FolderID: c.Query("folder_id"),
		Tag:      c.Query("tag"),
		Sort:     c.Query("sort", "created_at"),
		Order:    c.Query("order", "desc"),
	}

	result, err := h.svc.ListLinks(
		c.Context(), workspaceID, params,
	)
	if err != nil {
		return handleLinksError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result.Items,
		"meta": fiber.Map{
			"total":    result.Total,
			"page":     result.Page,
			"per_page": result.PerPage,
		},
	})
}

// GetLink handles GET /api/v1/links/:id
func (h *Handler) GetLink(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	link, err := h.svc.GetByID(c.Context(), id, workspaceID)
	if err != nil {
		return handleLinksError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    link,
	})
}

// UpdateLink handles PUT /api/v1/links/:id
func (h *Handler) UpdateLink(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	var input UpdateLinkInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "invalid request body",
			},
		})
	}

	link, err := h.svc.UpdateLink(
		c.Context(), id, workspaceID, input,
	)
	if err != nil {
		return handleLinksError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    link,
	})
}

// DeleteLink handles DELETE /api/v1/links/:id
func (h *Handler) DeleteLink(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	if err := h.svc.DeleteLink(
		c.Context(), id, workspaceID,
	); err != nil {
		return handleLinksError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"message": "link deleted",
		},
	})
}

// GetAnalytics handles GET /api/v1/links/:id/analytics
func (h *Handler) GetAnalytics(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	params := AnalyticsParams{
		From: c.Query("from"),
		To:   c.Query("to"),
	}

	analytics, err := h.svc.GetAnalytics(
		c.Context(), id, workspaceID, params,
	)
	if err != nil {
		return handleLinksError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    analytics,
	})
}

// handleLinksError converts errors to HTTP responses
func handleLinksError(c *fiber.Ctx, err error) error {
	if linkErr, ok := err.(*LinkError); ok {
		status := fiber.StatusBadRequest
		switch linkErr.Code {
		case "NOT_FOUND":
			status = fiber.StatusNotFound
		case "EXPIRED", "LINK_INACTIVE",
			"CLICK_LIMIT_REACHED":
			status = fiber.StatusGone
		}
		return c.Status(status).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    linkErr.Code,
				"message": linkErr.Message,
			},
		})
	}

	if valErr, ok := err.(*ValidationError); ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": valErr.Message,
				"field":   valErr.Field,
			},
		})
	}

	zap.L().Error("links handler error", zap.Error(err))

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    "INTERNAL_ERROR",
			"message": "an unexpected error occurred",
		},
	})
}
