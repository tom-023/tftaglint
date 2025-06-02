package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type Resource struct {
	Type     string
	Name     string
	Tags     map[string]string
	Location hcl.Range
	File     string
}

type ParseResult struct {
	Resources []Resource
	Errors    []error
}

func ParseTerraformFiles(paths []string) (*ParseResult, error) {
	result := &ParseResult{
		Resources: []Resource{},
		Errors:    []error{},
	}

	for _, path := range paths {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !strings.HasSuffix(filePath, ".tf") || info.IsDir() {
				return nil
			}

			resources, err := parseFile(filePath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", filePath, err))
				return nil
			}

			result.Resources = append(result.Resources, resources...)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func parseFile(filename string) ([]Resource, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type")
	}

	var resources []Resource

	for _, block := range body.Blocks {
		if block.Type == "resource" && len(block.Labels) >= 2 {
			resource := Resource{
				Type:     block.Labels[0],
				Name:     block.Labels[1],
				Tags:     make(map[string]string),
				Location: block.DefRange(),
				File:     filename,
			}

			// Extract tags
			tags := extractTags(block.Body)
			resource.Tags = tags
			

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

func extractTags(body *hclsyntax.Body) map[string]string {
	tags := make(map[string]string)

	for name, attr := range body.Attributes {
		if name == "tags" {
			extractTagsFromExpression(attr.Expr, tags)
		}
	}

	// Also check for tags block
	for _, block := range body.Blocks {
		if block.Type == "tags" {
			for name, attr := range block.Body.Attributes {
				if val, ok := getStringValue(attr.Expr); ok {
					tags[name] = val
				}
			}
		}
	}

	return tags
}

func extractTagsFromExpression(expr hclsyntax.Expression, tags map[string]string) {
	switch e := expr.(type) {
	case *hclsyntax.ObjectConsExpr:
		for _, item := range e.Items {
			var key string
			var keyOk bool
			
			// Handle different key expression types
			switch k := item.KeyExpr.(type) {
			case *hclsyntax.ObjectConsKeyExpr:
				if wrapped, ok := k.Wrapped.(*hclsyntax.ScopeTraversalExpr); ok && len(wrapped.Traversal) == 1 {
					if root, ok := wrapped.Traversal[0].(hcl.TraverseRoot); ok {
						key = root.Name
						keyOk = true
					}
				} else {
					key, keyOk = getStringValue(k.Wrapped)
				}
			case *hclsyntax.ScopeTraversalExpr:
				if len(k.Traversal) == 1 {
					if root, ok := k.Traversal[0].(hcl.TraverseRoot); ok {
						key = root.Name
						keyOk = true
					}
				}
			case *hclsyntax.LiteralValueExpr:
				key, keyOk = getStringValue(k)
			case *hclsyntax.TemplateExpr:
				key, keyOk = getStringValue(k)
			}
			
			if keyOk {
				if val, ok := getStringValue(item.ValueExpr); ok {
					tags[key] = val
				}
			}
		}
	}
}

func getStringValue(expr hclsyntax.Expression) (string, bool) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		if e.Val.Type() == cty.String {
			return e.Val.AsString(), true
		}
	case *hclsyntax.TemplateExpr:
		if e.IsStringLiteral() {
			val, _ := e.Value(nil)
			if val.Type() == cty.String {
				return val.AsString(), true
			}
		}
	}
	return "", false
}