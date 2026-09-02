package middleware

import (
	"kikundibora/config"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders adds security-related HTTP headers to responses.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prevent MIME type sniffing
		c.Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")

		// XSS protection (legacy browsers)
		c.Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict browser features
		c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// HSTS in production only (must not be set in development)
		if config.AppConfig != nil && config.AppConfig.Environment == "production" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Content Security Policy - restrict resource loading
		csp := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'"
		if config.AppConfig != nil && config.AppConfig.Environment == "development" {
			csp = "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'"
		}
		c.Set("Content-Security-Policy", csp)

		return c.Next()
	}
}
