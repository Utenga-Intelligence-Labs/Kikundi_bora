package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	JWTSecret     string
	Port          string
	CORSOrigins   string
	PublicBaseURL string
	Environment   string
}

var AppConfig *Config

func Load() {
	_ = godotenv.Load()

	secret := getEnv("JWT_SECRET", "")
	if secret == "" {
		log.Fatal("FATAL: JWT_SECRET is required. Generate with: openssl rand -hex 32")
	}
	if len(secret) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters. Generate with: openssl rand -hex 32")
	}

	publicBase := getEnv("PUBLIC_BASE_URL", "")
	if publicBase == "" {
		// Default API public URL for local development
		publicBase = "http://localhost:" + getEnv("PORT", "8080")
	}

	AppConfig = &Config{
		DBHost:        getEnv("DB_HOST", "127.0.0.1"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "kikundi_db"),
		DBSSLMode:     getEnv("DB_SSLMODE", "require"),
		JWTSecret:     secret,
		Port:          getEnv("PORT", "8080"),
		CORSOrigins:   getEnv("CORS_ORIGINS", ""),
		PublicBaseURL: publicBase,
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
