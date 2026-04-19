package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	appgroupservice "github.com/kadirbelkuyu/kubecfg/internal/application/groupservice"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/groupstore"
)

func TestRunGroupCreateFailsClearlyWhenContextDoesNotExist(t *testing.T) {
	setupGroupCommandTest(t, nil)

	err := runGroupCreate("prod", []string{"missing"}, "", "")
	if err == nil {
		t.Fatal("runGroupCreate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "context \"missing\" is not in your kubeconfig") {
		t.Fatalf("runGroupCreate() error = %q, want missing-context message", err.Error())
	}
	if !strings.Contains(err.Error(), "kubecfg list") {
		t.Fatalf("runGroupCreate() error = %q, want cleanup guidance", err.Error())
	}
}

func TestRunGroupUseSwitchesToResolvedContext(t *testing.T) {
	configPath := setupGroupCommandTest(t, []domaingroup.Group{{Name: "prod", Contexts: []string{"production"}}})

	err := runGroupUse("prod")
	if err != nil {
		t.Fatalf("runGroupUse() error = %v", err)
	}

	if got := readCurrentContext(t, configPath); got != "production" {
		t.Fatalf("current context after runGroupUse() = %q, want %q", got, "production")
	}
}

func TestRenderGroupTableIncludesColumns(t *testing.T) {
	output := renderGroupTable([]domaingroup.Group{{Name: "prod", Description: "All production clusters", Contexts: []string{"eks-prod", "gke-prod"}}}, false)

	for _, want := range []string{"NAME", "CONTEXTS", "DESCRIPTION", "prod", "All production clusters"} {
		if !strings.Contains(output, want) {
			t.Fatalf("renderGroupTable() output does not contain %q:\n%s", want, output)
		}
	}
}

func TestRunGroupDeleteRequiresForce(t *testing.T) {
	setupGroupCommandTest(t, []domaingroup.Group{{Name: "prod", Contexts: []string{"production"}}})

	err := runGroupDelete("prod", false)
	if err == nil {
		t.Fatal("runGroupDelete() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "requires --force") {
		t.Fatalf("runGroupDelete() error = %q, want force guidance", err.Error())
	}
}

func TestRunGroupRemoveRefusesToEmptyGroup(t *testing.T) {
	setupGroupCommandTest(t, []domaingroup.Group{{Name: "prod", Contexts: []string{"production"}}})

	err := runGroupRemove("prod", "production")
	if err == nil {
		t.Fatal("runGroupRemove() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "must contain at least one context") {
		t.Fatalf("runGroupRemove() error = %q, want empty-group message", err.Error())
	}
}

func setupGroupCommandTest(t *testing.T, groups []domaingroup.Group) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	configPath := filepath.Join(t.TempDir(), "config")
	cfg := domain.NewKubeConfig()
	cfg.CurrentContext = "staging"

	for _, name := range []string{"production", "staging"} {
		clusterName := name + "-cluster"
		userName := name + "-user"
		cfg.Clusters = append(cfg.Clusters, domain.ClusterEntry{Name: clusterName, Cluster: domain.Cluster{Server: "https://" + name + ".example.com"}})
		cfg.Users = append(cfg.Users, domain.UserEntry{Name: userName})
		cfg.Contexts = append(cfg.Contexts, domain.ContextEntry{Name: name, Context: domain.Context{Cluster: clusterName, User: userName}})
	}

	repo := infrastructure.NewFileRepository()
	if err := repo.Save(configPath, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	groupsPath := filepath.Join(home, ".kubecfg", "groups.yaml")
	store := groupstore.NewFileStore(groupsPath)
	for _, group := range groups {
		if err := store.Save(group); err != nil {
			t.Fatalf("group store Save(%q) error = %v", group.Name, err)
		}
	}

	kubeconfigPath = configPath
	service = application.NewService(
		repo,
		application.WithPreviousContextStore(infrastructure.NewPreviousContextStore(filepath.Join(home, ".kubecfg", "last-context"))),
	)
	groupService = appgroupservice.NewService(store, repo, configPath)

	return configPath
}
