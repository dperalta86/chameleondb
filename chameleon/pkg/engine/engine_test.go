package engine

import (
	"testing"
)

func TestEngineLoadSchema(t *testing.T) {
	engine := NewEngine()

	schemaSource := `
		entity User {
			id: uuid primary,
			email: string unique,
			age: int nullable,
		}
		
		entity Order {
			id: uuid primary,
			total: decimal,
			user: User,
		}
	`

	schema, err := engine.LoadSchemaFromString(schemaSource)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Verify entities
	if len(schema.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(schema.Entities))
	}

	// Verify User entity
	user, ok := schema.Entities["User"]
	if !ok {
		t.Fatal("User entity not found")
	}

	if len(user.Fields) != 3 {
		t.Errorf("Expected 3 fields in User, got %d", len(user.Fields))
	}

	// Verify primary key
	idField := user.Fields["id"]
	if !idField.PrimaryKey {
		t.Error("Expected id to be primary key")
	}

	// Verify unique constraint
	emailField := user.Fields["email"]
	if !emailField.Unique {
		t.Error("Expected email to be unique")
	}
}

func TestEngineLoadSchemaFromFile(t *testing.T) {
	engine := NewEngine()

	// Load from example file
	schema, err := engine.LoadSchemaFromFile("../../../examples/basic_schema.cham")
	if err != nil {
		t.Fatalf("Failed to load schema from file: %v", err)
	}

	if schema == nil {
		t.Fatal("Schema is nil")
	}

	t.Logf("Loaded schema with %d entities", len(schema.Entities))
}

func TestEngineVersion(t *testing.T) {
	engine := NewEngine()
	version := engine.Version()

	if version == "" {
		t.Error("Version should not be empty")
	}

	t.Logf("ChameleonDB version: %s", version)
}

func TestInvalidSchema(t *testing.T) {
	engine := NewEngine()

	_, err := engine.LoadSchemaFromString("invalid syntax!!!")
	if err == nil {
		t.Error("Expected error for invalid syntax")
	}

	t.Logf("Got expected error: %v", err)
}
