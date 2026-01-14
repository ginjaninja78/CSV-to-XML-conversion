# CSV to XML Converter

A high-performance, modular CSV to XML converter built in Go with Cobra CLI. Designed for financial transaction processing with support for multiple transaction types, department-specific configurations, and robust validation.

## Features

- **XLSX-Based Schema Templates**: Define your XML structure and validation rules in Excel files
- **Department-Specific Mappings**: Each department can have its own CSV format and transformation rules
- **Four Transaction Types**: Payments, Receipts, CLT (Cash Ledger Transactions), ACH/EFT/Wires
- **Conditional Transformations**: Apply transformations based on field values
- **Policy Number Formatting**: Prepend letters, pad with zeros, enforce fixed lengths
- **Robust Validation**: Character limits, data types, required fields, conditional requirements
- **File Archival**: Automatic archival of processed files
- **Lightweight & Fast**: Built in Go for maximum performance and minimal resource usage
- **Easy to Use**: Drop CSV files in a folder, click a batch file, get XML output

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/ginjaninja78/CSV-to-XML-conversion.git
cd CSV-to-XML-conversion
```

### 2. Build the Application

```bash
# Install dependencies
go mod tidy

# Build for your platform
go build -o csv2xml .

# Or build for Windows
GOOS=windows GOARCH=amd64 go build -o csv2xml.exe .
```

### 3. Configure Your Templates

1. Create your XLSX template files in the `templates/` directory
2. Configure your department mappings in `department_mappings/<dept>/department_config.yaml`
3. Update `config/app_config.yaml` with your settings

### 4. Run the Converter

**Windows:**
```batch
run_converter.bat
```

**Linux/macOS:**
```bash
chmod +x run_converter.sh
./run_converter.sh
```

**CLI:**
```bash
./csv2xml process --verbose
```

## Directory Structure

```
CSV-to-XML-conversion/
├── cmd/                          # Cobra CLI commands
│   ├── root.go                   # Root command
│   ├── process.go                # Process command
│   └── version.go                # Version command
├── config/                       # Application configuration
│   └── app_config.yaml           # Main configuration file
├── department_mappings/          # Department-specific configurations
│   ├── README.md                 # Configuration guide
│   └── claims/                   # Example department
│       └── department_config.yaml
├── input/                        # Place CSV files here
├── input_archive/                # Processed CSV files archived here
├── internal/                     # Internal packages
│   ├── config/                   # Configuration loader
│   ├── converter/                # Main conversion logic
│   ├── csvparser/                # CSV parsing
│   ├── validation/               # Validation engine
│   └── xmlwriter/                # XML generation
├── logs/                         # Application logs
├── output/                       # Generated XML files
├── output_archive/               # Archived XML files
├── pkg/                          # Public packages
│   └── utils/                    # Utility functions
├── templates/                    # XLSX template files
│   └── README.md                 # Template structure guide
├── testdata/                     # Test data files
├── run_converter.bat             # Windows batch file
├── run_converter.sh              # Linux/macOS script
├── main.go                       # Application entry point
├── go.mod                        # Go module file
├── DESIGN.md                     # Detailed design document
└── README.md                     # This file
```

## Configuration

### Application Configuration (`config/app_config.yaml`)

The main configuration file controls:
- Directory paths
- Logging settings
- Processing options
- Validation behavior
- Output formatting
- Template column positions
- Transaction type definitions

### Department Configuration (`department_mappings/<dept>/department_config.yaml`)

Each department has its own configuration file that defines:
- CSV parsing settings (delimiter, headers, encoding)
- Transaction grouping (which field groups rows into transactions)
- Static fields (constant values for all transactions)
- Field mappings (CSV column to XML tag)
- Transformation rules (how to convert values)
- Lookup tables (code-to-value translations)

### XLSX Templates (`templates/`)

Template files define the schema for each transaction type:
- Old system header → XML tag mapping
- Data types and validation rules
- Required/optional/conditional fields
- Field ordering within parent elements

See `templates/README.md` for detailed template structure.

## CLI Commands

```bash
# Process all CSV files in the input directory
./csv2xml process

# Process with verbose output
./csv2xml process --verbose

# Process specific department
./csv2xml process --department claims

# Process specific transaction type
./csv2xml process --type payments

# Dry run (validate without generating output)
./csv2xml process --dry-run

# Use custom configuration file
./csv2xml process --config /path/to/config.yaml

# Show version
./csv2xml version

# Show help
./csv2xml --help
./csv2xml process --help
```

## Transformation Rules

The converter supports various transformation types:

### String Manipulations
- `prepend_string`: Add text to the beginning
- `append_string`: Add text to the end
- `trim`: Remove whitespace
- `uppercase` / `lowercase`: Case conversion
- `replace`: Replace substrings

### Numeric Formatting
- `pad_zeros_to_length`: Pad with leading zeros
- `ensure_length`: Truncate or pad to fixed length
- `format_number`: Format decimal places

### Date/Time
- `format_date`: Convert between date formats

### Special
- `extract_digits`: Extract only numeric characters
- `lookup`: Replace using lookup table
- `if_empty_use_default`: Provide default for empty values

## XML Output Structure

The generated XML follows this nesting pattern:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<cashbook>
  <transaction n="1">
    <CheckNumber>12345</CheckNumber>
    <CheckAmount>1000.00</CheckAmount>
    <lineItem n="1">
      <PolicyNumber>A000123456</PolicyNumber>
      <InvoiceNumber>INV-001</InvoiceNumber>
    </lineItem>
    <lineItem n="2">
      <PolicyNumber>A000123457</PolicyNumber>
      <InvoiceNumber>INV-002</InvoiceNumber>
    </lineItem>
  </transaction>
  <transaction n="2">
    <lineItem n="3">
      <!-- Line item numbering is global -->
    </lineItem>
  </transaction>
</cashbook>
```

## Customization

This codebase is designed for easy customization:

1. **All business logic is externalized** into YAML configuration files
2. **All field names and mappings are placeholders** with clear instructions
3. **Every function has detailed comments** explaining what to customize
4. **Modular architecture** allows adding new departments/transaction types without code changes

Look for `CUSTOMIZATION:` and `QUESTION FOR USER:` comments in the code for guidance.

## Building for Production

```bash
# Build with optimizations
go build -ldflags="-s -w" -o csv2xml .

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o csv2xml.exe .

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o csv2xml-linux .

# Cross-compile for macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o csv2xml-darwin .
```

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [Excelize](https://github.com/qax-os/excelize) - XLSX file parsing
- [UUID](https://github.com/google/uuid) - UUID generation

Install dependencies:
```bash
go mod tidy
```

## Error Handling

- Validation errors are collected and reported in detail
- Error logs are generated in the output directory
- Processing summaries show success/failure statistics
- Failed files remain in the input directory for review

## License

This project is proprietary. All rights reserved.

## Support

For questions or issues, contact the development team.
