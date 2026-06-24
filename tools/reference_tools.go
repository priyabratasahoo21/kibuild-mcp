package tools

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReferenceMatch struct {
	Database string `json:"database"`
	Path     string `json:"path"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Snippet  string `json:"snippet"`
}

type ToolReferencesResponse struct {
	Kind           string           `json:"kind"`
	Queries        []string         `json:"queries,omitempty"`
	Query          string           `json:"query"`
	DatabaseFilter string           `json:"database_filter,omitempty"`
	Matches        []ReferenceMatch `json:"matches"`
	Count          int              `json:"count"`
	TotalFound     int              `json:"total_found"`
	Limit          int              `json:"limit"`
}

func serializeReferenceResponse(kind string, queries []string, database string, matches []ReferenceMatch) string {
	limit := 50
	total := len(matches)
	var finalMatches []ReferenceMatch
	if total > limit {
		finalMatches = matches[:limit]
	} else {
		finalMatches = matches
	}

	queryStr := strings.Join(queries, ", ")

	response := ToolReferencesResponse{
		Kind:           kind,
		Queries:        queries,
		Query:          queryStr,
		DatabaseFilter: database,
		Matches:        finalMatches,
		Count:          len(finalMatches),
		TotalFound:     total,
		Limit:          limit,
	}

	bytesVal, _ := json.MarshalIndent(response, "", "  ")
	return string(bytesVal)
}

func matchAnyContains(name string, queries []string) bool {
	if len(queries) == 0 {
		return true
	}
	nameLower := strings.ToLower(name)
	hasNonEmpty := false
	for _, q := range queries {
		qLower := strings.ToLower(strings.TrimSpace(q))
		if qLower != "" {
			hasNonEmpty = true
			if strings.Contains(nameLower, qLower) {
				return true
			}
		}
	}
	return !hasNonEmpty
}

// 1. Layout-Centric Tools

func FindLayoutReferencesToScripts(projectPath string, layoutNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			layName, _, scripts, _, err := parseLayoutXML(xmlPath)
			if err != nil {
				continue
			}
			if !matchAnyContains(layName, layoutNames) {
				continue
			}

			for _, script := range scripts {
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     layName,
					Type:     "Layout Script Reference",
					Snippet:  fmt.Sprintf("Triggers/Buttons script: %s", script),
				})
			}
		}
	}

	return serializeReferenceResponse("find_layout_references_to_scripts", layoutNames, database, matches), nil
}

func FindLayoutReferencesToValueLists(projectPath string, layoutNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			layName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			// Read layout file to extract layout name from contents
			if data, err := os.ReadFile(xmlPath); err == nil {
				var temp struct {
					Layout struct {
						Name string `xml:"name,attr"`
					} `xml:"Layout"`
				}
				if xml.Unmarshal(data, &temp) == nil && temp.Layout.Name != "" {
					layName = temp.Layout.Name
				}
			}

			if !matchAnyContains(layName, layoutNames) {
				continue
			}

			valueLists, err := parseLayoutValueLists(xmlPath)
			if err != nil {
				continue
			}

			for _, vl := range valueLists {
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     layName,
					Type:     "Layout Value List Reference",
					Snippet:  fmt.Sprintf("Layout references Value List: %s", vl),
				})
			}
		}
	}

	return serializeReferenceResponse("find_layout_references_to_valuelists", layoutNames, database, matches), nil
}

func parseLayoutValueLists(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var valueLists []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "ValueListReference" || se.Name.Local == "ValueList" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							valueLists = append(valueLists, name)
						}
					}
				}
			}
		}
	}
	return valueLists, nil
}

func FindLayoutReferencesToTables(projectPath string, layoutNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			layName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			if data, err := os.ReadFile(xmlPath); err == nil {
				var temp struct {
					Layout struct {
						Name string `xml:"name,attr"`
					} `xml:"Layout"`
				}
				if xml.Unmarshal(data, &temp) == nil && temp.Layout.Name != "" {
					layName = temp.Layout.Name
				}
			}

			if !matchAnyContains(layName, layoutNames) {
				continue
			}

			tables, err := parseLayoutTables(xmlPath)
			if err != nil {
				continue
			}

			for _, t := range tables {
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     layName,
					Type:     "Layout TO Reference",
					Snippet:  fmt.Sprintf("Layout references Table Occurrence: %s", t),
				})
			}
		}
	}

	return serializeReferenceResponse("find_layout_references_to_tables", layoutNames, database, matches), nil
}

func parseLayoutTables(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tables []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "TableOccurrenceReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							tables = append(tables, name)
						}
					}
				}
			}
			if se.Name.Local == "FieldReference" || se.Name.Local == "Field" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						val := attr.Value
						if idx := strings.Index(val, "::"); idx != -1 {
							to := val[:idx]
							if to != "" && !seen[to] {
								seen[to] = true
								tables = append(tables, to)
							}
						}
					}
				}
			}
		}
	}
	return tables, nil
}

// 2. Script-Centric Tools

func FindScriptReferencesInScripts(projectPath string, scriptNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		sanitizedDir := filepath.Join(dbDir, "scripts_sanitized")
		if _, err := os.Stat(sanitizedDir); os.IsNotExist(err) {
			continue
		}

		var files []string
		filepath.Walk(sanitizedDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
				files = append(files, path)
			}
			return nil
		})

		for _, txtPath := range files {
			data, err := os.ReadFile(txtPath)
			if err != nil {
				continue
			}

			lines := strings.Split(string(data), "\n")
			scriptBaseName := strings.TrimSuffix(filepath.Base(txtPath), ".txt")

			for lineNum, line := range lines {
				lineLower := strings.ToLower(line)
				if strings.Contains(lineLower, "perform script") {
					matched := false
					for _, q := range scriptNames {
						qLower := strings.ToLower(strings.TrimSpace(q))
						if qLower != "" && strings.Contains(lineLower, qLower) {
							matched = true
							break
						}
					}
					if matched {
						xmlPath := filepath.Join(dbDir, "scripts", scriptBaseName+".xml")
						matches = append(matches, ReferenceMatch{
							Database: filepath.Base(dbDir),
							Path:     xmlPath,
							Name:     scriptBaseName,
							Type:     "Script Perform Step",
							Snippet:  fmt.Sprintf("Line %d: %s", lineNum+1, strings.TrimSpace(line)),
						})
					}
				}
			}
		}
	}

	return serializeReferenceResponse("find_script_references_in_scripts", scriptNames, database, matches), nil
}

func FindScriptReferencesInLayouts(projectPath string, scriptNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			layName, _, scripts, _, err := parseLayoutXML(xmlPath)
			if err != nil {
				continue
			}

			for _, script := range scripts {
				scriptLower := strings.ToLower(script)
				matched := false
				for _, q := range scriptNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && (scriptLower == qLower || strings.Contains(scriptLower, qLower)) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     layName,
						Type:     "Layout Script Trigger/Button",
						Snippet:  fmt.Sprintf("Invokes script: %s", script),
					})
					break
				}
			}
		}
	}

	return serializeReferenceResponse("find_script_references_in_layouts", scriptNames, database, matches), nil
}

func FindScriptReferencesToLayouts(projectPath string, scriptNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		scriptsDir := filepath.Join(dbDir, "scripts")
		for _, xmlPath := range findXmlFiles(scriptsDir) {
			sName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			if !matchAnyContains(sName, scriptNames) {
				continue
			}

			layouts, err := parseScriptLayoutReferences(xmlPath)
			if err != nil {
				continue
			}

			for _, lay := range layouts {
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     sName,
					Type:     "Script Layout Reference",
					Snippet:  fmt.Sprintf("Go to Layout: %s", lay),
				})
			}
		}
	}

	return serializeReferenceResponse("find_script_references_to_layouts", scriptNames, database, matches), nil
}

func parseScriptLayoutReferences(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var layouts []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "LayoutReference" || se.Name.Local == "Layout" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							layouts = append(layouts, name)
						}
					}
				}
			}
		}
	}
	return layouts, nil
}

func FindScriptReferencesToValueLists(projectPath string, scriptNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		scriptsDir := filepath.Join(dbDir, "scripts")
		for _, xmlPath := range findXmlFiles(scriptsDir) {
			sName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			if !matchAnyContains(sName, scriptNames) {
				continue
			}

			valLists, err := parseScriptValueListReferences(xmlPath)
			if err != nil {
				continue
			}

			for _, vl := range valLists {
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     sName,
					Type:     "Script Value List Reference",
					Snippet:  fmt.Sprintf("References value list: %s", vl),
				})
			}
		}
	}

	return serializeReferenceResponse("find_script_references_to_valuelists", scriptNames, database, matches), nil
}

func parseScriptValueListReferences(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var valueLists []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "ValueListReference" || se.Name.Local == "ValueList" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							valueLists = append(valueLists, name)
						}
					}
				}
			}
		}
	}
	return valueLists, nil
}

// 3. Field-Centric Tools

func FindFieldReferencesInScripts(projectPath string, fieldNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		scriptsDir := filepath.Join(dbDir, "scripts")
		for _, xmlPath := range findXmlFiles(scriptsDir) {
			sName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			fields, err := parseScriptFieldReferences(xmlPath)
			if err != nil {
				continue
			}

			for _, f := range fields {
				fLower := strings.ToLower(f)
				matched := false
				for _, q := range fieldNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && (fLower == qLower || strings.HasSuffix(fLower, "::"+qLower) || strings.Contains(fLower, "::"+qLower)) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     sName,
						Type:     "Script Field Reference",
						Snippet:  fmt.Sprintf("References field: %s", f),
					})
					break
				}
			}
		}
	}

	return serializeReferenceResponse("find_field_references_in_scripts", fieldNames, database, matches), nil
}

func parseScriptFieldReferences(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fields []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "FieldReference" || se.Name.Local == "Field" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							fields = append(fields, name)
						}
					}
				}
			}
		}
	}
	return fields, nil
}

func FindFieldReferencesInLayouts(projectPath string, fieldNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			layName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			fields, err := parseLayoutFieldReferences(xmlPath)
			if err != nil {
				continue
			}

			for _, f := range fields {
				fLower := strings.ToLower(f)
				matched := false
				for _, q := range fieldNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && (fLower == qLower || strings.HasSuffix(fLower, "::"+qLower) || strings.Contains(fLower, "::"+qLower)) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     layName,
						Type:     "Layout Field Display",
						Snippet:  fmt.Sprintf("Displays field: %s", f),
					})
					break
				}
			}
		}
	}

	return serializeReferenceResponse("find_field_references_in_layouts", fieldNames, database, matches), nil
}

func parseLayoutFieldReferences(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fields []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "FieldReference" || se.Name.Local == "Field" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							fields = append(fields, name)
						}
					}
				}
			}
		}
	}
	return fields, nil
}

type CalcRef struct {
	FieldName string
	CalcText  string
	CalcType  string
}

func FindFieldReferencesInCalculations(projectPath string, fieldNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		tablesDir := filepath.Join(dbDir, "tables")
		for _, xmlPath := range findXmlFiles(tablesDir) {
			tableName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			calcs, err := parseTableCalculations(xmlPath)
			if err != nil {
				continue
			}

			for _, c := range calcs {
				calcLower := strings.ToLower(c.CalcText)
				matched := false
				for _, q := range fieldNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && strings.Contains(calcLower, qLower) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     fmt.Sprintf("%s::%s", tableName, c.FieldName),
						Type:     c.CalcType,
						Snippet:  c.CalcText,
					})
				}
			}
		}
	}

	return serializeReferenceResponse("find_field_references_in_calculations", fieldNames, database, matches), nil
}

func parseTableCalculations(filePath string) ([]CalcRef, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var refs []CalcRef
	decoder := xml.NewDecoder(file)
	var currentField string
	var inCalc, inAutoEnter, inValidation bool
	var calcText strings.Builder

	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "Field" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						currentField = attr.Value
					}
				}
			} else if se.Name.Local == "Calculation" {
				inCalc = true
				calcText.Reset()
			} else if se.Name.Local == "AutoEnter" {
				inAutoEnter = true
				calcText.Reset()
			} else if se.Name.Local == "Validation" {
				inValidation = true
				calcText.Reset()
			}
		case xml.CharData:
			if inCalc || inAutoEnter || inValidation {
				calcText.Write(se)
			}
		case xml.EndElement:
			if se.Name.Local == "Field" {
				currentField = ""
			} else if se.Name.Local == "Calculation" {
				inCalc = false
				if currentField != "" && calcText.Len() > 0 {
					refs = append(refs, CalcRef{
						FieldName: currentField,
						CalcText:  strings.TrimSpace(calcText.String()),
						CalcType:  "Calculation Field",
					})
				}
			} else if se.Name.Local == "AutoEnter" {
				inAutoEnter = false
				if currentField != "" && calcText.Len() > 0 {
					refs = append(refs, CalcRef{
						FieldName: currentField,
						CalcText:  strings.TrimSpace(calcText.String()),
						CalcType:  "AutoEnter Calculation",
					})
				}
			} else if se.Name.Local == "Validation" {
				inValidation = false
				if currentField != "" && calcText.Len() > 0 {
					refs = append(refs, CalcRef{
						FieldName: currentField,
						CalcText:  strings.TrimSpace(calcText.String()),
						CalcType:  "Validation Calculation",
					})
				}
			}
		}
	}
	return refs, nil
}

func FindFieldReferencesInRelationships(projectPath string, fieldNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		relationshipsDir := filepath.Join(dbDir, "relationships")
		for _, xmlPath := range findXmlFiles(relationshipsDir) {
			rel, err := parseRelationshipXML(xmlPath)
			if err != nil {
				continue
			}

			leftLower := strings.ToLower(rel.LeftFld)
			rightLower := strings.ToLower(rel.RightFld)

			matched := false
			for _, q := range fieldNames {
				qLower := strings.ToLower(strings.TrimSpace(q))
				if qLower != "" && (leftLower == qLower || rightLower == qLower || strings.Contains(leftLower, qLower) || strings.Contains(rightLower, qLower)) {
					matched = true
					break
				}
			}

			if matched {
				relName := fmt.Sprintf("%s (%s::%s %s %s::%s)", 
					strings.TrimSuffix(filepath.Base(xmlPath), ".xml"),
					rel.LeftTO, rel.LeftFld,
					rel.Type,
					rel.RightTO, rel.RightFld,
				)
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     relName,
					Type:     "Relationship Join Key",
					Snippet:  fmt.Sprintf("Join predicate: %s::%s %s %s::%s", rel.LeftTO, rel.LeftFld, rel.Type, rel.RightTO, rel.RightFld),
				})
			}
		}
	}

	return serializeReferenceResponse("find_field_references_in_relationships", fieldNames, database, matches), nil
}

// 4. Variable-Centric Tools

func FindVariableReferencesInScripts(projectPath string, variableNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var queries []string
	for _, vName := range variableNames {
		q := strings.ToLower(strings.TrimSpace(vName))
		if q != "" {
			if !strings.HasPrefix(q, "$") {
				q = "$" + q
			}
			queries = append(queries, q)
		}
	}
	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		sanitizedDir := filepath.Join(dbDir, "scripts_sanitized")
		if _, err := os.Stat(sanitizedDir); os.IsNotExist(err) {
			continue
		}

		var files []string
		filepath.Walk(sanitizedDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".txt") {
				files = append(files, path)
			}
			return nil
		})

		for _, txtPath := range files {
			data, err := os.ReadFile(txtPath)
			if err != nil {
				continue
			}

			lines := strings.Split(string(data), "\n")
			scriptBaseName := strings.TrimSuffix(filepath.Base(txtPath), ".txt")

			for lineNum, line := range lines {
				lineLower := strings.ToLower(line)
				matched := false
				var matchedQ string
				for _, q := range queries {
					if strings.Contains(lineLower, q) {
						matched = true
						matchedQ = q
						break
					}
				}
				if matched {
					xmlPath := filepath.Join(dbDir, "scripts", scriptBaseName+".xml")
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     scriptBaseName,
						Type:     "Script Variable Reference",
						Snippet:  fmt.Sprintf("Line %d: %s (matched %s)", lineNum+1, strings.TrimSpace(line), matchedQ),
					})
				}
			}
		}
	}

	return serializeReferenceResponse("find_variable_references_in_scripts", variableNames, database, matches), nil
}

// 5. Value List & Layout Calculations

func FindValueListReferencesInCalculations(projectPath string, valueListNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		tablesDir := filepath.Join(dbDir, "tables")
		for _, xmlPath := range findXmlFiles(tablesDir) {
			tableName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			calcs, err := parseTableCalculations(xmlPath)
			if err != nil {
				continue
			}

			for _, c := range calcs {
				calcLower := strings.ToLower(c.CalcText)
				matched := false
				for _, q := range valueListNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && strings.Contains(calcLower, qLower) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     fmt.Sprintf("%s::%s", tableName, c.FieldName),
						Type:     c.CalcType,
						Snippet:  c.CalcText,
					})
				}
			}
		}
	}

	return serializeReferenceResponse("find_valuelist_references_in_calculations", valueListNames, database, matches), nil
}

func FindLayoutReferencesInCalculations(projectPath string, layoutNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		tablesDir := filepath.Join(dbDir, "tables")
		for _, xmlPath := range findXmlFiles(tablesDir) {
			tableName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			calcs, err := parseTableCalculations(xmlPath)
			if err != nil {
				continue
			}

			for _, c := range calcs {
				calcLower := strings.ToLower(c.CalcText)
				matched := false
				for _, q := range layoutNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && strings.Contains(calcLower, qLower) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: filepath.Base(dbDir),
						Path:     xmlPath,
						Name:     fmt.Sprintf("%s::%s", tableName, c.FieldName),
						Type:     c.CalcType,
						Snippet:  c.CalcText,
					})
				}
			}
		}
	}

	return serializeReferenceResponse("find_layout_references_in_calculations", layoutNames, database, matches), nil
}

// 6. Relationship & Schema Tools

func FindTOReferences(projectPath string, toNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		dbName := filepath.Base(dbDir)

		// 1. Scan layouts
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			layName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			tos, err := parseLayoutTables(xmlPath)
			if err == nil {
				for _, t := range tos {
					tLower := strings.ToLower(t)
					matched := false
					for _, q := range toNames {
						qLower := strings.ToLower(strings.TrimSpace(q))
						if qLower != "" && tLower == qLower {
							matched = true
							break
						}
					}
					if matched {
						matches = append(matches, ReferenceMatch{
							Database: dbName,
							Path:     xmlPath,
							Name:     layName,
							Type:     "Layout TO Reference",
							Snippet:  fmt.Sprintf("Layout references Table Occurrence: %s", t),
						})
					}
				}
			}
		}

		// 2. Scan scripts
		scriptsDir := filepath.Join(dbDir, "scripts")
		for _, xmlPath := range findXmlFiles(scriptsDir) {
			scrName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			tos, err := parseScriptTOReferences(xmlPath)
			if err == nil {
				for _, t := range tos {
					tLower := strings.ToLower(t)
					matched := false
					for _, q := range toNames {
						qLower := strings.ToLower(strings.TrimSpace(q))
						if qLower != "" && tLower == qLower {
							matched = true
							break
						}
					}
					if matched {
						matches = append(matches, ReferenceMatch{
							Database: dbName,
							Path:     xmlPath,
							Name:     scrName,
							Type:     "Script TO Reference",
							Snippet:  fmt.Sprintf("Script references Table Occurrence: %s", t),
						})
					}
				}
			}
		}

		// 3. Scan relationships
		relationshipsDir := filepath.Join(dbDir, "relationships")
		for _, xmlPath := range findXmlFiles(relationshipsDir) {
			relName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
			rel, err := parseRelationshipXML(xmlPath)
			if err == nil {
				leftLower := strings.ToLower(rel.LeftTO)
				rightLower := strings.ToLower(rel.RightTO)
				matched := false
				for _, q := range toNames {
					qLower := strings.ToLower(strings.TrimSpace(q))
					if qLower != "" && (leftLower == qLower || rightLower == qLower) {
						matched = true
						break
					}
				}
				if matched {
					matches = append(matches, ReferenceMatch{
						Database: dbName,
						Path:     xmlPath,
						Name:     relName,
						Type:     "Relationship TO Reference",
						Snippet:  fmt.Sprintf("Join predicate references Table Occurrence left: %s or right: %s", rel.LeftTO, rel.RightTO),
					})
				}
			}
		}
	}

	return serializeReferenceResponse("find_to_references", toNames, database, matches), nil
}

func parseScriptTOReferences(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tos []string
	seen := make(map[string]bool)
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "TableOccurrenceReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						name := attr.Value
						if !seen[name] {
							seen[name] = true
							tos = append(tos, name)
						}
					}
				}
			}
			if se.Name.Local == "FieldReference" || se.Name.Local == "Field" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" && attr.Value != "" {
						val := attr.Value
						if idx := strings.Index(val, "::"); idx != -1 {
							to := val[:idx]
							if to != "" && !seen[to] {
								seen[to] = true
								tos = append(tos, to)
							}
						}
					}
				}
			}
		}
	}
	return tos, nil
}

func FindRelationshipPredicates(projectPath string, toNames []string, database string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	var matches []ReferenceMatch

	for _, dbDir := range dbDirs {
		relationshipsDir := filepath.Join(dbDir, "relationships")
		for _, xmlPath := range findXmlFiles(relationshipsDir) {
			rel, err := parseRelationshipXML(xmlPath)
			if err != nil {
				continue
			}

			leftLower := strings.ToLower(rel.LeftTO)
			rightLower := strings.ToLower(rel.RightTO)

			matched := false
			for _, q := range toNames {
				qLower := strings.ToLower(strings.TrimSpace(q))
				if qLower != "" && (leftLower == qLower || rightLower == qLower) {
					matched = true
					break
				}
			}

			if matched {
				relName := strings.TrimSuffix(filepath.Base(xmlPath), ".xml")
				matches = append(matches, ReferenceMatch{
					Database: filepath.Base(dbDir),
					Path:     xmlPath,
					Name:     relName,
					Type:     "Relationship Join Predicate",
					Snippet:  fmt.Sprintf("Left: %s::%s | Operator: %s | Right: %s::%s", rel.LeftTO, rel.LeftFld, rel.Type, rel.RightTO, rel.RightFld),
				})
			}
		}
	}

	return serializeReferenceResponse("find_relationship_predicates", toNames, database, matches), nil
}
