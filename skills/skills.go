package skills

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	Color       string `json:"color"`
	Letter      string `json:"letter"`
	Enabled     bool   `json:"enabled"`
	Content     string `json:"content"`
}

// LoadSkillsConfig loads the enabled/disabled states of skills from skillsDir/config.json
func LoadSkillsConfig(skillsDir string) (map[string]bool, error) {
	cfgPath := filepath.Join(skillsDir, "config.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, err
	}
	var config map[string]bool
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// SaveSkillsConfig saves the enabled/disabled states to skillsDir/config.json
func SaveSkillsConfig(skillsDir string, config map[string]bool) error {
	cfgPath := filepath.Join(skillsDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}

// ParseFrontmatter extracts YAML frontmatter and body from file contents
func ParseFrontmatter(content string) (map[string]string, string) {
	meta := make(map[string]string)
	trimmedContent := strings.TrimSpace(content)
	
	// Normalize line endings
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
			} else if i > 0 && !inFrontmatter {
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

	return meta, strings.Join(contentLines, "\n")
}

// FormatFrontmatter formats a Skill struct back into a markdown string with frontmatter
func FormatFrontmatter(s Skill) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", s.ID))
	sb.WriteString(fmt.Sprintf("name: %s\n", s.Name))
	sb.WriteString(fmt.Sprintf("desc: %s\n", s.Description))
	sb.WriteString(fmt.Sprintf("color: %s\n", s.Color))
	sb.WriteString(fmt.Sprintf("letter: %s\n", s.Letter))
	sb.WriteString("---\n")
	sb.WriteString(s.Content)
	return sb.String()
}

// ListSkills lists all skills in skillsDir, parsing frontmatter and merge enabled state
func ListSkills(skillsDir string) ([]Skill, error) {
	config, err := LoadSkillsConfig(skillsDir)
	if err != nil {
		return nil, err
	}

	var list []Skill
	err = filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != skillsDir {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		meta, body := ParseFrontmatter(string(data))
		id := meta["id"]
		if id == "" {
			id = strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		}

		enabled := true
		if val, exists := config[id]; exists {
			enabled = val
		}

		list = append(list, Skill{
			ID:          id,
			Name:        meta["name"],
			Description: meta["desc"],
			Color:       meta["color"],
			Letter:      meta["letter"],
			Enabled:     enabled,
			Content:     strings.TrimSpace(body),
		})
		return nil
	})

	return list, err
}
