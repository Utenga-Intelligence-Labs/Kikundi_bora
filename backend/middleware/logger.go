package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		log.Printf("%-7s %-30s %d %s",
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			latency.Round(time.Microsecond),
		)

		return err
	}
}
