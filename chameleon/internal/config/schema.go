package config

import (
	"time"
)

// Config represents the complete .chameleon.yml configuration
type Config struct {
	Version   string         `yaml:"version"`
	CreatedAt time.Time      `yaml:"created_at"`
	Database  DatabaseConfig `yaml:"database"`
	Schema    SchemaConfig   `yaml:"schema"`
	Features  FeaturesConfig `yaml:"features"`
	Safety    SafetyConfig   `yaml:"safety"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Driver            string `yaml:"driver"`            // postgresql, mysql, sqlite
	ConnectionString  string `yaml:"connection_string"` // ${DATABASE_URL} or hardcoded
	MaxConnections    int    `yaml:"max_connections,omitempty"`
	ConnectionTimeout int    `yaml:"connection_timeout,omitempty"` // seconds
	MigrationTimeout  int    `yaml:"migration_timeout,omitempty"`  // seconds
}

// SchemaConfig holds schema management settings
type SchemaConfig struct {
	Paths            []string `yaml:"paths"`                       // Paths to schema directories
	MergedOutput     string   `yaml:"merged_output,omitempty"`     // Where to save merged schema
	ValidationStrict bool     `yaml:"validation_strict,omitempty"` // Fail on warnings
}

// FeaturesConfig holds feature flags
type FeaturesConfig struct {
	AutoMigration   bool `yaml:"auto_migration,omitempty"`    // Auto-apply migrations
	RollbackEnabled bool `yaml:"rollback_enabled,omitempty"`  // Allow rollbacks
	AuditLogging    bool `yaml:"audit_logging,omitempty"`     // Enable journal
	BackupOnMigrate bool `yaml:"backup_on_migrate,omitempty"` // Backup before applying
	DryRunDefault   bool `yaml:"dry_run_default,omitempty"`   // Default to --dry-run
}

// SafetyConfig holds safety settings
type SafetyConfig struct {
	RequireConfirmation bool `yaml:"require_confirmation,omitempty"` // Ask before apply
	BackupBeforeApply   bool `yaml:"backup_before_apply,omitempty"`  // Always backup
	ValidateSchema      bool `yaml:"validate_schema,omitempty"`      // Validate before apply
}

// Defaults returns a Config with sensible defaults
func Defaults() *Config {
	return &Config{
		Version:   "0.1.4",
		CreatedAt: time.Now(),
		Database: DatabaseConfig{
			Driver:            "postgresql",
			ConnectionString:  "${DATABASE_URL}",
			MaxConnections:    10,
			ConnectionTimeout: 30,
			MigrationTimeout:  300,
		},
		Schema: SchemaConfig{
			Paths:            []string{"./schemas"},
			MergedOutput:     ".chameleon/state/schema.merged.json",
			ValidationStrict: false,
		},
		Features: FeaturesConfig{
			AutoMigration:   true,
			RollbackEnabled: true,
			AuditLogging:    true,
			BackupOnMigrate: true,
			DryRunDefault:   false,
		},
		Safety: SafetyConfig{
			RequireConfirmation: false,
			BackupBeforeApply:   true,
			ValidateSchema:      true,
		},
	}
}

// Validate checks if config is valid
func (c *Config) Validate() error {
	if c.Database.Driver == "" {
		return &ConfigError{
			Field:  "database.driver",
			Reason: "Database driver is required",
		}
	}

	if len(c.Schema.Paths) == 0 {
		return &ConfigError{
			Field:  "schema.paths",
			Reason: "At least one schema path is required",
		}
	}

	if c.Database.ConnectionTimeout < 1 {
		c.Database.ConnectionTimeout = 30
	}

	if c.Database.MigrationTimeout < 1 {
		c.Database.MigrationTimeout = 300
	}

	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field      string
	Reason     string
	Suggestion string
}

func (e *ConfigError) Error() string {
	msg := "Configuration error: " + e.Field + ": " + e.Reason
	if e.Suggestion != "" {
		msg += "\nSuggestion: " + e.Suggestion
	}
	return msg
}
