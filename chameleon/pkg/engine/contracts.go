package engine

import "context"

// ============================================================
// MUTATION TYPES
// ============================================================

type MutationType int

const (
	MutationInsert MutationType = iota
	MutationUpdate
	MutationDelete
)

type Mutation struct {
	Type         MutationType
	Entity       string
	HasFilter    bool
	AffectedRows int64
}

// ============================================================
// MUTATION RESULT TYPES
// ============================================================

type InsertResult struct {
	ID       interface{}            // Primary key
	Record   map[string]interface{} // Full record (if RETURNING)
	Affected int
}

type UpdateResult struct {
	Records  []map[string]interface{}
	Affected int
}

type DeleteResult struct {
	Affected int
}

// ============================================================
// MUTATION BUILDER INTERFACES
// ============================================================

// InsertMutation builds and executes INSERT operations
type InsertMutation interface {
	// Set adds a field to insert
	Set(field string, value interface{}) InsertMutation

	// Execute validates and runs the mutation
	Execute(ctx context.Context) (*InsertResult, error)
}

// UpdateMutation builds and executes UPDATE operations
type UpdateMutation interface {
	// Set adds a field to update
	Set(field string, value interface{}) UpdateMutation

	// Filter adds a filter condition (WHERE clause)
	// Field, operator, value format for type-safety
	// Operators: eq, neq, gt, gte, lt, lte, like, in
	Filter(field string, operator string, value interface{}) UpdateMutation

	// Execute validates and runs the mutation
	Execute(ctx context.Context) (*UpdateResult, error)
}

// DeleteMutation builds and executes DELETE operations
type DeleteMutation interface {
	// Filter adds a filter condition (WHERE clause)
	// Field, operator, value format for type-safety
	Filter(field string, operator string, value interface{}) DeleteMutation

	// Execute validates and runs the mutation
	Execute(ctx context.Context) (*DeleteResult, error)
}

// ============================================================
// FACTORY
// ============================================================

// MutationFactory creates mutation builders
//
// Factory is initialized once per Engine with the schema.
// Engine delegates all mutation creation to this factory.
//
// This allows multiple implementations:
//   - SQL mutations (v0.1)
//   - GraphQL mutations (v0.2)
//   - REST mutations (v0.2)
//   - Custom mutations (via SetMutationFactory)
type MutationFactory interface {
	// NewInsert creates a builder for INSERT operations
	NewInsert(entity string) InsertMutation

	// NewUpdate creates a builder for UPDATE operations
	NewUpdate(entity string) UpdateMutation

	// NewDelete creates a builder for DELETE operations
	NewDelete(entity string) DeleteMutation
}

// ============================================================
// AUXILIARY CONTRACTS (for future use)
// ============================================================

/* // Executor runs SQL against a database
// (Can be swapped for different backends: PostgreSQL, MySQL, SQLite, etc)
type Executor interface {
	Execute(ctx context.Context, sql string, params ...interface{}) (*ExecutionResult, error)
} */

type ExecutionResult struct {
	RowsAffected int64
	LastInsertID interface{}
	Rows         []map[string]interface{}
}

/* // Validator checks mutation inputs for correctness
type Validator interface {
	ValidateInsertInput(entity string, fields map[string]interface{}) error
	ValidateUpdateInput(entity string, filters map[string]interface{}, updates map[string]interface{}) error
	ValidateDeleteInput(entity string, filters map[string]interface{}, forceAll bool) error
} */
