package main

import (
	"fmt"
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate a ChameleonDB schema",
	Long: `Validate a schema file for syntax and semantic errors.

If no file is specified, looks for 'schema.cham' in current directory.

Examples:
  chameleon validate
  chameleon validate schema.cham
  chameleon validate path/to/schema.cham`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine schema file
		schemaFile := "schema.cham"
		if len(args) > 0 {
			schemaFile = args[0]
		}

		// Check file exists
		if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
			return fmt.Errorf("schema file not found: %s", schemaFile)
		}

		printInfo("Validating %s...", schemaFile)

		// Load and validate
		eng := engine.NewEngine()
		_, err := eng.LoadSchemaFromFile(schemaFile)
		if err != nil {
			printError("Validation failed")
			fmt.Println()

			// Try to format as structured error
			errStr := err.Error()
			formatted := engine.FormatError(errStr)
			fmt.Println(formatted)

			return fmt.Errorf("schema validation failed")
		}

		printSuccess("Schema is valid")

		if verbose {
			fmt.Println("\nValidation checks passed:")
			fmt.Println("  ✓ Syntax is correct")
			fmt.Println("  ✓ All entity references exist")
			fmt.Println("  ✓ Foreign keys are consistent")
			fmt.Println("  ✓ Primary keys are defined")
			fmt.Println("  ✓ No circular dependencies")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
