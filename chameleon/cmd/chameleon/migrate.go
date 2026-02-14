package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/schema"
	"github.com/chameleon-db/chameleondb/chameleon/internal/state"
	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/jackc/pgx/v5"
)

var (
	dryRun         bool
	applyMigration bool
	checkOnly      bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database migrations",
	Long: `Generate, validate, or apply database migrations from schema files.

By default, displays what would be migrated (--check).
Use --apply to execute the migration against the database.
Use --dry-run to preview without applying.

Examples:
  chameleon migrate              # Check for pending migrations
  chameleon migrate --dry-run    # Preview SQL without applying
  chameleon migrate --apply      # Apply pending migrations`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get working directory
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Initialize admin factory
		printInfo("Loading configuration...")
		factory := admin.NewManagerFactory(workDir)

		// Load config
		configLoader := factory.CreateConfigLoader()
		cfg, err := configLoader.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		printSuccess("Configuration loaded from .chameleon.yml")

		// Create journal logger
		journalLogger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Create state tracker
		stateTracker, err := factory.CreateStateTracker()
		if err != nil {
			return fmt.Errorf("failed to initialize state tracker: %w", err)
		}

		// Log migration start
		logDetails := map[string]interface{}{
			"action":  "check",
			"dry_run": dryRun,
			"apply":   applyMigration,
		}
		journalLogger.Log("migrate", "started", logDetails, nil)

		// Load and merge schemas
		printInfo("Loading schemas from: %v", cfg.Schema.Paths)
		eng := engine.NewEngine()

		// Load all schema files using FileLoader
		loader := schema.NewFileLoader(cfg.Schema.Paths)
		filenames, schemaContents, err := loader.LoadAll()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "load_schemas"})
			return fmt.Errorf("failed to load schemas: %w", err)
		}

		printSuccess("Found %d schema file(s): %v", len(filenames), filenames)

		// Merge schemas using SimpleMerger with source tracking
		merger := schema.NewSimpleMerger()
		mergedResult, err := merger.Merge(filenames, schemaContents)
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "merge_schemas"})
			return fmt.Errorf("failed to merge schemas: %w", err)
		}

		mergedSchema := mergedResult.Content
		lineMap := mergedResult.LineMap

		// Validate merged schema
		if err := merger.Validate(mergedSchema); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "validate_schemas"})
			return fmt.Errorf("schema validation failed: %w", err)
		}

		// Parse merged schema (capture errors with source mapping)
		_, err = eng.LoadSchemaFromString(mergedSchema)
		if err != nil {
			// Try to map error line to source file
			errMsg := err.Error()
			sourceInfo := tryMapErrorToSource(errMsg, lineMap)
			if sourceInfo != "" {
				errMsg = strings.ReplaceAll(errMsg, "schema.cham", sourceInfo)
				errMsg = sourceInfo + "\n" + errMsg
			}

			journalLogger.LogError("migrate", fmt.Errorf("%s", errMsg), map[string]interface{}{
				"action": "parse_schema",
				"files":  filenames,
			})

			// Save merged schema for debugging with timestamp
			if len(cfg.Schema.Paths) > 0 {
				debugDir := filepath.Join(filepath.Dir(cfg.Schema.Paths[0]), ".chameleon", "state", "debug")
				os.MkdirAll(debugDir, 0755)
				timestamp := time.Now().Format("20060102-150405")
				debugPath := filepath.Join(debugDir, fmt.Sprintf("schema.merged.%s.cham", timestamp))
				os.WriteFile(debugPath, []byte(mergedSchema), 0644)
				printError("Schema saved to %s for debugging", debugPath)
			}

			return fmt.Errorf("failed to parse merged schemas:\n%s", errMsg)
		}

		printSuccess("Schema loaded and validated")
		mergedSchemaPath := filepath.Join(filepath.Dir(cfg.Schema.Paths[0]), "schema.merged.cham")

		// Get current state
		currentState, err := stateTracker.LoadCurrent()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "load_state"})
			return fmt.Errorf("failed to load current state: %w", err)
		}

		// Generate migration
		printInfo("Generating migration SQL...")
		migrationSQL, err := eng.GenerateMigration()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "generate"})
			return fmt.Errorf("failed to generate migration: %w", err)
		}
		printSuccess("Migration SQL generated")

		// Get last migration
		lastMigration, err := stateTracker.GetLastMigration()
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "get_last_migration"})
			return fmt.Errorf("failed to get last migration: %w", err)
		}

		// Check if schema has changed
		if lastMigration != nil {
			// TODO: Compare hashes to detect changes
			// For now, assume migration is needed if not applied
		}

		// Display migration plan
		fmt.Println()
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println("Migration SQL:")
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println(migrationSQL)
		fmt.Println("─────────────────────────────────────────────────")
		fmt.Println()

		if dryRun || !applyMigration {
			printInfo("Dry-run mode. Use --apply to execute migration.")
			journalLogger.Log("migrate", "dry_run", map[string]interface{}{"action": "check"}, nil)
			return nil
		}

		// Apply migration
		if !applyMigration {
			printInfo("Use --apply to apply this migration.")
			return nil
		}

		printInfo("Connecting to database...")

		// Connect to database
		connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer connCancel()

		conn, err := pgx.Connect(connCtx, cfg.Database.ConnectionString)
		if err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "connect"})
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer conn.Close(connCtx)

		printSuccess("Connected to database")

		// Create backup before applying (if enabled)
		if cfg.Features.BackupOnMigrate {
			printInfo("Creating backup...")
			// TODO: Implement backup
			// backupPath, err := createBackup(conn, cfg)
		}

		// Apply migration
		printInfo("Applying migration...")
		startTime := time.Now()

		_, err = conn.Exec(ctx, migrationSQL)
		if err != nil {
			duration := time.Since(startTime).Milliseconds()
			journalLogger.LogMigration("", "failed", duration, "", map[string]interface{}{
				"error": err.Error(),
			})
			printError("Migration failed")
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		duration := time.Since(startTime).Milliseconds()
		printSuccess("Migration applied successfully")

		// Update state
		printInfo("Updating state...")
		currentState.Status = "in_sync"
		currentState.Migrations.AppliedCount++
		currentState.Migrations.LastAppliedAt = time.Now()

		if err := stateTracker.SaveCurrent(currentState); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "save_state"})
			// Don't fail on state update error, migration was successful
			printError("Warning: Failed to update state: %v", err)
		} else {
			printSuccess("State updated")
		}

		// Add migration to manifest
		migration := &state.Migration{
			Version:     time.Now().Format("20060102-150405"),
			Timestamp:   time.Now(),
			Type:        "initial", // TODO: Detect type (initial, alter, drop)
			Description: "Auto-generated migration",
			AppliedAt:   time.Now(),
			Status:      "applied",
			SchemaHash:  state.HashSchema(mergedSchemaPath),
			DDLHash:     state.HashDDL(migrationSQL),
			Checksum:    "verified",
		}

		if err := stateTracker.AddMigration(migration); err != nil {
			journalLogger.LogError("migrate", err, map[string]interface{}{"action": "add_migration"})
			// Don't fail, migration was successful
			printError("Warning: Failed to record migration: %v", err)
		}

		// Log migration success
		journalLogger.LogMigration(migration.Version, "applied", duration, "", map[string]interface{}{
			"tables_created": 0, // TODO: Count from DDL
		})

		fmt.Println()
		printSuccess("Migration completed successfully!")
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Printf("  Version:  %s\n", migration.Version)
		fmt.Printf("  Duration: %dms\n", duration)
		fmt.Printf("  Status:   applied\n")
		fmt.Println()

		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show migration SQL without applying")
	migrateCmd.Flags().BoolVar(&applyMigration, "apply", false, "apply migration to database")
	migrateCmd.Flags().BoolVar(&checkOnly, "check", false, "only check for pending migrations (default)")

	rootCmd.AddCommand(migrateCmd)
}

// tryMapErrorToSource intenta extraer el número de línea del error
// y mapearlo a archivo origen usando lineMap
// Mejorada - más robusta
func tryMapErrorToSource(errMsg string, lineMap map[int]schema.SourceLine) string {
	// Buscar patrón: "line 25" o "25:" o "line 25 column"
	patterns := []string{
		`line (\d+)`,    // "line 50"
		`-->.*?:(\d+):`, // "--> file:50:5"
		`\s(\d+)\s*│`,   // " 50 │" (formato con línea visual)
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(errMsg)
		if matches != nil && len(matches) > 1 {
			lineNum, _ := strconv.Atoi(matches[1])

			fmt.Fprintf(os.Stderr, "[DEBUG] Found line %d in error, searching lineMap (size: %d)\n", lineNum, len(lineMap))

			// Buscar en lineMap (buscar en rango porque puede haber offset)
			if source, exists := lineMap[lineNum]; exists {
				fmt.Fprintf(os.Stderr, "[DEBUG] Found in lineMap: %s:%d\n", source.File, source.LineNumber)
				return fmt.Sprintf("Error in %s:%d", source.File, source.LineNumber)
			}

			// Si no encuentra exacto, buscar nearest (±5 líneas)
			for offset := 1; offset <= 5; offset++ {
				if source, exists := lineMap[lineNum-offset]; exists {
					fmt.Fprintf(os.Stderr, "[DEBUG] Found nearby (-%d): %s:%d\n", offset, source.File, source.LineNumber+offset)
					return fmt.Sprintf("Error in %s:%d", source.File, source.LineNumber+offset)
				}
				if source, exists := lineMap[lineNum+offset]; exists {
					fmt.Fprintf(os.Stderr, "[DEBUG] Found nearby (+%d): %s:%d\n", offset, source.File, source.LineNumber-offset)
					return fmt.Sprintf("Error in %s:%d", source.File, source.LineNumber-offset)
				}
			}
		}
	}

	return ""
}
