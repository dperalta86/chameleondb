package main

import (
	"os"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool

	// Colors
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow)
	infoColor    = color.New(color.FgCyan)
)

var rootCmd = &cobra.Command{
	Use:   "chameleon",
	Short: "ChameleonDB - Type-safe database access language",
	Long: `ChameleonDB is a graph-oriented, strongly-typed database access language
that compiles schemas to migrations and validates queries at compile time.

Get started:
  chameleon init myproject
  cd myproject
  chameleon validate
  chameleon migrate --apply`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		errorColor.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Helper functions for consistent output
func printSuccess(format string, args ...interface{}) {
	successColor.Printf("✓ "+format+"\n", args...)
}

func printError(format string, args ...interface{}) {
	errorColor.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
}

func printWarning(format string, args ...interface{}) {
	warningColor.Printf("⚠ "+format+"\n", args...)
}

func printInfo(format string, args ...interface{}) {
	infoColor.Printf("ℹ "+format+"\n", args...)
}

func getConfigFromEnv() engine.ConnectorConfig {
	config, err := LoadConnectorConfig()
	if err != nil {
		printWarning("Could not read config: %v", err)
		return engine.DefaultConfig()
	}
	return config
}

/* func initEngine(schemaPath string) (*engine.Engine, error) {
	eng, err := engine.NewEngineWithSchema(schemaPath)
	if err != nil {
		return nil, err
	}

	mutation.Register(eng)
	return eng, nil
} */
