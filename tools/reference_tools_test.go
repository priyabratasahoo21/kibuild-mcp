package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceTools(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_ref_tools_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create exploded Schema structure
	dbName := "Contacts"
	schemaRoot := filepath.Join(tempDir, "Schema", dbName)
	err = os.MkdirAll(filepath.Join(schemaRoot, "scripts"), 0755)
	if err != nil {
		t.Fatalf("failed to create scripts dir: %v", err)
	}
	_ = os.MkdirAll(filepath.Join(schemaRoot, "scripts_sanitized"), 0755)
	_ = os.MkdirAll(filepath.Join(schemaRoot, "layouts"), 0755)
	_ = os.MkdirAll(filepath.Join(schemaRoot, "tables"), 0755)
	_ = os.MkdirAll(filepath.Join(schemaRoot, "relationships"), 0755)

	// 1. Write mock table XML (with calculations)
	tableXML := `
<Table name="ContactTable">
	<Field name="ID" datatype="Number" fieldtype="Normal" />
	<Field name="FullName" datatype="Text" fieldtype="Calculation">
		<Calculation>FirstName &amp; " " &amp; LastName</Calculation>
	</Field>
	<Field name="ModifiedBy" datatype="Text" fieldtype="Normal">
		<AutoEnter>Get(AccountName)</AutoEnter>
	</Field>
</Table>
`
	_ = os.WriteFile(filepath.Join(schemaRoot, "tables", "ContactTable.xml"), []byte(tableXML), 0644)

	// 2. Write mock layout XML
	layoutXML := `
<Layout name="ContactDetail">
	<TableOccurrenceReference name="ContactTO" />
	<FieldReference name="ContactTO::FullName" />
	<FieldReference name="ContactTO::ID" />
	<ScriptReference name="TriggerOnRecordLoad" />
	<ValueListReference name="ContactStatusList" />
</Layout>
`
	_ = os.WriteFile(filepath.Join(schemaRoot, "layouts", "ContactDetail.xml"), []byte(layoutXML), 0644)

	// 3. Write mock script XML & TXT
	scriptXML := `
<Script name="TriggerOnRecordLoad" id="1">
	<LayoutReference name="ContactDetail" />
	<ValueListReference name="ContactStatusList" />
	<FieldReference name="ContactTO::ID" />
	<TableOccurrenceReference name="ContactTO" />
</Script>
`
	_ = os.WriteFile(filepath.Join(schemaRoot, "scripts", "TriggerOnRecordLoad.xml"), []byte(scriptXML), 0644)

	scriptTxt := `
Perform Script [ Specified: From list ; "HelperScript" ; Parameter: $myVar ]
Set Variable [ $myVar ; Value: ContactTO::ID ]
`
	_ = os.WriteFile(filepath.Join(schemaRoot, "scripts_sanitized", "TriggerOnRecordLoad.txt"), []byte(scriptTxt), 0644)

	// 4. Write mock relationship XML
	relationXML := `
<Relationship name="Contact_Owner">
	<LeftTable>
		<TableOccurrenceReference name="ContactTO" />
	</LeftTable>
	<RightTable>
		<TableOccurrenceReference name="OwnerTO" />
	</RightTable>
	<JoinPredicate type="=">
		<LeftField>
			<FieldReference name="OwnerID" />
		</LeftField>
		<RightField>
			<FieldReference name="ID" />
		</RightField>
	</JoinPredicate>
</Relationship>
`
	_ = os.WriteFile(filepath.Join(schemaRoot, "relationships", "Contact_Owner.xml"), []byte(relationXML), 0644)

	// Test 1: FindLayoutReferencesToScripts
	res, err := FindLayoutReferencesToScripts(tempDir, []string{"ContactDetail"}, dbName)
	if err != nil {
		t.Fatalf("FindLayoutReferencesToScripts failed: %v", err)
	}
	var resp ToolReferencesResponse
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if resp.Count != 1 || resp.Matches[0].Snippet != "Triggers/Buttons script: TriggerOnRecordLoad" {
		t.Errorf("Unexpected layout scripts response: %v", res)
	}

	// Test 2: FindLayoutReferencesToValueLists
	res, err = FindLayoutReferencesToValueLists(tempDir, []string{"ContactDetail"}, dbName)
	if err != nil {
		t.Fatalf("FindLayoutReferencesToValueLists failed: %v", err)
	}
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if resp.Count != 1 || !strings.Contains(resp.Matches[0].Snippet, "ContactStatusList") {
		t.Errorf("Unexpected layout value lists response: %v", res)
	}

	// Test 3: FindScriptReferencesInScripts
	res, err = FindScriptReferencesInScripts(tempDir, []string{"HelperScript", "TriggerOnRecordLoad"}, dbName) // Batch test
	if err != nil {
		t.Fatalf("FindScriptReferencesInScripts failed: %v", err)
	}
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if resp.Count != 1 || !strings.Contains(resp.Matches[0].Snippet, "HelperScript") {
		t.Errorf("Unexpected script reference in scripts response: %v", res)
	}

	// Test 4: FindFieldReferencesInCalculations
	res, err = FindFieldReferencesInCalculations(tempDir, []string{"FirstName"}, dbName)
	if err != nil {
		t.Fatalf("FindFieldReferencesInCalculations failed: %v", err)
	}
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if resp.Count != 1 || resp.Matches[0].Name != "ContactTable::FullName" {
		t.Errorf("Unexpected field calculations reference: %v", res)
	}

	// Test 5: FindVariableReferencesInScripts
	res, err = FindVariableReferencesInScripts(tempDir, []string{"$myVar"}, dbName)
	if err != nil {
		t.Fatalf("FindVariableReferencesInScripts failed: %v", err)
	}
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if resp.Count != 2 || !strings.Contains(resp.Matches[0].Snippet, "$myVar") {
		t.Errorf("Unexpected variable reference: %v", res)
	}

	// Test 6: FindTOReferences
	res, err = FindTOReferences(tempDir, []string{"ContactTO"}, dbName)
	if err != nil {
		t.Fatalf("FindTOReferences failed: %v", err)
	}
	if err := json.Unmarshal([]byte(res), &resp); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	// It matches layouts, scripts, and relationships
	if resp.Count < 3 {
		t.Errorf("Expected at least 3 Table Occurrence matches, got %d. Response: %v", resp.Count, res)
	}
}
