package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dperalta86/chameleondb/chameleon/internal/ffi"
)

// --- Query types (mirror Rust Query AST) ---

/*
	 type FilterValue struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}
*/
type FilterValue map[string]interface{}

type FilterCondition struct {
	Field FieldPath   `json:"field"`
	Op    string      `json:"op"` // "Eq", "Neq", "Gt", etc.
	Value FilterValue `json:"value"`
}

type FieldPath struct {
	Segments []string `json:"segments"`
}

type FilterExpr struct {
	Condition *FilterCondition `json:"Condition,omitempty"`
	Binary    *BinaryExpr      `json:"Binary,omitempty"`
}

type BinaryExpr struct {
	Left  FilterExpr `json:"left"`
	Op    string     `json:"op"` // "And", "Or"
	Right FilterExpr `json:"right"`
}

type IncludePath struct {
	Path []string `json:"path"`
}

type OrderByClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "Asc", "Desc"
}

// QueryJSON is the serialization format matching Rust's Query
type QueryJSON struct {
	Entity   string          `json:"entity"`
	Filters  []FilterExpr    `json:"filters"`
	Includes []IncludePath   `json:"includes"`
	OrderBy  []OrderByClause `json:"order_by"`
	Limit    *uint64         `json:"limit"`
	Offset   *uint64         `json:"offset"`
}

// GeneratedSQL mirrors Rust's GeneratedSQL
type GeneratedSQL struct {
	MainQuery    string     `json:"main_query"`
	EagerQueries [][]string `json:"eager_queries"`
}

type EagerQuery struct {
	Relation string `json:"0"`
	SQL      string `json:"1"`
}

// --- Query Builder ---

// QueryBuilder provides a chainable API for building queries
type QueryBuilder struct {
	engine *Engine
	query  QueryJSON
}

// Query starts a new query for the given entity
func (e *Engine) Query(entity string) *QueryBuilder {
	return &QueryBuilder{
		engine: e,
		query: QueryJSON{
			Entity:   entity,
			Filters:  []FilterExpr{},
			Includes: []IncludePath{},
			OrderBy:  []OrderByClause{},
		},
	}
}

// Filter adds a filter condition
// field: "email" or "orders.total" (supports relation navigation)
// op: "eq", "neq", "gt", "gte", "lt", "lte", "like"
// value: string, int, float, or bool
func (qb *QueryBuilder) Filter(field string, op string, value interface{}) *QueryBuilder {
	rustOp := goOpToRust(op)

	qb.query.Filters = append(qb.query.Filters, FilterExpr{
		Condition: &FilterCondition{
			Field: parseFieldPath(field),
			Op:    rustOp,
			Value: goValueToFilter(value),
		},
	})
	return qb
}

// Include adds eager loading for a relation
// Supports nested paths: "orders", "orders.items"
func (qb *QueryBuilder) Include(path string) *QueryBuilder {
	qb.query.Includes = append(qb.query.Includes, IncludePath{
		Path: splitPath(path),
	})
	return qb
}

// OrderBy adds a sort clause
// direction: "asc" or "desc"
func (qb *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	dir := "Asc"
	if direction == "desc" {
		dir = "Desc"
	}
	qb.query.OrderBy = append(qb.query.OrderBy, OrderByClause{
		Field:     field,
		Direction: dir,
	})
	return qb
}

// Limit sets the maximum number of results
func (qb *QueryBuilder) Limit(n uint64) *QueryBuilder {
	qb.query.Limit = &n
	return qb
}

// Offset sets the number of results to skip
func (qb *QueryBuilder) Offset(n uint64) *QueryBuilder {
	qb.query.Offset = &n
	return qb
}

// ToSQL generates SQL without executing
// Useful for debugging and testing
func (qb *QueryBuilder) ToSQL() (*GeneratedSQL, error) {
	if qb.engine.schema == nil {
		return nil, fmt.Errorf("no schema loaded")
	}

	// Serialize query
	queryJSON, err := json.Marshal(qb.query)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize query: %w", err)
	}

	// Serialize schema
	schemaJSON, err := json.Marshal(qb.engine.schema)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize schema: %w", err)
	}

	// Call Rust SQL generator via FFI
	resultJSON, err := ffi.GenerateSQL(string(queryJSON), string(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("SQL generation failed: %w", err)
	}

	// Parse result
	var result GeneratedSQL
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse generated SQL: %w", err)
	}

	return &result, nil
}

// Execute generates SQL and runs it against the database
func (qb *QueryBuilder) Execute(ctx context.Context) (*QueryResult, error) {
	if qb.engine.executor == nil {
		return nil, fmt.Errorf("not connected to database, call Engine.Connect() first")
	}
	return qb.engine.executor.Execute(ctx, qb)
}

// --- Helpers ---
func parseFieldPath(path string) FieldPath {
	return FieldPath{Segments: splitPath(path)}
}

func splitPath(path string) []string {
	segments := []string{}
	current := ""
	for _, ch := range path {
		if ch == '.' {
			if current != "" {
				segments = append(segments, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		segments = append(segments, current)
	}
	return segments
}

func goOpToRust(op string) string {
	ops := map[string]string{
		"eq":   "Eq",
		"neq":  "Neq",
		"gt":   "Gt",
		"gte":  "Gte",
		"lt":   "Lt",
		"lte":  "Lte",
		"like": "Like",
		"in":   "In",
	}
	if rustOp, ok := ops[op]; ok {
		return rustOp
	}
	return op
}

func goValueToFilter(value interface{}) FilterValue {
	switch v := value.(type) {
	case string:
		return FilterValue{"String": v}
	case int:
		return FilterValue{"Int": v}
	case int64:
		return FilterValue{"Int": v}
	case float64:
		return FilterValue{"Float": v}
	case bool:
		return FilterValue{"Bool": v}
	case nil:
		return FilterValue{"Null": nil}
	default:
		return FilterValue{"String": fmt.Sprintf("%v", v)}
	}
}
