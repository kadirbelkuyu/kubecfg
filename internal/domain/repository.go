package domain

type Repository interface {
	Load(path string) (*KubeConfig, error)
	Save(path string, config *KubeConfig) error
	Backup(path string) error
	ListBackups(path string) ([]string, error)
	RestoreBackup(targetPath, backupPath string) error
	Exists(path string) bool
	GetDefaultPath() string
}
