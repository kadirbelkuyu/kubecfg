package application

import (
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type rollbackMockRepository struct {
	backups        []string
	backupCalls    []string
	restoreCalls   []string
	listBackupsErr error
	restoreErr     error
	exists         bool
}

func (m *rollbackMockRepository) Load(path string) (*domain.KubeConfig, error) {
	return nil, nil
}

func (m *rollbackMockRepository) Save(path string, config *domain.KubeConfig) error {
	return nil
}

func (m *rollbackMockRepository) Backup(path string) error {
	m.backupCalls = append(m.backupCalls, path)
	return nil
}

func (m *rollbackMockRepository) ListBackups(path string) ([]string, error) {
	if m.listBackupsErr != nil {
		return nil, m.listBackupsErr
	}
	return m.backups, nil
}

func (m *rollbackMockRepository) RestoreBackup(targetPath, backupPath string) error {
	m.restoreCalls = append(m.restoreCalls, targetPath+"::"+backupPath)
	return m.restoreErr
}

func (m *rollbackMockRepository) Exists(path string) bool {
	return m.exists
}

func (m *rollbackMockRepository) GetDefaultPath() string {
	return "/tmp/config"
}

func TestListBackupsParsesTimestamp(t *testing.T) {
	repo := &rollbackMockRepository{
		backups: []string{"/tmp/config.backup.20260407-103000"},
	}
	service := NewService(repo)

	backups, err := service.ListBackups("/tmp/config")
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 1 {
		t.Fatalf("ListBackups() count = %d, want 1", len(backups))
	}

	if backups[0].Name != "config.backup.20260407-103000" {
		t.Fatalf("ListBackups() name = %q", backups[0].Name)
	}

	if backups[0].CreatedAt.IsZero() {
		t.Fatal("ListBackups() CreatedAt is zero")
	}

	expected := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
	if !backups[0].CreatedAt.Equal(expected) {
		t.Fatalf("ListBackups() CreatedAt = %v, want %v", backups[0].CreatedAt, expected)
	}
}

func TestRestoreBackupCreatesSafetyBackup(t *testing.T) {
	repo := &rollbackMockRepository{
		backups: []string{"/tmp/config.backup.20260407-103000"},
		exists:  true,
	}
	service := NewService(repo)

	err := service.RestoreBackup("/tmp/config", "/tmp/config.backup.20260407-103000")
	if err != nil {
		t.Fatalf("RestoreBackup() error = %v", err)
	}

	if len(repo.backupCalls) != 1 || repo.backupCalls[0] != "/tmp/config" {
		t.Fatalf("RestoreBackup() backup calls = %v", repo.backupCalls)
	}

	if len(repo.restoreCalls) != 1 || repo.restoreCalls[0] != "/tmp/config::/tmp/config.backup.20260407-103000" {
		t.Fatalf("RestoreBackup() restore calls = %v", repo.restoreCalls)
	}
}

func TestRestoreBackupRejectsUnknownBackup(t *testing.T) {
	service := NewService(&rollbackMockRepository{
		backups: []string{"/tmp/config.backup.20260407-103000"},
		exists:  true,
	})

	err := service.RestoreBackup("/tmp/config", "/tmp/config.backup.20260407-104500")
	if err != domain.ErrBackupNotFound {
		t.Fatalf("RestoreBackup() error = %v, want %v", err, domain.ErrBackupNotFound)
	}
}
