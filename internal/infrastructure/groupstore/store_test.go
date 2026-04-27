package groupstore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
)

func TestFileStoreListReturnsEmptySliceWhenFileDoesNotExist(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "groups.yaml"))

	groups, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(groups) != 0 {
		t.Fatalf("List() len = %d, want 0", len(groups))
	}
}

func TestFileStoreSaveCreatesHumanReadableYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups.yaml")
	store := NewFileStore(path)

	if err := store.Save(testGroup("prod", "All production clusters", "red", "eks-prod", "gke-prod")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	text := string(data)
	for _, want := range []string{"groups:", "name: prod", "description: All production clusters", "color: red", "- eks-prod", "- gke-prod"} {
		if !strings.Contains(text, want) {
			t.Fatalf("groups.yaml does not contain %q:\n%s", want, text)
		}
	}
}

func TestFileStoreLoadsGroupsWithoutPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups.yaml")
	data := []byte(`groups:
  - name: prod
    description: All production clusters
    contexts:
      - eks-prod
`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	groups, err := NewFileStore(path).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("List() len = %d, want 1", len(groups))
	}
	if groups[0].Policy != "" {
		t.Fatalf("Policy = %q, want empty", groups[0].Policy)
	}
}

func TestFileStoreSaveWritesPolicyWhenSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups.yaml")
	store := NewFileStore(path)

	group := testGroup("prod", "All production clusters", "red", "eks-prod")
	group.Policy = "prod"
	if err := store.Save(group); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "policy: prod") {
		t.Fatalf("groups.yaml does not contain policy:\n%s", data)
	}
}

func TestFileStoreSaveOverwritesExistingGroupInPlace(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "groups.yaml"))

	if err := store.Save(testGroup("prod", "old", "red", "eks-prod")); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	if err := store.Save(testGroup("prod", "new", "blue", "gke-prod")); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	groups, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("List() len = %d, want 1", len(groups))
	}
	if groups[0].Description != "new" {
		t.Fatalf("Description = %q, want %q", groups[0].Description, "new")
	}
	if len(groups[0].Contexts) != 1 || groups[0].Contexts[0] != "gke-prod" {
		t.Fatalf("Contexts = %v, want [gke-prod]", groups[0].Contexts)
	}
}

func TestFileStoreSavePreservesInsertionOrder(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "groups.yaml"))

	for _, group := range []domaingroup.Group{
		testGroup("prod", "", "", "eks-prod"),
		testGroup("staging", "", "", "eks-staging"),
		testGroup("dev", "", "", "kind-dev"),
	} {
		if err := store.Save(group); err != nil {
			t.Fatalf("Save(%q) error = %v", group.Name, err)
		}
	}

	groups, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	got := []string{groups[0].Name, groups[1].Name, groups[2].Name}
	want := []string{"prod", "staging", "dev"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestFileStoreGetReturnsErrGroupNotFound(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "groups.yaml"))

	_, err := store.Get("missing")
	if !errors.Is(err, domaingroup.ErrGroupNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, domaingroup.ErrGroupNotFound)
	}
}

func TestFileStoreDeleteRemovesCorrectGroup(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "groups.yaml"))
	for _, group := range []domaingroup.Group{
		testGroup("prod", "", "", "eks-prod"),
		testGroup("staging", "", "", "eks-staging"),
	} {
		if err := store.Save(group); err != nil {
			t.Fatalf("Save(%q) error = %v", group.Name, err)
		}
	}

	if err := store.Delete("prod"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	groups, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(groups) != 1 || groups[0].Name != "staging" {
		t.Fatalf("remaining groups = %v, want [staging]", groups)
	}
}

func TestFileStoreWriteIsAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups.yaml")
	store := NewFileStore(path)

	if err := store.Save(testGroup("prod", "", "", "eks-prod")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file should not exist after save, stat error = %v", err)
	}
}

func testGroup(name, description, color string, contexts ...string) domaingroup.Group {
	return domaingroup.Group{
		Name:        name,
		Description: description,
		Contexts:    contexts,
		Color:       color,
	}
}
