# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tftaglint is a Go-based CLI tool that validates Terraform resource tags against YAML-defined rules. It supports both direct .tf file parsing and Terraform plan JSON validation for comprehensive tag compliance checking.

## Key Commands

### Build and Development
```bash
# Build the binary
go build -o tftaglint cmd/tftaglint/main.go

# Run without building
go run cmd/tftaglint/main.go validate

# Format code
go fmt ./...

# Check for issues
go vet ./...

# Install locally
go install ./cmd/tftaglint
```

### Testing Validation
```bash
# Test with example files
./tftaglint validate test_data/

# Test with plan JSON
./tftaglint validate --plan test_data/example-plan.json

# Test with custom rules
./tftaglint validate -c tag-rules.yaml test_data/
```

## Architecture

The codebase follows a clean modular structure:

- **cmd/tftaglint/main.go**: CLI entry point using Cobra framework. Handles command-line flags and orchestrates the validation flow
- **internal/config**: Parses YAML rule files into Go structs
- **internal/parser**: 
  - `parser.go`: Parses .tf files using HCL2 to extract resources and tags
  - `plan_parser.go`: Parses Terraform plan JSON to extract resolved resource configurations
- **internal/validator**: Core validation logic that checks resources against rules
- **internal/reporter**: Formats and displays validation results with file locations

### Key Design Patterns

1. **Dual Parser Architecture**: Separate parsers for HCL files and plan JSON, unified through common Resource/Tag structs
2. **Rule Hierarchy**: Global rules (always_required_tags) apply to all resources, while specific rules can target resource types
3. **Precise Error Location**: Parser tracks exact file positions for helpful error messages
4. **Exit Code Integration**: Returns non-zero exit code on violations for CI/CD pipelines

### Rule Configuration Format

Rules are defined in YAML with these key fields:
- `required_tags`: Tags that must be present
- `forbidden_tags`: Tags that must not be present
- `resource_types`: Specific resource types this rule applies to
- `condition`: Apply rule only when certain tag values exist
- `tag_constraints`: Allowed values for specific tags
- `tag_patterns`: Regex patterns tags must match

## Development Notes

- The tool uses Go 1.23.0 with modules
- Main dependencies: hashicorp/hcl/v2 for parsing, spf13/cobra for CLI
- Error messages and documentation are in Japanese
- Plan parsing is recommended for accurate validation when using Terraform variables
- Resource type extraction handles module prefixes (e.g., "module.vpc.aws_subnet.main" → "aws_subnet")