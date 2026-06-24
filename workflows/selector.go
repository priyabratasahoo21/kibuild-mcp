package workflows

import (
	"kibuild/mcp/skills"
)

// SelectWorkflow resolves the active workflow ID based on direct selection or query heuristics.
// priorWorkflowID is the workflow from the previous turn; passed to the heuristic for short follow-ups.
func SelectWorkflow(workflowID string, query string, priorWorkflowID string) string {
	if workflowID != "" && workflowID != "auto" && workflowID != "default" {
		return workflowID
	}
	return skills.ResolveWorkflowFromQuery(query, priorWorkflowID)
}

// ResolveWorkflowFromQuery is a convenience alias that delegates to the skills package.
// Kept for backward compatibility with any external callers.
func ResolveWorkflowFromQuery(query string) string {
	return skills.ResolveWorkflowFromQuery(query, "")
}
