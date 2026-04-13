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

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type GuardProxy struct {
	server    *http.Server
	auditSink domain.AuditStore
}

const guardHealthPath = "/.kubecfg/guard/healthz"

func NewGuardProxy(session *domain.Session, policy domain.GuardRequestPolicy, auditSink domain.AuditStore) (*GuardProxy, error) {
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

			if err := policy.Validate(request.Method, request.URL.RequestURI()); err != nil {
				appendGuardAuditEvent(auditSink, domain.AuditEvent{
					Type:      domain.AuditEventGuardRequestBlocked,
					SessionID: session.ID,
					Context:   session.TargetContext,
					Namespace: session.TargetNamespace,
					Mode:      string(session.Mode),
					Message:   err.Error(),
				})
				http.Error(writer, err.Error(), http.StatusForbidden)
				return
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
