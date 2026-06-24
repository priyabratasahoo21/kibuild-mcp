package tools

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// XMLStep represents a simplified FileMaker script step
type XMLStep struct {
	ID         string            `xml:"id,attr" json:"id"`
	Name       string            `xml:"name,attr" json:"stepName"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// XMLScript represents a simplified FileMaker script definition
type XMLScript struct {
	ID   string    `xml:"id,attr" json:"id"`
	Name string    `xml:"name,attr" json:"name"`
	Step []XMLStep `xml:"Step" json:"step"`
}

type XMLStepsList struct {
	XMLName xml.Name  `xml:"StepsList"`
	Steps   []XMLStep `xml:"Step"`
}

// ExtractScriptSteps parses XML content and returns script steps as JSON
func ExtractScriptSteps(xmlContent string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var steps []XMLStep

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "Step" {
				var step XMLStep
				step.Parameters = make(map[string]string)
				for _, attr := range se.Attr {
					if attr.Name.Local == "id" {
						step.ID = attr.Value
					} else if attr.Name.Local == "name" {
						step.Name = attr.Value
					}
				}
				
				// Parse inner tokens until </Step>
				for {
					t, err := decoder.Token()
					if err != nil {
						break
					}
					switch inner := t.(type) {
					case xml.StartElement:
						if inner.Name.Local == "Calculation" {
							var calcData string
							_ = decoder.DecodeElement(&calcData, &inner)
							step.Parameters["Calculation"] = strings.TrimSpace(calcData)
						} else if inner.Name.Local == "Text" {
							var textData string
							_ = decoder.DecodeElement(&textData, &inner)
							if step.Name == "# (comment)" || step.Name == "Comment" {
								step.Parameters["Calculation"] = strings.TrimSpace(textData)
							} else {
								step.Parameters["Text"] = strings.TrimSpace(textData)
							}
						} else if inner.Name.Local == "Field" {
							var table, field string
							for _, a := range inner.Attr {
								if a.Name.Local == "table" {
									table = a.Value
								} else if a.Name.Local == "name" {
									field = a.Value
								}
							}
							if table != "" {
								step.Parameters["TargetTable"] = table
							}
							if field != "" {
								step.Parameters["TargetField"] = field
							}
						}
					case xml.EndElement:
						if inner.Name.Local == "Step" {
							goto StepDone
						}
					}
				}
			StepDone:
				if len(step.Parameters) == 0 {
					step.Parameters = nil
				}
				if step.ID != "" || step.Name != "" {
					steps = append(steps, step)
				}
			}
		}
	}

	bytesVal, err := json.MarshalIndent(steps, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytesVal), nil
}

// LookupScriptName looks up a script name by ID inside the XML content
func LookupScriptName(xmlContent string, scriptID string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "Script" {
				var id, name string
				for _, attr := range se.Attr {
					if attr.Name.Local == "id" {
						id = attr.Value
					} else if attr.Name.Local == "name" {
						name = attr.Value
					}
				}
				if id == scriptID && name != "" {
					return name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("script ID %s not found in XML content", scriptID)
}

// DependencyInfo holds relationship and script references found in layout or script XML
type DependencyInfo struct {
	Scripts   []string `json:"scripts"`
	Layouts   []string `json:"layouts"`
	Tables    []string `json:"tables"`
	Fields    []string `json:"fields"`
}

// TraceDependencies analyzes XML to find table occurrences, script calls, layout links, etc.
func TraceDependencies(xmlContent string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	var info DependencyInfo

	scriptMap := make(map[string]bool)
	layoutMap := make(map[string]bool)
	tableMap := make(map[string]bool)
	fieldMap := make(map[string]bool)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "ScriptReference" || se.Name.Local == "Script" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" && !scriptMap[attr.Value] {
						scriptMap[attr.Value] = true
						info.Scripts = append(info.Scripts, attr.Value)
					}
				}
			} else if se.Name.Local == "LayoutReference" || se.Name.Local == "Layout" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" && !layoutMap[attr.Value] {
						layoutMap[attr.Value] = true
						info.Layouts = append(info.Layouts, attr.Value)
					}
				}
			} else if se.Name.Local == "BaseTableReference" || se.Name.Local == "TableOccurrenceReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" && !tableMap[attr.Value] {
						tableMap[attr.Value] = true
						info.Tables = append(info.Tables, attr.Value)
					}
				}
			} else if se.Name.Local == "FieldReference" || se.Name.Local == "Field" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" && !fieldMap[attr.Value] {
						fieldMap[attr.Value] = true
						info.Fields = append(info.Fields, attr.Value)
					}
				}
			}
		}
	}

	bytesVal, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytesVal), nil
}

// MatchRevision extracts and returns the revision or version elements from FileMaker XML header
func MatchRevision(xmlContent string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "FMPReport" || se.Name.Local == "FMPSnippet" || se.Name.Local == "fmxmlsnippet" {
				var version, revision string
				for _, attr := range se.Attr {
					if attr.Name.Local == "version" {
						version = attr.Value
					} else if attr.Name.Local == "revision" {
						revision = attr.Value
					}
				}
				if version != "" || revision != "" {
					return fmt.Sprintf("Version: %s | Revision: %s", version, revision), nil
				}
			}
		}
	}

	return "No version or revision attribute found in root XML node.", nil
}
