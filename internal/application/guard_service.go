package application

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type GuardStartOptions struct {
	SourcePath string
	TTL        time.Duration
}

type GuardStatus struct {
	Session   *domain.Session
	Active    bool
	Running   bool
	Expired   bool
	Health    string
	Remaining time.Duration
}

type GuardService struct {
	repo         domain.Repository
	sessionSvc   *SessionService
	writer       domain.GuardedKubeconfigWriter
	runtime      domain.GuardRuntime
	artifactsDir string
	defaultTTL   time.Duration
	now          func() time.Time
}

func NewGuardService(
	repo domain.Repository,
	sessionSvc *SessionService,
	writer domain.GuardedKubeconfigWriter,
	runtime domain.GuardRuntime,
	artifactsDir string,
	defaultTTL time.Duration,
) *GuardService {
	return &GuardService{
		repo:         repo,
		sessionSvc:   sessionSvc,
		writer:       writer,
		runtime:      runtime,
		artifactsDir: artifactsDir,
		defaultTTL:   defaultTTL,
		now:          time.Now,
	}
}

func (s *GuardService) StartReadonly(options GuardStartOptions) (*domain.Session, error) {
	if err := s.sessionSvc.PrepareForStart(); err != nil {
		return nil, err
	}

	ttl := options.TTL
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("guard ttl must be greater than zero")
	}

	contextName, namespace, err := s.resolveTargetContext(options.SourcePath)
	if err != nil {
		return nil, err
	}

	proxyAddress, err := s.runtime.NextListenAddress()
	if err != nil {
		return nil, fmt.Errorf("allocate proxy address: %w", err)
	}

	sessionID := uuid.NewString()
	generatedPath := filepath.Join(s.artifactsDir, sessionID, "config")

	result, err := s.writer.Write(options.SourcePath, contextName, proxyAddress, generatedPath)
	if err != nil {
		return nil, fmt.Errorf("write guarded kubeconfig: %w", err)
	}

	startedAt := s.now().UTC()
	session := &domain.Session{
		ID:                      sessionID,
		StartedAt:               startedAt,
		ExpiresAt:               startedAt.Add(ttl),
		Readonly:                true,
		Mode:                    domain.GuardModeReadonly,
		SourceKubeconfigPath:    options.SourcePath,
		GeneratedKubeconfigPath: result.Path,
		ProxyListenAddress:      proxyAddress,
		TargetContext:           result.Context,
		TargetNamespace:         firstNonEmpty(result.Namespace, namespace),
	}

	if err := s.sessionSvc.Save(session); err != nil {
		_ = s.writer.Cleanup(result.Path)
		return nil, fmt.Errorf("save guard session: %w", err)
	}

	if err := s.runtime.Start(session); err != nil {
		_ = s.sessionSvc.Delete()
		_ = s.writer.Cleanup(result.Path)
		return nil, fmt.Errorf("start guard proxy: %w", err)
	}

	if err := s.sessionSvc.Save(session); err != nil {
		_ = s.runtime.Stop(session)
		_ = s.sessionSvc.Delete()
		_ = s.writer.Cleanup(result.Path)
		return nil, fmt.Errorf("save guard session pid: %w", err)
	}

	return session, nil
}

func (s *GuardService) Status() (*GuardStatus, error) {
	return s.sessionSvc.Status()
}

func (s *GuardService) Stop() (*domain.Session, error) {
	return s.sessionSvc.Stop()
}

func (s *GuardService) resolveTargetContext(sourcePath string) (string, string, error) {
	if !s.repo.Exists(sourcePath) {
		return "", "", domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("load kubeconfig: %w", err)
	}

	if config.CurrentContext == "" {
		return "", "", domain.ErrNoCurrentContext
	}

	contextEntry, idx := config.FindContext(config.CurrentContext)
	if idx < 0 {
		return "", "", domain.ErrContextNotFound
	}

	return contextEntry.Name, contextEntry.Context.Namespace, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type SessionService struct {
	store   domain.SessionStore
	runtime domain.GuardRuntime
	writer  domain.GuardedKubeconfigWriter
	now     func() time.Time
}

func NewSessionService(store domain.SessionStore, runtime domain.GuardRuntime, writer domain.GuardedKubeconfigWriter) *SessionService {
	return &SessionService{
		store:   store,
		runtime: runtime,
		writer:  writer,
		now:     time.Now,
	}
}

func (s *SessionService) Save(session *domain.Session) error {
	if err := s.store.Save(session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func (s *SessionService) Delete() error {
	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SessionService) Status() (*GuardStatus, error) {
	session, err := s.store.Load()
	if err != nil {
		if errors.Is(err, domain.ErrGuardSessionNotFound) {
			return &GuardStatus{Health: "inactive"}, nil
		}
		return nil, fmt.Errorf("load session: %w", err)
	}

	now := s.now()
	status := &GuardStatus{
		Session:   session,
		Health:    "inactive",
		Remaining: session.Remaining(now),
	}

	if session.IsExpired(now) {
		status.Expired = true
		status.Health = "expired"
		if cleanupErr := s.cleanup(session); cleanupErr != nil {
			return nil, cleanupErr
		}
		return status, nil
	}

	status.Running = s.runtime.IsRunning(session)
	if !status.Running {
		status.Health = "proxy stopped"
		return status, nil
	}

	status.Active = true
	status.Health = "active"
	return status, nil
}

func (s *SessionService) PrepareForStart() error {
	status, err := s.Status()
	if err != nil {
		return err
	}

	if status.Session == nil || status.Expired {
		return nil
	}

	if status.Active {
		return fmt.Errorf("%w: %s", domain.ErrGuardSessionActive, status.Session.ID)
	}

	return s.cleanup(status.Session)
}

func (s *SessionService) Stop() (*domain.Session, error) {
	session, err := s.store.Load()
	if err != nil {
		if errors.Is(err, domain.ErrGuardSessionNotFound) {
			return nil, domain.ErrGuardSessionNotFound
		}
		return nil, fmt.Errorf("load session: %w", err)
	}

	if err := s.cleanup(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *SessionService) cleanup(session *domain.Session) error {
	if session == nil {
		return nil
	}

	if err := s.runtime.Stop(session); err != nil {
		return fmt.Errorf("stop proxy: %w", err)
	}

	if err := s.writer.Cleanup(session.GeneratedKubeconfigPath); err != nil {
		return fmt.Errorf("cleanup guarded kubeconfig: %w", err)
	}

	if err := s.store.Delete(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
