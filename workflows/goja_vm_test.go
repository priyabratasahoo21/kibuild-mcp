package workflows

import (
	"os"
	"testing"
	"time"
)

func TestRunScript(t *testing.T) {
	// 1. Simple math test
	val, err := RunScript("2 + 3", VMConfig{Timeout: 1 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.Export().(int64) != 5 {
		t.Errorf("expected 5, got %v", val.Export())
	}

	// 2. Timeout test
	_, err = RunScript("while(true) {}", VMConfig{Timeout: 50 * time.Millisecond})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	// 3. Filesystem test
	tmpDir, err := os.MkdirTemp("", "goja_vm_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := VMConfig{
		ProjectPath: tmpDir,
		Timeout:     2 * time.Second,
	}

	// Test write and read
	_, err = RunScript("fs.write('test.txt', 'hello from goja');", config)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	val, err = RunScript("fs.read('test.txt');", config)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if val.String() != "hello from goja" {
		t.Errorf("expected 'hello from goja', got %q", val.String())
	}

	// Test access violation
	_, err = RunScript("fs.read('../test.txt');", config)
	if err == nil {
		t.Error("expected access denied error, got nil")
	}
}
