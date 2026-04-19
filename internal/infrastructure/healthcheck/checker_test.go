package healthcheck

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

func TestCheckHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(testKubeconfigPath(t, map[string]string{"prod": server.URL}))
	result := checker.Check(context.Background(), "prod")

	if result.Status != health.StatusHealthy {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusHealthy)
	}
}

func TestCheckDegradedLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(health.DegradedThreshold + 100*time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewWithTimeout(testKubeconfigPath(t, map[string]string{"prod": server.URL}), health.CheckTimeout+time.Second)
	result := checker.Check(context.Background(), "prod")

	if result.Status != health.StatusDegraded {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusDegraded)
	}
}

func TestCheckConnectionRefused(t *testing.T) {
	checker := NewWithTimeout(testKubeconfigPath(t, map[string]string{"prod": "http://127.0.0.1:1"}), time.Second)
	result := checker.Check(context.Background(), "prod")

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusUnhealthy)
	}
	if result.Error != "connection refused" {
		t.Fatalf("Error = %q, want connection refused", result.Error)
	}
}

func TestCheckUnreachable(t *testing.T) {
	checker := New(testKubeconfigPath(t, map[string]string{"prod": "https://example.invalid"}))
	checker.dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, &net.DNSError{Err: "no such host", Name: addr}
	}

	result := checker.Check(context.Background(), "prod")

	if result.Status != health.StatusUnreachable {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusUnreachable)
	}
	if result.Error != "no such host" {
		t.Fatalf("Error = %q, want no such host", result.Error)
	}
}

func TestCheckTLSError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(testKubeconfigPath(t, map[string]string{"prod": server.URL}))
	result := checker.Check(context.Background(), "prod")

	if result.Status != health.StatusUnhealthy {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusUnhealthy)
	}
	if len(result.Error) < 3 || result.Error[:3] != "TLS" {
		t.Fatalf("Error = %q, want TLS-prefixed error", result.Error)
	}
}

func TestCheckAllReturnsPerContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := New(testKubeconfigPath(t, map[string]string{
		"prod":    server.URL,
		"staging": server.URL,
		"dev":     server.URL,
	}))

	results := checker.CheckAll(context.Background(), []string{"prod", "staging", "dev"})
	if len(results) != 3 {
		t.Fatalf("len(CheckAll()) = %d, want 3", len(results))
	}
}

func TestCheckAllTiming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	servers := make(map[string]string, 10)
	names := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		name := "ctx-" + string(rune('a'+i))
		servers[name] = server.URL
		names = append(names, name)
	}

	checker := NewWithTimeout(testKubeconfigPath(t, servers), health.CheckTimeout+time.Second)

	start := time.Now()
	results := checker.CheckAll(context.Background(), names)
	if len(results) != len(names) {
		t.Fatalf("len(CheckAll()) = %d, want %d", len(results), len(names))
	}
	if elapsed := time.Since(start); elapsed > health.CheckTimeout+time.Second {
		t.Fatalf("CheckAll() took %s, want <= %s", elapsed, health.CheckTimeout+time.Second)
	}
}

func TestCheckAllCancellation(t *testing.T) {
	checker := New(testKubeconfigPath(t, map[string]string{"prod": "https://example.invalid"}))
	checker.dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results := checker.CheckAll(ctx, []string{"prod"})
	if len(results) != 1 {
		t.Fatalf("len(CheckAll()) = %d, want 1", len(results))
	}
	if results[0].Status != health.StatusUnknown {
		t.Fatalf("Status = %v, want %v", results[0].Status, health.StatusUnknown)
	}
}

func testKubeconfigPath(t *testing.T, servers map[string]string) string {
	t.Helper()

	cfg := domain.NewKubeConfig()
	repo := infrastructure.NewFileRepository()
	for name, server := range servers {
		clusterName := name + "-cluster"
		userName := name + "-user"
		cfg.Clusters = append(cfg.Clusters, domain.ClusterEntry{
			Name:    clusterName,
			Cluster: domain.Cluster{Server: server},
		})
		cfg.Users = append(cfg.Users, domain.UserEntry{
			Name: userName,
			User: domain.User{},
		})
		cfg.Contexts = append(cfg.Contexts, domain.ContextEntry{
			Name: name,
			Context: domain.Context{
				Cluster: clusterName,
				User:    userName,
			},
		})
	}

	path := filepath.Join(t.TempDir(), "config")
	if err := repo.Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	return path
}

func TestClassifyTLSErrors(t *testing.T) {
	checker := New("")
	result := checker.classifyError(health.Result{ContextName: "prod"}, tls.RecordHeaderError{})
	if result.Status != health.StatusUnhealthy {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusUnhealthy)
	}
}

func TestClassifyConnectionRefused(t *testing.T) {
	checker := New("")
	refused := &net.OpError{Err: &os.SyscallError{Err: syscall.ECONNREFUSED}}
	result := checker.classifyError(health.Result{ContextName: "prod"}, refused)
	if result.Error != "connection refused" {
		t.Fatalf("Error = %q, want connection refused", result.Error)
	}
}

func TestClassifyCanceled(t *testing.T) {
	checker := New("")
	result := checker.classifyError(health.Result{ContextName: "prod"}, context.Canceled)
	if result.Status != health.StatusUnknown {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusUnknown)
	}
}
