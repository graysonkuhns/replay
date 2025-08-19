package main

import (
	"os"
	"path/filepath"

	"replay/cmd"
	"replay/logger"

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
	log := logger.NewLogger()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Error("Failed to create output directory", err)
		os.Exit(1)
	}

	// Get the root command from the main application
	rootCmd := cmd.GetRootCmd()

	// Generate Markdown documentation
	if err := doc.GenMarkdownTree(rootCmd, outputDir); err != nil {
		log.Error("Failed to generate documentation", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(outputDir)
	if err != nil {
		log.Error("Failed to get absolute path", err)
		os.Exit(1)
	}

	log.Info("Documentation successfully generated", logger.String("path", absPath))
}
