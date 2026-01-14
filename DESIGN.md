# CSV to XML Converter: Design Document

This document outlines the design and architecture of the Go-based CLI tool for converting CSV files to XML. It is designed to be highly configurable and extensible, allowing for proprietary business logic to be added without modifying the core application code.

## 1. Core Architecture

The application follows a modular, configuration-driven architecture. The core logic is separated from the business-specific rules, which are defined in external YAML and XLSX files.

### 1.1. Folder Structure

The application will be organized with the following folder structure:

```
/CSV-to-XML-conversion/
|-- cmd/                    # Cobra CLI command definitions
|   |-- root.go
|   +-- process.go
|-- internal/               # Core application logic (not for external use)
|   |-- config/
|   |-- converter/
|   |-- csvparser/
|   |-- validation/
|   +-- xmlwriter/
|-- pkg/                    # Shared libraries and utilities
|   +-- utils/
|-- configs/                # Department-specific configurations
|   |-- department_a.yaml
|   +-- department_b.yaml
|-- templates/              # XLSX schema definition templates
|   |-- payments.xlsx
|   +-- receipts.xlsx
|-- testdata/               # Sample data for testing
|-- .gitignore
|-- go.mod
|-- go.sum
|-- main.go
|-- DESIGN.md
+-- README.md
```

### 1.2. Data Flow

1.  **Initialization**: The user executes the CLI command (`converter process`).
2.  **Configuration Loading**: The application loads the main `config.yaml` and all department-specific configurations from the `/configs` directory.
3.  **File Discovery**: The application scans the `input` directory for new CSV files.
4.  **Template Matching**: For each CSV, it determines the appropriate XLSX template to use based on rules defined in the department configuration.
5.  **Schema Parsing**: The XLSX template is parsed to extract the CSV-to-XML mapping, validation rules, and nesting structure.
6.  **CSV Parsing**: The input CSV is parsed. The application handles multi-line headers and finds the data start row based on department-specific configuration.
7.  **Data Transformation**: The application applies department-specific transformation rules (e.g., zero-padding, prepending strings) to the data.
8.  **Validation**: The transformed data is validated against the rules extracted from the XLSX template (char limits, format, required/optional).
9.  **XML Generation**: A new XML document is constructed in memory based on the defined nesting structure.
10. **File Output**: The generated XML is written to the `output` directory with a UUID-based filename.
11. **Archival**: On successful completion, the processed CSV is moved to `input_archive` and the generated XML is moved to `output_archive`.
12. **Error Handling**: If any step fails, an error log is created in the `output` directory, and the original files are not moved.

## 2. Configuration Files

### 2.1. Main Configuration (`config.yaml`)

This file will contain global settings for the application.

```yaml
# config.yaml

# Directory settings
input_dir: "./input"
output_dir: "./output"
input_archive_dir: "./input_archive"
output_archive_dir: "./output_archive"
templates_dir: "./templates"
configs_dir: "./configs"

# Logging settings
log_file: "./logs/converter.log"
log_level: "info" # Can be debug, info, warn, error

# UUID format for output files
# Placeholder: You can define your desired UUID format here.
# Example: "{timestamp}_{uuid}"
uuid_format: "{uuid}"
```

### 2.2. Department Configuration (`/configs/department_a.yaml`)

Each department will have its own YAML file defining its specific rules.

```yaml
# /configs/department_a.yaml

department_name: "Department A"

# Rules to match input CSV files to this configuration
file_matching_patterns:
  - "dept_a_*.csv"
  - "payments_dept_a_*.csv"

# CSV parsing settings
csv_settings:
  delimiter: "," # or "|", "\t", etc.
  header_rows: 1 # Number of header rows to skip
  data_start_row: 2 # The row number where the actual data begins

# Template mapping rules
# How to select the XLSX template for a given CSV
template_mapping:
  - if_filename_contains: "payments"
    use_template: "payments.xlsx"
  - if_filename_contains: "receipts"
    use_template: "receipts.xlsx"

# Field transformation rules
# This is where you define your complex, department-specific logic
transformation_rules:
  - field: "policy_number" # The header from the *input CSV*
    actions:
      - type: "prepend_string"
        value: "A"
      - type: "pad_zeros_to_length"
        value: 12
  - field: "account_id"
    actions:
      - type: "ensure_length"
        value: 10

# Transaction grouping logic
# Defines which CSV column is used to group rows into a single <transaction>
transaction_grouping:
  group_by_field: "check_number"

# Static fields to be added to every transaction
static_fields:
  - xml_tag: "DepartmentCode"
    value: "DEPT-A"
  - xml_tag: "SourceSystem"
    value: "LegacySystemV1"
```

## 3. XLSX Template Structure

This is a pseudocode representation of your XLSX templates. The application will be built to parse this structure.

| Old System Header | XML Tag Name | Parent Tag | Data Type | Max Length | Required | Conditional Rule |
|---|---|---|---|---|---|---|
| `CHK_NUM` | `CheckNumber` | `transaction` | `numeric` | `10` | `yes` | | 
| `CHK_AMT` | `CheckAmount` | `transaction` | `decimal(2)` | `15` | `yes` | | 
| `POL_NUM` | `PolicyNumber` | `lineItem` | `alphanumeric` | `12` | `yes` | | 
| `INV_NUM` | `InvoiceNumber` | `lineItem` | `alphanumeric` | `20` | `no` | | 
| `PAY_REASON` | `PaymentReason` | `lineItem` | `string` | `50` | `conditional` | `if CheckAmount > 10000` |

## 4. Pseudocode for Core Logic

This section provides a high-level overview of the Go functions that will be implemented.

### `main.go`

```go
// main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	// Initialize Cobra CLI
	var rootCmd = &cobra.Command{Use: "converter"}

	// Add the 'process' command
	var cmdProcess = &cobra.Command{
		Use:   "process",
		Short: "Process CSV files and convert them to XML",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. Load main config
			// 2. Load all department configs
			// 3. Discover files in the input directory
			// 4. For each file, find the matching department config
			// 5. Create a new Converter instance and run the conversion
			// 6. Handle concurrency (process multiple files at once)
		},
	}

	rootCmd.AddCommand(cmdProcess)
	rootCmd.Execute()
}
```

### `internal/converter/converter.go`

```go
// internal/converter/converter.go
package converter

// Run starts the conversion process for a single file
func (c *Converter) Run() error {
	// 1. Parse the XLSX template to get the schema
	// schema, err := xlsxparser.Parse(c.templatePath)

	// 2. Parse the input CSV
	// csvData, err := csvparser.Parse(c.csvPath, c.departmentConfig.CSVSettings)

	// 3. Group CSV rows into transactions
	// transactions := groupTransactions(csvData, c.departmentConfig.TransactionGrouping)

	// 4. For each transaction and its line items:
	//    a. Apply transformation rules
	//    b. Validate the data against the schema

	// 5. Generate the XML structure
	// xmlDoc, err := xmlwriter.Generate(transactions, schema)

	// 6. Write the XML file to the output directory

	// 7. Archive files on success

	return nil
}
```

This design provides a solid foundation for building the converter. The next steps will be to implement this structure in Go, creating the files and directories as outlined.
