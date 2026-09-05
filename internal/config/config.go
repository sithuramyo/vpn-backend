// Package config loads application configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv  string
	AppPort string

	DatabaseURL string

	JWTSecret     string
	JWTExpiry     time.Duration
	SessionCookie string

	SecretEncryptionKey string

	VPNDomain string
	APIDomain string

	GoogleClientID string

	CORSAllowedOrigins []string

	RateLimitRPS   float64
	RateLimitBurst int

	OutlineAPIURL       string
	OutlineAPICertSHA256 string
}

func Load() *Config {
	cfg := &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("APP_PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		JWTExpiry:            getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		SessionCookie:        getEnv("SESSION_COOKIE_NAME", "vpn_admin_session"),
		SecretEncryptionKey:  getEnv("SECRET_ENCRYPTION_KEY", ""),
		VPNDomain:            getEnv("VPN_DOMAIN", "vpn.thestrm.space"),
		APIDomain:            getEnv("API_DOMAIN", "api.vpn.thestrm.space"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		CORSAllowedOrigins:   splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		RateLimitRPS:         getFloatEnv("RATE_LIMIT_RPS", 10),
		RateLimitBurst:       getIntEnv("RATE_LIMIT_BURST", 20),
		OutlineAPIURL:        getEnv("OUTLINE_API_URL", ""),
		OutlineAPICertSHA256: getEnv("OUTLINE_API_CERT_SHA256", ""),
	}
	return cfg
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
