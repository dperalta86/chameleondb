package engine

import (
	"fmt"
	"os"

	"github.com/dperalta86/chameleondb/chameleon/internal/ffi"
)

// Engine is the main entry point for ChameleonDB
type Engine struct {
	schema *Schema
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
