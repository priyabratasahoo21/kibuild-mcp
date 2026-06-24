package exploder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A minimal FMSaveAsXML ScriptCatalog with three scripts. "Beta" deliberately
// nests a <Script>/<ScriptReference> and an <ObjectList> inside a step's
// parameters — the exact shape that broke a name-only parser — to guard against
// the truncation/buffer-reset regression. "Empty" has no steps.
const sampleCatalog = `<?xml version="1.0" encoding="UTF-8"?>
<FMSaveAsXML version="2.3.0.0" Source="26.0.1" split_catalogs="True">
	<Structure>
		<AddAction>
			<ScriptCatalog membercount="3">
				<Script id="1" name="Alpha"><Options>0</Options></Script>
				<Script id="2" name="Beta"><Options>0</Options></Script>
				<Script id="3" name="Empty"><Options>0</Options></Script>
			</ScriptCatalog>
			<StepsForScripts membercount="3">
				<Script>
					<ScriptReference id="1" name="Alpha"></ScriptReference>
					<ObjectList membercount="2">
						<Step id="89" name="# (comment)" enable="True">
							<Text><![CDATA[hello]]></Text>
						</Step>
						<Step id="141" name="Set Variable" enable="True">
							<ParameterValues membercount="1">
								<Parameter type="Target">
									<Name><Calculation datatype="1"><Calculation><Text><![CDATA[$x]]></Text></Calculation></Calculation></Name>
								</Parameter>
							</ParameterValues>
						</Step>
					</ObjectList>
				</Script>
				<Script>
					<ScriptReference id="2" name="Beta"></ScriptReference>
					<ObjectList membercount="1">
						<Step id="1" name="Perform Script" enable="True">
							<ParameterValues membercount="1">
								<Parameter type="ScriptReference">
									<Script>
										<ScriptReference id="1" name="Alpha"></ScriptReference>
									</Script>
								</Parameter>
								<ObjectList membercount="0"></ObjectList>
							</ParameterValues>
						</Step>
					</ObjectList>
				</Script>
				<Script>
					<ScriptReference id="3" name="Empty"></ScriptReference>
					<ObjectList membercount="0"></ObjectList>
				</Script>
			</StepsForScripts>
		</AddAction>
	</Structure>
</FMSaveAsXML>`

// stubSanitize mimics SanitizeFMXmlSnippet's contract: it errors on a snippet
// with no steps, which the exploder treats as a valid empty script.
func stubSanitize(snippet string) (string, error) {
	if !strings.Contains(snippet, "<Step ") {
		return "", fmt.Errorf("no script steps found in fmxmlsnippet")
	}
	return "SANITIZED", nil
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func checkExploded(t *testing.T, res *Result) {
	t.Helper()
	if res.Scripts != 3 {
		t.Errorf("expected 3 scripts, got %d", res.Scripts)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", res.Warnings)
	}

	scripts := filepath.Join(res.Dest, "scripts")
	sanitized := filepath.Join(res.Dest, "scripts_sanitized")

	// Alpha: two steps present.
	alpha := readFile(t, filepath.Join(scripts, "Alpha.xml"))
	for _, want := range []string{`name="# (comment)"`, `name="Set Variable"`, "<fmxmlsnippet", "</fmxmlsnippet>"} {
		if !strings.Contains(alpha, want) {
			t.Errorf("Alpha.xml missing %q", want)
		}
	}

	// Beta: the nested <Script>/<ScriptReference> inside the Perform Script step
	// must be preserved (not truncated, not promoted to a separate script).
	beta := readFile(t, filepath.Join(scripts, "Beta.xml"))
	if !strings.Contains(beta, `name="Perform Script"`) {
		t.Errorf("Beta.xml missing the Perform Script step")
	}
	if !strings.Contains(beta, `name="Alpha"`) {
		t.Errorf("Beta.xml lost the nested ScriptReference to Alpha (truncation regression)")
	}
	if !strings.HasSuffix(strings.TrimSpace(beta), "</fmxmlsnippet>") {
		t.Errorf("Beta.xml is truncated — does not end with </fmxmlsnippet>")
	}

	// Empty: file exists, no steps, empty sanitized output (no warning).
	if _, err := os.Stat(filepath.Join(scripts, "Empty.xml")); err != nil {
		t.Errorf("Empty.xml not written: %v", err)
	}
	if got := readFile(t, filepath.Join(sanitized, "Empty.txt")); got != "" {
		t.Errorf("Empty.txt should be empty, got %q", got)
	}
	if got := readFile(t, filepath.Join(sanitized, "Alpha.txt")); got != "SANITIZED" {
		t.Errorf("Alpha.txt = %q, want SANITIZED", got)
	}
}

func TestExplodeSingleFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Contacts.xml")
	if err := os.WriteFile(src, []byte(sampleCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Explode(src, "Contacts", filepath.Join(dir, "out"), stubSanitize)
	if err != nil {
		t.Fatalf("Explode (single file): %v", err)
	}
	if res.Mode != "single-file" {
		t.Errorf("mode = %q, want single-file", res.Mode)
	}
	checkExploded(t, res)
}

func TestExplodeSplitCatalogs(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "Contacts")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Split mode routes by the *_ScriptCatalog.xml filename.
	if err := os.WriteFile(filepath.Join(srcDir, "Contacts_ScriptCatalog.xml"), []byte(sampleCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Explode(srcDir, "Contacts", filepath.Join(dir, "out"), stubSanitize)
	if err != nil {
		t.Fatalf("Explode (split catalogs): %v", err)
	}
	if res.Mode != "split-catalogs" {
		t.Errorf("mode = %q, want split-catalogs", res.Mode)
	}
	checkExploded(t, res)
}

func TestExplodeInfersDatabase(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "MyDB.xml")
	if err := os.WriteFile(src, []byte(sampleCatalog), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Explode(src, "", filepath.Join(dir, "out"), nil)
	if err != nil {
		t.Fatalf("Explode: %v", err)
	}
	if res.Database != "MyDB" {
		t.Errorf("inferred database = %q, want MyDB", res.Database)
	}
}
