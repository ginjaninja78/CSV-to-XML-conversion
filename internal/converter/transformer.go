// =============================================================================
// CSV to XML Converter - Transformation Engine
// =============================================================================
//
// This module provides the transformation logic for converting field values
// from the legacy CSV format to the format required by the target XML system.
//
// TRANSFORMATION TYPES:
//   - String manipulations (prepend, append, trim, case conversion)
//   - Numeric formatting (padding, precision, rounding)
//   - Date/time conversions
//   - Lookup table replacements
//   - Conditional transformations
//   - Regular expression replacements
//   - Custom transformations via plugins
//
// DEPARTMENT-SPECIFIC RULES:
//   Each department can have its own transformation rules defined in their
//   configuration file. Common use cases include:
//   - Policy number formatting (prepending letters, zero-padding)
//   - Account code transformations
//   - Date format conversions
//   - Currency formatting
//
// CUSTOMIZATION:
//   - Add new transformation types by extending the TransformationType enum
//   - Implement custom transformers for complex business logic
//   - Chain multiple transformations for complex conversions
//
// =============================================================================

package converter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/config"
)

// =============================================================================
// TRANSFORMER
// =============================================================================

// Transformer handles field value transformations.
type Transformer struct {
	rules []config.TransformationRule
}

// NewTransformer creates a new Transformer with the given rules.
func NewTransformer(rules []config.TransformationRule) *Transformer {
	return &Transformer{
		rules: rules,
	}
}

// =============================================================================
// TRANSFORMATION FUNCTIONS
// =============================================================================

// Transform applies all transformation rules to a field value.
//
// PARAMETERS:
//   - fieldName: The name of the field being transformed.
//   - value: The current value of the field.
//   - allFields: All fields in the current row (for conditional transformations).
//
// RETURNS:
//   - The transformed value.
//   - An error if any transformation fails.
func (t *Transformer) Transform(fieldName, value string, allFields map[string]string) (string, error) {
	// Find the rule for this field.
	var rule *config.TransformationRule
	for i := range t.rules {
		if t.rules[i].Field == fieldName {
			rule = &t.rules[i]
			break
		}
	}

	// If no rule found, return the original value.
	if rule == nil {
		return value, nil
	}

	// Apply each action in sequence.
	result := value
	for _, action := range rule.Actions {
		var err error
		result, err = ApplyTransformation(result, action, allFields)
		if err != nil {
			return "", fmt.Errorf("transformation '%s' failed: %w", action.Type, err)
		}
	}

	return result, nil
}

// ApplyTransformation applies a single transformation action.
//
// PARAMETERS:
//   - value: The current value.
//   - action: The transformation action to apply.
//   - allFields: All fields in the current row (for conditional transformations).
//
// RETURNS:
//   - The transformed value.
//   - An error if the transformation fails.
//
// SUPPORTED TRANSFORMATIONS:
//   See the switch statement below for all supported transformation types.
//
// CUSTOMIZATION:
//   Add new transformation types by adding cases to this switch statement.
func ApplyTransformation(value string, action config.TransformationAction, allFields map[string]string) (string, error) {
	switch action.Type {

	// =========================================================================
	// STRING MANIPULATIONS
	// =========================================================================

	case "prepend_string":
		// Add a string to the beginning of the value.
		//
		// EXAMPLE:
		//   Input: "123456"
		//   Action: prepend_string with value "A"
		//   Output: "A123456"
		//
		// USE CASE: Adding department prefixes to policy numbers.
		return action.Value + value, nil

	case "append_string":
		// Add a string to the end of the value.
		//
		// EXAMPLE:
		//   Input: "123456"
		//   Action: append_string with value "-00"
		//   Output: "123456-00"
		return value + action.Value, nil

	case "trim":
		// Remove leading and trailing whitespace.
		return strings.TrimSpace(value), nil

	case "trim_left":
		// Remove leading whitespace or specified characters.
		if action.Value != "" {
			return strings.TrimLeft(value, action.Value), nil
		}
		return strings.TrimLeft(value, " \t\n\r"), nil

	case "trim_right":
		// Remove trailing whitespace or specified characters.
		if action.Value != "" {
			return strings.TrimRight(value, action.Value), nil
		}
		return strings.TrimRight(value, " \t\n\r"), nil

	case "uppercase":
		// Convert to uppercase.
		return strings.ToUpper(value), nil

	case "lowercase":
		// Convert to lowercase.
		return strings.ToLower(value), nil

	case "title_case":
		// Convert to title case (first letter of each word capitalized).
		return strings.Title(strings.ToLower(value)), nil

	case "replace":
		// Replace a substring with another.
		//
		// EXAMPLE:
		//   Input: "hello-world"
		//   Action: replace with find "-" and value "_"
		//   Output: "hello_world"
		if action.Find == "" {
			return value, nil
		}
		return strings.ReplaceAll(value, action.Find, action.Value), nil

	case "regex_replace":
		// Replace using a regular expression.
		//
		// EXAMPLE:
		//   Input: "ABC-123-DEF"
		//   Action: regex_replace with find "[A-Z]+" and value "X"
		//   Output: "X-123-X"
		if action.Find == "" {
			return value, nil
		}
		re, err := regexp.Compile(action.Find)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
		return re.ReplaceAllString(value, action.Value), nil

	case "substring":
		// Extract a substring.
		//
		// VALUE FORMAT: "start,end" (0-indexed, end is exclusive)
		// EXAMPLE:
		//   Input: "ABCDEFGH"
		//   Action: substring with value "2,5"
		//   Output: "CDE"
		parts := strings.Split(action.Value, ",")
		if len(parts) != 2 {
			return value, nil
		}
		start, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, _ := strconv.Atoi(strings.TrimSpace(parts[1]))

		if start < 0 {
			start = 0
		}
		if end > len(value) {
			end = len(value)
		}
		if start >= end || start >= len(value) {
			return "", nil
		}

		return value[start:end], nil

	// =========================================================================
	// NUMERIC FORMATTING
	// =========================================================================

	case "pad_zeros_to_length":
		// Pad with leading zeros to a specific length.
		//
		// EXAMPLE:
		//   Input: "123"
		//   Action: pad_zeros_to_length with value "8"
		//   Output: "00000123"
		//
		// USE CASE: Formatting policy numbers that require fixed length.
		targetLength, err := strconv.Atoi(action.Value)
		if err != nil || targetLength <= 0 {
			return value, nil
		}
		return PadLeft(value, targetLength, '0'), nil

	case "pad_spaces_to_length":
		// Pad with trailing spaces to a specific length.
		targetLength, err := strconv.Atoi(action.Value)
		if err != nil || targetLength <= 0 {
			return value, nil
		}
		return PadRight(value, targetLength, ' '), nil

	case "ensure_length":
		// Truncate or pad to ensure a specific length.
		//
		// EXAMPLE:
		//   Input: "12345678901234"
		//   Action: ensure_length with value "10"
		//   Output: "1234567890" (truncated)
		//
		//   Input: "123"
		//   Action: ensure_length with value "10"
		//   Output: "0000000123" (padded)
		targetLength, err := strconv.Atoi(action.Value)
		if err != nil || targetLength <= 0 {
			return value, nil
		}

		if len(value) > targetLength {
			// Truncate from the right.
			// CUSTOMIZATION: Change to truncate from left if needed.
			return value[:targetLength], nil
		}

		// Pad with leading zeros.
		// CUSTOMIZATION: Change padding character or direction if needed.
		return PadLeft(value, targetLength, '0'), nil

	case "format_number":
		// Format a number with specific decimal places.
		//
		// VALUE FORMAT: Number of decimal places.
		// EXAMPLE:
		//   Input: "1234.5"
		//   Action: format_number with value "2"
		//   Output: "1234.50"
		decimalPlaces, err := strconv.Atoi(action.Value)
		if err != nil || decimalPlaces < 0 {
			return value, nil
		}

		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return value, nil // Not a number, return as-is
		}

		format := fmt.Sprintf("%%.%df", decimalPlaces)
		return fmt.Sprintf(format, num), nil

	case "remove_leading_zeros":
		// Remove leading zeros from a numeric string.
		//
		// EXAMPLE:
		//   Input: "00012345"
		//   Output: "12345"
		result := strings.TrimLeft(value, "0")
		if result == "" {
			return "0", nil // Keep at least one zero
		}
		return result, nil

	// =========================================================================
	// DATE/TIME CONVERSIONS
	// =========================================================================

	case "format_date":
		// Convert date from one format to another.
		//
		// VALUE FORMAT: "input_format|output_format"
		// Uses Go's time format strings.
		//
		// EXAMPLE:
		//   Input: "01/15/2024"
		//   Action: format_date with value "01/02/2006|2006-01-02"
		//   Output: "2024-01-15"
		//
		// COMMON FORMAT STRINGS:
		//   - "2006-01-02" : YYYY-MM-DD
		//   - "01/02/2006" : MM/DD/YYYY
		//   - "02/01/2006" : DD/MM/YYYY
		//   - "20060102"   : YYYYMMDD
		//   - "Jan 2, 2006": Mon D, YYYY
		parts := strings.Split(action.Value, "|")
		if len(parts) != 2 {
			return value, nil
		}

		inputFormat := strings.TrimSpace(parts[0])
		outputFormat := strings.TrimSpace(parts[1])

		t, err := time.Parse(inputFormat, value)
		if err != nil {
			// Try to parse with common formats if specified format fails.
			// CUSTOMIZATION: Add more fallback formats as needed.
			return value, nil
		}

		return t.Format(outputFormat), nil

	// =========================================================================
	// LOOKUP TABLE REPLACEMENTS
	// =========================================================================

	case "lookup":
		// Replace value using a lookup table.
		//
		// EXAMPLE:
		//   Input: "01"
		//   Action: lookup with lookup_table {"01": "January", "02": "February"}
		//   Output: "January"
		//
		// USE CASE: Converting codes to descriptions.
		if replacement, exists := action.LookupTable[value]; exists {
			return replacement, nil
		}
		// Return original value if not found in lookup table.
		// CUSTOMIZATION: You might want to return an error or default value instead.
		return value, nil

	case "lookup_with_default":
		// Replace value using a lookup table, with a default for unknown values.
		//
		// The default value is specified in action.Value.
		if replacement, exists := action.LookupTable[value]; exists {
			return replacement, nil
		}
		return action.Value, nil // Return default

	// =========================================================================
	// CONDITIONAL TRANSFORMATIONS
	// =========================================================================

	case "conditional":
		// Apply transformation based on a condition.
		//
		// CONDITION FORMAT: Uses the same syntax as validation conditions.
		// If the condition is true, the transformation is applied.
		//
		// EXAMPLE:
		//   Condition: "DepartmentCode == 'CLAIMS'"
		//   Action: prepend "C" to policy number
		//
		// CUSTOMIZATION: Implement your conditional logic here.
		//
		// PSEUDOCODE:
		// if evaluateCondition(action.Condition, allFields) {
		//     // Apply the transformation specified in action.Value
		//     // This could be another transformation type or a direct value
		// }
		return value, nil // Placeholder

	case "if_empty_use_default":
		// Use a default value if the field is empty.
		//
		// EXAMPLE:
		//   Input: ""
		//   Action: if_empty_use_default with value "N/A"
		//   Output: "N/A"
		if strings.TrimSpace(value) == "" {
			return action.Value, nil
		}
		return value, nil

	case "if_empty_use_field":
		// Use another field's value if this field is empty.
		//
		// VALUE: The name of the field to use.
		if strings.TrimSpace(value) == "" {
			if otherValue, exists := allFields[action.Value]; exists {
				return otherValue, nil
			}
		}
		return value, nil

	// =========================================================================
	// SPECIAL TRANSFORMATIONS
	// =========================================================================

	case "extract_digits":
		// Extract only digits from the value.
		//
		// EXAMPLE:
		//   Input: "ABC-123-DEF-456"
		//   Output: "123456"
		//
		// USE CASE: Cleaning phone numbers, extracting numeric IDs.
		re := regexp.MustCompile(`\d+`)
		matches := re.FindAllString(value, -1)
		return strings.Join(matches, ""), nil

	case "extract_letters":
		// Extract only letters from the value.
		re := regexp.MustCompile(`[a-zA-Z]+`)
		matches := re.FindAllString(value, -1)
		return strings.Join(matches, ""), nil

	case "remove_special_chars":
		// Remove special characters, keeping only alphanumeric.
		re := regexp.MustCompile(`[^a-zA-Z0-9]`)
		return re.ReplaceAllString(value, ""), nil

	case "normalize_whitespace":
		// Replace multiple spaces with a single space.
		re := regexp.MustCompile(`\s+`)
		return strings.TrimSpace(re.ReplaceAllString(value, " ")), nil

	// =========================================================================
	// DEPARTMENT-SPECIFIC TRANSFORMATIONS
	// =========================================================================
	// These are placeholders for your specific business logic.
	// CUSTOMIZATION: Implement your department-specific transformations here.

	case "format_policy_number":
		// Format a policy number according to department rules.
		//
		// This is a composite transformation that combines multiple steps:
		// 1. Remove any existing prefix
		// 2. Pad with zeros to the required length
		// 3. Add the department-specific prefix
		//
		// PSEUDOCODE:
		// prefix := action.LookupTable["prefix"]
		// length := action.LookupTable["length"]
		//
		// // Remove existing prefix if any
		// value = strings.TrimPrefix(value, prefix)
		//
		// // Pad to length
		// value = PadLeft(value, length, '0')
		//
		// // Add prefix
		// return prefix + value, nil

		return value, nil // Placeholder

	case "format_account_code":
		// Format an account code according to department rules.
		// CUSTOMIZATION: Implement your account code formatting logic.
		return value, nil // Placeholder

	case "format_currency":
		// Format a currency value.
		//
		// EXAMPLE:
		//   Input: "1234.5"
		//   Output: "1234.50" (or "$1,234.50" depending on settings)
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return value, nil
		}
		return fmt.Sprintf("%.2f", num), nil

	default:
		// Unknown transformation type.
		return "", fmt.Errorf("unknown transformation type: %s", action.Type)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// PadLeft pads a string with a character on the left to reach the target length.
func PadLeft(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	padding := make([]rune, length-len(s))
	for i := range padding {
		padding[i] = padChar
	}
	return string(padding) + s
}

// PadRight pads a string with a character on the right to reach the target length.
func PadRight(s string, length int, padChar rune) string {
	if len(s) >= length {
		return s
	}
	padding := make([]rune, length-len(s))
	for i := range padding {
		padding[i] = padChar
	}
	return s + string(padding)
}

// =============================================================================
// BATCH TRANSFORMATION
// =============================================================================

// TransformTransaction applies all transformations to a transaction.
func (t *Transformer) TransformTransaction(transaction *Transaction) error {
	for i := range transaction.LineItems {
		if err := t.TransformLineItem(&transaction.LineItems[i]); err != nil {
			return fmt.Errorf("error transforming line item %d: %w", transaction.LineItems[i].ID, err)
		}
	}
	return nil
}

// TransformLineItem applies all transformations to a line item.
func (t *Transformer) TransformLineItem(lineItem *LineItem) error {
	for fieldName, value := range lineItem.Fields {
		transformedValue, err := t.Transform(fieldName, value, lineItem.Fields)
		if err != nil {
			return fmt.Errorf("error transforming field '%s': %w", fieldName, err)
		}
		lineItem.Fields[fieldName] = transformedValue
	}
	return nil
}

// =============================================================================
// TRANSFORMATION CHAIN
// =============================================================================

// TransformationChain allows chaining multiple transformers.
type TransformationChain struct {
	transformers []*Transformer
}

// NewTransformationChain creates a new transformation chain.
func NewTransformationChain() *TransformationChain {
	return &TransformationChain{
		transformers: make([]*Transformer, 0),
	}
}

// Add adds a transformer to the chain.
func (c *TransformationChain) Add(t *Transformer) *TransformationChain {
	c.transformers = append(c.transformers, t)
	return c
}

// Transform applies all transformers in the chain.
func (c *TransformationChain) Transform(fieldName, value string, allFields map[string]string) (string, error) {
	result := value
	for _, t := range c.transformers {
		var err error
		result, err = t.Transform(fieldName, result, allFields)
		if err != nil {
			return "", err
		}
	}
	return result, nil
}
