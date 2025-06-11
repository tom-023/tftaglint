package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunValidate(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	tests := []struct {
		name         string
		configFile   string
		configContent string
		tfFiles      map[string]string
		planFile     string
		planContent  string
		args         []string
		wantOutput   []string
		wantErr      bool
		setupFunc    func() error
		cleanupFunc  func()
	}{
		{
			name: "valid terraform files with no violations",
			configContent: `
global:
  always_required_tags:
    - Name
    - Environment
rules:
  - name: Basic Tags
    description: Basic required tags
    required_tags:
      - Owner`,
			tfFiles: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  instance_type = "t2.micro"
  tags = {
    Name        = "web-server"
    Environment = "prod"
    Owner       = "team-a"
  }
}`,
			},
			wantOutput: []string{"✅ No tag violations found!"},
			wantErr:    false,
		},
		{
			name: "terraform files with violations",
			configContent: `
global:
  always_required_tags:
    - Name
    - Environment`,
			tfFiles: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  instance_type = "t2.micro"
  tags = {
    Name = "web-server"
  }
}`,
			},
			wantOutput: []string{
				"❌ Found",
				"tag violation(s):",
				"Missing required tag: Environment",
			},
			wantErr: true, // runValidate returns error when violations found
		},
		{
			name: "plan file validation",
			configContent: `
global:
  always_required_tags:
    - Name`,
			planContent: `{
  "planned_values": {
    "root_module": {
      "resources": [
        {
          "address": "aws_instance.web",
          "type": "aws_instance",
          "name": "web",
          "values": {
            "tags": {
              "Name": "web-server"
            }
          }
        }
      ]
    }
  }
}`,
			wantOutput: []string{"✅ No tag violations found!"},
			wantErr:    false,
		},
		{
			name: "invalid config file",
			configContent: `invalid yaml content: [`,
			tfFiles: map[string]string{
				"main.tf": `resource "aws_instance" "web" {}`,
			},
			wantErr: true,
		},
		{
			name: "parsing errors in terraform files",
			configContent: `rules: []`,
			tfFiles: map[string]string{
				"invalid.tf": `resource "aws_instance" {
  missing_name
}`,
			},
			wantOutput: []string{
				"⚠️  Parsing errors encountered:",
			},
			wantErr: false,
		},
		{
			name: "with summary flag",
			configContent: `
global:
  always_required_tags:
    - Name`,
			tfFiles: map[string]string{
				"main.tf": `
resource "aws_instance" "web" {
  tags = {}
}
resource "aws_instance" "db" {
  tags = {}
}`,
			},
			setupFunc: func() error {
				showSummary = true
				return nil
			},
			cleanupFunc: func() {
				showSummary = false
			},
			wantOutput: []string{
				"Summary:",
				"Total violations: 2",
				"global-required-tags: 2",
			},
			wantErr: true, // violations found
		},
		{
			name: "config file not found",
			setupFunc: func() error {
				configFile = "/non/existent/config.yaml"
				return nil
			},
			cleanupFunc: func() {
				configFile = "tag-rules.yaml"
			},
			wantErr: true,
		},
		{
			name: "plan file not found",
			configContent: `rules: []`,
			setupFunc: func() error {
				planFile = "/non/existent/plan.json"
				return nil
			},
			cleanupFunc: func() {
				planFile = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			
			// Setup
			if tt.setupFunc != nil {
				if err := tt.setupFunc(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}
			
			// Cleanup
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}
			
			// Create config file
			if tt.configContent != "" {
				configPath := filepath.Join(tmpDir, "config.yaml")
				if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
				configFile = configPath
				defer func() { configFile = "tag-rules.yaml" }()
			}
			
			// Create terraform files
			for name, content := range tt.tfFiles {
				tfPath := filepath.Join(tmpDir, name)
				if err := os.WriteFile(tfPath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write terraform file: %v", err)
				}
			}
			
			// Create plan file
			if tt.planContent != "" {
				planPath := filepath.Join(tmpDir, "plan.json")
				if err := os.WriteFile(planPath, []byte(tt.planContent), 0644); err != nil {
					t.Fatalf("Failed to write plan file: %v", err)
				}
				planFile = planPath
				defer func() { planFile = "" }()
			}
			
			// Capture stdout and stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = wErr
			
			// Prepare arguments
			args := tt.args
			if len(args) == 0 && len(tt.tfFiles) > 0 && tt.planContent == "" {
				args = []string{tmpDir}
			}
			
			// Run command
			cmd := &cobra.Command{}
			err := runValidate(cmd, args)
			
			// Close writers and read output
			w.Close()
			wErr.Close()
			var buf bytes.Buffer
			var bufErr bytes.Buffer
			io.Copy(&buf, r)
			io.Copy(&bufErr, rErr)
			output := buf.String() + bufErr.String()
			
			// Restore stderr
			os.Stderr = oldStderr
			
			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("runValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			// Check output
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", want, output)
				}
			}
		})
	}
}

func TestMain(t *testing.T) {
	// Test that the main function initializes properly
	// This is mainly to ensure coverage and that commands are registered
	
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	
	// Test help command
	os.Args = []string{"tftaglint", "--help"}
	
	// We can't directly test main() as it calls os.Exit
	// Instead, test that rootCmd is properly configured
	
	// Check that validate command exists
	var hasValidateCmd bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "validate" {
			hasValidateCmd = true
			break
		}
	}
	
	if !hasValidateCmd {
		t.Error("validate command not found in rootCmd")
	}
	
	// Check root command has proper description
	if rootCmd.Short == "" {
		t.Error("Root command short description is empty")
	}
}

func TestInit(t *testing.T) {
	// Test that flags are properly initialized
	var validateCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "validate" {
			validateCmd = cmd
			break
		}
	}
	
	if validateCmd == nil {
		t.Fatal("validate command not found")
	}
	
	// Check flags
	configFlag := validateCmd.Flags().Lookup("config")
	if configFlag == nil {
		t.Error("config flag not found")
	}
	if configFlag.DefValue != "tag-rules.yaml" {
		t.Errorf("Expected default config file 'tag-rules.yaml', got %s", configFlag.DefValue)
	}
	
	fileFlag := validateCmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Error("file flag not found")
	}
	
	summaryFlag := validateCmd.Flags().Lookup("summary")
	if summaryFlag == nil {
		t.Error("summary flag not found")
	}
	
	planFlag := validateCmd.Flags().Lookup("plan")
	if planFlag == nil {
		t.Error("plan flag not found")
	}
}