package middleware

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequestLogger() fiber.Handler {
	isDev := os.Getenv("ENVIRONMENT") != "production"

	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		status := c.Response().StatusCode()

		if isDev {
			// Detailed logging like Python/Flask
			log.Printf("──────────────────────────────────────────────────")
			log.Printf("  %s %s", c.Method(), c.OriginalURL())
			log.Printf("  Status: %d | Latency: %s | IP: %s", status, latency.Round(time.Microsecond), c.IP())
			log.Printf("  User-Agent: %s", c.Get("User-Agent"))
			if c.Get("Authorization") != "" {
				auth := c.Get("Authorization")
				if len(auth) > 20 {
					auth = auth[:20] + "..."
				}
				log.Printf("  Auth: %s", auth)
			}
			if len(c.Body()) > 0 && len(c.Body()) < 2000 {
				log.Printf("  Body: %s", string(c.Body()))
			}
			if err != nil {
				log.Printf("  ERROR: %v", err)
			}
			if status >= 400 {
				log.Printf("  Response: %s", string(c.Response().Body()))
			}
			log.Printf("──────────────────────────────────────────────────")
		} else {
			// Production: one-line summary
			log.Printf("%-7s %-30s %d %s",
				c.Method(),
				c.Path(),
				status,
				latency.Round(time.Microsecond),
			)
		}

		return err
	}
}
