package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
)

func TestReadPreviousReturnsEmptyWhenFileDoesNotExist(t *testing.T) {
	setupHistoryHome(t)

	got, err := ReadPrevious()
	if err != nil {
		t.Fatalf("ReadPrevious() error = %v", err)
	}

	if got != "" {
		t.Fatalf("ReadPrevious() = %q, want empty string", got)
	}
}

func TestReadPreviousReturnsStoredNameAfterWritePrevious(t *testing.T) {
	setupHistoryHome(t)

	if err := WritePrevious("staging"); err != nil {
		t.Fatalf("WritePrevious() error = %v", err)
	}

	got, err := ReadPrevious()
	if err != nil {
		t.Fatalf("ReadPrevious() error = %v", err)
	}

	if got != "staging" {
		t.Fatalf("ReadPrevious() = %q, want %q", got, "staging")
	}
}

func TestWritePreviousIsAtomic(t *testing.T) {
	setupHistoryHome(t)

	if err := WritePrevious("prod"); err != nil {
		t.Fatalf("WritePrevious() error = %v", err)
	}

	data, err := os.ReadFile(config.GetLastContextPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != "prod" {
		t.Fatalf("ReadFile() = %q, want %q", string(data), "prod")
	}

	if _, err := os.Stat(config.GetLastContextPath() + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file should not exist after rename, stat error = %v", err)
	}
}

func TestWritePreviousCreatesParentDirIfMissing(t *testing.T) {
	home := setupHistoryHome(t)
	targetDir := filepath.Join(home, ".kubecfg")

	if err := os.RemoveAll(targetDir); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	if err := WritePrevious("dev"); err != nil {
		t.Fatalf("WritePrevious() error = %v", err)
	}

	if _, err := os.Stat(config.GetLastContextPath()); err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
}

func setupHistoryHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	config.Init()
	return home
}
