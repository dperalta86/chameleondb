package main

import (
	"fmt"
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// LoadConnectorConfig loads database config from:
// 1. DATABASE_URL environment variable (priority)
// 2. .chameleon.yml file in current directory (v0.1.5+)
// 3. Default configuration (localhost:5432)
func LoadConnectorConfig() (engine.ConnectorConfig, error) {
	// 1. Try DATABASE_URL env var (Heroku, Railway, Docker, etc.)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		parsedConfig, err := engine.ParseConnectionString(databaseURL)
		if err != nil {
			return engine.ConnectorConfig{}, fmt.Errorf("invalid DATABASE_URL: %w", err)
		}
		if verbose {
			printInfo("Using DATABASE_URL from environment")
		}
		return parsedConfig, nil
	}

	// 2. Try .chameleon.yml file (v0.1.5+)
	workDir, err := os.Getwd()
	if err != nil {
		return engine.ConnectorConfig{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	loader := config.NewLoader(workDir)
	cfg, err := loader.Load()
	if err == nil {
		// Config loaded successfully
		if verbose {
			printInfo("Using .chameleon.yml configuration")
		}

		// Parse connection string from config
		connStr := cfg.Database.ConnectionString
		if connStr == "" {
			// Fallback to defaults if connection string is empty (e.g., ${DATABASE_URL} not set)
			if verbose {
				printInfo("Connection string empty in config, using defaults")
			}
			return engine.DefaultConfig(), nil
		}

		parsed, err := engine.ParseConnectionString(connStr)
		if err != nil {
			return engine.ConnectorConfig{}, fmt.Errorf("invalid connection string in .chameleon.yml: %w", err)
		}

		return parsed, nil
	}

	// 3. Return defaults
	if verbose {
		printInfo("Using default configuration (localhost:5432)")
	}
	return engine.DefaultConfig(), nil
}
