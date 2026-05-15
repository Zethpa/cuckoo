package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env            string
	HTTPAddr       string
	FrontendURL    string
	DatabaseDriver string
	DatabaseDSN    string
	JWTSecret      string
	TokenTTL       time.Duration
	AdminUsername  string
	AdminPassword  string
	AIServiceURL   string
}

func Load() Config {
	return Config{
		Env:            get("CUCKOO_ENV", "development"),
		HTTPAddr:       get("HTTP_ADDR", ":18081"),
		FrontendURL:    get("FRONTEND_URL", "http://localhost:15173"),
		DatabaseDriver: get("DB_DRIVER", "sqlite"),
		DatabaseDSN:    get("DB_DSN", "cuckoo.db"),
		JWTSecret:      get("JWT_SECRET", "dev-change-me"),
		TokenTTL:       time.Duration(getInt("JWT_TTL_HOURS", 72)) * time.Hour,
		AdminUsername:  get("CUCKOO_ADMIN_USERNAME", "admin"),
		AdminPassword:  get("CUCKOO_ADMIN_PASSWORD", "admin12345"),
		AIServiceURL:   get("AI_SERVICE_URL", "http://localhost:18787"),
	}
}

func get(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
