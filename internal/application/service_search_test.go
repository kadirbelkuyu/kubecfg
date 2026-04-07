package application

import (
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type searchMockRepository struct {
	config *domain.KubeConfig
}

func (m *searchMockRepository) Load(path string) (*domain.KubeConfig, error) {
	return m.config, nil
}

func (m *searchMockRepository) Save(path string, config *domain.KubeConfig) error {
	m.config = config
	return nil
}

func (m *searchMockRepository) Backup(path string) error {
	return nil
}

func (m *searchMockRepository) ListBackups(path string) ([]string, error) {
	return nil, nil
}

func (m *searchMockRepository) RestoreBackup(targetPath, backupPath string) error {
	return nil
}

func (m *searchMockRepository) Exists(path string) bool {
	return true
}

func (m *searchMockRepository) GetDefaultPath() string {
	return "/tmp/config"
}

func TestSearchContextsMatchesNamespace(t *testing.T) {
	service := NewService(&searchMockRepository{
		config: &domain.KubeConfig{
			CurrentContext: "prod",
			Clusters: []domain.ClusterEntry{
				{Name: "prod-cluster", Cluster: domain.Cluster{Server: "https://prod.example.com"}},
				{Name: "dev-cluster", Cluster: domain.Cluster{Server: "https://dev.example.com"}},
			},
			Users: []domain.UserEntry{
				{Name: "prod-user"},
				{Name: "dev-user"},
			},
			Contexts: []domain.ContextEntry{
				{Name: "prod", Context: domain.Context{Cluster: "prod-cluster", User: "prod-user", Namespace: "payments"}},
				{Name: "dev", Context: domain.Context{Cluster: "dev-cluster", User: "dev-user", Namespace: "default"}},
			},
		},
	})

	results, err := service.SearchContexts("/tmp/config", "payments")
	if err != nil {
		t.Fatalf("SearchContexts() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("SearchContexts() result count = %d, want 1", len(results))
	}

	if results[0].Name != "prod" {
		t.Fatalf("SearchContexts() result name = %q, want %q", results[0].Name, "prod")
	}
}
