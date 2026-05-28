package forms

import (
	"errors"
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

// Handler handles HTTP requests for forms
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler creates a new forms handler
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes registers all form routes
func (h *Handler) RegisterRoutes(app *fiber.App, v1 fiber.Router) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(jwtSecret, h.logger)
	wsMiddleware := middleware.WorkspaceRequired()

	// Public routes (no auth)
	app.Get("/f/:slug", h.publicView)
	app.Post("/f/:slug/submit", h.publicSubmit)
	app.Post("/f/:slug/view", h.recordView)

	// Protected API routes
	forms := v1.Group("/forms", authMiddleware, wsMiddleware)
	forms.Post("/", h.create)
	forms.Get("/", h.list)
	forms.Get("/:id", h.get)
	forms.Put("/:id", h.update)
	forms.Delete("/:id", h.delete)
	forms.Post("/:id/publish", h.publish)
	forms.Post("/:id/submit", h.submit)
	forms.Get("/:id/responses", h.responses)
	forms.Get("/:id/export", h.exportCSV)
	forms.Get("/:id/analytics", h.analytics)
}

// publicView handles GET /f/:slug
func (h *Handler) publicView(c *fiber.Ctx) error {
	slug := c.Params("slug")

	form, err := h.svc.GetPublicForm(c.Context(), slug)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "data": form})
}

// publicSubmit handles POST /f/:slug/submit
func (h *Handler) publicSubmit(c *fiber.Ctx) error {
	slug := c.Params("slug")

	var input SubmitInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "invalid request body"))
	}
	if input.Data == nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "data is required"))
	}

	meta := ResponseMeta{
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Referrer:  c.Get("Referer"),
	}

	response, err := h.svc.SubmitPublicResponse(c.Context(), slug, input, meta)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":      response.ID,
			"message": "response submitted successfully",
		},
	})
}

// recordView handles POST /f/:slug/view
func (h *Handler) recordView(c *fiber.Ctx) error {
	slug := c.Params("slug")

	meta := ResponseMeta{
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Referrer:  c.Get("Referer"),
	}

	if err := h.svc.RecordView(c.Context(), slug, meta); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true})
}

// create handles POST /api/v1/forms
func (h *Handler) create(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	userID := middleware.GetUserID(c)

	input := new(CreateFormInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "invalid request body"))
	}
	if input.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "title is required"))
	}

	form, err := h.svc.CreateForm(c.Context(), workspaceID, userID, *input)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": form})
}

// list handles GET /api/v1/forms
func (h *Handler) list(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
		Search:  c.Query("search"),
	}

	result, err := h.svc.ListForms(c.Context(), workspaceID, params)
	if err != nil {
		return h.handleError(c, err)
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

// get handles GET /api/v1/forms/:id
func (h *Handler) get(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	form, err := h.svc.GetForm(c.Context(), id, workspaceID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "data": form})
}

// update handles PUT /api/v1/forms/:id
func (h *Handler) update(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	input := new(UpdateFormInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "invalid request body"))
	}

	form, err := h.svc.UpdateForm(c.Context(), id, workspaceID, *input)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "data": form})
}

// delete handles DELETE /api/v1/forms/:id
func (h *Handler) delete(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	if err := h.svc.DeleteForm(c.Context(), id, workspaceID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"message": "form deleted"}})
}

// publish handles POST /api/v1/forms/:id/publish
func (h *Handler) publish(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	form, err := h.svc.PublishForm(c.Context(), id, workspaceID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "data": form})
}

// submit handles POST /api/v1/forms/:id/submit (authenticated)
func (h *Handler) submit(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	input := new(SubmitInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "invalid request body"))
	}
	if input.Data == nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody("VALIDATION_ERROR", "data is required"))
	}

	meta := ResponseMeta{
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Referrer:  c.Get("Referer"),
	}

	response, err := h.svc.SubmitResponse(c.Context(), id, workspaceID, *input, meta)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": response})
}

// responses handles GET /api/v1/forms/:id/responses
func (h *Handler) responses(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	params := ListParams{
		Page:    c.QueryInt("page", 1),
		PerPage: c.QueryInt("per_page", 20),
	}

	result, err := h.svc.GetResponses(c.Context(), id, workspaceID, params)
	if err != nil {
		return h.handleError(c, err)
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

// exportCSV handles GET /api/v1/forms/:id/export
func (h *Handler) exportCSV(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	csvBytes, err := h.svc.ExportCSV(c.Context(), id, workspaceID)
	if err != nil {
		return h.handleError(c, err)
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=responses.csv")
	return c.Send(csvBytes)
}

// analytics handles GET /api/v1/forms/:id/analytics
func (h *Handler) analytics(c *fiber.Ctx) error {
	workspaceID := middleware.GetWorkspaceID(c)
	id := c.Params("id")

	analytics, err := h.svc.GetAnalytics(c.Context(), id, workspaceID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "data": analytics})
}

// handleError converts domain errors to consistent HTTP responses
func (h *Handler) handleError(c *fiber.Ctx, err error) error {
	var formErr *FormError
	if errors.As(err, &formErr) {
		status := fiber.StatusBadRequest
		switch formErr.Code {
		case "NOT_FOUND":
			status = fiber.StatusNotFound
		case "FORM_NOT_PUBLISHED", "FORM_CLOSED", "RESPONSE_LIMIT_REACHED":
			status = fiber.StatusForbidden
		case "VALIDATION_ERROR":
			status = fiber.StatusBadRequest
		case "SLUG_TAKEN":
			status = fiber.StatusConflict
		}
		body := fiber.Map{"success": false, "error": fiber.Map{"code": formErr.Code, "message": formErr.Message}}
		if formErr.Field != "" {
			body["error"].(fiber.Map)["field"] = formErr.Field
		}
		return c.Status(status).JSON(body)
	}

	h.logger.Error("unhandled handler error", zap.Error(err))
	return c.Status(fiber.StatusInternalServerError).JSON(errorBody("INTERNAL_ERROR", "an unexpected error occurred"))
}

// errorBody creates a consistent error response body
func errorBody(code, message string) fiber.Map {
	return fiber.Map{"success": false, "error": fiber.Map{"code": code, "message": message}}
}
