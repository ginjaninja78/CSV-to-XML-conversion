// =============================================================================
// CSV to XML Converter - Main Entry Point
// =============================================================================
//
// This is the main entry point for the CSV to XML Converter CLI application.
// It initializes the Cobra CLI framework and delegates command execution to
// the cmd package.
//
// USAGE:
//   converter process       - Process all CSV files in the input directory
//   converter validate      - Validate configuration files without processing
//   converter version       - Display the application version
//
// ARCHITECTURE:
//   This application follows a modular design where:
//   - cmd/           : Contains all CLI command definitions (Cobra)
//   - internal/      : Contains core business logic (not for external import)
//   - pkg/           : Contains shared utilities
//   - configs/       : Contains department-specific YAML configurations
//   - templates/     : Contains XLSX schema definition templates
//
// =============================================================================

package main

import (
	"github.com/ginjaninja78/CSV-to-XML-conversion/cmd"
)

// main is the entry point of the application.
// It simply calls the Execute function from the cmd package, which
// initializes and runs the Cobra CLI.
func main() {
	cmd.Execute()
}
