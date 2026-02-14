package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles loading and parsing .chameleon.yml
type Loader struct {
	filePath string
	workDir  string
}

// NewLoader creates a new config loader
func NewLoader(workDir string) *Loader {
	return &Loader{
		filePath: filepath.Join(workDir, ".chameleon.yml"),
		workDir:  workDir,
	}
}

// Load reads and parses .chameleon.yml
func (l *Loader) Load() (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(l.filePath); err != nil {
		return nil, fmt.Errorf("config file not found: %s\nRun 'chameleon init' to create one", l.filePath)
	}

	// Read file
	data, err := os.ReadFile(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand environment variables in connection string
	cfg.Database.ConnectionString = os.ExpandEnv(cfg.Database.ConnectionString)

	// Resolve relative paths
	if err := l.resolvePaths(&cfg); err != nil {
		return nil, err
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// resolvePaths converts relative paths to absolute
func (l *Loader) resolvePaths(cfg *Config) error {
	// Resolve schema paths
	for i, path := range cfg.Schema.Paths {
		abs, err := l.resolvePath(path)
		if err != nil {
			return fmt.Errorf("invalid schema path '%s': %w", path, err)
		}
		cfg.Schema.Paths[i] = abs
	}

	// Resolve merged output path
	if cfg.Schema.MergedOutput != "" {
		abs, err := l.resolvePath(cfg.Schema.MergedOutput)
		if err != nil {
			return fmt.Errorf("invalid merged_output path '%s': %w", cfg.Schema.MergedOutput, err)
		}
		cfg.Schema.MergedOutput = abs
	}

	return nil
}

// resolvePath converts relative or absolute path to absolute
func (l *Loader) resolvePath(path string) (string, error) {
	// Already absolute
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Relative to work directory
	abs := filepath.Join(l.workDir, path)

	// Normalize
	abs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}

	return abs, nil
}

// LoadOrDefault loads config or returns defaults
func (l *Loader) LoadOrDefault() (*Config, error) {
	cfg, err := l.Load()
	if err != nil {
		// Not found = return defaults
		if strings.Contains(err.Error(), "not found") {
			return Defaults(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// Save writes config to file
func (l *Loader) Save(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(l.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Template returns the template content for .chameleon.yml
func Template() string {
	return `# ChameleonDB Configuration
# Generated at {{.CreatedAt}}

version: "0.1.4"
created_at: {{.CreatedAt}}

# Database connection settings
database:
  driver: "postgresql"
  # Use environment variable
  connection_string: ${DATABASE_URL}
  # OR hardcode (not recommended for production)
  # connection_string: "postgresql://localhost:5432/myapp_dev"
  
  # Connection pool settings
  max_connections: 10
  connection_timeout: 30  # seconds
  migration_timeout: 300  # seconds

# Schema management
schema:
  # Paths to schema directories (relative or absolute)
  paths:
    - "./schemas"
  
  # Where to save merged schema (for reference)
  merged_output: ".chameleon/state/schema.merged.json"
  
  # Fail on validation warnings
  validation_strict: false

# Feature flags
features:
  # Auto-apply pending migrations
  auto_migration: true
  
  # Allow rollbacks
  rollback_enabled: true
  
  # Enable audit logging
  audit_logging: true
  
  # Create backup before each migration
  backup_on_migrate: true
  
  # Default to --dry-run mode
  dry_run_default: false

# Safety settings
safety:
  # Require confirmation before applying migrations
  require_confirmation: false
  
  # Always create backup before applying
  backup_before_apply: true
  
  # Validate schema before applying
  validate_schema: true
`
}
