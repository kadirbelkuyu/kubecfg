package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

func TestGuardProcessWaitForReadySucceeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != guardHealthPath {
			http.NotFound(writer, request)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	runtime := &GuardProcessRuntime{
		httpClient:    server.Client(),
		readyTimeout:  time.Second,
		readyInterval: time.Millisecond,
	}

	session := &domain.Session{ProxyListenAddress: server.URL}
	if err := runtime.waitForReady(session); err != nil {
		t.Fatalf("waitForReady() error = %v", err)
	}
}

func TestGuardProcessWaitForReadyTimesOut(t *testing.T) {
	runtime := &GuardProcessRuntime{
		httpClient:    &http.Client{Timeout: 10 * time.Millisecond},
		readyTimeout:  25 * time.Millisecond,
		readyInterval: 5 * time.Millisecond,
	}

	session := &domain.Session{ProxyListenAddress: "http://127.0.0.1:1"}
	err := runtime.waitForReady(session)
	if err == nil {
		t.Fatal("waitForReady() error = nil, want timeout")
	}

	if !strings.Contains(err.Error(), "wait for guard proxy readiness") {
		t.Fatalf("waitForReady() error = %q, want readiness context", err.Error())
	}
}
