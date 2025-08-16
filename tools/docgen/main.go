package main

import (
	"log"
	"os"
	"path/filepath"

	"replay/cmd"

	"github.com/spf13/cobra/doc"
)

func main() {
	// Default output directory
	outputDir := "./docs/"

	// If an argument is provided, use it as the output directory
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Get the root command from the main application
	rootCmd := cmd.GetRootCmd()

	// Generate Markdown documentation
	if err := doc.GenMarkdownTree(rootCmd, outputDir); err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	absPath, err := filepath.Abs(outputDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	log.Printf("Documentation successfully generated in %s", absPath)
}
