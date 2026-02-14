package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CurrentState represents the current state of the database
type CurrentState struct {
	Version    string          `json:"version"`
	Timestamp  time.Time       `json:"timestamp"`
	Database   DatabaseState   `json:"database"`
	Schema     SchemaState     `json:"schema"`
	Migrations MigrationsState `json:"migrations"`
	Status     string          `json:"status"` // in_sync, pending_migration, conflict
	Validation ValidationState `json:"validation"`
}

// DatabaseState holds database metadata
type DatabaseState struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

// SchemaState holds schema metadata
type SchemaState struct {
	EntityCount       int    `json:"entity_count"`
	FieldCount        int    `json:"field_count"`
	RelationshipCount int    `json:"relationship_count"`
	MergedHash        string `json:"merged_hash"` // sha256 of merged schema
}

// MigrationsState tracks migration status
type MigrationsState struct {
	AppliedCount  int       `json:"applied_count"`
	LastApplied   string    `json:"last_applied"`
	LastAppliedAt time.Time `json:"last_applied_at"`
}

// ValidationState holds validation results
type ValidationState struct {
	Status string   `json:"status"` // valid, invalid
	Errors []string `json:"errors"`
}

// Migration represents a single migration record
type Migration struct {
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // initial, alter, drop
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
	Status      string    `json:"status"` // applied, rolled_back, pending
	SchemaHash  string    `json:"schema_hash"`
	DDLHash     string    `json:"ddl_hash"`
	Checksum    string    `json:"checksum"` // verified, pending
	Backups     []Backup  `json:"backups"`
}

// Backup represents a backup record
type Backup struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Verified  bool   `json:"verified"`
	CheckSum  string `json:"checksum"`
}

// Manifest holds all migration history
type Manifest struct {
	Migrations []*Migration `json:"migrations"`
}

// Tracker manages state files
type Tracker struct {
	stateDir string
}

// NewTracker creates a new state tracker
func NewTracker(stateDir string) (*Tracker, error) {
	// Create directory if not exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return &Tracker{
		stateDir: stateDir,
	}, nil
}

// LoadCurrent loads the current state
func (t *Tracker) LoadCurrent() (*CurrentState, error) {
	stateFile := filepath.Join(t.stateDir, "current.state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state (no migrations applied yet)
			return &CurrentState{
				Status: "pending_migration",
				Validation: ValidationState{
					Status: "valid",
					Errors: []string{},
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to read state: %w", err)
	}

	var state CurrentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	return &state, nil
}

// SaveCurrent saves the current state
func (t *Tracker) SaveCurrent(state *CurrentState) error {
	state.Timestamp = time.Now()
	state.Version = "0.1.4"

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	stateFile := filepath.Join(t.stateDir, "current.state.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// LoadManifest loads migration manifest
func (t *Tracker) LoadManifest() (*Manifest, error) {
	manifestFile := filepath.Join(t.stateDir, "migrations", "manifest.json")

	data, err := os.ReadFile(manifestFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Migrations: []*Migration{}}, nil
		}
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// SaveManifest saves migration manifest
func (t *Tracker) SaveManifest(manifest *Manifest) error {
	// Create migrations directory
	migrationsDir := filepath.Join(t.stateDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestFile := filepath.Join(migrationsDir, "manifest.json")
	if err := os.WriteFile(manifestFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// AddMigration adds a migration to the manifest
func (t *Tracker) AddMigration(migration *Migration) error {
	manifest, err := t.LoadManifest()
	if err != nil {
		return err
	}

	manifest.Migrations = append(manifest.Migrations, migration)

	return t.SaveManifest(manifest)
}

// GetLastMigration returns the last applied migration
func (t *Tracker) GetLastMigration() (*Migration, error) {
	manifest, err := t.LoadManifest()
	if err != nil {
		return nil, err
	}

	if len(manifest.Migrations) == 0 {
		return nil, nil
	}

	// Return last applied migration
	for i := len(manifest.Migrations) - 1; i >= 0; i-- {
		if manifest.Migrations[i].Status == "applied" {
			return manifest.Migrations[i], nil
		}
	}

	return nil, nil
}

// HashSchema computes SHA256 hash of schema
func HashSchema(schema string) string {
	hash := sha256.Sum256([]byte(schema))
	return hex.EncodeToString(hash[:])
}

// HashDDL computes SHA256 hash of DDL
func HashDDL(ddl string) string {
	hash := sha256.Sum256([]byte(ddl))
	return hex.EncodeToString(hash[:])
}
