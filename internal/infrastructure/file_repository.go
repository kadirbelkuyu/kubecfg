package infrastructure

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
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

	root, fileName, err := openRootForPath(path)
	if err != nil {
		if os.IsPermission(err) {
			return nil, domain.ErrPermissionDenied
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()

	data, err := root.ReadFile(fileName)
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

	root, fileName, err := openRootForPath(path)
	if err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fmt.Errorf("failed to open config directory: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()

	tempFile := fileName + ".tmp"
	if err := root.WriteFile(tempFile, data, filePermission); err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err := root.Rename(tempFile, fileName); err != nil {
		_ = root.Remove(tempFile)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func (r *FileRepository) Backup(path string) error {
	if !r.Exists(path) {
		return nil
	}

	backupPath := fmt.Sprintf("%s.backup.%s", path, time.Now().Format("20060102-150405"))
	return r.copyFile(path, backupPath, domain.ErrBackupFailed)
}

func (r *FileRepository) ListBackups(path string) ([]string, error) {
	backups, err := filepath.Glob(path + ".backup.*")
	if err != nil {
		return nil, err
	}

	sort.Slice(backups, func(i, j int) bool {
		leftInfo, leftErr := os.Stat(backups[i])
		rightInfo, rightErr := os.Stat(backups[j])

		if leftErr != nil || rightErr != nil {
			return backups[i] > backups[j]
		}

		return leftInfo.ModTime().After(rightInfo.ModTime())
	})

	return backups, nil
}

func (r *FileRepository) RestoreBackup(targetPath, backupPath string) error {
	if !r.Exists(backupPath) {
		return domain.ErrBackupNotFound
	}

	return r.copyFile(backupPath, targetPath, domain.ErrRestoreFailed)
}

func (r *FileRepository) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *FileRepository) GetDefaultPath() string {
	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		return envPath
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, defaultConfigDir, defaultConfigFile)
}

func (r *FileRepository) copyFile(sourcePath, targetPath string, fallback error) (err error) {
	sourceRoot, sourceName, err := openRootForPath(sourcePath)
	if err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fallback
	}
	defer func() {
		_ = sourceRoot.Close()
	}()

	src, err := sourceRoot.Open(sourceName)
	if err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fallback
	}
	defer func() {
		if closeErr := src.Close(); err == nil && closeErr != nil {
			err = fallback
		}
	}()

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, dirPermission); err != nil {
		return fallback
	}

	targetRoot, targetName, err := openRootForPath(targetPath)
	if err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fallback
	}
	defer func() {
		_ = targetRoot.Close()
	}()

	tempPath := targetName + ".tmp"
	dst, err := targetRoot.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePermission)
	if err != nil {
		if os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		return fallback
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = targetRoot.Remove(tempPath)
		return fallback
	}

	if err := dst.Close(); err != nil {
		_ = targetRoot.Remove(tempPath)
		return fallback
	}

	if err := targetRoot.Rename(tempPath, targetName); err != nil {
		_ = targetRoot.Remove(tempPath)
		return fallback
	}

	return nil
}

func openRootForPath(path string) (*os.Root, string, error) {
	cleanPath := filepath.Clean(path)
	fileName := filepath.Base(cleanPath)
	if fileName == "." || fileName == string(filepath.Separator) {
		return nil, "", fmt.Errorf("invalid file path: %s", path)
	}

	root, err := os.OpenRoot(filepath.Dir(cleanPath))
	if err != nil {
		return nil, "", err
	}

	return root, fileName, nil
}
