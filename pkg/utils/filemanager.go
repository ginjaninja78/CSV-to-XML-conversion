// =============================================================================
// CSV to XML Converter - File Manager Utility
// =============================================================================
//
// This module provides file management utilities for the converter, including:
//   - File discovery and scanning
//   - File archival (moving processed files)
//   - Error log generation
//   - Directory management
//   - File naming utilities
//
// ARCHIVAL STRATEGY:
//   - Input files are moved to input_archive after successful processing
//   - Output files are copied to output_archive for long-term storage
//   - Failed files remain in their original location
//   - Error logs are created in the output directory
//
// CUSTOMIZATION:
//   - Modify archival behavior (copy vs. move, date-based subdirectories)
//   - Add file compression for archives
//   - Implement retention policies for old archives
//
// =============================================================================

package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// FILE MANAGER
// =============================================================================

// FileManager handles file operations for the converter.
type FileManager struct {
	// InputDir is the directory where input files are placed.
	InputDir string

	// OutputDir is the directory where output files are placed.
	OutputDir string

	// InputArchiveDir is the directory for archived input files.
	InputArchiveDir string

	// OutputArchiveDir is the directory for archived output files.
	OutputArchiveDir string

	// UseTimestampSubdirs creates date-based subdirectories in archives.
	// Example: input_archive/2024/01/15/file.csv
	UseTimestampSubdirs bool

	// ArchiveOnSuccess determines whether to archive files after successful processing.
	ArchiveOnSuccess bool
}

// NewFileManager creates a new FileManager with the specified directories.
func NewFileManager(inputDir, outputDir, inputArchiveDir, outputArchiveDir string) *FileManager {
	return &FileManager{
		InputDir:            inputDir,
		OutputDir:           outputDir,
		InputArchiveDir:     inputArchiveDir,
		OutputArchiveDir:    outputArchiveDir,
		UseTimestampSubdirs: false,
		ArchiveOnSuccess:    true,
	}
}

// =============================================================================
// DIRECTORY MANAGEMENT
// =============================================================================

// EnsureDirectories creates all required directories if they don't exist.
//
// RETURNS:
//   - An error if any directory cannot be created.
func (fm *FileManager) EnsureDirectories() error {
	dirs := []string{
		fm.InputDir,
		fm.OutputDir,
		fm.InputArchiveDir,
		fm.OutputArchiveDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// =============================================================================
// FILE DISCOVERY
// =============================================================================

// DiscoverInputFiles scans the input directory for files matching the pattern.
//
// PARAMETERS:
//   - pattern: A glob pattern to match files (e.g., "*.csv").
//              If empty, defaults to "*.csv".
//
// RETURNS:
//   - A slice of file paths.
//   - An error if the directory cannot be read.
//
// CUSTOMIZATION:
//   - Add filtering by file size, modification date, etc.
//   - Add support for recursive directory scanning.
func (fm *FileManager) DiscoverInputFiles(pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*.csv"
	}

	// Construct the full pattern path.
	fullPattern := filepath.Join(fm.InputDir, pattern)

	// Find matching files.
	files, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan input directory: %w", err)
	}

	// Filter out directories.
	var result []string
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			result = append(result, file)
		}
	}

	return result, nil
}

// DiscoverInputFilesRecursive scans the input directory recursively.
//
// PARAMETERS:
//   - extension: The file extension to match (e.g., ".csv").
//
// RETURNS:
//   - A slice of file paths.
//   - An error if the directory cannot be read.
func (fm *FileManager) DiscoverInputFilesRecursive(extension string) ([]string, error) {
	var files []string

	err := filepath.Walk(fm.InputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if extension == "" || strings.HasSuffix(strings.ToLower(path), strings.ToLower(extension)) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk input directory: %w", err)
	}

	return files, nil
}

// =============================================================================
// FILE ARCHIVAL
// =============================================================================

// ArchiveInputFile moves an input file to the archive directory.
//
// PARAMETERS:
//   - filePath: The path to the file to archive.
//
// RETURNS:
//   - The path to the archived file.
//   - An error if archival fails.
func (fm *FileManager) ArchiveInputFile(filePath string) (string, error) {
	if !fm.ArchiveOnSuccess {
		return filePath, nil
	}

	// Determine the archive path.
	archivePath := fm.getArchivePath(fm.InputArchiveDir, filePath)

	// Ensure the archive directory exists.
	archiveDir := filepath.Dir(archivePath)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Move the file.
	if err := os.Rename(filePath, archivePath); err != nil {
		// If rename fails (e.g., cross-device), try copy and delete.
		if err := copyFile(filePath, archivePath); err != nil {
			return "", fmt.Errorf("failed to copy file to archive: %w", err)
		}
		if err := os.Remove(filePath); err != nil {
			return "", fmt.Errorf("failed to remove original file: %w", err)
		}
	}

	return archivePath, nil
}

// ArchiveOutputFile copies an output file to the archive directory.
//
// PARAMETERS:
//   - filePath: The path to the file to archive.
//
// RETURNS:
//   - The path to the archived file.
//   - An error if archival fails.
//
// NOTE: Output files are copied, not moved, so they remain in the output directory.
func (fm *FileManager) ArchiveOutputFile(filePath string) (string, error) {
	if !fm.ArchiveOnSuccess {
		return filePath, nil
	}

	// Determine the archive path.
	archivePath := fm.getArchivePath(fm.OutputArchiveDir, filePath)

	// Ensure the archive directory exists.
	archiveDir := filepath.Dir(archivePath)
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Copy the file.
	if err := copyFile(filePath, archivePath); err != nil {
		return "", fmt.Errorf("failed to copy file to archive: %w", err)
	}

	return archivePath, nil
}

// getArchivePath constructs the archive path for a file.
func (fm *FileManager) getArchivePath(archiveDir, filePath string) string {
	fileName := filepath.Base(filePath)

	if fm.UseTimestampSubdirs {
		// Create date-based subdirectory structure.
		now := time.Now()
		subDir := filepath.Join(
			archiveDir,
			fmt.Sprintf("%d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
		)
		return filepath.Join(subDir, fileName)
	}

	return filepath.Join(archiveDir, fileName)
}

// =============================================================================
// OUTPUT FILE NAMING
// =============================================================================

// GenerateOutputFileName generates a unique output file name.
//
// PARAMETERS:
//   - format: The format string for the file name.
//             Placeholders:
//               {uuid}      - A random UUID
//               {timestamp} - Current timestamp (YYYYMMDD_HHMMSS)
//               {date}      - Current date (YYYYMMDD)
//               {time}      - Current time (HHMMSS)
//               {dept}      - Department code
//               {type}      - Transaction type
//               {original}  - Original file name (without extension)
//   - params: A map of placeholder values.
//
// RETURNS:
//   - The generated file name.
//
// EXAMPLE:
//   format: "{dept}_{timestamp}_{uuid}.xml"
//   params: {"dept": "CLAIMS"}
//   output: "CLAIMS_20240115_143022_a1b2c3d4-e5f6-7890-abcd-ef1234567890.xml"
func GenerateOutputFileName(format string, params map[string]string) string {
	now := time.Now()

	// Generate UUID.
	id := uuid.New().String()

	// Build replacements.
	replacements := map[string]string{
		"{uuid}":      id,
		"{timestamp}": now.Format("20060102_150405"),
		"{date}":      now.Format("20060102"),
		"{time}":      now.Format("150405"),
	}

	// Add custom params.
	for key, value := range params {
		replacements["{"+key+"}"] = value
	}

	// Apply replacements.
	result := format
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Ensure .xml extension.
	if !strings.HasSuffix(strings.ToLower(result), ".xml") {
		result += ".xml"
	}

	return result
}

// =============================================================================
// ERROR LOG GENERATION
// =============================================================================

// ErrorLogEntry represents a single error log entry.
type ErrorLogEntry struct {
	Timestamp     time.Time
	FileName      string
	ErrorType     string
	ErrorMessage  string
	RowNumber     int
	FieldName     string
	FieldValue    string
	TransactionID int
	LineItemID    int
}

// WriteErrorLog writes error entries to a log file.
//
// PARAMETERS:
//   - entries: The error entries to write.
//   - outputDir: The directory to write the log file.
//
// RETURNS:
//   - The path to the error log file.
//   - An error if writing fails.
func WriteErrorLog(entries []ErrorLogEntry, outputDir string) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}

	// Generate log file name.
	timestamp := time.Now().Format("20060102_150405")
	logFileName := fmt.Sprintf("error_log_%s.txt", timestamp)
	logPath := filepath.Join(outputDir, logFileName)

	// Create the file.
	file, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to create error log: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write header.
	header := fmt.Sprintf("CSV to XML Converter - Error Log\n"+
		"Generated: %s\n"+
		"Total Errors: %d\n"+
		"================================================================================\n\n",
		time.Now().Format("2006-01-02 15:04:05"),
		len(entries))
	writer.WriteString(header)

	// Write each entry.
	for i, entry := range entries {
		entryStr := fmt.Sprintf("Error #%d\n"+
			"  Timestamp:      %s\n"+
			"  File:           %s\n"+
			"  Error Type:     %s\n"+
			"  Message:        %s\n",
			i+1,
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.FileName,
			entry.ErrorType,
			entry.ErrorMessage)

		if entry.RowNumber > 0 {
			entryStr += fmt.Sprintf("  Row Number:     %d\n", entry.RowNumber)
		}
		if entry.FieldName != "" {
			entryStr += fmt.Sprintf("  Field:          %s\n", entry.FieldName)
		}
		if entry.FieldValue != "" {
			entryStr += fmt.Sprintf("  Value:          %s\n", entry.FieldValue)
		}
		if entry.TransactionID > 0 {
			entryStr += fmt.Sprintf("  Transaction ID: %d\n", entry.TransactionID)
		}
		if entry.LineItemID > 0 {
			entryStr += fmt.Sprintf("  Line Item ID:   %d\n", entry.LineItemID)
		}

		entryStr += "\n"
		writer.WriteString(entryStr)
	}

	// Write footer.
	footer := "================================================================================\n" +
		"End of Error Log\n"
	writer.WriteString(footer)

	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("failed to flush error log: %w", err)
	}

	return logPath, nil
}

// =============================================================================
// PROCESSING SUMMARY
// =============================================================================

// ProcessingSummary contains summary information about a processing run.
type ProcessingSummary struct {
	StartTime        time.Time
	EndTime          time.Time
	TotalFiles       int
	SuccessfulFiles  int
	FailedFiles      int
	TotalRows        int
	TotalTransactions int
	TotalLineItems   int
	ValidationErrors int
	ProcessedFiles   []ProcessedFileInfo
	FailedFilesList  []FailedFileInfo
}

// ProcessedFileInfo contains information about a successfully processed file.
type ProcessedFileInfo struct {
	InputFile    string
	OutputFile   string
	ArchivePath  string
	Rows         int
	Transactions int
	LineItems    int
	ProcessTime  time.Duration
}

// FailedFileInfo contains information about a failed file.
type FailedFileInfo struct {
	InputFile    string
	ErrorMessage string
	ErrorType    string
}

// WriteSummaryLog writes a processing summary to a log file.
//
// PARAMETERS:
//   - summary: The processing summary.
//   - outputDir: The directory to write the summary file.
//
// RETURNS:
//   - The path to the summary file.
//   - An error if writing fails.
func WriteSummaryLog(summary ProcessingSummary, outputDir string) (string, error) {
	// Generate summary file name.
	timestamp := time.Now().Format("20060102_150405")
	summaryFileName := fmt.Sprintf("processing_summary_%s.txt", timestamp)
	summaryPath := filepath.Join(outputDir, summaryFileName)

	// Create the file.
	file, err := os.Create(summaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create summary file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write header.
	duration := summary.EndTime.Sub(summary.StartTime)
	header := fmt.Sprintf("CSV to XML Converter - Processing Summary\n"+
		"================================================================================\n\n"+
		"Run Information:\n"+
		"  Start Time:     %s\n"+
		"  End Time:       %s\n"+
		"  Duration:       %s\n\n"+
		"Statistics:\n"+
		"  Total Files:        %d\n"+
		"  Successful:         %d\n"+
		"  Failed:             %d\n"+
		"  Total Rows:         %d\n"+
		"  Total Transactions: %d\n"+
		"  Total Line Items:   %d\n"+
		"  Validation Errors:  %d\n\n",
		summary.StartTime.Format("2006-01-02 15:04:05"),
		summary.EndTime.Format("2006-01-02 15:04:05"),
		duration.String(),
		summary.TotalFiles,
		summary.SuccessfulFiles,
		summary.FailedFiles,
		summary.TotalRows,
		summary.TotalTransactions,
		summary.TotalLineItems,
		summary.ValidationErrors)
	writer.WriteString(header)

	// Write successful files.
	if len(summary.ProcessedFiles) > 0 {
		writer.WriteString("Successful Files:\n")
		writer.WriteString("--------------------------------------------------------------------------------\n")
		for _, pf := range summary.ProcessedFiles {
			writer.WriteString(fmt.Sprintf("  Input:        %s\n", pf.InputFile))
			writer.WriteString(fmt.Sprintf("  Output:       %s\n", pf.OutputFile))
			writer.WriteString(fmt.Sprintf("  Rows:         %d\n", pf.Rows))
			writer.WriteString(fmt.Sprintf("  Transactions: %d\n", pf.Transactions))
			writer.WriteString(fmt.Sprintf("  Process Time: %s\n\n", pf.ProcessTime.String()))
		}
	}

	// Write failed files.
	if len(summary.FailedFilesList) > 0 {
		writer.WriteString("Failed Files:\n")
		writer.WriteString("--------------------------------------------------------------------------------\n")
		for _, ff := range summary.FailedFilesList {
			writer.WriteString(fmt.Sprintf("  File:  %s\n", ff.InputFile))
			writer.WriteString(fmt.Sprintf("  Error: %s\n\n", ff.ErrorMessage))
		}
	}

	// Write footer.
	footer := "================================================================================\n" +
		"End of Summary\n"
	writer.WriteString(footer)

	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("failed to flush summary file: %w", err)
	}

	return summaryPath, nil
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetFileSize returns the size of a file in bytes.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileModTime returns the modification time of a file.
func GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// CleanOldArchives removes archive files older than the specified duration.
//
// PARAMETERS:
//   - archiveDir: The archive directory to clean.
//   - maxAge: The maximum age of files to keep.
//
// RETURNS:
//   - The number of files removed.
//   - An error if cleaning fails.
//
// CUSTOMIZATION:
//   Implement a retention policy for your archives.
func CleanOldArchives(archiveDir string, maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	removed := 0

	err := filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				return err
			}
			removed++
		}

		return nil
	})

	if err != nil {
		return removed, fmt.Errorf("failed to clean archives: %w", err)
	}

	return removed, nil
}
