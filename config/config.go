package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Alpaca API Configuration
	AlpacaAPIKey    string
	AlpacaAPISecret string
	AlpacaBaseURL   string

	// Database Configuration
	DatabasePath string

	// Application Configuration
	Port        string
	Environment string
	LogLevel    string

	// Trading Configuration
	InitialBalance  float64
	MaxPositionSize float64
	RiskPercentage  float64
	TradingEnabled  bool

	// Performance Configuration
	RefreshInterval time.Duration
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load("config/.env"); err != nil {
		log.Printf("Warning: .env file not found, using environment variables: %v", err)
	}

	config := &Config{
		// Alpaca API defaults
		AlpacaAPIKey:    getEnv("ALPACA_API_KEY", ""),
		AlpacaAPISecret: getEnv("ALPACA_API_SECRET", ""),
		AlpacaBaseURL:   getEnv("ALPACA_BASE_URL", "https://paper-api.alpaca.markets"),

		// Database defaults
		DatabasePath: getEnv("DATABASE_PATH", "./data/trades.db"),

		// Application defaults
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Trading defaults
		InitialBalance:  getEnvFloat("INITIAL_BALANCE", 100000.0),
		MaxPositionSize: getEnvFloat("MAX_POSITION_SIZE", 10000.0),
		RiskPercentage:  getEnvFloat("RISK_PERCENTAGE", 0.02),
		TradingEnabled:  getEnvBool("TRADING_ENABLED", true),

		// Performance defaults
		RefreshInterval: getEnvDuration("REFRESH_INTERVAL", 5*time.Second),
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.AlpacaAPIKey == "" {
		return fmt.Errorf("ALPACA_API_KEY is required")
	}
	if c.AlpacaAPISecret == "" {
		return fmt.Errorf("ALPACA_API_SECRET is required")
	}
	if c.InitialBalance <= 0 {
		return fmt.Errorf("INITIAL_BALANCE must be positive")
	}
	if c.RiskPercentage <= 0 || c.RiskPercentage > 1 {
		return fmt.Errorf("RISK_PERCENTAGE must be between 0 and 1")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
