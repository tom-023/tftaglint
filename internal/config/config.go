package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Rules  []Rule `yaml:"rules"`
	Global Global `yaml:"global"`
}

type Rule struct {
	Name             string           `yaml:"name"`
	Description      string           `yaml:"description"`
	RequiredTags     []string         `yaml:"required_tags"`
	ForbiddenTags    []string         `yaml:"forbidden_tags"`
	Condition        *Condition       `yaml:"condition"`
	ResourceTypes    []string         `yaml:"resource_types"`
	TagConstraints   []TagConstraint  `yaml:"tag_constraints"`
	TagPatterns      []TagPattern     `yaml:"tag_patterns"`
}

type Condition struct {
	Tag   string `yaml:"tag"`
	Value string `yaml:"value"`
}

type TagConstraint struct {
	Tag           string   `yaml:"tag"`
	AllowedValues []string `yaml:"allowed_values"`
}

type TagPattern struct {
	Pattern string `yaml:"pattern"`
	Message string `yaml:"message"`
	regex   *regexp.Regexp
}

type Global struct {
	AlwaysRequiredTags  []string `yaml:"always_required_tags"`
	IgnoreResourceTypes []string `yaml:"ignore_resource_types"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Compile regex patterns
	for i := range config.Rules {
		for j := range config.Rules[i].TagPatterns {
			pattern := &config.Rules[i].TagPatterns[j]
			regex, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern in rule %s: %w", config.Rules[i].Name, err)
			}
			pattern.regex = regex
		}
	}

	return &config, nil
}

func (tp *TagPattern) Validate(tagName string) bool {
	if tp.regex == nil {
		return true
	}
	return tp.regex.MatchString(tagName)
}