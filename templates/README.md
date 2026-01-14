# Templates Directory

This directory contains the XLSX template files that define the schema for each transaction type. These templates are the **source of truth** for field mappings, validation rules, and XML structure.

## Template Structure

Each template file should be an XLSX file with the following column structure:

| Column | Header Name | Description | Example |
|--------|-------------|-------------|---------|
| A | Old Header | The column header from the legacy CSV system | `CHECK_NUM` |
| B | XML Tag | The corresponding XML element name | `CheckNumber` |
| C | Data Type | The expected data type | `numeric`, `alphanumeric`, `date` |
| D | Max Length | Maximum character length | `10` |
| E | Required Type | Whether the field is required | `required`, `optional`, `conditional` |
| F | Conditional Rule | The condition (if Required Type is "conditional") | `if PaymentType == 'CHECK'` |
| G | Parent Element | The parent XML element | `cashbook`, `transaction`, `lineItem` |
| H | Field Order | The order of this field within its parent | `1`, `2`, `3` |
| I | Notes | Any additional notes or comments | `Must be unique per batch` |

## Template Files

Create one template file for each transaction type:

```
templates/
├── payments_template.xlsx
├── receipts_template.xlsx
├── clt_template.xlsx
└── ach_eft_wire_template.xlsx
```

## Data Types

The following data types are supported:

| Data Type | Description | Validation |
|-----------|-------------|------------|
| `string` | Any text value | No validation |
| `numeric` | Integer numbers only | Must be a valid integer |
| `decimal` | Decimal numbers | Must be a valid decimal |
| `decimal(2)` | Decimal with 2 places | Max 2 decimal places |
| `alphanumeric` | Letters and numbers only | No special characters |
| `alpha` | Letters only | No numbers or special characters |
| `date` | Date value | Must be a valid date |
| `date(MM/DD/YYYY)` | Date with specific format | Must match format |
| `boolean` | True/false value | true, false, yes, no, 1, 0 |

## Required Types

| Type | Description |
|------|-------------|
| `required` | Field must have a value |
| `optional` | Field may be empty |
| `conditional` | Field is required based on a condition |

## Conditional Rules

Conditional rules use the following syntax:

```
if FieldName == 'value'
if FieldName != 'value'
if FieldName > 100
if FieldName < 100
if FieldName starts_with 'prefix'
if FieldName is_empty
if FieldName is_not_empty
```

## Parent Elements

| Parent | Description |
|--------|-------------|
| `cashbook` | Root-level field (appears once per document) |
| `transaction` | Transaction-level field (appears once per transaction) |
| `lineItem` | Line item-level field (appears for each line item) |

## Example Template Row

| Old Header | XML Tag | Data Type | Max Length | Required Type | Conditional Rule | Parent Element | Field Order | Notes |
|------------|---------|-----------|------------|---------------|------------------|----------------|-------------|-------|
| CHECK_NUM | CheckNumber | numeric | 10 | required | | transaction | 1 | |
| CHECK_AMT | CheckAmount | decimal(2) | 15 | required | | transaction | 2 | |
| POLICY_NO | PolicyNumber | alphanumeric | 12 | required | | lineItem | 1 | |
| INVOICE_NO | InvoiceNumber | alphanumeric | 20 | conditional | if PaymentType == 'INVOICE' | lineItem | 2 | |

## Updating Templates

When you update a template file:

1. The converter will automatically detect the changes on the next run (if `auto_reload` is enabled in the config).
2. The XSD schema will be regenerated based on the new template.
3. Existing department configurations may need to be updated if field names change.

## Best Practices

1. **Keep templates in sync**: Ensure all transaction type templates follow the same column structure.
2. **Document changes**: Add notes in the Notes column when making changes.
3. **Version control**: Keep templates under version control to track changes.
4. **Test after changes**: Run a test conversion after updating templates.
