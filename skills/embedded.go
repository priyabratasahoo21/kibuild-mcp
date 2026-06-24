package skills

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed default_skills/*.md proposed_skills/*.md
var defaultSkillsFS embed.FS

// InitDefaultSkills extracts the embedded default and proposed skills to target directory if not present
func InitDefaultSkills(skillsDir string) error {
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	// Sync from workspace if found
	for _, subDirName := range []string{"default_skills", "proposed_skills"} {
		if wsDir := findWorkspaceSkills(subDirName); wsDir != "" {
			if entries, err := os.ReadDir(wsDir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
						continue
					}
					srcPath := filepath.Join(wsDir, entry.Name())
					destPath := filepath.Join(skillsDir, entry.Name())
					if data, err := os.ReadFile(srcPath); err == nil {
						_ = os.WriteFile(destPath, data, 0644)
					}
				}
			}
		}
	}

	// Load and copy from embedded FS
	for _, subDirName := range []string{"default_skills", "proposed_skills"} {
		entries, err := fs.ReadDir(defaultSkillsFS, subDirName)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			destPath := filepath.Join(skillsDir, entry.Name())
			// Write the skill if it doesn't already exist
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				data, err := defaultSkillsFS.ReadFile(subDirName + "/" + entry.Name())
				if err != nil {
					return err
				}
				if err := os.WriteFile(destPath, data, 0644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func findWorkspaceSkills(subDirName string) string {
	// Try executable parent traversal
	if execPath, err := os.Executable(); err == nil {
		dir := filepath.Dir(execPath)
		for i := 0; i < 5; i++ {
			checkPath := filepath.Join(dir, "sidecar", "skills", subDirName)
			if info, err := os.Stat(checkPath); err == nil && info.IsDir() {
				return checkPath
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Try standard home documents path
	if home, err := os.UserHomeDir(); err == nil {
		checkPath := filepath.Join(home, "Documents", "KiBuild Plugin", "sidecar", "skills", subDirName)
		if info, err := os.Stat(checkPath); err == nil && info.IsDir() {
			return checkPath
		}
	}
	return ""
}

