package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	Port        string
}

// Load function reads environment variables and validates them.
func Load() (*Config, error) {
	// Database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Jeśli brak zmiennej - zwracamy błąd. Aplikacja nie może bez tego działać.
		return nil, fmt.Errorf("required environment variable DATABASE_URL is missing")
	}

	// POST
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		DatabaseURL: dbURL,
		Port:        port,
	}, nil
}
