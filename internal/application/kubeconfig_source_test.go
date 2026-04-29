package application

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

func TestListKubeconfigSourcesDiscoversValidFiles(t *testing.T) {
	dir := t.TempDir()
	activePath := filepath.Join(dir, "prod.yaml")
	otherPath := filepath.Join(dir, "dev.yaml")
	invalidPath := filepath.Join(dir, "notes.txt")

	repo := newSourceTestRepo()
	writeSourceConfig(t, repo, activePath, "prod")
	writeSourceConfig(t, repo, otherPath, "dev")
	if err := os.WriteFile(invalidPath, []byte("not a kubeconfig"), 0600); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo)
	sources := service.ListKubeconfigSources(activePath, []string{dir})

	if len(sources) != 2 {
		t.Fatalf("sources len = %d, want 2: %#v", len(sources), sources)
	}
	if !sources[0].Active || sources[0].Path != activePath {
		t.Fatalf("first source = %#v, want active %q", sources[0], activePath)
	}
	if sources[0].ContextCount != 1 || sources[1].ContextCount != 1 {
		t.Fatalf("context counts = %#v, want one context in each source", sources)
	}
}

func TestValidateKubeconfigSourceRejectsEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	repo := newSourceTestRepo()
	if err := repo.Save(path, domain.NewKubeConfig()); err != nil {
		t.Fatal(err)
	}

	service := NewService(repo)
	if err := service.ValidateKubeconfigSource(path); err == nil {
		t.Fatal("ValidateKubeconfigSource() error = nil, want error")
	}
}

func writeSourceConfig(t *testing.T, repo *sourceTestRepo, path, contextName string) {
	t.Helper()

	cfg := domain.NewKubeConfig()
	cfg.CurrentContext = contextName
	cfg.Clusters = append(cfg.Clusters, domain.ClusterEntry{
		Name: contextName,
		Cluster: domain.Cluster{
			Server: "https://" + contextName + ".example.test",
		},
	})
	cfg.Users = append(cfg.Users, domain.UserEntry{Name: contextName + "-user"})
	cfg.Contexts = append(cfg.Contexts, domain.ContextEntry{
		Name: contextName,
		Context: domain.Context{
			Cluster: contextName,
			User:    contextName + "-user",
		},
	})

	if err := repo.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
}

type sourceTestRepo struct {
	configs map[string]*domain.KubeConfig
}

func newSourceTestRepo() *sourceTestRepo {
	return &sourceTestRepo{configs: make(map[string]*domain.KubeConfig)}
}

func (r *sourceTestRepo) Load(path string) (*domain.KubeConfig, error) {
	cfg, ok := r.configs[path]
	if !ok {
		return nil, domain.ErrInvalidConfig
	}
	return cfg, nil
}

func (r *sourceTestRepo) Save(path string, cfg *domain.KubeConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte("apiVersion: v1\nkind: Config\n"), 0600); err != nil {
		return err
	}
	r.configs[path] = cfg
	return nil
}

func (r *sourceTestRepo) Exists(path string) bool {
	_, ok := r.configs[path]
	return ok
}

func (r *sourceTestRepo) GetDefaultPath() string {
	return ""
}

func (r *sourceTestRepo) Backup(path string) error {
	return nil
}

func (r *sourceTestRepo) ListBackups(path string) ([]string, error) {
	return nil, nil
}

func (r *sourceTestRepo) RestoreBackup(targetPath, backupPath string) error {
	return fmt.Errorf("not implemented")
}
