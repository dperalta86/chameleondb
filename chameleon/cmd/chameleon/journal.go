package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/chameleon-db/chameleondb/chameleon/internal/admin"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
)

var (
	journalLimit  int
	journalFormat string
)

var journalCmd = &cobra.Command{
	Use:   "journal <subcommand>",
	Short: "Query and audit the operation journal",
	Long: `View and search the operation journal (audit log).

The journal is an append-only log of all ChameleonDB operations.
Stored in .chameleon/journal/ with daily rotation.

Subcommands:
  journal last       Show last N operations
  journal errors     Show error operations
  journal migrations Show migration history
  journal search     Search journal entries`,
	Args: cobra.MinimumNArgs(1),
}

var journalLastCmd = &cobra.Command{
	Use:   "last [n]",
	Short: "Show last N journal entries",
	Long: `Display the most recent journal entries.

Examples:
  chameleon journal last        # Last 10 entries
  chameleon journal last 20     # Last 20 entries
  chameleon journal last 5 --format=json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Get limit
		limit := 10
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid number: %s", args[0])
			}
			limit = n
		}

		// Initialize journal logger
		factory := admin.NewManagerFactory(workDir)
		logger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Get last entries
		entries, err := logger.Last(limit)
		if err != nil {
			return fmt.Errorf("failed to read journal: %w", err)
		}

		if len(entries) == 0 {
			printInfo("No journal entries found")
			return nil
		}

		// Format output
		if journalFormat == "json" {
			printEntriesJSON(entries)
		} else {
			printEntriesTable(entries)
		}

		return nil
	},
}

var journalErrorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Show error journal entries",
	Long: `Display all error operations from today's journal.

Examples:
  chameleon journal errors
  chameleon journal errors --format=json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Initialize journal logger
		factory := admin.NewManagerFactory(workDir)
		logger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Get error entries
		entries, err := logger.Errors()
		if err != nil {
			return fmt.Errorf("failed to read journal: %w", err)
		}

		if len(entries) == 0 {
			printSuccess("No errors found")
			return nil
		}

		// Format output
		if journalFormat == "json" {
			printEntriesJSON(entries)
		} else {
			printEntriesTable(entries)
		}

		return nil
	},
}

var journalMigrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "Show migration history",
	Long: `Display all migration operations from the journal.

Examples:
  chameleon journal migrations
  chameleon journal migrations --format=json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Initialize journal logger
		factory := admin.NewManagerFactory(workDir)
		logger, err := factory.CreateJournalLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize journal: %w", err)
		}

		// Get migration entries
		entries, err := logger.Migrations()
		if err != nil {
			return fmt.Errorf("failed to read journal: %w", err)
		}

		if len(entries) == 0 {
			printInfo("No migration entries found")
			return nil
		}

		// Format output
		if journalFormat == "json" {
			printEntriesJSON(entries)
		} else {
			printMigrationsTable(entries)
		}

		return nil
	},
}

func init() {
	// Add journal subcommands
	journalCmd.AddCommand(journalLastCmd)
	journalCmd.AddCommand(journalErrorsCmd)
	journalCmd.AddCommand(journalMigrationsCmd)

	// Add flags
	journalLastCmd.Flags().IntVar(&journalLimit, "limit", 10, "number of entries to show")
	journalCmd.PersistentFlags().StringVar(&journalFormat, "format", "table", "output format (table|json)")

	rootCmd.AddCommand(journalCmd)
}

// printEntriesTable prints entries in table format
func printEntriesTable(entries []*journal.Entry) {
	fmt.Println()
	fmt.Println("Timestamp                Action      Status      Details")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	for _, entry := range entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		status := entry.Status
		if entry.Error != "" {
			status = "error"
		}

		details := ""
		if entry.Duration > 0 {
			details = fmt.Sprintf("duration=%dms", entry.Duration)
		}
		if entry.Error != "" {
			if details != "" {
				details += " "
			}
			details += fmt.Sprintf("error=%s", truncate(entry.Error, 50))
		}

		fmt.Printf("%-25s %-11s %-11s %s\n", timestamp, entry.Action, status, details)
	}

	fmt.Println()
}

// printMigrationsTable prints migration entries in table format
func printMigrationsTable(entries []*journal.Entry) {
	fmt.Println()
	fmt.Println("Timestamp                Version              Status    Duration")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	for _, entry := range entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

		version := ""
		if v, ok := entry.Details["version"].(string); ok {
			version = v
		}

		status := entry.Status
		duration := ""
		if entry.Duration > 0 {
			duration = fmt.Sprintf("%dms", entry.Duration)
		}

		fmt.Printf("%-25s %-20s %-9s %s\n", timestamp, version, status, duration)
	}

	fmt.Println()
}

// printEntriesJSON prints entries in JSON format
func printEntriesJSON(entries []*journal.Entry) {
	fmt.Println("[")
	for i, entry := range entries {
		fmt.Printf("  {\n")
		fmt.Printf("    \"timestamp\": \"%s\",\n", entry.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
		fmt.Printf("    \"action\": \"%s\",\n", entry.Action)
		fmt.Printf("    \"status\": \"%s\",\n", entry.Status)

		if entry.Duration > 0 {
			fmt.Printf("    \"duration_ms\": %d,\n", entry.Duration)
		}

		if entry.Error != "" {
			fmt.Printf("    \"error\": \"%s\"\n", entry.Error)
		} else {
			fmt.Printf("    \"error\": null\n")
		}

		fmt.Printf("  }")
		if i < len(entries)-1 {
			fmt.Printf(",")
		}
		fmt.Printf("\n")
	}
	fmt.Println("]")
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
