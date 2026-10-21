package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration options for the godp server.
type Config struct {
	Host             string
	Port             int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	MaxRequestSizeMB int64
	APIKey           string
	CORSAllowOrigin  string
}

// Load loads configuration from environment variables with production-ready defaults.
func Load() *Config {
	return &Config{
		Host:             getEnv("HOST", "0.0.0.0"),
		Port:             getEnvAsInt("PORT", 5050),
		ReadTimeout:      getEnvAsDuration("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:     getEnvAsDuration("WRITE_TIMEOUT", 30*time.Second),
		MaxRequestSizeMB: int64(getEnvAsInt("MAX_REQUEST_SIZE_MB", 15)), // 15MB default for bulk images
		APIKey:           getEnv("API_KEY", ""),                         // Empty means disabled
		CORSAllowOrigin:  getEnv("CORS_ALLOW_ORIGIN", "*"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if valStr, ok := os.LookupEnv(key); ok {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr, ok := os.LookupEnv(key); ok {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
