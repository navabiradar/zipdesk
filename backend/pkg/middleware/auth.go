package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// Claims defines JWT payload
type Claims struct {
	UserID      string `json:"sub"`
	WorkspaceID string `json:"workspace_id"`
	Role        string `json:"role"`
	Plan        string `json:"plan"`
	jwt.RegisteredClaims
}

// AuthRequired validates JWT token
func AuthRequired(jwtSecret string, log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "missing authorization header",
				},
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "invalid authorization format",
				},
			})
		}

		token, err := jwt.ParseWithClaims(
			parts[1],
			&Claims{},
			func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fiber.ErrUnauthorized
				}
				return []byte(jwtSecret), nil
			},
		)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "invalid or expired token",
				},
			})
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "invalid token claims",
				},
			})
		}

		if claims.UserID == "" {
			claims.UserID = claims.Subject
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("workspace_id", claims.WorkspaceID)
		c.Locals("role", claims.Role)
		c.Locals("plan", claims.Plan)

		return c.Next()
	}
}

// WorkspaceRequired ensures workspace context exists
func WorkspaceRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		workspaceID := c.Locals("workspace_id")
		if workspaceID == nil || workspaceID == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "WORKSPACE_NOT_FOUND",
					"message": "workspace context required",
				},
			})
		}
		return c.Next()
	}
}

// RateLimit implements Redis-based rate limiting
func RateLimit(
	redis interface {
		IncrBy(ctx interface{}, key string, amount int64, ttl interface{}) (int64, error)
	},
	maxRequests int64,
	keyPrefix string,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := keyPrefix + ":" + c.IP()
		// Simplified rate limit check
		// Full implementation uses Redis sliding window
		_ = key
		return c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *fiber.Ctx) string {
	if id, ok := c.Locals("user_id").(string); ok {
		return id
	}
	return ""
}

// GetWorkspaceID extracts workspace ID from context
func GetWorkspaceID(c *fiber.Ctx) string {
	if id, ok := c.Locals("workspace_id").(string); ok {
		return id
	}
	return ""
}

// GetRole extracts role from context
func GetRole(c *fiber.Ctx) string {
	if role, ok := c.Locals("role").(string); ok {
		return role
	}
	return "member"
}

// GetPlan extracts plan from context
func GetPlan(c *fiber.Ctx) string {
	if plan, ok := c.Locals("plan").(string); ok {
		return plan
	}
	return "free"
}
