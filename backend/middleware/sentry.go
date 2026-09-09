package middleware

import (
	"time"

	"kikundibora/services"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
)

// Sentry attaches a per-request Sentry hub, recovers panics into Sentry +
// a 500 response, and reports handler errors / 5xx responses. No-op when
// Sentry is disabled (no SENTRY_DSN). Only the opaque user_id is tagged —
// never names, phones, or request bodies.
func Sentry() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !services.Enabled {
			return c.Next()
		}

		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetTag("method", c.Method())
		hub.Scope().SetTag("route", c.Route().Path)

		defer func() {
			if r := recover(); r != nil {
				hub.Recover(r)
				// Deliver before the response goes out — a panic may be
				// followed by shutdown.
				sentry.Flush(2 * time.Second)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"message": "Hitilafu ya mfumo",
				})
			}
		}()

		err := c.Next()

		// Tag the authenticated user (set by AuthRequired for protected routes).
		if userID := GetUserID(c); userID != "" {
			hub.Scope().SetUser(sentry.User{ID: userID})
		}

		status := c.Response().StatusCode()
		if err != nil {
			hub.CaptureException(err)
		} else if status >= 500 {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetContext("response", sentry.Context{
					"status": status,
					"path":   c.Path(),
				})
				hub.CaptureMessage("5xx response")
			})
		}

		return err
	}
}
