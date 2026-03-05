package middleware

import (
	"kazi-backend/internal/common/util"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Authorization header is required",
			})
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid authorization header format",
			})
		}

		token := tokenParts[1]
		claims, err := util.ValidateJWT(token, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid or expired token",
			})
		}

		c.Locals("userID", claims.UserID)
		c.Locals("roles", claims.Roles)
		return c.Next()
	}
}

func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Access denied",
			})
		}

		for _, userRole := range roles {
			for _, allowedRole := range allowedRoles {
				if userRole == allowedRole {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Insufficient permissions",
		})
	}
}

func RequireAdmin(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Authorization header is required",
			})
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid authorization header format",
			})
		}

		token := tokenParts[1]
		claims, err := util.ValidateJWT(token, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid or expired token",
			})
		}

		// Check if user has admin role
		isAdmin := false
		for _, role := range claims.Roles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error":   "Admin access required",
			})
		}

		c.Locals("userID", claims.UserID)
		c.Locals("roles", claims.Roles)
		return c.Next()
	}
}