package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

func TestAddWithoutNameReportsErrorAndImportsNothing(t *testing.T) {
	configPath := setupUseCommandTest(t, "existing", []string{"existing"})

	sourcePath := filepath.Join(t.TempDir(), "prod.yaml")
	src := domain.NewKubeConfig()
	src.Clusters = append(src.Clusters, domain.ClusterEntry{
		Name:    "prod-cluster",
		Cluster: domain.Cluster{Server: "https://prod.example.com"},
	})
	src.Users = append(src.Users, domain.UserEntry{Name: "prod-user"})
	src.Contexts = append(src.Contexts, domain.ContextEntry{
		Name:    "prod",
		Context: domain.Context{Cluster: "prod-cluster", User: "prod-user"},
	})
	if err := infrastructure.NewFileRepository().Save(sourcePath, src); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	oldName := addName
	addName = ""
	t.Cleanup(func() { addName = oldName })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	addCmd.Run(addCmd, []string{sourcePath})
	os.Stderr = oldStderr
	w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(out), ErrNameRequired.Error()) {
		t.Fatalf("stderr = %q, want it to contain %q", out, ErrNameRequired.Error())
	}

	got, err := infrastructure.NewFileRepository().Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Contexts) != 1 {
		t.Fatalf("contexts = %d, want 1 (nothing imported)", len(got.Contexts))
	}
}
