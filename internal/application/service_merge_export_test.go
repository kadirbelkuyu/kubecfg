package application

import (
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type mergeExportMockRepository struct {
	configs      map[string]*domain.KubeConfig
	savedConfigs map[string]*domain.KubeConfig
}

func (m *mergeExportMockRepository) Load(path string) (*domain.KubeConfig, error) {
	config, ok := m.configs[path]
	if !ok {
		return nil, domain.ErrConfigNotFound
	}
	return config, nil
}

func (m *mergeExportMockRepository) Save(path string, config *domain.KubeConfig) error {
	if m.savedConfigs == nil {
		m.savedConfigs = make(map[string]*domain.KubeConfig)
	}
	m.savedConfigs[path] = config
	m.configs[path] = config
	return nil
}

func (m *mergeExportMockRepository) Backup(path string) error {
	return nil
}

func (m *mergeExportMockRepository) Exists(path string) bool {
	_, ok := m.configs[path]
	return ok
}

func (m *mergeExportMockRepository) GetDefaultPath() string {
	return "/tmp/config"
}

func TestMergeConfigsRenameStrategy(t *testing.T) {
	repo := &mergeExportMockRepository{
		configs: map[string]*domain.KubeConfig{
			"/tmp/one": {
				CurrentContext: "prod",
				Clusters: []domain.ClusterEntry{
					{Name: "cluster", Cluster: domain.Cluster{Server: "https://one.example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "user", User: domain.User{Token: "one"}},
				},
				Contexts: []domain.ContextEntry{
					{Name: "prod", Context: domain.Context{Cluster: "cluster", User: "user", Namespace: "payments"}},
				},
			},
			"/tmp/two": {
				CurrentContext: "prod",
				Clusters: []domain.ClusterEntry{
					{Name: "cluster", Cluster: domain.Cluster{Server: "https://two.example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "user", User: domain.User{Token: "two"}},
				},
				Contexts: []domain.ContextEntry{
					{Name: "prod", Context: domain.Context{Cluster: "cluster", User: "user", Namespace: "billing"}},
				},
			},
		},
	}
	service := NewService(repo)

	if err := service.MergeConfigs([]string{"/tmp/one", "/tmp/two"}, "/tmp/out", "rename"); err != nil {
		t.Fatalf("MergeConfigs() error = %v", err)
	}

	merged := repo.savedConfigs["/tmp/out"]
	if merged == nil {
		t.Fatal("MergeConfigs() did not save output")
	}

	if len(merged.Contexts) != 2 || len(merged.Clusters) != 2 || len(merged.Users) != 2 {
		t.Fatalf("MergeConfigs() counts = contexts:%d clusters:%d users:%d", len(merged.Contexts), len(merged.Clusters), len(merged.Users))
	}

	if merged.Contexts[1].Name != "prod-2" {
		t.Fatalf("MergeConfigs() renamed context = %q, want %q", merged.Contexts[1].Name, "prod-2")
	}

	if merged.Contexts[1].Context.Cluster != "cluster-2" {
		t.Fatalf("MergeConfigs() renamed cluster ref = %q, want %q", merged.Contexts[1].Context.Cluster, "cluster-2")
	}

	if merged.Contexts[1].Context.User != "user-2" {
		t.Fatalf("MergeConfigs() renamed user ref = %q, want %q", merged.Contexts[1].Context.User, "user-2")
	}
}

func TestMergeConfigsFailStrategy(t *testing.T) {
	repo := &mergeExportMockRepository{
		configs: map[string]*domain.KubeConfig{
			"/tmp/one": {
				Clusters: []domain.ClusterEntry{
					{Name: "cluster", Cluster: domain.Cluster{Server: "https://one.example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "user", User: domain.User{Token: "one"}},
				},
				Contexts: []domain.ContextEntry{
					{Name: "prod", Context: domain.Context{Cluster: "cluster", User: "user"}},
				},
			},
			"/tmp/two": {
				Clusters: []domain.ClusterEntry{
					{Name: "cluster", Cluster: domain.Cluster{Server: "https://two.example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "user", User: domain.User{Token: "two"}},
				},
				Contexts: []domain.ContextEntry{
					{Name: "prod", Context: domain.Context{Cluster: "cluster", User: "user"}},
				},
			},
		},
	}
	service := NewService(repo)

	if err := service.MergeConfigs([]string{"/tmp/one", "/tmp/two"}, "/tmp/out", "fail"); err == nil {
		t.Fatal("MergeConfigs() error = nil, want conflict error")
	}
}

func TestExportContextExportsCurrentContext(t *testing.T) {
	repo := &mergeExportMockRepository{
		configs: map[string]*domain.KubeConfig{
			"/tmp/config": {
				CurrentContext: "prod",
				Clusters: []domain.ClusterEntry{
					{Name: "prod-cluster", Cluster: domain.Cluster{Server: "https://prod.example.com"}},
				},
				Users: []domain.UserEntry{
					{Name: "prod-user", User: domain.User{Token: "secret"}},
				},
				Contexts: []domain.ContextEntry{
					{Name: "prod", Context: domain.Context{Cluster: "prod-cluster", User: "prod-user", Namespace: "payments"}},
				},
			},
		},
	}
	service := NewService(repo)

	if err := service.ExportContext("/tmp/config", "", "/tmp/exported"); err != nil {
		t.Fatalf("ExportContext() error = %v", err)
	}

	exported := repo.savedConfigs["/tmp/exported"]
	if exported == nil {
		t.Fatal("ExportContext() did not save output")
	}

	if exported.CurrentContext != "prod" {
		t.Fatalf("ExportContext() current context = %q, want %q", exported.CurrentContext, "prod")
	}

	if len(exported.Contexts) != 1 || exported.Contexts[0].Name != "prod" {
		t.Fatalf("ExportContext() contexts = %+v", exported.Contexts)
	}

	if len(exported.Clusters) != 1 || exported.Clusters[0].Name != "prod-cluster" {
		t.Fatalf("ExportContext() clusters = %+v", exported.Clusters)
	}

	if len(exported.Users) != 1 || exported.Users[0].Name != "prod-user" {
		t.Fatalf("ExportContext() users = %+v", exported.Users)
	}
}
