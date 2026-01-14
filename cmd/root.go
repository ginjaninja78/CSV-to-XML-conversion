// =============================================================================
// CSV to XML Converter - Root Command
// =============================================================================
//
// This file defines the root command for the Cobra CLI. The root command is
// the base command that all other commands (like 'process', 'validate') are
// attached to.
//
// COBRA CLI STRUCTURE:
//   rootCmd (converter)
//   ├── processCmd (converter process)
//   ├── validateCmd (converter validate)
//   └── versionCmd (converter version)
//
// CONFIGURATION:
//   The root command is responsible for:
//   1. Setting up global flags (e.g., --config, --verbose)
//   2. Initializing the configuration system
//   3. Setting up logging
//
// =============================================================================

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// =============================================================================
// GLOBAL VARIABLES
// =============================================================================

// cfgFile holds the path to the main configuration file.
// This can be overridden using the --config flag.
var cfgFile string

// verbose enables verbose logging when set to true.
var verbose bool

// =============================================================================
// ROOT COMMAND DEFINITION
// =============================================================================

// rootCmd represents the base command when called without any subcommands.
// This is the entry point for the CLI application.
var rootCmd = &cobra.Command{
	// Use is the one-line usage message.
	// This is what appears in help text and error messages.
	Use: "converter",

	// Short is a short description shown in the 'help' output.
	Short: "CSV to XML Converter - Transform legacy CSV exports to XML for bulk upload",

	// Long is a longer description shown in the 'help <command>' output.
	Long: `CSV to XML Converter is a high-performance CLI tool designed to transform
CSV files exported from legacy reporting systems into XML files suitable for
bulk upload to modern financial systems.

Key Features:
  - Dynamic schema definition via XLSX templates
  - Department-specific configuration and transformation rules
  - Robust validation with detailed error reporting
  - Concurrent processing for high throughput
  - Automatic file archival on successful processing

Example Usage:
  converter process                    # Process all files in the input directory
  converter process --config ./my.yaml # Use a custom configuration file
  converter validate                   # Validate configuration without processing`,

	// Run is the function that will be executed when the root command is called
	// without any subcommands. In this case, we just print the help message.
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, print the help message.
		cmd.Help()
	},
}

// =============================================================================
// EXECUTE FUNCTION
// =============================================================================

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Execute the root command. If there's an error, print it and exit.
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// =============================================================================
// INITIALIZATION
// =============================================================================

// init is called automatically when the package is loaded.
// It sets up the global flags and configuration initialization.
func init() {
	// ==========================================================================
	// PERSISTENT FLAGS
	// ==========================================================================
	// Persistent flags are available to this command and all subcommands.

	// --config flag: Allows the user to specify a custom configuration file.
	// Default is "config.yaml" in the current directory.
	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"config.yaml",
		"Path to the main configuration file (default is config.yaml)",
	)

	// --verbose flag: Enables verbose/debug logging.
	rootCmd.PersistentFlags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"Enable verbose output for debugging",
	)

	// ==========================================================================
	// CONFIGURATION INITIALIZATION
	// ==========================================================================
	// This is where you would initialize Viper or another configuration library
	// to read the configuration file. For now, we'll handle this in the
	// individual commands.

	// PSEUDOCODE:
	// cobra.OnInitialize(initConfig)
	//
	// func initConfig() {
	//     if cfgFile != "" {
	//         viper.SetConfigFile(cfgFile)
	//     } else {
	//         viper.SetConfigName("config")
	//         viper.SetConfigType("yaml")
	//         viper.AddConfigPath(".")
	//     }
	//     viper.AutomaticEnv()
	//     if err := viper.ReadInConfig(); err == nil {
	//         fmt.Println("Using config file:", viper.ConfigFileUsed())
	//     }
	// }
}
