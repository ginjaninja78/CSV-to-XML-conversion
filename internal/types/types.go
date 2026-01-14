// =============================================================================
// CSV to XML Converter - Shared Types
// =============================================================================
//
// This package contains shared types used across multiple modules to avoid
// import cycles. Types defined here are used by:
//   - converter
//   - validation
//   - xmlwriter
//
// =============================================================================

package types

// =============================================================================
// TRANSACTION TYPES
// =============================================================================

// Transaction represents a single transaction in the XML output.
// A transaction contains one or more line items.
type Transaction struct {
	// ID is the transaction number (1-indexed).
	ID int

	// LineItems contains all line items belonging to this transaction.
	LineItems []LineItem

	// GroupKey is the value used to group rows into this transaction.
	// For example, if grouping by CheckNumber, this would be the check number.
	GroupKey string
}

// LineItem represents a single line item within a transaction.
type LineItem struct {
	// ID is the line item number.
	// If using global numbering, this is the global index.
	// If using per-transaction numbering, this is the index within the transaction.
	ID int

	// Fields contains the field values for this line item.
	// Key is the old system header name, value is the (possibly transformed) value.
	Fields map[string]string

	// OriginalRowNumber is the row number in the original CSV file.
	// Useful for error reporting.
	OriginalRowNumber int
}
