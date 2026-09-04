package middleware

import (
	"log"
	"strings"
	"time"

	"kikundibora/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func SetupCORS() fiber.Handler {
	origins := config.AppConfig.CORSOrigins
	if origins == "" {
		origins = "http://localhost:3000,http://localhost:5173,http://localhost:8080,http://localhost:8081,http://127.0.0.1:3000,http://127.0.0.1:5173,http://127.0.0.1:8080,http://127.0.0.1:8081"
	}
	// CORS-M01: wildcard origins are incompatible with AllowCredentials —
	// refuse to boot with an insecure configuration rather than silently
	// widening the policy.
	for _, o := range strings.Split(origins, ",") {
		if strings.TrimSpace(o) == "*" {
			log.Fatal("FATAL: CORS_ORIGINS='*' is not allowed while AllowCredentials is enabled. List explicit origins.")
		}
	}
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		ExposeHeaders:    "Content-Length",
		AllowCredentials: true,
		MaxAge:           int((12 * time.Hour).Seconds()),
	})
}
