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
	// SMS channel (Part 1): provider selection + credentials come from env
	// only, never the database. Group on/off + per-type prefs live in the DB.
	SMSProvider string
	SMSAPIKey   string
	SMSSenderID string
	SMSBaseURL  string
	// OTP verification (Part 2): off by default. When false the auth flow
	// behaves exactly as before; the OTP model/endpoints stay dormant.
	OTPVerificationEnabled bool
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
		SMSProvider:   getEnv("SMS_PROVIDER", "noop"),
		SMSAPIKey:     getEnv("SMS_API_KEY", ""),
		SMSSenderID:   getEnv("SMS_SENDER_ID", ""),
		SMSBaseURL:    getEnv("SMS_BASE_URL", ""),
		OTPVerificationEnabled: getEnvBool("OTP_VERIFICATION_ENABLED", false),
	}
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
