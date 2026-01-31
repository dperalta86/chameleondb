package engine

import "encoding/json"

// Schema represents the complete database schema
type Schema struct {
	Entities map[string]*Entity `json:"entities"`
}

// Entity represents a database entity (table)
type Entity struct {
	Name      string               `json:"name"`
	Fields    map[string]*Field    `json:"fields"`
	Relations map[string]*Relation `json:"relations"`
}

// Field represents an entity field (column)
type Field struct {
	Name       string       `json:"name"`
	Type       FieldType    `json:"field_type"`
	Nullable   bool         `json:"nullable"`
	Unique     bool         `json:"unique"`
	PrimaryKey bool         `json:"primary_key"`
	Default    *interface{} `json:"default,omitempty"`
}

// FieldType represents the type of a field
type FieldType string

const (
	FieldTypeUUID      FieldType = "UUID"
	FieldTypeString    FieldType = "String"
	FieldTypeInt       FieldType = "Int"
	FieldTypeDecimal   FieldType = "Decimal"
	FieldTypeBool      FieldType = "Bool"
	FieldTypeTimestamp FieldType = "Timestamp"
)

// Relation represents a relationship between entities
type Relation struct {
	Name         string       `json:"name"`
	Kind         RelationKind `json:"kind"`
	TargetEntity string       `json:"target_entity"`
	ForeignKey   *string      `json:"foreign_key,omitempty"`
	Through      *string      `json:"through,omitempty"`
}

// RelationKind represents the type of relationship
type RelationKind string

const (
	RelationHasOne     RelationKind = "HasOne"
	RelationHasMany    RelationKind = "HasMany"
	RelationBelongsTo  RelationKind = "BelongsTo"
	RelationManyToMany RelationKind = "ManyToMany"
)

// ParseSchemaJSON parses a JSON string into a Schema
func ParseSchemaJSON(jsonStr string) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ToJSON converts a Schema to JSON string
func (s *Schema) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}