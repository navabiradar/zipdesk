package auth

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/zipdesk/backend/pkg/middleware"
)

type Handler struct {
	svc    *Service
	logger *zap.Logger
}

func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

func (h *Handler) RegisterRoutes(app *fiber.App, v1 fiber.Router) {
	jwtSecret := os.Getenv("JWT_SECRET")
	authMiddleware := middleware.AuthRequired(jwtSecret, h.logger)

	authGroup := v1.Group("/auth")
	authGroup.Get("/health", h.health)
	authGroup.Post("/register", h.register)
	authGroup.Post("/login", h.login)
	authGroup.Get("/me", authMiddleware, h.getMe)
}

func (h *Handler) getMe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	workspaceID := c.Locals("workspace_id").(string)

	resp, err := h.svc.GetMe(c.Context(), userID, workspaceID)
	if err != nil {
		h.logger.Error("failed to get user", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "failed to get user info",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) register(c *fiber.Ctx) error {
	input := &RegisterInput{}
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "BAD_REQUEST",
				"message": "invalid request body",
			},
		})
	}

	resp, err := h.svc.Register(c.Context(), input)
	if err != nil {
		h.logger.Error("registration failed", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "REGISTER_FAILED",
				"message": err.Error(),
			},
		})
	}

	h.setAuthCookie(c, resp.AccessToken)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) login(c *fiber.Ctx) error {
	input := &LoginInput{}
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "BAD_REQUEST",
				"message": "invalid request body",
			},
		})
	}

	resp, err := h.svc.Login(c.Context(), input)
	if err != nil {
		h.logger.Warn("login failed", zap.Error(err))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "invalid credentials",
			},
		})
	}

	h.setAuthCookie(c, resp.AccessToken)
	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

func (h *Handler) setAuthCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: "Lax",
		MaxAge:   int(AccessTokenTTL.Seconds()),
	})
}
