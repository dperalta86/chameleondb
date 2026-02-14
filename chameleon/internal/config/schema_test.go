package config

import (
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Version != "0.1.4" {
		t.Errorf("Expected version 0.1.4, got %s", cfg.Version)
	}

	if len(cfg.Schema.Paths) == 0 {
		t.Error("Expected schema paths")
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Driver: "postgresql",
		},
		Schema: SchemaConfig{
			Paths: []string{"./schemas"},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}
