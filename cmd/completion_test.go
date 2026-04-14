package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

func TestCompletionCommandGeneratesOutput(t *testing.T) {
	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			completionCmd.SetOut(&buf)

			if err := completionCmd.RunE(completionCmd, []string{tt.shell}); err != nil {
				t.Fatalf("RunE() error = %v", err)
			}

			if buf.Len() == 0 {
				t.Fatalf("completion output for %s is empty", tt.shell)
			}
		})
	}
}

func TestUseValidArgsFunctionReturnsContextNames(t *testing.T) {
	setupCompletionTestConfig(t, []string{"prod", "staging"})

	got, directive := useCmd.ValidArgsFunction(useCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v, want %v", directive, cobra.ShellCompDirectiveNoFileComp)
	}

	for _, want := range []string{"prod", "staging"} {
		if !containsCompletion(got, want) {
			t.Fatalf("completion results %v do not contain %q", got, want)
		}
	}
}

func TestUseValidArgsFunctionIncludesPreviousContextShortcut(t *testing.T) {
	setupCompletionTestConfig(t, []string{"prod", "staging"})
	if err := infrastructure.WritePrevious("prod"); err != nil {
		t.Fatalf("WritePrevious() error = %v", err)
	}

	got, _ := useCmd.ValidArgsFunction(useCmd, nil, "")
	if !containsCompletion(got, "-\tswitch to previous context") {
		t.Fatalf("completion results %v do not contain previous context shortcut", got)
	}
}

func setupCompletionTestConfig(t *testing.T, names []string) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := domain.NewKubeConfig()
	if len(names) > 0 {
		cfg.CurrentContext = names[0]
	}

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

	path := filepath.Join(t.TempDir(), "config")
	repo := infrastructure.NewFileRepository()
	if err := repo.Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	kubeconfigPath = path
}

func containsCompletion(items []string, want string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == want {
			return true
		}
	}

	return false
}
