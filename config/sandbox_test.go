package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProjectPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home: %v", err)
	}
	resolvedHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		resolvedHome = home
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path inside home",
			path:    filepath.Join(resolvedHome, "Documents", "KiBuild Projects"),
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "invalid escape path",
			path:    "/etc",
			wantErr: true,
		},
		{
			name:    "traversal path inside home but escaping parent",
			path:    filepath.Join(resolvedHome, "..", "..", "etc"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateProjectPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
