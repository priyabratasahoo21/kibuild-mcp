package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"kibuild/mcp/skills"
	"kibuild/mcp/workflows"
)

var (
	globalSkillsDir   string
	skillsDirMu       sync.RWMutex
)

// SetSkillsDir registers the skills directory so list_workflows/load_skill tools can use it.
func SetSkillsDir(dir string) {
	skillsDirMu.Lock()
	defer skillsDirMu.Unlock()
	globalSkillsDir = dir
}

func getSkillsDir() string {
	skillsDirMu.RLock()
	defer skillsDirMu.RUnlock()
	return globalSkillsDir
}

// ListWorkflowsTool returns a JSON summary of all available workflows (id + name + description).
func ListWorkflowsTool() (string, error) {
	list, err := workflows.ListWorkflows()
	if err != nil {
		return "", fmt.Errorf("failed to list workflows: %w", err)
	}

	type workflowSummary struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	var summaries []workflowSummary
	for _, wf := range list {
		summaries = append(summaries, workflowSummary{
			ID:          wf.ID,
			Name:        wf.Name,
			Description: wf.Description,
		})
	}

	b, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize workflows: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Available workflows (call get_workflow with the id to load full instructions, or load_skill to load a specialist skill):\n\n")
	sb.Write(b)
	return sb.String(), nil
}

// LoadSkillTool returns the full markdown content of a skill by ID.
func LoadSkillTool(skillID string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skill_id is required")
	}

	dir := getSkillsDir()
	if dir == "" {
		return "", fmt.Errorf("skills directory not configured")
	}

	allSkills, err := skills.ListSkills(dir)
	if err != nil {
		return "", fmt.Errorf("failed to list skills: %w", err)
	}

	for _, s := range allSkills {
		if s.ID == skillID {
			if s.Content == "" {
				return "", fmt.Errorf("skill %q exists but has no content", skillID)
			}
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# Skill: %s\n\n", s.Name))
			sb.WriteString(s.Content)
			return sb.String(), nil
		}
	}

	// Build list of available IDs for a helpful error
	var available []string
	for _, s := range allSkills {
		available = append(available, s.ID)
	}
	return "", fmt.Errorf("skill %q not found. Available skills: %s", skillID, strings.Join(available, ", "))
}

// GetWorkflowTool returns the full markdown content of a workflow by ID.
func GetWorkflowTool(workflowID string) (string, error) {
	if workflowID == "" {
		return "", fmt.Errorf("workflow_id is required")
	}

	wf, err := workflows.GetWorkflow(workflowID)
	if err != nil {
		return "", fmt.Errorf("workflow %q not found: %w", workflowID, err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Workflow: %s\n\n", wf.Name))
	if wf.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", wf.Description))
	}
	sb.WriteString(wf.Content)
	return sb.String(), nil
}
