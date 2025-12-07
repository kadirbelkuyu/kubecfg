package domain

type Repository interface {
	Load(path string) (*KubeConfig, error)
	Save(path string, config *KubeConfig) error
	Backup(path string) error
	Exists(path string) bool
	GetDefaultPath() string
}
