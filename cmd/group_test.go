package cmd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	appgroupservice "github.com/kadirbelkuyu/kubecfg/internal/application/groupservice"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/groupstore"
)

var lastTestGuardStore *testGuardStore

func TestRunGroupCreateFailsClearlyWhenContextDoesNotExist(t *testing.T) {
	setupGroupCommandTest(t, nil)

	err := runGroupCreate("prod", []string{"missing"}, "", "", "")
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

func TestRunGroupCreateStoresPolicy(t *testing.T) {
	setupGroupCommandTest(t, nil)

	err := runGroupCreate("prod", []string{"production"}, "", "red", "prod")
	if err != nil {
		t.Fatalf("runGroupCreate() error = %v", err)
	}

	group, err := groupService.Get("prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if group.Policy != "prod" {
		t.Fatalf("Policy = %q, want prod", group.Policy)
	}
}

func TestRunGroupCreateRejectsUnknownPolicy(t *testing.T) {
	setupGroupCommandTest(t, nil)

	err := runGroupCreate("prod", []string{"production"}, "", "", "missing")
	if err == nil {
		t.Fatal("runGroupCreate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown group policy") {
		t.Fatalf("runGroupCreate() error = %q, want unknown policy message", err.Error())
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

func TestRunGroupUseActivatesBoundPolicyGuard(t *testing.T) {
	configPath := setupGroupCommandTest(t, []domaingroup.Group{{Name: "prod", Policy: "prod", Contexts: []string{"production"}}})

	err := runGroupUse("prod")
	if err != nil {
		t.Fatalf("runGroupUse() error = %v", err)
	}

	if got := readCurrentContext(t, configPath); got != "production" {
		t.Fatalf("current context after runGroupUse() = %q, want production", got)
	}
	status, err := guardService.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.Active || status.Session == nil {
		t.Fatalf("guard status = %+v, want active session", status)
	}
	if status.Session.PolicyName != "prod" {
		t.Fatalf("PolicyName = %q, want prod", status.Session.PolicyName)
	}
	if status.Session.TargetContext != "production" {
		t.Fatalf("TargetContext = %q, want production", status.Session.TargetContext)
	}
}

func TestRunGroupUseOverridesActiveGuardPolicy(t *testing.T) {
	setupGroupCommandTest(t, []domaingroup.Group{{Name: "prod", Policy: "prod", Contexts: []string{"production"}}})

	lastTestGuardStore.session = &domain.Session{
		ID:                      "active-debug",
		StartedAt:               time.Now().Add(-time.Minute),
		ExpiresAt:               time.Now().Add(time.Hour),
		PolicyName:              "debug",
		GeneratedKubeconfigPath: filepath.Join(t.TempDir(), "guard", "active-debug", "config"),
		ProxyPID:                4242,
	}

	err := runGroupUse("prod")
	if err != nil {
		t.Fatalf("runGroupUse() error = %v", err)
	}

	status, err := guardService.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Session.PolicyName != "prod" {
		t.Fatalf("PolicyName after override = %q, want prod", status.Session.PolicyName)
	}
	if status.Session.ID == "active-debug" {
		t.Fatal("runGroupUse() kept the old guard session")
	}
}

func TestRunGroupUseWithoutPolicyDoesNotStartGuard(t *testing.T) {
	setupGroupCommandTest(t, []domaingroup.Group{{Name: "prod", Contexts: []string{"production"}}})

	err := runGroupUse("prod")
	if err != nil {
		t.Fatalf("runGroupUse() error = %v", err)
	}

	status, err := guardService.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Active || status.Session != nil {
		t.Fatalf("guard status = %+v, want inactive", status)
	}
}

func TestRenderGroupTableIncludesColumns(t *testing.T) {
	output := renderGroupTable([]domaingroup.Group{{Name: "prod", Description: "All production clusters", Policy: "prod", Contexts: []string{"eks-prod", "gke-prod"}}}, false)

	for _, want := range []string{"NAME", "CONTEXTS", "POLICY", "DESCRIPTION", "prod", "All production clusters"} {
		if !strings.Contains(output, want) {
			t.Fatalf("renderGroupTable() output does not contain %q:\n%s", want, output)
		}
	}
}

func TestRenderGroupDetailsIncludesPolicy(t *testing.T) {
	output := renderGroupDetails(domaingroup.Group{Name: "prod", Policy: "prod", Contexts: []string{"production"}}, nil)

	if !strings.Contains(output, "Policy: prod") {
		t.Fatalf("renderGroupDetails() output missing policy:\n%s", output)
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
	policyService = application.NewPolicyService(nil)
	groupService = appgroupservice.NewService(store, repo, configPath, appgroupservice.WithPolicyResolver(policyService))
	guardStore := &testGuardStore{}
	lastTestGuardStore = guardStore
	guardRuntime := &testGuardRuntime{address: "http://127.0.0.1:41001", running: true}
	guardWriter := &testGuardWriter{}
	sessionService := application.NewSessionService(guardStore, guardRuntime, guardWriter, nil)
	guardService = application.NewGuardService(
		repo,
		sessionService,
		guardWriter,
		guardRuntime,
		nil,
		filepath.Join(home, ".kubecfg", "guard"),
		30*time.Minute,
		application.WithGuardPolicyResolver(policyService),
	)

	return configPath
}

type testGuardStore struct {
	session *domain.Session
}

func (s *testGuardStore) Load() (*domain.Session, error) {
	if s.session == nil {
		return nil, domain.ErrGuardSessionNotFound
	}
	return s.session, nil
}

func (s *testGuardStore) Save(session *domain.Session) error {
	copy := *session
	s.session = &copy
	return nil
}

func (s *testGuardStore) Delete() error {
	s.session = nil
	return nil
}

type testGuardRuntime struct {
	address string
	running bool
}

func (r *testGuardRuntime) NextListenAddress() (string, error) {
	return r.address, nil
}

func (r *testGuardRuntime) Start(session *domain.Session) error {
	r.running = true
	session.ProxyPID = 4242
	return nil
}

func (r *testGuardRuntime) Stop(session *domain.Session) error {
	r.running = false
	return nil
}

func (r *testGuardRuntime) IsRunning(session *domain.Session) bool {
	return r.running
}

type testGuardWriter struct{}

func (w *testGuardWriter) Write(sourcePath, contextName, proxyAddress, outputPath string) (*domain.GuardedKubeconfigResult, error) {
	return &domain.GuardedKubeconfigResult{Path: outputPath, Context: contextName}, nil
}

func (w *testGuardWriter) Cleanup(outputPath string) error {
	return nil
}
