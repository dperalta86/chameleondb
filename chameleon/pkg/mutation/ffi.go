package mutation

import (
	"encoding/json"
	"fmt"

	"github.com/chameleon-db/chameleondb/chameleon/internal/ffi"
)

// ============================================================
// FFI WRAPPER FOR MUTATION SQL GENERATION
// ============================================================

// MutationSQL represents generated SQL from FFI
type MutationSQL struct {
	Valid    bool     `json:"valid"`
	SQL      string   `json:"sql"`
	Params   []string `json:"params"`
	Affected int      `json:"affected"`
	Error    string   `json:"error,omitempty"`
}

// GenerateMutationSQL calls Rust FFI to generate SQL for a mutation
//
// Calling patterns:
//  1. Stateless: Pass schema_json every time
//  2. Cached: Call SetSchemaCache() once, then pass nil for schema_json
//  3. Hybrid: Pass schema_json when available, falls back to cache
func GenerateMutationSQL(
	mutationType string,
	entity string,
	fields map[string]interface{},
	filters map[string]interface{},
	schemaJSON string,
) (*MutationSQL, error) {
	// Build mutation JSON
	mutation := map[string]interface{}{
		"type":   mutationType,
		"entity": entity,
	}

	if len(fields) > 0 {
		mutation["fields"] = fields
	}

	if len(filters) > 0 {
		mutation["filters"] = filters
	}

	mutationJSON, err := json.Marshal(mutation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mutation: %w", err)
	}

	// Call FFI
	// Note: If schemaJSON is "", FFI will use cached schema
	resultJSON := ffi.GenerateMutationSQL(string(mutationJSON), schemaJSON)

	// Parse result
	var result MutationSQL
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse FFI response: %w", err)
	}

	if !result.Valid {
		return nil, fmt.Errorf("mutation generation failed: %s", result.Error)
	}

	return &result, nil
}

// SetSchemaCache sets the schema cache in FFI for batch operations
// Call this once before batch mutations, then pass nil for schema_json in GenerateMutationSQL
func SetSchemaCache(schemaJSON string) error {
	resultJSON := ffi.SetSchemaCache(schemaJSON)

	var result struct {
		Valid bool   `json:"valid"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return fmt.Errorf("failed to parse FFI response: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("failed to set schema cache: %s", result.Error)
	}

	return nil
}

// ClearSchemaCache clears the schema cache in FFI
// Call this after batch operations to free memory
func ClearSchemaCache() error {
	resultJSON := ffi.ClearSchemaCache()

	var result struct {
		Valid bool   `json:"valid"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return fmt.Errorf("failed to parse FFI response: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("failed to clear schema cache: %s", result.Error)
	}

	return nil
}
