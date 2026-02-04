package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var (
	dryRun         bool
	applyMigration bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate [file]",
	Short: "Generate or apply database migrations",
	Long: `Generate SQL migration from schema or apply it to database.

By default, displays the generated SQL without applying it.
Use --apply to execute the migration against the database.

Examples:
  chameleon migrate                    # Show migration SQL
  chameleon migrate --dry-run          # Same as above
  chameleon migrate --apply            # Apply to database`,
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

		printInfo("Loading schema from %s...", schemaFile)

		// Load schema
		eng := engine.NewEngine()
		_, err := eng.LoadSchemaFromFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema: %w", err)
		}

		printSuccess("Schema loaded and validated")

		// Generate migration
		printInfo("Generating migration SQL...")
		sql, err := eng.GenerateMigration()
		if err != nil {
			return fmt.Errorf("failed to generate migration: %w", err)
		}

		if dryRun || !applyMigration {
			// Just display SQL
			fmt.Println()
			fmt.Println("─────────────────────────────────────────────────")
			fmt.Println(sql)
			fmt.Println("─────────────────────────────────────────────────")
			fmt.Println()

			if !applyMigration {
				printInfo("Dry run mode. Use --apply to execute migration.")
			}

			return nil
		}

		// Apply migration
		printInfo("Connecting to database...")

		// TODO: Read from .chameleon config
		config := engine.DefaultConfig()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = eng.Connect(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer eng.Close()

		printSuccess("Connected to database")
		printInfo("Applying migration...")

		// Execute migration
		connStr := config.ConnectionString()
		conn, err := pgx.Connect(ctx, connStr)
		if err != nil {
			return fmt.Errorf("failed to connect for migration: %w", err)
		}
		defer conn.Close(ctx)

		_, err = conn.Exec(ctx, sql)
		if err != nil {
			printError("Migration failed")
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		printSuccess("Migration applied successfully")

		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show migration SQL without applying")
	migrateCmd.Flags().BoolVar(&applyMigration, "apply", false, "apply migration to database")

	rootCmd.AddCommand(migrateCmd)
}
