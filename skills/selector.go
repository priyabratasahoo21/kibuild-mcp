package skills

import (
	"strings"
)

var workflowSkills = map[string][]string{
	"create_script":          {"fm_xml_serializer", "fm_xml_validator", "pro_scriptwriter"},
	"refactor_script":        {"dependency_analyst", "fm_xml_serializer", "fm_xml_validator", "pro_scriptwriter", "script_analysis"},
	"analyze_script":         {"dependency_analyst", "context_router", "script_analysis"},
	"add_to_script":          {"fm_xml_serializer", "fm_xml_validator", "pro_scriptwriter", "script_analysis"},
	"test_script":            {"script_test_designer", "fm_xml_validator", "filemaker_data_access_guard"},
	"implementation_plan":    {"implementation_planner", "dependency_analyst", "context_router"},
	"solution_blueprint":     {"solution_architect", "context_router", "implementation_planner"},
	"verify_build":           {"build_verifier", "context_router", "dependency_analyst"},
	"create_layout":          {"fm_xml_serializer", "fm_xml_validator", "pro_scriptwriter"},
	"schema_audit":           {"dependency_analyst", "context_router", "fm_xml_validator"},
	"document_solution":      {"implementation_planner", "context_router"},
	"analyze_layout":         {"dependency_analyst", "context_router", "script_analysis"},
	"analyze_table":          {"dependency_analyst", "context_router"},
	"analyze_relationship":   {"dependency_analyst", "context_router"},
	"analyze_valuelist":      {"dependency_analyst", "context_router"},
	"explore_schema_object":  {"dependency_analyst", "context_router", "script_analysis"},
}

// GetSkillsForWorkflow returns the skill IDs required for a given workflow ID.
func GetSkillsForWorkflow(workflowID string) []string {
	if ids, ok := workflowSkills[workflowID]; ok {
		return ids
	}
	// Fallback: only fm_xml_serializer is always-on
	return []string{"fm_xml_serializer"}
}

// objectTypeCtx detects which FileMaker schema object type the query is likely referring to.
// Returns one of: "script", "layout", "table", "relationship", "valuelist", or "" (unknown).
func objectTypeCtx(q string) string {
	if strings.Contains(q, "script") || strings.Contains(q, "step") ||
		strings.Contains(q, "subscript") || strings.Contains(q, "perform script") {
		return "script"
	}
	if strings.Contains(q, "value list") || strings.Contains(q, "valuelist") ||
		strings.Contains(q, "picklist") || strings.Contains(q, "drop-down") ||
		strings.Contains(q, "dropdown") {
		return "valuelist"
	}
	if strings.Contains(q, "relationship") || strings.Contains(q, "join") ||
		strings.Contains(q, "predicate") || strings.Contains(q, "foreign key") ||
		strings.Contains(q, "related table") {
		return "relationship"
	}
	if strings.Contains(q, "table") || strings.Contains(q, "field") ||
		strings.Contains(q, "entity") || strings.Contains(q, "column") {
		return "table"
	}
	if strings.Contains(q, "layout") || strings.Contains(q, "screen") ||
		strings.Contains(q, "form") || strings.Contains(q, "portal") {
		return "layout"
	}
	return ""
}

// isImprovementIntent returns true when the query expresses a desire to improve,
// fix, or modify something — regardless of which object type is mentioned.
func isImprovementIntent(q string) bool {
	return strings.Contains(q, "improve") || strings.Contains(q, "refactor") ||
		strings.Contains(q, "rewrite") || strings.Contains(q, "optimize") ||
		strings.Contains(q, "fix") || strings.Contains(q, "update") ||
		strings.Contains(q, "change") || strings.Contains(q, "modify") ||
		strings.Contains(q, "clean up") || strings.Contains(q, "clean-up") ||
		strings.Contains(q, "enhance") || strings.Contains(q, "tighten") ||
		strings.Contains(q, "better") || strings.Contains(q, "restructure") ||
		strings.Contains(q, "simplify") || strings.Contains(q, "repair")
}

// isAnalysisIntent returns true when the query asks to analyze, review, inspect, or explain.
func isAnalysisIntent(q string) bool {
	return strings.Contains(q, "analyze") || strings.Contains(q, "analyse") ||
		strings.Contains(q, "inspect") || strings.Contains(q, "review") ||
		strings.Contains(q, "explain") || strings.Contains(q, "check") ||
		strings.Contains(q, "audit") || strings.Contains(q, "show me") ||
		strings.Contains(q, "what does") || strings.Contains(q, "how does") ||
		strings.Contains(q, "walk me through") || strings.Contains(q, "describe")
}

// isAdditionIntent returns true when the query wants to add something to an existing object.
func isAdditionIntent(q string) bool {
	return strings.Contains(q, "add") || strings.Contains(q, "append") ||
		strings.Contains(q, "insert") || strings.Contains(q, "extend") ||
		strings.Contains(q, "include")
}

// isCreationIntent returns true when the query wants to create something new.
func isCreationIntent(q string) bool {
	hasWrite := strings.Contains(q, "write") && !strings.Contains(q, "rewrite")
	return strings.Contains(q, "create") || strings.Contains(q, "new ") ||
		strings.Contains(q, "generate") || hasWrite ||
		strings.Contains(q, "build") || strings.Contains(q, "make")
}

// ResolveWorkflowFromQuery uses heuristics to determine the most likely workflow ID from the query text.
// priorWorkflowID is the workflow resolved for the previous mission in the same conversation.
// If the current query is a short follow-up (≤ 5 words) with no contradicting keywords,
// and the prior workflow is a sticky one, it is inherited unchanged.
// IMPORTANT: This logic is the single source of truth — workflows/selector.go delegates here.
func ResolveWorkflowFromQuery(query string, priorWorkflowID string) string {
	q := strings.ToLower(query)

	// ── Inherit prior workflow for short follow-up messages ──────────────────
	// Covers: "Error handling", "Also add logging", "Add parameter validation", etc.
	stickyWorkflows := map[string]bool{
		"refactor_script":      true,
		"add_to_script":        true,
		"analyze_layout":       true,
		"analyze_table":        true,
		"analyze_relationship": true,
		"analyze_valuelist":    true,
		"explore_schema_object": true,
	}
	if priorWorkflowID != "" && stickyWorkflows[priorWorkflowID] && len(strings.Fields(q)) <= 5 {
		hasStrongKeyword := strings.Contains(q, "create") || strings.Contains(q, "new ") ||
			strings.Contains(q, "generate") || strings.Contains(q, "write") ||
			strings.Contains(q, "test") || strings.Contains(q, "plan") ||
			strings.Contains(q, "blueprint") || strings.Contains(q, "document")
		if !hasStrongKeyword {
			return priorWorkflowID
		}
	}

	objType := objectTypeCtx(q)

	// ── Highest-priority fixed routes (build/test/plan) ───────────────────────
	switch {
	case strings.Contains(q, "verify build") || strings.Contains(q, "validate build") ||
		strings.Contains(q, "check build"):
		return "verify_build"

	case strings.Contains(q, "test") || strings.Contains(q, "spec") ||
		strings.Contains(q, "assertion"):
		return "test_script"

	case strings.Contains(q, "plan") || strings.Contains(q, "roadmap") ||
		strings.Contains(q, "implementation"):
		return "implementation_plan"

	case strings.Contains(q, "architecture") || strings.Contains(q, "blueprint") ||
		strings.Contains(q, "design"):
		return "solution_blueprint"

	case strings.Contains(q, "schema") && (strings.Contains(q, "audit") ||
		strings.Contains(q, "validate")):
		return "schema_audit"

	case strings.Contains(q, "document") || strings.Contains(q, "doc") ||
		strings.Contains(q, "writeup"):
		return "document_solution"
	}

	// ── Layout creation ───────────────────────────────────────────────────────
	if objType == "layout" && isCreationIntent(q) {
		return "create_layout"
	}

	// ── Script creation ───────────────────────────────────────────────────────
	if objType == "script" && isCreationIntent(q) {
		return "create_script"
	}

	// ── Generic creation (no object type) → create_script ────────────────────
	if objType == "" && isCreationIntent(q) {
		return "create_script"
	}

	// ── Improvement / modification intent — route by object type ─────────────
	if isImprovementIntent(q) {
		switch objType {
		case "script":
			return "refactor_script"
		case "layout":
			return "analyze_layout"
		case "table":
			return "analyze_table"
		case "relationship":
			return "analyze_relationship"
		case "valuelist":
			return "analyze_valuelist"
		default:
			// Object type not mentioned — agent will ask for clarification
			return "explore_schema_object"
		}
	}

	// ── Addition intent — route by object type ────────────────────────────────
	if isAdditionIntent(q) {
		switch objType {
		case "script":
			return "add_to_script"
		case "layout":
			return "analyze_layout"
		case "table":
			return "analyze_table"
		case "relationship":
			return "analyze_relationship"
		case "valuelist":
			return "analyze_valuelist"
		default:
			return "explore_schema_object"
		}
	}

	// ── Analysis / explain intent — route by object type ──────────────────────
	if isAnalysisIntent(q) {
		switch objType {
		case "script":
			return "analyze_script"
		case "layout":
			return "analyze_layout"
		case "table":
			return "analyze_table"
		case "relationship":
			return "analyze_relationship"
		case "valuelist":
			return "analyze_valuelist"
		default:
			return "explore_schema_object"
		}
	}

	// ── No clear intent but object type mentioned → explore ───────────────────
	if objType != "" && objType != "script" {
		return "explore_schema_object"
	}

	return "create_script" // Default fallback
}

// GetSkillsForQuery returns the skill IDs based on query heuristics.
func GetSkillsForQuery(query string) []string {
	wf := ResolveWorkflowFromQuery(query, "")
	return GetSkillsForWorkflow(wf)
}
