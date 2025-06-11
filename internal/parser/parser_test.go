package parser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTerraformFiles(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantCount int
		wantErr   bool
		check     func(t *testing.T, result *ParseResult)
	}{
		{
			name: "single resource with tags attribute",
			files: map[string]string{
				"main.tf": `
resource "aws_instance" "example" {
  instance_type = "t2.micro"
  
  tags = {
    Name        = "example-instance"
    Environment = "dev"
    Owner       = "team-a"
  }
}`,
			},
			wantCount: 1,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 1 {
					t.Fatalf("Expected 1 resource, got %d", len(result.Resources))
				}
				
				res := result.Resources[0]
				if res.Type != "aws_instance" {
					t.Errorf("Expected type 'aws_instance', got %s", res.Type)
				}
				if res.Name != "example" {
					t.Errorf("Expected name 'example', got %s", res.Name)
				}
				
				expectedTags := map[string]string{
					"Name":        "example-instance",
					"Environment": "dev",
					"Owner":       "team-a",
				}
				if !reflect.DeepEqual(res.Tags, expectedTags) {
					t.Errorf("Tags mismatch. Expected %v, got %v", expectedTags, res.Tags)
				}
			},
		},
		{
			name: "resource with tags block",
			files: map[string]string{
				"vpc.tf": `
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  
  tags {
    Name = "main-vpc"
    Type = "network"
  }
}`,
			},
			wantCount: 1,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				res := result.Resources[0]
				expectedTags := map[string]string{
					"Name": "main-vpc",
					"Type": "network",
				}
				if !reflect.DeepEqual(res.Tags, expectedTags) {
					t.Errorf("Tags mismatch. Expected %v, got %v", expectedTags, res.Tags)
				}
			},
		},
		{
			name: "multiple resources in one file",
			files: map[string]string{
				"resources.tf": `
resource "aws_instance" "web" {
  instance_type = "t2.micro"
  tags = {
    Name = "web-server"
  }
}

resource "aws_instance" "db" {
  instance_type = "t2.small"
  tags = {
    Name = "db-server"
  }
}

resource "aws_s3_bucket" "logs" {
  bucket = "my-logs"
  tags = {
    Purpose = "logging"
  }
}`,
			},
			wantCount: 3,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 3 {
					t.Fatalf("Expected 3 resources, got %d", len(result.Resources))
				}
				
				// Verify each resource
				resourceMap := make(map[string]Resource)
				for _, res := range result.Resources {
					resourceMap[res.Name] = res
				}
				
				if web, ok := resourceMap["web"]; ok {
					if web.Tags["Name"] != "web-server" {
						t.Errorf("Expected web server Name tag 'web-server', got %s", web.Tags["Name"])
					}
				} else {
					t.Error("Resource 'web' not found")
				}
				
				if db, ok := resourceMap["db"]; ok {
					if db.Tags["Name"] != "db-server" {
						t.Errorf("Expected db server Name tag 'db-server', got %s", db.Tags["Name"])
					}
				} else {
					t.Error("Resource 'db' not found")
				}
				
				if logs, ok := resourceMap["logs"]; ok {
					if logs.Type != "aws_s3_bucket" {
						t.Errorf("Expected logs type 'aws_s3_bucket', got %s", logs.Type)
					}
					if logs.Tags["Purpose"] != "logging" {
						t.Errorf("Expected logs Purpose tag 'logging', got %s", logs.Tags["Purpose"])
					}
				} else {
					t.Error("Resource 'logs' not found")
				}
			},
		},
		{
			name: "resource without tags",
			files: map[string]string{
				"no_tags.tf": `
resource "aws_instance" "no_tags" {
  instance_type = "t2.micro"
}`,
			},
			wantCount: 1,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				res := result.Resources[0]
				if len(res.Tags) != 0 {
					t.Errorf("Expected no tags, got %v", res.Tags)
				}
			},
		},
		{
			name: "invalid HCL syntax",
			files: map[string]string{
				"invalid.tf": `
resource "aws_instance" "bad" {
  instance_type = 
}`,
			},
			wantCount: 0,
			wantErr:   false, // ParseTerraformFiles doesn't return error, adds to result.Errors
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Errors) == 0 {
					t.Error("Expected parse error, got none")
				}
			},
		},
		{
			name: "mixed valid and invalid files",
			files: map[string]string{
				"valid.tf": `
resource "aws_instance" "good" {
  tags = {
    Name = "valid"
  }
}`,
				"invalid.tf": `
resource "aws_instance" {
  missing_name_label
}`,
			},
			wantCount: 1,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 1 {
					t.Errorf("Expected 1 valid resource, got %d", len(result.Resources))
				}
				if len(result.Errors) != 1 {
					t.Errorf("Expected 1 error, got %d", len(result.Errors))
				}
			},
		},
		{
			name: "nested directories",
			files: map[string]string{
				"main.tf": `resource "aws_instance" "root" { tags = { Level = "root" } }`,
				"modules/vpc/main.tf": `resource "aws_vpc" "main" { tags = { Level = "module" } }`,
				"modules/ec2/instance.tf": `resource "aws_instance" "app" { tags = { Level = "module" } }`,
			},
			wantCount: 3,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 3 {
					t.Errorf("Expected 3 resources, got %d", len(result.Resources))
				}
			},
		},
		{
			name: "tags with template expressions",
			files: map[string]string{
				"template_tags.tf": `
resource "aws_instance" "example" {
  tags = {
    Name = "instance-${var.environment}"
    Environment = var.environment
  }
}`,
			},
			wantCount: 1,
			wantErr:   false,
			check: func(t *testing.T, result *ParseResult) {
				// Template expressions won't be evaluated, should be empty or partial
				res := result.Resources[0]
				// Tags with variables won't be extracted as static strings
				if len(res.Tags) != 0 {
					t.Logf("Note: Tags with variables: %v", res.Tags)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tmpDir := t.TempDir()
			
			// Create files
			for relPath, content := range tt.files {
				fullPath := filepath.Join(tmpDir, relPath)
				dir := filepath.Dir(fullPath)
				
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("Failed to create directory %s: %v", dir, err)
				}
				
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write file %s: %v", fullPath, err)
				}
			}
			
			// Parse files
			result, err := ParseTerraformFiles([]string{tmpDir})
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTerraformFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, resources []Resource)
	}{
		{
			name: "tags with quoted keys",
			content: `
resource "aws_instance" "example" {
  tags = {
    "Name"        = "example"
    "Cost-Center" = "engineering"
  }
}`,
			wantErr: false,
			check: func(t *testing.T, resources []Resource) {
				if len(resources) != 1 {
					t.Fatalf("Expected 1 resource, got %d", len(resources))
				}
				res := resources[0]
				if res.Tags["Name"] != "example" {
					t.Errorf("Expected Name tag 'example', got %s", res.Tags["Name"])
				}
				if res.Tags["Cost-Center"] != "engineering" {
					t.Errorf("Expected Cost-Center tag 'engineering', got %s", res.Tags["Cost-Center"])
				}
			},
		},
		{
			name: "empty tags",
			content: `
resource "aws_instance" "example" {
  tags = {}
}`,
			wantErr: false,
			check: func(t *testing.T, resources []Resource) {
				res := resources[0]
				if len(res.Tags) != 0 {
					t.Errorf("Expected empty tags, got %v", res.Tags)
				}
			},
		},
		{
			name: "resource location tracking",
			content: `resource "aws_instance" "test" {
  tags = {
    Name = "test"
  }
}`,
			wantErr: false,
			check: func(t *testing.T, resources []Resource) {
				res := resources[0]
				if res.Location.Start.Line != 1 {
					t.Errorf("Expected resource to start at line 1, got %d", res.Location.Start.Line)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := filepath.Join(t.TempDir(), "test.tf")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			
			resources, err := parseFile(tmpFile)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, resources)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	// This test would require creating hclsyntax.Body objects directly
	// which is complex. The functionality is tested through the integration
	// tests above.
	t.Skip("Covered by integration tests")
}

func TestGetStringValue(t *testing.T) {
	// This test would require creating hclsyntax.Expression objects directly
	// which is complex. The functionality is tested through the integration
	// tests above.
	t.Skip("Covered by integration tests")
}