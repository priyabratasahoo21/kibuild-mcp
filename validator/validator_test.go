package validator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateXML(t *testing.T) {
	tests := []struct {
		name        string
		xml         string
		projectPath string
		dbName      string
		expectPass  bool
		expectErr   int
		expectWarn  int
	}{
		{
			name: "Valid XML Snippet",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="89" name="Comment">
		<Text><![CDATA[# Perfect step]]></Text>
	</Step>
	<Step enable="True" id="6" name="Set Variable">
		<Value><Calculation><![CDATA["Hello"]]></Calculation></Value>
	</Step>
</fmxmlsnippet>`,
			expectPass: true,
		},
		{
			name: "Missing Root Wrapper",
			xml: `<Step enable="True" id="89" name="Comment">
	<Text><![CDATA[# Invalid snippet]]></Text>
</Step>`,
			expectPass: false,
			expectErr:  2, // Starts with + ends with wrapper errors
		},
		{
			name: "Namespace Pollution",
			xml: `<fmxmlsnippet type="FMObjectList" xmlns:ns0="http://www.filemaker.com/fmp">
	<ns0:Step enable="True" id="89" name="Comment"></ns0:Step>
</fmxmlsnippet>`,
			expectPass: false,
			expectErr:  2, // ns0 in attribute + ns0 in tag
		},
		{
			name: "Dynamic Identifier",
			xml: `<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="89" name="Comment" uuid="12345-abcde">
		<Text><![CDATA[# Dynamic uuid is bad]]></Text>
	</Step>
</fmxmlsnippet>`,
			expectPass: false,
			expectErr:  1,
		},
		{
			name: "Missing Calculation CDATA",
			xml: `<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="6" name="Set Variable">
		<Value><Calculation>Get ( CurrentDate )</Calculation></Value>
	</Step>
</fmxmlsnippet>`,
			expectPass: false,
			expectErr:  1,
		},
		{
			name: "Missing Step Enablement",
			xml: `<fmxmlsnippet type="FMObjectList">
	<Step id="89" name="Comment">
		<Text><![CDATA[# Lacks enable attribute]]></Text>
	</Step>
</fmxmlsnippet>`,
			expectPass: false,
			expectErr:  1,
		},
		{
			name: "XML Comment Warning",
			xml: `<fmxmlsnippet type="FMObjectList">
	<!-- This is an XML comment -->
	<Step enable="True" id="89" name="Comment">
		<Text><![CDATA[# OK]]></Text>
	</Step>
</fmxmlsnippet>`,
			expectPass: true, // Warnings do not fail validation
			expectWarn: 1,
		},
		{
			name: "Malformed XML Syntax Error",
			xml: `<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="89" name="Comment">
		<Text><![CDATA[# OK]]></Text>
	<!-- Missing closing tag for Step and fmxmlsnippet -->`,
			expectPass: false,
			expectErr:  2, // Ends with wrapper error + XML syntax error
			expectWarn: 1,
		},
		{
			name: "Panic Safety - Out of Order Tags",
			xml: `<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="6" name="Set Variable">
		</Calculation> <Calculation> Get(CurrentDate) </Calculation>
	</Step>
</fmxmlsnippet>`,
			expectPass: false,
			expectErr:  1, // Missing CDATA wrapper, but shouldn't panic
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ValidateXML(tc.xml, tc.projectPath, tc.dbName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Passed != tc.expectPass {
				t.Errorf("res.Passed = %v; want %v. Errors: %+v", res.Passed, tc.expectPass, res.Errors)
			}
			if len(res.Errors) != tc.expectErr {
				t.Errorf("len(res.Errors) = %d; want %d. Errors: %+v", len(res.Errors), tc.expectErr, res.Errors)
			}
			if len(res.Warnings) != tc.expectWarn {
				t.Errorf("len(res.Warnings) = %d; want %d. Warnings: %+v", len(res.Warnings), tc.expectWarn, res.Warnings)
			}
		})
	}
}

func TestContextIntegrityValidation(t *testing.T) {
	// Create temporary project and index
	tempDir, err := os.MkdirTemp("", "kibuild-validator-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	indexDir := filepath.Join(tempDir, "files", "Schema", "Contacts", ".kibuild_index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write mock script index
	scripts := []map[string]interface{}{
		{"name": "Save Customer"},
		{"name": "Delete Record"},
	}
	data, _ := json.Marshal(scripts)
	_ = os.WriteFile(filepath.Join(indexDir, "script_index.json"), data, 0644)

	// XML referencing existing and missing scripts
	xmlWithRefs := `<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="1" name="Perform Script">
		<Script name="Save Customer"></Script>
	</Step>
	<Step enable="True" id="1" name="Perform Script">
		<Script name="Nonexistent Script"></Script>
	</Step>
</fmxmlsnippet>`

	res, err := ValidateXML(xmlWithRefs, tempDir, "Contacts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pass because context integrity violations are only Warnings, not Errors
	if !res.Passed {
		t.Errorf("expected validation to pass, got failed")
	}

	// Should warn about the nonexistent script
	if len(res.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d. Warnings: %+v", len(res.Warnings), res.Warnings)
	} else {
		warn := res.Warnings[0]
		if warn.Rule != "context_integrity" {
			t.Errorf("expected rule 'context_integrity', got %q", warn.Rule)
		}
		if warn.Line != 6 {
			t.Errorf("expected warning on line 6, got line %d", warn.Line)
		}
		if !strings.Contains(warn.Message, "Nonexistent Script") {
			t.Errorf("unexpected warning message: %q", warn.Message)
		}
	}
}
