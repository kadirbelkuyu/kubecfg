package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetActiveKubeconfigPathPersistsSource(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KUBECONFIG", "")
	Init()

	configPath := filepath.Join(home, "configs", "team.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("apiVersion: v1\nkind: Config\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := SetActiveKubeconfigPath(configPath); err != nil {
		t.Fatalf("SetActiveKubeconfigPath() error = %v", err)
	}

	if got := GetKubeconfigPath(); got != configPath {
		t.Fatalf("GetKubeconfigPath() = %q, want %q", got, configPath)
	}

	data, err := os.ReadFile(filepath.Join(home, ".kubecfg", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "active: " + configPath; !strings.Contains(string(data), want) {
		t.Fatalf("config file = %q, want %q", string(data), want)
	}
}

func TestKubeconfigSourceDirsCanBeManaged(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("KUBECONFIG", "")
	Init()

	dir := filepath.Join(home, "team-kubeconfigs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := AddKubeconfigSourceDir(dir); err != nil {
		t.Fatalf("AddKubeconfigSourceDir() error = %v", err)
	}
	if !containsPath(GetKubeconfigSourceDirs(), dir) {
		t.Fatalf("GetKubeconfigSourceDirs() does not include %q: %#v", dir, GetKubeconfigSourceDirs())
	}

	if err := RemoveKubeconfigSourceDir(dir); err != nil {
		t.Fatalf("RemoveKubeconfigSourceDir() error = %v", err)
	}
	if containsPath(GetKubeconfigSourceDirs(), dir) {
		t.Fatalf("GetKubeconfigSourceDirs() still includes %q: %#v", dir, GetKubeconfigSourceDirs())
	}
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if path == target {
			return true
		}
	}
	return false
}
