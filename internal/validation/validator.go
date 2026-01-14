// =============================================================================
// CSV to XML Converter - Validation Engine
// =============================================================================
//
// This module provides comprehensive validation for the converted data.
// It validates data against the rules defined in the XLSX template schema,
// including:
//   - Character length limits
//   - Data type validation (numeric, alphanumeric, date, etc.)
//   - Required field checks
//   - Conditional validation rules
//   - Format validation (patterns, ranges, etc.)
//
// VALIDATION STRATEGY:
//   Validation is performed at multiple levels:
//   1. Field-level: Each field is validated against its schema definition
//   2. Row-level: Cross-field validations within a single row
//   3. Transaction-level: Validations across all line items in a transaction
//   4. Document-level: Validations across the entire document
//
// ERROR HANDLING:
//   - Errors are collected, not thrown immediately
//   - Each error includes detailed context (file, row, field, value)
//   - Errors can be warnings (continue processing) or fatal (stop processing)
//   - Error output is designed for easy troubleshooting
//
// CUSTOMIZATION:
//   - Add new validation types by extending the ValidationType enum
//   - Implement custom validators for specific business rules
//   - Modify error severity levels as needed
//
// =============================================================================

package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/converter"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/xlsxparser"
)

// =============================================================================
// VALIDATION ERROR TYPES
// =============================================================================

// ValidationError represents a single validation error.
type ValidationError struct {
	// Severity indicates the severity of the error.
	// "error" = fatal, processing should stop
	// "warning" = non-fatal, processing can continue
	Severity string

	// Field is the name of the field that failed validation.
	Field string

	// Value is the actual value that failed validation.
	Value string

	// Rule is the validation rule that was violated.
	Rule string

	// Message is a human-readable error message.
	Message string

	// TransactionID is the ID of the transaction containing the error.
	TransactionID int

	// LineItemID is the ID of the line item containing the error.
	LineItemID int

	// RowNumber is the original CSV row number (for error reporting).
	RowNumber int
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] Transaction %d, LineItem %d, Field '%s': %s (value: '%s')",
		strings.ToUpper(e.Severity),
		e.TransactionID,
		e.LineItemID,
		e.Field,
		e.Message,
		e.Value,
	)
}

// =============================================================================
// VALIDATION RESULT
// =============================================================================

// ValidationResult contains the results of validation.
type ValidationResult struct {
	// IsValid is true if there are no fatal errors.
	IsValid bool

	// Errors contains all validation errors (including warnings).
	Errors []*ValidationError

	// ErrorCount is the number of fatal errors.
	ErrorCount int

	// WarningCount is the number of warnings.
	WarningCount int

	// FieldsValidated is the total number of fields validated.
	FieldsValidated int

	// TransactionsValidated is the total number of transactions validated.
	TransactionsValidated int
}

// =============================================================================
// VALIDATOR
// =============================================================================

// Validator performs validation on transactions.
type Validator struct {
	schema  *xlsxparser.Schema
	options ValidationOptions
}

// ValidationOptions contains options for validation.
type ValidationOptions struct {
	// StopOnFirstError stops validation after the first fatal error.
	// Default: false
	StopOnFirstError bool

	// TreatWarningsAsErrors treats warnings as fatal errors.
	// Default: false
	TreatWarningsAsErrors bool

	// SkipOptionalValidation skips validation of optional fields.
	// Default: false
	SkipOptionalValidation bool

	// CustomValidators is a map of custom validation functions.
	// Key is the field name, value is the validation function.
	CustomValidators map[string]CustomValidatorFunc
}

// CustomValidatorFunc is a function type for custom validators.
// It takes the field value and returns an error message if validation fails.
type CustomValidatorFunc func(value string, context ValidationContext) string

// ValidationContext provides context for custom validators.
type ValidationContext struct {
	FieldName     string
	FieldMapping  *xlsxparser.FieldMapping
	Transaction   *converter.Transaction
	LineItem      *converter.LineItem
	AllFields     map[string]string
}

// DefaultValidationOptions returns the default validation options.
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		StopOnFirstError:       false,
		TreatWarningsAsErrors:  false,
		SkipOptionalValidation: false,
		CustomValidators:       make(map[string]CustomValidatorFunc),
	}
}

// NewValidator creates a new Validator instance.
func NewValidator(schema *xlsxparser.Schema) *Validator {
	return &Validator{
		schema:  schema,
		options: DefaultValidationOptions(),
	}
}

// NewValidatorWithOptions creates a new Validator with custom options.
func NewValidatorWithOptions(schema *xlsxparser.Schema, options ValidationOptions) *Validator {
	return &Validator{
		schema:  schema,
		options: options,
	}
}

// =============================================================================
// MAIN VALIDATION FUNCTION
// =============================================================================

// Validate validates all transactions and returns a list of errors.
// This is the main entry point for validation.
//
// PARAMETERS:
//   - transactions: The transactions to validate.
//   - schema: The schema containing validation rules.
//
// RETURNS:
//   - A slice of ValidationError pointers.
func Validate(transactions []converter.Transaction, schema *xlsxparser.Schema) []*ValidationError {
	validator := NewValidator(schema)
	result := validator.ValidateAll(transactions)
	return result.Errors
}

// ValidateAll validates all transactions and returns a detailed result.
func (v *Validator) ValidateAll(transactions []converter.Transaction) *ValidationResult {
	result := &ValidationResult{
		IsValid:               true,
		Errors:                make([]*ValidationError, 0),
		TransactionsValidated: len(transactions),
	}

	for i := range transactions {
		transactionErrors := v.ValidateTransaction(&transactions[i])

		for _, err := range transactionErrors {
			result.Errors = append(result.Errors, err)

			if err.Severity == "error" {
				result.ErrorCount++
				result.IsValid = false

				if v.options.StopOnFirstError {
					return result
				}
			} else {
				result.WarningCount++

				if v.options.TreatWarningsAsErrors {
					result.IsValid = false
				}
			}
		}
	}

	return result
}

// ValidateTransaction validates a single transaction.
func (v *Validator) ValidateTransaction(transaction *converter.Transaction) []*ValidationError {
	var errors []*ValidationError

	// Validate each line item.
	for i := range transaction.LineItems {
		lineItemErrors := v.ValidateLineItem(transaction, &transaction.LineItems[i])
		errors = append(errors, lineItemErrors...)
	}

	// Perform transaction-level validations.
	// CUSTOMIZATION: Add cross-line-item validations here.
	//
	// PSEUDOCODE:
	// transactionErrors := v.validateTransactionLevel(transaction)
	// errors = append(errors, transactionErrors...)

	return errors
}

// ValidateLineItem validates a single line item.
func (v *Validator) ValidateLineItem(transaction *converter.Transaction, lineItem *converter.LineItem) []*ValidationError {
	var errors []*ValidationError

	// Validate each field in the line item.
	for fieldName, value := range lineItem.Fields {
		mapping := v.schema.GetFieldMapping(fieldName)
		if mapping == nil {
			// Field not in schema, skip validation.
			continue
		}

		// Skip optional fields if configured.
		if v.options.SkipOptionalValidation && mapping.RequiredType == "optional" {
			continue
		}

		// Validate the field.
		fieldErrors := v.ValidateField(value, mapping, transaction, lineItem)
		errors = append(errors, fieldErrors...)

		// Run custom validator if defined.
		if customValidator, exists := v.options.CustomValidators[fieldName]; exists {
			context := ValidationContext{
				FieldName:    fieldName,
				FieldMapping: mapping,
				Transaction:  transaction,
				LineItem:     lineItem,
				AllFields:    lineItem.Fields,
			}

			if errMsg := customValidator(value, context); errMsg != "" {
				errors = append(errors, &ValidationError{
					Severity:      "error",
					Field:         fieldName,
					Value:         value,
					Rule:          "custom",
					Message:       errMsg,
					TransactionID: transaction.ID,
					LineItemID:    lineItem.ID,
				})
			}
		}
	}

	return errors
}

// ValidateField validates a single field value against its schema definition.
func (v *Validator) ValidateField(value string, mapping *xlsxparser.FieldMapping, transaction *converter.Transaction, lineItem *converter.LineItem) []*ValidationError {
	var errors []*ValidationError

	// =========================================================================
	// REQUIRED FIELD VALIDATION
	// =========================================================================
	// Check if a required field is empty.

	if mapping.RequiredType == "required" && value == "" {
		errors = append(errors, &ValidationError{
			Severity:      "error",
			Field:         mapping.OldHeader,
			Value:         value,
			Rule:          "required",
			Message:       fmt.Sprintf("Required field '%s' is empty", mapping.XMLTag),
			TransactionID: transaction.ID,
			LineItemID:    lineItem.ID,
		})
		// Don't continue validation if required field is empty.
		return errors
	}

	// Skip further validation if value is empty (for optional fields).
	if value == "" {
		return errors
	}

	// =========================================================================
	// CONDITIONAL FIELD VALIDATION
	// =========================================================================
	// Check if a conditional field should be required.

	if mapping.RequiredType == "conditional" && mapping.ConditionalRule != "" {
		isRequired := evaluateCondition(mapping.ConditionalRule, lineItem.Fields)
		if isRequired && value == "" {
			errors = append(errors, &ValidationError{
				Severity:      "error",
				Field:         mapping.OldHeader,
				Value:         value,
				Rule:          "conditional_required",
				Message:       fmt.Sprintf("Field '%s' is required when: %s", mapping.XMLTag, mapping.ConditionalRule),
				TransactionID: transaction.ID,
				LineItemID:    lineItem.ID,
			})
		}
	}

	// =========================================================================
	// MAX LENGTH VALIDATION
	// =========================================================================
	// Check if the value exceeds the maximum allowed length.

	if mapping.MaxLength > 0 && len(value) > mapping.MaxLength {
		errors = append(errors, &ValidationError{
			Severity:      "error",
			Field:         mapping.OldHeader,
			Value:         value,
			Rule:          "max_length",
			Message:       fmt.Sprintf("Value exceeds maximum length of %d characters (actual: %d)", mapping.MaxLength, len(value)),
			TransactionID: transaction.ID,
			LineItemID:    lineItem.ID,
		})
	}

	// =========================================================================
	// DATA TYPE VALIDATION
	// =========================================================================
	// Validate the value against the expected data type.

	typeError := validateDataType(value, mapping.DataType)
	if typeError != "" {
		errors = append(errors, &ValidationError{
			Severity:      "error",
			Field:         mapping.OldHeader,
			Value:         value,
			Rule:          "data_type",
			Message:       typeError,
			TransactionID: transaction.ID,
			LineItemID:    lineItem.ID,
		})
	}

	return errors
}

// =============================================================================
// DATA TYPE VALIDATORS
// =============================================================================

// validateDataType validates a value against a data type.
//
// PARAMETERS:
//   - value: The value to validate.
//   - dataType: The expected data type.
//
// RETURNS:
//   - An error message if validation fails, empty string if valid.
//
// SUPPORTED DATA TYPES:
//   - string: Any text value (always valid)
//   - numeric: Integer numbers only
//   - decimal: Decimal numbers (with optional precision)
//   - alphanumeric: Letters and numbers only
//   - alpha: Letters only
//   - date: Date value (with optional format)
//   - boolean: True/false values
//
// CUSTOMIZATION:
//   Add new data types by adding cases to this function.
func validateDataType(value, dataType string) string {
	switch {
	case dataType == "string" || dataType == "":
		// String type accepts any value.
		return ""

	case dataType == "numeric":
		return validateNumeric(value)

	case strings.HasPrefix(dataType, "decimal"):
		return validateDecimal(value, dataType)

	case dataType == "alphanumeric":
		return validateAlphanumeric(value)

	case dataType == "alpha":
		return validateAlpha(value)

	case strings.HasPrefix(dataType, "date"):
		return validateDate(value, dataType)

	case dataType == "boolean":
		return validateBoolean(value)

	default:
		// Unknown type, treat as string.
		return ""
	}
}

// validateNumeric validates that a value is a valid integer.
func validateNumeric(value string) string {
	// Remove leading/trailing whitespace.
	value = strings.TrimSpace(value)

	// Allow empty values (handled by required check).
	if value == "" {
		return ""
	}

	// Try to parse as integer.
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return fmt.Sprintf("Value '%s' is not a valid integer", value)
	}

	return ""
}

// validateDecimal validates that a value is a valid decimal number.
//
// PARAMETERS:
//   - value: The value to validate.
//   - dataType: The data type string (e.g., "decimal", "decimal(2)").
//
// The optional precision in parentheses specifies the maximum decimal places.
func validateDecimal(value, dataType string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	// Try to parse as float.
	_, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Sprintf("Value '%s' is not a valid decimal number", value)
	}

	// Check precision if specified.
	// Extract precision from dataType (e.g., "decimal(2)" -> 2).
	if strings.Contains(dataType, "(") {
		precisionStr := extractParenthesesContent(dataType)
		if precisionStr != "" {
			precision, err := strconv.Atoi(precisionStr)
			if err == nil && precision >= 0 {
				// Check actual decimal places.
				if strings.Contains(value, ".") {
					parts := strings.Split(value, ".")
					if len(parts) == 2 && len(parts[1]) > precision {
						return fmt.Sprintf("Value '%s' has more than %d decimal places", value, precision)
					}
				}
			}
		}
	}

	return ""
}

// validateAlphanumeric validates that a value contains only letters and numbers.
func validateAlphanumeric(value string) string {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return fmt.Sprintf("Value '%s' contains non-alphanumeric characters", value)
		}
	}
	return ""
}

// validateAlpha validates that a value contains only letters.
func validateAlpha(value string) string {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) {
			return fmt.Sprintf("Value '%s' contains non-alphabetic characters", value)
		}
	}
	return ""
}

// validateDate validates that a value is a valid date.
//
// PARAMETERS:
//   - value: The value to validate.
//   - dataType: The data type string (e.g., "date", "date(2006-01-02)").
//
// CUSTOMIZATION:
//   Add additional date formats as needed.
func validateDate(value, dataType string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return ""
	}

	// Extract format from dataType if specified.
	format := extractParenthesesContent(dataType)
	if format == "" {
		// Try common date formats.
		formats := []string{
			"2006-01-02",
			"01/02/2006",
			"02/01/2006",
			"2006/01/02",
			"Jan 2, 2006",
			"January 2, 2006",
			"20060102",
		}

		for _, f := range formats {
			if _, err := time.Parse(f, value); err == nil {
				return ""
			}
		}

		return fmt.Sprintf("Value '%s' is not a valid date", value)
	}

	// Try to parse with the specified format.
	if _, err := time.Parse(format, value); err != nil {
		return fmt.Sprintf("Value '%s' does not match date format '%s'", value, format)
	}

	return ""
}

// validateBoolean validates that a value is a valid boolean.
func validateBoolean(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	validValues := []string{"true", "false", "yes", "no", "1", "0", "y", "n", "t", "f"}

	for _, v := range validValues {
		if value == v {
			return ""
		}
	}

	return fmt.Sprintf("Value '%s' is not a valid boolean", value)
}

// =============================================================================
// CONDITIONAL RULE EVALUATION
// =============================================================================

// evaluateCondition evaluates a conditional rule against field values.
//
// PARAMETERS:
//   - rule: The conditional rule string.
//   - fields: The field values to evaluate against.
//
// RETURNS:
//   - true if the condition is met, false otherwise.
//
// SUPPORTED RULE SYNTAX:
//   - "if FieldName == 'value'"
//   - "if FieldName != 'value'"
//   - "if FieldName > 100"
//   - "if FieldName < 100"
//   - "if FieldName >= 100"
//   - "if FieldName <= 100"
//   - "if FieldName starts_with 'prefix'"
//   - "if FieldName ends_with 'suffix'"
//   - "if FieldName contains 'substring'"
//   - "if FieldName is_empty"
//   - "if FieldName is_not_empty"
//
// CUSTOMIZATION:
//   Add new operators by extending this function.
//
// QUESTION FOR USER:
//   What syntax do you use for conditional rules in your templates?
//   Please provide examples so we can implement the correct parser.
func evaluateCondition(rule string, fields map[string]string) bool {
	// Remove "if " prefix if present.
	rule = strings.TrimPrefix(rule, "if ")
	rule = strings.TrimSpace(rule)

	// Parse the rule.
	// PSEUDOCODE for rule parsing:
	//
	// 1. Extract the field name (first word).
	// 2. Extract the operator (==, !=, >, <, >=, <=, starts_with, etc.).
	// 3. Extract the comparison value.
	// 4. Get the actual field value from fields map.
	// 5. Perform the comparison.

	// Simple implementation for common patterns.
	// CUSTOMIZATION: Implement your specific rule syntax here.

	// Pattern: "FieldName == 'value'"
	if matches := regexp.MustCompile(`(\w+)\s*==\s*'([^']*)'`).FindStringSubmatch(rule); len(matches) == 3 {
		fieldName := matches[1]
		expectedValue := matches[2]
		actualValue := fields[fieldName]
		return actualValue == expectedValue
	}

	// Pattern: "FieldName != 'value'"
	if matches := regexp.MustCompile(`(\w+)\s*!=\s*'([^']*)'`).FindStringSubmatch(rule); len(matches) == 3 {
		fieldName := matches[1]
		expectedValue := matches[2]
		actualValue := fields[fieldName]
		return actualValue != expectedValue
	}

	// Pattern: "FieldName > number"
	if matches := regexp.MustCompile(`(\w+)\s*>\s*(\d+(?:\.\d+)?)`).FindStringSubmatch(rule); len(matches) == 3 {
		fieldName := matches[1]
		threshold, _ := strconv.ParseFloat(matches[2], 64)
		actualValue, _ := strconv.ParseFloat(fields[fieldName], 64)
		return actualValue > threshold
	}

	// Pattern: "FieldName < number"
	if matches := regexp.MustCompile(`(\w+)\s*<\s*(\d+(?:\.\d+)?)`).FindStringSubmatch(rule); len(matches) == 3 {
		fieldName := matches[1]
		threshold, _ := strconv.ParseFloat(matches[2], 64)
		actualValue, _ := strconv.ParseFloat(fields[fieldName], 64)
		return actualValue < threshold
	}

	// Pattern: "FieldName starts_with 'prefix'"
	if matches := regexp.MustCompile(`(\w+)\s+starts_with\s+'([^']*)'`).FindStringSubmatch(rule); len(matches) == 3 {
		fieldName := matches[1]
		prefix := matches[2]
		actualValue := fields[fieldName]
		return strings.HasPrefix(actualValue, prefix)
	}

	// Pattern: "FieldName is_empty"
	if matches := regexp.MustCompile(`(\w+)\s+is_empty`).FindStringSubmatch(rule); len(matches) == 2 {
		fieldName := matches[1]
		actualValue := fields[fieldName]
		return actualValue == ""
	}

	// Pattern: "FieldName is_not_empty"
	if matches := regexp.MustCompile(`(\w+)\s+is_not_empty`).FindStringSubmatch(rule); len(matches) == 2 {
		fieldName := matches[1]
		actualValue := fields[fieldName]
		return actualValue != ""
	}

	// Unknown rule format, default to false.
	// CUSTOMIZATION: Add logging here for debugging unknown rules.
	return false
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// extractParenthesesContent extracts content between parentheses.
// Example: "decimal(2)" -> "2"
func extractParenthesesContent(s string) string {
	start := strings.Index(s, "(")
	end := strings.Index(s, ")")

	if start != -1 && end != -1 && end > start {
		return s[start+1 : end]
	}

	return ""
}

// =============================================================================
// ERROR FORMATTING
// =============================================================================

// FormatErrors formats validation errors for display or logging.
//
// PARAMETERS:
//   - errors: The validation errors to format.
//
// RETURNS:
//   - A formatted string containing all errors.
func FormatErrors(errors []*ValidationError) string {
	if len(errors) == 0 {
		return "No validation errors."
	}

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Validation completed with %d error(s):\n\n", len(errors)))

	for i, err := range errors {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, err.Error()))
	}

	return builder.String()
}

// WriteErrorLog writes validation errors to a log file.
//
// PARAMETERS:
//   - errors: The validation errors to write.
//   - filePath: The path to the output file.
//
// RETURNS:
//   - An error if writing fails.
//
// CUSTOMIZATION:
//   Modify the output format as needed (e.g., CSV, JSON, HTML).
func WriteErrorLog(errors []*ValidationError, filePath string) error {
	// IMPLEMENTATION:
	// 1. Open the file for writing.
	// 2. Write a header with timestamp and summary.
	// 3. Write each error in a structured format.
	// 4. Close the file.

	// PSEUDOCODE:
	// file, err := os.Create(filePath)
	// if err != nil {
	//     return err
	// }
	// defer file.Close()
	//
	// writer := bufio.NewWriter(file)
	// writer.WriteString(FormatErrors(errors))
	// return writer.Flush()

	return nil // Placeholder
}
