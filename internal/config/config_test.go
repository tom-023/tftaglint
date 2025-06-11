package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, config *Config)
	}{
		{
			name: "valid config with all fields",
			content: `
global:
  always_required_tags:
    - Environment
    - Owner
  ignore_resource_types:
    - aws_iam_role_policy
    - aws_iam_policy_document
rules:
  - name: EC2 Instance Tags
    description: EC2インスタンスには特定のタグが必要です
    required_tags:
      - Name
      - Environment
      - Owner
    forbidden_tags:
      - TempTag
    resource_types:
      - aws_instance
    tag_constraints:
      - tag: Environment
        allowed_values:
          - dev
          - stg
          - prod
    tag_patterns:
      - pattern: "^[A-Z][a-zA-Z0-9-]*$"
        message: "タグ名は大文字で始まる必要があります"
  - name: S3 Bucket Tags
    description: S3バケットのタグルール
    required_tags:
      - Purpose
    resource_types:
      - aws_s3_bucket
    condition:
      tag: Environment
      value: prod
`,
			wantErr: false,
			check: func(t *testing.T, config *Config) {
				if len(config.Global.AlwaysRequiredTags) != 2 {
					t.Errorf("Expected 2 always required tags, got %d", len(config.Global.AlwaysRequiredTags))
				}
				if len(config.Global.IgnoreResourceTypes) != 2 {
					t.Errorf("Expected 2 ignore resource types, got %d", len(config.Global.IgnoreResourceTypes))
				}
				if len(config.Rules) != 2 {
					t.Errorf("Expected 2 rules, got %d", len(config.Rules))
				}
				
				// Check first rule
				rule1 := config.Rules[0]
				if rule1.Name != "EC2 Instance Tags" {
					t.Errorf("Expected rule name 'EC2 Instance Tags', got %s", rule1.Name)
				}
				if len(rule1.RequiredTags) != 3 {
					t.Errorf("Expected 3 required tags, got %d", len(rule1.RequiredTags))
				}
				if len(rule1.ForbiddenTags) != 1 {
					t.Errorf("Expected 1 forbidden tag, got %d", len(rule1.ForbiddenTags))
				}
				if len(rule1.TagConstraints) != 1 {
					t.Errorf("Expected 1 tag constraint, got %d", len(rule1.TagConstraints))
				}
				if len(rule1.TagPatterns) != 1 {
					t.Errorf("Expected 1 tag pattern, got %d", len(rule1.TagPatterns))
				}
				
				// Check second rule condition
				rule2 := config.Rules[1]
				if rule2.Condition == nil {
					t.Error("Expected condition to be set")
				} else {
					if rule2.Condition.Tag != "Environment" || rule2.Condition.Value != "prod" {
						t.Errorf("Unexpected condition: tag=%s, value=%s", rule2.Condition.Tag, rule2.Condition.Value)
					}
				}
			},
		},
		{
			name: "minimal valid config",
			content: `
rules:
  - name: Basic Rule
    description: Basic rule for testing
    required_tags:
      - Name
`,
			wantErr: false,
			check: func(t *testing.T, config *Config) {
				if len(config.Rules) != 1 {
					t.Errorf("Expected 1 rule, got %d", len(config.Rules))
				}
				if config.Rules[0].Name != "Basic Rule" {
					t.Errorf("Expected rule name 'Basic Rule', got %s", config.Rules[0].Name)
				}
			},
		},
		{
			name:    "invalid yaml",
			content: `invalid: [yaml content`,
			wantErr: true,
		},
		{
			name: "invalid regex pattern",
			content: `
rules:
  - name: Invalid Regex Rule
    description: Rule with invalid regex
    tag_patterns:
      - pattern: "[invalid(regex"
        message: "Invalid regex"
`,
			wantErr: true,
		},
		{
			name:    "empty config",
			content: "",
			wantErr: false,
			check: func(t *testing.T, config *Config) {
				if len(config.Rules) != 0 {
					t.Errorf("Expected 0 rules, got %d", len(config.Rules))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test-config.yaml")
			
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			
			// Test LoadConfig
			config, err := LoadConfig(tmpFile)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, config)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/non/existent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestTagPattern_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		tagName string
		want    bool
	}{
		{
			name:    "match uppercase start",
			pattern: "^[A-Z][a-zA-Z0-9-]*$",
			tagName: "Environment",
			want:    true,
		},
		{
			name:    "no match lowercase start",
			pattern: "^[A-Z][a-zA-Z0-9-]*$",
			tagName: "environment",
			want:    false,
		},
		{
			name:    "match with hyphen",
			pattern: "^[A-Z][a-zA-Z0-9-]*$",
			tagName: "Cost-Center",
			want:    true,
		},
		{
			name:    "no match special character",
			pattern: "^[A-Z][a-zA-Z0-9-]*$",
			tagName: "Cost_Center",
			want:    false,
		},
		{
			name:    "nil regex always validates",
			pattern: "",
			tagName: "anything",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp := &TagPattern{Pattern: tt.pattern}
			if tt.pattern != "" {
				regex, err := regexp.Compile(tt.pattern)
				if err != nil {
					t.Fatalf("Failed to compile regex: %v", err)
				}
				tp.Regex = regex
			}
			
			got := tp.Validate(tt.tagName)
			if got != tt.want {
				t.Errorf("TagPattern.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}