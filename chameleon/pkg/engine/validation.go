package engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// ============================================================
// VALIDATOR CONFIG
// ============================================================

type ValidatorConfig struct {
	StrictTypes bool
	ValidateFK  bool
}

func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		StrictTypes: true,
		ValidateFK:  true,
	}
}

// ============================================================
// VALIDATOR
// ============================================================

type Validator struct {
	schema *Schema
	config ValidatorConfig
}

func NewValidator(schema *Schema, config ValidatorConfig) *Validator {
	return &Validator{
		schema: schema,
		config: config,
	}
}

// ============================================================
// INSERT VALIDATION
// ============================================================

func (v *Validator) ValidateInsertInput(
	entity string,
	fields map[string]interface{},
) error {
	ent := v.schema.GetEntity(entity)
	if ent == nil {
		return &UnknownEntityError{
			Entity:    entity,
			Available: v.getAvailableEntities(),
		}
	}

	for fieldName, value := range fields {
		if err := v.validateInsertField(ent, fieldName, value); err != nil {
			return err
		}
	}

	return v.validateRequiredFields(ent, fields)
}

func (v *Validator) validateInsertField(
	ent *Entity,
	fieldName string,
	value interface{},
) error {
	field, ok := ent.Fields[fieldName]
	if !ok {
		return &UnknownFieldError{
			Entity:    ent.Name,
			Field:     fieldName,
			Available: v.getAvailableFields(ent),
		}
	}

	if err := v.validateFieldType(field, fieldName, value); err != nil {
		return err
	}

	if err := v.validateFieldFormat(fieldName, value); err != nil {
		return err
	}

	if value == nil && !field.Nullable {
		return &NotNullError{
			Field:      fieldName,
			Suggestion: "Provide a value for this field",
		}
	}

	return nil
}

// ============================================================
// UPDATE VALIDATION
// ============================================================

func (v *Validator) ValidateUpdateInput(
	entity string,
	filters map[string]interface{},
	updates map[string]interface{},
) error {
	ent := v.schema.GetEntity(entity)
	if ent == nil {
		return &UnknownEntityError{
			Entity:    entity,
			Available: v.getAvailableEntities(),
		}
	}

	if len(filters) == 0 {
		return &SafetyError{
			Operation:  "update_without_filter",
			Message:    "UPDATE requires a WHERE clause",
			Suggestion: "Use Filter() or ForceUpdateAll()",
		}
	}

	for fieldName := range filters {
		if _, ok := ent.Fields[fieldName]; !ok {
			return &UnknownFieldError{
				Entity:    ent.Name,
				Field:     fieldName,
				Available: v.getAvailableFields(ent),
			}
		}
	}

	for fieldName, value := range updates {
		field, ok := ent.Fields[fieldName]
		if !ok {
			return &UnknownFieldError{
				Entity:    ent.Name,
				Field:     fieldName,
				Available: v.getAvailableFields(ent),
			}
		}

		if field.PrimaryKey {
			return &ConstraintError{
				Type:       "primary_key",
				Field:      fieldName,
				Suggestion: "Primary keys cannot be updated",
			}
		}

		if err := v.validateFieldType(field, fieldName, value); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// DELETE VALIDATION
// ============================================================

func (v *Validator) ValidateDeleteInput(
	entity string,
	filters map[string]interface{},
	forceDeleteAll bool,
) error {
	ent := v.schema.GetEntity(entity)
	if ent == nil {
		return &UnknownEntityError{
			Entity:    entity,
			Available: v.getAvailableEntities(),
		}
	}

	if len(filters) == 0 && !forceDeleteAll {
		return &SafetyError{
			Operation:  "delete_without_filter",
			Message:    "DELETE without WHERE is blocked",
			Suggestion: "Use Filter() or ForceDeleteAll()",
		}
	}

	return nil
}

// ============================================================
// FIELD TYPE VALIDATION
// ============================================================

func (v *Validator) validateFieldType(
	field *Field,
	fieldName string,
	value interface{},
) error {
	if value == nil {
		if field.Nullable {
			return nil
		}
		return &NotNullError{
			Field:      fieldName,
			Suggestion: "This field cannot be null",
		}
	}

	switch field.Type.Kind {
	case "UUID":
		str, ok := value.(string)
		if !ok || !isValidUUID(str) {
			return &FieldFormatError{
				Field:      fieldName,
				Format:     "UUID",
				Value:      fmt.Sprintf("%v", value),
				Suggestion: "Use uuid.New().String()",
			}
		}

	case "String":
		if _, ok := value.(string); !ok {
			return &TypeMismatchError{
				Field:        fieldName,
				ExpectedType: "string",
				ReceivedType: fmt.Sprintf("%T", value),
				Value:        value,
				Suggestion:   "Pass a string",
			}
		}
	}

	return nil
}

// ============================================================
// FORMAT VALIDATION
// ============================================================

func (v *Validator) validateFieldFormat(
	fieldName string,
	value interface{},
) error {
	str, ok := value.(string)
	if !ok || str == "" {
		return nil
	}

	if strings.Contains(strings.ToLower(fieldName), "email") {
		if !isValidEmail(str) {
			return &FieldFormatError{
				Field:      fieldName,
				Format:     "email",
				Value:      str,
				Suggestion: "Use a valid email address",
			}
		}
	}

	return nil
}

// ============================================================
// REQUIRED FIELDS
// ============================================================

func (v *Validator) validateRequiredFields(
	ent *Entity,
	provided map[string]interface{},
) error {
	for _, field := range ent.Fields {
		if field.Nullable || field.Default != nil || field.PrimaryKey {
			continue
		}

		if _, ok := provided[field.Name]; !ok {
			return &NotNullError{
				Field:      field.Name,
				Suggestion: "This field is required",
			}
		}
	}
	return nil
}

// ============================================================
// HELPERS
// ============================================================

func (v *Validator) getAvailableFields(ent *Entity) []string {
	var fields []string
	for name := range ent.Fields {
		fields = append(fields, name)
	}
	return fields
}

func (v *Validator) getAvailableEntities() []string {
	var entities []string
	for _, ent := range v.schema.Entities {
		entities = append(entities, ent.Name)
	}
	return entities
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func isValidEmail(s string) bool {
	re := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	return re.MatchString(s)
}
