package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"unsafe"

	"github.com/chameleon-db/chameleondb/chameleon/internal/ffi"
)

// Engine is the main entry point for ChameleonDB
type Engine struct {
	schema    *Schema
	connector *Connector
	executor  *Executor
	ffiHandle unsafe.Pointer

	// Debug context
	Debug *DebugContext
}

// NewEngine creates a new ChameleonDB engine
func NewEngine() *Engine {
	return &Engine{
		Debug: DefaultDebugContext(),
	}
}

// WithDebug returns a new engine with debug enabled
func (e *Engine) WithDebug(level DebugLevel) *Engine {
	e.Debug = &DebugContext{
		Level:       level,
		Writer:      os.Stdout,
		ColorOutput: true,
	}
	return e
}

// LoadSchemaFromString parses a schema from a string
func (e *Engine) LoadSchemaFromString(input string) (*Schema, error) {
	schemaJSON, err := ffi.ParseSchema(input)
	if err != nil {
		// Check if it's a structured parse error (JSON)
		formattedErr := FormatError(err.Error())
		return nil, fmt.Errorf("%s", formattedErr)
	}

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, fmt.Errorf("failed to deserialize schema: %w", err)
	}

	e.schema = &schema
	return &schema, nil
}

// LoadSchemaFromFile loads a schema from a .cham file
func (e *Engine) LoadSchemaFromFile(filepath string) (*Schema, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}
	return e.LoadSchemaFromString(string(content))
}

// GetSchema returns the currently loaded schema
func (e *Engine) GetSchema() *Schema {
	return e.schema
}

// Version returns the engine version
func (e *Engine) Version() string {
	return ffi.Version()
}

// Connect establishes a database connection
func (e *Engine) Connect(ctx context.Context, config ConnectorConfig) error {
	e.connector = NewConnector(config)
	if err := e.connector.Connect(ctx); err != nil {
		return err
	}
	e.executor = NewExecutor(e.connector)
	return nil
}

// Close closes the database connection
func (e *Engine) Close() {
	if e.connector != nil {
		e.connector.Close()
	}
}

// IsConnected returns true if connected to a database
func (e *Engine) IsConnected() bool {
	return e.connector != nil && e.connector.IsConnected()
}

// Ping verifies the database connection is alive
func (e *Engine) Ping(ctx context.Context) error {
	if e.connector == nil {
		return fmt.Errorf("not connected")
	}
	return e.connector.Ping(ctx)
}

// GenerateMigration generates DDL SQL from the loaded schema
func (e *Engine) GenerateMigration() (string, error) {
	if e.schema == nil {
		return "", fmt.Errorf("no schema loaded")
	}

	schemaJSON, err := json.Marshal(e.schema)
	if err != nil {
		return "", fmt.Errorf("failed to serialize schema: %w", err)
	}

	return ffi.GenerateMigration(string(schemaJSON))
}

// GetEntity returns an entity by name, or nil if not found
func (s *Schema) GetEntity(name string) *Entity {
	for _, entity := range s.Entities {
		if entity.Name == name {
			return entity
		}
	}
	return nil
}
