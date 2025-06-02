package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
)

// TerraformPlan represents the structure of terraform plan JSON output
type TerraformPlan struct {
	PlannedValues PlannedValues `json:"planned_values"`
}

type PlannedValues struct {
	RootModule RootModule `json:"root_module"`
}

type RootModule struct {
	Resources      []PlannedResource `json:"resources"`
	ChildModules   []ChildModule     `json:"child_modules"`
}

type ChildModule struct {
	Address        string            `json:"address"`
	Resources      []PlannedResource `json:"resources"`
	ChildModules   []ChildModule     `json:"child_modules"`
}

type PlannedResource struct {
	Address      string                 `json:"address"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Values       map[string]interface{} `json:"values"`
	ModulePath   []string              `json:"module_path,omitempty"`
}

// ParseTerraformPlan parses a terraform plan JSON file and extracts resources with tags
func ParseTerraformPlan(filename string) (*ParseResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	result := &ParseResult{
		Resources: []Resource{},
		Errors:    []error{},
	}

	// Process root module resources
	processModuleResources(&plan.PlannedValues.RootModule, filename, result)

	return result, nil
}

func processModuleResources(module *RootModule, filename string, result *ParseResult) {
	// Process resources in this module
	for _, resource := range module.Resources {
		res := convertPlannedResource(resource, filename)
		if res != nil {
			result.Resources = append(result.Resources, *res)
		}
	}

	// Process child modules recursively
	for _, child := range module.ChildModules {
		processChildModule(&child, filename, result)
	}
}

func processChildModule(module *ChildModule, filename string, result *ParseResult) {
	// Process resources in this module
	for _, resource := range module.Resources {
		res := convertPlannedResource(resource, filename)
		if res != nil {
			result.Resources = append(result.Resources, *res)
		}
	}

	// Process nested child modules recursively
	for _, child := range module.ChildModules {
		processChildModule(&child, filename, result)
	}
}

func convertPlannedResource(planned PlannedResource, filename string) *Resource {
	// Extract resource type and name from address
	// Format: module.name.resource_type.resource_name or resource_type.resource_name
	parts := strings.Split(planned.Address, ".")
	if len(parts) < 2 {
		return nil
	}

	// Find resource type and name
	resourceType := ""
	resourceName := ""
	
	// Look for the resource type (starts with provider prefix like aws_, google_, etc.)
	for i := 0; i < len(parts)-1; i++ {
		if isResourceType(parts[i]) {
			resourceType = parts[i]
			if i+1 < len(parts) {
				resourceName = parts[i+1]
			}
			break
		}
	}

	if resourceType == "" || resourceName == "" {
		// Fallback: assume last two parts are type and name
		if len(parts) >= 2 {
			resourceType = parts[len(parts)-2]
			resourceName = parts[len(parts)-1]
		} else {
			return nil
		}
	}

	resource := &Resource{
		Type: resourceType,
		Name: resourceName,
		Tags: extractTagsFromValues(planned.Values),
		Location: hcl.Range{
			Filename: filename,
			Start: hcl.Pos{
				Line:   1, // Plan doesn't have line numbers
				Column: 1,
				Byte:   0,
			},
			End: hcl.Pos{
				Line:   1,
				Column: 1,
				Byte:   0,
			},
		},
		File: filename,
	}

	return resource
}

func isResourceType(s string) bool {
	// Common cloud provider prefixes
	prefixes := []string{"aws_", "google_", "azurerm_", "alicloud_", "oci_"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func extractTagsFromValues(values map[string]interface{}) map[string]string {
	tags := make(map[string]string)

	// Look for "tags" field
	if tagsValue, ok := values["tags"]; ok {
		switch t := tagsValue.(type) {
		case map[string]interface{}:
			for k, v := range t {
				if str, ok := v.(string); ok {
					tags[k] = str
				}
			}
		}
	}

	// Also check for "tags_all" (AWS provider sometimes uses this)
	if tagsAllValue, ok := values["tags_all"]; ok {
		switch t := tagsAllValue.(type) {
		case map[string]interface{}:
			for k, v := range t {
				if str, ok := v.(string); ok {
					tags[k] = str
				}
			}
		}
	}

	return tags
}

