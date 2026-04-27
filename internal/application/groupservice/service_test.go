package groupservice

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
)

type mockGroupStore struct {
	groups    []domaingroup.Group
	saveCalls int
}

func (m *mockGroupStore) List() ([]domaingroup.Group, error) {
	return append([]domaingroup.Group(nil), m.groups...), nil
}

func (m *mockGroupStore) Get(name string) (domaingroup.Group, error) {
	for _, group := range m.groups {
		if group.Name == name {
			return group, nil
		}
	}

	return domaingroup.Group{}, domaingroup.ErrGroupNotFound
}

func (m *mockGroupStore) Save(g domaingroup.Group) error {
	m.saveCalls++
	for index := range m.groups {
		if m.groups[index].Name == g.Name {
			m.groups[index] = g
			return nil
		}
	}

	m.groups = append(m.groups, g)
	return nil
}

func (m *mockGroupStore) Delete(name string) error {
	for index := range m.groups {
		if m.groups[index].Name == name {
			m.groups = append(m.groups[:index], m.groups[index+1:]...)
			return nil
		}
	}

	return domaingroup.ErrGroupNotFound
}

func (m *mockGroupStore) ReplaceAll(groups []domaingroup.Group) error {
	m.groups = append([]domaingroup.Group(nil), groups...)
	return nil
}

type mockKubeConfigRepository struct {
	config *domain.KubeConfig
}

func (m *mockKubeConfigRepository) Load(path string) (*domain.KubeConfig, error) {
	return m.config, nil
}

func (m *mockKubeConfigRepository) Exists(path string) bool {
	return true
}

func (m *mockKubeConfigRepository) GetDefaultPath() string {
	return "/tmp/config"
}

type mockPolicyResolver struct {
	policies map[string]*domain.Policy
}

func (m mockPolicyResolver) GetPolicy(name string) (*domain.Policy, error) {
	policy, ok := m.policies[name]
	if !ok {
		return nil, fmt.Errorf("unknown policy")
	}
	return policy, nil
}

func (m mockPolicyResolver) ValidatePolicy(policy *domain.Policy) error {
	return policy.Validate()
}

func TestCreateRejectsUnknownContextNames(t *testing.T) {
	service := newTestGroupService(nil, []string{"prod"})

	err := service.Create(domaingroup.Group{Name: "prod-team", Contexts: []string{"prod", "missing"}})
	var missingErr MissingContextsError
	if !errors.As(err, &missingErr) {
		t.Fatalf("Create() error = %v, want MissingContextsError", err)
	}
	if len(missingErr.Names) != 1 || missingErr.Names[0] != "missing" {
		t.Fatalf("missing contexts = %v, want [missing]", missingErr.Names)
	}
}

func TestCreateStoresPolicyWhenPolicyExists(t *testing.T) {
	service := newTestGroupServiceWithPolicies(nil, []string{"prod"}, map[string]*domain.Policy{
		"prod": {Name: "prod", Readonly: true},
	})

	err := service.Create(domaingroup.Group{Name: "prod-team", Contexts: []string{"prod"}, Policy: "prod"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	g, err := service.Get("prod-team")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if g.Policy != "prod" {
		t.Fatalf("Policy = %q, want prod", g.Policy)
	}
}

func TestCreateRejectsUnknownPolicy(t *testing.T) {
	service := newTestGroupServiceWithPolicies(nil, []string{"prod"}, map[string]*domain.Policy{
		"staging": {Name: "staging"},
	})

	err := service.Create(domaingroup.Group{Name: "prod-team", Contexts: []string{"prod"}, Policy: "missing"})
	if err == nil {
		t.Fatal("Create() error = nil, want unknown policy error")
	}
	if !strings.Contains(err.Error(), "unknown group policy") {
		t.Fatalf("Create() error = %q, want unknown policy message", err.Error())
	}
}

func TestSetAndUnsetPolicy(t *testing.T) {
	service := newTestGroupServiceWithPolicies([]domaingroup.Group{{Name: "prod", Contexts: []string{"prod"}}}, []string{"prod"}, map[string]*domain.Policy{
		"prod": {Name: "prod", Readonly: true},
	})

	if err := service.SetPolicy("prod", "prod"); err != nil {
		t.Fatalf("SetPolicy() error = %v", err)
	}
	g, err := service.Get("prod")
	if err != nil {
		t.Fatalf("Get() after SetPolicy error = %v", err)
	}
	if g.Policy != "prod" {
		t.Fatalf("Policy after SetPolicy = %q, want prod", g.Policy)
	}

	if err := service.UnsetPolicy("prod"); err != nil {
		t.Fatalf("UnsetPolicy() error = %v", err)
	}
	g, err = service.Get("prod")
	if err != nil {
		t.Fatalf("Get() after UnsetPolicy error = %v", err)
	}
	if g.Policy != "" {
		t.Fatalf("Policy after UnsetPolicy = %q, want empty", g.Policy)
	}
}

func TestCreateReturnsErrGroupAlreadyExists(t *testing.T) {
	service := newTestGroupService([]domaingroup.Group{{Name: "prod", Contexts: []string{"prod"}}}, []string{"prod"})

	err := service.Create(domaingroup.Group{Name: "prod", Contexts: []string{"prod"}})
	if !errors.Is(err, domaingroup.ErrGroupAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v", err, domaingroup.ErrGroupAlreadyExists)
	}
}

func TestAddContextNoOpsWhenContextAlreadyExists(t *testing.T) {
	store := &mockGroupStore{groups: []domaingroup.Group{{Name: "prod", Contexts: []string{"prod"}}}}
	service := NewService(store, &mockKubeConfigRepository{config: kubeConfigWithContexts("prod")}, "/tmp/config")

	err := service.AddContext("prod", "prod")
	if err != nil {
		t.Fatalf("AddContext() error = %v", err)
	}
	if store.saveCalls != 0 {
		t.Fatalf("Save() calls = %d, want 0", store.saveCalls)
	}
}

func TestRemoveContextReturnsErrEmptyContextListOnLastMember(t *testing.T) {
	service := newTestGroupService([]domaingroup.Group{{Name: "prod", Contexts: []string{"prod"}}}, []string{"prod"})

	err := service.RemoveContext("prod", "prod")
	if !errors.Is(err, domaingroup.ErrEmptyContextList) {
		t.Fatalf("RemoveContext() error = %v, want %v", err, domaingroup.ErrEmptyContextList)
	}
}

func TestResolveReturnsMissingContextsSeparately(t *testing.T) {
	service := newTestGroupService([]domaingroup.Group{{Name: "prod", Contexts: []string{"prod", "retired"}}}, []string{"prod"})

	group, missing, err := service.Resolve("prod")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if group.Name != "prod" {
		t.Fatalf("Resolve() group name = %q, want %q", group.Name, "prod")
	}
	if len(missing) != 1 || missing[0] != "retired" {
		t.Fatalf("Resolve() missing = %v, want [retired]", missing)
	}
}

func newTestGroupService(groups []domaingroup.Group, contexts []string) *Service {
	store := &mockGroupStore{groups: append([]domaingroup.Group(nil), groups...)}
	return NewService(store, &mockKubeConfigRepository{config: kubeConfigWithContexts(contexts...)}, "/tmp/config")
}

func newTestGroupServiceWithPolicies(groups []domaingroup.Group, contexts []string, policies map[string]*domain.Policy) *Service {
	store := &mockGroupStore{groups: append([]domaingroup.Group(nil), groups...)}
	return NewService(
		store,
		&mockKubeConfigRepository{config: kubeConfigWithContexts(contexts...)},
		"/tmp/config",
		WithPolicyResolver(mockPolicyResolver{policies: policies}),
	)
}

func kubeConfigWithContexts(names ...string) *domain.KubeConfig {
	config := domain.NewKubeConfig()
	for _, name := range names {
		clusterName := name + "-cluster"
		userName := name + "-user"
		config.Clusters = append(config.Clusters, domain.ClusterEntry{Name: clusterName, Cluster: domain.Cluster{Server: "https://" + name + ".example.com"}})
		config.Users = append(config.Users, domain.UserEntry{Name: userName})
		config.Contexts = append(config.Contexts, domain.ContextEntry{Name: name, Context: domain.Context{Cluster: clusterName, User: userName}})
	}

	return config
}
