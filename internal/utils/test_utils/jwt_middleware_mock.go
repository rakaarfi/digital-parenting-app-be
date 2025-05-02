// internal/api/v1/handlers/test_utils/jwt_middleware_mock.go
package test_utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rakaarfi/digital-parenting-app-be/internal/utils"
)

// MockJWTMiddleware creates a middleware that simulates the JWT authentication middleware
// by setting the appropriate values in the Fiber context.
func MockJWTMiddleware(userID int, username string, role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create a mock JwtClaims object
		claims := &utils.JwtClaims{
			UserID:   userID,
			Username: username,
			Role:     role,
		}

		// Set the claims in the context as the JWT middleware would
		c.Locals("user", claims)
		
		return c.Next()
	}
}
