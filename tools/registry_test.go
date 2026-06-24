package tools

import (
	"testing"
)

func TestGetToolsSchema(t *testing.T) {
	schemas := GetToolsSchema()
	if len(schemas) == 0 {
		t.Fatalf("expected tools schema list to be non-empty")
	}

	foundReadFile := false
	requiredTools := map[string]bool{
		"read_file":               false,
		"find_table":              false,
		"find_layout":             false,
		"inspect_relationships":   false,
		"validate_webviewer_html": false,
	}
	for _, s := range schemas {
		if _, ok := requiredTools[s.Name]; ok {
			requiredTools[s.Name] = true
			if s.Description == "" {
				t.Errorf("%s tool description is empty", s.Name)
			}
			if s.Parameters == nil {
				t.Errorf("%s tool parameters are nil", s.Name)
			}
		}
		if s.Name == "read_file" {
			foundReadFile = true
			if s.Description == "" {
				t.Errorf("read_file tool description is empty")
			}
			if s.Parameters == nil {
				t.Errorf("read_file tool parameters are nil")
			}
		}
	}

	if !foundReadFile {
		t.Errorf("expected read_file tool to be in registry schemas")
	}

	for name, found := range requiredTools {
		if !found {
			t.Errorf("expected %s tool to be in registry schemas", name)
		}
	}
}
