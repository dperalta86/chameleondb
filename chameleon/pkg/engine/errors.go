package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chameleon-db/chameleondb/chameleon/internal/ffi"
	"github.com/fatih/color"
)

// ParseErrorDetail matches Rust's ParseErrorDetail
type ParseErrorDetail struct {
	Message    string  `json:"message"`
	Line       int     `json:"line"`
	Column     int     `json:"column"`
	Snippet    *string `json:"snippet"`
	Suggestion *string `json:"suggestion"`
	Token      *string `json:"token"`
}

type ChameleonError struct {
	Kind string           `json:"kind"`
	Data ParseErrorDetail `json:"data"`
}

// FormatError tries to parse and format a detailed error
func FormatError(errMsg string) string {
	var chErr ChameleonError
	if err := json.Unmarshal([]byte(errMsg), &chErr); err != nil {
		// Not JSON structure -> fallback
		return errMsg
	}

	switch chErr.Kind {
	case "ParseError":
		return formatParseError(chErr.Data)

	default:
		// Others -> fallback
		return errMsg
	}
}

func formatParseError(detail ParseErrorDetail) string {
	var b strings.Builder

	// Error header
	errorColor := color.New(color.FgRed, color.Bold)
	errorColor.Fprintf(&b, "Error: ")
	fmt.Fprintf(&b, "%s\n\n", detail.Message)

	// Location
	locationColor := color.New(color.FgCyan)
	locationColor.Fprintf(&b, "  --> ")
	fmt.Fprintf(&b, "schema.cham:%d:%d\n", detail.Line, detail.Column)

	// Snippet if available
	if detail.Snippet != nil && *detail.Snippet != "" {
		b.WriteString("\n")
		b.WriteString(*detail.Snippet)
		b.WriteString("\n")
	}

	// Suggestion if available
	if detail.Suggestion != nil && *detail.Suggestion != "" {
		b.WriteString("\n")
		helpColor := color.New(color.FgYellow, color.Bold)
		helpColor.Fprintf(&b, "  Help: ")
		fmt.Fprintf(&b, "%s\n", strings.ReplaceAll(*detail.Suggestion, "\n", "\n  "))

	}

	return b.String()
}

// LoadSchemaFromStringRaw loads schema and returns raw error (no formatting)
func (e *Engine) LoadSchemaFromStringRaw(input string) (*Schema, string, error) {
	// 1. Validate schema (handles BOTH parse errors and type check errors)
	rawErr, err := ffi.ValidateSchemaRaw(input)
	if err != nil {
		return nil, rawErr, err
	}

	// 2. If validation passed, parse schema
	schemaJSON, err := ffi.ParseSchema(input)
	if err != nil {
		return nil, err.Error(), err
	}

	var schema Schema
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, "", fmt.Errorf("failed to deserialize schema: %w", err)
	}

	e.schema = &schema
	return &schema, "", nil
}
