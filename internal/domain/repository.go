package domain

type KubeConfigRepository interface {
	Load(path string) (*KubeConfig, error)
	Exists(path string) bool
	GetDefaultPath() string
}

type Repository interface {
	KubeConfigRepository
	Save(path string, config *KubeConfig) error
	Backup(path string) error
	ListBackups(path string) ([]string, error)
	RestoreBackup(targetPath, backupPath string) error
}

type PreviousContextStore interface {
	ReadPrevious() (string, error)
	WritePrevious(name string) error
}
