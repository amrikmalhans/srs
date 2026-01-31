package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name          string
		userSpecified string
		wantErr       bool
	}{
		{
			name:          "user specified path",
			userSpecified: "/tmp/test-config",
			wantErr:       false,
		},
		{
			name:          "empty path uses defaults",
			userSpecified: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveConfigPath(tt.userSpecified)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Error("ResolveConfigPath() returned empty path")
			}
		})
	}
}

func TestDatabasePath(t *testing.T) {
	configDir := "/tmp/test-config"
	want := filepath.Join(configDir, "srs.db")
	got := DatabasePath(configDir)
	if got != want {
		t.Errorf("DatabasePath() = %v, want %v", got, want)
	}
}

func TestCardsPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	got, err := CardsPath()
	if err != nil {
		t.Errorf("CardsPath() error = %v", err)
		return
	}

	// Verify cards directory was created
	if _, err := os.Stat(got); os.IsNotExist(err) {
		t.Errorf("CardsPath() did not create cards directory at %v", got)
	}

	// Verify it's actually a directory
	info, err := os.Stat(got)
	if err != nil {
		t.Errorf("CardsPath() returned invalid path: %v", err)
		return
	}
	if !info.IsDir() {
		t.Errorf("CardsPath() did not create a directory")
	}
}
