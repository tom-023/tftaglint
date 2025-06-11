package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/tom-023/tftaglint/internal/parser"
	"github.com/tom-023/tftaglint/internal/validator"
)

func TestReport(t *testing.T) {
	tests := []struct {
		name       string
		violations []validator.Violation
		wantOutput []string
		notWant    []string
	}{
		{
			name:       "no violations",
			violations: []validator.Violation{},
			wantOutput: []string{"✅ No tag violations found!"},
		},
		{
			name: "single violation",
			violations: []validator.Violation{
				{
					Rule:        "required-tags",
					Description: "Required tags must be present",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "web",
						File: "main.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 10},
						},
					},
					Message: "Missing required tag: Name",
				},
			},
			wantOutput: []string{
				"❌ Found 1 tag violation(s):",
				"📄 main.tf",
				"Line 10: aws_instance.web",
				"Rule: required-tags",
				"Message: Missing required tag: Name",
				"Description: Required tags must be present",
			},
		},
		{
			name: "multiple violations in same file",
			violations: []validator.Violation{
				{
					Rule: "required-tags",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "web",
						File: "main.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 10},
						},
					},
					Message: "Missing required tag: Name",
				},
				{
					Rule: "forbidden-tags",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "web",
						File: "main.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 10},
						},
					},
					Message: "Forbidden tag found: Temp",
				},
			},
			wantOutput: []string{
				"❌ Found 2 tag violation(s):",
				"📄 main.tf",
				"Line 10: aws_instance.web",
				"Rule: required-tags",
				"Message: Missing required tag: Name",
				"Rule: forbidden-tags",
				"Message: Forbidden tag found: Temp",
			},
		},
		{
			name: "violations in multiple files",
			violations: []validator.Violation{
				{
					Rule: "required-tags",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "web",
						File: "main.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 10},
						},
					},
					Message: "Missing required tag: Name",
				},
				{
					Rule: "required-tags",
					Resource: parser.Resource{
						Type: "aws_s3_bucket",
						Name: "data",
						File: "storage.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 5},
						},
					},
					Message: "Missing required tag: Owner",
				},
			},
			wantOutput: []string{
				"❌ Found 2 tag violation(s):",
				"📄 main.tf",
				"Line 10: aws_instance.web",
				"📄 storage.tf",
				"Line 5: aws_s3_bucket.data",
			},
		},
		{
			name: "violations sorted by line number",
			violations: []validator.Violation{
				{
					Rule: "rule1",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "web2",
						File: "main.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 20},
						},
					},
					Message: "Violation at line 20",
				},
				{
					Rule: "rule2",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "web1",
						File: "main.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 10},
						},
					},
					Message: "Violation at line 10",
				},
			},
			wantOutput: []string{
				"Line 10: aws_instance.web1",
				"Line 20: aws_instance.web2",
			},
		},
		{
			name: "violation without description",
			violations: []validator.Violation{
				{
					Rule: "simple-rule",
					Resource: parser.Resource{
						Type: "aws_instance",
						Name: "test",
						File: "test.tf",
						Location: hcl.Range{
							Start: hcl.Pos{Line: 1},
						},
					},
					Message: "Simple violation",
				},
			},
			wantOutput: []string{
				"Rule: simple-rule",
				"Message: Simple violation",
			},
			notWant: []string{
				"Description:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewReporter(&buf)
			
			err := reporter.Report(tt.violations)
			if err != nil {
				t.Fatalf("Report() error = %v", err)
			}
			
			output := buf.String()
			
			// Check for expected strings
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", want, output)
				}
			}
			
			// Check for strings that should not be present
			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("Expected output NOT to contain %q, but it does.\nOutput:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestReportSummary(t *testing.T) {
	tests := []struct {
		name       string
		violations []validator.Violation
		wantOutput []string
	}{
		{
			name:       "no violations",
			violations: []validator.Violation{},
			wantOutput: []string{},
		},
		{
			name: "single rule violations",
			violations: []validator.Violation{
				{Rule: "required-tags"},
				{Rule: "required-tags"},
			},
			wantOutput: []string{
				"Summary:",
				"Total violations: 2",
				"Violations by rule:",
				"required-tags: 2",
			},
		},
		{
			name: "multiple rules violations",
			violations: []validator.Violation{
				{Rule: "required-tags"},
				{Rule: "forbidden-tags"},
				{Rule: "required-tags"},
				{Rule: "tag-constraints"},
				{Rule: "forbidden-tags"},
				{Rule: "forbidden-tags"},
			},
			wantOutput: []string{
				"Summary:",
				"Total violations: 6",
				"Violations by rule:",
				"forbidden-tags: 3",
				"required-tags: 2",
				"tag-constraints: 1",
			},
		},
		{
			name: "sorted rule names",
			violations: []validator.Violation{
				{Rule: "z-rule"},
				{Rule: "a-rule"},
				{Rule: "m-rule"},
			},
			wantOutput: []string{
				"a-rule: 1",
				"m-rule: 1",
				"z-rule: 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewReporter(&buf)
			
			err := reporter.ReportSummary(tt.violations)
			if err != nil {
				t.Fatalf("ReportSummary() error = %v", err)
			}
			
			output := buf.String()
			
			if len(tt.violations) == 0 && output != "" {
				t.Errorf("Expected no output for empty violations, got:\n%s", output)
			}
			
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", want, output)
				}
			}
		})
	}
}

func TestReportViolation(t *testing.T) {
	tests := []struct {
		name       string
		violation  validator.Violation
		wantOutput []string
	}{
		{
			name: "full violation details",
			violation: validator.Violation{
				Rule:        "test-rule",
				Description: "Test description",
				Resource: parser.Resource{
					Type: "aws_instance",
					Name: "example",
					Location: hcl.Range{
						Start: hcl.Pos{Line: 42},
					},
				},
				Message: "Test message",
			},
			wantOutput: []string{
				"Line 42: aws_instance.example",
				"Rule: test-rule",
				"Message: Test message",
				"Description: Test description",
			},
		},
		{
			name: "violation without description",
			violation: validator.Violation{
				Rule: "test-rule",
				Resource: parser.Resource{
					Type: "aws_s3_bucket",
					Name: "data",
					Location: hcl.Range{
						Start: hcl.Pos{Line: 10},
					},
				},
				Message: "Missing tag",
			},
			wantOutput: []string{
				"Line 10: aws_s3_bucket.data",
				"Rule: test-rule",
				"Message: Missing tag",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			reporter := NewReporter(&buf)
			
			reporter.reportViolation(tt.violation)
			
			output := buf.String()
			
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", want, output)
				}
			}
		})
	}
}

func TestNewReporter(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)
	
	if reporter == nil {
		t.Fatal("NewReporter() returned nil")
	}
	
	if reporter.writer != &buf {
		t.Error("NewReporter() did not set writer correctly")
	}
}