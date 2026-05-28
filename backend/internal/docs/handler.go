package docs

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

// Handler handles HTTP requests for docs
type Handler struct {
	svc *Service
	log *zap.Logger
}

// NewHandler creates a new docs handler
func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// RegisterRoutes registers all doc routes
func (h *Handler) RegisterRoutes(
	app *fiber.App,
	v1 fiber.Router,
) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(
		jwtSecret, h.log,
	)
	wsMiddleware := middleware.WorkspaceRequired()

	// Public routes
	app.Get("/d/:slug", h.GetPublicDocument)

	// Protected routes
	docs := v1.Group("/docs",
		authMiddleware,
		wsMiddleware,
	)
	docs.Post("/", h.CreateDocument)
	docs.Get("/", h.ListDocuments)
	docs.Get("/:id", h.GetDocument)
	docs.Put("/:id", h.UpdateDocument)
	docs.Delete("/:id", h.DeleteDocument)
	docs.Post("/:id/publish", h.PublishDocument)
}

// CreateDocument handles POST /api/v1/docs
func (h *Handler) CreateDocument(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	userID := middleware.GetUserID(c)

	var input CreateDocInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "VALIDATION_ERROR",
					"message": "invalid request body",
				},
			},
		)
	}

	doc, err := h.svc.CreateDocument(
		c.Context(), workspaceID, userID, input,
	)
	if err != nil {
		return handleDocsError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    doc,
	})
}

// ListDocuments handles GET /api/v1/docs
func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
		Search:  c.Query("search"),
	}

	docs, total, err := h.svc.ListDocuments(
		c.Context(), workspaceID, params,
	)
	if err != nil {
		return handleDocsError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    docs,
		"meta": fiber.Map{
			"total":    total,
			"page":     params.Page,
			"per_page": params.PerPage,
		},
	})
}

// GetDocument handles GET /api/v1/docs/:id
func (h *Handler) GetDocument(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	doc, err := h.svc.GetDocument(
		c.Context(), id, workspaceID,
	)
	if err != nil {
		return handleDocsError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    doc,
	})
}

// UpdateDocument handles PUT /api/v1/docs/:id
func (h *Handler) UpdateDocument(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	var input UpdateDocInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "VALIDATION_ERROR",
					"message": "invalid request body",
				},
			},
		)
	}

	doc, err := h.svc.UpdateDocument(
		c.Context(), id, workspaceID, input,
	)
	if err != nil {
		return handleDocsError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    doc,
	})
}

// DeleteDocument handles DELETE /api/v1/docs/:id
func (h *Handler) DeleteDocument(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	if err := h.svc.DeleteDocument(
		c.Context(), id, workspaceID,
	); err != nil {
		return handleDocsError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"message": "document deleted",
		},
	})
}

// PublishDocument handles POST /docs/:id/publish
func (h *Handler) PublishDocument(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	doc, err := h.svc.PublishDocument(
		c.Context(), id, workspaceID,
	)
	if err != nil {
		return handleDocsError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    doc,
	})
}

// GetPublicDocument handles GET /d/:slug
func (h *Handler) GetPublicDocument(c *fiber.Ctx) error {
	slug := c.Params("slug")

	doc, err := h.svc.GetPublicDocument(
		c.Context(), slug,
	)
	if err != nil {
		return handleDocsError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    doc,
	})
}

// handleDocsError converts errors to responses
func handleDocsError(c *fiber.Ctx, err error) error {
	if docsErr, ok := err.(*DocsError); ok {
		status := fiber.StatusBadRequest
		if docsErr.Code == "NOT_FOUND" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    docsErr.Code,
				"message": docsErr.Message,
			},
		})
	}

	return c.Status(fiber.StatusInternalServerError).
		JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "an unexpected error occurred",
			},
		})
}
