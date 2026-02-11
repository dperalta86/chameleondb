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

	// Mutation factory (abstract, injected)
	mutations MutationFactory
}

func (e *Engine) Schema() *Schema {
	panic("unimplemented")
}

// ============================================================
// ENGINE INITIALIZATION
// ============================================================

// NewEngine creates and initializes a new ChameleonDB engine
//
// Default behavior:
//   - Loads schema from "schema.cham" if it exists
//   - Auto-initializes mutation factory
//   - Ready to use immediately
//
// If "schema.cham" doesn't exist, returns engine without schema
// (user must call LoadSchemaFromFile manually)
func NewEngine() *Engine {
	return newEngineWithPath("schema.cham")
}

// NewEngineWithSchema creates and initializes engine with a specific schema file
//
// Usage:
//
//	engine, err := engine.NewEngineWithSchema("path/to/my_schema.cham")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result := engine.Insert("User").Set("email", "ana@mail.com").Execute(ctx)
func NewEngineWithSchema(schemaPath string) (*Engine, error) {
	eng := newEngineWithPath(schemaPath)

	// If file doesn't exist, return error explicitly
	if _, err := os.Stat(schemaPath); err != nil {
		return nil, fmt.Errorf(
			"schema file not found: %s\n"+
				"Create a schema.cham file or use engine.NewEngineWithoutSchema()",
			schemaPath,
		)
	}

	return eng, nil
}

// NewEngineWithoutSchema creates an engine without loading a schema
// User must call LoadSchemaFromFile or LoadSchemaFromString manually
func NewEngineWithoutSchema() *Engine {
	return &Engine{
		Debug: DefaultDebugContext(),
	}
}

// newEngineWithPath is the internal helper
func newEngineWithPath(schemaPath string) *Engine {
	eng := &Engine{
		Debug: DefaultDebugContext(),
	}

	// Try to load schema silently (don't fail if missing)
	if _, err := os.Stat(schemaPath); err == nil {
		if _, err := eng.LoadSchemaFromFile(schemaPath); err == nil {
			// Initialize mutation factory after schema load
			//eng.SetMutationFactory(mutation.NewFactory(schema))
		}
	}

	return eng
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

// ─────────────────────────────────────────────────────────────
// Schema handling
// ─────────────────────────────────────────────────────────────
//

// LoadSchemaFromString parses a schema from a string
func (e *Engine) LoadSchemaFromString(input string) (*Schema, error) {
	schemaJSON, err := ffi.ParseSchema(input)
	if err != nil {
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

//
// ─────────────────────────────────────────────────────────────
// Connection handling
// ─────────────────────────────────────────────────────────────
//

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

//
// ─────────────────────────────────────────────────────────────
// Migrations
// ─────────────────────────────────────────────────────────────
//

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

// ─────────────────────────────────────────────────────────────
// Mutation wiring (NO concrete dependencies)
// ─────────────────────────────────────────────────────────────

// SetMutationFactory injects a mutation factory implementation
func (e *Engine) SetMutationFactory(factory MutationFactory) {
	e.mutations = factory
}

func (e *Engine) ensureMutationFactory() {
	if e.mutations == nil {
		panic(
			"mutation factory not initialized\n" +
				"Call mutation.InitFactory(engine) after loading schema",
		)
	}
}

// Insert starts a new INSERT mutation
func (e *Engine) Insert(entity string) InsertMutation {
	e.ensureSchemaLoaded()
	e.ensureMutationFactory()
	return e.mutations.NewInsert(entity)
}

// Update starts a new UPDATE mutation
func (e *Engine) Update(entity string) UpdateMutation {
	e.ensureSchemaLoaded()
	e.ensureMutationFactory()
	return e.mutations.NewUpdate(entity)
}

// Delete starts a new DELETE mutation
func (e *Engine) Delete(entity string) DeleteMutation {
	e.ensureSchemaLoaded()
	e.ensureMutationFactory()
	return e.mutations.NewDelete(entity)
}

// ─────────────────────────────────────────────────────────────
// Schema helpers
// ─────────────────────────────────────────────────────────────

// GetEntity returns an entity by name, or nil if not found
func (s *Schema) GetEntity(name string) *Entity {
	for _, entity := range s.Entities {
		if entity.Name == name {
			return entity
		}
	}
	return nil
}

func (e *Engine) ensureSchemaLoaded() {
	if e.schema == nil {
		panic("schema not loaded")
	}
}
