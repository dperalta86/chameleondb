package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/jackc/pgx/v5"
)

// testConfig returns config for the test database
func testConfig() engine.ConnectorConfig {
	return engine.ConnectorConfig{
		Host:     getEnv("CHAMELEON_TEST_DB_HOST", "localhost"),
		Port:     getEnvInt("CHAMELEON_TEST_DB_PORT", 5433),
		Database: getEnv("CHAMELEON_TEST_DB_NAME", "chameleon_test"),
		User:     getEnv("CHAMELEON_TEST_DB_USER", "postgres"),
		Password: getEnv("CHAMELEON_TEST_DB_PASS", "postgres"),
		MaxConns: 5,
		MinConns: 1,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		fmt.Sscanf(value, "%d", &result)
		return result
	}
	return fallback
}

// setupTestDB creates a fresh engine with loaded schema and DB connection
func setupTestDB(t *testing.T) (*engine.Engine, context.Context, func()) {
	t.Helper()

	ctx := context.Background()

	// Load schema
	eng := engine.NewEngine()
	_, err := eng.LoadSchemaFromFile("../fixtures/test_schema.cham")
	if err != nil {
		t.Fatalf("Failed to load test schema: %v", err)
	}

	// Connect to DB
	config := testConfig()
	err = eng.Connect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Wait for DB to be ready
	for i := 0; i < 10; i++ {
		if err := eng.Ping(ctx); err == nil {
			break
		}
		if i == 9 {
			t.Fatal("Database not ready after 10 retries")
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Cleanup function
	cleanup := func() {
		// Drop all tables
		dropAllTables(t, ctx, config)
		eng.Close()
	}

	return eng, ctx, cleanup
}

// dropAllTables removes all tables from the test database
func dropAllTables(t *testing.T, ctx context.Context, config engine.ConnectorConfig) {
	t.Helper()

	connStr := config.ConnectionString()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Logf("Warning: failed to connect for cleanup: %v", err)
		return
	}
	defer conn.Close(ctx)

	// Drop tables in reverse dependency order
	tables := []string{"order_items", "orders", "users"}
	for _, table := range tables {
		_, err := conn.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to drop table %s: %v", table, err)
		}
	}
}

// runMigration executes the migration SQL
func runMigration(t *testing.T, eng *engine.Engine, ctx context.Context) {
	t.Helper()

	sql, err := eng.GenerateMigration()
	if err != nil {
		t.Fatalf("failed to generate migration: %v", err)
	}

	conn, err := pgx.Connect(ctx, testConfig().ConnectionString())
	if err != nil {
		t.Fatalf("failed to connect for migration: %v", err)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	if _, err := tx.Exec(ctx, sql); err != nil {
		tx.Rollback(ctx)
		t.Fatalf("migration failed:\n%v\nSQL:\n%s", err, sql)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("failed to commit migration: %v", err)
	}
}

// insertTestData inserts sample data for testing
func insertTestData(t *testing.T, ctx context.Context, config engine.ConnectorConfig) {
	t.Helper()

	connStr := config.ConnectionString()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("Failed to connect for test data: %v", err)
	}
	defer conn.Close(ctx)

	// Insert users
	_, err = conn.Exec(ctx, `
		INSERT INTO users (id, email, name, age, created_at) VALUES
		('11111111-1111-1111-1111-111111111111', 'ana@mail.com', 'Ana Garcia', 25, NOW()),
		('22222222-2222-2222-2222-222222222222', 'bob@mail.com', 'Bob Smith', 30, NOW()),
		('33333333-3333-3333-3333-333333333333', 'charlie@mail.com', 'Charlie Brown', NULL, NOW())
	`)
	if err != nil {
		t.Fatalf("Failed to insert users: %v", err)
	}

	// Insert orders
	_, err = conn.Exec(ctx, `
		INSERT INTO orders (id, total, status, user_id, created_at) VALUES
		('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 150.50, 'completed', '11111111-1111-1111-1111-111111111111', NOW()),
		('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 75.00, 'pending', '11111111-1111-1111-1111-111111111111', NOW()),
		('cccccccc-cccc-cccc-cccc-cccccccccccc', 200.00, 'completed', '22222222-2222-2222-2222-222222222222', NOW())
	`)
	if err != nil {
		t.Fatalf("Failed to insert orders: %v", err)
	}

	// Insert order items
	_, err = conn.Exec(ctx, `
		INSERT INTO order_items (id, quantity, price, order_id) VALUES
		('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 2, 50.25, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
		('ffffffff-ffff-ffff-ffff-ffffffffffff', 1, 50.00, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
		('dddddddd-dddd-dddd-dddd-dddddddddddd', 3, 25.00, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb')
	`)
	if err != nil {
		t.Fatalf("Failed to insert order items: %v", err)
	}
}

// skipIfNoDocker skips the test if Docker is not available
func skipIfNoDocker(t *testing.T) {
	t.Helper()

	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test (SKIP_INTEGRATION set)")
	}

	// Try to connect to test DB
	ctx := context.Background()
	config := testConfig()
	connStr := config.ConnectionString()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Skipf("Docker not available or test DB not running: %v", err)
	}
	conn.Close(ctx)
}

func cleanupDatabase(t *testing.T, ctx context.Context, config engine.ConnectorConfig) {
	t.Helper()

	conn, err := pgx.Connect(ctx, config.ConnectionString())
	if err != nil {
		t.Fatalf("cleanup failed to connect: %v", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
	`)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}
