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
	ErrPermissionDenied = errors.New("permission denied")
)
