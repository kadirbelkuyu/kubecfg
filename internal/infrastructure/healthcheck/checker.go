package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"

	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
)

type KubeChecker struct {
	kubeConfigPath string
	timeout        time.Duration
	dialContext    func(context.Context, string, string) (net.Conn, error)
}

func New(kubeConfigPath string) *KubeChecker {
	return NewWithTimeout(kubeConfigPath, health.CheckTimeout)
}

func NewWithTimeout(kubeConfigPath string, timeout time.Duration) *KubeChecker {
	if timeout <= 0 {
		timeout = health.CheckTimeout
	}

	return &KubeChecker{
		kubeConfigPath: kubeConfigPath,
		timeout:        timeout,
	}
}

func (c *KubeChecker) Check(ctx context.Context, contextName string) health.Result {
	start := time.Now()
	result := health.Result{
		ContextName: contextName,
		Status:      health.StatusUnknown,
		CheckedAt:   start,
	}

	client, serverURL, err := c.clientForContext(contextName)
	if err != nil {
		return c.classifyError(result, err)
	}

	readyzURL := serverURL.ResolveReference(&url.URL{Path: "/readyz"})
	healthzURL := serverURL.ResolveReference(&url.URL{Path: "/healthz"})

	readyzResult, readyzCode, readyzErr := c.checkEndpoint(ctx, client, readyzURL.String(), result)
	if readyzErr == nil && readyzCode == http.StatusOK {
		return readyzResult
	}
	if readyzErr != nil {
		return readyzResult
	}
	if readyzCode < http.StatusBadRequest {
		return readyzResult
	}

	healthzResult, healthzCode, healthzErr := c.checkEndpoint(ctx, client, healthzURL.String(), result)
	if healthzErr != nil {
		return healthzResult
	}
	if healthzCode == http.StatusOK {
		return healthzResult
	}
	if healthzCode >= http.StatusBadRequest {
		healthzResult.Status = health.StatusDegraded
		healthzResult.Error = fmt.Sprintf("HTTP %d from /readyz and /healthz", healthzCode)
	}

	return healthzResult
}

func (c *KubeChecker) CheckAll(ctx context.Context, contextNames []string) []health.Result {
	sem := make(chan struct{}, health.MaxConcurrency)
	results := make(chan health.Result, len(contextNames))
	var wg sync.WaitGroup

	for _, name := range contextNames {
		wg.Add(1)
		go func(contextName string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results <- health.Result{
					ContextName: contextName,
					Status:      health.StatusUnknown,
					CheckedAt:   time.Now().UTC(),
				}
				return
			}
			defer func() { <-sem }()

			results <- c.Check(ctx, contextName)
		}(name)
	}

	wg.Wait()
	close(results)

	output := make([]health.Result, 0, len(contextNames))
	for result := range results {
		output = append(output, result)
	}

	return output
}

func (c *KubeChecker) clientForContext(contextName string) (*http.Client, *url.URL, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: c.kubeConfigPath}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("load context %q: %w", contextName, err)
	}

	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("parse server url for %q: %w", contextName, err)
	}

	transportConfig, err := restConfig.TransportConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("build transport config for %q: %w", contextName, err)
	}

	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build TLS config for %q: %w", contextName, err)
	}

	baseTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           c.transportDialContext(),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: c.timeout,
		TLSClientConfig:       tlsConfig,
	}

	roundTripper, err := transport.HTTPWrappersForConfig(transportConfig, baseTransport)
	if err != nil {
		return nil, nil, fmt.Errorf("wrap transport for %q: %w", contextName, err)
	}

	return &http.Client{
		Transport: roundTripper,
		Timeout:   c.timeout,
	}, serverURL, nil
}

func (c *KubeChecker) transportDialContext() func(context.Context, string, string) (net.Conn, error) {
	if c.dialContext != nil {
		return c.dialContext
	}
	return (&net.Dialer{Timeout: c.timeout, KeepAlive: 30 * time.Second}).DialContext
}

func (c *KubeChecker) checkEndpoint(ctx context.Context, client *http.Client, endpoint string, base health.Result) (health.Result, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	result := base
	start := time.Now()
	firstByteLatency := time.Duration(0)
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			firstByteLatency = time.Since(start)
		},
	}

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(reqCtx, trace), http.MethodGet, endpoint, nil)
	if err != nil {
		return c.classifyError(result, err), 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return c.classifyError(result, err), 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if firstByteLatency <= 0 {
		firstByteLatency = time.Since(start)
	}

	result.CheckedAt = start.UTC()
	result.Latency = firstByteLatency

	switch {
	case resp.StatusCode == http.StatusOK && result.Latency <= health.DegradedThreshold:
		result.Status = health.StatusHealthy
	case resp.StatusCode == http.StatusOK:
		result.Status = health.StatusDegraded
		result.Error = fmt.Sprintf("response slower than %s", health.DegradedThreshold)
	case resp.StatusCode >= http.StatusBadRequest:
		result.Status = health.StatusDegraded
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	default:
		result.Status = health.StatusHealthy
	}

	return result, resp.StatusCode, nil
}

func (c *KubeChecker) classifyError(result health.Result, err error) health.Result {
	result.CheckedAt = time.Now().UTC()
	result.Latency = 0

	switch {
	case err == nil:
		return result
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		result.Status = health.StatusUnknown
		result.Error = ""
		return result
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		result.Status = health.StatusUnreachable
		result.Error = "no such host"
		return result
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		result.Status = health.StatusUnhealthy
		result.Error = "TLS hostname validation failed"
		return result
	}

	var unknownAuthorityErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthorityErr) {
		result.Status = health.StatusUnhealthy
		result.Error = "TLS certificate authority validation failed"
		return result
	}

	var certInvalidErr x509.CertificateInvalidError
	if errors.As(err, &certInvalidErr) {
		result.Status = health.StatusUnhealthy
		result.Error = "TLS certificate validation failed"
		return result
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Timeout() {
			result.Status = health.StatusUnhealthy
			result.Error = "i/o timeout"
			return result
		}

		if errors.As(opErr.Err, &dnsErr) {
			result.Status = health.StatusUnreachable
			result.Error = "no such host"
			return result
		}

		if errors.Is(opErr.Err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNREFUSED) {
			result.Status = health.StatusUnhealthy
			result.Error = "connection refused"
			return result
		}
	}

	var recordHeaderErr tls.RecordHeaderError
	if errors.As(err, &recordHeaderErr) {
		result.Status = health.StatusUnhealthy
		result.Error = "TLS handshake failed"
		return result
	}

	if strings.Contains(strings.ToLower(err.Error()), "tls") || strings.Contains(strings.ToLower(err.Error()), "x509") {
		result.Status = health.StatusUnhealthy
		result.Error = "TLS handshake failed"
		return result
	}

	if strings.Contains(strings.ToLower(err.Error()), "connection refused") {
		result.Status = health.StatusUnhealthy
		result.Error = "connection refused"
		return result
	}

	if strings.Contains(strings.ToLower(err.Error()), "no such host") {
		result.Status = health.StatusUnreachable
		result.Error = "no such host"
		return result
	}

	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		result.Status = health.StatusUnhealthy
		result.Error = "i/o timeout"
		return result
	}

	result.Status = health.StatusUnhealthy
	result.Error = "request failed"
	return result
}
