package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

const (
	guardHealthPath     = "/.kubecfg/guard/healthz"
	confirmPollInterval = 250 * time.Millisecond
	confirmTimeout      = 30 * time.Second
)

type GuardProxy struct {
	server    *http.Server
	auditSink domain.AuditStore
}

func NewGuardProxy(
	session *domain.Session,
	policy domain.GuardRequestPolicy,
	auditSink domain.AuditStore,
	confirmStore domain.ConfirmationStore,
) (*GuardProxy, error) {
	if session == nil {
		return nil, fmt.Errorf("session is required")
	}

	targetURL, transport, err := buildGuardTransport(session.SourceKubeconfigPath, session.TargetContext)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	reverseProxy.Transport = transport
	reverseProxy.FlushInterval = 100 * time.Millisecond
	reverseProxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, proxyErr error) {
		http.Error(writer, fmt.Sprintf("guard proxy upstream error: %v", proxyErr), http.StatusBadGateway)
	}

	serverURL, err := url.Parse(session.ProxyListenAddress)
	if err != nil {
		return nil, fmt.Errorf("parse proxy listen address: %w", err)
	}

	isDebug := session.PolicyName == domain.PolicyProfileDebug

	server := &http.Server{
		Addr:              serverURL.Host,
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == guardHealthPath {
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte("ok"))
				return
			}

			if session.IsExpired(time.Now()) {
				appendGuardAuditEvent(auditSink, domain.AuditEvent{
					Type:      domain.AuditEventGuardSessionExpired,
					SessionID: session.ID,
					Context:   session.TargetContext,
					Namespace: session.TargetNamespace,
					Mode:      string(session.Mode),
					Message:   "guard session expired while handling request",
				})
				http.Error(writer, "guard session expired", http.StatusForbidden)
				return
			}

			validationErr := policy.Validate(request.Method, request.URL.RequestURI())

			if confirmErr, ok := application.IsConfirmRequired(validationErr); ok {
				appendGuardAuditEvent(auditSink, domain.AuditEvent{
					Type:      domain.AuditEventGuardRequestPending,
					SessionID: session.ID,
					Context:   session.TargetContext,
					Namespace: session.TargetNamespace,
					Mode:      string(session.Mode),
					Message:   confirmErr.Error(),
				})

				if confirmStore == nil {
					http.Error(writer, "[kubecfg guard] confirmation required but no confirmation store configured", http.StatusForbidden)
					return
				}

				pending := &domain.PendingConfirmation{
					ID:        uuid.NewString(),
					SessionID: session.ID,
					Method:    request.Method,
					Resource:  request.URL.RequestURI(),
					Namespace: confirmErr.Namespace,
					CreatedAt: time.Now().UTC(),
					Decision:  domain.ConfirmDecisionPending,
				}

				if err := confirmStore.Create(pending); err != nil {
					http.Error(writer, fmt.Sprintf("[kubecfg guard] failed to create confirmation: %v", err), http.StatusInternalServerError)
					return
				}

				decision := waitForDecision(confirmStore, pending.ID, confirmTimeout)
				_ = confirmStore.Delete(pending.ID)

				switch decision {
				case domain.ConfirmDecisionApproved:
					appendGuardAuditEvent(auditSink, domain.AuditEvent{
						Type:      domain.AuditEventGuardRequestApproved,
						SessionID: session.ID,
						Context:   session.TargetContext,
						Namespace: session.TargetNamespace,
						Mode:      string(session.Mode),
						Message:   fmt.Sprintf("approved: %s %s", request.Method, request.URL.RequestURI()),
					})
					reverseProxy.ServeHTTP(writer, request)
				default:
					appendGuardAuditEvent(auditSink, domain.AuditEvent{
						Type:      domain.AuditEventGuardRequestDenied,
						SessionID: session.ID,
						Context:   session.TargetContext,
						Namespace: session.TargetNamespace,
						Mode:      string(session.Mode),
						Message:   fmt.Sprintf("denied (timeout or explicit): %s %s", request.Method, request.URL.RequestURI()),
					})
					http.Error(writer, "[kubecfg guard] destructive operation denied or confirmation timed out", http.StatusForbidden)
				}
				return
			}

			if validationErr != nil {
				appendGuardAuditEvent(auditSink, domain.AuditEvent{
					Type:      domain.AuditEventGuardRequestBlocked,
					SessionID: session.ID,
					Context:   session.TargetContext,
					Namespace: session.TargetNamespace,
					Mode:      string(session.Mode),
					Message:   validationErr.Error(),
				})
				http.Error(writer, validationErr.Error(), http.StatusForbidden)
				return
			}

			if isDebug {
				appendGuardAuditEvent(auditSink, domain.AuditEvent{
					Type:      domain.AuditEventGuardRequestAllowed,
					SessionID: session.ID,
					Context:   session.TargetContext,
					Namespace: session.TargetNamespace,
					Mode:      string(session.Mode),
					Message:   fmt.Sprintf("%s %s", request.Method, request.URL.RequestURI()),
				})
			}

			reverseProxy.ServeHTTP(writer, request)
		}),
	}

	return &GuardProxy{server: server, auditSink: auditSink}, nil
}

func (p *GuardProxy) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = p.server.Shutdown(shutdownCtx)
	}()

	if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve guard proxy: %w", err)
	}

	return nil
}

func waitForDecision(store domain.ConfirmationStore, id string, timeout time.Duration) domain.ConfirmDecision {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(confirmPollInterval)
		pending, err := store.Read(id)
		if err != nil {
			return domain.ConfirmDecisionDenied
		}
		if pending.Decision != domain.ConfirmDecisionPending {
			return pending.Decision
		}
	}
	return domain.ConfirmDecisionDenied
}

func buildGuardTransport(sourcePath, contextName string) (*url.URL, http.RoundTripper, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: sourcePath}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("build upstream client config: %w", err)
	}

	targetURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("parse upstream server url: %w", err)
	}

	transport, err := rest.TransportFor(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build upstream transport: %w", err)
	}

	return targetURL, transport, nil
}

func appendGuardAuditEvent(store domain.AuditStore, event domain.AuditEvent) {
	if store == nil {
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	_ = store.Append(event)
}
