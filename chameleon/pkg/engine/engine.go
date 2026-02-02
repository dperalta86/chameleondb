package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dperalta86/chameleondb/chameleon/internal/ffi"
)

// Engine is the main entry point for ChameleonDB
type Engine struct {
	schema    *Schema
	connector *Connector
	executor  *Executor
}

// NewEngine creates a new ChameleonDB engine
func NewEngine() *Engine {
	return &Engine{}
}

// LoadSchemaFromString parses a schema from a string
func (e *Engine) LoadSchemaFromString(schemaSource string) (*Schema, error) {
	// Parse via Rust FFI
	jsonStr, err := ffi.ParseSchema(schemaSource)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Validate
	if err := ffi.ValidateSchema(jsonStr); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Deserialize to Go struct
	schema, err := ParseSchemaJSON(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("json error: %w", err)
	}

	e.schema = schema
	return schema, nil
}

// LoadSchemaFromFile loads a schema from a .cham file
func (e *Engine) LoadSchemaFromFile(filepath string) (*Schema, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("read file error: %w", err)
	}

	return e.LoadSchemaFromString(string(data))
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
