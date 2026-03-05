package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	JWTSecret    string
	SMSAPIToken  string
	SMSSenderID  string
	SMSBaseURL   string
	Port         string
	Environment  string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		SMSAPIToken:  getEnv("SMS_API_TOKEN", ""),
		SMSSenderID:  getEnv("SMS_SENDER_ID", "32"),
		SMSBaseURL:   getEnv("SMS_BASE_URL", "https://api.notify.africa/api/v1/api/messages/send"),
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("ENV", "development"),
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
	if c.SMSAPIToken == "" {
		return fmt.Errorf("SMS_API_TOKEN is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}