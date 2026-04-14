package fzf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelect(t *testing.T) {
	tests := []struct {
		name       string
		scriptBody string
		want       string
		wantErr    error
	}{
		{
			name: "returns ErrAborted when fzf exits 130",
			scriptBody: `#!/bin/sh
exit 130
`,
			wantErr: ErrAborted,
		},
		{
			name: "returns chosen item when fzf exits 0",
			scriptBody: `#!/bin/sh
cat >/dev/null
printf 'staging\n'
`,
			want: "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("PATH", tempDir)

			path := filepath.Join(tempDir, "fzf")
			if err := os.WriteFile(path, []byte(tt.scriptBody), 0755); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			got, err := Select([]string{"prod", "staging"}, Options{Prompt: "context> "})
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("Select() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Select() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Select() = %q, want %q", got, tt.want)
			}
		})
	}
}
