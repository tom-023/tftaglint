package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/tom-023/tftaglint/internal/validator"
)

type Reporter struct {
	writer io.Writer
}

func NewReporter(w io.Writer) *Reporter {
	return &Reporter{writer: w}
}

func (r *Reporter) Report(violations []validator.Violation) error {
	if len(violations) == 0 {
		fmt.Fprintln(r.writer, "✅ No tag violations found!")
		return nil
	}

	// Group violations by file
	violationsByFile := make(map[string][]validator.Violation)
	for _, v := range violations {
		violationsByFile[v.Resource.File] = append(violationsByFile[v.Resource.File], v)
	}

	// Sort files for consistent output
	var files []string
	for file := range violationsByFile {
		files = append(files, file)
	}
	sort.Strings(files)

	fmt.Fprintf(r.writer, "❌ Found %d tag violation(s):\n\n", len(violations))

	for _, file := range files {
		fileViolations := violationsByFile[file]
		fmt.Fprintf(r.writer, "📄 %s\n", file)

		// Sort violations by line number
		sort.Slice(fileViolations, func(i, j int) bool {
			return fileViolations[i].Resource.Location.Start.Line < fileViolations[j].Resource.Location.Start.Line
		})

		for _, v := range fileViolations {
			r.reportViolation(v)
		}
		fmt.Fprintln(r.writer)
	}

	return nil
}

func (r *Reporter) reportViolation(v validator.Violation) {
	location := v.Resource.Location.Start
	fmt.Fprintf(r.writer, "  Line %d: %s.%s\n", location.Line, v.Resource.Type, v.Resource.Name)
	fmt.Fprintf(r.writer, "    Rule: %s\n", v.Rule)
	fmt.Fprintf(r.writer, "    Message: %s\n", v.Message)
	if v.Description != "" {
		fmt.Fprintf(r.writer, "    Description: %s\n", v.Description)
	}
}

func (r *Reporter) ReportSummary(violations []validator.Violation) error {
	if len(violations) == 0 {
		return nil
	}

	// Count violations by rule
	violationsByRule := make(map[string]int)
	for _, v := range violations {
		violationsByRule[v.Rule]++
	}

	fmt.Fprintln(r.writer, strings.Repeat("-", 50))
	fmt.Fprintln(r.writer, "Summary:")
	fmt.Fprintf(r.writer, "Total violations: %d\n", len(violations))
	fmt.Fprintln(r.writer, "\nViolations by rule:")

	// Sort rules for consistent output
	var rules []string
	for rule := range violationsByRule {
		rules = append(rules, rule)
	}
	sort.Strings(rules)

	for _, rule := range rules {
		fmt.Fprintf(r.writer, "  %s: %d\n", rule, violationsByRule[rule])
	}

	return nil
}