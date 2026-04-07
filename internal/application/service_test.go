package application

import (
	"errors"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type mockRepository struct {
	config      *domain.KubeConfig
	loadErr     error
	exists      bool
	defaultPath string
}

func (m *mockRepository) Load(path string) (*domain.KubeConfig, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return m.config, nil
}

func (m *mockRepository) Save(path string, config *domain.KubeConfig) error {
	m.config = config
	return nil
}

func (m *mockRepository) Backup(path string) error {
	return nil
}

func (m *mockRepository) Exists(path string) bool {
	return m.exists
}

func (m *mockRepository) GetDefaultPath() string {
	return m.defaultPath
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *domain.KubeConfig
		loadErr        error
		wantErr        bool
		wantValid      bool
		wantIssueCount int
	}{
		{
			name: "valid config",
			config: &domain.KubeConfig{
				CurrentContext: "dev",
				Clusters: []domain.ClusterEntry{
					{Name: "dev-cluster", Cluster: domain.Cluster{Server: "https://example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "dev-user"},
				},
				Contexts: []domain.ContextEntry{
					{Name: "dev", Context: domain.Context{Cluster: "dev-cluster", User: "dev-user", Namespace: "default"}},
				},
			},
			wantValid: true,
		},
		{
			name: "invalid references and missing current context",
			config: &domain.KubeConfig{
				CurrentContext: "missing",
				Clusters: []domain.ClusterEntry{
					{Name: "dev-cluster", Cluster: domain.Cluster{Server: "https://example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "dev-user"},
				},
				Contexts: []domain.ContextEntry{
					{Name: "dev", Context: domain.Context{Cluster: "missing-cluster", User: "missing-user"}},
				},
			},
			wantValid:      false,
			wantIssueCount: 3,
		},
		{
			name: "duplicate and missing names",
			config: &domain.KubeConfig{
				Clusters: []domain.ClusterEntry{
					{Name: "", Cluster: domain.Cluster{}},
					{Name: "shared", Cluster: domain.Cluster{Server: "https://example.com"}},
					{Name: "shared", Cluster: domain.Cluster{Server: "https://example.org"}},
				},
				Users: []domain.UserEntry{
					{Name: ""},
					{Name: "shared"},
					{Name: "shared"},
				},
				Contexts: []domain.ContextEntry{
					{Name: "", Context: domain.Context{}},
					{Name: "shared", Context: domain.Context{Cluster: "shared", User: "shared"}},
					{Name: "shared", Context: domain.Context{Cluster: "shared", User: "shared"}},
				},
			},
			wantValid:      false,
			wantIssueCount: 9,
		},
		{
			name:    "load failure",
			loadErr: errors.New("boom"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(&mockRepository{config: tt.config, loadErr: tt.loadErr})

			report, err := service.ValidateConfig("/tmp/config")
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if report == nil {
				t.Fatal("ValidateConfig() report is nil")
			}

			if report.IsValid() != tt.wantValid {
				t.Fatalf("ValidateConfig() valid = %v, want %v", report.IsValid(), tt.wantValid)
			}

			if len(report.Issues) != tt.wantIssueCount {
				t.Fatalf("ValidateConfig() issue count = %d, want %d", len(report.Issues), tt.wantIssueCount)
			}
		})
	}
}
