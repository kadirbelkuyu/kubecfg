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
	SourcePath    string
	TTL           time.Duration
	Profile       domain.PolicyProfile
	TargetContext string
	ReplaceActive bool
}

type GuardStatus struct {
	Session      *domain.Session
	Active       bool
	Running      bool
	Expired      bool
	Health       string
	Detail       string
	Remaining    time.Duration
	RecentEvents []domain.AuditEvent
}

type GuardService struct {
	repo         domain.Repository
	sessionSvc   *SessionService
	writer       domain.GuardedKubeconfigWriter
	runtime      domain.GuardRuntime
	audit        *AuditService
	policies     guardPolicyResolver
	artifactsDir string
	defaultTTL   time.Duration
	now          func() time.Time
}

type guardPolicyResolver interface {
	GetPolicy(name string) (*domain.Policy, error)
}

type GuardServiceOption func(*GuardService)

func WithGuardPolicyResolver(resolver guardPolicyResolver) GuardServiceOption {
	return func(service *GuardService) {
		service.policies = resolver
	}
}

func NewGuardService(
	repo domain.Repository,
	sessionSvc *SessionService,
	writer domain.GuardedKubeconfigWriter,
	runtime domain.GuardRuntime,
	audit *AuditService,
	artifactsDir string,
	defaultTTL time.Duration,
	options ...GuardServiceOption,
) *GuardService {
	service := &GuardService{
		repo:         repo,
		sessionSvc:   sessionSvc,
		writer:       writer,
		runtime:      runtime,
		audit:        audit,
		artifactsDir: artifactsDir,
		defaultTTL:   defaultTTL,
		now:          time.Now,
	}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *GuardService) StartReadonly(options GuardStartOptions) (*domain.Session, error) {
	if err := s.sessionSvc.PrepareForStart(options.ReplaceActive); err != nil {
		return nil, err
	}

	readonly, mode, err := s.resolvePolicy(options.Profile)
	if err != nil {
		return nil, err
	}

	ttl := options.TTL
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("guard ttl must be greater than zero")
	}

	contextName, namespace, err := s.resolveTargetContext(options.SourcePath, options.TargetContext)
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
		Readonly:                readonly,
		Mode:                    mode,
		PolicyName:              options.Profile,
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
		_ = s.recordAuditEvent(domain.AuditEvent{
			Type:      domain.AuditEventGuardStartFailed,
			SessionID: session.ID,
			Context:   session.TargetContext,
			Namespace: session.TargetNamespace,
			Mode:      string(session.Mode),
			Message:   fmt.Sprintf("failed to start readonly guard: %v", err),
		})
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

	_ = s.recordAuditEvent(domain.AuditEvent{
		Type:      domain.AuditEventGuardSessionStarted,
		SessionID: session.ID,
		Context:   session.TargetContext,
		Namespace: session.TargetNamespace,
		Mode:      string(session.Mode),
		Message:   fmt.Sprintf("readonly guard started at %s", session.ProxyListenAddress),
	})

	return session, nil
}

func (s *GuardService) Status() (*GuardStatus, error) {
	return s.sessionSvc.Status()
}

func (s *GuardService) Stop() (*domain.Session, error) {
	return s.sessionSvc.Stop()
}

func (s *GuardService) resolveTargetContext(sourcePath, targetContext string) (string, string, error) {
	if !s.repo.Exists(sourcePath) {
		return "", "", domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("load kubeconfig: %w", err)
	}

	if targetContext == "" {
		targetContext = config.CurrentContext
	}
	if targetContext == "" {
		return "", "", domain.ErrNoCurrentContext
	}

	contextEntry, idx := config.FindContext(targetContext)
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

func (s *GuardService) resolvePolicy(profile domain.PolicyProfile) (readonly bool, mode domain.GuardMode, err error) {
	if profile == "" {
		return true, domain.GuardModeReadonly, nil
	}
	p, err := s.findPolicy(profile)
	if err != nil {
		return false, "", err
	}
	if p.Readonly {
		return true, domain.GuardModeReadonly, nil
	}
	return false, domain.GuardModePolicy, nil
}

func (s *GuardService) findPolicy(profile domain.PolicyProfile) (*domain.Policy, error) {
	if s.policies != nil {
		return s.policies.GetPolicy(profile)
	}
	return domain.FindProfile(profile)
}

type SessionService struct {
	store   domain.SessionStore
	runtime domain.GuardRuntime
	writer  domain.GuardedKubeconfigWriter
	audit   *AuditService
	now     func() time.Time
}

func NewSessionService(store domain.SessionStore, runtime domain.GuardRuntime, writer domain.GuardedKubeconfigWriter, audit *AuditService) *SessionService {
	return &SessionService{
		store:   store,
		runtime: runtime,
		writer:  writer,
		audit:   audit,
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
	recentEvents, err := s.recentAuditEvents()
	if err != nil {
		return nil, err
	}

	session, err := s.store.Load()
	if err != nil {
		if errors.Is(err, domain.ErrGuardSessionNotFound) {
			return &GuardStatus{
				Health:       "inactive",
				Detail:       "no active guard session",
				RecentEvents: recentEvents,
			}, nil
		}
		return nil, fmt.Errorf("load session: %w", err)
	}

	now := s.now()
	status := &GuardStatus{
		Session:      session,
		Health:       "inactive",
		Detail:       "guard session loaded",
		Remaining:    session.Remaining(now),
		RecentEvents: recentEvents,
	}

	if session.IsExpired(now) {
		status.Expired = true
		status.Health = "expired"
		status.Detail = "session ttl elapsed"
		_ = s.recordAuditEvent(domain.AuditEvent{
			Type:      domain.AuditEventGuardSessionExpired,
			SessionID: session.ID,
			Context:   session.TargetContext,
			Namespace: session.TargetNamespace,
			Mode:      string(session.Mode),
			Message:   "guard session expired and was cleaned up",
		})
		if cleanupErr := s.cleanup(session); cleanupErr != nil {
			return nil, cleanupErr
		}
		status.RecentEvents, _ = s.recentAuditEvents()
		return status, nil
	}

	status.Running = s.runtime.IsRunning(session)
	if !status.Running {
		status.Health = "proxy stopped"
		status.Detail = "guard proxy process is not running"
		return status, nil
	}

	status.Active = true
	status.Health = "active"
	status.Detail = "guard proxy is ready"
	return status, nil
}

func (s *SessionService) PrepareForStart(replaceActive bool) error {
	status, err := s.Status()
	if err != nil {
		return err
	}

	if status.Session == nil || status.Expired {
		return nil
	}

	if status.Active && !replaceActive {
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

	_ = s.recordAuditEvent(domain.AuditEvent{
		Type:      domain.AuditEventGuardSessionStopped,
		SessionID: session.ID,
		Context:   session.TargetContext,
		Namespace: session.TargetNamespace,
		Mode:      string(session.Mode),
		Message:   "guard session stopped and artifacts cleaned up",
	})

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

func (s *SessionService) recentAuditEvents() ([]domain.AuditEvent, error) {
	if s.audit == nil {
		return nil, nil
	}

	events, err := s.audit.Recent(defaultAuditRecentLimit)
	if err != nil {
		return nil, fmt.Errorf("load recent audit events: %w", err)
	}

	return events, nil
}

func (s *SessionService) recordAuditEvent(event domain.AuditEvent) error {
	if s.audit == nil {
		return nil
	}

	if err := s.audit.Record(event); err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}

	return nil
}

func (s *GuardService) recordAuditEvent(event domain.AuditEvent) error {
	if s.audit == nil {
		return nil
	}

	if err := s.audit.Record(event); err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}

	return nil
}
