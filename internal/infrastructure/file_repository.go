package infrastructure

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	defaultConfigDir  = ".kube"
	defaultConfigFile = "config"
	filePermission    = 0600
	dirPermission     = 0700
)

type FileRepository struct{}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) Load(path string) (*domain.KubeConfig, error) {
	if !r.Exists(path) {
		return nil, domain.ErrConfigNotFound
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsPermission(err) {
			return nil, domain.ErrPermissionDenied
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config domain.KubeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, domain.ErrInvalidConfig
	}

	return &config, nil
}

func (r *FileRepository) Save(path string, config *domain.KubeConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPermission); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, filePermission); err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := os.Rename(tempFile, path); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func (r *FileRepository) Backup(path string) (err error) {
	if !r.Exists(path) {
		return nil
	}

	src, err := os.Open(path)
	if err != nil {
		return domain.ErrBackupFailed
	}
	defer func() {
		if closeErr := src.Close(); err == nil && closeErr != nil {
			err = domain.ErrBackupFailed
		}
	}()

	backupPath := fmt.Sprintf("%s.backup.%s", path, time.Now().Format("20060102-150405"))
	dst, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, filePermission)
	if err != nil {
		return domain.ErrBackupFailed
	}
	defer func() {
		if closeErr := dst.Close(); err == nil && closeErr != nil {
			err = domain.ErrBackupFailed
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(backupPath)
		return domain.ErrBackupFailed
	}

	return nil
}

func (r *FileRepository) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *FileRepository) GetDefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, defaultConfigDir, defaultConfigFile)
}
