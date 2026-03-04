package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	SMSAPIKey   string
	SMSSenderID string
	Port        string
	Environment string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		SMSAPIKey:   getEnv("SMS_API_KEY", ""),
		SMSSenderID: getEnv("SMS_SENDER_ID", "KAZI"),
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENV", "development"),
	}

	if err := cfg.Validate(); err != nil {
		log.Fatal("Configuration error:", err)
	}

	return cfg
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}