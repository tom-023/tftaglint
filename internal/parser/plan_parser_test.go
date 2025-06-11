package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseTerraformPlan(t *testing.T) {
	tests := []struct {
		name    string
		plan    interface{}
		wantErr bool
		check   func(t *testing.T, result *ParseResult)
	}{
		{
			name: "simple plan with root module resources",
			plan: TerraformPlan{
				PlannedValues: PlannedValues{
					RootModule: RootModule{
						Resources: []PlannedResource{
							{
								Address: "aws_instance.web",
								Type:    "aws_instance",
								Name:    "web",
								Values: map[string]interface{}{
									"instance_type": "t2.micro",
									"tags": map[string]interface{}{
										"Name":        "web-server",
										"Environment": "production",
									},
								},
							},
							{
								Address: "aws_s3_bucket.data",
								Type:    "aws_s3_bucket",
								Name:    "data",
								Values: map[string]interface{}{
									"bucket": "my-data-bucket",
									"tags": map[string]interface{}{
										"Purpose": "data-storage",
										"Owner":   "data-team",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 2 {
					t.Fatalf("Expected 2 resources, got %d", len(result.Resources))
				}
				
				// Check first resource
				res1 := result.Resources[0]
				if res1.Type != "aws_instance" || res1.Name != "web" {
					t.Errorf("Expected aws_instance.web, got %s.%s", res1.Type, res1.Name)
				}
				expectedTags1 := map[string]string{
					"Name":        "web-server",
					"Environment": "production",
				}
				if !reflect.DeepEqual(res1.Tags, expectedTags1) {
					t.Errorf("Resource 1 tags mismatch. Expected %v, got %v", expectedTags1, res1.Tags)
				}
				
				// Check second resource
				res2 := result.Resources[1]
				if res2.Type != "aws_s3_bucket" || res2.Name != "data" {
					t.Errorf("Expected aws_s3_bucket.data, got %s.%s", res2.Type, res2.Name)
				}
				expectedTags2 := map[string]string{
					"Purpose": "data-storage",
					"Owner":   "data-team",
				}
				if !reflect.DeepEqual(res2.Tags, expectedTags2) {
					t.Errorf("Resource 2 tags mismatch. Expected %v, got %v", expectedTags2, res2.Tags)
				}
			},
		},
		{
			name: "plan with child modules",
			plan: TerraformPlan{
				PlannedValues: PlannedValues{
					RootModule: RootModule{
						Resources: []PlannedResource{
							{
								Address: "aws_instance.root",
								Type:    "aws_instance",
								Name:    "root",
								Values: map[string]interface{}{
									"tags": map[string]interface{}{
										"Level": "root",
									},
								},
							},
						},
						ChildModules: []ChildModule{
							{
								Address: "module.vpc",
								Resources: []PlannedResource{
									{
										Address: "module.vpc.aws_vpc.main",
										Type:    "aws_vpc",
										Name:    "main",
										Values: map[string]interface{}{
											"cidr_block": "10.0.0.0/16",
											"tags": map[string]interface{}{
												"Name":   "main-vpc",
												"Module": "vpc",
											},
										},
									},
									{
										Address: "module.vpc.aws_subnet.public",
										Type:    "aws_subnet",
										Name:    "public",
										Values: map[string]interface{}{
											"tags": map[string]interface{}{
												"Type": "public",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 3 {
					t.Fatalf("Expected 3 resources, got %d", len(result.Resources))
				}
				
				// Create a map for easier checking
				resourceMap := make(map[string]Resource)
				for _, res := range result.Resources {
					key := res.Type + "." + res.Name
					resourceMap[key] = res
				}
				
				// Check root resource
				if root, ok := resourceMap["aws_instance.root"]; ok {
					if root.Tags["Level"] != "root" {
						t.Errorf("Expected root Level tag 'root', got %s", root.Tags["Level"])
					}
				} else {
					t.Error("Root instance not found")
				}
				
				// Check VPC resource
				if vpc, ok := resourceMap["aws_vpc.main"]; ok {
					if vpc.Tags["Module"] != "vpc" {
						t.Errorf("Expected vpc Module tag 'vpc', got %s", vpc.Tags["Module"])
					}
				} else {
					t.Error("VPC resource not found")
				}
				
				// Check subnet resource
				if subnet, ok := resourceMap["aws_subnet.public"]; ok {
					if subnet.Tags["Type"] != "public" {
						t.Errorf("Expected subnet Type tag 'public', got %s", subnet.Tags["Type"])
					}
				} else {
					t.Error("Subnet resource not found")
				}
			},
		},
		{
			name: "nested child modules",
			plan: TerraformPlan{
				PlannedValues: PlannedValues{
					RootModule: RootModule{
						ChildModules: []ChildModule{
							{
								Address: "module.network",
								Resources: []PlannedResource{
									{
										Address: "module.network.aws_vpc.main",
										Type:    "aws_vpc",
										Name:    "main",
										Values: map[string]interface{}{
											"tags": map[string]interface{}{
												"Level": "1",
											},
										},
									},
								},
								ChildModules: []ChildModule{
									{
										Address: "module.network.module.subnets",
										Resources: []PlannedResource{
											{
												Address: "module.network.module.subnets.aws_subnet.private",
												Type:    "aws_subnet",
												Name:    "private",
												Values: map[string]interface{}{
													"tags": map[string]interface{}{
														"Level": "2",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 2 {
					t.Fatalf("Expected 2 resources, got %d", len(result.Resources))
				}
			},
		},
		{
			name: "resources with tags_all",
			plan: TerraformPlan{
				PlannedValues: PlannedValues{
					RootModule: RootModule{
						Resources: []PlannedResource{
							{
								Address: "aws_instance.example",
								Type:    "aws_instance",
								Name:    "example",
								Values: map[string]interface{}{
									"tags": map[string]interface{}{
										"Name": "example",
									},
									"tags_all": map[string]interface{}{
										"Name":        "example",
										"Environment": "dev",
										"ManagedBy":   "terraform",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ParseResult) {
				res := result.Resources[0]
				// Should include tags from tags_all
				if res.Tags["ManagedBy"] != "terraform" {
					t.Errorf("Expected ManagedBy tag from tags_all, got %v", res.Tags)
				}
			},
		},
		{
			name: "resources without tags",
			plan: TerraformPlan{
				PlannedValues: PlannedValues{
					RootModule: RootModule{
						Resources: []PlannedResource{
							{
								Address: "aws_iam_role.example",
								Type:    "aws_iam_role",
								Name:    "example",
								Values: map[string]interface{}{
									"name": "example-role",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, result *ParseResult) {
				res := result.Resources[0]
				if len(res.Tags) != 0 {
					t.Errorf("Expected no tags, got %v", res.Tags)
				}
			},
		},
		{
			name:    "invalid json",
			plan:    "invalid json string",
			wantErr: true,
		},
		{
			name:    "empty plan",
			plan:    TerraformPlan{},
			wantErr: false,
			check: func(t *testing.T, result *ParseResult) {
				if len(result.Resources) != 0 {
					t.Errorf("Expected no resources, got %d", len(result.Resources))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := filepath.Join(t.TempDir(), "plan.json")
			
			// Marshal plan to JSON
			var data []byte
			var err error
			if str, ok := tt.plan.(string); ok {
				data = []byte(str)
			} else {
				data, err = json.MarshalIndent(tt.plan, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal test plan: %v", err)
				}
			}
			
			if err := os.WriteFile(tmpFile, data, 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}
			
			// Parse the plan
			result, err := ParseTerraformPlan(tmpFile)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTerraformPlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func TestParseTerraformPlan_FileNotFound(t *testing.T) {
	_, err := ParseTerraformPlan("/non/existent/file.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestConvertPlannedResource(t *testing.T) {
	tests := []struct {
		name     string
		resource PlannedResource
		wantType string
		wantName string
		wantNil  bool
	}{
		{
			name: "simple resource address",
			resource: PlannedResource{
				Address: "aws_instance.web",
				Values: map[string]interface{}{
					"tags": map[string]interface{}{
						"Name": "web",
					},
				},
			},
			wantType: "aws_instance",
			wantName: "web",
			wantNil:  false,
		},
		{
			name: "module resource address",
			resource: PlannedResource{
				Address: "module.vpc.aws_vpc.main",
				Values:  map[string]interface{}{},
			},
			wantType: "aws_vpc",
			wantName: "main",
			wantNil:  false,
		},
		{
			name: "nested module resource",
			resource: PlannedResource{
				Address: "module.network.module.subnets.aws_subnet.private",
				Values:  map[string]interface{}{},
			},
			wantType: "aws_subnet",
			wantName: "private",
			wantNil:  false,
		},
		{
			name: "google cloud resource",
			resource: PlannedResource{
				Address: "google_compute_instance.example",
				Values:  map[string]interface{}{},
			},
			wantType: "google_compute_instance",
			wantName: "example",
			wantNil:  false,
		},
		{
			name: "azure resource in module",
			resource: PlannedResource{
				Address: "module.compute.azurerm_virtual_machine.main",
				Values:  map[string]interface{}{},
			},
			wantType: "azurerm_virtual_machine",
			wantName: "main",
			wantNil:  false,
		},
		{
			name: "invalid address - too short",
			resource: PlannedResource{
				Address: "invalid",
				Values:  map[string]interface{}{},
			},
			wantNil: true,
		},
		{
			name: "data source address",
			resource: PlannedResource{
				Address: "data.aws_ami.ubuntu",
				Values:  map[string]interface{}{},
			},
			wantType: "aws_ami",
			wantName: "ubuntu",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPlannedResource(tt.resource, "test.json")
			
			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil result, got %+v", result)
				}
				return
			}
			
			if result == nil {
				t.Fatal("Expected non-nil result, got nil")
			}
			
			if result.Type != tt.wantType {
				t.Errorf("Expected type %s, got %s", tt.wantType, result.Type)
			}
			
			if result.Name != tt.wantName {
				t.Errorf("Expected name %s, got %s", tt.wantName, result.Name)
			}
		})
	}
}

func TestIsResourceType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"aws_instance", true},
		{"aws_s3_bucket", true},
		{"google_compute_instance", true},
		{"azurerm_virtual_machine", true},
		{"alicloud_instance", true},
		{"oci_core_instance", true},
		{"module", false},
		{"data", false},
		{"instance", false},
		{"my_custom_resource", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isResourceType(tt.input)
			if got != tt.want {
				t.Errorf("isResourceType(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractTagsFromValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]interface{}
		want   map[string]string
	}{
		{
			name: "tags field only",
			values: map[string]interface{}{
				"tags": map[string]interface{}{
					"Name":        "test",
					"Environment": "dev",
				},
			},
			want: map[string]string{
				"Name":        "test",
				"Environment": "dev",
			},
		},
		{
			name: "tags_all field only",
			values: map[string]interface{}{
				"tags_all": map[string]interface{}{
					"Owner": "team-a",
					"Cost":  "100",
				},
			},
			want: map[string]string{
				"Owner": "team-a",
				"Cost":  "100",
			},
		},
		{
			name: "both tags and tags_all",
			values: map[string]interface{}{
				"tags": map[string]interface{}{
					"Name": "test",
				},
				"tags_all": map[string]interface{}{
					"Name":        "test",
					"Environment": "dev",
				},
			},
			want: map[string]string{
				"Name":        "test",
				"Environment": "dev",
			},
		},
		{
			name: "no tags",
			values: map[string]interface{}{
				"instance_type": "t2.micro",
			},
			want: map[string]string{},
		},
		{
			name: "non-string tag values ignored",
			values: map[string]interface{}{
				"tags": map[string]interface{}{
					"Name":    "test",
					"Count":   123,
					"Enabled": true,
				},
			},
			want: map[string]string{
				"Name": "test",
			},
		},
		{
			name: "tags not a map",
			values: map[string]interface{}{
				"tags": "invalid",
			},
			want: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTagsFromValues(tt.values)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractTagsFromValues() = %v, want %v", got, tt.want)
			}
		})
	}
}