package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// OpenAI
	OpenAIAPIKey string
	ModelName    string

	// Google Cloud
	GoogleCloudProject string
	SubscriptionID     string
	TopicName          string

	// Database
	DatabasePath string

	// App settings
	NumWorkers           int
	InitialEmailsToFetch int64
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		OpenAIAPIKey:         getEnv("OPENAI_API_KEY", ""),
		ModelName:            getEnv("MODEL_NAME", "gpt-4o-mini"),
		GoogleCloudProject:   getEnv("GOOGLE_CLOUD_PROJECT", ""),
		SubscriptionID:       getEnv("SUBSCRIPTION_ID", ""),
		DatabasePath:         getEnv("DATABASE_PATH", "mailai.db"),
		NumWorkers:           5,
		InitialEmailsToFetch: 20,
	}

	// Validate required fields
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	if cfg.GoogleCloudProject == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT is required")
	}
	if cfg.SubscriptionID == "" {
		return nil, fmt.Errorf("SUBSCRIPTION_ID is required")
	}

	cfg.TopicName = fmt.Sprintf("projects/%s/topics/gmail-topic", cfg.GoogleCloudProject)

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
