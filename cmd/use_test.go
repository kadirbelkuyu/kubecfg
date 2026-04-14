package cmd

import (
	"path/filepath"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

func TestRunUseWithPreviousContext(t *testing.T) {
	tests := []struct {
		name              string
		initialCurrent    string
		previousContext   string
		contexts          []string
		steps             []string
		wantCurrentAfter  []string
		wantErrorAtStep   int
		wantErrorContains string
	}{
		{
			name:             "kubecfg use - switches to previous context",
			initialCurrent:   "staging",
			previousContext:  "production",
			contexts:         []string{"production", "staging"},
			steps:            []string{"-"},
			wantCurrentAfter: []string{"production"},
		},
		{
			name:              "kubecfg use - returns error when no previous context exists",
			initialCurrent:    "staging",
			contexts:          []string{"production", "staging"},
			steps:             []string{"-"},
			wantErrorAtStep:   1,
			wantErrorContains: "no previous context recorded",
		},
		{
			name:             "switching context A to B to dash lands on A",
			initialCurrent:   "A",
			contexts:         []string{"A", "B"},
			steps:            []string{"B", "-"},
			wantCurrentAfter: []string{"B", "A"},
		},
		{
			name:             "switching context A to B to C to dash to dash lands on B then C",
			initialCurrent:   "A",
			contexts:         []string{"A", "B", "C"},
			steps:            []string{"B", "C", "-", "-"},
			wantCurrentAfter: []string{"B", "C", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := setupUseCommandTest(t, tt.initialCurrent, tt.contexts)
			if tt.previousContext != "" {
				if err := infrastructure.WritePrevious(tt.previousContext); err != nil {
					t.Fatalf("WritePrevious() error = %v", err)
				}
			}

			for index, step := range tt.steps {
				err := runUse(useCmd, []string{step})
				if tt.wantErrorAtStep == index+1 {
					if err == nil {
						t.Fatalf("runUse() error = nil, want error containing %q", tt.wantErrorContains)
					}
					if err.Error() != tt.wantErrorContains {
						t.Fatalf("runUse() error = %q, want %q", err.Error(), tt.wantErrorContains)
					}
					return
				}

				if err != nil {
					t.Fatalf("runUse() error = %v", err)
				}

				got := readCurrentContext(t, configPath)
				if got != tt.wantCurrentAfter[index] {
					t.Fatalf("current context after step %d = %q, want %q", index+1, got, tt.wantCurrentAfter[index])
				}
			}
		})
	}
}

func setupUseCommandTest(t *testing.T, current string, names []string) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(t.TempDir(), "config")
	cfg := domain.NewKubeConfig()
	cfg.CurrentContext = current

	for _, name := range names {
		clusterName := name + "-cluster"
		userName := name + "-user"
		cfg.Clusters = append(cfg.Clusters, domain.ClusterEntry{
			Name:    clusterName,
			Cluster: domain.Cluster{Server: "https://" + name + ".example.com"},
		})
		cfg.Users = append(cfg.Users, domain.UserEntry{Name: userName})
		cfg.Contexts = append(cfg.Contexts, domain.ContextEntry{
			Name: name,
			Context: domain.Context{
				Cluster: clusterName,
				User:    userName,
			},
		})
	}

	repo := infrastructure.NewFileRepository()
	if err := repo.Save(configPath, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	kubeconfigPath = configPath
	service = application.NewService(
		repo,
		application.WithPreviousContextStore(infrastructure.NewPreviousContextStore(filepath.Join(home, ".kubecfg", "last-context"))),
	)
	namespaceFlag = ""
	useCmd.Flags().Lookup("namespace").Changed = false

	return configPath
}

func readCurrentContext(t *testing.T, path string) string {
	t.Helper()

	repo := infrastructure.NewFileRepository()
	cfg, err := repo.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	return cfg.CurrentContext
}
