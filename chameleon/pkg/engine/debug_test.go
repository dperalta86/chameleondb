package engine

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupTestSchema creates a minimal in-memory schema for testing
func setupTestSchema() *Schema {
	return &Schema{
		Entities: []*Entity{
			{
				Name: "User",
				Fields: map[string]*Field{
					"id": {
						Name:       "id",
						Type:       FieldTypeUUID,
						PrimaryKey: true,
					},
					"email": {
						Name:   "email",
						Type:   FieldTypeString,
						Unique: true,
					},
					"name": {
						Name: "name",
						Type: FieldTypeString,
					},
				},
				Relations: map[string]*Relation{},
			},
		},
	}
}

// TestDebugContext verifies debug context creation
func TestDebugContextCreation(t *testing.T) {
	dc := DefaultDebugContext()

	assert.Equal(t, DebugNone, dc.Level)
	assert.NotNil(t, dc.Writer)
	assert.True(t, dc.ColorOutput)
}

// TestDebugLevel verifies debug level comparison
func TestDebugLevelComparison(t *testing.T) {
	tests := []struct {
		name     string
		level    DebugLevel
		checkLvl DebugLevel
		expected bool
	}{
		{"None < SQL", DebugNone, DebugSQL, true},
		{"SQL < Trace", DebugSQL, DebugTrace, true},
		{"Trace < Explain", DebugTrace, DebugExplain, true},
		{"SQL >= SQL", DebugSQL, DebugSQL, false},
		{"Explain >= SQL", DebugExplain, DebugSQL, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level < tt.checkLvl
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDebugLog verifies SQL logging with different levels
func TestDebugLogSQL(t *testing.T) {
	tests := []struct {
		name     string
		level    DebugLevel
		sqlText  string
		expected string
	}{
		{
			name:     "DebugSQL outputs SQL",
			level:    DebugSQL,
			sqlText:  "SELECT * FROM users;",
			expected: "SELECT * FROM users;",
		},
		{
			name:     "DebugNone outputs nothing",
			level:    DebugNone,
			sqlText:  "SELECT * FROM users;",
			expected: "",
		},
		{
			name:     "DebugTrace outputs SQL",
			level:    DebugTrace,
			sqlText:  "SELECT id FROM posts;",
			expected: "SELECT id FROM posts;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			dc := &DebugContext{
				Level:       tt.level,
				Writer:      &buf,
				ColorOutput: false,
			}

			dc.LogSQL(tt.sqlText)
			output := buf.String()

			if tt.expected == "" {
				assert.Empty(t, output, "expected no output for level %d", tt.level)
			} else {
				assert.Contains(t, output, tt.expected)
				assert.Contains(t, output, "[SQL]")
			}
		})
	}
}

// TestDebugLogQuery verifies query trace logging
func TestDebugLogQuery(t *testing.T) {
	var buf bytes.Buffer

	dc := &DebugContext{
		Level:       DebugTrace,
		Writer:      &buf,
		ColorOutput: false,
	}

	dc.LogQuery("SELECT * FROM users;", 5, 10)
	output := buf.String()

	assert.Contains(t, output, "Query Trace")
	assert.Contains(t, output, "SELECT * FROM users;")
	assert.Contains(t, output, "Rows: 10")
}

// TestDebugLogFilter verifies logging respects debug level
func TestDebugLogFilter(t *testing.T) {
	tests := []struct {
		name       string
		debugLevel DebugLevel
		logLevel   DebugLevel
		shouldLog  bool
	}{
		{"DebugSQL >= DebugSQL", DebugSQL, DebugSQL, true},
		{"DebugTrace >= DebugSQL", DebugTrace, DebugSQL, true},
		{"DebugSQL < DebugTrace", DebugSQL, DebugTrace, false},
		{"DebugNone < DebugSQL", DebugNone, DebugSQL, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			dc := &DebugContext{
				Level:       tt.debugLevel,
				Writer:      &buf,
				ColorOutput: false,
			}

			dc.Log(tt.logLevel, "test message")
			output := buf.String()

			if tt.shouldLog {
				assert.Contains(t, output, "test message")
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

// TestDebugPrefixes verifies color and text prefixes
func TestDebugPrefixes(t *testing.T) {
	tests := []struct {
		name        string
		level       DebugLevel
		colorPrefix string
		textPrefix  string
	}{
		{
			name:        "DebugSQL",
			level:       DebugSQL,
			colorPrefix: "\033[36m[DEBUG]\033[0m ",
			textPrefix:  "[DEBUG] ",
		},
		{
			name:        "DebugTrace",
			level:       DebugTrace,
			colorPrefix: "\033[33m[TRACE]\033[0m ",
			textPrefix:  "[TRACE] ",
		},
		{
			name:        "DebugExplain",
			level:       DebugExplain,
			colorPrefix: "\033[35m[EXPLAIN]\033[0m ",
			textPrefix:  "[EXPLAIN] ",
		},
		{
			name:        "DebugNone",
			level:       DebugNone,
			colorPrefix: "",
			textPrefix:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.colorPrefix, colorPrefix(tt.level))
			assert.Equal(t, tt.textPrefix, textPrefix(tt.level))
		})
	}
}

// TestEngineWithDebug verifies engine debug context
func TestEngineWithDebug(t *testing.T) {
	eng := NewEngine()
	assert.Equal(t, DebugNone, eng.Debug.Level)

	eng2 := eng.WithDebug(DebugSQL)
	assert.Equal(t, DebugSQL, eng2.Debug.Level)

	eng3 := eng.WithDebug(DebugTrace)
	assert.Equal(t, DebugTrace, eng3.Debug.Level)
}

// TestQueryBuilderDebugMethods verifies query-level debug flags
func TestQueryBuilderDebugMethods(t *testing.T) {
	eng := NewEngine()
	eng.schema = setupTestSchema()

	qb := eng.Query("User")
	assert.Nil(t, qb.debugLevel)

	qb.Debug()
	assert.NotNil(t, qb.debugLevel)
	assert.Equal(t, DebugSQL, *qb.debugLevel)

	qb2 := eng.Query("User")
	qb2.DebugTrace()
	assert.NotNil(t, qb2.debugLevel)
	assert.Equal(t, DebugTrace, *qb2.debugLevel)
}

// TestGetDebugContext verifies query debug context resolution
func TestGetDebugContext(t *testing.T) {
	tests := []struct {
		name          string
		engineLevel   DebugLevel
		queryLevel    *DebugLevel
		expectedLevel DebugLevel
	}{
		{
			name:          "use engine debug if query not set",
			engineLevel:   DebugSQL,
			queryLevel:    nil,
			expectedLevel: DebugSQL,
		},
		{
			name:          "use query debug if set",
			engineLevel:   DebugNone,
			queryLevel:    func() *DebugLevel { l := DebugTrace; return &l }(),
			expectedLevel: DebugTrace,
		},
		{
			name:          "query debug overrides engine",
			engineLevel:   DebugSQL,
			queryLevel:    func() *DebugLevel { l := DebugExplain; return &l }(),
			expectedLevel: DebugExplain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine()
			eng.Debug.Level = tt.engineLevel

			qb := eng.Query("User")
			qb.debugLevel = tt.queryLevel

			dc := qb.getDebugContext()
			assert.Equal(t, tt.expectedLevel, dc.Level)
		})
	}
}
