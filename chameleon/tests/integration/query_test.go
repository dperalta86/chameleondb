package integration

import (
	"testing"
)

// This test do not can done because the current implementation of the query does not support null values in filters.
// The test is left here as a reminder to implement this feature in v1+
/* func TestQueryFilterIsNull(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	result, err := eng.Query("User").
		Filter("age", "is_null", nil).
		Execute(ctx)

	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if result.Count() != 1 {
		t.Fatalf("expected 1 user with age NULL, got %d", result.Count())
	}

	if result.Rows[0].String("email") != "charlie@mail.com" {
		t.Fatalf("unexpected user returned")
	}
} */

func TestQueryInvalidFieldFails(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	_, err := eng.Query("User").
		Filter("does_not_exist", "eq", 1).
		Execute(ctx)

	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestQueryInvalidOperatorFails(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	_, err := eng.Query("User").
		Filter("age", "approx", 30).
		Execute(ctx)

	if err == nil {
		t.Fatal("expected error for invalid operator")
	}
}

func TestQueryInvalidIncludeFails(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)

	_, err := eng.Query("User").
		Include("payments").
		Execute(ctx)

	if err == nil {
		t.Fatal("expected error for invalid include")
	}
}

func TestQueryFetchAll(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query all users
	result, err := eng.Query("User").Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() != 3 {
		t.Errorf("Expected 3 users, got %d", result.Count())
	}
}

func TestQueryFilterEquality(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query user by email
	result, err := eng.Query("User").
		Filter("email", "eq", "ana@mail.com").
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() != 1 {
		t.Errorf("Expected 1 user, got %d", result.Count())
	}

	if result.Count() > 0 {
		email := result.Rows[0].String("email")
		if email != "ana@mail.com" {
			t.Errorf("Expected email 'ana@mail.com', got '%s'", email)
		}
	}
}

func TestQueryFilterComparison(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query users age >= 30
	result, err := eng.Query("User").
		Filter("age", "gte", 30).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() != 1 {
		t.Errorf("Expected 1 user with age >= 30, got %d", result.Count())
	}

	if result.Count() > 0 {
		age := result.Rows[0].Int("age")
		if age < 30 {
			t.Errorf("Expected age >= 30, got %d", age)
		}
	}
}

func TestQueryMultipleFilters(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query users age >= 20 AND age <= 28
	result, err := eng.Query("User").
		Filter("age", "gte", 20).
		Filter("age", "lte", 28).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() != 1 {
		t.Errorf("Expected 1 user in age range, got %d", result.Count())
	}
}

func TestQueryOrderBy(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query users ordered by name
	result, err := eng.Query("User").
		OrderBy("name", "asc").
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() < 2 {
		t.Fatal("Not enough users to test ordering")
	}

	first := result.Rows[0].String("name")
	second := result.Rows[1].String("name")

	if first > second {
		t.Errorf("Results not ordered correctly: %s should come before %s", first, second)
	}
}

func TestQueryLimitOffset(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query with limit
	result, err := eng.Query("User").
		OrderBy("name", "asc").
		Limit(2).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() != 2 {
		t.Errorf("Expected 2 users (limit), got %d", result.Count())
	}

	// Query with offset
	result2, err := eng.Query("User").
		OrderBy("name", "asc").
		Offset(1).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query with offset failed: %v", err)
	}

	if result2.Count() != 2 {
		t.Errorf("Expected 2 users (offset 1 from 3), got %d", result2.Count())
	}
}

func TestQueryIncludeEagerLoading(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query user with orders included
	result, err := eng.Query("User").
		Filter("email", "eq", "ana@mail.com").
		Include("orders").
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() != 1 {
		t.Fatalf("Expected 1 user, got %d", result.Count())
	}

	// Check eager-loaded orders
	orders, ok := result.Relations["orders"]
	if !ok {
		t.Fatal("orders relation not loaded")
	}

	if len(orders) != 2 {
		t.Errorf("Expected 2 orders for ana@mail.com, got %d", len(orders))
	}
}

func TestQueryNestedInclude(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query user with orders and items
	result, err := eng.Query("User").
		Filter("email", "eq", "ana@mail.com").
		Include("orders").
		Include("orders.items").
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Check orders loaded
	orders, ok := result.Relations["orders"]
	if !ok {
		t.Fatal("orders relation not loaded")
	}

	if len(orders) < 1 {
		t.Fatal("Expected at least 1 order")
	}

	// Check items loaded
	items, ok := result.Relations["items"]
	if !ok {
		t.Fatal("items relation not loaded")
	}

	if len(items) < 1 {
		t.Error("Expected at least 1 item for ana's orders")
	}
}

func TestQueryFilterOnRelation(t *testing.T) {
	skipIfNoDocker(t)

	eng, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	runMigration(t, eng, ctx)
	insertTestData(t, ctx, testConfig())

	// Query users who have orders with total > 100
	result, err := eng.Query("User").
		Filter("orders.total", "gt", 100).
		Execute(ctx)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count() < 1 {
		t.Error("Expected at least 1 user with orders > 100")
	}

	// Ana has order of 150.50, Bob has 200
	// Charlie has no orders, or orders <= 100
	for _, row := range result.Rows {
		email := row.String("email")
		if email != "ana@mail.com" && email != "bob@mail.com" {
			t.Errorf("Unexpected user in results: %s", email)
		}
	}
}
