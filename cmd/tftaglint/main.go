package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tom-023/tftaglint/internal/config"
	"github.com/tom-023/tftaglint/internal/parser"
	"github.com/tom-023/tftaglint/internal/reporter"
	"github.com/tom-023/tftaglint/internal/validator"
)

var (
	configFile  string
	showSummary bool
	planFile    string
)

var rootCmd = &cobra.Command{
	Use:   "tftaglint",
	Short: "A linter for Terraform resource tags",
	Long:  `tftaglint validates Terraform resource tags against defined rules to ensure consistency and compliance.`,
}

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate Terraform files for tag violations",
	Long:  `Validate Terraform files in the specified paths (or current directory) for tag violations based on the rules defined in the configuration file.`,
	Args:  cobra.ArbitraryArgs,
	RunE:  runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&configFile, "config", "c", "tag-rules.yaml", "Path to the configuration file")
	validateCmd.Flags().StringVarP(&configFile, "file", "f", "tag-rules.yaml", "Path to the configuration file (alias for --config)")
	validateCmd.Flags().BoolVarP(&showSummary, "summary", "s", false, "Show summary of violations")
	validateCmd.Flags().StringVarP(&planFile, "plan", "p", "", "Path to terraform plan JSON file (use instead of .tf files)")
	rootCmd.AddCommand(validateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var parseResult *parser.ParseResult

	// Check if plan file is provided
	if planFile != "" {
		// Parse terraform plan JSON
		parseResult, err = parser.ParseTerraformPlan(planFile)
		if err != nil {
			return fmt.Errorf("failed to parse terraform plan: %w", err)
		}
	} else {
		// Default to current directory if no paths specified
		paths := args
		if len(paths) == 0 {
			paths = []string{"."}
		}

		// Parse Terraform files
		parseResult, err = parser.ParseTerraformFiles(paths)
		if err != nil {
			return fmt.Errorf("failed to parse Terraform files: %w", err)
		}
	}

	// Report parsing errors if any
	if len(parseResult.Errors) > 0 {
		fmt.Fprintln(os.Stderr, "⚠️  Parsing errors encountered:")
		for _, err := range parseResult.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Validate resources
	v := validator.NewValidator(cfg)
	violations := v.Validate(parseResult.Resources)

	// Report violations
	r := reporter.NewReporter(os.Stdout)
	if err := r.Report(violations); err != nil {
		return fmt.Errorf("failed to report violations: %w", err)
	}

	// Show summary if requested
	if showSummary && len(violations) > 0 {
		if err := r.ReportSummary(violations); err != nil {
			return fmt.Errorf("failed to report summary: %w", err)
		}
	}

	// Exit with non-zero status if violations found
	if len(violations) > 0 {
		// Return error to allow cobra to handle exit
		return fmt.Errorf("found %d tag violations", len(violations))
	}

	return nil
}