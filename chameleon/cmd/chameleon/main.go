package main

import (
	"fmt"
	"os"

	"github.com/dperalta86/chameleondb/chameleon/pkg/engine"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version":
		cmdVersion()
	case "parse":
		if len(os.Args) < 3 {
			fmt.Println("Usage: chameleon parse <schema-file>")
			os.Exit(1)
		}
		cmdParse(os.Args[2])
	case "validate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: chameleon validate <schema-file>")
			os.Exit(1)
		}
		cmdValidate(os.Args[2])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ChameleonDB - Graph-oriented database access language")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  chameleon version              Show version")
	fmt.Println("  chameleon parse <file>         Parse and display schema")
	fmt.Println("  chameleon validate <file>      Validate schema")
}

func cmdVersion() {
	eng := engine.NewEngine()
	fmt.Printf("ChameleonDB v%s\n", eng.Version())
}

func cmdParse(filepath string) {
	eng := engine.NewEngine()

	schema, err := eng.LoadSchemaFromFile(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	json, err := schema.ToJSON()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(json)
}

func cmdValidate(filepath string) {
	eng := engine.NewEngine()

	_, err := eng.LoadSchemaFromFile(filepath)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Schema is valid")
}
