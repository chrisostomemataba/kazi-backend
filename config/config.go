package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL          string
	JWTSecret            string
	SMSAPIToken          string
	SMSSenderID          string
	SMSBaseURL           string
	MinIOEndpoint        string
	MinIOAccessKey       string
	MinIOSecretKey       string
	MinIOBucket          string
	MinIOUseSSL          bool
	Port                 string
	Environment          string
	PaymentServiceURL    string
	PaymentWebhookSecret string
	CasdoorEndpoint      string
	CasdoorClientID      string
	CasdoorClientSecret  string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		SMSAPIToken:          getEnv("SMS_API_TOKEN", ""),
		SMSSenderID:          getEnv("SMS_SENDER_ID", "32"),
		SMSBaseURL:           getEnv("SMS_BASE_URL", "https://api.notify.africa/api/v1/api/messages/send"),
		MinIOEndpoint:        getEnv("MINIO_INTERNAL_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:       getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey:       getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:          getEnv("MINIO_BUCKET", "taskmaid-tz"),
		MinIOUseSSL:          getEnv("MINIO_SECURE", "false") == "true",
		Port:                 getEnv("PORT", "8080"),
		Environment:          getEnv("ENV", "development"),
		PaymentServiceURL:    getEnv("PAYMENT_SERVICE_URL", ""),
		PaymentWebhookSecret: getEnv("PAYMENT_WEBHOOK_SECRET", ""),
		CasdoorEndpoint:      getEnv("CASDOOR_ENDPOINT", ""),
		CasdoorClientID:      getEnv("CASDOOR_CLIENT_ID", ""),
		CasdoorClientSecret:  getEnv("CASDOOR_CLIENT_SECRET", ""),
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
	if c.MinIOAccessKey == "" {
		return fmt.Errorf("MINIO_ACCESS_KEY is required")
	}
	if c.MinIOSecretKey == "" {
		return fmt.Errorf("MINIO_SECRET_KEY is required")
	}
	if c.CasdoorEndpoint == "" {
		return fmt.Errorf("CASDOOR_ENDPOINT is required")
	}
	if c.CasdoorClientID == "" {
		return fmt.Errorf("CASDOOR_CLIENT_ID is required")
	}
	if c.CasdoorClientSecret == "" {
		return fmt.Errorf("CASDOOR_CLIENT_SECRET is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
