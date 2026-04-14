package fzf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAvailable(t *testing.T) {
	tests := []struct {
		name        string
		ignoreValue string
		createFZF   bool
		want        bool
	}{
		{
			name:        "returns false when ignore flag is set",
			ignoreValue: "1",
			createFZF:   true,
			want:        false,
		},
		{
			name:      "returns false when fzf is not in path",
			createFZF: false,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("PATH", tempDir)
			t.Setenv("KUBECFG_IGNORE_FZF", tt.ignoreValue)

			if tt.createFZF {
				path := filepath.Join(tempDir, "fzf")
				if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			}

			if got := Available(); got != tt.want {
				t.Fatalf("Available() = %v, want %v", got, tt.want)
			}
		})
	}
}
