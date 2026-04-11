package application

import (
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type guardRepo struct {
	config *domain.KubeConfig
	exists bool
}

func (r *guardRepo) Load(path string) (*domain.KubeConfig, error) {
	return r.config, nil
}

func (r *guardRepo) Save(path string, config *domain.KubeConfig) error {
	r.config = config
	return nil
}

func (r *guardRepo) Backup(path string) error {
	return nil
}

func (r *guardRepo) ListBackups(path string) ([]string, error) {
	return nil, nil
}

func (r *guardRepo) RestoreBackup(targetPath, backupPath string) error {
	return nil
}

func (r *guardRepo) Exists(path string) bool {
	return r.exists
}

func (r *guardRepo) GetDefaultPath() string {
	return "/tmp/config"
}

type guardStore struct {
	session *domain.Session
}

func (s *guardStore) Load() (*domain.Session, error) {
	if s.session == nil {
		return nil, domain.ErrGuardSessionNotFound
	}
	return s.session, nil
}

func (s *guardStore) Save(session *domain.Session) error {
	copy := *session
	s.session = &copy
	return nil
}

func (s *guardStore) Delete() error {
	s.session = nil
	return nil
}

type guardRuntime struct {
	address     string
	startCalled bool
	stopCalled  bool
	running     bool
}

func (r *guardRuntime) NextListenAddress() (string, error) {
	return r.address, nil
}

func (r *guardRuntime) Start(session *domain.Session) error {
	r.startCalled = true
	r.running = true
	session.ProxyPID = 4242
	return nil
}

func (r *guardRuntime) Stop(session *domain.Session) error {
	r.stopCalled = true
	r.running = false
	return nil
}

func (r *guardRuntime) IsRunning(session *domain.Session) bool {
	return r.running
}

type guardWriter struct {
	cleanupCount int
}

func (w *guardWriter) Write(sourcePath, contextName, proxyAddress, outputPath string) (*domain.GuardedKubeconfigResult, error) {
	return &domain.GuardedKubeconfigResult{
		Path:      outputPath,
		Context:   contextName,
		Namespace: "payments",
	}, nil
}

func (w *guardWriter) Cleanup(outputPath string) error {
	w.cleanupCount++
	return nil
}

type guardAuditStore struct {
	events []domain.AuditEvent
}

func (s *guardAuditStore) Append(event domain.AuditEvent) error {
	s.events = append(s.events, event)
	return nil
}

func (s *guardAuditStore) ListRecent(limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 || len(s.events) == 0 {
		return nil, nil
	}

	if limit > len(s.events) {
		limit = len(s.events)
	}

	events := make([]domain.AuditEvent, 0, limit)
	for index := len(s.events) - 1; index >= len(s.events)-limit; index-- {
		events = append(events, s.events[index])
	}

	return events, nil
}

func TestGuardSessionLifecycle(t *testing.T) {
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	repo := &guardRepo{
		exists: true,
		config: &domain.KubeConfig{
			CurrentContext: "prod",
			Clusters: []domain.ClusterEntry{
				{Name: "prod-cluster", Cluster: domain.Cluster{Server: "https://prod.example.com"}},
			},
			Users: []domain.UserEntry{
				{Name: "prod-user"},
			},
			Contexts: []domain.ContextEntry{
				{Name: "prod", Context: domain.Context{Cluster: "prod-cluster", User: "prod-user", Namespace: "payments"}},
			},
		},
	}
	store := &guardStore{}
	runtime := &guardRuntime{address: "http://127.0.0.1:41001"}
	writer := &guardWriter{}
	auditStore := &guardAuditStore{}
	auditService := NewAuditService(auditStore, true)
	auditService.now = func() time.Time { return now }
	sessionService := NewSessionService(store, runtime, writer, auditService)
	sessionService.now = func() time.Time { return now }
	guardService := NewGuardService(repo, sessionService, writer, runtime, auditService, "/tmp/guard", 30*time.Minute)
	guardService.now = func() time.Time { return now }

	session, err := guardService.StartReadonly(GuardStartOptions{
		SourcePath: "/tmp/config",
		TTL:        30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("StartReadonly() error = %v", err)
	}

	if session.TargetContext != "prod" {
		t.Fatalf("StartReadonly() context = %q, want %q", session.TargetContext, "prod")
	}

	if session.TargetNamespace != "payments" {
		t.Fatalf("StartReadonly() namespace = %q, want %q", session.TargetNamespace, "payments")
	}

	if session.GeneratedKubeconfigPath != "/tmp/guard/"+session.ID+"/config" {
		t.Fatalf("StartReadonly() generated path = %q", session.GeneratedKubeconfigPath)
	}

	if !runtime.startCalled {
		t.Fatal("StartReadonly() did not start runtime")
	}

	status, err := guardService.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if !status.Active || !status.Running {
		t.Fatalf("Status() active=%v running=%v, want both true", status.Active, status.Running)
	}

	if status.Remaining != 30*time.Minute {
		t.Fatalf("Status() remaining = %v, want %v", status.Remaining, 30*time.Minute)
	}

	if status.Detail != "guard proxy is ready" {
		t.Fatalf("Status() detail = %q, want %q", status.Detail, "guard proxy is ready")
	}

	if len(status.RecentEvents) == 0 || status.RecentEvents[0].Type != domain.AuditEventGuardSessionStarted {
		t.Fatalf("Status() recent events = %+v, want started event", status.RecentEvents)
	}

	stopped, err := guardService.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if stopped.ID != session.ID {
		t.Fatalf("Stop() session id = %q, want %q", stopped.ID, session.ID)
	}

	if !runtime.stopCalled {
		t.Fatal("Stop() did not stop runtime")
	}

	if writer.cleanupCount != 1 {
		t.Fatalf("Stop() cleanup count = %d, want 1", writer.cleanupCount)
	}

	status, err = guardService.Status()
	if err != nil {
		t.Fatalf("Status() after stop error = %v", err)
	}

	if status.Session != nil || status.Active {
		t.Fatalf("Status() after stop = %+v, want no session", status)
	}

	if len(status.RecentEvents) == 0 || status.RecentEvents[0].Type != domain.AuditEventGuardSessionStopped {
		t.Fatalf("Status() after stop recent events = %+v, want stopped event", status.RecentEvents)
	}
}

func TestGuardStatusCleansExpiredSession(t *testing.T) {
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	store := &guardStore{
		session: &domain.Session{
			ID:                      "expired",
			StartedAt:               now.Add(-time.Hour),
			ExpiresAt:               now.Add(-time.Minute),
			Readonly:                true,
			GeneratedKubeconfigPath: "/tmp/guard/expired/config",
			ProxyPID:                9911,
		},
	}
	runtime := &guardRuntime{running: true}
	writer := &guardWriter{}
	auditStore := &guardAuditStore{}
	auditService := NewAuditService(auditStore, true)
	auditService.now = func() time.Time { return now }
	sessionService := NewSessionService(store, runtime, writer, auditService)
	sessionService.now = func() time.Time { return now }

	status, err := sessionService.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if !status.Expired || status.Health != "expired" {
		t.Fatalf("Status() = %+v, want expired status", status)
	}

	if status.Detail != "session ttl elapsed" {
		t.Fatalf("Status() detail = %q, want %q", status.Detail, "session ttl elapsed")
	}

	if store.session != nil {
		t.Fatal("Status() did not delete expired session")
	}

	if !runtime.stopCalled {
		t.Fatal("Status() did not stop expired runtime")
	}

	if writer.cleanupCount != 1 {
		t.Fatalf("Status() cleanup count = %d, want 1", writer.cleanupCount)
	}

	if len(auditStore.events) == 0 || auditStore.events[0].Type != domain.AuditEventGuardSessionExpired {
		t.Fatalf("Status() audit events = %+v, want expired event", auditStore.events)
	}
}

func TestGuardStartReplacesExpiredSession(t *testing.T) {
	now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	repo := &guardRepo{
		exists: true,
		config: &domain.KubeConfig{
			CurrentContext: "prod",
			Clusters: []domain.ClusterEntry{
				{Name: "prod-cluster", Cluster: domain.Cluster{Server: "https://prod.example.com"}},
			},
			Users: []domain.UserEntry{
				{Name: "prod-user"},
			},
			Contexts: []domain.ContextEntry{
				{Name: "prod", Context: domain.Context{Cluster: "prod-cluster", User: "prod-user"}},
			},
		},
	}
	store := &guardStore{
		session: &domain.Session{
			ID:                      "old-session",
			StartedAt:               now.Add(-time.Hour),
			ExpiresAt:               now.Add(-time.Second),
			Readonly:                true,
			GeneratedKubeconfigPath: "/tmp/guard/old-session/config",
			ProxyPID:                1234,
		},
	}
	runtime := &guardRuntime{address: "http://127.0.0.1:41002", running: true}
	writer := &guardWriter{}
	auditStore := &guardAuditStore{}
	auditService := NewAuditService(auditStore, true)
	auditService.now = func() time.Time { return now }
	sessionService := NewSessionService(store, runtime, writer, auditService)
	sessionService.now = func() time.Time { return now }
	guardService := NewGuardService(repo, sessionService, writer, runtime, auditService, "/tmp/guard", 45*time.Minute)
	guardService.now = func() time.Time { return now }

	session, err := guardService.StartReadonly(GuardStartOptions{
		SourcePath: "/tmp/config",
	})
	if err != nil {
		t.Fatalf("StartReadonly() error = %v", err)
	}

	if session.ID == "old-session" {
		t.Fatal("StartReadonly() reused expired session")
	}

	if writer.cleanupCount != 1 {
		t.Fatalf("StartReadonly() cleanup count = %d, want 1", writer.cleanupCount)
	}

	if !runtime.startCalled {
		t.Fatal("StartReadonly() did not start replacement runtime")
	}
}
