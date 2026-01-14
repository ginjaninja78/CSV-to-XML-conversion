// =============================================================================
// CSV to XML Converter - Version Command
// =============================================================================
//
// This file defines the 'version' command, which displays the application
// version and build information.
//
// COMMAND USAGE:
//   converter version
//
// OUTPUT:
//   CSV to XML Converter
//   Version:    1.0.0
//   Build Date: 2024-01-01
//   Go Version: go1.22.0
//
// =============================================================================

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// =============================================================================
// VERSION INFORMATION
// =============================================================================
// These variables are set at build time using ldflags.
// Example build command:
//   go build -ldflags "-X 'cmd.Version=1.0.0' -X 'cmd.BuildDate=2024-01-01'"

// Version is the application version.
// Set at build time using ldflags.
var Version = "1.0.0"

// BuildDate is the date the application was built.
// Set at build time using ldflags.
var BuildDate = "unknown"

// =============================================================================
// VERSION COMMAND DEFINITION
// =============================================================================

// versionCmd represents the 'version' command.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the application version",
	Long:  `Display the application version, build date, and Go runtime version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CSV to XML Converter")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("Go Version: %s\n", runtime.Version())
	},
}

// =============================================================================
// INITIALIZATION
// =============================================================================

// init registers the version command with the root command.
func init() {
	rootCmd.AddCommand(versionCmd)
}
