package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateProjectPath cleans and checks if a project path is within safe boundaries.
// It resolves symlinks, cleans relative segments, and ensures the target directory
// is located under the user's home directory or configured workspace roots.
func ValidateProjectPath(projectPath string) (string, error) {
	if projectPath == "" {
		return "", fmt.Errorf("project path cannot be empty")
	}

	// Clean the path
	cleaned := filepath.Clean(projectPath)

	// Resolve symlinks to prevent bypasses
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// If path doesn't exist, we clean it as absolute and check boundaries
		var absErr error
		resolved, absErr = filepath.Abs(cleaned)
		if absErr != nil {
			return "", fmt.Errorf("failed to make absolute path: %v", absErr)
		}
	}

	// Fetch user's home directory as the general security root boundary
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		resolvedHome = home
	}

	// Check if path is within resolvedHome
	rel, err := filepath.Rel(resolvedHome, resolved)
	if err != nil {
		return "", fmt.Errorf("invalid path relationship: %v", err)
	}

	if strings.HasPrefix(rel, "..") || rel == "." {
		// Verify if it resides in another allowed directory (e.g. system temporary folder used for sandbox/test)
		tempDir := os.TempDir()
		resolvedTemp, errTemp := filepath.EvalSymlinks(tempDir)
		if errTemp == nil {
			relTemp, errRelTemp := filepath.Rel(resolvedTemp, resolved)
			if errRelTemp == nil && !strings.HasPrefix(relTemp, "..") && relTemp != "." {
				return resolved, nil
			}
		}
		return "", fmt.Errorf("path %q escapes safe home directory sandbox %q", resolved, resolvedHome)
	}

	return resolved, nil
}
