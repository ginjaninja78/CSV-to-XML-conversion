// =============================================================================
// CSV to XML Converter - Configuration Module
// =============================================================================
//
// This module is responsible for loading and managing all configuration files.
// It handles both the main application configuration and department-specific
// configurations.
//
// CONFIGURATION FILES:
//   1. Main Config (config.yaml): Global application settings
//   2. Department Configs (configs/*.yaml): Department-specific rules
//
// ARCHITECTURE:
//   The configuration system is designed to be:
//   - Modular: Each department has its own configuration file
//   - Extensible: New departments can be added without code changes
//   - Validated: All configurations are validated on load
//
// =============================================================================

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// MAIN CONFIGURATION STRUCTURE
// =============================================================================

// MainConfig holds the global application configuration.
// This is loaded from the main config.yaml file.
type MainConfig struct {
	// =========================================================================
	// DIRECTORY SETTINGS
	// =========================================================================

	// InputDir is the directory where input CSV files are placed.
	// The application will scan this directory for files to process.
	// Default: "./input"
	InputDir string `yaml:"input_dir"`

	// OutputDir is the directory where generated XML files are placed.
	// Default: "./output"
	OutputDir string `yaml:"output_dir"`

	// InputArchiveDir is the directory where processed CSV files are moved.
	// Files are only moved here after successful processing.
	// Default: "./input_archive"
	InputArchiveDir string `yaml:"input_archive_dir"`

	// OutputArchiveDir is the directory where generated XML files are archived.
	// This is for long-term storage of successfully processed files.
	// Default: "./output_archive"
	OutputArchiveDir string `yaml:"output_archive_dir"`

	// TemplatesDir is the directory containing XLSX schema templates.
	// Each template defines the structure for a specific transaction type.
	// Default: "./templates"
	TemplatesDir string `yaml:"templates_dir"`

	// ConfigsDir is the directory containing department-specific configurations.
	// Each YAML file in this directory represents a department's rules.
	// Default: "./configs"
	ConfigsDir string `yaml:"configs_dir"`

	// =========================================================================
	// LOGGING SETTINGS
	// =========================================================================

	// LogFile is the path to the application log file.
	// Default: "./logs/converter.log"
	LogFile string `yaml:"log_file"`

	// LogLevel controls the verbosity of logging.
	// Valid values: "debug", "info", "warn", "error"
	// Default: "info"
	LogLevel string `yaml:"log_level"`

	// =========================================================================
	// OUTPUT SETTINGS
	// =========================================================================

	// UUIDFormat defines the format for output file names.
	// Placeholders:
	//   {uuid}      - A random UUID
	//   {timestamp} - Current timestamp (YYYYMMDD_HHMMSS)
	//   {dept}      - Department code
	//   {type}      - Transaction type
	//
	// CUSTOMIZATION: Define your desired format here.
	// Example: "{dept}_{type}_{timestamp}_{uuid}.xml"
	// Default: "{uuid}.xml"
	UUIDFormat string `yaml:"uuid_format"`

	// =========================================================================
	// PROCESSING SETTINGS
	// =========================================================================

	// MaxConcurrency is the maximum number of files to process concurrently.
	// Set to 1 for sequential processing.
	// Default: 4
	MaxConcurrency int `yaml:"max_concurrency"`

	// ContinueOnError determines whether to continue processing other files
	// if one file fails.
	// Default: true
	ContinueOnError bool `yaml:"continue_on_error"`
}

// =============================================================================
// DEPARTMENT CONFIGURATION STRUCTURE
// =============================================================================

// DepartmentConfig holds the configuration for a specific department.
// Each department can have its own rules for file matching, CSV parsing,
// field transformations, and validation.
type DepartmentConfig struct {
	// =========================================================================
	// DEPARTMENT IDENTIFICATION
	// =========================================================================

	// DepartmentName is the human-readable name of the department.
	// This is used in logs and error messages.
	DepartmentName string `yaml:"department_name"`

	// DepartmentCode is a short code for the department.
	// This can be used in output file names and XML tags.
	DepartmentCode string `yaml:"department_code"`

	// =========================================================================
	// FILE MATCHING RULES
	// =========================================================================

	// FileMatchingPatterns is a list of glob patterns to match input files.
	// If a file name matches any of these patterns, this configuration is used.
	//
	// CUSTOMIZATION: Add patterns that match your department's file naming convention.
	// Examples:
	//   - "dept_a_*.csv"           : Matches files starting with "dept_a_"
	//   - "payments_*.csv"         : Matches files starting with "payments_"
	//   - "*_claims_*.csv"         : Matches files containing "_claims_"
	FileMatchingPatterns []string `yaml:"file_matching_patterns"`

	// =========================================================================
	// CSV PARSING SETTINGS
	// =========================================================================

	// CSVSettings contains settings for parsing the input CSV file.
	CSVSettings CSVSettings `yaml:"csv_settings"`

	// =========================================================================
	// TEMPLATE MAPPING
	// =========================================================================

	// TemplateMapping defines rules for selecting the XLSX template.
	// The first matching rule is used.
	//
	// CUSTOMIZATION: Define rules based on your file naming conventions.
	TemplateMapping []TemplateRule `yaml:"template_mapping"`

	// =========================================================================
	// TRANSFORMATION RULES
	// =========================================================================

	// TransformationRules defines field-level transformation rules.
	// These are applied to the data before validation and XML generation.
	//
	// CUSTOMIZATION: Define your department-specific transformation rules here.
	TransformationRules []TransformationRule `yaml:"transformation_rules"`

	// =========================================================================
	// TRANSACTION GROUPING
	// =========================================================================

	// TransactionGrouping defines how CSV rows are grouped into transactions.
	TransactionGrouping TransactionGrouping `yaml:"transaction_grouping"`

	// =========================================================================
	// STATIC FIELDS
	// =========================================================================

	// StaticFields are fields with constant values added to every transaction.
	// These are not derived from the input CSV.
	//
	// CUSTOMIZATION: Add any fields that are constant for this department.
	StaticFields []StaticField `yaml:"static_fields"`
}

// =============================================================================
// CSV SETTINGS STRUCTURE
// =============================================================================

// CSVSettings contains settings for parsing CSV files.
type CSVSettings struct {
	// Delimiter is the character used to separate fields in the CSV.
	// Common values: "," (comma), "|" (pipe), "\t" (tab)
	// Default: ","
	Delimiter string `yaml:"delimiter"`

	// HeaderRows is the number of header rows in the CSV file.
	// These rows are used to identify column names.
	// Default: 1
	//
	// CUSTOMIZATION: Some departments have multi-line headers.
	// Set this to the number of header rows in your CSV.
	HeaderRows int `yaml:"header_rows"`

	// DataStartRow is the row number where the actual data begins.
	// Row numbering starts at 1.
	// Default: 2 (assuming 1 header row)
	//
	// CUSTOMIZATION: If your CSV has additional metadata rows before the data,
	// set this to the row number where the data starts.
	// Example: If headers are on rows 1-3 and data starts on row 4, set to 4.
	DataStartRow int `yaml:"data_start_row"`

	// Encoding is the character encoding of the CSV file.
	// Common values: "UTF-8", "ISO-8859-1", "Windows-1252"
	// Default: "UTF-8"
	Encoding string `yaml:"encoding"`

	// QuoteChar is the character used to quote fields containing special characters.
	// Default: '"'
	QuoteChar string `yaml:"quote_char"`

	// EscapeChar is the character used to escape special characters.
	// Default: '"' (double quote to escape a quote)
	EscapeChar string `yaml:"escape_char"`
}

// =============================================================================
// TEMPLATE RULE STRUCTURE
// =============================================================================

// TemplateRule defines a rule for selecting an XLSX template based on the file name.
type TemplateRule struct {
	// IfFilenameContains is a substring to match in the file name.
	// If the file name contains this substring, this rule matches.
	//
	// CUSTOMIZATION: Define substrings that identify the transaction type.
	// Examples: "payments", "receipts", "clt", "ach"
	IfFilenameContains string `yaml:"if_filename_contains"`

	// UseTemplate is the name of the XLSX template file to use.
	// This file should be located in the templates directory.
	//
	// CUSTOMIZATION: Specify the template file for each transaction type.
	UseTemplate string `yaml:"use_template"`
}

// =============================================================================
// TRANSFORMATION RULE STRUCTURE
// =============================================================================

// TransformationRule defines a transformation to apply to a specific field.
type TransformationRule struct {
	// Field is the name of the field to transform.
	// This should match the column header in the input CSV.
	//
	// CUSTOMIZATION: Use the exact column header from your CSV.
	Field string `yaml:"field"`

	// Actions is a list of transformations to apply to this field.
	// Actions are applied in order.
	Actions []TransformationAction `yaml:"actions"`
}

// TransformationAction defines a single transformation action.
type TransformationAction struct {
	// Type is the type of transformation to apply.
	// Supported types:
	//   - "prepend_string"      : Add a string to the beginning of the value
	//   - "append_string"       : Add a string to the end of the value
	//   - "pad_zeros_to_length" : Pad with leading zeros to a specific length
	//   - "ensure_length"       : Truncate or pad to ensure a specific length
	//   - "uppercase"           : Convert to uppercase
	//   - "lowercase"           : Convert to lowercase
	//   - "trim"                : Remove leading and trailing whitespace
	//   - "replace"             : Replace a substring with another
	//   - "regex_replace"       : Replace using a regular expression
	//   - "format_date"         : Convert date format
	//   - "format_number"       : Format a number (decimal places, thousands separator)
	//   - "lookup"              : Replace value using a lookup table
	//   - "conditional"         : Apply transformation based on a condition
	//
	// CUSTOMIZATION: Add new transformation types as needed.
	Type string `yaml:"type"`

	// Value is the parameter for the transformation.
	// The meaning depends on the transformation type:
	//   - "prepend_string"      : The string to prepend
	//   - "append_string"       : The string to append
	//   - "pad_zeros_to_length" : The target length (as a string, e.g., "12")
	//   - "ensure_length"       : The target length (as a string, e.g., "10")
	//   - "replace"             : The replacement string
	//   - "format_date"         : The target date format (e.g., "2006-01-02")
	//   - "format_number"       : The number format (e.g., "2" for 2 decimal places)
	Value string `yaml:"value"`

	// Find is used for "replace" and "regex_replace" transformations.
	// It specifies the substring or pattern to find.
	Find string `yaml:"find,omitempty"`

	// Condition is used for "conditional" transformations.
	// It specifies when the transformation should be applied.
	//
	// CUSTOMIZATION: Define conditions using a simple expression language.
	// Examples:
	//   - "value == 'ABC'"
	//   - "length > 10"
	//   - "starts_with 'P'"
	Condition string `yaml:"condition,omitempty"`

	// LookupTable is used for "lookup" transformations.
	// It maps input values to output values.
	//
	// CUSTOMIZATION: Define your lookup mappings here.
	// Example:
	//   lookup_table:
	//     "01": "January"
	//     "02": "February"
	LookupTable map[string]string `yaml:"lookup_table,omitempty"`
}

// =============================================================================
// TRANSACTION GROUPING STRUCTURE
// =============================================================================

// TransactionGrouping defines how CSV rows are grouped into transactions.
type TransactionGrouping struct {
	// GroupByField is the name of the field used to group rows.
	// All rows with the same value in this field belong to the same transaction.
	//
	// CUSTOMIZATION: Specify the field that uniquely identifies a transaction.
	// Examples: "check_number", "batch_id", "transaction_id"
	//
	// QUESTION FOR USER: What field in your CSV identifies which rows belong
	// to the same transaction? This could be a check number, batch ID, or
	// any other unique identifier.
	GroupByField string `yaml:"group_by_field"`

	// SortByField is an optional field to sort rows within a transaction.
	// This ensures consistent ordering of line items.
	SortByField string `yaml:"sort_by_field,omitempty"`

	// SortOrder is the order for sorting: "asc" or "desc".
	// Default: "asc"
	SortOrder string `yaml:"sort_order,omitempty"`
}

// =============================================================================
// STATIC FIELD STRUCTURE
// =============================================================================

// StaticField defines a field with a constant value.
type StaticField struct {
	// XMLTag is the name of the XML element to create.
	XMLTag string `yaml:"xml_tag"`

	// Value is the constant value for this field.
	Value string `yaml:"value"`

	// ParentTag specifies where this field should be placed in the XML.
	// Options: "cashbook", "transaction", "lineItem"
	// Default: "transaction"
	ParentTag string `yaml:"parent_tag,omitempty"`
}

// =============================================================================
// CONFIGURATION LOADING FUNCTIONS
// =============================================================================

// LoadMainConfig loads the main configuration from a YAML file.
//
// PARAMETERS:
//   - configPath: The path to the main configuration file.
//
// RETURNS:
//   - A pointer to the MainConfig struct.
//   - An error if the file cannot be read or parsed.
//
// CUSTOMIZATION:
//   - Add default values for any new configuration options.
//   - Add validation for required fields.
func LoadMainConfig(configPath string) (*MainConfig, error) {
	// Read the configuration file.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML.
	var config MainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply default values.
	applyMainConfigDefaults(&config)

	// Validate the configuration.
	if err := validateMainConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// applyMainConfigDefaults sets default values for any unset configuration options.
func applyMainConfigDefaults(config *MainConfig) {
	if config.InputDir == "" {
		config.InputDir = "./input"
	}
	if config.OutputDir == "" {
		config.OutputDir = "./output"
	}
	if config.InputArchiveDir == "" {
		config.InputArchiveDir = "./input_archive"
	}
	if config.OutputArchiveDir == "" {
		config.OutputArchiveDir = "./output_archive"
	}
	if config.TemplatesDir == "" {
		config.TemplatesDir = "./templates"
	}
	if config.ConfigsDir == "" {
		config.ConfigsDir = "./configs"
	}
	if config.LogFile == "" {
		config.LogFile = "./logs/converter.log"
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.UUIDFormat == "" {
		config.UUIDFormat = "{uuid}.xml"
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 4
	}
}

// validateMainConfig validates the main configuration.
func validateMainConfig(config *MainConfig) error {
	// Validate that required directories exist.
	dirs := []string{
		config.InputDir,
		config.OutputDir,
		config.TemplatesDir,
		config.ConfigsDir,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Create the directory if it doesn't exist.
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	return nil
}

// LoadDepartmentConfigs loads all department configurations from a directory.
//
// PARAMETERS:
//   - configsDir: The path to the directory containing department configuration files.
//
// RETURNS:
//   - A map of department configurations, keyed by department code.
//   - An error if the directory cannot be read or any file cannot be parsed.
//
// CUSTOMIZATION:
//   - Add validation for department-specific required fields.
//   - Add support for inheritance (e.g., a base configuration that others extend).
func LoadDepartmentConfigs(configsDir string) (map[string]*DepartmentConfig, error) {
	configs := make(map[string]*DepartmentConfig)

	// Find all YAML files in the configs directory.
	files, err := filepath.Glob(filepath.Join(configsDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list config files: %w", err)
	}

	// Also check for .yml extension.
	ymlFiles, err := filepath.Glob(filepath.Join(configsDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list config files: %w", err)
	}
	files = append(files, ymlFiles...)

	// Load each configuration file.
	for _, file := range files {
		config, err := loadDepartmentConfig(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}

		// Use department code as the key.
		// If no code is specified, use the file name.
		key := config.DepartmentCode
		if key == "" {
			key = filepath.Base(file)
		}

		configs[key] = config
	}

	return configs, nil
}

// loadDepartmentConfig loads a single department configuration file.
func loadDepartmentConfig(filePath string) (*DepartmentConfig, error) {
	// Read the configuration file.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the YAML.
	var config DepartmentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Apply default values.
	applyDepartmentConfigDefaults(&config)

	return &config, nil
}

// applyDepartmentConfigDefaults sets default values for department configuration.
func applyDepartmentConfigDefaults(config *DepartmentConfig) {
	// CSV settings defaults.
	if config.CSVSettings.Delimiter == "" {
		config.CSVSettings.Delimiter = ","
	}
	if config.CSVSettings.HeaderRows == 0 {
		config.CSVSettings.HeaderRows = 1
	}
	if config.CSVSettings.DataStartRow == 0 {
		config.CSVSettings.DataStartRow = 2
	}
	if config.CSVSettings.Encoding == "" {
		config.CSVSettings.Encoding = "UTF-8"
	}
	if config.CSVSettings.QuoteChar == "" {
		config.CSVSettings.QuoteChar = "\""
	}
	if config.CSVSettings.EscapeChar == "" {
		config.CSVSettings.EscapeChar = "\""
	}

	// Transaction grouping defaults.
	if config.TransactionGrouping.SortOrder == "" {
		config.TransactionGrouping.SortOrder = "asc"
	}

	// Static fields defaults.
	for i := range config.StaticFields {
		if config.StaticFields[i].ParentTag == "" {
			config.StaticFields[i].ParentTag = "transaction"
		}
	}
}
