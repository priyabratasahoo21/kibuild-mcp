// Package exploder converts FileMaker "Save a Copy as XML" output (the
// FMSaveAsXML dialect) into the exploded, one-file-per-object schema layout
// that the navigation and reference tools index.
//
// It accepts either form FileMaker produces:
//   - a single FMSaveAsXML file (split_catalogs="False") containing every
//     *Catalog under one <Structure><AddAction>, or
//   - a folder of split catalog files (split_catalogs="True"), one
//     <DB>_<Catalog>Catalog.xml per catalog.
//
// Both are the same dialect; the parsers stream and locate catalog elements by
// name regardless of nesting, so the single file is simply handed to every
// catalog parser while the split folder routes each file to its parser.
package exploder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sanitizer turns an fmxmlsnippet into human-readable script text. It is
// injected by the caller (the tools package owns SanitizeFMXmlSnippet) so this
// package stays dependency-free and avoids an import cycle. May be nil.
type Sanitizer func(snippet string) (string, error)

// Result reports what was written.
type Result struct {
	Database string   `json:"database"`
	Dest     string   `json:"dest"`
	Source   string   `json:"source"`
	Mode     string   `json:"mode"` // "single-file" or "split-catalogs"
	Scripts  int      `json:"scripts"`
	Warnings []string `json:"warnings,omitempty"`
}

// Explode reads the FileMaker XML export at source and writes the exploded
// schema under dest/Schema/<database>/. If database is empty it is inferred
// from the file/folder name. dest defaults to the source's parent when empty.
func Explode(source, database, dest string, sanitize Sanitizer) (*Result, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}
	isDir := info.IsDir()

	if database == "" {
		database = inferDatabase(source, isDir)
	}
	if dest == "" {
		if isDir {
			dest = filepath.Dir(source)
		} else {
			dest = filepath.Dir(source)
		}
	}

	res := &Result{Database: database, Source: source}
	if isDir {
		res.Mode = "split-catalogs"
	} else {
		res.Mode = "single-file"
	}

	schemaRoot := filepath.Join(dest, "Schema", database)
	res.Dest = schemaRoot

	scriptXML, err := readCatalog(source, isDir, "ScriptCatalog")
	if err != nil {
		return nil, err
	}
	n, warns, err := explodeScripts(scriptXML, schemaRoot, sanitize)
	if err != nil {
		return nil, fmt.Errorf("exploding scripts: %w", err)
	}
	res.Scripts = n
	res.Warnings = append(res.Warnings, warns...)

	return res, nil
}

// readCatalog returns the XML to parse for the given catalog. A single-file
// export returns the whole file (the parser finds the catalog element within);
// a split folder returns the matching <…>_<catalog>.xml.
func readCatalog(source string, isDir bool, catalog string) (string, error) {
	if !isDir {
		data, err := os.ReadFile(source)
		return string(data), err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// FileMaker names split files "<DB>_<Catalog>.xml" (e.g.
		// Contacts_ScriptCatalog.xml); also accept a bare "<Catalog>.xml".
		if strings.HasSuffix(name, "_"+catalog+".xml") || name == catalog+".xml" {
			data, err := os.ReadFile(filepath.Join(source, name))
			return string(data), err
		}
	}
	return "", fmt.Errorf("no %s file found in %s", catalog, source)
}

// inferDatabase derives the database name from the source path: the file base
// for a single file, or the folder name for a split-catalog directory.
func inferDatabase(source string, isDir bool) string {
	base := filepath.Base(source)
	if isDir {
		return base
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
