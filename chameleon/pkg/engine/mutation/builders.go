package mutation

import (
	"context"
	"fmt"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
)

// ============================================================
// INSERT BUILDER
// ============================================================
//

type InsertBuilder struct {
	schema *engine.Schema
	entity string
	values map[string]interface{}
	config engine.ValidatorConfig

	debug  bool
	dryRun bool
	sql    string
}

func NewInsertBuilder(schema *engine.Schema, entity string) *InsertBuilder {
	return &InsertBuilder{
		schema: schema,
		entity: entity,
		values: make(map[string]interface{}),
		config: engine.DefaultValidatorConfig(),
	}
}

func (ib *InsertBuilder) Set(field string, value interface{}) *InsertBuilder {
	ib.values[field] = value
	return ib
}

func (ib *InsertBuilder) Debug() *InsertBuilder {
	ib.debug = true
	return ib
}

func (ib *InsertBuilder) DryRun() *InsertBuilder {
	ib.dryRun = true
	return ib
}

// Build implements engine.MutationBuilder
func (ib *InsertBuilder) Build() (*engine.Mutation, error) {
	validator := engine.NewValidator(ib.schema, ib.config)
	if err := validator.ValidateInsertInput(ib.entity, ib.values); err != nil {
		return nil, err
	}

	ib.sql = ib.generateInsertSQL()

	return &engine.Mutation{
		Type:      engine.MutationInsert,
		Entity:    ib.entity,
		HasFilter: false,
	}, nil
}

// Exec implements engine.MutationBuilder
func (ib *InsertBuilder) Exec() error {
	if ib.sql == "" {
		if _, err := ib.Build(); err != nil {
			return err
		}
	}

	if ib.debug {
		fmt.Printf("\n[SQL]\n%s\n\n", ib.sql)
	}

	if ib.dryRun {
		return nil
	}

	// TODO: ejecutar vía executor / FFI
	return nil
}

// Execute mantiene compatibilidad con tu API actual
func (ib *InsertBuilder) Execute(ctx context.Context) (*InsertResult, error) {
	if err := ib.Exec(); err != nil {
		return nil, err
	}

	return &InsertResult{
		SQL:      ib.sql,
		Affected: 1,
		DryRun:   ib.dryRun,
	}, nil
}

func (ib *InsertBuilder) generateInsertSQL() string {
	var fields []string
	var placeholders []string
	i := 1

	for field := range ib.values {
		fields = append(fields, fmt.Sprintf(`"%s"`, field))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	return fmt.Sprintf(
		`INSERT INTO "%s" (%s) VALUES (%s) RETURNING *`,
		ib.entity,
		formatList(fields),
		formatList(placeholders),
	)
}

type InsertResult struct {
	SQL      string
	Affected int
	DryRun   bool
}

//
// ============================================================
// UPDATE BUILDER
// ============================================================
//

type UpdateBuilder struct {
	schema   *engine.Schema
	entity   string
	filters  map[string]interface{}
	updates  map[string]interface{}
	config   engine.ValidatorConfig
	debug    bool
	dryRun   bool
	sql      string
	forceAll bool
}

func NewUpdateBuilder(schema *engine.Schema, entity string) *UpdateBuilder {
	return &UpdateBuilder{
		schema:  schema,
		entity:  entity,
		filters: make(map[string]interface{}),
		updates: make(map[string]interface{}),
		config:  engine.DefaultValidatorConfig(),
	}
}

func (ub *UpdateBuilder) Filter(field string, op string, value interface{}) *UpdateBuilder {
	key := fmt.Sprintf("%s:%s", field, op)
	ub.filters[key] = value
	return ub
}

func (ub *UpdateBuilder) Set(field string, value interface{}) *UpdateBuilder {
	ub.updates[field] = value
	return ub
}

func (ub *UpdateBuilder) ForceUpdateAll() *UpdateBuilder {
	ub.forceAll = true
	return ub
}

func (ub *UpdateBuilder) Debug() *UpdateBuilder {
	ub.debug = true
	return ub
}

func (ub *UpdateBuilder) DryRun() *UpdateBuilder {
	ub.dryRun = true
	return ub
}

func (ub *UpdateBuilder) Build() (*engine.Mutation, error) {
	validator := engine.NewValidator(ub.schema, ub.config)
	if err := validator.ValidateUpdateInput(
		ub.entity,
		ub.parseFilters(),
		ub.updates,
	); err != nil {
		return nil, err
	}

	ub.sql = ub.generateUpdateSQL()

	return &engine.Mutation{
		Type:      engine.MutationUpdate,
		Entity:    ub.entity,
		HasFilter: len(ub.filters) > 0,
	}, nil
}

// Exec executes the validated mutation
func (ub *UpdateBuilder) Exec() error {
	if ub.sql == "" {
		if _, err := ub.Build(); err != nil {
			return err
		}
	}

	if ub.debug {
		fmt.Printf("\n[SQL]\n%s\n\n", ub.sql)
	}

	if ub.dryRun {
		return nil
	}

	// TODO: ejecutar vía executor / FFI
	return nil
}

func (ub *UpdateBuilder) generateUpdateSQL() string {
	var setClauses []string
	i := 1

	for field := range ub.updates {
		setClauses = append(setClauses, fmt.Sprintf(`"%s"=$%d`, field, i))
		i++
	}

	var filterClauses []string
	for field := range ub.filters {
		fieldName := strings.Split(field, ":")[0]
		filterClauses = append(filterClauses, fmt.Sprintf(`"%s"=$%d`, fieldName, i))
		i++
	}

	table := strings.ToLower(ub.entity)

	return fmt.Sprintf(
		`UPDATE "%s" SET %s WHERE %s RETURNING *`,
		table,
		strings.Join(setClauses, ", "),
		strings.Join(filterClauses, " AND "),
	)
}

//
// ============================================================
// DELETE BUILDER
// ============================================================
//

type DeleteBuilder struct {
	schema         *engine.Schema
	entity         string
	filters        map[string]interface{}
	config         engine.ValidatorConfig
	debug          bool
	dryRun         bool
	sql            string
	forceDeleteAll bool
}

func NewDeleteBuilder(schema *engine.Schema, entity string) *DeleteBuilder {
	return &DeleteBuilder{
		schema:  schema,
		entity:  entity,
		filters: make(map[string]interface{}),
		config:  engine.DefaultValidatorConfig(),
	}
}

func (db *DeleteBuilder) Filter(field string, op string, value interface{}) *DeleteBuilder {
	key := fmt.Sprintf("%s:%s", field, op)
	db.filters[key] = value
	return db
}

func (db *DeleteBuilder) ForceDeleteAll() *DeleteBuilder {
	db.forceDeleteAll = true
	return db
}

func (db *DeleteBuilder) Debug() *DeleteBuilder {
	db.debug = true
	return db
}

func (db *DeleteBuilder) DryRun() *DeleteBuilder {
	db.dryRun = true
	return db
}

func (db *DeleteBuilder) Build() (*engine.Mutation, error) {
	validator := engine.NewValidator(db.schema, db.config)
	if err := validator.ValidateDeleteInput(
		db.entity,
		db.parseFilters(),
		db.forceDeleteAll,
	); err != nil {
		return nil, err
	}

	db.sql = db.generateDeleteSQL()

	return &engine.Mutation{
		Type:      engine.MutationDelete,
		Entity:    db.entity,
		HasFilter: len(db.filters) > 0,
	}, nil
}

// Exec executes the validated mutation
func (db *DeleteBuilder) Exec() error {
	if db.sql == "" {
		if _, err := db.Build(); err != nil {
			return err
		}
	}

	if db.debug {
		fmt.Printf("\n[SQL]\n%s\n\n", db.sql)
	}

	if db.dryRun {
		return nil
	}

	// TODO: ejecutar vía executor / FFI
	return nil
}

func (db *DeleteBuilder) generateDeleteSQL() string {
	var filterClauses []string
	i := 1

	for field := range db.filters {
		fieldName := strings.Split(field, ":")[0]
		filterClauses = append(filterClauses, fmt.Sprintf(`"%s"=$%d`, fieldName, i))
		i++
	}

	table := strings.ToLower(db.entity)

	return fmt.Sprintf(
		`DELETE FROM "%s" WHERE %s`,
		table,
		strings.Join(filterClauses, " AND "),
	)
}

//
// ============================================================
// UTILS
// ============================================================
//

func formatList(items []string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += item
	}
	return result
}

func (db *DeleteBuilder) parseFilters() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range db.filters {
		parts := formatStringSplit(key, ":")
		if len(parts) > 0 {
			result[parts[0]] = value
		}
	}
	return result
}

func (ub *UpdateBuilder) parseFilters() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range ub.filters {
		parts := formatStringSplit(key, ":")
		if len(parts) > 0 {
			result[parts[0]] = value
		}
	}
	return result
}

func formatStringSplit(s, sep string) []string {
	var result []string
	current := ""
	for _, char := range s {
		if string(char) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
