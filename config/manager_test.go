package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "settings.json")
	configJSON := `{
		"model": "claude-3-5-sonnet",
		"maxTokens": 2048,
		"apiKey": "test-api-key"
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.GetModel() != "claude-3-5-sonnet" {
		t.Errorf("expected model to be claude-3-5-sonnet, got %s", mgr.GetModel())
	}

	if mgr.GetMaxTokens() != 2048 {
		t.Errorf("expected maxTokens to be 2048, got %d", mgr.GetMaxTokens())
	}

	if mgr.GetAPIKey("anthropic") != "test-api-key" {
		t.Errorf("expected anthropic key to be test-api-key, got %s", mgr.GetAPIKey("anthropic"))
	}

	if mgr.GetActiveProvider() != "anthropic" {
		t.Errorf("expected active provider to be anthropic, got %s", mgr.GetActiveProvider())
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	os.Setenv("KIBUILD_MODEL", "gemini-1.5-pro")
	os.Setenv("GEMINI_API_KEY", "env-gemini-key")
	defer func() {
		os.Unsetenv("KIBUILD_MODEL")
		os.Unsetenv("GEMINI_API_KEY")
	}()

	mgr, err := NewManager("nonexistent.json")
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr.GetModel() != "gemini-1.5-pro" {
		t.Errorf("expected model to be gemini-1.5-pro, got %s", mgr.GetModel())
	}

	if mgr.GetAPIKey("gemini") != "env-gemini-key" {
		t.Errorf("expected gemini key to be env-gemini-key, got %s", mgr.GetAPIKey("gemini"))
	}

	if mgr.GetActiveProvider() != "gemini" {
		t.Errorf("expected active provider to be gemini, got %s", mgr.GetActiveProvider())
	}
}

func TestConfigDisabledMCPTools(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kibuild_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "settings.json")
	configJSON := `{
		"model": "gemini-3.5-flash",
		"maxTokens": 4096,
		"disabled_mcp_tools": ["write_outbox_artifact", "generate_schema_map"]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	mgr, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	cfg := mgr.GetConfig()
	if len(cfg.DisabledMCPTools) != 2 {
		t.Fatalf("expected 2 disabled tools, got %d", len(cfg.DisabledMCPTools))
	}

	if cfg.DisabledMCPTools[0] != "write_outbox_artifact" || cfg.DisabledMCPTools[1] != "generate_schema_map" {
		t.Errorf("unexpected disabled mcp tools slice contents: %v", cfg.DisabledMCPTools)
	}

	safeCfg := mgr.GetSafeConfig()
	if len(safeCfg.DisabledMCPTools) != 2 {
		t.Fatalf("expected 2 disabled tools in SafeConfig, got %d", len(safeCfg.DisabledMCPTools))
	}
}

