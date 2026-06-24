package tools

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// fmStep is an intermediate representation of a parsed FM script step
type fmStep struct {
	ID          string
	Name        string
	Enable      string
	Calculation string // primary Calculation CDATA
	SecondCalc  string // second Calculation (for Show Custom Dialog title etc.)
	VarName     string // <Name> child text (Set Variable)
	VarValue    string // <Value><Calculation> text (Set Variable / Exit Script)
	FieldTable  string // <Field table="...">
	FieldName   string // <Field name="...">
	ScriptName  string // <Script name="...">
	LayoutName  string // <LayoutReference name="...">
	StateValue  string // <State state="..."> or <NoInteract state="...">
	CommentText string // <Text> CDATA (Comment step)
	MailTo      string // Send Mail <To><Calculation>
	MailSubject string // Send Mail <Subject><Calculation>
	ChildXML    []fmChild
}

type fmChild struct {
	Local  string
	Attrs  map[string]string
	CData  string
}

// SanitizeFMXmlSnippet parses a fmxmlsnippet and returns human-readable sanitized text.
// Format follows FileMaker Script Workspace display conventions.
func SanitizeFMXmlSnippet(xmlContent string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	var lines []string
	var currentStep *fmStep
	var depth int
	var inStep bool
	var elementStack []string
	var cdataTarget *string // pointer to the field we're collecting CDATA into
	var currentCalcTarget *string
	var calcCount int // track how many Calculation elements we've seen in this step
	var indentLevel int

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			local := t.Name.Local
			elementStack = append(elementStack, local)

			attrs := map[string]string{}
			for _, a := range t.Attr {
				attrs[a.Name.Local] = a.Value
			}

			if local == "Step" && depth == 2 {
				// New step at top level inside fmxmlsnippet
				currentStep = &fmStep{
					ID:     attrs["id"],
					Name:   attrs["name"],
					Enable: attrs["enable"],
				}
				inStep = true
				calcCount = 0
				cdataTarget = nil
				// Initialise to primary calc field so CharData is never lost
				// even if a Calculation element is not encountered first.
				currentCalcTarget = &currentStep.Calculation
				continue
			}

			if !inStep || currentStep == nil {
				continue
			}

			// Count current Calculation depth
			calcDepth := 0
			for _, s := range elementStack {
				if s == "Calculation" {
					calcDepth++
				}
			}

			switch local {
			case "Calculation":
				// Send Mail has many sibling named-wrapper calcs (To/Subject/Message/
				// SMTPServer/…). Routing them by calcCount would concatenate them into
				// SecondCalc, so capture To/Subject by their wrapper element instead.
				if currentStep.Name == "Send Mail" && calcDepth == 1 {
					switch parentElement(elementStack) {
					case "To":
						cdataTarget = &currentStep.MailTo
					case "Subject":
						cdataTarget = &currentStep.MailSubject
					default:
						cdataTarget = nil
					}
					break
				}
				if calcDepth == 1 {
					calcCount++
					if calcCount == 1 {
						currentCalcTarget = &currentStep.Calculation
					} else if calcCount == 2 {
						currentCalcTarget = &currentStep.SecondCalc
					}
					// calcCount > 2: keep currentCalcTarget on SecondCalc —
					// additional calcs are rare edge cases and accumulate there.
				}
				cdataTarget = currentCalcTarget

			case "Name":
				if calcDepth == 0 {
					if val, ok := attrs["value"]; ok && val != "" {
						currentStep.VarName = val
					}
					cdataTarget = &currentStep.VarName
				} else {
					cdataTarget = currentCalcTarget
				}

			case "Text":
				if calcDepth == 0 {
					cdataTarget = &currentStep.CommentText
				} else {
					cdataTarget = currentCalcTarget
				}

			case "Value":
				if calcDepth > 0 {
					cdataTarget = currentCalcTarget
				}

			case "Field":
				currentStep.FieldTable = attrs["table"]
				currentStep.FieldName = attrs["name"]
				cdataTarget = nil

			case "FieldReference":
				if n, ok := attrs["name"]; ok && n != "" {
					currentStep.FieldName = n
				}
				if t, ok := attrs["table"]; ok && t != "" {
					currentStep.FieldTable = t
				}
				cdataTarget = nil

			case "TableOccurrenceReference":
				if n, ok := attrs["name"]; ok && n != "" {
					currentStep.FieldTable = n
				}
				cdataTarget = nil

			case "Script":
				if n, ok := attrs["name"]; ok && n != "" {
					currentStep.ScriptName = n
				}
				cdataTarget = nil

			case "ScriptReference":
				if n, ok := attrs["name"]; ok && n != "" {
					currentStep.ScriptName = n
				}
				cdataTarget = nil

			case "LayoutReference":
				if n, ok := attrs["name"]; ok && n != "" {
					currentStep.LayoutName = n
				}
				cdataTarget = nil

			case "State", "NoInteract", "Set":
				if s, ok := attrs["state"]; ok {
					currentStep.StateValue = s
				}
				cdataTarget = nil

			default:
				if calcDepth > 0 {
					cdataTarget = currentCalcTarget
				} else {
					cdataTarget = nil
				}
			}

		case xml.CharData:
			if cdataTarget != nil && inStep {
				*cdataTarget += string(t)
			}

		case xml.EndElement:
			local := t.Name.Local
			if len(elementStack) > 0 {
				elementStack = elementStack[:len(elementStack)-1]
			}

			// Compute calcDepth after elementStack pop
			calcDepth := 0
			for _, s := range elementStack {
				if s == "Calculation" {
					calcDepth++
				}
			}

			// Reset CDATA target when leaving a Calculation/Name/Text element
			switch local {
			case "Calculation":
				if calcDepth == 0 {
					cdataTarget = nil
					currentCalcTarget = nil
				} else {
					cdataTarget = currentCalcTarget
				}
			case "Name", "Text":
				if calcDepth == 0 {
					cdataTarget = nil
				} else {
					cdataTarget = currentCalcTarget
				}
			}

			if local == "Step" && inStep && currentStep != nil && depth == 2 {
				currentStep.Calculation = strings.TrimSpace(currentStep.Calculation)
				currentStep.SecondCalc = strings.TrimSpace(currentStep.SecondCalc)
				currentStep.VarName = strings.TrimSpace(currentStep.VarName)
				currentStep.VarValue = strings.TrimSpace(currentStep.VarValue)
				currentStep.CommentText = strings.TrimSpace(currentStep.CommentText)
				currentStep.MailTo = strings.TrimSpace(currentStep.MailTo)
				currentStep.MailSubject = strings.TrimSpace(currentStep.MailSubject)

				line := formatStep(currentStep)
				if line != "" {
					stepName := currentStep.Name
					// Decrease indent BEFORE dedent keywords
					switch stepName {
					case "End If", "End Loop", "Else", "Else If":
						if indentLevel > 0 {
							indentLevel--
						}
					}
					lines = append(lines, strings.Repeat("  ", indentLevel)+line)
					// Increase indent AFTER indent keywords
					switch stepName {
					case "If", "Loop", "Else", "Else If":
						indentLevel++
					}
				}
				currentStep = nil
				inStep = false
				cdataTarget = nil
				currentCalcTarget = nil
			}
			depth--
		}
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no script steps found in fmxmlsnippet")
	}
	return strings.Join(lines, "\n"), nil
}

func parentElement(stack []string) string {
	if len(stack) >= 2 {
		return stack[len(stack)-2]
	}
	return ""
}

// formatStep converts a parsed fmStep into a readable FileMaker-style line
func formatStep(s *fmStep) string {
	name := s.Name
	calc := s.Calculation
	second := s.SecondCalc

	switch name {
	// ── Control ──────────────────────────────────────────────────────────────
	case "If":
		return fmt.Sprintf("If [ %s ]", calc)
	case "Else If":
		return fmt.Sprintf("Else If [ %s ]", calc)
	case "Else":
		return "Else"
	case "End If":
		return "End If"
	case "Loop":
		return "Loop"
	case "Exit Loop If":
		return fmt.Sprintf("Exit Loop If [ %s ]", calc)
	case "End Loop":
		return "End Loop"

	// ── Variables & Fields ────────────────────────────────────────────────────
	case "Set Variable":
		varName := s.VarName
		varVal := calc
		if varVal == "" {
			varVal = second
		}
		return fmt.Sprintf("Set Variable [ %s ; Value: %s ]", varName, varVal)

	case "Set Field":
		target := formatFieldRef(s.FieldTable, s.FieldName)
		return fmt.Sprintf("Set Field [ %s ; %s ]", target, calc)

	case "Set Field By Name":
		return fmt.Sprintf("Set Field By Name [ %s ; %s ]", calc, second)

	// ── Records ───────────────────────────────────────────────────────────────
	case "Commit Records/Requests":
		noInteract := "With dialog"
		if strings.EqualFold(s.StateValue, "true") {
			noInteract = "No dialog"
		}
		return fmt.Sprintf("Commit Records/Requests [ %s ]", noInteract)

	case "Revert Record/Request":
		return "Revert Record/Request [ No dialog ]"

	case "New Record/Request":
		return "New Record/Request"

	case "Delete Record/Request":
		return "Delete Record/Request [ No dialog ]"

	case "Duplicate Record/Request":
		return "Duplicate Record/Request"

	case "Show All Records":
		return "Show All Records"

	case "Sort Records":
		return "Sort Records [ Restore ; No dialog ]"

	case "Unsort Records":
		return "Unsort Records"

	// ── Navigation ────────────────────────────────────────────────────────────
	case "Go to Layout":
		if s.LayoutName != "" {
			return fmt.Sprintf("Go to Layout [ \"%s\" ]", s.LayoutName)
		}
		return fmt.Sprintf("Go to Layout [ %s ]", calc)

	case "Go to Record/Request/Page":
		return fmt.Sprintf("Go to Record/Request/Page [ %s ]", calc)

	case "Go to Field":
		target := formatFieldRef(s.FieldTable, s.FieldName)
		if target != "" {
			return fmt.Sprintf("Go to Field [ %s ]", target)
		}
		return "Go to Field"

	case "Go to Portal Row":
		return fmt.Sprintf("Go to Portal Row [ %s ]", calc)

	case "Go to Object":
		return fmt.Sprintf("Go to Object [ %s ]", calc)

	// ── Scripts ───────────────────────────────────────────────────────────────
	case "Perform Script":
		scriptRef := s.ScriptName
		param := calc
		if scriptRef != "" && param != "" {
			return fmt.Sprintf("Perform Script [ \"%s\" ; Parameter: %s ]", scriptRef, param)
		} else if scriptRef != "" {
			return fmt.Sprintf("Perform Script [ \"%s\" ]", scriptRef)
		}
		return "Perform Script"

	case "Perform Script on Server":
		scriptRef := s.ScriptName
		if scriptRef != "" {
			return fmt.Sprintf("Perform Script on Server [ \"%s\" ]", scriptRef)
		}
		return "Perform Script on Server"

	case "Exit Script":
		return fmt.Sprintf("Exit Script [ Text Result: %s ]", calc)

	case "Halt Script":
		return "Halt Script"

	// ── Dialogs ───────────────────────────────────────────────────────────────
	case "Show Custom Dialog":
		if second != "" {
			return fmt.Sprintf("Show Custom Dialog [ Title: %s ; Message: %s ]", second, calc)
		}
		return fmt.Sprintf("Show Custom Dialog [ %s ]", calc)

	// ── Error Handling ────────────────────────────────────────────────────────
	case "Set Error Capture":
		state := "Off"
		if strings.EqualFold(s.StateValue, "true") {
			state = "On"
		}
		return fmt.Sprintf("Set Error Capture [ %s ]", state)

	// ── Transactions ──────────────────────────────────────────────────────────
	case "Open Transaction":
		return "Open Transaction"
	case "Commit Transaction":
		return "Commit Transaction"
	case "Revert Transaction":
		return "Revert Transaction"

	// ── Data ─────────────────────────────────────────────────────────────────
	case "Import Records":
		return "Import Records [ No dialog ]"

	case "Export Records":
		return "Export Records [ No dialog ]"

	case "Execute SQL":
		return fmt.Sprintf("Execute SQL [ %s ]", calc)

	case "Insert from URL":
		return fmt.Sprintf("Insert from URL [ %s ]", calc)

	case "Send Mail":
		dialog := "With dialog"
		if strings.EqualFold(s.StateValue, "true") {
			dialog = "No dialog"
		}
		var parts []string
		if s.MailTo != "" {
			parts = append(parts, "To: "+s.MailTo)
		}
		if s.MailSubject != "" {
			parts = append(parts, "Subject: "+s.MailSubject)
		}
		parts = append(parts, dialog)
		return fmt.Sprintf("Send Mail [ %s ]", strings.Join(parts, " ; "))

	// ── Comment ───────────────────────────────────────────────────────────────
	case "Comment", "# (comment)":
		text := s.CommentText
		if text == "" {
			text = calc
		}
		return fmt.Sprintf("# %s", text)

	// ── Allow User Abort ──────────────────────────────────────────────────────
	case "Allow User Abort":
		state := "Off"
		if strings.EqualFold(s.StateValue, "true") {
			state = "On"
		}
		return fmt.Sprintf("Allow User Abort [ %s ]", state)

	// ── Refresh ───────────────────────────────────────────────────────────────
	case "Refresh Window":
		return "Refresh Window"

	case "Refresh Object":
		return fmt.Sprintf("Refresh Object [ %s ]", calc)

	// ── Fallback — unknown step ───────────────────────────────────────────────
	default:
		if calc != "" {
			return fmt.Sprintf("%s [ %s ]", name, calc)
		}
		return name
	}
}

func formatFieldRef(table, field string) string {
	if table != "" && field != "" {
		return fmt.Sprintf("%s::%s", table, field)
	} else if field != "" {
		return field
	}
	return ""
}
