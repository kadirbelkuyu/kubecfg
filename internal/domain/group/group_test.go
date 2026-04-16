package group

import (
	"errors"
	"testing"
)

func TestGroupValidate(t *testing.T) {
	tests := []struct {
		name    string
		group   Group
		wantErr error
	}{
		{
			name:    "rejects empty name",
			group:   Group{Contexts: []string{"prod"}},
			wantErr: ErrInvalidGroupName,
		},
		{
			name:    "rejects uppercase names",
			group:   Group{Name: "Prod", Contexts: []string{"prod"}},
			wantErr: ErrInvalidGroupName,
		},
		{
			name:    "rejects underscores",
			group:   Group{Name: "prod_team", Contexts: []string{"prod"}},
			wantErr: ErrInvalidGroupName,
		},
		{
			name:    "rejects empty context list",
			group:   Group{Name: "prod"},
			wantErr: ErrEmptyContextList,
		},
		{
			name:  "accepts prod",
			group: Group{Name: "prod", Contexts: []string{"cluster-a"}},
		},
		{
			name:  "accepts eu-1",
			group: Group{Name: "eu-1", Contexts: []string{"cluster-a"}},
		},
		{
			name:  "accepts a",
			group: Group{Name: "a", Contexts: []string{"cluster-a"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.group.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
