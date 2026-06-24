package exploder

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	snippetHeader = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<fmxmlsnippet type=\"FMObjectList\">\n"
	snippetFooter = "\n</fmxmlsnippet>"
)

// explodeScripts streams the <StepsForScripts> section of a ScriptCatalog and
// writes one file per script. Each script's <ObjectList> steps are wrapped in
// an <fmxmlsnippet type="FMObjectList"> — the format the sanitizer and
// reference tools already parse — and saved to scripts/<name>.xml, with a
// matching scripts_sanitized/<name>.txt when a sanitizer is supplied.
//
// Folder and marker entries (isFolder) live only in the catalog listing and
// carry no steps, so they never appear here. Scripts with no steps are skipped.
func explodeScripts(catalogXML, schemaRoot string, sanitize Sanitizer) (int, []string, error) {
	scriptsDir := filepath.Join(schemaRoot, "scripts")
	sanitizedDir := filepath.Join(schemaRoot, "scripts_sanitized")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return 0, nil, err
	}
	if sanitize != nil {
		if err := os.MkdirAll(sanitizedDir, 0o755); err != nil {
			return 0, nil, err
		}
	}

	dec := xml.NewDecoder(strings.NewReader(catalogXML))

	// Element names (Script, ScriptReference, ObjectList) also occur nested
	// inside step parameters, so we cannot match by name alone — we track depth
	// and act only on the elements at the structural levels that matter:
	//   StepsForScripts (stepsDepth)
	//     └ Script        (scriptDepth = stepsDepth+1)   ← one per script
	//         ├ ScriptReference (identity, not emitted)
	//         └ ObjectList (listDepth = scriptDepth+1)    ← the step list; its
	//              └ Step … (everything below listDepth is emitted verbatim)
	var (
		depth      int
		stepsDepth = -1
		scriptDepth = -1
		listDepth  = -1
		scriptName string
		scriptID   string
		buf        strings.Builder
		enc        *xml.Encoder
		seen       = map[string]int{}
		count      int
		warnings   []string
	)

	writeScript := func() error {
		if scriptName == "" {
			return nil
		}
		snippet := snippetHeader + buf.String() + snippetFooter
		fileName := uniqueName(seen, sanitizeFileName(scriptName), scriptID)
		if err := os.WriteFile(filepath.Join(scriptsDir, fileName+".xml"), []byte(snippet), 0o644); err != nil {
			return err
		}
		// An empty script (no steps) is valid; the sanitizer rejects an empty
		// snippet, so write an empty .txt rather than treating it as a failure.
		hasSteps := strings.Contains(buf.String(), "<Step")
		if sanitize != nil {
			if !hasSteps {
				_ = os.WriteFile(filepath.Join(sanitizedDir, fileName+".txt"), nil, 0o644)
			} else if txt, err := sanitize(snippet); err == nil {
				_ = os.WriteFile(filepath.Join(sanitizedDir, fileName+".txt"), []byte(txt), 0o644)
			} else {
				warnings = append(warnings, fmt.Sprintf("sanitize failed for %q: %v", scriptName, err))
			}
		}
		count++
		return nil
	}

	for {
		tok, err := dec.Token()
		if err != nil {
			break // io.EOF or malformed tail; whatever was written stands
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			name := t.Name.Local
			switch {
			case name == "StepsForScripts" && stepsDepth == -1:
				stepsDepth = depth
			case stepsDepth != -1 && depth == stepsDepth+1 && name == "Script":
				// One script in the steps section begins.
				scriptDepth = depth
				listDepth = -1
				scriptName, scriptID = "", ""
				buf.Reset()
				enc = xml.NewEncoder(&buf)
			case scriptDepth != -1 && depth == scriptDepth+1 && name == "ScriptReference":
				// Script identity; not part of the snippet body.
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "name":
						scriptName = a.Value
					case "id":
						scriptID = a.Value
					}
				}
			case scriptDepth != -1 && depth == scriptDepth+1 && name == "ObjectList":
				// The step list begins; unwrap it (emit <Step> children directly).
				listDepth = depth
			case listDepth != -1 && depth > listDepth:
				// Anything inside the step list — including nested Script,
				// ScriptReference, ObjectList in step parameters — is verbatim body.
				_ = enc.EncodeToken(t)
			}
		case xml.EndElement:
			name := t.Name.Local
			switch {
			case listDepth != -1 && depth > listDepth:
				_ = enc.EncodeToken(t)
			case listDepth != -1 && depth == listDepth && name == "ObjectList":
				_ = enc.Flush()
				listDepth = -1
			case scriptDepth != -1 && depth == scriptDepth && name == "Script":
				if err := writeScript(); err != nil {
					return count, warnings, err
				}
				scriptDepth = -1
			case stepsDepth != -1 && depth == stepsDepth && name == "StepsForScripts":
				stepsDepth = -1
			}
			depth--
		case xml.CharData:
			if listDepth != -1 && depth >= listDepth {
				_ = enc.EncodeToken(xml.CharData(append([]byte(nil), t...)))
			}
		}
	}
	return count, warnings, nil
}

// uniqueName disambiguates colliding script names (e.g. several "New Script")
// by appending the script id on the second and later sightings, keeping the
// first occurrence's name clean and the result stable across runs.
func uniqueName(seen map[string]int, safeName, id string) string {
	seen[safeName]++
	if seen[safeName] == 1 {
		return safeName
	}
	if id != "" {
		return fmt.Sprintf("%s (%s)", safeName, id)
	}
	return fmt.Sprintf("%s (%d)", safeName, seen[safeName])
}

// sanitizeFileName strips path separators and control characters so a script
// name is safe as a filename. Matches the generator's "/" → "-" convention.
func sanitizeFileName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\n", " ", "\r", " ")
	return strings.TrimSpace(r.Replace(name))
}
