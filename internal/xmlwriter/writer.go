// =============================================================================
// CSV to XML Converter - XML Writer Module
// =============================================================================
//
// This module is responsible for generating XML documents from the parsed and
// transformed CSV data. It handles the specific nesting structure required by
// the target financial system.
//
// XML STRUCTURE:
//   The generated XML follows this nesting pattern:
//
//   <cashbook>                           <!-- Root element -->
//     <transaction n="1">                <!-- Transaction element with index -->
//       <CheckNumber>12345</CheckNumber> <!-- Transaction-level fields -->
//       <CheckAmount>1000.00</CheckAmount>
//       <lineItem n="1">                 <!-- Line item element with global index -->
//         <PolicyNumber>A000123456</PolicyNumber>
//         <InvoiceNumber>INV-001</InvoiceNumber>
//       </lineItem>
//       <lineItem n="2">
//         <PolicyNumber>A000123457</PolicyNumber>
//         <InvoiceNumber>INV-002</InvoiceNumber>
//       </lineItem>
//     </transaction>
//     <transaction n="2">
//       <CheckNumber>12346</CheckNumber>
//       <lineItem n="3">                 <!-- Note: global numbering continues -->
//         <PolicyNumber>B000789012</PolicyNumber>
//       </lineItem>
//     </transaction>
//   </cashbook>
//
// CUSTOMIZATION:
//   - Modify element names via the Schema struct
//   - Add attributes to elements as needed
//   - Change the numbering scheme (global vs. per-transaction)
//   - Add XML namespaces if required
//
// =============================================================================

package xmlwriter

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/config"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/converter"
	"github.com/ginjaninja78/CSV-to-XML-conversion/internal/xlsxparser"
)

// =============================================================================
// XML GENERATION OPTIONS
// =============================================================================

// GenerateOptions contains options for XML generation.
type GenerateOptions struct {
	// Indent is the string used for indentation.
	// Default: "  " (two spaces)
	Indent string

	// IncludeXMLDeclaration determines whether to include the XML declaration.
	// Default: true
	IncludeXMLDeclaration bool

	// XMLVersion is the XML version for the declaration.
	// Default: "1.0"
	XMLVersion string

	// Encoding is the encoding for the XML declaration.
	// Default: "UTF-8"
	Encoding string

	// RootAttributes are additional attributes for the root element.
	// Example: {"xmlns": "http://example.com/schema"}
	RootAttributes map[string]string

	// LineItemNumberingGlobal determines if line item numbering is global.
	// If true: line items are numbered 1, 2, 3, 4... across all transactions.
	// If false: line items restart at 1 for each transaction.
	// Default: true (as per your specification)
	LineItemNumberingGlobal bool

	// TransactionIndexAttribute is the attribute name for transaction index.
	// Default: "n"
	TransactionIndexAttribute string

	// LineItemIndexAttribute is the attribute name for line item index.
	// Default: "n"
	LineItemIndexAttribute string
}

// DefaultGenerateOptions returns the default generation options.
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		Indent:                    "  ",
		IncludeXMLDeclaration:     true,
		XMLVersion:                "1.0",
		Encoding:                  "UTF-8",
		RootAttributes:            make(map[string]string),
		LineItemNumberingGlobal:   true, // Global numbering as specified
		TransactionIndexAttribute: "n",
		LineItemIndexAttribute:    "n",
	}
}

// =============================================================================
// XML GENERATION FUNCTIONS
// =============================================================================

// Generate creates an XML document from the transactions and schema.
//
// PARAMETERS:
//   - transactions: The grouped and transformed transactions.
//   - schema: The parsed XLSX template schema.
//   - deptConfig: The department configuration (for static fields).
//
// RETURNS:
//   - The XML document as a byte slice.
//   - An error if generation fails.
//
// GENERATION PROCESS:
//   1. Create the root element (cashbook)
//   2. Add any cashbook-level fields
//   3. For each transaction:
//      a. Create the transaction element with index attribute
//      b. Add transaction-level fields
//      c. For each line item:
//         i. Create the line item element with global index attribute
//         ii. Add line item-level fields
//   4. Marshal the XML with proper indentation
func Generate(transactions []converter.Transaction, schema *xlsxparser.Schema, deptConfig *config.DepartmentConfig) ([]byte, error) {
	return GenerateWithOptions(transactions, schema, deptConfig, DefaultGenerateOptions())
}

// GenerateWithOptions creates an XML document with custom options.
func GenerateWithOptions(transactions []converter.Transaction, schema *xlsxparser.Schema, deptConfig *config.DepartmentConfig, options GenerateOptions) ([]byte, error) {
	var buffer bytes.Buffer

	// Write XML declaration if requested.
	if options.IncludeXMLDeclaration {
		buffer.WriteString(fmt.Sprintf("<?xml version=\"%s\" encoding=\"%s\"?>\n",
			options.XMLVersion, options.Encoding))
	}

	// Build the XML document.
	doc := buildDocument(transactions, schema, deptConfig, options)

	// Marshal the document.
	xmlBytes, err := marshalWithIndent(doc, options.Indent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	buffer.Write(xmlBytes)

	return buffer.Bytes(), nil
}

// =============================================================================
// XML DOCUMENT BUILDING
// =============================================================================

// XMLDocument represents the root of the XML document.
type XMLDocument struct {
	XMLName    xml.Name
	Attributes []xml.Attr
	Children   []interface{}
}

// XMLElement represents a generic XML element.
type XMLElement struct {
	XMLName    xml.Name
	Attributes []xml.Attr   `xml:",attr"`
	Value      string       `xml:",chardata"`
	Children   []XMLElement `xml:",any"`
}

// buildDocument constructs the XML document structure.
func buildDocument(transactions []converter.Transaction, schema *xlsxparser.Schema, deptConfig *config.DepartmentConfig, options GenerateOptions) *XMLDocument {
	doc := &XMLDocument{
		XMLName: xml.Name{Local: schema.XMLRootElement},
	}

	// Add root attributes.
	for key, value := range options.RootAttributes {
		doc.Attributes = append(doc.Attributes, xml.Attr{
			Name:  xml.Name{Local: key},
			Value: value,
		})
	}

	// Add cashbook-level static fields.
	for _, staticField := range deptConfig.StaticFields {
		if strings.ToLower(staticField.ParentTag) == "cashbook" {
			doc.Children = append(doc.Children, createSimpleElement(staticField.XMLTag, staticField.Value))
		}
	}

	// Add cashbook-level fields from schema.
	// CUSTOMIZATION: Add any fields that should appear at the cashbook level.

	// Add transactions.
	globalLineItemIndex := 1 // Global counter for line items

	for _, transaction := range transactions {
		transactionElement := buildTransactionElement(
			transaction,
			schema,
			deptConfig,
			options,
			&globalLineItemIndex,
		)
		doc.Children = append(doc.Children, transactionElement)
	}

	return doc
}

// buildTransactionElement constructs a transaction XML element.
//
// PARAMETERS:
//   - transaction: The transaction data.
//   - schema: The parsed schema.
//   - deptConfig: The department configuration.
//   - options: The generation options.
//   - globalLineItemIndex: Pointer to the global line item counter.
//
// RETURNS:
//   - The transaction element.
//
// STRUCTURE:
//   <transaction n="1">
//     <TransactionField1>value</TransactionField1>
//     <TransactionField2>value</TransactionField2>
//     <lineItem n="1">...</lineItem>
//     <lineItem n="2">...</lineItem>
//   </transaction>
func buildTransactionElement(transaction converter.Transaction, schema *xlsxparser.Schema, deptConfig *config.DepartmentConfig, options GenerateOptions, globalLineItemIndex *int) XMLElement {
	element := XMLElement{
		XMLName: xml.Name{Local: schema.XMLTransactionElement},
		Attributes: []xml.Attr{
			{
				Name:  xml.Name{Local: options.TransactionIndexAttribute},
				Value: fmt.Sprintf("%d", transaction.ID),
			},
		},
	}

	// Add transaction-level static fields.
	for _, staticField := range deptConfig.StaticFields {
		if strings.ToLower(staticField.ParentTag) == "transaction" {
			element.Children = append(element.Children,
				createSimpleElement(staticField.XMLTag, staticField.Value))
		}
	}

	// Add transaction-level fields from the first line item.
	// Transaction-level fields are typically the same across all line items in a transaction.
	if len(transaction.LineItems) > 0 {
		firstLineItem := transaction.LineItems[0]

		// Get transaction fields in order.
		transactionFields := getOrderedFields(schema.TransactionFields, schema)

		for _, oldHeader := range transactionFields {
			mapping := schema.GetFieldMapping(oldHeader)
			if mapping == nil {
				continue
			}

			value := firstLineItem.Fields[oldHeader]
			if value != "" || mapping.RequiredType == "required" {
				element.Children = append(element.Children,
					createSimpleElement(mapping.XMLTag, value))
			}
		}
	}

	// Add line items.
	for _, lineItem := range transaction.LineItems {
		lineItemElement := buildLineItemElement(
			lineItem,
			schema,
			deptConfig,
			options,
			globalLineItemIndex,
		)
		element.Children = append(element.Children, lineItemElement)

		// Increment global counter.
		if options.LineItemNumberingGlobal {
			(*globalLineItemIndex)++
		}
	}

	return element
}

// buildLineItemElement constructs a line item XML element.
//
// PARAMETERS:
//   - lineItem: The line item data.
//   - schema: The parsed schema.
//   - deptConfig: The department configuration.
//   - options: The generation options.
//   - globalLineItemIndex: Pointer to the global line item counter.
//
// RETURNS:
//   - The line item element.
//
// STRUCTURE:
//   <lineItem n="1">
//     <PolicyNumber>A000123456</PolicyNumber>
//     <InvoiceNumber>INV-001</InvoiceNumber>
//   </lineItem>
func buildLineItemElement(lineItem converter.LineItem, schema *xlsxparser.Schema, deptConfig *config.DepartmentConfig, options GenerateOptions, globalLineItemIndex *int) XMLElement {
	// Determine the index to use.
	index := lineItem.ID
	if options.LineItemNumberingGlobal {
		index = *globalLineItemIndex
	}

	element := XMLElement{
		XMLName: xml.Name{Local: schema.XMLLineItemElement},
		Attributes: []xml.Attr{
			{
				Name:  xml.Name{Local: options.LineItemIndexAttribute},
				Value: fmt.Sprintf("%d", index),
			},
		},
	}

	// Add line item-level static fields.
	for _, staticField := range deptConfig.StaticFields {
		if strings.ToLower(staticField.ParentTag) == "lineitem" {
			element.Children = append(element.Children,
				createSimpleElement(staticField.XMLTag, staticField.Value))
		}
	}

	// Add line item-level fields.
	lineItemFields := getOrderedFields(schema.LineItemFields, schema)

	for _, oldHeader := range lineItemFields {
		mapping := schema.GetFieldMapping(oldHeader)
		if mapping == nil {
			continue
		}

		value := lineItem.Fields[oldHeader]

		// Include the field if:
		// - It has a value, OR
		// - It's required (include empty to show validation error), OR
		// - We want to include all fields
		//
		// CUSTOMIZATION: Modify this logic based on your requirements.
		if value != "" || mapping.RequiredType == "required" {
			element.Children = append(element.Children,
				createSimpleElement(mapping.XMLTag, value))
		}
	}

	return element
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// createSimpleElement creates a simple XML element with a text value.
func createSimpleElement(name, value string) XMLElement {
	return XMLElement{
		XMLName: xml.Name{Local: name},
		Value:   value,
	}
}

// getOrderedFields returns fields in the order defined by the schema.
func getOrderedFields(fields []string, schema *xlsxparser.Schema) []string {
	// Create a copy to avoid modifying the original.
	ordered := make([]string, len(fields))
	copy(ordered, fields)

	// Sort by the Order field in the mapping.
	sort.Slice(ordered, func(i, j int) bool {
		mappingI := schema.GetFieldMapping(ordered[i])
		mappingJ := schema.GetFieldMapping(ordered[j])

		if mappingI == nil || mappingJ == nil {
			return false
		}

		return mappingI.Order < mappingJ.Order
	})

	return ordered
}

// marshalWithIndent marshals the document with indentation.
func marshalWithIndent(doc *XMLDocument, indent string) ([]byte, error) {
	// Use a custom marshaling approach for better control.
	var buffer bytes.Buffer

	// Write the root element opening tag.
	buffer.WriteString("<")
	buffer.WriteString(doc.XMLName.Local)

	// Write root attributes.
	for _, attr := range doc.Attributes {
		buffer.WriteString(fmt.Sprintf(" %s=\"%s\"", attr.Name.Local, escapeXML(attr.Value)))
	}

	buffer.WriteString(">\n")

	// Write children.
	for _, child := range doc.Children {
		switch c := child.(type) {
		case XMLElement:
			writeElement(&buffer, c, indent, 1)
		}
	}

	// Write the root element closing tag.
	buffer.WriteString("</")
	buffer.WriteString(doc.XMLName.Local)
	buffer.WriteString(">\n")

	return buffer.Bytes(), nil
}

// writeElement writes an XML element to the buffer with indentation.
func writeElement(buffer *bytes.Buffer, element XMLElement, indent string, level int) {
	// Write indentation.
	for i := 0; i < level; i++ {
		buffer.WriteString(indent)
	}

	// Write opening tag.
	buffer.WriteString("<")
	buffer.WriteString(element.XMLName.Local)

	// Write attributes.
	for _, attr := range element.Attributes {
		buffer.WriteString(fmt.Sprintf(" %s=\"%s\"", attr.Name.Local, escapeXML(attr.Value)))
	}

	// Check if element has children or value.
	if len(element.Children) == 0 && element.Value == "" {
		// Self-closing tag.
		buffer.WriteString("/>\n")
		return
	}

	buffer.WriteString(">")

	// Write value or children.
	if element.Value != "" {
		// Simple element with text value.
		buffer.WriteString(escapeXML(element.Value))
	} else {
		// Element with children.
		buffer.WriteString("\n")

		for _, child := range element.Children {
			writeElement(buffer, child, indent, level+1)
		}

		// Write indentation for closing tag.
		for i := 0; i < level; i++ {
			buffer.WriteString(indent)
		}
	}

	// Write closing tag.
	buffer.WriteString("</")
	buffer.WriteString(element.XMLName.Local)
	buffer.WriteString(">\n")
}

// escapeXML escapes special characters for XML.
func escapeXML(s string) string {
	var buffer bytes.Buffer

	for _, r := range s {
		switch r {
		case '&':
			buffer.WriteString("&amp;")
		case '<':
			buffer.WriteString("&lt;")
		case '>':
			buffer.WriteString("&gt;")
		case '"':
			buffer.WriteString("&quot;")
		case '\'':
			buffer.WriteString("&apos;")
		default:
			buffer.WriteRune(r)
		}
	}

	return buffer.String()
}

// =============================================================================
// XSD GENERATION (AUTO-GENERATE FROM TEMPLATE)
// =============================================================================

// GenerateXSD creates an XSD schema from the parsed template.
//
// PARAMETERS:
//   - schema: The parsed XLSX template schema.
//
// RETURNS:
//   - The XSD document as a byte slice.
//   - An error if generation fails.
//
// CUSTOMIZATION:
//   This function generates a basic XSD. Modify it to add:
//   - Custom data type restrictions
//   - Pattern matching for specific formats
//   - Enumeration values
//   - Complex type definitions
func GenerateXSD(schema *xlsxparser.Schema) ([]byte, error) {
	var buffer bytes.Buffer

	// Write XSD header.
	buffer.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
`)

	// Write root element definition.
	buffer.WriteString(fmt.Sprintf(`  <xs:element name="%s">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="%s" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>

`, schema.XMLRootElement, schema.XMLTransactionElement))

	// Write transaction element definition.
	buffer.WriteString(fmt.Sprintf(`  <xs:element name="%s">
    <xs:complexType>
      <xs:sequence>
`, schema.XMLTransactionElement))

	// Add transaction fields.
	for _, oldHeader := range schema.TransactionFields {
		mapping := schema.GetFieldMapping(oldHeader)
		if mapping != nil {
			writeXSDElement(&buffer, mapping, 4)
		}
	}

	// Add line item reference.
	buffer.WriteString(fmt.Sprintf(`        <xs:element ref="%s" minOccurs="0" maxOccurs="unbounded"/>
`, schema.XMLLineItemElement))

	buffer.WriteString(`      </xs:sequence>
      <xs:attribute name="n" type="xs:positiveInteger" use="required"/>
    </xs:complexType>
  </xs:element>

`)

	// Write line item element definition.
	buffer.WriteString(fmt.Sprintf(`  <xs:element name="%s">
    <xs:complexType>
      <xs:sequence>
`, schema.XMLLineItemElement))

	// Add line item fields.
	for _, oldHeader := range schema.LineItemFields {
		mapping := schema.GetFieldMapping(oldHeader)
		if mapping != nil {
			writeXSDElement(&buffer, mapping, 4)
		}
	}

	buffer.WriteString(`      </xs:sequence>
      <xs:attribute name="n" type="xs:positiveInteger" use="required"/>
    </xs:complexType>
  </xs:element>

</xs:schema>
`)

	return buffer.Bytes(), nil
}

// writeXSDElement writes an XSD element definition.
func writeXSDElement(buffer *bytes.Buffer, mapping *xlsxparser.FieldMapping, indentLevel int) {
	indent := strings.Repeat("  ", indentLevel)

	// Determine XSD type based on data type.
	xsdType := getXSDType(mapping.DataType)

	// Determine minOccurs based on required type.
	minOccurs := "0"
	if mapping.RequiredType == "required" {
		minOccurs = "1"
	}

	// Write element with restrictions if needed.
	if mapping.MaxLength > 0 && (mapping.DataType == "string" || mapping.DataType == "alphanumeric") {
		// Element with length restriction.
		buffer.WriteString(fmt.Sprintf(`%s<xs:element name="%s" minOccurs="%s">
%s  <xs:simpleType>
%s    <xs:restriction base="%s">
%s      <xs:maxLength value="%d"/>
%s    </xs:restriction>
%s  </xs:simpleType>
%s</xs:element>
`, indent, mapping.XMLTag, minOccurs,
			indent, indent, xsdType,
			indent, mapping.MaxLength,
			indent, indent, indent))
	} else {
		// Simple element.
		buffer.WriteString(fmt.Sprintf(`%s<xs:element name="%s" type="%s" minOccurs="%s"/>
`, indent, mapping.XMLTag, xsdType, minOccurs))
	}
}

// getXSDType maps internal data types to XSD types.
func getXSDType(dataType string) string {
	switch dataType {
	case "numeric":
		return "xs:integer"
	case "decimal":
		return "xs:decimal"
	case "boolean":
		return "xs:boolean"
	case "date":
		return "xs:date"
	default:
		return "xs:string"
	}
}
