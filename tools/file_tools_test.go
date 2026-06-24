package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAndSandboxPathTraversals(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_file_tools_sandbox_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		projectPath string
		inputPath   string
		expectError bool
	}{
		{
			name:        "Relative path inside project",
			projectPath: tempDir,
			inputPath:   "foo/bar/baz.txt",
			expectError: false,
		},
		{
			name:        "Absolute path inside project",
			projectPath: tempDir,
			inputPath:   filepath.Join(tempDir, "foo/bar/baz.txt"),
			expectError: false,
		},
		{
			name:        "Relative path traversal escaping project",
			projectPath: tempDir,
			inputPath:   "../../outside.txt",
			expectError: true,
		},
		{
			name:        "Absolute path escaping project",
			projectPath: tempDir,
			inputPath:   "/usr/bin/some_binary",
			expectError: true,
		},
		{
			name:        "Empty project path",
			projectPath: "",
			inputPath:   "some/file.txt",
			expectError: true,
		},
		{
			name:        "Path traversing back and forth but staying inside",
			projectPath: tempDir,
			inputPath:   "foo/../foo/bar.txt",
			expectError: false,
		},
		{
			name:        "Clean traversal to root of project",
			projectPath: tempDir,
			inputPath:   ".",
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveAndSandboxPath(tc.projectPath, tc.inputPath)
			if (err != nil) != tc.expectError {
				t.Errorf("expected error: %v, got error: %v", tc.expectError, err)
			}
		})
	}
}

func TestGenerateSchemaMap(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_schema_map_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Verify succeeds with "no databases found" on empty folder
	emptyRes, err := GenerateSchemaMap(tempDir)
	if err != nil {
		t.Fatalf("GenerateSchemaMap failed on empty dir: %v", err)
	}
	if emptyRes == "" {
		t.Error("expected non-empty output summary on empty dir")
	}

	// 2. Setup mock schema dirs recursively
	dbPath := filepath.Join(tempDir, "scratch", "test_explode", "TestDatabase")
	tablesPath := filepath.Join(dbPath, "tables")
	layoutsPath := filepath.Join(dbPath, "layouts")
	scriptsPath := filepath.Join(dbPath, "scripts")
	toPath := filepath.Join(dbPath, "table_occurrences")
	relPath := filepath.Join(dbPath, "relationships")

	if err := os.MkdirAll(tablesPath, 0755); err != nil {
		t.Fatalf("failed to create mock tables dir: %v", err)
	}
	if err := os.MkdirAll(layoutsPath, 0755); err != nil {
		t.Fatalf("failed to create mock layouts dir: %v", err)
	}
	if err := os.MkdirAll(scriptsPath, 0755); err != nil {
		t.Fatalf("failed to create mock scripts dir: %v", err)
	}
	if err := os.MkdirAll(toPath, 0755); err != nil {
		t.Fatalf("failed to create mock table occurrences dir: %v", err)
	}
	if err := os.MkdirAll(relPath, 0755); err != nil {
		t.Fatalf("failed to create mock relationships dir: %v", err)
	}

	// 3. Write mock table XML
	mockTableXML := `<?xml version="1.0" encoding="UTF-8"?>
<BaseTable>
	<BaseTableReference id="10" name="Customers" />
	<FieldCatalog>
		<Field id="1" name="First Name" datatype="Text" fieldtype="Normal" />
		<Field id="2" name="Created At" datatype="Timestamp" fieldtype="Calculation" />
	</FieldCatalog>
</BaseTable>`
	if err := os.WriteFile(filepath.Join(tablesPath, "Customers.xml"), []byte(mockTableXML), 0644); err != nil {
		t.Fatalf("failed to write mock table XML: %v", err)
	}

	// 4. Write mock layout XML
	mockLayoutXML := `<?xml version="1.0" encoding="UTF-8"?>
<Layout id="5" name="Customer Detail" width="1024">
	<TableOccurrenceReference id="20" name="Customers_TO" />
	<ScriptTrigger id="101" action="OnLayoutEnter">
		<ScriptReference id="15" name="Log Access" />
	</ScriptTrigger>
</Layout>`
	if err := os.WriteFile(filepath.Join(layoutsPath, "CustomerDetail.xml"), []byte(mockLayoutXML), 0644); err != nil {
		t.Fatalf("failed to write mock layout XML: %v", err)
	}

	// 5. Write mock table occurrence XML
	mockTOXML := `<?xml version="1.0" encoding="UTF-8"?>
<TableOccurrence id="20" name="Customers_TO" type="Local">
	<BaseTableSourceReference type="BaseTableReference">
		<BaseTableReference id="10" name="Customers" />
	</BaseTableSourceReference>
</TableOccurrence>`
	if err := os.WriteFile(filepath.Join(toPath, "Customers_TO.xml"), []byte(mockTOXML), 0644); err != nil {
		t.Fatalf("failed to write mock TO XML: %v", err)
	}

	// 6. Write mock relationship XML
	mockRelXML := `<?xml version="1.0" encoding="UTF-8"?>
<Relationship id="1">
	<LeftTable>
		<TableOccurrenceReference id="20" name="Customers_TO" />
	</LeftTable>
	<RightTable>
		<TableOccurrenceReference id="30" name="Orders_TO" />
	</RightTable>
	<JoinPredicateList>
		<JoinPredicate type="Equal">
			<LeftField>
				<FieldReference id="1" name="CustomerID" />
			</LeftField>
			<RightField>
				<FieldReference id="2" name="CustomerID" />
			</RightField>
		</JoinPredicate>
	</JoinPredicateList>
</Relationship>`
	if err := os.WriteFile(filepath.Join(relPath, "Customers_Orders.xml"), []byte(mockRelXML), 0644); err != nil {
		t.Fatalf("failed to write mock relationship XML: %v", err)
	}

	// 7. Write mock script files
	if err := os.WriteFile(filepath.Join(scriptsPath, "Initialize Customers.txt"), []byte("Set Field..."), 0644); err != nil {
		t.Fatalf("failed to write mock script: %v", err)
	}

	// 8. Run GenerateSchemaMap
	res, err := GenerateSchemaMap(tempDir)
	if err != nil {
		t.Fatalf("GenerateSchemaMap failed: %v", err)
	}

	if res == "" {
		t.Error("expected non-empty output summary response")
	}

	// 9. Verify map file exists and contains schema items
	mapFile := filepath.Join(tempDir, "workspace_map.md")
	contentBytes, err := os.ReadFile(mapFile)
	if err != nil {
		t.Fatalf("failed to read created workspace_map.md: %v", err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "Database: TestDatabase") {
		t.Error("workspace map missing database details")
	}
	if !strings.Contains(content, "Customers") {
		t.Error("workspace map missing table name")
	}
	if !strings.Contains(content, "Customer Detail") {
		t.Error("workspace map missing layout name")
	}
	if !strings.Contains(content, "Initialize Customers") {
		t.Error("workspace map missing script details")
	}
	if !strings.Contains(content, "Customers_TO") {
		t.Error("workspace map missing Table Occurrence details")
	}
	if !strings.Contains(content, "Base Table: `Customers`") {
		t.Error("workspace map missing Base Table mapping in TO details")
	}
}

func TestResolveAndSandboxPathFallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_fallback_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create files/Schema/Contacts/scripts/test.txt
	targetDir := filepath.Join(tempDir, "files", "Schema", "Contacts", "scripts")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	targetFile := filepath.Join(targetDir, "test.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Resolve schema/Contacts/scripts/test.txt
	resolved, err := ResolveAndSandboxPath(tempDir, "schema/Contacts/scripts/test.txt")
	if err != nil {
		t.Fatalf("ResolveAndSandboxPath failed: %v", err)
	}

	expectedClean := filepath.Clean(targetFile)
	if filepath.Clean(resolved) != expectedClean {
		t.Errorf("expected resolved path %q, got %q", expectedClean, resolved)
	}
}

func TestWorkbenchLookupTools(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_workbench_lookup_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "files", "Schema", "CRM")
	tablesPath := filepath.Join(dbPath, "tables")
	layoutsPath := filepath.Join(dbPath, "layouts")
	relPath := filepath.Join(dbPath, "relationships")
	for _, dir := range []string{tablesPath, layoutsPath, relPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}

	tableXML := `<?xml version="1.0" encoding="UTF-8"?>
<BaseTable>
	<BaseTableReference id="10" name="Contacts" />
	<FieldCatalog>
		<Field id="1" name="ContactID" datatype="Text" fieldtype="Normal" />
		<Field id="2" name="Full Name" datatype="Text" fieldtype="Calculation" />
	</FieldCatalog>
</BaseTable>`
	if err := os.WriteFile(filepath.Join(tablesPath, "Contacts.xml"), []byte(tableXML), 0644); err != nil {
		t.Fatalf("failed to write table XML: %v", err)
	}

	layoutXML := `<?xml version="1.0" encoding="UTF-8"?>
<Layout id="5" name="Contact Detail">
	<TableOccurrenceReference id="20" name="Contacts_TO" />
	<ScriptTrigger id="101" action="OnLayoutEnter">
		<ScriptReference id="15" name="Contact_Load" />
	</ScriptTrigger>
</Layout>`
	if err := os.WriteFile(filepath.Join(layoutsPath, "ContactDetail.xml"), []byte(layoutXML), 0644); err != nil {
		t.Fatalf("failed to write layout XML: %v", err)
	}

	relationshipXML := `<?xml version="1.0" encoding="UTF-8"?>
<Relationship id="1">
	<LeftTable>
		<TableOccurrenceReference id="20" name="Contacts_TO" />
	</LeftTable>
	<RightTable>
		<TableOccurrenceReference id="30" name="Activities_TO" />
	</RightTable>
	<JoinPredicateList>
		<JoinPredicate type="Equal">
			<LeftField>
				<FieldReference id="1" name="ContactID" />
			</LeftField>
			<RightField>
				<FieldReference id="2" name="ContactID" />
			</RightField>
		</JoinPredicate>
	</JoinPredicateList>
</Relationship>`
	if err := os.WriteFile(filepath.Join(relPath, "Contacts_Activities.xml"), []byte(relationshipXML), 0644); err != nil {
		t.Fatalf("failed to write relationship XML: %v", err)
	}

	tableRes, err := FindTable(tempDir, "Contacts", "CRM")
	if err != nil {
		t.Fatalf("FindTable failed: %v", err)
	}
	if !strings.Contains(tableRes, `"table": "Contacts"`) || !strings.Contains(tableRes, "ContactID") {
		t.Errorf("FindTable response missing expected table details: %s", tableRes)
	}

	layoutRes, err := FindLayout(tempDir, "Contact Detail", "CRM")
	if err != nil {
		t.Fatalf("FindLayout failed: %v", err)
	}
	if !strings.Contains(layoutRes, `"layout": "Contact Detail"`) || !strings.Contains(layoutRes, "Contacts_TO") || !strings.Contains(layoutRes, "Contact_Load") {
		t.Errorf("FindLayout response missing expected layout details: %s", layoutRes)
	}

	relationshipRes, err := InspectRelationships(tempDir, "CRM", "Contacts_TO")
	if err != nil {
		t.Fatalf("InspectRelationships failed: %v", err)
	}
	if !strings.Contains(relationshipRes, `"count": 1`) || !strings.Contains(relationshipRes, "Activities_TO") {
		t.Errorf("InspectRelationships response missing expected relationship details: %s", relationshipRes)
	}
}

func TestValidateWebViewerHTML(t *testing.T) {
	validHTML := `<!doctype html><html><body><button onclick="FileMaker.PerformScript('Save', '{}')">Save</button><script>function init(){}</script></body></html>`
	validRes, err := ValidateWebViewerHTML(validHTML, false)
	if err != nil {
		t.Fatalf("ValidateWebViewerHTML failed: %v", err)
	}
	if !strings.Contains(validRes, `"passed": true`) || !strings.Contains(validRes, `"has_filemaker_bridge": true`) {
		t.Errorf("expected valid WebViewer HTML to pass: %s", validRes)
	}

	unsafeHTML := `<html><head><script src="https://example.com/app.js"></script></head><body><script>eval("alert(1)")</script></body></html>`
	unsafeRes, err := ValidateWebViewerHTML(unsafeHTML, false)
	if err != nil {
		t.Fatalf("ValidateWebViewerHTML failed on unsafe HTML: %v", err)
	}
	if !strings.Contains(unsafeRes, `"passed": false`) || !strings.Contains(unsafeRes, "remote or embedded external dependency") || !strings.Contains(unsafeRes, "eval(") {
		t.Errorf("expected unsafe WebViewer HTML to fail: %s", unsafeRes)
	}
}
