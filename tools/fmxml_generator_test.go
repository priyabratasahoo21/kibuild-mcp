package tools

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileScript(t *testing.T) {
	// Use the project directory to find templates and catalogs.
	// We'll pass a relative path if tests are run from sidecar/tools
	projectPath := filepath.Join("..", "..") 

	// 1. Test standard step with parameters
	jsonData := []byte(`[
		{
			"stepName": "If",
			"parameters": {
				"Calculation": "Get ( LastError ) ≠ 0"
			}
		},
		{
			"stepName": "Set Field",
			"parameters": {
				"TargetTable": "Invoices",
				"TargetField": "Status",
				"Calculation": "\"Paid\""
			}
		}
	]`)

	xmlOutput, err := CompileScript(projectPath, jsonData)
	if err != nil {
		t.Fatalf("CompileScript failed: %v", err)
	}

	if !strings.Contains(xmlOutput, `name="If"`) {
		t.Errorf("Expected output to contain name=\"If\", got: %s", xmlOutput)
	}
	if !strings.Contains(xmlOutput, `<![CDATA[Get ( LastError ) ≠ 0]]>`) {
		t.Errorf("Expected output to contain calculation, got: %s", xmlOutput)
	}
	if !strings.Contains(xmlOutput, `name="Set Field"`) {
		t.Errorf("Expected output to contain name=\"Set Field\", got: %s", xmlOutput)
	}

	// 2. Test raw_xml fallback
	rawJsonData := []byte(`[
		{
			"stepName": "Insert from URL",
			"raw_xml": "<Step enable=\"True\" id=\"160\" name=\"Insert from URL\"></Step>"
		}
	]`)

	rawXmlOutput, err := CompileScript(projectPath, rawJsonData)
	if err != nil {
		t.Fatalf("CompileScript failed on raw_xml: %v", err)
	}

	if !strings.Contains(rawXmlOutput, `<Step enable="True" id="160" name="Insert from URL"></Step>`) {
		t.Errorf("Expected output to contain verbatim raw_xml, got: %s", rawXmlOutput)
	}

	// 3. Test validation error (missing required field)
	invalidJsonData := []byte(`[
		{
			"stepName": "Set Field",
			"parameters": {}
		}
	]`)

	_, err = CompileScript(projectPath, invalidJsonData)
	if err == nil {
		t.Error("CompileScript should have failed due to missing Calculation for Set Field step, but it succeeded")
	}

	// 4. Test type-safety (Calculation param contains non-string type like an array)
	uncomparableJsonData := []byte(`[
		{
			"stepName": "Set Field",
			"parameters": {
				"Calculation": [1, 2, 3]
			}
		}
	]`)

	_, err = CompileScript(projectPath, uncomparableJsonData)
	if err == nil {
		t.Error("CompileScript should have failed due to non-string Calculation parameter, but it succeeded without panic")
	} else if !strings.Contains(err.Error(), "requires Calculation") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestCompileSendMail(t *testing.T) {
	projectPath := filepath.Join("..", "..")

	jsonData := []byte(`[
		{
			"stepName": "Send Mail",
			"parameters": {
				"To": "$toEmail",
				"Subject": "\"Welcome\"",
				"Message": "$htmlBody",
				"FromEmail": "$$SMTP_FROM",
				"SMTPServer": "$$SMTP_HOST",
				"SMTPPort": "587",
				"SMTPUsername": "$$SMTP_USER",
				"SMTPPassword": "$$SMTP_PASS"
			}
		}
	]`)

	out, err := CompileScript(projectPath, jsonData)
	if err != nil {
		t.Fatalf("CompileScript failed for Send Mail: %v", err)
	}

	// Correct FileMaker step id (NOT the 64 the model hallucinated).
	if !strings.Contains(out, `id="63" name="Send Mail"`) {
		t.Errorf("Expected Send Mail with id=63, got: %s", out)
	}
	// Real clipboard element names, in CDATA.
	for _, want := range []string{
		`<To UseFoundSet="False"><Calculation><![CDATA[$toEmail]]>`,
		`<Subject><Calculation><![CDATA["Welcome"]]>`,
		`<Message><Calculation><![CDATA[$htmlBody]]>`,
		`<SMTPServer><Calculation><![CDATA[$$SMTP_HOST]]>`,
		`<SendViaSMTP state="True"/>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Expected output to contain %q, got: %s", want, out)
		}
	}
	// Fabricated elements must never appear, and absent optional fields must not
	// leak Go's "<no value>" placeholder.
	for _, bad := range []string{"SMTPOptions", "ToRecipients", "HTMLMessage", "<no value>"} {
		if strings.Contains(out, bad) {
			t.Errorf("Output should NOT contain %q, got: %s", bad, out)
		}
	}

	// The sanitizer should render To/Subject + dialog mode, not a bare "[ No dialog ]".
	readable, sErr := SanitizeFMXmlSnippet(out)
	if sErr != nil {
		t.Fatalf("SanitizeFMXmlSnippet failed: %v", sErr)
	}
	if !strings.Contains(readable, "Send Mail [ To: $toEmail ; Subject: \"Welcome\" ; No dialog ]") {
		t.Errorf("Unexpected sanitized Send Mail line: %s", readable)
	}
}
