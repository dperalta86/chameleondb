package mutation

import (
	"fmt"
)

// ============================================================
// BASE ERROR INTERFACES
// ============================================================

// MutationError is the base interface for all mutation errors
type MutationError interface {
	error
	Code() string     // Error code for programmatic handling
	IsMutationError() // Marker method
}

// ============================================================
// VALIDATION ERRORS (Before SQL generation)
// ============================================================

// ValidationError: Schema/type/constraint validation failure
type ValidationError struct {
	Field    string      // "email", "age"
	Type     string      // "type_mismatch", "length_exceeded", "invalid_format"
	Value    interface{} // actual value provided
	Expected string      // "string(255)", "uuid", "int"
	Message  string      // User-friendly message
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf(
		"ValidationError: Field '%s' - %s\n"+
			"  Expected: %s\n"+
			"  Got: %v\n"+
			"  Details: %s",
		e.Field, e.Type, e.Expected, e.Value, e.Message,
	)
}

func (e *ValidationError) Code() string     { return "VALIDATION_ERROR" }
func (e *ValidationError) IsMutationError() {}

// ============================================================
// TYPE ERRORS
// ============================================================

// TypeMismatchError: Value doesn't match field type
type TypeMismatchError struct {
	Field        string
	ExpectedType string // "uuid", "int", "string"
	ReceivedType string // "string", "bool"
	Value        interface{}
	Suggestion   string // "Use uuid.Parse() to convert"
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf(
		"TypeMismatchError: Field '%s'\n"+
			"  Expected type: %s\n"+
			"  Received type: %s (value: %v)\n"+
			"  Suggestion: %s",
		e.Field, e.ExpectedType, e.ReceivedType, e.Value, e.Suggestion,
	)
}

func (e *TypeMismatchError) Code() string     { return "TYPE_MISMATCH" }
func (e *TypeMismatchError) IsMutationError() {}

// ============================================================
// LENGTH/FORMAT ERRORS
// ============================================================

// LengthExceededError: String exceeds max length
type LengthExceededError struct {
	Field  string
	MaxLen int
	Actual int
	Value  string
}

func (e *LengthExceededError) Error() string {
	return fmt.Sprintf(
		"LengthExceededError: Field '%s'\n"+
			"  Max length: %d characters\n"+
			"  Provided length: %d characters\n"+
			"  Value: %q",
		e.Field, e.MaxLen, e.Actual, e.Value,
	)
}

func (e *LengthExceededError) Code() string     { return "LENGTH_EXCEEDED" }
func (e *LengthExceededError) IsMutationError() {}

// FormatError: Invalid format (e.g., email, uuid)
type FormatError struct {
	Field      string
	Format     string // "uuid", "email", "iso8601"
	Value      string
	Suggestion string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf(
		"FormatError: Field '%s'\n"+
			"  Expected format: %s\n"+
			"  Provided value: %q\n"+
			"  Suggestion: %s",
		e.Field, e.Format, e.Value, e.Suggestion,
	)
}

func (e *FormatError) Code() string     { return "FORMAT_ERROR" }
func (e *FormatError) IsMutationError() {}

// ============================================================
// CONSTRAINT ERRORS (Data integrity)
// ============================================================

// ConstraintError: Generic constraint violation
type ConstraintError struct {
	Type       string // "unique", "not_null", "check", "foreign_key"
	Field      string
	Value      interface{}
	Suggestion string
}

func (e *ConstraintError) Error() string {
	return fmt.Sprintf(
		"ConstraintError: %s constraint violation\n"+
			"  Field: %s\n"+
			"  Value: %v\n"+
			"  Suggestion: %s",
		e.Type, e.Field, e.Value, e.Suggestion,
	)
}

func (e *ConstraintError) Code() string     { return fmt.Sprintf("%s_CONSTRAINT", e.Type) }
func (e *ConstraintError) IsMutationError() {}

// UniqueConstraintError: Value already exists (UNIQUE constraint)
type UniqueConstraintError struct {
	Field          string
	Value          interface{}
	ConflictingRow map[string]interface{} // The existing row
	Table          string
	Suggestion     string
}

func (e *UniqueConstraintError) Error() string {
	return fmt.Sprintf(
		"UniqueConstraintError: Field '%s' must be unique\n"+
			"  Value: %v\n"+
			"  Conflict: %s(id=%v) already has this value\n"+
			"  Suggestion: %s",
		e.Field, e.Value,
		e.Table, e.ConflictingRow["id"],
		e.Suggestion,
	)
}

func (e *UniqueConstraintError) Code() string     { return "UNIQUE_CONSTRAINT_VIOLATION" }
func (e *UniqueConstraintError) IsMutationError() {}

// NotNullError: Required field is null
type NotNullError struct {
	Field      string
	Suggestion string
}

func (e *NotNullError) Error() string {
	return fmt.Sprintf(
		"NotNullError: Field '%s' cannot be null\n"+
			"  This field is required\n"+
			"  Suggestion: %s",
		e.Field, e.Suggestion,
	)
}

func (e *NotNullError) Code() string     { return "NOT_NULL_VIOLATION" }
func (e *NotNullError) IsMutationError() {}

// ============================================================
// FOREIGN KEY ERRORS
// ============================================================

// ForeignKeyError: Referenced record doesn't exist
type ForeignKeyError struct {
	Field            string      // "authorId"
	Value            interface{} // "uuid-999"
	ReferencedTable  string      // "User"
	ReferencedField  string      // "id"
	ReferencedEntity string      // "User" (entity name)
	Suggestion       string
}

func (e *ForeignKeyError) Error() string {
	return fmt.Sprintf(
		"ForeignKeyError: Invalid reference\n"+
			"  Field: %s\n"+
			"  Referenced: %s(%s=%v)\n"+
			"  The referenced %s does not exist\n"+
			"  Suggestion: %s",
		e.Field,
		e.ReferencedTable, e.ReferencedField, e.Value,
		e.ReferencedEntity,
		e.Suggestion,
	)
}

func (e *ForeignKeyError) Code() string     { return "FOREIGN_KEY_VIOLATION" }
func (e *ForeignKeyError) IsMutationError() {}

// ForeignKeyConstraintError: Attempt to delete/update row with dependents
type ForeignKeyConstraintError struct {
	Entity         string      // "User"
	ID             interface{} // "uuid-123"
	DependentTable string      // "Post"
	DependentCount int
	Suggestion     string
}

func (e *ForeignKeyConstraintError) Error() string {
	return fmt.Sprintf(
		"ForeignKeyConstraintError: Cannot delete/update - dependents exist\n"+
			"  Entity: %s(id=%v)\n"+
			"  Dependent records: %d %s(s) reference this\n"+
			"  Suggestion: %s",
		e.Entity, e.ID,
		e.DependentCount, e.DependentTable,
		e.Suggestion,
	)
}

func (e *ForeignKeyConstraintError) Code() string     { return "FOREIGN_KEY_CONSTRAINT_VIOLATION" }
func (e *ForeignKeyConstraintError) IsMutationError() {}

// ============================================================
// SCHEMA ERRORS
// ============================================================

// UnknownFieldError: Field doesn't exist in schema
type UnknownFieldError struct {
	Entity    string
	Field     string
	Available []string // Valid field names
}

func (e *UnknownFieldError) Error() string {
	return fmt.Sprintf(
		"UnknownFieldError: Entity '%s' has no field '%s'\n"+
			"  Available fields: %v",
		e.Entity, e.Field, e.Available,
	)
}

func (e *UnknownFieldError) Code() string     { return "UNKNOWN_FIELD" }
func (e *UnknownFieldError) IsMutationError() {}

// UnknownEntityError: Entity doesn't exist in schema
type UnknownEntityError struct {
	Entity    string
	Available []string
}

func (e *UnknownEntityError) Error() string {
	return fmt.Sprintf(
		"UnknownEntityError: Entity '%s' not found in schema\n"+
			"  Available entities: %v",
		e.Entity, e.Available,
	)
}

func (e *UnknownEntityError) Code() string     { return "UNKNOWN_ENTITY" }
func (e *UnknownEntityError) IsMutationError() {}

// ============================================================
// EXECUTION ERRORS (After SQL generation)
// ============================================================

// NotFoundError: Record to update/delete doesn't exist
type NotFoundError struct {
	Entity string
	ID     interface{}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf(
		"NotFoundError: %s with id %v not found\n"+
			"  The record you're trying to update/delete doesn't exist",
		e.Entity, e.ID,
	)
}

func (e *NotFoundError) Code() string     { return "NOT_FOUND" }
func (e *NotFoundError) IsMutationError() {}

// ConflictError: Concurrent modification (optimistic locking - v0.2)
type ConflictError struct {
	Entity          string
	ID              interface{}
	ExpectedVersion int
	ActualVersion   int
	Suggestion      string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf(
		"ConflictError: Concurrent modification detected\n"+
			"  Entity: %s(id=%v)\n"+
			"  Expected version: %d\n"+
			"  Actual version: %d\n"+
			"  Suggestion: %s",
		e.Entity, e.ID, e.ExpectedVersion, e.ActualVersion, e.Suggestion,
	)
}

func (e *ConflictError) Code() string     { return "CONFLICT" }
func (e *ConflictError) IsMutationError() {}

// ============================================================
// SAFETY/PERMISSION ERRORS
// ============================================================

// SafetyError: Safety guard prevented operation
type SafetyError struct {
	Operation  string // "delete_without_filter", "delete_all", "large_update"
	Rows       int
	Threshold  int
	Message    string
	Suggestion string
}

func (e *SafetyError) Error() string {
	return fmt.Sprintf(
		"SafetyError: Operation blocked by safety guard\n"+
			"  Operation: %s\n"+
			"  Would affect: %d rows (threshold: %d)\n"+
			"  Message: %s\n"+
			"  Suggestion: %s",
		e.Operation, e.Rows, e.Threshold, e.Message, e.Suggestion,
	)
}

func (e *SafetyError) Code() string     { return "SAFETY_VIOLATION" }
func (e *SafetyError) IsMutationError() {}

// AuthorizationError: User not authorized (v0.2)
type AuthorizationError struct {
	Operation string
	Entity    string
	Message   string
}

func (e *AuthorizationError) Error() string {
	return fmt.Sprintf(
		"AuthorizationError: Not authorized\n"+
			"  Operation: %s on %s\n"+
			"  Message: %s",
		e.Operation, e.Entity, e.Message,
	)
}

func (e *AuthorizationError) Code() string     { return "AUTHORIZATION_DENIED" }
func (e *AuthorizationError) IsMutationError() {}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// IsMutationError checks if error is a mutation error
func IsMutationError(err error) bool {
	_, ok := err.(MutationError)
	return ok
}

// ErrorCode extracts the error code
func ErrorCode(err error) string {
	if me, ok := err.(MutationError); ok {
		return me.Code()
	}
	return "UNKNOWN_ERROR"
}

// IsSafetyError checks if error is a safety violation
func IsSafetyError(err error) bool {
	_, ok := err.(*SafetyError)
	return ok
}

// IsConstraintError checks if error is constraint-related
func IsConstraintError(err error) bool {
	switch err.(type) {
	case *UniqueConstraintError, *NotNullError, *ForeignKeyError, *ForeignKeyConstraintError:
		return true
	}
	return false
}
