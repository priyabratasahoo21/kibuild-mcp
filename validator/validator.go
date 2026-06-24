package validator

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ValidationError struct {
	Rule    string `json:"rule"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type ValidationResult struct {
	Passed   bool              `json:"passed"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// Index schemas to load for context integrity checks
type indexScript struct {
	Name string `json:"name"`
}
type indexLayout struct {
	Name string `json:"name"`
}
type indexTable struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

// ValidateXML validates a FileMaker XML snippet against the 7 structural rules
func ValidateXML(xmlContent string, projectPath, dbName string) (ValidationResult, error) {
	result := ValidationResult{Passed: true}

	trimmed := strings.TrimSpace(xmlContent)
	if trimmed == "" {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			Rule:    "root_wrapper",
			Line:    1,
			Message: "XML snippet is empty",
		})
		return result, nil
	}

	// Helper to add error
	addError := func(rule string, line int, msg string) {
		result.Passed = false
		result.Errors = append(result.Errors, ValidationError{
			Rule:    rule,
			Line:    line,
			Message: msg,
		})
	}

	// Helper to add warning
	addWarning := func(rule string, line int, msg string) {
		result.Warnings = append(result.Warnings, ValidationError{
			Rule:    rule,
			Line:    line,
			Message: msg,
		})
	}

	// 1. Root Wrapper Check (Line-by-line / content scan)
	hasXmlDecl := strings.HasPrefix(trimmed, "<?xml")
	rootCheckStr := trimmed
	if hasXmlDecl {
		// strip xml declaration for simple checks
		if idx := strings.Index(trimmed, "?>"); idx != -1 {
			rootCheckStr = strings.TrimSpace(trimmed[idx+2:])
		}
	}

	if !strings.HasPrefix(rootCheckStr, "<fmxmlsnippet") {
		addError("root_wrapper", 1, "Snippet must start with <fmxmlsnippet> root wrapper")
	}
	if !strings.HasSuffix(rootCheckStr, "</fmxmlsnippet>") {
		// Find last line
		lines := strings.Split(xmlContent, "\n")
		addError("root_wrapper", len(lines), "Snippet must end with </fmxmlsnippet> root wrapper")
	}

	// Split by line to validate line-specific rules (Rules 2, 3, 4, 5, 6)
	lines := strings.Split(xmlContent, "\n")
	for i, line := range lines {
		lineNum := i + 1

		// Rule 2: No Namespace Pollution
		if strings.Contains(line, "ns0:") || strings.Contains(line, "xmlns:ns0") {
			addError("no_namespace", lineNum, "Namespace prefix 'ns0:' is not allowed in FileMaker XML")
		}

		// Rule 3: No Dynamic Identifiers
		if strings.Contains(line, "uuid=") || strings.Contains(line, "hash=") || strings.Contains(line, "<uuid>") || strings.Contains(line, "<uuid ") {
			addError("no_dynamic_identifiers", lineNum, "Dynamic attributes 'uuid' and 'hash' must not be generated")
		}

		// Rule 4: CDATA Wrap for Calculations & Text
		if strings.Contains(line, "<Calculation") {
			// Find content between <Calculation...> and </Calculation>
			calcStart := strings.Index(line, "<Calculation")
			calcEnd := strings.Index(line, "</Calculation>")
			if calcStart != -1 && calcEnd != -1 && calcStart < calcEnd {
				innerStart := strings.Index(line[calcStart:calcEnd], ">")
				if innerStart != -1 {
					content := line[calcStart+innerStart+1 : calcEnd]
					if len(strings.TrimSpace(content)) > 0 && !strings.Contains(content, "<![CDATA[") {
						addError("cdata_wrapper", lineNum, "Calculation formula must be wrapped in <![CDATA[...]]>")
					}
				}
			}
		}

		if strings.Contains(line, "<Text") {
			textStart := strings.Index(line, "<Text")
			textEnd := strings.Index(line, "</Text>")
			if textStart != -1 && textEnd != -1 && textStart < textEnd {
				innerStart := strings.Index(line[textStart:textEnd], ">")
				if innerStart != -1 {
					content := line[textStart+innerStart+1 : textEnd]
					// Only require CDATA if contains special characters
					if len(strings.TrimSpace(content)) > 0 && (strings.ContainsAny(content, "<>&\"'") || strings.Contains(content, "≠")) {
						if !strings.Contains(content, "<![CDATA[") {
							addWarning("cdata_wrapper", lineNum, "Text containing special characters should be wrapped in <![CDATA[...]]>")
						}
					}
				}
			}
		}

		// Rule 5: Explicit Step Enablement
		if strings.Contains(line, "<Step") {
			if !strings.Contains(line, `enable="True"`) {
				addError("step_enablement", lineNum, "Every <Step> element must have enable=\"True\" attribute")
			}
		}

		// Rule 6: FileMaker Comment Step format
		if strings.Contains(line, "<!--") && !strings.Contains(line, "<?xml") && !strings.Contains(line, "doctype") {
			addWarning("xml_comments", lineNum, "Do not use XML comments for code comments. Use Step ID 89 Comment steps")
		}

		// Rule 1 extension: Check if the XML is wrapped in Script instead of fmxmlsnippet
		if (strings.HasPrefix(rootCheckStr, "<Script>") || strings.HasPrefix(rootCheckStr, "<Script ")) && lineNum == 1 {
			addError("root_wrapper", lineNum, "Do not wrap step snippets in a <Script> element")
		}
	}

	// Initialize index maps for Context Integrity Checks
	scriptsMap := make(map[string]bool)
	layoutsMap := make(map[string]bool)
	tablesMap := make(map[string]bool)
	fieldsMap := make(map[string]bool)

	// Rule 7: Context Integrity Checks
	if projectPath != "" && dbName != "" {
		schemaDir := resolveSchemaDir(projectPath, dbName)
		indexDir := filepath.Join(schemaDir, ".kibuild_index")
		
		// Load scripts index
		if data, err := os.ReadFile(filepath.Join(indexDir, "script_index.json")); err == nil {
			var list []indexScript
			if json.Unmarshal(data, &list) == nil {
				for _, s := range list {
					scriptsMap[strings.ToLower(s.Name)] = true
				}
			}
		}

		// Load layouts index
		if data, err := os.ReadFile(filepath.Join(indexDir, "layout_index.json")); err == nil {
			var list []indexLayout
			if json.Unmarshal(data, &list) == nil {
				for _, l := range list {
					layoutsMap[strings.ToLower(l.Name)] = true
				}
			}
		}

		// Load tables index
		if data, err := os.ReadFile(filepath.Join(indexDir, "table_index.json")); err == nil {
			var list []indexTable
			if json.Unmarshal(data, &list) == nil {
				for _, t := range list {
					tablesMap[strings.ToLower(t.Name)] = true
					for _, f := range t.Fields {
						fName := f
						if idx := strings.Index(f, " ["); idx != -1 {
							fName = f[:idx]
						}
						fieldsMap[strings.ToLower(fName)] = true
					}
				}
			}
		}
	}

	// Parse XML structure unconditionally to validate syntax and context integrity
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			addError("xml_syntax", lineNumFromOffset(xmlContent, decoder.InputOffset()), fmt.Sprintf("XML parsing error: %v", err))
			break
		}

		// Only validate references if indices actually exist/loaded
		if len(scriptsMap) > 0 || len(layoutsMap) > 0 || len(tablesMap) > 0 {
			switch se := token.(type) {
			case xml.StartElement:
				lineOffset := lineNumFromOffset(xmlContent, decoder.InputOffset())

				// Validate Script Reference
				if se.Name.Local == "Script" || se.Name.Local == "ScriptReference" {
					for _, attr := range se.Attr {
						if attr.Name.Local == "name" && attr.Value != "" {
							name := attr.Value
							if len(scriptsMap) > 0 && !scriptsMap[strings.ToLower(name)] {
								addWarning("context_integrity", lineOffset, fmt.Sprintf("Referenced script %q does not exist in schema context", name))
							}
						}
					}
				}

				// Validate Layout Reference
				if se.Name.Local == "Layout" || se.Name.Local == "LayoutReference" {
					for _, attr := range se.Attr {
						if attr.Name.Local == "name" && attr.Value != "" {
							name := attr.Value
							if len(layoutsMap) > 0 && !layoutsMap[strings.ToLower(name)] {
								addWarning("context_integrity", lineOffset, fmt.Sprintf("Referenced layout %q does not exist in schema context", name))
							}
						}
					}
				}

				// Validate Table Reference
				if se.Name.Local == "Table" || se.Name.Local == "BaseTable" || se.Name.Local == "BaseTableReference" {
					for _, attr := range se.Attr {
						if attr.Name.Local == "name" && attr.Value != "" {
							name := attr.Value
							if len(tablesMap) > 0 && !tablesMap[strings.ToLower(name)] {
								addWarning("context_integrity", lineOffset, fmt.Sprintf("Referenced table %q does not exist in schema context", name))
							}
						}
					}
				}

				// Validate Field Reference
				if se.Name.Local == "Field" || se.Name.Local == "FieldReference" {
					for _, attr := range se.Attr {
						if attr.Name.Local == "name" && attr.Value != "" {
							name := attr.Value
							if len(fieldsMap) > 0 && !fieldsMap[strings.ToLower(name)] {
								addWarning("context_integrity", lineOffset, fmt.Sprintf("Referenced field %q does not exist in schema context", name))
							}
						}
					}
				}
			}
		}
	}

	return result, nil
}

func lineNumFromOffset(content string, offset int64) int {
	if offset < 0 || offset > int64(len(content)) {
		offset = int64(len(content))
	}
	return strings.Count(content[:offset], "\n") + 1
}

func resolveSchemaDir(projectPath, dbName string) string {
	if projectPath != "" {
		filesPath := filepath.Join(projectPath, "files", "Schema", dbName)
		if _, err := os.Stat(filesPath); err == nil {
			return filesPath
		}
		flatPath := filepath.Join(projectPath, "Schema", dbName)
		if _, err := os.Stat(flatPath); err == nil {
			return flatPath
		}
		return filesPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	filesPath := filepath.Join(home, "Library", "Application Support", "KiBuildPlugin", "projects", dbName, "files", "Schema", dbName)
	if _, err := os.Stat(filesPath); err == nil {
		return filesPath
	}
	flatPath := filepath.Join(home, "Library", "Application Support", "KiBuildPlugin", "projects", dbName, "Schema", dbName)
	if _, err := os.Stat(flatPath); err == nil {
		return flatPath
	}
	return filesPath
}
