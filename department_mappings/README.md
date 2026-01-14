# Department Mappings Directory

This directory contains department-specific configuration files. Each department has its own subdirectory with a `department_config.yaml` file that defines:

- CSV parsing settings specific to that department's output format
- Static field values (constants for all transactions)
- Field mappings (how CSV columns map to XML fields)
- Transformation rules (how to convert field values)
- Lookup tables (code-to-value translations)

## Directory Structure

```
department_mappings/
├── README.md                           # This file
├── claims/
│   └── department_config.yaml          # Claims department configuration
├── underwriting/
│   └── department_config.yaml          # Underwriting department configuration
├── accounting/
│   └── department_config.yaml          # Accounting department configuration
└── operations/
    └── department_config.yaml          # Operations department configuration
```

## Creating a New Department Configuration

1. Create a new subdirectory with the department code as the name:
   ```
   mkdir department_mappings/new_department
   ```

2. Copy an existing configuration file as a template:
   ```
   cp department_mappings/claims/department_config.yaml department_mappings/new_department/
   ```

3. Edit the configuration file to match the new department's requirements.

## Configuration File Structure

### Department Identification

```yaml
department:
  code: "DEPT_CODE"           # Unique identifier (used in file naming)
  name: "Department Name"     # Human-readable name
  description: "Description"  # What this department does
  contact_email: "email"      # Contact for error notifications
```

### CSV Settings

```yaml
csv_settings:
  delimiter: ","              # Field delimiter
  quote_char: "\""            # Quote character
  header_rows: 1              # Number of header rows
  data_start_row: 2           # Row where data begins (1-indexed)
  encoding: "UTF-8"           # File encoding
  skip_empty_rows: true       # Skip blank rows
```

### Transaction Grouping

```yaml
grouping:
  group_by_field: "CheckNumber"   # Field that groups rows into transactions
  sort_by_field: "LineNumber"     # Field to sort line items by
  sort_direction: "asc"           # Sort direction (asc/desc)
```

### Static Fields

Static fields have constant values for all transactions from this department:

```yaml
static_fields:
  - xml_tag: "DepartmentCode"
    value: "CLAIMS"
    parent_tag: "transaction"
```

### Transformation Rules

Transformation rules define how to convert field values:

```yaml
transformation_rules:
  - field: "POLICY_NO"
    actions:
      - type: "pad_zeros_to_length"
        value: "9"
      - type: "prepend_string"
        value: "A"
```

## Available Transformation Types

### String Manipulations

| Type | Description | Value |
|------|-------------|-------|
| `prepend_string` | Add text to the beginning | Text to prepend |
| `append_string` | Add text to the end | Text to append |
| `trim` | Remove leading/trailing whitespace | - |
| `uppercase` | Convert to uppercase | - |
| `lowercase` | Convert to lowercase | - |
| `replace` | Replace substring | `find` and `value` |

### Numeric Formatting

| Type | Description | Value |
|------|-------------|-------|
| `pad_zeros_to_length` | Pad with leading zeros | Target length |
| `ensure_length` | Truncate or pad to length | Target length |
| `format_number` | Format decimal places | Number of decimals |
| `remove_leading_zeros` | Remove leading zeros | - |

### Date/Time

| Type | Description | Value |
|------|-------------|-------|
| `format_date` | Convert date format | `input_format\|output_format` |

### Lookup Tables

| Type | Description | Value |
|------|-------------|-------|
| `lookup` | Replace using lookup table | - |
| `lookup_with_default` | Lookup with default value | Default value |

### Special

| Type | Description | Value |
|------|-------------|-------|
| `extract_digits` | Extract only digits | - |
| `extract_letters` | Extract only letters | - |
| `remove_special_chars` | Remove non-alphanumeric | - |
| `if_empty_use_default` | Default for empty values | Default value |

## Policy Number Formatting Examples

### Example 1: Prepend letter and pad to 10 digits

CSV value: `123456`
Required: `A000123456`

```yaml
transformation_rules:
  - field: "POLICY_NO"
    actions:
      - type: "extract_digits"
      - type: "pad_zeros_to_length"
        value: "9"
      - type: "prepend_string"
        value: "A"
```

### Example 2: Fixed 12-digit format with prefix

CSV value: `789`
Required: `POL000000789`

```yaml
transformation_rules:
  - field: "POLICY_NO"
    actions:
      - type: "extract_digits"
      - type: "pad_zeros_to_length"
        value: "9"
      - type: "prepend_string"
        value: "POL"
```

### Example 3: Different prefix based on department

CSV value: `456789`
Claims: `C000456789`
Underwriting: `U000456789`

Configure in each department's `department_config.yaml`:

```yaml
# In claims/department_config.yaml
transformation_rules:
  - field: "POLICY_NO"
    actions:
      - type: "pad_zeros_to_length"
        value: "9"
      - type: "prepend_string"
        value: "C"

# In underwriting/department_config.yaml
transformation_rules:
  - field: "POLICY_NO"
    actions:
      - type: "pad_zeros_to_length"
        value: "9"
      - type: "prepend_string"
        value: "U"
```

## Handling Multi-Line Headers

For CSVs with multi-line headers or data starting on row 8+:

```yaml
csv_settings:
  header_rows: 3              # Number of header rows to merge
  data_start_row: 8           # Data starts on row 8
```

The converter will merge the header rows and skip to the data start row.
