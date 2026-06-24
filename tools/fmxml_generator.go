package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
)

type StepJSON struct {
	StepName   string                 `json:"stepName"`
	Parameters map[string]interface{} `json:"parameters"`
	RawXML     string                 `json:"raw_xml"`
}

type StepParam struct {
	XmlElement string `json:"xmlElement"`
	Required   bool   `json:"required"`
	HrLabel    string `json:"hrLabel"`
}

type CatalogStep struct {
	ID     int         `json:"id"`
	Name   string      `json:"name"`
	Params []StepParam `json:"params"`
}

var templateSubdirs = []string{
	"control_flow",
	"variables_data",
	"records",
	"navigation_ui",
	"scripts_apis",
	"transactions_misc",
	"miscellaneous",
}

var (
	stepCatalog     map[string]CatalogStep
	stepCatalogNorm map[string]CatalogStep // lowercase-keyed for case-insensitive lookup
	catalogOnce     sync.Once
	catalogLoadErr  error
)

func loadCatalog(projectPath string) error {
	catalogOnce.Do(func() {
		var catalogPath string
		var basePaths []string
		if projectPath != "" {
			basePaths = append(basePaths, filepath.Join(projectPath, "sidecar", "tools", "catalogs"))
		}
		if cwd, err := os.Getwd(); err == nil {
			basePaths = append(basePaths, filepath.Join(cwd, "sidecar", "tools", "catalogs"))
		}
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			basePaths = append(basePaths, filepath.Join(execDir, "tools", "catalogs"))
			basePaths = append(basePaths, filepath.Join(execDir, "sidecar", "tools", "catalogs"))
			parent := execDir
			for i := 0; i < 5; i++ {
				basePaths = append(basePaths, filepath.Join(parent, "sidecar", "tools", "catalogs"))
				basePaths = append(basePaths, filepath.Join(parent, "tools", "catalogs"))
				pDir := filepath.Dir(parent)
				if pDir == parent {
					break
				}
				parent = pDir
			}
		}

		for _, base := range basePaths {
			p := filepath.Join(base, "step-catalog-en.json")
			if _, err := os.Stat(p); err == nil {
				catalogPath = p
				break
			}
		}

		if catalogPath == "" {
			catalogLoadErr = fmt.Errorf("step-catalog-en.json not found in sidecar/tools/catalogs")
			// Reset so the next call with a valid projectPath can retry.
			catalogOnce = sync.Once{}
			return
		}

		data, err := os.ReadFile(catalogPath)
		if err != nil {
			catalogLoadErr = err
			catalogOnce = sync.Once{}
			return
		}

		var catalog []CatalogStep
		if err := json.Unmarshal(data, &catalog); err != nil {
			catalogLoadErr = err
			catalogOnce = sync.Once{}
			return
		}

		stepCatalog = make(map[string]CatalogStep, len(catalog))
		stepCatalogNorm = make(map[string]CatalogStep, len(catalog))
		for _, step := range catalog {
			stepCatalog[step.Name] = step
			stepCatalogNorm[strings.ToLower(step.Name)] = step
		}
		// Clear any previous error on success.
		catalogLoadErr = nil
	})
	return catalogLoadErr
}

// lookupCatalogStep finds a step by exact name first, then falls back to
// case-insensitive match so "commit records" resolves to "Commit Records/Requests".
func lookupCatalogStep(name string) (CatalogStep, bool) {
	if s, ok := stepCatalog[name]; ok {
		return s, true
	}
	s, ok := stepCatalogNorm[strings.ToLower(name)]
	return s, ok
}

// escapeCDATA prevents a calculation string from breaking a CDATA section by
// escaping any ]]> sequence (CDATA close marker) inside the value.
func escapeCDATA(s string) string {
	return strings.ReplaceAll(s, "]]>", "]]]]><![CDATA[>")
}

var rawXMLStepNameRe = regexp.MustCompile(`(?i)<Step[^>]+name="([^"]*)"`)

// extractStepNameFromRawXML returns the name="..." attribute from a raw <Step> element, or "".
func extractStepNameFromRawXML(rawXML string) string {
	if m := rawXMLStepNameRe.FindStringSubmatch(rawXML); len(m) > 1 {
		return m[1]
	}
	return ""
}

// simpleOnlySteps is the set of steps whose template covers ALL valid forms.
// raw_xml is never justified for these — the LLM must use the structured path.
var simpleOnlySteps = func() map[string]struct{} {
	names := []string{
		"# (comment)",
		"If", "Else If", "Else", "End If",
		"Loop", "Exit Loop If", "End Loop",
		"Set Variable", "Set Field", "Set Field By Name",
		"Commit Records/Requests", "Revert Record/Request",
		"New Record/Request", "Delete Record/Request", "Duplicate Record/Request",
		"Show All Records", "Sort Records", "Unsort Records",
		"Go to Layout", "Go to Field", "Go to Object",
		"Go to Record/Request/Page", "Go to Portal Row",
		"Perform Script", "Perform Script on Server",
		"Exit Script", "Halt Script",
		"Show Custom Dialog",
		"Set Error Capture", "Allow User Abort",
		"Open Transaction", "Commit Transaction", "Revert Transaction",
		"Refresh Window",
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[strings.ToLower(n)] = struct{}{}
	}
	return m
}()

func getTemplatePath(projectPath string, stepName string) string {
	safeName := strings.ReplaceAll(stepName, "/", "-")

	var basePaths []string
	if projectPath != "" {
		basePaths = append(basePaths, filepath.Join(projectPath, "sidecar", "templates", "fmxml"))
	}
	if cwd, err := os.Getwd(); err == nil {
		basePaths = append(basePaths, filepath.Join(cwd, "sidecar", "templates", "fmxml"))
	}
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		basePaths = append(basePaths, filepath.Join(execDir, "templates", "fmxml"))
		basePaths = append(basePaths, filepath.Join(execDir, "sidecar", "templates", "fmxml"))
		parent := execDir
		for i := 0; i < 5; i++ {
			basePaths = append(basePaths, filepath.Join(parent, "sidecar", "templates", "fmxml"))
			basePaths = append(basePaths, filepath.Join(parent, "templates", "fmxml"))
			pDir := filepath.Dir(parent)
			if pDir == parent {
				break
			}
			parent = pDir
		}
	}

	for _, base := range basePaths {
		for _, subdir := range templateSubdirs {
			path := filepath.Join(base, subdir, safeName+".xml")
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// CompileScript converts JSON script steps to FileMaker XML snippet
func CompileScript(projectPath string, aiJsonData []byte) (string, error) {
	if err := loadCatalog(projectPath); err != nil {
		return "", err
	}

	var steps []StepJSON
	if err := json.Unmarshal(aiJsonData, &steps); err != nil {
		return "", fmt.Errorf("failed to parse AI JSON: %v", err)
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<fmxmlsnippet type=\"FMObjectList\">\n")

	for i, step := range steps {
		if step.RawXML != "" {
			// Guard: reject raw_xml for steps whose template covers all valid forms.
			// Steps like Insert from URL (cURL variant) are intentionally excluded
			// from this list because their complex form genuinely needs raw_xml.
			if name := extractStepNameFromRawXML(step.RawXML); name != "" {
				if _, blocked := simpleOnlySteps[strings.ToLower(name)]; blocked {
					return "", fmt.Errorf(
						"step %d uses raw_xml for %q — this step has a template, use {\"stepName\": %q, \"parameters\": {...}} instead",
						i+1, name, name,
					)
				}
			}
			sb.WriteString(step.RawXML)
			sb.WriteString("\n")
			continue
		}

		if step.StepName == "" {
			return "", fmt.Errorf("step %d missing stepName and raw_xml", i+1)
		}

		// Catalog validation — case-insensitive lookup so minor name variations don't abort the whole script.
		catalogStep, exists := lookupCatalogStep(step.StepName)
		if exists {
			for _, param := range catalogStep.Params {
				if param.Required {
					if step.Parameters == nil {
						return "", fmt.Errorf("step '%s' requires parameters but none provided", step.StepName)
					}
					if param.XmlElement == "Calculation" {
						val, ok := step.Parameters["Calculation"]
						strVal, isStr := val.(string)
						if !ok || !isStr || strVal == "" {
							return "", fmt.Errorf("step '%s' requires Calculation parameter (got non-string or empty value)", step.StepName)
						}
					}
				}
			}
		}

		// Template lookup — use canonical name from catalog when available so that
		// "commit records" resolves to the correct "Commit Records-Requests.xml" file.
		canonicalName := step.StepName
		if exists {
			canonicalName = catalogStep.Name
		}
		tmplPath := getTemplatePath(projectPath, canonicalName)
		if tmplPath == "" {
			return "", fmt.Errorf("step '%s' has no template — use raw_xml for custom steps", step.StepName)
		}

		tmplContent, err := os.ReadFile(tmplPath)
		if err != nil {
			return "", fmt.Errorf("failed to read template for '%s': %v", step.StepName, err)
		}

		t, err := template.New(step.StepName).Parse(string(tmplContent))
		if err != nil {
			return "", fmt.Errorf("failed to parse template for '%s': %v", step.StepName, err)
		}

		// Escape ]]> in all string parameters to prevent CDATA boundary injection.
		escapedParams := make(map[string]interface{}, len(step.Parameters))
		for k, v := range step.Parameters {
			if str, ok := v.(string); ok {
				escapedParams[k] = escapeCDATA(str)
			} else {
				escapedParams[k] = v
			}
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, escapedParams); err != nil {
			return "", fmt.Errorf("failed to execute template for '%s' (params: %v): %v", step.StepName, step.Parameters, err)
		}

		sb.WriteString(buf.String())
		sb.WriteString("\n")
	}

	sb.WriteString("</fmxmlsnippet>")

	return sb.String(), nil
}
