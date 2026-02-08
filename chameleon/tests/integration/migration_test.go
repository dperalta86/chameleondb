package integration

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestMigrationCreatesAllTables(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	// Run migration
	runMigration(t, eng, ctx)

	// Verify tables exist
	config := testConfig()
	conn, err := pgx.Connect(ctx, config.ConnectionString())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	tables := []string{"users", "orders", "order_items"}
	for _, table := range tables {
		var exists bool
		err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s was not created", table)
		}
	}
}

func TestMigrationCreatesPrimaryKeys(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	config := testConfig()
	conn, err := pgx.Connect(ctx, config.ConnectionString())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	// Check users.id is primary key
	var constraint string
	err = conn.QueryRow(ctx, `
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE table_name = 'users' 
		AND constraint_type = 'PRIMARY KEY'
	`).Scan(&constraint)

	if err != nil {
		t.Fatalf("Failed to check primary key: %v", err)
	}
	if constraint == "" {
		t.Error("users table has no primary key")
	}
}

func TestMigrationCreatesForeignKeys(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	config := testConfig()
	conn, err := pgx.Connect(ctx, config.ConnectionString())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	// Check orders.user_id has FK to users
	var constraint string
	err = conn.QueryRow(ctx, `
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE table_name = 'orders' 
		AND constraint_type = 'FOREIGN KEY'
	`).Scan(&constraint)

	if err != nil {
		t.Fatalf("Failed to check foreign key: %v", err)
	}
	if constraint == "" {
		t.Error("orders table has no foreign key")
	}
}

func TestMigrationCreatesUniqueConstraints(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	config := testConfig()
	conn, err := pgx.Connect(ctx, config.ConnectionString())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	// Check users.email is unique
	var constraint string
	err = conn.QueryRow(ctx, `
		SELECT constraint_name 
		FROM information_schema.table_constraints 
		WHERE table_name = 'users' 
		AND constraint_type = 'UNIQUE'
	`).Scan(&constraint)

	if err != nil {
		t.Fatalf("Failed to check unique constraint: %v", err)
	}
	if constraint == "" {
		t.Error("users.email has no unique constraint")
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	runMigration(t, eng, ctx) // no debe fallar
}

func TestOrdersUserForeignKeyIsCorrect(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	conn, _ := pgx.Connect(ctx, testConfig().ConnectionString())
	defer conn.Close(ctx)

	var count int
	err := conn.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_name = tc.constraint_name
		WHERE tc.table_name = 'orders'
		  AND tc.constraint_type = 'FOREIGN KEY'
		  AND kcu.column_name = 'user_id'
		  AND ccu.table_name = 'users'
		  AND ccu.column_name = 'id'
	`).Scan(&count)

	if err != nil {
		t.Fatalf("failed to inspect foreign key: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected exactly 1 FK orders.user_id → users.id, got %d", count)
	}
}

func TestForeignKeyIsEnforced(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	conn, _ := pgx.Connect(ctx, testConfig().ConnectionString())
	defer conn.Close(ctx)

	_, err := conn.Exec(ctx, `
		INSERT INTO orders (id, total, user_id)
		VALUES ('00000000-0000-0000-0000-000000000000', 10, '99999999-9999-9999-9999-999999999999')
	`)

	if err == nil {
		t.Fatal("expected FK violation, got nil error")
	}
}
