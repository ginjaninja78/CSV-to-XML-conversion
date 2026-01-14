# CSV to XML Converter - Copilot Instructions

This document provides instructions for GitHub Copilot and other AI assistants working on this project.

## Project Overview

This is a Go-based CSV to XML converter designed for financial transaction processing. It uses:
- **Cobra CLI** for command-line interface
- **Viper** for configuration management
- **Excelize** for XLSX template parsing
- **YAML** for configuration files

## Architecture

```
main.go                    → Entry point, initializes Cobra
cmd/                       → CLI commands (root, process, version)
internal/config/           → Configuration loading and parsing
internal/converter/        → Main conversion orchestration
internal/csvparser/        → CSV file parsing
internal/xlsxparser/       → XLSX template parsing
internal/validation/       → Validation engine
internal/xmlwriter/        → XML generation
pkg/utils/                 → File management utilities
```

## Key Design Patterns

1. **Configuration-Driven**: All business logic is externalized to YAML files
2. **Template-Based Schema**: XLSX files define the XML structure
3. **Modular Transformations**: Transformation rules are composable
4. **Streaming for Large Files**: Memory-efficient processing

## Code Style Guidelines

1. **Comments**: Every function should have detailed comments explaining:
   - Purpose
   - Parameters
   - Return values
   - Customization points

2. **Error Handling**: Always wrap errors with context:
   ```go
   return fmt.Errorf("failed to parse CSV: %w", err)
   ```

3. **Configuration**: Use Viper for all configuration access

4. **Logging**: Use structured logging with levels (debug, info, warn, error)

## Current TODO List

### High Priority
- [ ] Complete the `internal/converter/converter.go` orchestration logic
- [ ] Implement the `cmd/process.go` command handler
- [ ] Add Viper configuration loading in `internal/config/config.go`
- [ ] Implement XLSX file reading with Excelize

### Medium Priority
- [ ] Add unit tests for all modules
- [ ] Implement concurrent file processing
- [ ] Add email notification on errors
- [ ] Create sample XLSX template files

### Low Priority
- [ ] Add compression for archive files
- [ ] Implement archive retention policies
- [ ] Add metrics/telemetry
- [ ] Create Docker container

## Customization Points

Look for these markers in the code:

- `CUSTOMIZATION:` - Areas that need to be modified for specific business logic
- `QUESTION FOR USER:` - Information needed from the user
- `PSEUDOCODE:` - Placeholder logic to be implemented
- `TODO:` - Tasks that need to be completed

## Testing Strategy

1. **Unit Tests**: Test individual functions in isolation
2. **Integration Tests**: Test module interactions
3. **End-to-End Tests**: Test complete conversion workflow
4. **Validation Tests**: Test all validation rules

## Build Commands

```bash
# Development build
go build -o csv2xml .

# Production build (optimized)
go build -ldflags="-s -w" -o csv2xml .

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Format code
go fmt ./...

# Lint code
golangci-lint run
```

## Configuration Files

1. `config/app_config.yaml` - Main application configuration
2. `department_mappings/<dept>/department_config.yaml` - Department-specific settings
3. `templates/*.xlsx` - Schema definition templates

## Key Functions to Implement

### internal/converter/converter.go
```go
// ProcessFile orchestrates the conversion of a single CSV file
func ProcessFile(inputPath string, config *Config) error {
    // 1. Load department configuration
    // 2. Parse XLSX template for schema
    // 3. Parse CSV file
    // 4. Group rows into transactions
    // 5. Apply transformations
    // 6. Validate data
    // 7. Generate XML
    // 8. Write output file
    // 9. Archive input file
}
```

### cmd/process.go
```go
// processCmd handles the 'process' command
var processCmd = &cobra.Command{
    Use:   "process",
    Short: "Process CSV files and generate XML",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Load configuration
        // 2. Discover input files
        // 3. Process each file
        // 4. Generate summary report
    },
}
```

## Dependencies to Add

```bash
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/qax-os/excelize/v2
go get github.com/google/uuid
go get gopkg.in/yaml.v3
```

## Workflow for Adding New Features

1. Create a feature branch: `git checkout -b feature/feature-name`
2. Implement the feature with comprehensive comments
3. Add unit tests
4. Update documentation
5. Commit with detailed message
6. Push and create PR

## Common Issues and Solutions

### Issue: XLSX parsing fails
- Check that Excelize is properly imported
- Verify the template file structure matches expected columns

### Issue: CSV encoding problems
- Check the `encoding` setting in department config
- Consider adding encoding detection

### Issue: XML validation fails
- Check the schema definition in the template
- Verify all required fields are present

## Contact

For questions about this project, refer to the design document (`DESIGN.md`) or contact the development team.
