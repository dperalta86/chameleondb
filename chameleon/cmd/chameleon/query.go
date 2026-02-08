package main

import (
	"context"
	"fmt"

	"github.com/chameleon-db/chameleondb/chameleon/pkg/engine"
	"github.com/spf13/cobra"
)

var (
	queryDebug   bool
	queryTrace   bool
	queryExplain bool
)

var queryCmd = &cobra.Command{
	Use:   "query [entity]",
	Short: "Interactive query execution (for testing)",
	Long: `Execute queries interactively with debug output.
    
Examples:
  chameleon query User --debug
  chameleon query Post --trace
  chameleon query Order --explain`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entity := args[0]

		// Setup engine
		eng := engine.NewEngine()
		eng.LoadSchemaFromFile("schema.cham")

		// Set debug level
		if queryExplain {
			eng.Debug.Level = engine.DebugExplain
		} else if queryTrace {
			eng.Debug.Level = engine.DebugTrace
		} else if queryDebug {
			eng.Debug.Level = engine.DebugSQL
		}

		// Connect
		config := getConfigFromEnv()
		ctx := context.Background()
		eng.Connect(ctx, config)
		defer eng.Close()

		// Execute query
		result, err := eng.Query(entity).Execute(ctx)
		if err != nil {
			return err
		}

		// Display results
		fmt.Printf("\n✓ Retrieved %d row(s)\n", len(result.Rows))

		return nil
	},
}

func init() {
	queryCmd.Flags().BoolVar(&queryDebug, "debug", false, "show generated SQL")
	queryCmd.Flags().BoolVar(&queryTrace, "trace", false, "show full query trace")
	queryCmd.Flags().BoolVar(&queryExplain, "explain", false, "show query plan")

	rootCmd.AddCommand(queryCmd)
}
