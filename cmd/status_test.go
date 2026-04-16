package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/application/healthservice"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

type statusChecker struct {
	results map[string]health.Result
}

func (s statusChecker) Check(ctx context.Context, contextName string) health.Result {
	return s.results[contextName]
}

func (s statusChecker) CheckAll(ctx context.Context, contextNames []string) []health.Result {
	results := make([]health.Result, 0, len(contextNames))
	for _, name := range contextNames {
		results = append(results, s.results[name])
	}
	return results
}

type statusCache struct {
	values map[string]health.Result
}

func (s *statusCache) Get(contextName string) (health.Result, bool) {
	result, ok := s.values[contextName]
	return result, ok
}

func (s *statusCache) Set(result health.Result) {
	s.values[result.ContextName] = result
}

func (s *statusCache) Invalidate(contextName string) {
	delete(s.values, contextName)
}

func (s *statusCache) InvalidateAll() {
	s.values = make(map[string]health.Result)
}

func TestStatusExitCodeHealthy(t *testing.T) {
	setupStatusTest(t, map[string]health.Result{
		"prod": {ContextName: "prod", Status: health.StatusHealthy, CheckedAt: time.Now()},
	})

	err := statusCmd.RunE(statusCmd, nil)
	if err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
}

func TestStatusExitCodeUnhealthy(t *testing.T) {
	setupStatusTest(t, map[string]health.Result{
		"prod": {ContextName: "prod", Status: health.StatusUnhealthy, CheckedAt: time.Now()},
	})

	err := statusCmd.RunE(statusCmd, nil)
	if err == nil {
		t.Fatal("RunE() error = nil, want exitError")
	}
	code, ok := err.(exitCoder)
	if !ok {
		t.Fatalf("error type = %T, want exitCoder", err)
	}
	if code.ExitCode() != 1 {
		t.Fatalf("ExitCode() = %d, want 1", code.ExitCode())
	}
}

func TestStatusJSONOutput(t *testing.T) {
	setupStatusTest(t, map[string]health.Result{
		"prod": {ContextName: "prod", Status: health.StatusHealthy, CheckedAt: time.Now()},
	})
	statusOutput = "json"

	output := captureStdout(t, func() {
		if err := statusCmd.RunE(statusCmd, nil); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	var results []health.Result
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
}

func TestStatusSingleContextOutputIncludesServer(t *testing.T) {
	setupStatusTest(t, map[string]health.Result{
		"prod": {ContextName: "prod", Status: health.StatusHealthy, CheckedAt: time.Now()},
	})

	output := captureStdout(t, func() {
		if err := statusCmd.RunE(statusCmd, []string{"prod"}); err != nil {
			t.Fatalf("RunE() error = %v", err)
		}
	})

	if !strings.Contains(output, "https://prod.example.com") {
		t.Fatalf("output missing server URL: %s", output)
	}
}

func TestSuggestionForConnectionRefused(t *testing.T) {
	if suggestion := suggestionForError("connection refused"); !strings.Contains(suggestion, "minikube") {
		t.Fatalf("suggestion = %q, want minikube hint", suggestion)
	}
}

func TestSuggestionForTLS(t *testing.T) {
	if suggestion := suggestionForError("TLS handshake failed"); !strings.Contains(suggestion, "Certificate") {
		t.Fatalf("suggestion = %q, want TLS hint", suggestion)
	}
}

func setupStatusTest(t *testing.T, results map[string]health.Result) {
	t.Helper()

	statusContext = ""
	statusWatch = false
	statusTimeout = health.CheckTimeout
	statusOutput = "table"

	cfg := domain.NewKubeConfig()
	cfg.CurrentContext = "prod"
	cfg.Clusters = append(cfg.Clusters, domain.ClusterEntry{
		Name:    "prod-cluster",
		Cluster: domain.Cluster{Server: "https://prod.example.com"},
	})
	cfg.Users = append(cfg.Users, domain.UserEntry{Name: "prod-user"})
	cfg.Contexts = append(cfg.Contexts, domain.ContextEntry{
		Name: "prod",
		Context: domain.Context{
			Cluster: "prod-cluster",
			User:    "prod-user",
		},
	})

	repo := infrastructure.NewFileRepository()
	path := filepath.Join(t.TempDir(), "config")
	if err := repo.Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	kubeconfigPath = path
	service = application.NewService(repo)
	newStatusHealthServiceFn = func(timeout time.Duration) *healthservice.Service {
		return healthservice.New(
			statusChecker{results: results},
			&statusCache{values: make(map[string]health.Result)},
			repo,
			path,
		)
	}

	t.Cleanup(func() {
		newStatusHealthServiceFn = newStatusHealthService
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	_ = writer.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}

	return string(bytes.TrimSpace(data))
}
