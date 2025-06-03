# tftaglint

tftaglint is a tool for validating Terraform resource tags. It validates tag combinations and values based on rules defined in a configuration file, and reports violations along with file names and line numbers.

## Features

- 🏷️ Comprehensive validation of Terraform resource tags
- 📝 Flexible YAML-based rule configuration
- 📍 Precise location reporting with file names and line numbers
- 🎯 Resource type-specific rule configuration
- ⚡ Fast parsing and validation

## Installation

```bash
go install github.com/tom-023/tftaglint/cmd/tftaglint@latest
```

Or build from source:

```bash
git clone https://github.com/tom-023/tftaglint.git
cd tftaglint
go build -o tftaglint cmd/tftaglint/main.go
```

## Usage

### Basic Usage

```bash
# Validate Terraform files in current directory
tftaglint validate

# Validate specific directory
tftaglint validate ./terraform/

# Use custom configuration file
tftaglint validate -c custom-rules.yaml

# Use -f option as an alias for -c
tftaglint validate -f my-tag-rules.yaml

# Show summary
tftaglint validate -s
```

### Validation using Terraform Plan (Recommended)

When managing tags with `locals` or variables, you can validate with actual resolved values by using terraform plan output.

```bash
# Output terraform plan in JSON format
terraform plan -out=tfplan
terraform show -json tfplan > tfplan.json

# Validate using plan file
tftaglint validate --plan tfplan.json

# Or use short form
tftaglint validate -p tfplan.json -s
```

Benefits of this approach:
- Validates with actual values after variable expansion
- Includes resources within modules
- Correctly recognizes tags defined in `locals`

## Configuration File

tftaglint defines rules in a configuration file called `tag-rules.yaml`.

### Configuration Example

```yaml
rules:
  # Define required tags
  - name: "environment-required"
    description: "All resources must have an Environment tag"
    required_tags:
      - Environment

  # Conditional rules
  - name: "production-tags"
    description: "Production resources require additional tags"
    condition:
      tag: Environment
      value: production
    required_tags:
      - Owner
      - CostCenter
      - BackupRequired

  # Define forbidden tags
  - name: "no-test-in-production"
    description: "Test tag cannot be used in production environment"
    condition:
      tag: Environment
      value: production
    forbidden_tags:
      - Test
      - Temporary

  # Validate tag values
  - name: "valid-environment-values"
    description: "Environment tag must have predefined values"
    tag_constraints:
      - tag: Environment
        allowed_values:
          - development
          - staging
          - production

# Global settings
global:
  always_required_tags:
    - Project
    - ManagedBy
  ignore_resource_types:
    - data.aws_ami
```

## Rule Types

### 1. Required Tags (`required_tags`)
Requires specified tags to be present.

### 2. Forbidden Tags (`forbidden_tags`)
Requires specified tags to be absent.

### 3. Conditional Rules (`condition`)
Applies rules only when specific tag-value combinations exist.

### 4. Tag Constraints (`tag_constraints`)
Validates that tag values are within allowed lists.

### 5. Tag Patterns (`tag_patterns`)
Validates that tag names match regular expression patterns.

### 6. Resource Type-specific Rules (`resource_types`)
Applies rules only to specific resource types.

## Output Example

```
❌ Found 4 tag violation(s):

📄 test_data/example.tf
  Line 15: aws_instance.db
    Rule: no-test-in-production
    Message: Forbidden tag found: Test
    Description: Test tag cannot be used in production environment

  Line 15: aws_instance.db
    Rule: global-required-tags
    Message: Missing required tag: ManagedBy
    Description: Global required tags

  Line 26: aws_s3_bucket.logs
    Rule: global-required-tags
    Message: Missing required tag: ManagedBy
    Description: Global required tags

  Line 37: aws_instance.test
    Rule: valid-environment-values
    Message: Invalid value for tag Environment: 'invalid-env'. Allowed values: development, staging, production
    Description: Environment tag must have predefined values
```

## License

MIT License