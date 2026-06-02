package handlers

import "github.com/gofiber/fiber/v2"

// errUnauthorized writes the canonical 401 response for an authenticated
// endpoint reached without the character identity that NewAuthMiddleware
// normally supplies. It centralizes what used to be a dozen divergent inline
// responses ("Authentication required" / "authentication required" /
// "Invalid authentication context") behind a single, uniform body.
func errUnauthorized(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
}
