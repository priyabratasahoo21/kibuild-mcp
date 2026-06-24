package workflows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"kibuild/mcp/skills"
)

// BuildSystemPrompt constructs the system prompt dynamically from workspace and workflow parameters.
// When cliMode is "mcp", workflow and skill content is NOT injected — the agent pulls them on demand
// via list_workflows / get_workflow / load_skill MCP tools. For "contract" or API mode, skills and
// workflow are injected upfront as before.
func BuildSystemPrompt(
	wf *Workflow,
	baseRules string,
	projectPath string,
	activeSkills []skills.Skill,
	pluginConnected bool,
	folders map[string]string,
	cliMode string,
) string {
	var sb strings.Builder

	// 1. Base Agent Rules
	sb.WriteString(baseRules)
	sb.WriteString("\n\n")

	// 2. Active Context and Folder Structure
	sb.WriteString("## Active Workspace Context\n")

	schemaDir := folders["schema"]
	sourceDir := folders["source"]
	archivesDir := folders["archives"]
	outboxDir := folders["outbox"]
	docsDir := folders["docs"]

	sb.WriteString(fmt.Sprintf("- Folder Structure:\n"+
		"  - '%s/': Explosions (XML representations of scripts/layouts/etc).\n"+
		"  - '%s/': Source code and script files.\n"+
		"  - '%s/': Versioned backups and archives.\n"+
		"  - '%s/': Versioned outbox for generated artifacts.\n"+
		"  - '%s/': Documentation and plans.\n",
		schemaDir, sourceDir, archivesDir, outboxDir, docsDir,
	))

	if !pluginConnected {
		sb.WriteString(fmt.Sprintf("\n> [!WARNING]\n"+
			"> The FileMaker C++ plugin is OFFLINE.\n"+
			"> Do NOT call: execute_sql, run_script, read_layout, export_schema, get_active_context.\n"+
			"> Access files via file-based tools in '%s/' and '%s/' instead.\n",
			schemaDir, sourceDir,
		))
	}
	sb.WriteString("\n")

	// 3. Business Glossary Injection
	if projectPath != "" {
		glossaryPath := filepath.Join(projectPath, "context", "business_glossary.md")
		if data, err := os.ReadFile(glossaryPath); err == nil {
			sb.WriteString("## Business Glossary\n")
			sb.WriteString("Use the following business definitions and rules to guide your work:\n\n")
			sb.WriteString(string(data))
			sb.WriteString("\n\n")
		}
	}

	// In MCP mode: skip workflow/skill injection — agent calls list_workflows → get_workflow → load_skill
	if cliMode == "mcp" {
		sb.WriteString("## Discovery Mode\n")
		sb.WriteString("You are running in MCP mode. Workflows and skills are NOT pre-loaded.\n")
		sb.WriteString("Start by calling `list_workflows` to see available workflows, then call `get_workflow` to load the right procedure and `load_skill` to pull any specialist skill you need.\n\n")
		return sb.String()
	}

	// 4. Selected Workflow instructions (contract / API mode)
	if wf != nil {
		sb.WriteString(fmt.Sprintf("## Active Workflow: %s\n", wf.Name))
		sb.WriteString(fmt.Sprintf("Workflow Description: %s\n\n", wf.Description))
		sb.WriteString(wf.Content)
		sb.WriteString("\n\n")
	}

	// 5. Active Specialist Skills (contract / API mode)
	var skillBlocks []string
	for _, s := range activeSkills {
		if s.Enabled && s.Content != "" {
			skillBlocks = append(skillBlocks, fmt.Sprintf("### Skill: %s\n%s", s.Name, s.Content))
		}
	}
	if len(skillBlocks) > 0 {
		sb.WriteString("## Active Skill Protocols\n")
		sb.WriteString(strings.Join(skillBlocks, "\n\n"))
		sb.WriteString("\n")
	}

	return sb.String()
}
