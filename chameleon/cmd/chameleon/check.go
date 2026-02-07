package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var (
	outputJSON bool
)

var checkCmd = &cobra.Command{
	Use:   "check [file]",
	Short: "Check schema for errors (used by editor extensions)",
	Long: `Check a schema file and report errors in JSON format.

This command is designed for editor integrations (VSCode, vim, etc).
It validates the schema and outputs structured error information.

Examples:
  chameleon check schema.cham
  chameleon check schema.cham --json
  chameleon check --json < schema.cham`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine schema file or read from stdin
		var input string
		var filename string

		if len(args) > 0 {
			filename = args[0]
			content, err := os.ReadFile(filename)
			if err != nil {
				if outputJSON {
					return printJSONError(filename, fmt.Sprintf("Failed to read file: %v", err))
				}
				return fmt.Errorf("failed to read file: %w", err)
			}
			input = string(content)
		} else {
			// Read from stdin
			filename = "schema.cham"
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				// No stdin, look for schema.cham
				if _, err := os.Stat("schema.cham"); err == nil {
					content, err := os.ReadFile("schema.cham")
					if err != nil {
						if outputJSON {
							return printJSONError(filename, fmt.Sprintf("Failed to read file: %v", err))
						}
						return fmt.Errorf("failed to read schema.cham: %w", err)
					}
					input = string(content)
				} else {
					if outputJSON {
						return printJSONError(filename, "No input provided and schema.cham not found")
					}
					return fmt.Errorf("no input provided")
				}
			} else {
				// Read from stdin
				content, err := io.ReadAll(os.Stdin)
				if err != nil {
					if outputJSON {
						return printJSONError(filename, fmt.Sprintf("Failed to read stdin: %v", err))
					}
					return fmt.Errorf("failed to read stdin: %w", err)
				}
				input = string(content)
			}
		}

		// Check the schema
		eng := engine.NewEngine()
		_, rawErr, err := eng.LoadSchemaFromStringRaw(input)

		if err != nil {
			if outputJSON {
				return printCheckErrors(filename, rawErr)
			}
			// Human-readable output (use formatted error)
			_, normalErr := eng.LoadSchemaFromString(input)
			if normalErr != nil {
				fmt.Println(normalErr.Error())
			}
			return fmt.Errorf("validation failed")
		}

		// Success
		if outputJSON {
			printJSONSuccess()
		} else {
			printSuccess("Schema is valid")
		}

		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&outputJSON, "json", false, "output errors in JSON format")
	rootCmd.AddCommand(checkCmd)
}

// CheckError represents a single validation error
type CheckError struct {
	Message    string  `json:"message"`
	Line       int     `json:"line"`
	Column     int     `json:"column"`
	File       string  `json:"file"`
	Severity   string  `json:"severity"` // "error" or "warning"
	Snippet    *string `json:"snippet,omitempty"`
	Suggestion *string `json:"suggestion,omitempty"`
}

// CheckResult is the JSON output format
type CheckResult struct {
	Valid  bool         `json:"valid"`
	Errors []CheckError `json:"errors"`
}

func printCheckErrors(filename string, rawErrMsg string) error {
	var result struct {
		Valid  bool `json:"valid"`
		Errors []struct {
			Kind       string  `json:"kind"`
			Message    string  `json:"message"`
			Line       *int    `json:"line"`
			Column     *int    `json:"column"`
			Snippet    *string `json:"snippet"`
			Suggestion *string `json:"suggestion"`
		} `json:"errors"`
	}

	// Try to parse as JSON first
	if err := json.Unmarshal([]byte(rawErrMsg), &result); err == nil && !result.Valid {
		// Successfully parsed as validation JSON
		var errors []CheckError
		for _, err := range result.Errors {
			checkErr := CheckError{
				Message:  err.Message,
				File:     filename,
				Severity: "error",
			}
			if err.Line != nil {
				checkErr.Line = *err.Line
			} else {
				checkErr.Line = 1
			}
			if err.Column != nil {
				checkErr.Column = *err.Column
			} else {
				checkErr.Column = 1
			}
			if err.Snippet != nil {
				checkErr.Snippet = err.Snippet
			}
			if err.Suggestion != nil {
				checkErr.Suggestion = err.Suggestion
			}
			errors = append(errors, checkErr)
		}

		output := CheckResult{
			Valid:  false,
			Errors: errors,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
func printJSONError(filename, message string) error {
	result := CheckResult{
		Valid: false,
		Errors: []CheckError{
			{
				Message:  message,
				Line:     1,
				Column:   1,
				File:     filename,
				Severity: "error",
			},
		},
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func printJSONSuccess() {
	result := CheckResult{
		Valid:  true,
		Errors: []CheckError{},
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}
