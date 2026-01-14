// =============================================================================
// CSV to XML Converter - XLSX Template Parser
// =============================================================================
//
// This module is responsible for parsing XLSX template files that define the
// schema for CSV-to-XML conversion. The templates contain:
//   - Column mappings (old system header -> XML tag name)
//   - Validation rules (character limits, data types, required/optional)
//   - XML nesting structure (which fields belong to which parent element)
//
// TEMPLATE STRUCTURE (Expected Columns):
//   The parser expects the XLSX template to have the following columns.
//   Column positions are configurable via the TemplateColumns struct.
//
//   | Column A          | Column B      | Column C   | Column D  | Column E   | Column F              | Column G           |
//   |-------------------|---------------|------------|-----------|------------|-----------------------|--------------------|
//   | Old System Header | XML Tag Name  | Parent Tag | Data Type | Max Length | Required/Optional     | Conditional Rule   |
//   | CHK_NUM           | CheckNumber   | transaction| numeric   | 10         | required              |                    |
//   | CHK_AMT           | CheckAmount   | transaction| decimal   | 15         | required              |                    |
//   | POL_NUM           | PolicyNumber  | lineItem   | alphanum  | 12         | required              |                    |
//   | INV_NUM           | InvoiceNumber | lineItem   | alphanum  | 20         | optional              |                    |
//   | PAY_REASON        | PaymentReason | lineItem   | string    | 50         | conditional           | if CheckAmount>10000|
//
// CUSTOMIZATION:
//   - Modify the TemplateColumns struct to match your actual column positions
//   - Add new validation rule types as needed
//   - Extend the Schema struct to capture additional metadata
//
// =============================================================================

package xlsxparser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// =============================================================================
// SCHEMA STRUCTURE
// =============================================================================

// Schema represents the parsed template schema.
// It contains all the information needed to:
//   - Map CSV columns to XML tags
//   - Validate data according to the template rules
//   - Generate the correct XML structure
type Schema struct {
	// TemplateFile is the path to the source template file.
	TemplateFile string

	// FieldMappings contains the mapping for each field.
	// The key is the old system header (CSV column name).
	FieldMappings map[string]*FieldMapping

	// TransactionFields are fields that belong to the <transaction> element.
	TransactionFields []string

	// LineItemFields are fields that belong to the <lineItem> element.
	LineItemFields []string

	// CashbookFields are fields that belong to the root <cashbook> element.
	CashbookFields []string

	// XMLRootElement is the name of the root XML element.
	// Default: "cashbook"
	//
	// CUSTOMIZATION: Change this if your XML uses a different root element name.
	XMLRootElement string

	// XMLTransactionElement is the name of the transaction element.
	// Default: "transaction"
	//
	// CUSTOMIZATION: Change this if your XML uses a different transaction element name.
	XMLTransactionElement string

	// XMLLineItemElement is the name of the line item element.
	// Default: "lineItem"
	//
	// CUSTOMIZATION: Change this if your XML uses a different line item element name.
	XMLLineItemElement string
}

// FieldMapping represents the mapping and validation rules for a single field.
type FieldMapping struct {
	// OldHeader is the column header from the old/legacy CSV system.
	// This is used to find the column in the input CSV.
	OldHeader string

	// XMLTag is the name of the XML element to create.
	// This is the tag name that will appear in the output XML.
	XMLTag string

	// ParentTag indicates which XML element this field belongs to.
	// Valid values: "cashbook", "transaction", "lineItem"
	//
	// CUSTOMIZATION: Add additional parent types if your XML structure is more complex.
	ParentTag string

	// DataType specifies the expected data type for validation.
	// Valid values:
	//   - "string"      : Any text value
	//   - "numeric"     : Integer numbers only
	//   - "decimal"     : Decimal numbers (with optional precision, e.g., "decimal(2)")
	//   - "alphanumeric": Letters and numbers only
	//   - "alpha"       : Letters only
	//   - "date"        : Date value (with optional format, e.g., "date(YYYY-MM-DD)")
	//   - "boolean"     : True/false values
	//
	// CUSTOMIZATION: Add additional data types as needed.
	DataType string

	// MaxLength is the maximum allowed character length.
	// A value of 0 means no limit.
	MaxLength int

	// RequiredType indicates whether the field is required, optional, or conditional.
	// Valid values: "required", "optional", "conditional"
	RequiredType string

	// ConditionalRule contains the condition for conditional fields.
	// This is only used when RequiredType is "conditional".
	//
	// CUSTOMIZATION: Define your conditional rule syntax here.
	// Examples:
	//   - "if CheckAmount > 10000"
	//   - "if TransactionType == 'PAYMENT'"
	//   - "if PolicyNumber starts_with 'A'"
	//
	// QUESTION FOR USER: What is the syntax for your conditional rules?
	// Please provide examples so we can implement the parser correctly.
	ConditionalRule string

	// DefaultValue is the value to use if the field is empty.
	// Leave empty if there is no default.
	DefaultValue string

	// Order is the position of this field in the output XML.
	// Fields are sorted by this value when generating XML.
	Order int
}

// =============================================================================
// TEMPLATE COLUMN CONFIGURATION
// =============================================================================

// TemplateColumns defines which columns in the XLSX template contain which data.
// This allows the parser to be configured for different template layouts.
//
// CUSTOMIZATION: Modify these values to match your actual template column positions.
// Column indices are 0-based (A=0, B=1, C=2, etc.)
type TemplateColumns struct {
	// OldHeaderColumn is the column containing the old system header.
	// This is the CSV column name from the legacy system.
	// Default: 0 (Column A)
	//
	// QUESTION FOR USER: Which column contains the old system header?
	OldHeaderColumn int

	// XMLTagColumn is the column containing the XML tag name.
	// Default: 1 (Column B)
	//
	// QUESTION FOR USER: Which column contains the XML tag name?
	XMLTagColumn int

	// ParentTagColumn is the column containing the parent XML element.
	// Default: 2 (Column C)
	//
	// QUESTION FOR USER: Which column indicates the parent element (transaction/lineItem)?
	ParentTagColumn int

	// DataTypeColumn is the column containing the data type.
	// Default: 3 (Column D)
	//
	// QUESTION FOR USER: Which column contains the data type (numeric/alphanumeric/etc.)?
	DataTypeColumn int

	// MaxLengthColumn is the column containing the maximum character length.
	// Default: 4 (Column E)
	//
	// QUESTION FOR USER: Which column contains the character limit?
	MaxLengthColumn int

	// RequiredColumn is the column containing required/optional/conditional status.
	// Default: 5 (Column F)
	//
	// QUESTION FOR USER: Which column contains the required/optional indicator?
	RequiredColumn int

	// ConditionalRuleColumn is the column containing conditional rules.
	// Default: 6 (Column G)
	//
	// QUESTION FOR USER: Which column contains the conditional rule (if any)?
	ConditionalRuleColumn int

	// HeaderRow is the row number containing column headers (0-based).
	// Default: 0 (Row 1)
	HeaderRow int

	// DataStartRow is the row number where data begins (0-based).
	// Default: 1 (Row 2)
	DataStartRow int
}

// DefaultTemplateColumns returns the default column configuration.
// CUSTOMIZATION: Modify these defaults to match your template layout.
func DefaultTemplateColumns() TemplateColumns {
	return TemplateColumns{
		OldHeaderColumn:       0, // Column A
		XMLTagColumn:          1, // Column B
		ParentTagColumn:       2, // Column C
		DataTypeColumn:        3, // Column D
		MaxLengthColumn:       4, // Column E
		RequiredColumn:        5, // Column F
		ConditionalRuleColumn: 6, // Column G
		HeaderRow:             0, // Row 1
		DataStartRow:          1, // Row 2
	}
}

// =============================================================================
// PARSER FUNCTIONS
// =============================================================================

// Parse reads an XLSX template file and extracts the schema.
//
// PARAMETERS:
//   - templatePath: The path to the XLSX template file.
//
// RETURNS:
//   - A pointer to the Schema struct containing all field mappings.
//   - An error if the file cannot be read or parsed.
//
// CUSTOMIZATION:
//   - Modify the column configuration by passing a custom TemplateColumns struct.
//   - Add additional parsing logic for custom template formats.
func Parse(templatePath string) (*Schema, error) {
	return ParseWithConfig(templatePath, DefaultTemplateColumns())
}

// ParseWithConfig reads an XLSX template file using a custom column configuration.
//
// PARAMETERS:
//   - templatePath: The path to the XLSX template file.
//   - columns: The column configuration for parsing.
//
// RETURNS:
//   - A pointer to the Schema struct containing all field mappings.
//   - An error if the file cannot be read or parsed.
func ParseWithConfig(templatePath string, columns TemplateColumns) (*Schema, error) {
	// Open the XLSX file.
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template file: %w", err)
	}
	defer f.Close()

	// Initialize the schema.
	schema := &Schema{
		TemplateFile:          templatePath,
		FieldMappings:         make(map[string]*FieldMapping),
		TransactionFields:     []string{},
		LineItemFields:        []string{},
		CashbookFields:        []string{},
		XMLRootElement:        "cashbook",     // CUSTOMIZATION: Change if different
		XMLTransactionElement: "transaction",  // CUSTOMIZATION: Change if different
		XMLLineItemElement:    "lineItem",     // CUSTOMIZATION: Change if different
	}

	// Get the first sheet name.
	// CUSTOMIZATION: If your template has multiple sheets, modify this logic.
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("template file has no sheets")
	}

	// Get all rows from the sheet.
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	// Parse each data row.
	for i := columns.DataStartRow; i < len(rows); i++ {
		row := rows[i]

		// Skip empty rows.
		if len(row) == 0 || isRowEmpty(row) {
			continue
		}

		// Parse the field mapping from this row.
		mapping, err := parseRow(row, columns, i)
		if err != nil {
			return nil, fmt.Errorf("error parsing row %d: %w", i+1, err)
		}

		// Skip rows with empty old header (no mapping defined).
		if mapping.OldHeader == "" {
			continue
		}

		// Add the mapping to the schema.
		schema.FieldMappings[mapping.OldHeader] = mapping

		// Categorize the field by parent tag.
		switch strings.ToLower(mapping.ParentTag) {
		case "transaction":
			schema.TransactionFields = append(schema.TransactionFields, mapping.OldHeader)
		case "lineitem":
			schema.LineItemFields = append(schema.LineItemFields, mapping.OldHeader)
		case "cashbook":
			schema.CashbookFields = append(schema.CashbookFields, mapping.OldHeader)
		default:
			// Default to lineItem if parent tag is not recognized.
			schema.LineItemFields = append(schema.LineItemFields, mapping.OldHeader)
		}
	}

	return schema, nil
}

// parseRow extracts a FieldMapping from a single row.
//
// PARAMETERS:
//   - row: The row data as a slice of strings.
//   - columns: The column configuration.
//   - rowIndex: The row index (for error messages).
//
// RETURNS:
//   - A pointer to the FieldMapping struct.
//   - An error if parsing fails.
func parseRow(row []string, columns TemplateColumns, rowIndex int) (*FieldMapping, error) {
	mapping := &FieldMapping{
		Order: rowIndex,
	}

	// Helper function to safely get a cell value.
	getCell := func(index int) string {
		if index < len(row) {
			return strings.TrimSpace(row[index])
		}
		return ""
	}

	// Extract values from each column.
	mapping.OldHeader = getCell(columns.OldHeaderColumn)
	mapping.XMLTag = getCell(columns.XMLTagColumn)
	mapping.ParentTag = getCell(columns.ParentTagColumn)
	mapping.DataType = getCell(columns.DataTypeColumn)
	mapping.RequiredType = getCell(columns.RequiredColumn)
	mapping.ConditionalRule = getCell(columns.ConditionalRuleColumn)

	// Parse max length as integer.
	maxLengthStr := getCell(columns.MaxLengthColumn)
	if maxLengthStr != "" {
		maxLength, err := strconv.Atoi(maxLengthStr)
		if err != nil {
			// Log warning but don't fail - treat as no limit.
			mapping.MaxLength = 0
		} else {
			mapping.MaxLength = maxLength
		}
	}

	// Normalize required type.
	mapping.RequiredType = normalizeRequiredType(mapping.RequiredType)

	// Normalize data type.
	mapping.DataType = normalizeDataType(mapping.DataType)

	return mapping, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// isRowEmpty checks if a row contains only empty cells.
func isRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// normalizeRequiredType normalizes the required type to a standard value.
//
// CUSTOMIZATION: Add additional mappings for your template's terminology.
// For example, if your template uses "Y" for required and "N" for optional.
func normalizeRequiredType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "required", "req", "r", "yes", "y", "true", "1", "mandatory":
		return "required"
	case "optional", "opt", "o", "no", "n", "false", "0":
		return "optional"
	case "conditional", "cond", "c", "if":
		return "conditional"
	default:
		// Default to optional if not recognized.
		return "optional"
	}
}

// normalizeDataType normalizes the data type to a standard value.
//
// CUSTOMIZATION: Add additional mappings for your template's terminology.
func normalizeDataType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	// Handle data types with parameters (e.g., "decimal(2)", "date(YYYY-MM-DD)")
	if strings.HasPrefix(value, "decimal") {
		return value // Keep the full value including precision
	}
	if strings.HasPrefix(value, "date") {
		return value // Keep the full value including format
	}

	switch value {
	case "string", "str", "text", "varchar":
		return "string"
	case "numeric", "num", "number", "int", "integer":
		return "numeric"
	case "decimal", "dec", "float", "double", "money", "currency":
		return "decimal"
	case "alphanumeric", "alphanum", "an":
		return "alphanumeric"
	case "alpha", "a", "letters":
		return "alpha"
	case "boolean", "bool", "bit":
		return "boolean"
	default:
		// Default to string if not recognized.
		return "string"
	}
}

// =============================================================================
// SCHEMA METHODS
// =============================================================================

// GetFieldMapping returns the field mapping for a given old header.
//
// PARAMETERS:
//   - oldHeader: The old system header (CSV column name).
//
// RETURNS:
//   - The FieldMapping for this header, or nil if not found.
func (s *Schema) GetFieldMapping(oldHeader string) *FieldMapping {
	return s.FieldMappings[oldHeader]
}

// GetXMLTag returns the XML tag name for a given old header.
//
// PARAMETERS:
//   - oldHeader: The old system header (CSV column name).
//
// RETURNS:
//   - The XML tag name, or the old header if no mapping exists.
func (s *Schema) GetXMLTag(oldHeader string) string {
	if mapping, exists := s.FieldMappings[oldHeader]; exists {
		return mapping.XMLTag
	}
	return oldHeader
}

// IsTransactionField checks if a field belongs to the transaction element.
func (s *Schema) IsTransactionField(oldHeader string) bool {
	if mapping, exists := s.FieldMappings[oldHeader]; exists {
		return strings.ToLower(mapping.ParentTag) == "transaction"
	}
	return false
}

// IsLineItemField checks if a field belongs to the line item element.
func (s *Schema) IsLineItemField(oldHeader string) bool {
	if mapping, exists := s.FieldMappings[oldHeader]; exists {
		return strings.ToLower(mapping.ParentTag) == "lineitem"
	}
	return true // Default to line item
}

// =============================================================================
// MULTI-SHEET SUPPORT
// =============================================================================

// ParseMultiSheet parses a template file with multiple sheets.
// Each sheet represents a different transaction type.
//
// PARAMETERS:
//   - templatePath: The path to the XLSX template file.
//
// RETURNS:
//   - A map of Schema structs, keyed by sheet name.
//   - An error if the file cannot be read or parsed.
//
// CUSTOMIZATION:
//   Use this function if your template has separate sheets for different
//   transaction types (e.g., "Payments", "Receipts", "CLT", "ACH").
func ParseMultiSheet(templatePath string) (map[string]*Schema, error) {
	return ParseMultiSheetWithConfig(templatePath, DefaultTemplateColumns())
}

// ParseMultiSheetWithConfig parses a multi-sheet template with custom configuration.
func ParseMultiSheetWithConfig(templatePath string, columns TemplateColumns) (map[string]*Schema, error) {
	// Open the XLSX file.
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open template file: %w", err)
	}
	defer f.Close()

	schemas := make(map[string]*Schema)

	// Get all sheet names.
	sheetNames := f.GetSheetList()

	// Parse each sheet.
	for _, sheetName := range sheetNames {
		// Skip hidden sheets or sheets with specific prefixes.
		// CUSTOMIZATION: Add logic to skip certain sheets if needed.
		if strings.HasPrefix(sheetName, "_") {
			continue
		}

		schema, err := parseSheet(f, sheetName, columns)
		if err != nil {
			return nil, fmt.Errorf("error parsing sheet '%s': %w", sheetName, err)
		}

		schema.TemplateFile = templatePath
		schemas[sheetName] = schema
	}

	return schemas, nil
}

// parseSheet parses a single sheet from an open XLSX file.
func parseSheet(f *excelize.File, sheetName string, columns TemplateColumns) (*Schema, error) {
	// Initialize the schema.
	schema := &Schema{
		FieldMappings:         make(map[string]*FieldMapping),
		TransactionFields:     []string{},
		LineItemFields:        []string{},
		CashbookFields:        []string{},
		XMLRootElement:        "cashbook",
		XMLTransactionElement: "transaction",
		XMLLineItemElement:    "lineItem",
	}

	// Get all rows from the sheet.
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	// Parse each data row.
	for i := columns.DataStartRow; i < len(rows); i++ {
		row := rows[i]

		if len(row) == 0 || isRowEmpty(row) {
			continue
		}

		mapping, err := parseRow(row, columns, i)
		if err != nil {
			return nil, fmt.Errorf("error parsing row %d: %w", i+1, err)
		}

		if mapping.OldHeader == "" {
			continue
		}

		schema.FieldMappings[mapping.OldHeader] = mapping

		switch strings.ToLower(mapping.ParentTag) {
		case "transaction":
			schema.TransactionFields = append(schema.TransactionFields, mapping.OldHeader)
		case "lineitem":
			schema.LineItemFields = append(schema.LineItemFields, mapping.OldHeader)
		case "cashbook":
			schema.CashbookFields = append(schema.CashbookFields, mapping.OldHeader)
		default:
			schema.LineItemFields = append(schema.LineItemFields, mapping.OldHeader)
		}
	}

	return schema, nil
}
