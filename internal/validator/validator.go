package validator

import (
	"fmt"
	"strings"

	"github.com/tom-023/tftaglint/internal/config"
	"github.com/tom-023/tftaglint/internal/parser"
)

type Violation struct {
	Rule        string
	Description string
	Resource    parser.Resource
	Message     string
}

type Validator struct {
	config *config.Config
}

func NewValidator(cfg *config.Config) *Validator {
	return &Validator{config: cfg}
}

func (v *Validator) Validate(resources []parser.Resource) []Violation {
	var violations []Violation

	for _, resource := range resources {
		// Check if resource type should be ignored
		if v.shouldIgnoreResource(resource.Type) {
			continue
		}

		// Check global required tags
		violations = append(violations, v.checkGlobalRequiredTags(resource)...)

		// Check each rule
		for _, rule := range v.config.Rules {
			violations = append(violations, v.checkRule(resource, rule)...)
		}
	}

	return violations
}

func (v *Validator) shouldIgnoreResource(resourceType string) bool {
	for _, ignored := range v.config.Global.IgnoreResourceTypes {
		if resourceType == ignored {
			return true
		}
	}
	return false
}

func (v *Validator) checkGlobalRequiredTags(resource parser.Resource) []Violation {
	var violations []Violation

	for _, requiredTag := range v.config.Global.AlwaysRequiredTags {
		if _, exists := resource.Tags[requiredTag]; !exists {
			violations = append(violations, Violation{
				Rule:        "global-required-tags",
				Description: "Global required tags",
				Resource:    resource,
				Message:     fmt.Sprintf("Missing required tag: %s", requiredTag),
			})
		}
	}

	return violations
}

func (v *Validator) checkRule(resource parser.Resource, rule config.Rule) []Violation {
	var violations []Violation

	// Check if rule applies to this resource type
	if len(rule.ResourceTypes) > 0 && !v.isResourceTypeInList(resource.Type, rule.ResourceTypes) {
		return violations
	}

	// Check condition
	if rule.Condition != nil && !v.checkCondition(resource, rule.Condition) {
		return violations
	}

	// Check required tags
	for _, requiredTag := range rule.RequiredTags {
		if _, exists := resource.Tags[requiredTag]; !exists {
			violations = append(violations, Violation{
				Rule:        rule.Name,
				Description: rule.Description,
				Resource:    resource,
				Message:     fmt.Sprintf("Missing required tag: %s", requiredTag),
			})
		}
	}

	// Check forbidden tags
	for _, forbiddenTag := range rule.ForbiddenTags {
		if _, exists := resource.Tags[forbiddenTag]; exists {
			violations = append(violations, Violation{
				Rule:        rule.Name,
				Description: rule.Description,
				Resource:    resource,
				Message:     fmt.Sprintf("Forbidden tag found: %s", forbiddenTag),
			})
		}
	}

	// Check tag constraints
	for _, constraint := range rule.TagConstraints {
		if value, exists := resource.Tags[constraint.Tag]; exists {
			if !v.isValueAllowed(value, constraint.AllowedValues) {
				violations = append(violations, Violation{
					Rule:        rule.Name,
					Description: rule.Description,
					Resource:    resource,
					Message:     fmt.Sprintf("Invalid value for tag %s: '%s'. Allowed values: %s", 
						constraint.Tag, value, strings.Join(constraint.AllowedValues, ", ")),
				})
			}
		}
	}

	// Check tag patterns
	for tagName := range resource.Tags {
		for _, pattern := range rule.TagPatterns {
			if !pattern.Validate(tagName) {
				violations = append(violations, Violation{
					Rule:        rule.Name,
					Description: rule.Description,
					Resource:    resource,
					Message:     fmt.Sprintf("Tag name '%s' does not match pattern: %s", tagName, pattern.Message),
				})
			}
		}
	}

	return violations
}

func (v *Validator) isResourceTypeInList(resourceType string, list []string) bool {
	for _, t := range list {
		if t == resourceType {
			return true
		}
	}
	return false
}

func (v *Validator) checkCondition(resource parser.Resource, condition *config.Condition) bool {
	value, exists := resource.Tags[condition.Tag]
	return exists && value == condition.Value
}

func (v *Validator) isValueAllowed(value string, allowedValues []string) bool {
	for _, allowed := range allowedValues {
		if value == allowed {
			return true
		}
	}
	return false
}