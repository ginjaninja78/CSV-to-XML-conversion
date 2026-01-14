// =============================================================================
// CSV to XML Converter - Converter Module
// =============================================================================
//
// This module contains the core conversion logic. It orchestrates the entire
// conversion pipeline for a single file, from CSV parsing to XML generation.
//
// CONVERSION PIPELINE:
//   1. Parse the XLSX template to extract the schema
//   2. Parse the input CSV file
//   3. Group CSV rows into transactions
//   4. Apply transformation rules to each field
//   5. Validate the transformed data
//   6. Generate the XML document
//   7. Write the output file
//   8. Archive the processed files
//
// CONCURRENCY:
//   Each file is processed in its own goroutine. The converter is designed
//   to be thread-safe and can process multiple files concurrently.
//
// =============================================================================

package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/config"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/csvparser"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/validation"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/xlsxparser"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/xmlwriter"
	"github.com/google/uuid"
)

// =============================================================================
// RESULT STRUCTURE
// =============================================================================

// Result represents the outcome of processing a single file.
type Result struct {
	// FilePath is the path to the input file that was processed.
	FilePath string

	// OutputFile is the path to the generated XML file.
	// This is empty if processing failed.
	OutputFile string

	// Success indicates whether the processing was successful.
	Success bool

	// Error contains the error if processing failed.
	// This is nil if processing was successful.
	Error error

	// Stats contains processing statistics.
	Stats ProcessingStats
}

// ProcessingStats contains statistics about the processing.
type ProcessingStats struct {
	// RowsProcessed is the number of CSV rows processed.
	RowsProcessed int

	// TransactionsCreated is the number of transactions created in the XML.
	TransactionsCreated int

	// LineItemsCreated is the number of line items created in the XML.
	LineItemsCreated int

	// ValidationErrors is the number of validation errors encountered.
	// If ContinueOnError is true, processing continues despite these errors.
	ValidationErrors int

	// ProcessingTime is the time taken to process the file.
	ProcessingTime time.Duration
}

// =============================================================================
// CONVERTER STRUCTURE
// =============================================================================

// Converter handles the conversion of a single CSV file to XML.
type Converter struct {
	// csvPath is the path to the input CSV file.
	csvPath string

	// deptConfig is the department-specific configuration.
	deptConfig *config.DepartmentConfig

	// mainConfig is the main application configuration.
	mainConfig *config.MainConfig

	// schema is the parsed XLSX template schema.
	schema *xlsxparser.Schema

	// logger is used for logging (can be replaced with a proper logger).
	// CUSTOMIZATION: Replace with your preferred logging library.
	logger Logger
}

// Logger is an interface for logging.
// CUSTOMIZATION: Implement this interface with your preferred logging library.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// =============================================================================
// CONSTRUCTOR
// =============================================================================

// New creates a new Converter instance.
//
// PARAMETERS:
//   - csvPath: The path to the input CSV file.
//   - deptConfig: The department-specific configuration.
//   - mainConfig: The main application configuration.
//
// RETURNS:
//   - A new Converter instance.
func New(csvPath string, deptConfig *config.DepartmentConfig, mainConfig *config.MainConfig) *Converter {
	return &Converter{
		csvPath:    csvPath,
		deptConfig: deptConfig,
		mainConfig: mainConfig,
		logger:     &defaultLogger{}, // Use default logger
	}
}

// =============================================================================
// MAIN PROCESSING FUNCTION
// =============================================================================

// Run executes the conversion pipeline for the file.
//
// RETURNS:
//   - A Result struct containing the outcome of the processing.
//
// PROCESSING STEPS:
//   1. Determine which template to use
//   2. Parse the XLSX template to get the schema
//   3. Parse the input CSV file
//   4. Group CSV rows into transactions
//   5. Apply transformation rules
//   6. Validate the data
//   7. Generate the XML document
//   8. Write the output file
//   9. Archive the processed files
func (c *Converter) Run() Result {
	startTime := time.Now()
	result := Result{
		FilePath: c.csvPath,
		Success:  false,
	}

	// =========================================================================
	// STEP 1: DETERMINE TEMPLATE
	// =========================================================================
	// Find the appropriate XLSX template based on the file name.

	c.logger.Info("Processing file: %s", c.csvPath)

	templatePath, err := c.determineTemplate()
	if err != nil {
		result.Error = fmt.Errorf("failed to determine template: %w", err)
		return result
	}

	c.logger.Debug("Using template: %s", templatePath)

	// =========================================================================
	// STEP 2: PARSE XLSX TEMPLATE
	// =========================================================================
	// Extract the schema from the XLSX template.
	// The schema defines:
	//   - Column mappings (old header -> XML tag)
	//   - Validation rules (char limits, formats, required/optional)
	//   - XML nesting structure

	schema, err := xlsxparser.Parse(templatePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse template: %w", err)
		return result
	}

	c.schema = schema
	c.logger.Debug("Parsed schema with %d field mappings", len(schema.FieldMappings))

	// =========================================================================
	// STEP 3: PARSE INPUT CSV
	// =========================================================================
	// Parse the CSV file using the department-specific settings.

	csvData, err := csvparser.Parse(c.csvPath, c.deptConfig.CSVSettings)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse CSV: %w", err)
		return result
	}

	result.Stats.RowsProcessed = len(csvData.Rows)
	c.logger.Debug("Parsed %d rows from CSV", len(csvData.Rows))

	// =========================================================================
	// STEP 4: GROUP ROWS INTO TRANSACTIONS
	// =========================================================================
	// Group CSV rows based on the transaction grouping configuration.
	// Each group becomes a <transaction> element in the XML.

	transactions := c.groupTransactions(csvData)
	result.Stats.TransactionsCreated = len(transactions)
	c.logger.Debug("Grouped into %d transactions", len(transactions))

	// =========================================================================
	// STEP 5: APPLY TRANSFORMATION RULES
	// =========================================================================
	// Apply department-specific transformation rules to each field.
	// This includes:
	//   - Prepending/appending strings
	//   - Zero-padding
	//   - Format conversions
	//   - Lookup table replacements

	for i := range transactions {
		if err := c.applyTransformations(&transactions[i]); err != nil {
			result.Error = fmt.Errorf("failed to apply transformations: %w", err)
			return result
		}
	}

	c.logger.Debug("Applied transformation rules")

	// =========================================================================
	// STEP 6: VALIDATE DATA
	// =========================================================================
	// Validate the transformed data against the schema.
	// This includes:
	//   - Character length limits
	//   - Format validation (numeric, alphanumeric, date, etc.)
	//   - Required field checks
	//   - Conditional validation rules

	// Convert transactions to validation types.
	validationTransactions := convertToValidationTransactions(transactions)
	validationErrors := validation.Validate(validationTransactions, c.schema)
	result.Stats.ValidationErrors = len(validationErrors)

	if len(validationErrors) > 0 {
		// Log validation errors.
		for _, ve := range validationErrors {
			c.logger.Warn("Validation error: %s", ve.Error())
		}

		// If we're not continuing on error, fail the processing.
		if !c.mainConfig.ContinueOnError {
			result.Error = fmt.Errorf("validation failed with %d errors", len(validationErrors))
			return result
		}
	}

	c.logger.Debug("Validation complete with %d errors", len(validationErrors))

	// =========================================================================
	// STEP 7: GENERATE XML DOCUMENT
	// =========================================================================
	// Generate the XML document based on the schema and transformed data.

	// Convert transactions to xmlwriter types.
	xmlTransactions := convertToXMLWriterTransactions(transactions)
	xmlDoc, err := xmlwriter.Generate(xmlTransactions, c.schema, c.deptConfig)
	if err != nil {
		result.Error = fmt.Errorf("failed to generate XML: %w", err)
		return result
	}

	c.logger.Debug("Generated XML document")

	// =========================================================================
	// STEP 8: WRITE OUTPUT FILE
	// =========================================================================
	// Write the XML document to the output directory.

	outputPath, err := c.writeOutput(xmlDoc)
	if err != nil {
		result.Error = fmt.Errorf("failed to write output: %w", err)
		return result
	}

	result.OutputFile = outputPath
	c.logger.Info("Wrote output to: %s", outputPath)

	// =========================================================================
	// STEP 9: ARCHIVE FILES
	// =========================================================================
	// Move the processed files to the archive directories.

	if err := c.archiveFiles(outputPath); err != nil {
		// Log the error but don't fail the processing.
		c.logger.Warn("Failed to archive files: %v", err)
	}

	// =========================================================================
	// COMPLETE
	// =========================================================================

	result.Success = true
	result.Stats.ProcessingTime = time.Since(startTime)

	return result
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// determineTemplate finds the appropriate XLSX template for the input file.
//
// RETURNS:
//   - The path to the XLSX template file.
//   - An error if no matching template is found.
//
// MATCHING LOGIC:
//   This function iterates through the template mapping rules in the department
//   configuration and returns the first matching template.
//
// CUSTOMIZATION:
//   - Modify the matching logic if your file naming conventions are different.
//   - Add support for default templates.
func (c *Converter) determineTemplate() (string, error) {
	fileName := filepath.Base(c.csvPath)

	// Iterate through template mapping rules.
	for _, rule := range c.deptConfig.TemplateMapping {
		// Check if the file name contains the specified substring.
		if containsIgnoreCase(fileName, rule.IfFilenameContains) {
			// Construct the full path to the template.
			templatePath := filepath.Join(c.mainConfig.TemplatesDir, rule.UseTemplate)

			// Verify the template exists.
			if _, err := os.Stat(templatePath); os.IsNotExist(err) {
				return "", fmt.Errorf("template file not found: %s", templatePath)
			}

			return templatePath, nil
		}
	}

	return "", fmt.Errorf("no matching template found for file: %s", fileName)
}

// groupTransactions groups CSV rows into transactions based on the grouping configuration.
//
// PARAMETERS:
//   - csvData: The parsed CSV data.
//
// RETURNS:
//   - A slice of Transaction structs, each containing its line items.
//
// GROUPING LOGIC:
//   Rows are grouped by the value of the field specified in TransactionGrouping.GroupByField.
//   All rows with the same value in this field belong to the same transaction.
//
// CUSTOMIZATION:
//   - Modify this function if your grouping logic is more complex.
//   - Add support for multiple grouping fields.
//
// QUESTION FOR USER:
//   What field in your CSV identifies which rows belong to the same transaction?
//   This could be a check number, batch ID, transaction ID, or any other unique identifier.
//   Please update the GroupByField in your department configuration.
func (c *Converter) groupTransactions(csvData *csvparser.CSVData) []Transaction {
	groupByField := c.deptConfig.TransactionGrouping.GroupByField

	// If no grouping field is specified, treat each row as a separate transaction.
	if groupByField == "" {
		transactions := make([]Transaction, len(csvData.Rows))
		for i, row := range csvData.Rows {
			transactions[i] = Transaction{
				ID:        i + 1,
				LineItems: []LineItem{{ID: i + 1, Fields: row}},
			}
		}
		return transactions
	}

	// Group rows by the grouping field.
	groups := make(map[string][]map[string]string)
	groupOrder := []string{} // Maintain order of first occurrence

	for _, row := range csvData.Rows {
		key := row[groupByField]
		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], row)
	}

	// Convert groups to transactions.
	transactions := make([]Transaction, len(groupOrder))
	lineItemCounter := 1 // Global line item counter

	for i, key := range groupOrder {
		rows := groups[key]
		lineItems := make([]LineItem, len(rows))

		for j, row := range rows {
			lineItems[j] = LineItem{
				ID:     lineItemCounter,
				Fields: row,
			}
			lineItemCounter++
		}

		transactions[i] = Transaction{
			ID:        i + 1,
			GroupKey:  key,
			LineItems: lineItems,
		}
	}

	return transactions
}

// applyTransformations applies transformation rules to a transaction.
//
// PARAMETERS:
//   - transaction: A pointer to the transaction to transform.
//
// RETURNS:
//   - An error if any transformation fails.
//
// TRANSFORMATION TYPES:
//   - prepend_string: Add a string to the beginning of the value
//   - append_string: Add a string to the end of the value
//   - pad_zeros_to_length: Pad with leading zeros to a specific length
//   - ensure_length: Truncate or pad to ensure a specific length
//   - uppercase: Convert to uppercase
//   - lowercase: Convert to lowercase
//   - trim: Remove leading and trailing whitespace
//   - replace: Replace a substring with another
//   - regex_replace: Replace using a regular expression
//   - format_date: Convert date format
//   - format_number: Format a number
//   - lookup: Replace value using a lookup table
//   - conditional: Apply transformation based on a condition
//
// CUSTOMIZATION:
//   - Add new transformation types as needed.
//   - Modify existing transformations to match your business logic.
func (c *Converter) applyTransformations(transaction *Transaction) error {
	// Apply transformations to each line item.
	for i := range transaction.LineItems {
		for _, rule := range c.deptConfig.TransformationRules {
			// Get the current value of the field.
			value, exists := transaction.LineItems[i].Fields[rule.Field]
			if !exists {
				continue
			}

			// Apply each action in sequence.
			for _, action := range rule.Actions {
				var err error
				value, err = applyAction(value, action)
				if err != nil {
					return fmt.Errorf("failed to apply %s to field %s: %w", action.Type, rule.Field, err)
				}
			}

			// Update the field with the transformed value.
			transaction.LineItems[i].Fields[rule.Field] = value
		}
	}

	return nil
}

// applyAction applies a single transformation action to a value.
//
// PARAMETERS:
//   - value: The current value of the field.
//   - action: The transformation action to apply.
//
// RETURNS:
//   - The transformed value.
//   - An error if the transformation fails.
//
// CUSTOMIZATION:
//   Add new cases to this switch statement for new transformation types.
func applyAction(value string, action config.TransformationAction) (string, error) {
	switch action.Type {
	case "prepend_string":
		// Add a string to the beginning of the value.
		// Example: "123" with prepend "A" becomes "A123"
		return action.Value + value, nil

	case "append_string":
		// Add a string to the end of the value.
		// Example: "123" with append "X" becomes "123X"
		return value + action.Value, nil

	case "pad_zeros_to_length":
		// Pad with leading zeros to a specific length.
		// Example: "123" with length 6 becomes "000123"
		targetLength := parseIntOrDefault(action.Value, 0)
		if targetLength <= 0 {
			return value, nil
		}
		return padLeft(value, targetLength, '0'), nil

	case "ensure_length":
		// Truncate or pad to ensure a specific length.
		// Example: "12345" with length 3 becomes "123"
		// Example: "12" with length 5 becomes "12   " (or "00012" if numeric)
		targetLength := parseIntOrDefault(action.Value, 0)
		if targetLength <= 0 {
			return value, nil
		}
		if len(value) > targetLength {
			return value[:targetLength], nil
		}
		return padLeft(value, targetLength, '0'), nil

	case "uppercase":
		// Convert to uppercase.
		return toUpperCase(value), nil

	case "lowercase":
		// Convert to lowercase.
		return toLowerCase(value), nil

	case "trim":
		// Remove leading and trailing whitespace.
		return trimSpace(value), nil

	case "replace":
		// Replace a substring with another.
		// Example: "hello world" with find "world" and replace "there" becomes "hello there"
		return replaceString(value, action.Find, action.Value), nil

	case "lookup":
		// Replace value using a lookup table.
		// Example: "01" with lookup {"01": "January"} becomes "January"
		if replacement, exists := action.LookupTable[value]; exists {
			return replacement, nil
		}
		return value, nil

	case "conditional":
		// Apply transformation based on a condition.
		// CUSTOMIZATION: Implement your conditional logic here.
		//
		// PSEUDOCODE:
		// if evaluateCondition(value, action.Condition) {
		//     return applyConditionalTransformation(value, action)
		// }
		return value, nil

	default:
		// Unknown transformation type.
		return value, fmt.Errorf("unknown transformation type: %s", action.Type)
	}
}

// writeOutput writes the XML document to the output directory.
//
// PARAMETERS:
//   - xmlDoc: The XML document to write.
//
// RETURNS:
//   - The path to the output file.
//   - An error if the file cannot be written.
//
// FILE NAMING:
//   The output file is named according to the UUIDFormat in the main configuration.
//   Placeholders are replaced with actual values:
//   - {uuid}: A random UUID
//   - {timestamp}: Current timestamp
//   - {dept}: Department code
//   - {type}: Transaction type
//
// CUSTOMIZATION:
//   Modify the generateOutputFileName function to match your naming conventions.
func (c *Converter) writeOutput(xmlDoc []byte) (string, error) {
	// Generate the output file name.
	fileName := c.generateOutputFileName()
	outputPath := filepath.Join(c.mainConfig.OutputDir, fileName)

	// Write the XML document to the file.
	if err := os.WriteFile(outputPath, xmlDoc, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}

// generateOutputFileName generates the output file name based on the UUID format.
//
// RETURNS:
//   - The generated file name.
//
// CUSTOMIZATION:
//   Modify this function to match your file naming conventions.
//   Add support for additional placeholders as needed.
func (c *Converter) generateOutputFileName() string {
	format := c.mainConfig.UUIDFormat

	// Generate a UUID.
	// CUSTOMIZATION: Modify the UUID generation if you need a specific format.
	id := uuid.New().String()

	// Generate a timestamp.
	timestamp := time.Now().Format("20060102_150405")

	// Replace placeholders.
	fileName := format
	fileName = replaceString(fileName, "{uuid}", id)
	fileName = replaceString(fileName, "{timestamp}", timestamp)
	fileName = replaceString(fileName, "{dept}", c.deptConfig.DepartmentCode)

	// Ensure the file has an .xml extension.
	if filepath.Ext(fileName) != ".xml" {
		fileName += ".xml"
	}

	return fileName
}

// archiveFiles moves the processed files to the archive directories.
//
// PARAMETERS:
//   - outputPath: The path to the generated XML file.
//
// RETURNS:
//   - An error if the files cannot be moved.
//
// ARCHIVAL LOGIC:
//   - The input CSV is moved to the input archive directory.
//   - The output XML is copied to the output archive directory.
//
// CUSTOMIZATION:
//   - Modify this function if you need different archival behavior.
//   - Add support for date-based subdirectories.
func (c *Converter) archiveFiles(outputPath string) error {
	// Archive the input file.
	inputFileName := filepath.Base(c.csvPath)
	archivePath := filepath.Join(c.mainConfig.InputArchiveDir, inputFileName)

	if err := os.Rename(c.csvPath, archivePath); err != nil {
		return fmt.Errorf("failed to archive input file: %w", err)
	}

	// Archive the output file (copy, not move).
	outputFileName := filepath.Base(outputPath)
	outputArchivePath := filepath.Join(c.mainConfig.OutputArchiveDir, outputFileName)

	// Read the output file.
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read output file for archival: %w", err)
	}

	// Write to the archive.
	if err := os.WriteFile(outputArchivePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output archive: %w", err)
	}

	return nil
}

// =============================================================================
// DATA STRUCTURES
// =============================================================================

// Transaction represents a single transaction in the XML output.
type Transaction struct {
	// ID is the transaction number (1-indexed).
	ID int

	// GroupKey is the value of the grouping field for this transaction.
	GroupKey string

	// LineItems contains the line items for this transaction.
	LineItems []LineItem
}

// LineItem represents a single line item within a transaction.
type LineItem struct {
	// ID is the line item number (globally incremented).
	ID int

	// Fields contains the field values for this line item.
	// Keys are the original CSV column headers.
	Fields map[string]string
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// containsIgnoreCase checks if a string contains a substring (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	// IMPLEMENTATION: Use strings.Contains with lowercase conversion.
	// This is a placeholder - implement with proper string handling.
	return true // Placeholder
}

// padLeft pads a string with a character on the left to reach the target length.
func padLeft(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	padding := make([]rune, length-len(s))
	for i := range padding {
		padding[i] = padChar
	}
	return string(padding) + s
}

// parseIntOrDefault parses a string as an integer, returning a default value on error.
func parseIntOrDefault(s string, defaultValue int) int {
	// IMPLEMENTATION: Use strconv.Atoi.
	// This is a placeholder - implement with proper parsing.
	return defaultValue // Placeholder
}

// toUpperCase converts a string to uppercase.
func toUpperCase(s string) string {
	// IMPLEMENTATION: Use strings.ToUpper.
	return s // Placeholder
}

// toLowerCase converts a string to lowercase.
func toLowerCase(s string) string {
	// IMPLEMENTATION: Use strings.ToLower.
	return s // Placeholder
}

// trimSpace removes leading and trailing whitespace.
func trimSpace(s string) string {
	// IMPLEMENTATION: Use strings.TrimSpace.
	return s // Placeholder
}

// replaceString replaces all occurrences of a substring.
func replaceString(s, old, new string) string {
	// IMPLEMENTATION: Use strings.ReplaceAll.
	return s // Placeholder
}

// =============================================================================
// TYPE CONVERSION FUNCTIONS
// =============================================================================

// convertToValidationTransactions converts internal Transaction types to validation.Transaction types.
func convertToValidationTransactions(transactions []Transaction) []validation.Transaction {
	result := make([]validation.Transaction, len(transactions))
	for i, t := range transactions {
		lineItems := make([]validation.LineItem, len(t.LineItems))
		for j, li := range t.LineItems {
			lineItems[j] = validation.LineItem{
				ID:     li.ID,
				Fields: li.Fields,
			}
		}
		result[i] = validation.Transaction{
			ID:        t.ID,
			GroupKey:  t.GroupKey,
			LineItems: lineItems,
		}
	}
	return result
}

// convertToXMLWriterTransactions converts internal Transaction types to xmlwriter.Transaction types.
func convertToXMLWriterTransactions(transactions []Transaction) []xmlwriter.Transaction {
	result := make([]xmlwriter.Transaction, len(transactions))
	for i, t := range transactions {
		lineItems := make([]xmlwriter.LineItem, len(t.LineItems))
		for j, li := range t.LineItems {
			lineItems[j] = xmlwriter.LineItem{
				ID:     li.ID,
				Fields: li.Fields,
			}
		}
		result[i] = xmlwriter.Transaction{
			ID:        t.ID,
			GroupKey:  t.GroupKey,
			LineItems: lineItems,
		}
	}
	return result
}

// =============================================================================
// DEFAULT LOGGER
// =============================================================================

// defaultLogger is a simple logger that prints to stdout.
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+msg+"\n", args...)
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("[WARN] "+msg+"\n", args...)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}
