package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Executor runs queries against PostgreSQL
type Executor struct {
	connector *Connector
}

// NewExecutor creates an executor from a connector
func NewExecutor(connector *Connector) *Executor {
	return &Executor{connector: connector}
}

// Execute runs a QueryBuilder against the database
func (ex *Executor) Execute(ctx context.Context, qb *QueryBuilder) (*QueryResult, error) {
	if !ex.connector.IsConnected() {
		return nil, fmt.Errorf("not connected to database")
	}

	// Generate SQL
	generated, err := qb.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("SQL generation failed: %w", err)
	}

	// Execute main query
	mainRows, err := ex.executeQuery(ctx, generated.MainQuery)
	if err != nil {
		return nil, fmt.Errorf("main query failed: %w", err)
	}

	// Execute eager queries
	relations := make(map[string][]Row)
	parentIDs := extractIDs(mainRows, "id")

	for _, eager := range generated.EagerQueries {
		relName := eager[0]
		relSQL := eager[1]

		// Replace $PARENT_IDS placeholder with actual values
		sql, err := replacePlaceholder(relSQL, parentIDs)
		if err != nil {
			return nil, fmt.Errorf("eager query '%s' failed: %w", relName, err)
		}

		eagerRows, err := ex.executeQuery(ctx, sql)
		if err != nil {
			return nil, fmt.Errorf("eager query '%s' failed: %w", relName, err)
		}

		relations[relName] = eagerRows

		// Update parentIDs for next level (nested includes)
		// The FK field name is in the WHERE clause, extract the relevant IDs
		parentIDs = extractIDs(eagerRows, "id")
	}

	return &QueryResult{
		Entity:    qb.query.Entity,
		Rows:      mainRows,
		Relations: relations,
	}, nil
}

// executeQuery runs a single SQL query and returns rows
func (ex *Executor) executeQuery(ctx context.Context, sql string) ([]Row, error) {
	rows, err := ex.connector.Pool().Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// scanRows converts pgx rows into our Row type
func scanRows(rows pgx.Rows) ([]Row, error) {
	var result []Row
	columns := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(Row)
		for i, col := range columns {
			row[col.Name] = values[i]
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// extractIDs pulls a specific field from all rows and converts to string if needed
func extractIDs(rows []Row, field string) []interface{} {
	ids := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		if id, ok := row[field]; ok {
			// Convert UUID types to string
			switch v := id.(type) {
			case []byte:
				ids = append(ids, string(v))
			case [16]byte: // PostgreSQL UUID type
				// Convert to standard UUID string format
				ids = append(ids, uuidToString(v))
			case string:
				ids = append(ids, v)
			default:
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// uuidToString converts a [16]byte UUID to standard string format
func uuidToString(uuid [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16])
}

// replacePlaceholder replaces $PARENT_IDS with actual IN clause values
func replacePlaceholder(sql string, ids []interface{}) (string, error) {
	if len(ids) == 0 {
		return strings.Replace(sql, "$PARENT_IDS", "NULL", 1), nil
	}

	placeholders := make([]string, len(ids))
	for i, id := range ids {
		switch v := id.(type) {
		case string:
			// Escape single quotes and wrap in quotes
			escaped := strings.ReplaceAll(v, "'", "''")
			placeholders[i] = fmt.Sprintf("'%s'", escaped)
		case int, int32, int64, uint, uint32, uint64:
			placeholders[i] = fmt.Sprintf("%d", v)
		case float32, float64:
			placeholders[i] = fmt.Sprintf("%f", v)
		default:
			// Fallback: convert to string and quote
			placeholders[i] = fmt.Sprintf("'%v'", v)
		}
	}

	return strings.Replace(
		sql,
		"$PARENT_IDS",
		strings.Join(placeholders, ", "),
		1,
	), nil
}
