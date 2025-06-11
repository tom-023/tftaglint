package validator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/tom-023/tftaglint/internal/config"
	"github.com/tom-023/tftaglint/internal/parser"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name         string
		config       *config.Config
		resources    []parser.Resource
		wantViolations int
		checkViolations func(t *testing.T, violations []Violation)
	}{
		{
			name: "global required tags",
			config: &config.Config{
				Global: config.Global{
					AlwaysRequiredTags: []string{"Environment", "Owner"},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Environment": "prod",
						"Owner":       "team-a",
					},
				},
				{
					Type: "aws_instance",
					Name: "db",
					Tags: map[string]string{
						"Environment": "prod",
						// Missing Owner tag
					},
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Message != "Missing required tag: Owner" {
					t.Errorf("Expected missing Owner tag violation, got: %s", violations[0].Message)
				}
			},
		},
		{
			name: "ignored resource types",
			config: &config.Config{
				Global: config.Global{
					AlwaysRequiredTags:  []string{"Name"},
					IgnoreResourceTypes: []string{"aws_iam_role", "aws_iam_policy"},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{}, // Missing Name tag
				},
				{
					Type: "aws_iam_role",
					Name: "ignored",
					Tags: map[string]string{}, // Missing Name tag but should be ignored
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Resource.Type != "aws_instance" {
					t.Errorf("Expected violation for aws_instance, got: %s", violations[0].Resource.Type)
				}
			},
		},
		{
			name: "rule with resource type filter",
			config: &config.Config{
				Rules: []config.Rule{
					{
						Name:          "EC2 Tags",
						Description:   "EC2 specific tags",
						RequiredTags:  []string{"Name", "InstanceType"},
						ResourceTypes: []string{"aws_instance"},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Name": "web-server",
						// Missing InstanceType
					},
				},
				{
					Type: "aws_s3_bucket",
					Name: "data",
					Tags: map[string]string{
						// Should not require InstanceType
					},
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Resource.Type != "aws_instance" {
					t.Errorf("Expected violation for aws_instance, got: %s", violations[0].Resource.Type)
				}
			},
		},
		{
			name: "rule with condition",
			config: &config.Config{
				Rules: []config.Rule{
					{
						Name:         "Production Tags",
						Description:  "Additional tags for production",
						RequiredTags: []string{"Monitoring", "Backup"},
						Condition: &config.Condition{
							Tag:   "Environment",
							Value: "prod",
						},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "prod-server",
					Tags: map[string]string{
						"Environment": "prod",
						"Monitoring":  "enabled",
						// Missing Backup
					},
				},
				{
					Type: "aws_instance",
					Name: "dev-server",
					Tags: map[string]string{
						"Environment": "dev",
						// Should not require Monitoring or Backup
					},
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Resource.Name != "prod-server" {
					t.Errorf("Expected violation for prod-server, got: %s", violations[0].Resource.Name)
				}
			},
		},
		{
			name: "forbidden tags",
			config: &config.Config{
				Rules: []config.Rule{
					{
						Name:          "No Temp Tags",
						Description:   "Temporary tags are not allowed",
						ForbiddenTags: []string{"Temp", "Test", "TODO"},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Name": "web-server",
						"Temp": "remove-later",
					},
				},
				{
					Type: "aws_instance",
					Name: "db",
					Tags: map[string]string{
						"Name": "db-server",
					},
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Message != "Forbidden tag found: Temp" {
					t.Errorf("Expected forbidden tag violation, got: %s", violations[0].Message)
				}
			},
		},
		{
			name: "tag value constraints",
			config: &config.Config{
				Rules: []config.Rule{
					{
						Name:        "Environment Values",
						Description: "Valid environment values",
						TagConstraints: []config.TagConstraint{
							{
								Tag:           "Environment",
								AllowedValues: []string{"dev", "staging", "prod"},
							},
						},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Environment": "production", // Invalid value
					},
				},
				{
					Type: "aws_instance",
					Name: "db",
					Tags: map[string]string{
						"Environment": "prod", // Valid value
					},
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Resource.Name != "web" {
					t.Errorf("Expected violation for web resource, got: %s", violations[0].Resource.Name)
				}
			},
		},
		{
			name: "tag name patterns",
			config: &config.Config{
				Rules: []config.Rule{
					{
						Name:        "Tag Naming Convention",
						Description: "Tags must follow naming convention",
						TagPatterns: []config.TagPattern{
							{
								Pattern: "^[A-Z][a-zA-Z0-9]*$",
								Message: "Tag names must start with uppercase",
							},
						},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Name":        "web-server",
						"environment": "prod", // lowercase start
					},
				},
			},
			wantViolations: 1,
			checkViolations: func(t *testing.T, violations []Violation) {
				if violations[0].Message != "Tag name 'environment' does not match pattern: Tag names must start with uppercase" {
					t.Errorf("Unexpected violation message: %s", violations[0].Message)
				}
			},
		},
		{
			name: "multiple rules and violations",
			config: &config.Config{
				Global: config.Global{
					AlwaysRequiredTags: []string{"Owner"},
				},
				Rules: []config.Rule{
					{
						Name:         "Basic Tags",
						Description:  "Basic required tags",
						RequiredTags: []string{"Name"},
					},
					{
						Name:          "No Test Tags",
						Description:   "Test tags forbidden",
						ForbiddenTags: []string{"Test"},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Test": "true",
						// Missing Owner and Name
					},
				},
			},
			wantViolations: 3, // Missing Owner (global), Missing Name, Forbidden Test
		},
		{
			name: "no violations",
			config: &config.Config{
				Global: config.Global{
					AlwaysRequiredTags: []string{"Name", "Environment"},
				},
				Rules: []config.Rule{
					{
						Name:         "Basic Tags",
						Description:  "Basic tags",
						RequiredTags: []string{"Owner"},
						TagConstraints: []config.TagConstraint{
							{
								Tag:           "Environment",
								AllowedValues: []string{"dev", "prod"},
							},
						},
					},
				},
			},
			resources: []parser.Resource{
				{
					Type: "aws_instance",
					Name: "web",
					Tags: map[string]string{
						"Name":        "web-server",
						"Environment": "prod",
						"Owner":       "team-a",
					},
				},
			},
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup tag patterns with compiled regex
			for i := range tt.config.Rules {
				for j := range tt.config.Rules[i].TagPatterns {
					pattern := &tt.config.Rules[i].TagPatterns[j]
					if pattern.Pattern != "" {
						regex, err := regexp.Compile(pattern.Pattern)
						if err != nil {
							t.Fatalf("Failed to compile regex: %v", err)
						}
						pattern.Regex = regex
					}
				}
			}

			validator := NewValidator(tt.config)
			violations := validator.Validate(tt.resources)

			if len(violations) != tt.wantViolations {
				t.Errorf("Expected %d violations, got %d", tt.wantViolations, len(violations))
				for i, v := range violations {
					t.Logf("Violation %d: %s - %s", i+1, v.Rule, v.Message)
				}
			}

			if tt.checkViolations != nil && len(violations) > 0 {
				tt.checkViolations(t, violations)
			}
		})
	}
}

func TestShouldIgnoreResource(t *testing.T) {
	v := &Validator{
		config: &config.Config{
			Global: config.Global{
				IgnoreResourceTypes: []string{"aws_iam_role", "aws_iam_policy"},
			},
		},
	}

	tests := []struct {
		resourceType string
		wantIgnore   bool
	}{
		{"aws_iam_role", true},
		{"aws_iam_policy", true},
		{"aws_instance", false},
		{"aws_s3_bucket", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got := v.shouldIgnoreResource(tt.resourceType)
			if got != tt.wantIgnore {
				t.Errorf("shouldIgnoreResource(%s) = %v, want %v", tt.resourceType, got, tt.wantIgnore)
			}
		})
	}
}

func TestCheckCondition(t *testing.T) {
	v := &Validator{}

	tests := []struct {
		name      string
		resource  parser.Resource
		condition *config.Condition
		want      bool
	}{
		{
			name: "condition met",
			resource: parser.Resource{
				Tags: map[string]string{
					"Environment": "prod",
				},
			},
			condition: &config.Condition{
				Tag:   "Environment",
				Value: "prod",
			},
			want: true,
		},
		{
			name: "condition not met - different value",
			resource: parser.Resource{
				Tags: map[string]string{
					"Environment": "dev",
				},
			},
			condition: &config.Condition{
				Tag:   "Environment",
				Value: "prod",
			},
			want: false,
		},
		{
			name: "condition not met - tag missing",
			resource: parser.Resource{
				Tags: map[string]string{},
			},
			condition: &config.Condition{
				Tag:   "Environment",
				Value: "prod",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := v.checkCondition(tt.resource, tt.condition)
			if got != tt.want {
				t.Errorf("checkCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValueAllowed(t *testing.T) {
	v := &Validator{}

	tests := []struct {
		value         string
		allowedValues []string
		want          bool
	}{
		{"prod", []string{"dev", "staging", "prod"}, true},
		{"production", []string{"dev", "staging", "prod"}, false},
		{"dev", []string{"dev"}, true},
		{"", []string{"dev", "prod"}, false},
		{"test", []string{}, false},
	}

	for _, tt := range tests {
		name := tt.value + " in " + strings.Join(tt.allowedValues, ",")
		t.Run(name, func(t *testing.T) {
			got := v.isValueAllowed(tt.value, tt.allowedValues)
			if got != tt.want {
				t.Errorf("isValueAllowed(%s, %v) = %v, want %v", tt.value, tt.allowedValues, got, tt.want)
			}
		})
	}
}

func TestViolationStructure(t *testing.T) {
	// Test that Violation struct contains expected fields
	violation := Violation{
		Rule:        "test-rule",
		Description: "Test rule description",
		Resource: parser.Resource{
			Type: "aws_instance",
			Name: "test",
			Location: hcl.Range{
				Filename: "test.tf",
				Start:    hcl.Pos{Line: 10, Column: 1},
			},
		},
		Message: "Test violation message",
	}

	if violation.Rule != "test-rule" {
		t.Errorf("Expected Rule 'test-rule', got %s", violation.Rule)
	}
	if violation.Description != "Test rule description" {
		t.Errorf("Expected Description 'Test rule description', got %s", violation.Description)
	}
	if violation.Message != "Test violation message" {
		t.Errorf("Expected Message 'Test violation message', got %s", violation.Message)
	}
	if violation.Resource.Type != "aws_instance" {
		t.Errorf("Expected Resource.Type 'aws_instance', got %s", violation.Resource.Type)
	}
}