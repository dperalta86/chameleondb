package mutation

import (
	"context"
	"testing"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// Helper: create test schema
func testSchema() *engine.Schema {
	schema := &engine.Schema{
		Entities: []*engine.Entity{
			{
				Name: "User",
				Fields: map[string]*engine.Field{
					"id": {
						Name:       "id",
						Type:       engine.FieldType{Kind: "UUID"},
						Nullable:   false,
						Unique:     false,
						PrimaryKey: true,
						Default:    nil,
						Backend:    nil,
					},
					"email": {
						Name:       "email",
						Type:       engine.FieldType{Kind: "String"},
						Nullable:   false,
						Unique:     true,
						PrimaryKey: false,
						Default:    nil,
						Backend:    nil,
					},
					"name": {
						Name:       "name",
						Type:       engine.FieldType{Kind: "String"},
						Nullable:   false,
						Unique:     false,
						PrimaryKey: false,
						Default:    nil,
						Backend:    nil,
					},
					"age": {
						Name:       "age",
						Type:       engine.FieldType{Kind: "Int"},
						Nullable:   true,
						Unique:     false,
						PrimaryKey: false,
						Default:    nil,
						Backend:    nil,
					},
				},
			},
		},
	}
	return schema
}

// ============================================================
// INSERT BUILDER TESTS
// ============================================================

func TestInsertBuilder_Set(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

	// Test chainable API
	result := builder.Set("email", "ana@mail.com").Set("name", "Ana")

	if result != builder {
		t.Error("Set() should return builder for chaining")
	}

	if builder.values["email"] != "ana@mail.com" {
		t.Errorf("Expected email='ana@mail.com', got '%v'", builder.values["email"])
	}

	if builder.values["name"] != "Ana" {
		t.Errorf("Expected name='Ana', got '%v'", builder.values["name"])
	}
}

func TestInsertBuilder_Debug(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

	result := builder.Debug()

	if result != builder {
		t.Error("Debug() should return builder")
	}

	if !builder.debug {
		t.Error("Debug flag not set")
	}
}

func TestInsertBuilder_DryRun(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

	result := builder.DryRun()

	if result != builder {
		t.Error("DryRun() should return builder")
	}

	if !builder.dryRun {
		t.Error("DryRun flag not set")
	}
}

func TestInsertBuilder_Build(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("email", "ana@mail.com").Set("name", "Ana")

	mutation, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if mutation.Type != engine.MutationInsert {
		t.Errorf("Expected MutationInsert, got %v", mutation.Type)
	}

	if mutation.Entity != "User" {
		t.Errorf("Expected entity='User', got '%s'", mutation.Entity)
	}
}

func TestInsertBuilder_Build_InvalidEntity(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "NonExistent")
	builder.Set("field", "value")

	_, err := builder.Build()

	if err == nil {
		t.Error("Build() should fail for non-existent entity")
	}
}

func TestInsertBuilder_Build_UnknownField(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("unknown_field", "value")

	_, err := builder.Build()

	if err == nil {
		t.Error("Build() should fail for unknown field")
	}
}

func TestInsertBuilder_Execute(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("email", "ana@mail.com").Set("name", "Ana")

	result, err := builder.Execute(context.Background())

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if result == nil {
		t.Error("Execute() returned nil result")
	}

	if !result.DryRun && result.SQL == "" {
		t.Error("Execute() should generate SQL")
	}
}

func TestInsertBuilder_DryRun_NoExecution(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")
	builder.Set("email", "ana@mail.com")
	builder.Set("name", "Ana")
	builder.DryRun()

	result, err := builder.Execute(context.Background())

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if !result.DryRun {
		t.Error("Result should have DryRun=true")
	}
}

// ============================================================
// UPDATE BUILDER TESTS
// ============================================================

func TestUpdateBuilder_Filter(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	result := builder.Filter("id", "eq", "uuid-123")

	if result != builder {
		t.Error("Filter() should return builder for chaining")
	}

	if len(builder.filters) == 0 {
		t.Error("Filter() should add filter")
	}
}

func TestUpdateBuilder_Set(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	result := builder.Set("name", "Ana")

	if result != builder {
		t.Error("Set() should return builder")
	}

	if builder.updates["name"] != "Ana" {
		t.Errorf("Expected name='Ana', got '%v'", builder.updates["name"])
	}
}

func TestUpdateBuilder_Filter_And_Set(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	builder.Filter("id", "eq", "uuid-123").Set("name", "Ana").Set("age", 30)

	if len(builder.filters) == 0 {
		t.Error("Filter not added")
	}

	if len(builder.updates) != 2 {
		t.Errorf("Expected 2 updates, got %d", len(builder.updates))
	}
}

func TestUpdateBuilder_Build(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123").Set("name", "Ana")

	mutation, err := builder.Build()

	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if mutation.Type != engine.MutationUpdate {
		t.Errorf("Expected MutationUpdate, got %v", mutation.Type)
	}

	if !mutation.HasFilter {
		t.Error("Update should have WHERE clause")
	}
}

func TestUpdateBuilder_Build_NoFilter(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Set("name", "Ana")

	_, err := builder.Build()

	if err == nil {
		t.Error("Build() should fail without filter (safety guard)")
	}
}

func TestUpdateBuilder_Build_UpdatePrimaryKey(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Filter("email", "eq", "old@mail.com").Set("id", "new-uuid")

	_, err := builder.Build()

	if err == nil {
		t.Error("Build() should fail when updating primary key")
	}
}

func TestUpdateBuilder_ForceUpdateAll(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	result := builder.ForceUpdateAll()

	if result != builder {
		t.Error("ForceUpdateAll() should return builder")
	}

	if !builder.forceAll {
		t.Error("forceAll flag should be set")
	}
}

func TestUpdateBuilder_Exec(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123").Set("name", "Ana")

	err := builder.Exec()

	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	if builder.sql == "" {
		t.Error("Exec() should generate SQL")
	}
}

// ============================================================
// DELETE BUILDER TESTS
// ============================================================

func TestDeleteBuilder_Filter(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	result := builder.Filter("id", "eq", "uuid-123")

	if result != builder {
		t.Error("Filter() should return builder")
	}

	if len(builder.filters) == 0 {
		t.Error("Filter() should add filter")
	}
}

func TestDeleteBuilder_Build(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123")

	mutation, err := builder.Build()

	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if mutation.Type != engine.MutationDelete {
		t.Errorf("Expected MutationDelete, got %v", mutation.Type)
	}

	if !mutation.HasFilter {
		t.Error("Delete should have WHERE clause")
	}
}

func TestDeleteBuilder_Build_NoFilter(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	_, err := builder.Build()

	if err == nil {
		t.Error("Build() should fail without filter (safety guard)")
	}
}

func TestDeleteBuilder_ForceDeleteAll(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	result := builder.ForceDeleteAll()

	if result != builder {
		t.Error("ForceDeleteAll() should return builder")
	}

	if !builder.forceDeleteAll {
		t.Error("forceDeleteAll flag should be set")
	}
}

func TestDeleteBuilder_Build_ForceDeleteAll(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")
	builder.ForceDeleteAll()

	mutation, err := builder.Build()

	if err != nil {
		t.Fatalf("Build() with ForceDeleteAll should succeed: %v", err)
	}

	if mutation.Type != engine.MutationDelete {
		t.Errorf("Expected MutationDelete, got %v", mutation.Type)
	}
}

func TestDeleteBuilder_Exec(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")
	builder.Filter("id", "eq", "uuid-123")

	err := builder.Exec()

	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	if builder.sql == "" {
		t.Error("Exec() should generate SQL")
	}
}

// ============================================================
// CHAINING TESTS
// ============================================================

func TestInsertBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	result, err := NewInsertBuilder(schema, "User").
		Set("email", "ana@mail.com").
		Set("name", "Ana").
		Set("age", 28).
		Debug().
		DryRun().
		Execute(context.Background())

	if err != nil {
		t.Fatalf("Chaining failed: %v", err)
	}

	if result == nil {
		t.Error("Result is nil")
	}
}

func TestUpdateBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	builder := NewUpdateBuilder(schema, "User").
		Filter("id", "eq", "uuid-123").
		Set("name", "Ana").
		Set("age", 30).
		Debug().
		DryRun()

	err := builder.Exec()

	if err != nil {
		t.Fatalf("Chaining failed: %v", err)
	}

	if builder.sql == "" {
		t.Error("SQL not generated")
	}
}

func TestDeleteBuilder_Chaining(t *testing.T) {
	schema := testSchema()

	builder := NewDeleteBuilder(schema, "User").
		Filter("id", "eq", "uuid-123").
		Debug().
		DryRun()

	err := builder.Exec()

	if err != nil {
		t.Fatalf("Chaining failed: %v", err)
	}

	if builder.sql == "" {
		t.Error("SQL not generated")
	}
}

// ============================================================
// EDGE CASES
// ============================================================

func TestInsertBuilder_MultipleValues(t *testing.T) {
	schema := testSchema()
	builder := NewInsertBuilder(schema, "User")

	// Set same field twice (last one wins)
	builder.Set("name", "Ana").Set("name", "Ana María")

	if builder.values["name"] != "Ana María" {
		t.Errorf("Expected last value to win, got '%v'", builder.values["name"])
	}
}

func TestUpdateBuilder_MultipleFilters(t *testing.T) {
	schema := testSchema()
	builder := NewUpdateBuilder(schema, "User")

	builder.Filter("id", "eq", "uuid-123").
		Filter("email", "eq", "ana@mail.com").
		Set("name", "Ana")

	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}
}

func TestDeleteBuilder_MultipleFilters(t *testing.T) {
	schema := testSchema()
	builder := NewDeleteBuilder(schema, "User")

	builder.Filter("id", "eq", "uuid-123").
		Filter("email", "eq", "ana@mail.com")

	if len(builder.filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(builder.filters))
	}
}
