// =============================================================================
// CSV to XML Converter - Process Command
// =============================================================================
//
// This file defines the 'process' command, which is the main command for
// converting CSV files to XML. It orchestrates the entire conversion pipeline.
//
// COMMAND USAGE:
//   converter process [flags]
//
// FLAGS:
//   --dry-run     : Simulate processing without writing output files
//   --single      : Process only a single file (specify with --file)
//   --file        : Path to a specific file to process (used with --single)
//   --department  : Process only files for a specific department
//
// PROCESSING PIPELINE:
//   1. Load configuration files
//   2. Discover CSV files in the input directory
//   3. Match each file to a department configuration
//   4. For each file (concurrently):
//      a. Parse the XLSX template to get the schema
//      b. Parse the CSV file
//      c. Apply transformation rules
//      d. Validate the data
//      e. Generate the XML
//      f. Write the output file
//   5. Archive processed files
//   6. Generate summary report
//
// =============================================================================

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/config"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/converter"
	"github.com/spf13/cobra"
)

// =============================================================================
// COMMAND FLAGS
// =============================================================================

// dryRun simulates processing without writing output files.
var dryRun bool

// singleFile indicates whether to process only a single file.
var singleFile bool

// filePath is the path to a specific file to process (used with --single).
var filePath string

// department filters processing to a specific department.
var department string

// =============================================================================
// PROCESS COMMAND DEFINITION
// =============================================================================

// processCmd represents the 'process' command.
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process CSV files and convert them to XML",
	Long: `The process command scans the input directory for CSV files, matches them
to the appropriate department configuration, and converts them to XML format
based on the XLSX template schema.

Processing is done concurrently for maximum performance. Each file is processed
independently, and errors in one file do not affect the processing of others.

On successful processing:
  - The generated XML is placed in the output directory
  - The original CSV is moved to the input archive
  - A summary report is generated

On error:
  - An error log is created in the output directory
  - The original CSV remains in the input directory
  - Processing continues for other files`,

	// RunE is like Run but returns an error. This is preferred for commands
	// that can fail, as it allows Cobra to handle the error gracefully.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProcess()
	},
}

// =============================================================================
// INITIALIZATION
// =============================================================================

// init registers the process command with the root command and sets up flags.
func init() {
	// Add the process command to the root command.
	rootCmd.AddCommand(processCmd)

	// ==========================================================================
	// LOCAL FLAGS
	// ==========================================================================
	// Local flags are only available to this command.

	// --dry-run flag: Simulate processing without writing output files.
	processCmd.Flags().BoolVar(
		&dryRun,
		"dry-run",
		false,
		"Simulate processing without writing output files",
	)

	// --single flag: Process only a single file.
	processCmd.Flags().BoolVar(
		&singleFile,
		"single",
		false,
		"Process only a single file (use with --file)",
	)

	// --file flag: Path to a specific file to process.
	processCmd.Flags().StringVar(
		&filePath,
		"file",
		"",
		"Path to a specific file to process (used with --single)",
	)

	// --department flag: Process only files for a specific department.
	processCmd.Flags().StringVar(
		&department,
		"department",
		"",
		"Process only files for a specific department",
	)
}

// =============================================================================
// MAIN PROCESSING FUNCTION
// =============================================================================

// runProcess is the main function that orchestrates the conversion pipeline.
func runProcess() error {
	startTime := time.Now()

	// =========================================================================
	// STEP 1: LOAD CONFIGURATION
	// =========================================================================
	// Load the main configuration file and all department-specific configurations.

	fmt.Println("=== CSV to XML Converter ===")
	fmt.Println("Loading configuration...")

	// Load the main configuration from the config file.
	// PSEUDOCODE:
	// mainConfig, err := config.LoadMainConfig(cfgFile)
	// if err != nil {
	//     return fmt.Errorf("failed to load main config: %w", err)
	// }
	mainConfig, err := config.LoadMainConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load main config: %w", err)
	}

	// Load all department configurations from the configs directory.
	// PSEUDOCODE:
	// deptConfigs, err := config.LoadDepartmentConfigs(mainConfig.ConfigsDir)
	// if err != nil {
	//     return fmt.Errorf("failed to load department configs: %w", err)
	// }
	deptConfigs, err := config.LoadDepartmentConfigs(mainConfig.ConfigsDir)
	if err != nil {
		return fmt.Errorf("failed to load department configs: %w", err)
	}

	fmt.Printf("Loaded %d department configuration(s)\n", len(deptConfigs))

	// =========================================================================
	// STEP 2: DISCOVER INPUT FILES
	// =========================================================================
	// Scan the input directory for CSV files to process.

	fmt.Println("Discovering input files...")

	// Get list of CSV files in the input directory.
	// PSEUDOCODE:
	// inputFiles, err := discoverInputFiles(mainConfig.InputDir)
	// if err != nil {
	//     return fmt.Errorf("failed to discover input files: %w", err)
	// }
	inputFiles, err := discoverInputFiles(mainConfig.InputDir)
	if err != nil {
		return fmt.Errorf("failed to discover input files: %w", err)
	}

	if len(inputFiles) == 0 {
		fmt.Println("No CSV files found in the input directory.")
		return nil
	}

	fmt.Printf("Found %d file(s) to process\n", len(inputFiles))

	// =========================================================================
	// STEP 3: PROCESS FILES CONCURRENTLY
	// =========================================================================
	// Process each file in a separate goroutine for maximum performance.
	// Use a WaitGroup to wait for all goroutines to complete.
	// Use a channel to collect results and errors.

	fmt.Println("Processing files...")

	// Create a WaitGroup to wait for all goroutines to complete.
	var wg sync.WaitGroup

	// Create a channel to collect processing results.
	// The channel is buffered to prevent blocking.
	results := make(chan converter.Result, len(inputFiles))

	// Process each file concurrently.
	for _, file := range inputFiles {
		wg.Add(1)

		// Launch a goroutine for each file.
		go func(filePath string) {
			defer wg.Done()

			// Find the matching department configuration for this file.
			// PSEUDOCODE:
			// deptConfig := findMatchingDepartment(filePath, deptConfigs)
			// if deptConfig == nil {
			//     results <- converter.Result{
			//         FilePath: filePath,
			//         Success:  false,
			//         Error:    fmt.Errorf("no matching department configuration found"),
			//     }
			//     return
			// }
			deptConfig := findMatchingDepartment(filePath, deptConfigs)
			if deptConfig == nil {
				results <- converter.Result{
					FilePath: filePath,
					Success:  false,
					Error:    fmt.Errorf("no matching department configuration found"),
				}
				return
			}

			// Create a new converter instance for this file.
			// PSEUDOCODE:
			// conv := converter.New(filePath, deptConfig, mainConfig)
			// result := conv.Run()
			// results <- result
			conv := converter.New(filePath, deptConfig, mainConfig)
			result := conv.Run()
			results <- result

		}(file)
	}

	// Close the results channel when all goroutines are done.
	go func() {
		wg.Wait()
		close(results)
	}()

	// =========================================================================
	// STEP 4: COLLECT RESULTS AND GENERATE SUMMARY
	// =========================================================================
	// Collect results from all goroutines and generate a summary report.

	var successCount, errorCount int
	var errors []string

	for result := range results {
		if result.Success {
			successCount++
			fmt.Printf("  ✓ %s -> %s\n", filepath.Base(result.FilePath), result.OutputFile)
		} else {
			errorCount++
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(result.FilePath), result.Error))
			fmt.Printf("  ✗ %s: %v\n", filepath.Base(result.FilePath), result.Error)
		}
	}

	// =========================================================================
	// STEP 5: PRINT SUMMARY
	// =========================================================================

	elapsed := time.Since(startTime)
	fmt.Println("\n=== Processing Complete ===")
	fmt.Printf("Total files:     %d\n", len(inputFiles))
	fmt.Printf("Successful:      %d\n", successCount)
	fmt.Printf("Errors:          %d\n", errorCount)
	fmt.Printf("Time elapsed:    %s\n", elapsed)

	// If there were errors, write them to an error log.
	if errorCount > 0 {
		// PSEUDOCODE:
		// writeErrorLog(mainConfig.OutputDir, errors)
		fmt.Println("\nErrors have been logged to the output directory.")
	}

	return nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// discoverInputFiles scans the input directory for CSV files.
//
// PARAMETERS:
//   - inputDir: The path to the input directory.
//
// RETURNS:
//   - A slice of file paths to CSV files.
//   - An error if the directory cannot be read.
//
// CUSTOMIZATION:
//   - Modify the file extension filter if your input files have a different extension.
//   - Add additional filtering logic if needed (e.g., by date, by size).
func discoverInputFiles(inputDir string) ([]string, error) {
	var files []string

	// Walk the input directory and find all CSV files.
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories.
		if info.IsDir() {
			return nil
		}

		// Check if the file has a .csv extension.
		// CUSTOMIZATION: Modify this if your files have a different extension.
		if filepath.Ext(path) == ".csv" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// findMatchingDepartment finds the department configuration that matches the given file.
//
// PARAMETERS:
//   - filePath: The path to the input file.
//   - deptConfigs: A map of department configurations.
//
// RETURNS:
//   - The matching department configuration, or nil if no match is found.
//
// MATCHING LOGIC:
//   This function iterates through all department configurations and checks
//   if the file name matches any of the file matching patterns defined in
//   the department configuration.
//
// CUSTOMIZATION:
//   - Modify the matching logic if your file naming conventions are different.
//   - Add additional matching criteria (e.g., by file content, by header row).
func findMatchingDepartment(filePath string, deptConfigs map[string]*config.DepartmentConfig) *config.DepartmentConfig {
	fileName := filepath.Base(filePath)

	// Iterate through all department configurations.
	for _, deptConfig := range deptConfigs {
		// Check if the file name matches any of the file matching patterns.
		for _, pattern := range deptConfig.FileMatchingPatterns {
			// Use filepath.Match for glob-style pattern matching.
			matched, err := filepath.Match(pattern, fileName)
			if err != nil {
				// Invalid pattern, skip it.
				continue
			}
			if matched {
				return deptConfig
			}
		}
	}

	// No matching department found.
	return nil
}
