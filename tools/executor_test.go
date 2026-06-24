package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kibuild/mcp/providers"
)

func TestExecutorFilesystemTools(t *testing.T) {
	// 1. Setup temp folder and files
	tempDir, err := os.MkdirTemp("", "kibuild_executor_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.WithValue(context.Background(), providers.ProjectPathKey, tempDir)

	filePath := filepath.Join(tempDir, "test.txt")

	// 2. Test write_file tool
	writeArgs := map[string]string{
		"path":    filePath,
		"content": "Hello FileMaker Sidecar Agent!",
	}
	writeArgsJSON, _ := json.Marshal(writeArgs)
	writeRes, err := ExecuteTool(ctx, "write_file", string(writeArgsJSON))
	if err != nil {
		t.Fatalf("write_file tool failed: %v", err)
	}
	if !strings.Contains(writeRes, "Successfully wrote") {
		t.Errorf("unexpected write result: %s", writeRes)
	}

	// 3. Test read_file tool
	readArgs := map[string]string{
		"path": filePath,
	}
	readArgsJSON, _ := json.Marshal(readArgs)
	readRes, err := ExecuteTool(ctx, "read_file", string(readArgsJSON))
	if err != nil {
		t.Fatalf("read_file tool failed: %v", err)
	}
	if readRes != "Hello FileMaker Sidecar Agent!" {
		t.Errorf("expected file content to be 'Hello FileMaker Sidecar Agent!', got %s", readRes)
	}

	// 4. Test list_dir tool
	listArgs := map[string]string{
		"path": tempDir,
	}
	listArgsJSON, _ := json.Marshal(listArgs)
	listRes, err := ExecuteTool(ctx, "list_dir", string(listArgsJSON))
	if err != nil {
		t.Fatalf("list_dir tool failed: %v", err)
	}

	var items []FileItem
	if err := json.Unmarshal([]byte(listRes), &items); err != nil {
		t.Fatalf("failed to parse list_dir JSON output: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 file in temp dir, got %d", len(items))
	} else if items[0].Name != "test.txt" {
		t.Errorf("expected file name to be test.txt, got %s", items[0].Name)
	}
}

func TestSandboxPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_sandbox_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Resolve relative path
	relPath := "src/App.tsx"
	resolved, err := ResolveAndSandboxPath(tempDir, relPath)
	if err != nil {
		t.Fatalf("expected relative path to resolve successfully: %v", err)
	}
	expected := filepath.Join(tempDir, "src/App.tsx")
	if resolved != expected {
		t.Errorf("expected resolved path to be %s, got %s", expected, resolved)
	}

	// 2. Reject path traversal escaping sandbox
	escapedPath := "../outside.txt"
	_, err = ResolveAndSandboxPath(tempDir, escapedPath)
	if err == nil {
		t.Error("expected traversal path escaping workspace to fail")
	}

	// 3. Test absolute path inside sandbox
	insideAbs := filepath.Join(tempDir, "docs/index.md")
	resolvedInside, err := ResolveAndSandboxPath(tempDir, insideAbs)
	if err != nil {
		t.Fatalf("expected inside absolute path to succeed: %v", err)
	}
	if resolvedInside != insideAbs {
		t.Errorf("expected %s, got %s", insideAbs, resolvedInside)
	}

	// 4. Test absolute path outside sandbox
	outsideAbs := "/etc/passwd"
	_, err = ResolveAndSandboxPath(tempDir, outsideAbs)
	if err == nil {
		t.Error("expected absolute path outside workspace to fail")
	}
}
