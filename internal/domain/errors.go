package domain

import "errors"

var (
	ErrConfigNotFound   = errors.New("kubeconfig file not found")
	ErrContextNotFound  = errors.New("context not found")
	ErrClusterNotFound  = errors.New("cluster not found")
	ErrUserNotFound     = errors.New("user not found")
	ErrContextExists    = errors.New("context already exists")
	ErrInvalidConfig    = errors.New("invalid kubeconfig format")
	ErrBackupFailed     = errors.New("failed to create backup")
	ErrBackupNotFound   = errors.New("backup not found")
	ErrRestoreFailed    = errors.New("failed to restore backup")
	ErrPermissionDenied = errors.New("permission denied")
	ErrNoCurrentContext = errors.New("no current context set")
)
