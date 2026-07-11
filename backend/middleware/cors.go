package middleware

import (
	"time"

	"kikundibora/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func SetupCORS() fiber.Handler {
	origins := config.AppConfig.CORSOrigins
	if origins == "" {
		origins = "http://localhost:3000,http://localhost:5173"
	}
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: true,
		MaxAge:           int(12 * time.Hour),
	})
}
