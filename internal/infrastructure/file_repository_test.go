package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListBackupsReturnsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	oldBackup := configPath + ".backup.20260407-101500"
	newBackup := configPath + ".backup.20260407-103000"

	if err := os.WriteFile(oldBackup, []byte("old"), 0600); err != nil {
		t.Fatalf("WriteFile() old backup error = %v", err)
	}
	if err := os.WriteFile(newBackup, []byte("new"), 0600); err != nil {
		t.Fatalf("WriteFile() new backup error = %v", err)
	}

	oldTime := time.Date(2026, 4, 7, 10, 15, 0, 0, time.UTC)
	newTime := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
	if err := os.Chtimes(oldBackup, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes() old backup error = %v", err)
	}
	if err := os.Chtimes(newBackup, newTime, newTime); err != nil {
		t.Fatalf("Chtimes() new backup error = %v", err)
	}

	repo := NewFileRepository()
	backups, err := repo.ListBackups(configPath)
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("ListBackups() count = %d, want 2", len(backups))
	}

	if backups[0] != newBackup || backups[1] != oldBackup {
		t.Fatalf("ListBackups() order = %v", backups)
	}
}

func TestRestoreBackupCopiesBackupContent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	backupPath := configPath + ".backup.20260407-103000"

	if err := os.WriteFile(configPath, []byte("current"), 0600); err != nil {
		t.Fatalf("WriteFile() config error = %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("backup"), 0600); err != nil {
		t.Fatalf("WriteFile() backup error = %v", err)
	}

	repo := NewFileRepository()
	if err := repo.RestoreBackup(configPath, backupPath); err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(data) != "backup" {
		t.Fatalf("RestoreBackup() content = %q, want %q", string(data), "backup")
	}
}
