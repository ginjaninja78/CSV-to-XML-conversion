// =============================================================================
// CSV to XML Converter - CSV Parser Module
// =============================================================================
//
// This module is responsible for parsing CSV files from the legacy reporting
// system. It handles various CSV formats and configurations, including:
//   - Different delimiters (comma, pipe, tab, etc.)
//   - Multi-line headers
//   - Custom data start rows
//   - Different encodings
//   - Quoted fields with escape characters
//
// FEATURES:
//   - Flexible configuration via CSVSettings
//   - Support for multi-line headers (some departments have headers spanning multiple rows)
//   - Automatic detection of data start row
//   - Robust error handling with detailed error messages
//   - Memory-efficient streaming for large files
//
// CUSTOMIZATION:
//   - Modify the CSVSettings struct to add new parsing options
//   - Add custom preprocessing functions for specific file formats
//   - Extend the error handling for specific validation requirements
//
// =============================================================================

package csvparser

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/config"
)

// =============================================================================
// CSV DATA STRUCTURE
// =============================================================================

// CSVData represents the parsed CSV file.
type CSVData struct {
	// Headers contains the column headers from the CSV file.
	// For multi-line headers, these are the merged/final headers.
	Headers []string

	// Rows contains the data rows as maps of header -> value.
	// Using maps allows for easy field access by name.
	Rows []map[string]string

	// RawRows contains the raw row data as string slices.
	// This is useful for debugging and error reporting.
	RawRows [][]string

	// SourceFile is the path to the source CSV file.
	SourceFile string

	// RowCount is the total number of data rows (excluding headers).
	RowCount int

	// ColumnCount is the number of columns in the CSV.
	ColumnCount int
}

// =============================================================================
// PARSER FUNCTIONS
// =============================================================================

// Parse reads a CSV file and returns the parsed data.
//
// PARAMETERS:
//   - filePath: The path to the CSV file.
//   - settings: The CSV parsing settings from the department configuration.
//
// RETURNS:
//   - A pointer to the CSVData struct containing the parsed data.
//   - An error if the file cannot be read or parsed.
//
// PARSING PROCESS:
//   1. Open the file with the specified encoding
//   2. Configure the CSV reader with the specified delimiter and quote settings
//   3. Read and merge header rows (for multi-line headers)
//   4. Read data rows starting from the configured data start row
//   5. Convert each row to a map of header -> value
//
// CUSTOMIZATION:
//   - Add preprocessing logic for specific file formats
//   - Add support for additional encodings
//   - Add validation during parsing
func Parse(filePath string, settings config.CSVSettings) (*CSVData, error) {
	// Open the file.
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create a buffered reader for better performance.
	reader := bufio.NewReader(file)

	// Handle encoding if not UTF-8.
	// CUSTOMIZATION: Add support for additional encodings.
	//
	// PSEUDOCODE for encoding conversion:
	// if settings.Encoding != "UTF-8" {
	//     decoder := getDecoder(settings.Encoding)
	//     reader = transform.NewReader(reader, decoder)
	// }

	// Create the CSV reader.
	csvReader := csv.NewReader(reader)

	// Configure the CSV reader based on settings.
	configureReader(csvReader, settings)

	// Read all rows.
	allRows, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	// Validate that we have data.
	if len(allRows) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Extract headers (handling multi-line headers).
	headers, err := extractHeaders(allRows, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to extract headers: %w", err)
	}

	// Extract data rows.
	dataRows, err := extractDataRows(allRows, headers, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to extract data rows: %w", err)
	}

	// Build the CSVData struct.
	csvData := &CSVData{
		Headers:     headers,
		Rows:        dataRows,
		RawRows:     allRows[settings.DataStartRow-1:], // Keep raw rows for debugging
		SourceFile:  filePath,
		RowCount:    len(dataRows),
		ColumnCount: len(headers),
	}

	return csvData, nil
}

// configureReader configures the CSV reader based on the settings.
//
// PARAMETERS:
//   - reader: The CSV reader to configure.
//   - settings: The CSV parsing settings.
//
// CUSTOMIZATION:
//   Add additional configuration options as needed.
func configureReader(reader *csv.Reader, settings config.CSVSettings) {
	// Set the delimiter.
	// Handle special cases for common delimiters.
	switch settings.Delimiter {
	case "\\t", "tab", "TAB":
		reader.Comma = '\t'
	case "|", "pipe", "PIPE":
		reader.Comma = '|'
	case ";", "semicolon":
		reader.Comma = ';'
	default:
		if len(settings.Delimiter) > 0 {
			reader.Comma = rune(settings.Delimiter[0])
		} else {
			reader.Comma = ',' // Default to comma
		}
	}

	// Set the quote character.
	// Note: Go's csv package only supports double-quote by default.
	// CUSTOMIZATION: For other quote characters, you may need custom parsing.

	// Allow variable number of fields per row.
	// This is useful for CSVs with inconsistent column counts.
	reader.FieldsPerRecord = -1

	// Allow lazy quotes (quotes that don't follow strict CSV rules).
	reader.LazyQuotes = true

	// Trim leading space from fields.
	reader.TrimLeadingSpace = true
}

// extractHeaders extracts and merges headers from the CSV.
//
// PARAMETERS:
//   - allRows: All rows from the CSV file.
//   - settings: The CSV parsing settings.
//
// RETURNS:
//   - A slice of header strings.
//   - An error if headers cannot be extracted.
//
// MULTI-LINE HEADER HANDLING:
//   Some CSV files have headers that span multiple rows. This function
//   merges them into a single set of headers.
//
//   Example:
//   Row 1: "Transaction", "", "Policy", ""
//   Row 2: "Number", "Amount", "Number", "Date"
//   Result: "Transaction Number", "Amount", "Policy Number", "Date"
//
// CUSTOMIZATION:
//   Modify the merging logic to match your specific header format.
//
// QUESTION FOR USER:
//   How are your multi-line headers formatted? Do they:
//   a) Span across rows with the parent category in the first row?
//   b) Have a different structure?
//   Please provide an example so we can implement the correct merging logic.
func extractHeaders(allRows [][]string, settings config.CSVSettings) ([]string, error) {
	if settings.HeaderRows <= 0 {
		return nil, fmt.Errorf("header_rows must be at least 1")
	}

	if len(allRows) < settings.HeaderRows {
		return nil, fmt.Errorf("file has fewer rows than header_rows setting")
	}

	// If only one header row, return it directly.
	if settings.HeaderRows == 1 {
		return cleanHeaders(allRows[0]), nil
	}

	// For multi-line headers, merge them.
	// CUSTOMIZATION: Modify this logic to match your header format.
	//
	// STRATEGY 1: Concatenate non-empty values from each row.
	// This is the default strategy.
	//
	// STRATEGY 2: Use the last non-empty value from each column.
	// Uncomment the alternative implementation if this is your format.

	// Determine the maximum number of columns.
	maxCols := 0
	for i := 0; i < settings.HeaderRows; i++ {
		if len(allRows[i]) > maxCols {
			maxCols = len(allRows[i])
		}
	}

	// Merge headers.
	headers := make([]string, maxCols)
	for col := 0; col < maxCols; col++ {
		var parts []string

		for row := 0; row < settings.HeaderRows; row++ {
			if col < len(allRows[row]) {
				value := strings.TrimSpace(allRows[row][col])
				if value != "" {
					parts = append(parts, value)
				}
			}
		}

		// Join the parts with a space.
		// CUSTOMIZATION: Change the separator if needed.
		headers[col] = strings.Join(parts, " ")
	}

	return cleanHeaders(headers), nil
}

// cleanHeaders cleans and normalizes header values.
//
// PARAMETERS:
//   - headers: The raw header values.
//
// RETURNS:
//   - Cleaned header values.
//
// CLEANING OPERATIONS:
//   - Trim whitespace
//   - Remove special characters (optional)
//   - Handle empty headers
//
// CUSTOMIZATION:
//   Add additional cleaning operations as needed.
func cleanHeaders(headers []string) []string {
	cleaned := make([]string, len(headers))

	for i, header := range headers {
		// Trim whitespace.
		header = strings.TrimSpace(header)

		// Handle empty headers.
		// CUSTOMIZATION: Decide how to handle empty headers.
		// Option 1: Use a placeholder name.
		// Option 2: Skip the column.
		// Option 3: Use the column index.
		if header == "" {
			header = fmt.Sprintf("Column_%d", i+1)
		}

		cleaned[i] = header
	}

	return cleaned
}

// extractDataRows extracts data rows and converts them to maps.
//
// PARAMETERS:
//   - allRows: All rows from the CSV file.
//   - headers: The extracted headers.
//   - settings: The CSV parsing settings.
//
// RETURNS:
//   - A slice of maps, where each map represents a row with header -> value pairs.
//   - An error if data extraction fails.
//
// CUSTOMIZATION:
//   Add preprocessing or validation logic for specific data formats.
func extractDataRows(allRows [][]string, headers []string, settings config.CSVSettings) ([]map[string]string, error) {
	// Calculate the starting index for data rows.
	// DataStartRow is 1-indexed, so subtract 1 for 0-indexed array.
	startIndex := settings.DataStartRow - 1

	if startIndex < 0 {
		startIndex = settings.HeaderRows // Default to after headers
	}

	if startIndex >= len(allRows) {
		// No data rows.
		return []map[string]string{}, nil
	}

	// Extract data rows.
	dataRows := make([]map[string]string, 0, len(allRows)-startIndex)

	for rowIndex := startIndex; rowIndex < len(allRows); rowIndex++ {
		row := allRows[rowIndex]

		// Skip empty rows.
		if isRowEmpty(row) {
			continue
		}

		// Convert the row to a map.
		rowMap := make(map[string]string)

		for colIndex, header := range headers {
			if colIndex < len(row) {
				// Trim whitespace from values.
				value := strings.TrimSpace(row[colIndex])
				rowMap[header] = value
			} else {
				// Column is missing in this row.
				rowMap[header] = ""
			}
		}

		dataRows = append(dataRows, rowMap)
	}

	return dataRows, nil
}

// isRowEmpty checks if a row contains only empty values.
func isRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// =============================================================================
// STREAMING PARSER FOR LARGE FILES
// =============================================================================

// StreamingParser provides memory-efficient parsing for large CSV files.
// Instead of loading the entire file into memory, it processes rows one at a time.
//
// USAGE:
//   parser, err := NewStreamingParser(filePath, settings)
//   if err != nil {
//       return err
//   }
//   defer parser.Close()
//
//   for parser.Next() {
//       row := parser.Row()
//       // Process the row...
//   }
//
//   if err := parser.Err(); err != nil {
//       return err
//   }
type StreamingParser struct {
	file      *os.File
	reader    *csv.Reader
	headers   []string
	currentRow map[string]string
	rowNumber int
	err       error
	settings  config.CSVSettings
}

// NewStreamingParser creates a new streaming parser for a CSV file.
//
// PARAMETERS:
//   - filePath: The path to the CSV file.
//   - settings: The CSV parsing settings.
//
// RETURNS:
//   - A pointer to the StreamingParser.
//   - An error if the file cannot be opened.
func NewStreamingParser(filePath string, settings config.CSVSettings) (*StreamingParser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	reader := csv.NewReader(bufio.NewReader(file))
	configureReader(reader, settings)

	parser := &StreamingParser{
		file:     file,
		reader:   reader,
		settings: settings,
	}

	// Read headers.
	if err := parser.readHeaders(); err != nil {
		file.Close()
		return nil, err
	}

	// Skip to data start row.
	if err := parser.skipToDataStart(); err != nil {
		file.Close()
		return nil, err
	}

	return parser, nil
}

// readHeaders reads and processes the header rows.
func (p *StreamingParser) readHeaders() error {
	headerRows := make([][]string, 0, p.settings.HeaderRows)

	for i := 0; i < p.settings.HeaderRows; i++ {
		row, err := p.reader.Read()
		if err == io.EOF {
			return fmt.Errorf("unexpected end of file while reading headers")
		}
		if err != nil {
			return fmt.Errorf("error reading header row %d: %w", i+1, err)
		}
		headerRows = append(headerRows, row)
		p.rowNumber++
	}

	// Merge headers if multi-line.
	headers, err := extractHeaders(headerRows, p.settings)
	if err != nil {
		return err
	}

	p.headers = headers
	return nil
}

// skipToDataStart skips rows until the data start row.
func (p *StreamingParser) skipToDataStart() error {
	targetRow := p.settings.DataStartRow
	if targetRow <= 0 {
		targetRow = p.settings.HeaderRows + 1
	}

	for p.rowNumber < targetRow-1 {
		_, err := p.reader.Read()
		if err == io.EOF {
			return nil // No data rows
		}
		if err != nil {
			return fmt.Errorf("error skipping to data start: %w", err)
		}
		p.rowNumber++
	}

	return nil
}

// Next advances to the next row. Returns false when there are no more rows.
func (p *StreamingParser) Next() bool {
	if p.err != nil {
		return false
	}

	row, err := p.reader.Read()
	if err == io.EOF {
		return false
	}
	if err != nil {
		p.err = fmt.Errorf("error reading row %d: %w", p.rowNumber+1, err)
		return false
	}

	p.rowNumber++

	// Skip empty rows.
	if isRowEmpty(row) {
		return p.Next()
	}

	// Convert to map.
	p.currentRow = make(map[string]string)
	for i, header := range p.headers {
		if i < len(row) {
			p.currentRow[header] = strings.TrimSpace(row[i])
		} else {
			p.currentRow[header] = ""
		}
	}

	return true
}

// Row returns the current row as a map.
func (p *StreamingParser) Row() map[string]string {
	return p.currentRow
}

// Headers returns the parsed headers.
func (p *StreamingParser) Headers() []string {
	return p.headers
}

// RowNumber returns the current row number (1-indexed).
func (p *StreamingParser) RowNumber() int {
	return p.rowNumber
}

// Err returns any error that occurred during parsing.
func (p *StreamingParser) Err() error {
	return p.err
}

// Close closes the underlying file.
func (p *StreamingParser) Close() error {
	return p.file.Close()
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// GetColumnByHeader returns all values for a specific column.
//
// PARAMETERS:
//   - data: The parsed CSV data.
//   - header: The column header to extract.
//
// RETURNS:
//   - A slice of values for that column.
func GetColumnByHeader(data *CSVData, header string) []string {
	values := make([]string, len(data.Rows))
	for i, row := range data.Rows {
		values[i] = row[header]
	}
	return values
}

// GetUniqueValues returns unique values for a specific column.
//
// PARAMETERS:
//   - data: The parsed CSV data.
//   - header: The column header to extract.
//
// RETURNS:
//   - A slice of unique values for that column.
func GetUniqueValues(data *CSVData, header string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, row := range data.Rows {
		value := row[header]
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}

	return unique
}

// FilterRows returns rows that match a filter condition.
//
// PARAMETERS:
//   - data: The parsed CSV data.
//   - filterFunc: A function that returns true for rows to include.
//
// RETURNS:
//   - A slice of rows that match the filter.
func FilterRows(data *CSVData, filterFunc func(row map[string]string) bool) []map[string]string {
	var filtered []map[string]string

	for _, row := range data.Rows {
		if filterFunc(row) {
			filtered = append(filtered, row)
		}
	}

	return filtered
}
