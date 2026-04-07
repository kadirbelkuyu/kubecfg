package domain

import (
	"strings"
	"time"
)

type Session struct {
	ID                      string    `json:"id"`
	StartedAt               time.Time `json:"started_at"`
	ExpiresAt               time.Time `json:"expires_at"`
	Readonly                bool      `json:"readonly"`
	Mode                    GuardMode `json:"mode,omitempty"`
	SourceKubeconfigPath    string    `json:"source_kubeconfig_path"`
	GeneratedKubeconfigPath string    `json:"generated_kubeconfig_path"`
	ProxyListenAddress      string    `json:"proxy_listen_address"`
	TargetContext           string    `json:"target_context"`
	TargetNamespace         string    `json:"target_namespace,omitempty"`
	ProxyPID                int       `json:"proxy_pid,omitempty"`
}

func (s *Session) IsExpired(now time.Time) bool {
	if s == nil || s.ExpiresAt.IsZero() {
		return false
	}
	return !s.ExpiresAt.After(now)
}

func (s *Session) Remaining(now time.Time) time.Duration {
	if s == nil || s.IsExpired(now) {
		return 0
	}
	return s.ExpiresAt.Sub(now)
}

func (s *Session) NamespaceDisplay() string {
	if s == nil || strings.TrimSpace(s.TargetNamespace) == "" {
		return "all namespaces"
	}
	return s.TargetNamespace
}

func (s *Session) ModeDisplay() string {
	if s == nil {
		return ""
	}
	if s.Readonly || s.Mode == GuardModeReadonly {
		return "Readonly"
	}
	return "Standard"
}

type SessionStore interface {
	Load() (*Session, error)
	Save(session *Session) error
	Delete() error
}
