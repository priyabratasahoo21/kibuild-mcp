package tools

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ResolveAndSandboxPath(projectPath string, inputPath string) (string, error) {
	if projectPath == "" {
		return "", fmt.Errorf("no active project directory set")
	}

	// Resolve absolute path
	absPath := inputPath
	if !filepath.IsAbs(inputPath) {
		absPath = filepath.Join(projectPath, inputPath)
	}

	absPath = filepath.Clean(absPath)
	projClean := filepath.Clean(projectPath)

	// Use Rel to detect traversal — prevents the HasPrefix ambiguity where
	// /proj/foobar would pass the prefix check for project /proj/foo.
	rel, err := filepath.Rel(projClean, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("security boundary violation: access denied to path outside workspace: %s", inputPath)
	}

	// Fallback to case-insensitive or structural match if path does not exist
	if _, errStat := os.Stat(absPath); os.IsNotExist(errStat) {
		if altPath := findAlternativePath(projClean, rel); altPath != "" {
			altRel, errRel := filepath.Rel(projClean, altPath)
			if errRel == nil && !strings.HasPrefix(altRel, "..") {
				return altPath, nil
			}
		}
	}

	return absPath, nil
}

func findAlternativePath(projectPath string, relPath string) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	var cleanParts []string
	for _, p := range parts {
		if p != "" && p != "." && p != ".." {
			cleanParts = append(cleanParts, p)
		}
	}
	if len(cleanParts) == 0 {
		return ""
	}

	for suffixLen := len(cleanParts); suffixLen >= 1; suffixLen-- {
		startIdx := len(cleanParts) - suffixLen
		firstPartLower := strings.ToLower(cleanParts[startIdx])
		if startIdx == 0 && (firstPartLower == "schema" || firstPartLower == "source" || firstPartLower == "archives") {
			if suffixLen > 1 {
				startIdx++
			}
		}

		suffixParts := cleanParts[startIdx:]
		suffixPath := filepath.Join(suffixParts...)

		foundPath := ""
		filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == ".gemini" {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			rel, err := filepath.Rel(projectPath, path)
			if err != nil {
				return nil
			}

			relLower := strings.ToLower(filepath.ToSlash(rel))
			suffixLower := strings.ToLower(filepath.ToSlash(suffixPath))

			if relLower == suffixLower || strings.HasSuffix(relLower, "/"+suffixLower) {
				foundPath = path
				return filepath.SkipAll
			}
			return nil
		})

		if foundPath != "" {
			return foundPath
		}
	}

	return ""
}

type FileItem struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

func ListDir(path string) (string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var items []FileItem
	for _, entry := range entries {
		info, err := entry.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		items = append(items, FileItem{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	bytesVal, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytesVal), nil
}

func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func WriteFile(path string, content string) (string, error) {
	// Create parent dir if not exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

func GenerateSchemaMap(projectPath string) (string, error) {
	if projectPath == "" {
		return "", fmt.Errorf("no project path provided")
	}

	// Automatically ensure the business glossary is created in the project
	_ = EnsureBusinessGlossary(projectPath)

	dbPaths, err := findDatabaseDirs(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to discover database directories: %v", err)
	}

	// Freshness guard: skip regen if workspace_map.md is newer than all schema files.
	mapFilePath := filepath.Join(projectPath, "workspace_map.md")
	if mapInfo, err := os.Stat(mapFilePath); err == nil {
		mapModTime := mapInfo.ModTime()
		fresh := true
		for _, dbPath := range dbPaths {
			if walkErr := filepath.WalkDir(dbPath, func(_ string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil || d.IsDir() {
					return walkErr
				}
				info, statErr := d.Info()
				if statErr == nil && info.ModTime().After(mapModTime) {
					fresh = false
				}
				return nil
			}); walkErr != nil {
				fresh = false
				break
			}
			if !fresh {
				break
			}
		}
		if fresh {
			return "workspace_map.md is up-to-date (schema unchanged since last index).", nil
		}
	}

	var sb strings.Builder
	sb.WriteString("# KiBuild Project Workspace Map\n\n")
	sb.WriteString("This file summarizes the structures of the database files in this workspace.\n\n")

	var foundDbs int
	for _, dbPath := range dbPaths {
		foundDbs++
		relDbPath, _ := filepath.Rel(projectPath, dbPath)
		dbName := filepath.Base(dbPath)

		sb.WriteString(fmt.Sprintf("## 🗄️ Database: %s (`%s`)\n\n", dbName, relDbPath))
		// 1. Tables (No fields)
		sb.WriteString("### 📊 Tables\n\n")
		tablesPath := filepath.Join(dbPath, "tables")
		if _, err := os.Stat(tablesPath); err == nil {
			tableFiles, _ := os.ReadDir(tablesPath)
			var foundTables bool
			for _, tf := range tableFiles {
				if tf.IsDir() || !strings.HasSuffix(tf.Name(), ".xml") {
					continue
				}
				tfPath := filepath.Join(tablesPath, tf.Name())
				tName, _, err := parseTableXML(tfPath)
				if err == nil && tName != "" {
					foundTables = true
					relTfPath, _ := filepath.Rel(projectPath, tfPath)
					fileURI := fmt.Sprintf("file://%s", filepath.ToSlash(tfPath))
					sb.WriteString(fmt.Sprintf("- **%s** ([%s](%s))\n", tName, filepath.ToSlash(relTfPath), fileURI))
				}
			}
			if !foundTables {
				sb.WriteString("*No tables found.*\n")
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("*No tables folder found.*\n\n")
		}

		// 2. Layouts (No script triggers/nested links)
		sb.WriteString("### 🖥️ Layouts\n\n")
		layoutsPath := filepath.Join(dbPath, "layouts")
		if _, err := os.Stat(layoutsPath); err == nil {
			layoutFiles := findXmlFiles(layoutsPath)
			var foundLayouts bool
			for _, lfPath := range layoutFiles {
				layName, toName, _, _, err := parseLayoutXML(lfPath)
				if err == nil && layName != "" {
					foundLayouts = true
					relLfPath, _ := filepath.Rel(projectPath, lfPath)
					fileURI := fmt.Sprintf("file://%s", filepath.ToSlash(lfPath))
					if toName != "" {
						sb.WriteString(fmt.Sprintf("- **%s** (bound to table occurrence `::%s`) ([%s](%s))\n", layName, toName, filepath.ToSlash(relLfPath), fileURI))
					} else {
						sb.WriteString(fmt.Sprintf("- **%s** ([%s](%s))\n", layName, filepath.ToSlash(relLfPath), fileURI))
					}
				}
			}
			if !foundLayouts {
				sb.WriteString("*No layouts found.*\n")
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("*No layouts folder found.*\n\n")
		}

		// 3. Table Occurrences Mapping (Replaces relationships graph)
		sb.WriteString("### 📊 Table Occurrences\n\n")
		toPath := filepath.Join(dbPath, "table_occurrences")
		if _, err := os.Stat(toPath); err == nil {
			toFiles, _ := os.ReadDir(toPath)
			var foundTOs bool
			for _, tf := range toFiles {
				if !tf.IsDir() && strings.HasSuffix(tf.Name(), ".xml") {
					toDef, err := parseTableOccurrenceXML(filepath.Join(toPath, tf.Name()))
					if err == nil && toDef.Name != "" {
						foundTOs = true
						sb.WriteString(fmt.Sprintf("- **%s** (Base Table: `%s`)\n", toDef.Name, toDef.BaseTable))
					}
				}
			}
			if !foundTOs {
				sb.WriteString("*No Table Occurrences found.*\n")
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("*No table occurrences folder found.*\n\n")
		}

		// 4. Scripts (Truncated to top 30)
		sb.WriteString("### 📜 Script Hierarchy\n\n")
		scriptsPath := filepath.Join(dbPath, "scripts_sanitized")
		if _, err := os.Stat(scriptsPath); os.IsNotExist(err) {
			scriptsPath = filepath.Join(dbPath, "scripts")
		}
		if _, err := os.Stat(scriptsPath); err == nil {
			scriptLines, err := buildScriptList(scriptsPath, "")
			if err == nil && len(scriptLines) > 0 {
				limit := 30
				for i, line := range scriptLines {
					if i >= limit {
						sb.WriteString(fmt.Sprintf("... and %d more scripts.\n", len(scriptLines)-limit))
						break
					}
					sb.WriteString(line + "\n")
				}
			} else {
				sb.WriteString("*No scripts found.*\n")
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString("*No scripts folder found.*\n\n")
		}

		sb.WriteString("---\n\n")
	}

	if foundDbs == 0 {
		sb.WriteString("*No databases found in project workspace.*\n")
	}

	mapContent := sb.String()

	mapFilePath = filepath.Join(projectPath, "workspace_map.md")
	if err := os.WriteFile(mapFilePath, []byte(mapContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write workspace map file: %v", err)
	}

	return fmt.Sprintf("Successfully generated workspace schema map and saved to workspace_map.md (%d bytes)", len(mapContent)), nil
}

func EnsureBusinessGlossary(projectPath string) error {
	if projectPath == "" {
		return fmt.Errorf("empty project path")
	}
	contextDir := filepath.Join(projectPath, "context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}
	glossaryPath := filepath.Join(contextDir, "business_glossary.md")
	if _, err := os.Stat(glossaryPath); os.IsNotExist(err) {
		defaultContent := `# Business Glossary

Define your business terms, acronyms, and rules here to help the agent understand your domain.

- **Term**: Definition or description of the term.
`
		if errWrite := os.WriteFile(glossaryPath, []byte(defaultContent), 0644); errWrite != nil {
			return fmt.Errorf("failed to write default business glossary: %w", errWrite)
		}
	}
	return nil
}

func ParseTableXML(filePath string) (string, []string, error) {
	return parseTableXML(filePath)
}

func ParseLayoutXML(filePath string) (string, string, []string, []string, error) {
	return parseLayoutXML(filePath)
}

func parseTableXML(filePath string) (string, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	var tableName string
	var fields []string

	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "BaseTableReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						tableName = attr.Value
					}
				}
			}
			if se.Name.Local == "Field" {
				var name, dataType, fieldType string
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						name = attr.Value
					} else if attr.Name.Local == "datatype" {
						dataType = attr.Value
					} else if attr.Name.Local == "fieldtype" {
						fieldType = attr.Value
					}
				}
				if name != "" {
					typeDetail := dataType
					if fieldType != "" && fieldType != "Normal" {
						typeDetail += " " + fieldType
					}
					fields = append(fields, fmt.Sprintf("%s [%s]", name, typeDetail))
				}
			}
		}
	}
	return tableName, fields, nil
}

func parseLayoutXML(filePath string) (string, string, []string, []string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer file.Close()

	var layoutName string
	var toName string
	var scripts []string
	var layouts []string
	scriptMap := make(map[string]bool)
	layoutMap := make(map[string]bool)

	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "Layout" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						layoutName = attr.Value
					}
				}
			}
			if se.Name.Local == "TableOccurrenceReference" {
				if toName == "" {
					for _, attr := range se.Attr {
						if attr.Name.Local == "name" {
							toName = attr.Value
						}
					}
				}
			}
			if se.Name.Local == "ScriptReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						name := attr.Value
						if name != "" && !scriptMap[name] {
							scriptMap[name] = true
							scripts = append(scripts, name)
						}
					}
				}
			}
			if se.Name.Local == "LayoutReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						name := attr.Value
						if name != "" && !layoutMap[name] {
							layoutMap[name] = true
							layouts = append(layouts, name)
						}
					}
				}
			}
		}
	}
	return layoutName, toName, scripts, layouts, nil
}

func findDatabaseDirs(root string) ([]string, error) {
	var dbDirs []string
	dbDirMap := make(map[string]bool)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if name == ".git" || name == "node_modules" || name == ".gemini" || name == "build" || name == "Plugin" || name == "WebUI" || name == "sidecar" {
			return filepath.SkipDir
		}

		pathLower := strings.ToLower(filepath.ToSlash(path))
		if strings.Contains(pathLower, "/outbox") || strings.HasSuffix(pathLower, "/outbox") {
			return filepath.SkipDir
		}

		if name == "tables" || name == "layouts" || name == "scripts" || name == "scripts_sanitized" {
			parent := filepath.Dir(path)
			if !dbDirMap[parent] {
				dbDirMap[parent] = true
				dbDirs = append(dbDirs, parent)
			}
		}
		return nil
	})
	return dbDirs, err
}

func findXmlFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".xml") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

type TableOccurrenceDef struct {
	Name      string
	BaseTable string
}

func parseTableOccurrenceXML(filePath string) (TableOccurrenceDef, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return TableOccurrenceDef{}, err
	}
	defer file.Close()

	var to TableOccurrenceDef
	decoder := xml.NewDecoder(file)
	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "TableOccurrence" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						to.Name = attr.Value
					}
				}
			}
			if se.Name.Local == "BaseTableReference" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						to.BaseTable = attr.Value
					}
				}
			}
		}
	}
	return to, nil
}

type RelationshipDef struct {
	LeftTO   string
	RightTO  string
	LeftFld  string
	RightFld string
	Type     string
}

func ParseRelationshipXML(filePath string) (RelationshipDef, error) {
	return parseRelationshipXML(filePath)
}

func parseRelationshipXML(filePath string) (RelationshipDef, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return RelationshipDef{}, err
	}
	defer file.Close()

	var rel RelationshipDef
	decoder := xml.NewDecoder(file)
	var elementStack []string

	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			localName := se.Name.Local
			elementStack = append(elementStack, localName)

			if localName == "JoinPredicate" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "type" {
						rel.Type = attr.Value
					}
				}
			}

			if localName == "TableOccurrenceReference" {
				var name string
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						name = attr.Value
					}
				}
				if hasParent(elementStack, "LeftTable") {
					rel.LeftTO = name
				} else if hasParent(elementStack, "RightTable") {
					rel.RightTO = name
				}
			}

			if localName == "FieldReference" {
				var name string
				for _, attr := range se.Attr {
					if attr.Name.Local == "name" {
						name = attr.Value
					}
				}
				if hasParent(elementStack, "LeftField") {
					rel.LeftFld = name
				} else if hasParent(elementStack, "RightField") {
					rel.RightFld = name
				}
			}

		case xml.EndElement:
			if len(elementStack) > 0 {
				elementStack = elementStack[:len(elementStack)-1]
			}
		}
	}
	return rel, nil
}

func hasParent(stack []string, name string) bool {
	for _, s := range stack {
		if s == name {
			return true
		}
	}
	return false
}

func sanitizeMermaidID(name string) string {
	id := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
	return strings.Trim(id, "_")
}

func buildScriptList(dirPath string, prefix string) ([]string, error) {
	var lines []string
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			lines = append(lines, fmt.Sprintf("%s- 📁 %s/", prefix, entry.Name()))
			subLines, _ := buildScriptList(filepath.Join(dirPath, entry.Name()), prefix+"  ")
			lines = append(lines, subLines...)
		} else {
			name := entry.Name()
			if strings.HasSuffix(name, ".xml") {
				name = strings.TrimSuffix(name, ".xml")
			} else if strings.HasSuffix(name, ".txt") {
				name = strings.TrimSuffix(name, ".txt")
			}
			lines = append(lines, fmt.Sprintf("%s- 📄 %s", prefix, name))
		}
	}
	return lines, nil
}

// SearchFile searches for a text pattern in files under rootPath using grep.
// Returns matching lines with file names and line numbers, capped at 50 matches.
func SearchFile(rootPath string, pattern string, caseInsensitive bool) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("search pattern must not be empty")
	}

	args := []string{"-rnI", "--max-count=50"}
	if caseInsensitive {
		args = append(args, "-i")
	}

	// Exclude common non-useful directories
	args = append(args,
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=build",
		"--exclude-dir=.gemini",
	)
	args = append(args, pattern, rootPath)

	cmd := exec.Command("grep", args...)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))

	if err != nil {
		// grep returns exit code 1 when no matches found — not a real error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found.", nil
		}
		if result != "" {
			return result, nil // partial output is still useful
		}
		return "", fmt.Errorf("search failed: %v", err)
	}

	// Count matches and truncate if over 50 lines
	lines := strings.Split(result, "\n")
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... truncated (%d total matches)", len(lines))
	}

	return result, nil
}

// DiffPatch overwrites file content at target path and returns sizing metrics
func DiffPatch(filePath string, patchedContent string) (string, error) {
	original, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filePath, []byte(patchedContent), 0644); err != nil {
				return "", err
			}
			return fmt.Sprintf("Created new file at %s with %d bytes", filePath, len(patchedContent)), nil
		}
		return "", err
	}

	if err := os.WriteFile(filePath, []byte(patchedContent), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Updated %s. Original size: %d bytes, New size: %d bytes", filePath, len(original), len(patchedContent)), nil
}

// ReadXMLGuide reads the FileMaker XML snippet guide from Docs/FileMaker_XML_Guide.md with self-healing fallbacks
func ReadXMLGuide(projectPath string) (string, error) {
	var pathsToTry []string

	if projectPath != "" {
		pathsToTry = append(pathsToTry, filepath.Join(projectPath, "Docs", "FileMaker_XML_Guide.md"))
		// Try relative to projectPath's sibling "KiBuild Plugin"
		pathsToTry = append(pathsToTry, filepath.Join(filepath.Dir(projectPath), "KiBuild Plugin", "Docs", "FileMaker_XML_Guide.md"))
	}

	if cwd, err := os.Getwd(); err == nil {
		pathsToTry = append(pathsToTry, filepath.Join(cwd, "Docs", "FileMaker_XML_Guide.md"))
	}

	// Try standard home documents path
	if home, err := os.UserHomeDir(); err == nil {
		pathsToTry = append(pathsToTry, filepath.Join(home, "Documents", "KiBuild Plugin", "Docs", "FileMaker_XML_Guide.md"))
	}

	// Try executable parent traversal
	if execPath, err := os.Executable(); err == nil {
		dir := filepath.Dir(execPath)
		for i := 0; i < 5; i++ {
			pathsToTry = append(pathsToTry, filepath.Join(dir, "Docs", "FileMaker_XML_Guide.md"))
			if filepath.Base(dir) == "Docs" {
				pathsToTry = append(pathsToTry, filepath.Join(dir, "FileMaker_XML_Guide.md"))
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, guidePath := range pathsToTry {
		if data, err := os.ReadFile(guidePath); err == nil {
			return string(data), nil
		}
	}

	errorMessage := "FileMaker XML guide not found. Tried paths:\n"
	for _, p := range pathsToTry {
		errorMessage += fmt.Sprintf("- %s\n", p)
	}
	return "", fmt.Errorf("%sEnsure you are in the correct workspace.", errorMessage)
}

func schemaDatabaseDirs(projectPath string, database string) ([]string, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("no active project directory set")
	}

	dbQuery := strings.ToLower(strings.TrimSpace(database))
	var roots []string

	if dbQuery != "" {
		candidates := []string{
			filepath.Join(projectPath, "files", "Schema", database),
			filepath.Join(projectPath, "Schema", database),
			filepath.Join(projectPath, database),
			projectPath,
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				base := strings.ToLower(filepath.Base(candidate))
				if candidate == projectPath || base == dbQuery || strings.Contains(base, dbQuery) {
					roots = append(roots, candidate)
				}
			}
		}
	}

	discovered, err := findDatabaseDirs(projectPath)
	if err != nil {
		return nil, err
	}
	for _, dbPath := range discovered {
		if dbQuery == "" {
			roots = append(roots, dbPath)
			continue
		}
		base := strings.ToLower(filepath.Base(dbPath))
		if base == dbQuery || strings.Contains(base, dbQuery) {
			roots = append(roots, dbPath)
		}
	}

	seen := make(map[string]bool)
	var unique []string
	for _, root := range roots {
		clean := filepath.Clean(root)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		unique = append(unique, clean)
	}
	return unique, nil
}

func nameMatchScore(name string, fileName string, query string) int {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	fileLower := strings.ToLower(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	queryLower := strings.ToLower(strings.TrimSpace(query))

	switch {
	case nameLower == queryLower:
		return 0
	case fileLower == queryLower:
		return 1
	case strings.HasPrefix(nameLower, queryLower):
		return 2
	case strings.HasPrefix(fileLower, queryLower):
		return 3
	case strings.Contains(nameLower, queryLower):
		return 4
	case strings.Contains(fileLower, queryLower):
		return 5
	default:
		return -1
	}
}

// FindTable searches exploded schema for a base table and returns a compact JSON payload.
func FindTable(projectPath string, tableName string, database string) (string, error) {
	if strings.TrimSpace(tableName) == "" {
		return "", fmt.Errorf("table_name is required")
	}

	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	type tableMatch struct {
		Database string   `json:"database"`
		Table    string   `json:"table"`
		XMLPath  string   `json:"xml_path"`
		Fields   []string `json:"fields"`
		Score    int      `json:"-"`
	}

	var matches []tableMatch
	for _, dbDir := range dbDirs {
		tablesDir := filepath.Join(dbDir, "tables")
		entries, err := os.ReadDir(tablesDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".xml" {
				continue
			}
			xmlPath := filepath.Join(tablesDir, entry.Name())
			parsedName, fields, err := parseTableXML(xmlPath)
			if err != nil {
				continue
			}
			score := nameMatchScore(parsedName, entry.Name(), tableName)
			if score < 0 {
				continue
			}
			matches = append(matches, tableMatch{
				Database: filepath.Base(dbDir),
				Table:    parsedName,
				XMLPath:  xmlPath,
				Fields:   fields,
				Score:    score,
			})
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("Table %q not found. Searched %d database schema root(s).", tableName, len(dbDirs)), nil
	}

	best := matches[0]
	for _, m := range matches[1:] {
		if m.Score < best.Score {
			best = m
		}
	}

	response := struct {
		Kind           string     `json:"kind"`
		Match          tableMatch `json:"match"`
		CandidateCount int        `json:"candidate_count"`
		Note           string     `json:"note"`
	}{
		Kind:           "table",
		Match:          best,
		CandidateCount: len(matches),
		Note:           "Use xml_path only when exact table XML is needed. Use fields for token-efficient planning.",
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FindLayout searches exploded schema for a layout and returns references useful for planning.
func FindLayout(projectPath string, layoutName string, database string) (string, error) {
	if strings.TrimSpace(layoutName) == "" {
		return "", fmt.Errorf("layout_name is required")
	}

	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	type layoutMatch struct {
		Database          string   `json:"database"`
		Layout            string   `json:"layout"`
		TableOccurrence   string   `json:"table_occurrence"`
		XMLPath           string   `json:"xml_path"`
		ReferencedScripts []string `json:"referenced_scripts"`
		ReferencedLayouts []string `json:"referenced_layouts"`
		Score             int      `json:"-"`
	}

	var matches []layoutMatch
	for _, dbDir := range dbDirs {
		layoutsDir := filepath.Join(dbDir, "layouts")
		for _, xmlPath := range findXmlFiles(layoutsDir) {
			parsedName, toName, scripts, layouts, err := parseLayoutXML(xmlPath)
			if err != nil {
				continue
			}
			score := nameMatchScore(parsedName, filepath.Base(xmlPath), layoutName)
			if score < 0 {
				continue
			}
			matches = append(matches, layoutMatch{
				Database:          filepath.Base(dbDir),
				Layout:            parsedName,
				TableOccurrence:   toName,
				XMLPath:           xmlPath,
				ReferencedScripts: scripts,
				ReferencedLayouts: layouts,
				Score:             score,
			})
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("Layout %q not found. Searched %d database schema root(s).", layoutName, len(dbDirs)), nil
	}

	best := matches[0]
	for _, m := range matches[1:] {
		if m.Score < best.Score {
			best = m
		}
	}

	response := struct {
		Kind           string      `json:"kind"`
		Match          layoutMatch `json:"match"`
		CandidateCount int         `json:"candidate_count"`
		Note           string      `json:"note"`
	}{
		Kind:           "layout",
		Match:          best,
		CandidateCount: len(matches),
		Note:           "Use referenced scripts/layouts for impact review before proposing layout or WebViewer changes.",
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// InspectRelationships returns relationship predicates from exploded relationship XML.
func InspectRelationships(projectPath string, database string, tableOccurrence string) (string, error) {
	dbDirs, err := schemaDatabaseDirs(projectPath, database)
	if err != nil {
		return "", err
	}

	toFilter := strings.ToLower(strings.TrimSpace(tableOccurrence))
	type relationshipItem struct {
		Database   string `json:"database"`
		XMLPath    string `json:"xml_path"`
		LeftTO     string `json:"left_to"`
		LeftField  string `json:"left_field"`
		Operator   string `json:"operator"`
		RightTO    string `json:"right_to"`
		RightField string `json:"right_field"`
	}

	var items []relationshipItem
	for _, dbDir := range dbDirs {
		relationshipsDir := filepath.Join(dbDir, "relationships")
		for _, xmlPath := range findXmlFiles(relationshipsDir) {
			rel, err := parseRelationshipXML(xmlPath)
			if err != nil {
				continue
			}
			if rel.LeftTO == "" && rel.RightTO == "" {
				continue
			}
			if toFilter != "" &&
				!strings.Contains(strings.ToLower(rel.LeftTO), toFilter) &&
				!strings.Contains(strings.ToLower(rel.RightTO), toFilter) {
				continue
			}
			operator := rel.Type
			if operator == "" {
				operator = "Unknown"
			}
			items = append(items, relationshipItem{
				Database:   filepath.Base(dbDir),
				XMLPath:    xmlPath,
				LeftTO:     rel.LeftTO,
				LeftField:  rel.LeftFld,
				Operator:   operator,
				RightTO:    rel.RightTO,
				RightField: rel.RightFld,
			})
		}
	}

	response := struct {
		Kind           string             `json:"kind"`
		DatabaseFilter string             `json:"database_filter,omitempty"`
		TOFilter       string             `json:"table_occurrence_filter,omitempty"`
		Count          int                `json:"count"`
		Relationships  []relationshipItem `json:"relationships"`
		Note           string             `json:"note"`
	}{
		Kind:           "relationships",
		DatabaseFilter: database,
		TOFilter:       tableOccurrence,
		Count:          len(items),
		Relationships:  items,
		Note:           "Use this for relationship impact review. It is read-only and based on exploded XML.",
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ValidateWebViewerHTML performs a lightweight security and portability scan for generated WebViewer HTML.
func ValidateWebViewerHTML(html string, allowRemoteAssets bool) (string, error) {
	if strings.TrimSpace(html) == "" {
		return "", fmt.Errorf("html is required")
	}

	lower := strings.ToLower(html)
	var errors []string
	var warnings []string

	remotePatterns := []string{
		`src="http`, `src='http`,
		`href="http`, `href='http`,
		`url(http`, `@import`,
		`<iframe`,
	}
	var remoteHits []string
	for _, pattern := range remotePatterns {
		if strings.Contains(lower, pattern) {
			remoteHits = append(remoteHits, pattern)
		}
	}
	if len(remoteHits) > 0 {
		msg := fmt.Sprintf("remote or embedded external dependency markers found: %s", strings.Join(remoteHits, ", "))
		if allowRemoteAssets {
			warnings = append(warnings, msg)
		} else {
			errors = append(errors, msg)
		}
	}

	blockedAPIs := []string{
		"eval(",
		"new function(",
		"document.cookie",
		"navigator.sendbeacon",
		"websocket(",
		"xmlhttprequest",
		"window.open(",
	}
	for _, api := range blockedAPIs {
		if strings.Contains(lower, api) {
			errors = append(errors, fmt.Sprintf("risky JavaScript API found: %s", api))
		}
	}

	warnAPIs := []string{
		"localstorage",
		"sessionstorage",
		"fetch(",
		"postmessage(",
		"innerhtml",
	}
	for _, api := range warnAPIs {
		if strings.Contains(lower, api) {
			warnings = append(warnings, fmt.Sprintf("review JavaScript API usage: %s", api))
		}
	}

	sizeBytes := len([]byte(html))
	if sizeBytes > 500*1024 {
		warnings = append(warnings, fmt.Sprintf("HTML bundle is %d bytes; consider splitting or simplifying for FileMaker WebViewer use", sizeBytes))
	}

	hasFileMakerBridge := strings.Contains(lower, "filemaker.performscript")
	if !hasFileMakerBridge {
		warnings = append(warnings, "no FileMaker.PerformScript bridge call found; this may be preview-only HTML")
	}

	response := struct {
		Kind     string   `json:"kind"`
		Passed   bool     `json:"passed"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
		Stats    struct {
			Bytes              int  `json:"bytes"`
			HasScriptTag       bool `json:"has_script_tag"`
			HasFileMakerBridge bool `json:"has_filemaker_bridge"`
			RemoteHitCount     int  `json:"remote_hit_count"`
		} `json:"stats"`
	}{
		Kind:     "webviewer_html_validation",
		Passed:   len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
	response.Stats.Bytes = sizeBytes
	response.Stats.HasScriptTag = strings.Contains(lower, "<script")
	response.Stats.HasFileMakerBridge = hasFileMakerBridge
	response.Stats.RemoteHitCount = len(remoteHits)

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SearchIndex searches the workspace_map.md index for matching lines using a keyword query.
// Returns only the matching lines (token-efficient) rather than the entire index file.
// type can be: "script", "layout", "table", "field", "all" (default: all).
func SearchIndex(projectPath string, query string, filterType string) (string, error) {
	if projectPath == "" {
		return "", fmt.Errorf("no active project directory set")
	}
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Locate workspace_map.md
	candidates := []string{
		filepath.Join(projectPath, "workspace_map.md"),
		filepath.Join(projectPath, "files", "workspace_map.md"),
	}
	var mapPath string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			mapPath = c
			break
		}
	}
	if mapPath == "" {
		return "workspace_map.md not found. Call generate_schema_map first to build the index.", nil
	}

	data, err := os.ReadFile(mapPath)
	if err != nil {
		return "", fmt.Errorf("failed to read workspace_map.md: %w", err)
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	filterLower := strings.ToLower(strings.TrimSpace(filterType))

	// Section headings that indicate what type of content follows
	sectionTypeMap := map[string]string{
		"## scripts":  "script",
		"## layouts":  "layout",
		"## tables":   "table",
		"## fields":   "field",
		"### scripts": "script",
		"### layouts": "layout",
		"### tables":  "table",
		"### fields":  "field",
	}

	lines := strings.Split(string(data), "\n")
	var matches []string
	currentSection := "all"

	const maxMatches = 60

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Detect section
		for heading, secType := range sectionTypeMap {
			if strings.HasPrefix(strings.ToLower(trimmed), heading) {
				currentSection = secType
			}
		}

		// Filter by type if specified
		if filterLower != "" && filterLower != "all" && currentSection != "all" && currentSection != filterLower {
			continue
		}

		// Match query
		if strings.Contains(strings.ToLower(line), queryLower) {
			matches = append(matches, line)
			if len(matches) >= maxMatches {
				break
			}
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No results for %q in workspace_map.md (type filter: %q).\nTip: Try a broader query or call search_file for full-text search across scripts.", query, filterType), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("search_index results: %d match(es) for %q", len(matches), query))
	if filterLower != "" && filterLower != "all" {
		sb.WriteString(fmt.Sprintf(" [type=%s]", filterLower))
	}
	sb.WriteString("\n")
	sb.WriteString(strings.Join(matches, "\n"))
	if len(matches) >= maxMatches {
		sb.WriteString(fmt.Sprintf("\n... (capped at %d results — refine query for more precise results)", maxMatches))
	}
	return sb.String(), nil
}

// FindScript searches the project Schema directories for a script by name (case-insensitive fuzzy match).
// Returns: sanitized step content, txt_path, xml_path, and sibling scripts in the same folder.
// Strategy C: .txt first (token-efficient), xml_path provided for when output generation needs it.
func FindScript(projectPath string, scriptName string, database string) (string, error) {
	if projectPath == "" {
		return "", fmt.Errorf("no active project directory set")
	}
	if scriptName == "" {
		return "", fmt.Errorf("script_name is required")
	}

	query := strings.ToLower(strings.TrimSpace(scriptName))

	// Build candidate schema roots — support both old and new layout
	var schemaRoots []string
	if database != "" {
		schemaRoots = []string{
			filepath.Join(projectPath, "files", "Schema", database),
			filepath.Join(projectPath, "Schema", database),
		}
	} else {
		// Auto-discover database dirs
		for _, base := range []string{filepath.Join(projectPath, "files", "Schema"), filepath.Join(projectPath, "Schema")} {
			entries, err := os.ReadDir(base)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					schemaRoots = append(schemaRoots, filepath.Join(base, e.Name()))
				}
			}
		}
	}

	type scriptMatch struct {
		xmlPath string
		txtPath string
		name    string
		folder  string // parent folder name if inside a script group, else ""
		score   int    // 0=exact,1=prefix,2=contains,3=content-fallback
	}

	var matches []scriptMatch

	for _, root := range schemaRoots {
		scriptsDir := filepath.Join(root, "scripts")
		sanitizedDir := filepath.Join(root, "scripts_sanitized")
		if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
			continue
		}

		// Walk scripts dir (one level deep including sub-folders for script groups)
		entries, err := os.ReadDir(scriptsDir)
		if err != nil {
			continue
		}

		var scanDir func(dir string, sanitizedBase string, groupName string)
		scanDir = func(dir string, sanitizedBase string, groupName string) {
			subEntries, err := os.ReadDir(dir)
			if err != nil {
				return
			}
			for _, e := range subEntries {
				fullPath := filepath.Join(dir, e.Name())
				if e.IsDir() {
					// Script group folder — recurse one level
					scanDir(fullPath, filepath.Join(sanitizedBase, e.Name()), e.Name())
					continue
				}
				if strings.ToLower(filepath.Ext(e.Name())) != ".xml" {
					continue
				}
				baseName := strings.ToLower(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
				var score int
				switch {
				case baseName == query:
					score = 0
				case strings.HasPrefix(baseName, query):
					score = 1
				case strings.Contains(baseName, query):
					score = 2
				default:
					continue
				}
				txtName := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())) + ".txt"
				txtPath := filepath.Join(sanitizedBase, txtName)
				matches = append(matches, scriptMatch{
					xmlPath: fullPath,
					txtPath: txtPath,
					name:    strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())),
					folder:  groupName,
					score:   score,
				})
			}
		}
		_ = entries
		scanDir(scriptsDir, sanitizedDir, "")
	}

	if len(matches) == 0 {
		return fmt.Sprintf("Script %q not found.\nSearched schema roots:\n%s\nTip: Use search_file with pattern=%q for broader search.",
			scriptName, strings.Join(schemaRoots, "\n"), scriptName), nil
	}

	// Find the best score across all matches
	bestScore := matches[0].score
	for _, m := range matches[1:] {
		if m.score < bestScore {
			bestScore = m.score
		}
	}

	// Collect all matches that share the top score
	var topMatches []scriptMatch
	for _, m := range matches {
		if m.score == bestScore {
			topMatches = append(topMatches, m)
		}
	}

	// If multiple candidates tie at the top score, surface all of them and ask for disambiguation
	if len(topMatches) > 1 {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("AMBIGUOUS: %d scripts named %q found across databases. Specify the 'database' parameter to disambiguate.\n\n", len(topMatches), scriptName))
		sb.WriteString("Candidates:\n")
		for _, m := range topMatches {
			dbDir := filepath.Base(filepath.Dir(filepath.Dir(m.xmlPath))) // Schema/<database>/scripts
			sb.WriteString(fmt.Sprintf("  - name: %s\n    database: %s\n    txt_path: %s\n    xml_path: %s\n\n",
				m.name, dbDir, m.txtPath, m.xmlPath))
		}
		sb.WriteString("Action required: call find_script again with the correct 'database' parameter, or ask the user which file they mean.")
		return sb.String(), nil
	}

	best := topMatches[0]

	// Read sanitized .txt content
	txtContent := ""
	if data, err := os.ReadFile(best.txtPath); err == nil {
		txtContent = strings.TrimSpace(string(data))
	} else {
		// Fallback: generate from XML
		if xmlData, err := os.ReadFile(best.xmlPath); err == nil {
			txtContent, _ = SanitizeFMXmlSnippet(string(xmlData))
		}
	}

	// Collect sibling scripts (same folder)
	var siblings []string
	siblingDir := filepath.Dir(best.xmlPath)
	sibEntries, _ := os.ReadDir(siblingDir)
	for _, sib := range sibEntries {
		if sib.IsDir() || strings.ToLower(filepath.Ext(sib.Name())) != ".xml" {
			continue
		}
		sibBaseName := strings.TrimSuffix(sib.Name(), filepath.Ext(sib.Name()))
		if sibBaseName == best.name {
			continue // skip the matched script itself
		}
		sibXml := filepath.Join(siblingDir, sib.Name())
		sibTxt := filepath.Join(filepath.Dir(best.txtPath), sibBaseName+".txt")
		siblings = append(siblings, fmt.Sprintf("  - %s\n    txt: %s\n    xml: %s", sibBaseName, sibTxt, sibXml))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("script: %s\n", best.name))
	if best.folder != "" {
		sb.WriteString(fmt.Sprintf("folder: %s\n", best.folder))
	}
	sb.WriteString(fmt.Sprintf("txt_path: %s\n", best.txtPath))
	sb.WriteString(fmt.Sprintf("xml_path: %s\n", best.xmlPath))
	if len(matches) > 1 {
		sb.WriteString(fmt.Sprintf("note: %d other candidate(s) with lower score found\n", len(matches)-len(topMatches)))
	}
	sb.WriteString("\n--- Steps (sanitized .txt) ---\n")
	if txtContent != "" {
		sb.WriteString(txtContent)
	} else {
		sb.WriteString("(empty — script may have no steps or sanitized file missing)")
	}
	if len(siblings) > 0 {
		sb.WriteString("\n\n--- Sibling Scripts (same folder) ---\n")
		sb.WriteString(strings.Join(siblings, "\n"))
	}
	sb.WriteString("\n\n--- Note ---\n")
	sb.WriteString("Read xml_path only when you need to generate output XML. Use txt_path content for analysis.")
	return sb.String(), nil
}
