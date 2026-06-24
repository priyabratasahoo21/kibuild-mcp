package workflows

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed default_workflows/*.md
var defaultWorkflowsFS embed.FS

type Workflow struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"desc"`
	AllowedTools    []string `json:"allowed_tools"`
	FallbackTools   []string `json:"fallback_tools"`
	RequiredOutputs []string `json:"required_outputs"`
	Content         string   `json:"content"`
}

// ParseWorkflow parses a markdown string, extracting frontmatter YAML and markdown body content.
func ParseWorkflow(rawContent string) (*Workflow, error) {
	wf := &Workflow{}
	trimmed := strings.TrimSpace(rawContent)
	trimmed = strings.ReplaceAll(trimmed, "\r\n", "\n")

	if !strings.HasPrefix(trimmed, "---\n") {
		wf.Content = rawContent
		return wf, nil
	}

	lines := strings.Split(trimmed, "\n")
	var frontmatterLines []string
	var contentLines []string
	inFrontmatter := false
	processedFrontmatter := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "---" {
			if !inFrontmatter && !processedFrontmatter {
				inFrontmatter = true
				continue
			} else if inFrontmatter {
				inFrontmatter = false
				processedFrontmatter = true
				continue
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else {
			if processedFrontmatter {
				contentLines = append(contentLines, line)
			} else if i > 0 {
				contentLines = append(contentLines, line)
			}
		}
	}

	wf.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	// Parse simple YAML frontmatter fields and list values
	var currentList *[]string
	for _, fLine := range frontmatterLines {
		trimmedFLine := strings.TrimSpace(fLine)
		if trimmedFLine == "" {
			continue
		}

		// Handle list items starting with "- "
		if strings.HasPrefix(trimmedFLine, "-") && currentList != nil {
			item := strings.TrimSpace(strings.TrimPrefix(trimmedFLine, "-"))
			if item != "" {
				*currentList = append(*currentList, item)
			}
			continue
		}

		// If a new key is found, parse it
		parts := strings.SplitN(fLine, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "id":
			wf.ID = val
			currentList = nil
		case "name":
			wf.Name = val
			currentList = nil
		case "description":
			wf.Description = val
			currentList = nil
		case "allowed_tools":
			currentList = &wf.AllowedTools
		case "fallback_tools":
			currentList = &wf.FallbackTools
		case "required_outputs":
			currentList = &wf.RequiredOutputs
		default:
			currentList = nil
		}
	}

	return wf, nil
}

// ListWorkflows loads and parses all embedded workflow templates under default_workflows/
func ListWorkflows() ([]Workflow, error) {
	var list []Workflow
	entries, err := fs.ReadDir(defaultWorkflowsFS, "default_workflows")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded default_workflows dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		if entry.Name() == "README.md" || entry.Name() == "base_agent_rules.md" {
			continue
		}

		data, err := defaultWorkflowsFS.ReadFile("default_workflows/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded file %s: %w", entry.Name(), err)
		}

		wf, err := ParseWorkflow(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to parse workflow %s: %w", entry.Name(), err)
		}

		if wf.ID == "" {
			wf.ID = strings.TrimSuffix(entry.Name(), ".md")
		}
		list = append(list, *wf)
	}

	return list, nil
}

// GetWorkflow loads a specific workflow template by its ID
func GetWorkflow(id string) (*Workflow, error) {
	filename := fmt.Sprintf("default_workflows/%s.md", id)
	data, err := defaultWorkflowsFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("workflow template %q not found: %w", id, err)
	}

	wf, err := ParseWorkflow(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow %s: %w", id, err)
	}
	if wf.ID == "" {
		wf.ID = id
	}
	return wf, nil
}

// GetBaseAgentRules loads the shared rules template
func GetBaseAgentRules() (string, error) {
	data, err := defaultWorkflowsFS.ReadFile("default_workflows/base_agent_rules.md")
	if err != nil {
		return "", fmt.Errorf("base agent rules not found: %w", err)
	}
	_, content := parseFrontmatterContent(string(data))
	return content, nil
}

// Helper to extract frontmatter content from base rules
func parseFrontmatterContent(content string) (map[string]string, string) {
	meta := make(map[string]string)
	trimmedContent := strings.TrimSpace(content)
	trimmedContent = strings.ReplaceAll(trimmedContent, "\r\n", "\n")
	
	if !strings.HasPrefix(trimmedContent, "---\n") {
		return meta, content
	}

	lines := strings.Split(trimmedContent, "\n")
	var yamlLines []string
	var contentLines []string
	inFrontmatter := false
	processedFrontmatter := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFrontmatter && !processedFrontmatter {
				inFrontmatter = true
				continue
			} else if inFrontmatter {
				inFrontmatter = false
				processedFrontmatter = true
				continue
			}
		}

		if inFrontmatter {
			yamlLines = append(yamlLines, line)
		} else {
			if processedFrontmatter {
				contentLines = append(contentLines, line)
			} else if i > 0 {
				contentLines = append(contentLines, line)
			}
		}
	}

	for _, yline := range yamlLines {
		parts := strings.SplitN(yline, ":", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			meta[k] = v
		}
	}

	return meta, strings.TrimSpace(strings.Join(contentLines, "\n"))
}
