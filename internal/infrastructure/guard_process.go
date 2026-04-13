package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

const (
	defaultGuardReadyTimeout  = 5 * time.Second
	defaultGuardReadyInterval = 100 * time.Millisecond
)

type GuardProcessRuntime struct {
	binaryPath    string
	sessionPath   string
	httpClient    *http.Client
	readyTimeout  time.Duration
	readyInterval time.Duration
}

func NewGuardProcessRuntime(binaryPath, sessionPath string) (*GuardProcessRuntime, error) {
	if binaryPath == "" {
		resolvedPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve executable path: %w", err)
		}
		binaryPath = resolvedPath
	}

	return &GuardProcessRuntime{
		binaryPath:    binaryPath,
		sessionPath:   sessionPath,
		httpClient:    &http.Client{Timeout: time.Second},
		readyTimeout:  defaultGuardReadyTimeout,
		readyInterval: defaultGuardReadyInterval,
	}, nil
}

func (r *GuardProcessRuntime) NextListenAddress() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen on ephemeral port: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	return "http://" + listener.Addr().String(), nil
}

func (r *GuardProcessRuntime) Start(session *domain.Session) error {
	if err := r.validateLaunch(session); err != nil {
		return err
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, filePermission)
	if err != nil {
		return fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer func() {
		_ = devNull.Close()
	}()

	process, err := os.StartProcess(
		r.binaryPath,
		[]string{
			r.binaryPath,
			"guard",
			"proxy",
			"--session-file",
			r.sessionPath,
			"--session-id",
			session.ID,
		},
		&os.ProcAttr{
			Files: []*os.File{devNull, devNull, devNull},
		},
	)
	if err != nil {
		return fmt.Errorf("start guard proxy process: %w", err)
	}

	session.ProxyPID = process.Pid

	if err := process.Release(); err != nil {
		return fmt.Errorf("detach guard proxy process: %w", err)
	}

	if err := r.waitForReady(session); err != nil {
		_ = r.Stop(session)
		return err
	}

	return nil
}

func (r *GuardProcessRuntime) validateLaunch(session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session is required")
	}

	if _, err := uuid.Parse(session.ID); err != nil {
		return fmt.Errorf("validate session id: %w", err)
	}

	binaryPath, err := filepath.Abs(filepath.Clean(r.binaryPath))
	if err != nil {
		return fmt.Errorf("resolve binary path: %w", err)
	}

	fileInfo, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("stat binary path: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("binary path is a directory")
	}

	sessionPath, err := filepath.Abs(filepath.Clean(r.sessionPath))
	if err != nil {
		return fmt.Errorf("resolve session path: %w", err)
	}

	r.binaryPath = binaryPath
	r.sessionPath = sessionPath

	return nil
}

func (r *GuardProcessRuntime) Stop(session *domain.Session) error {
	if session == nil || session.ProxyPID == 0 {
		return nil
	}

	process, err := os.FindProcess(session.ProxyPID)
	if err != nil {
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("signal guard proxy process %d: %w", session.ProxyPID, err)
	}

	return nil
}

func (r *GuardProcessRuntime) IsRunning(session *domain.Session) bool {
	if session == nil || session.ProxyPID == 0 {
		return false
	}

	process, err := os.FindProcess(session.ProxyPID)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func (r *GuardProcessRuntime) waitForReady(session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session is required")
	}

	client := r.httpClient
	if client == nil {
		client = &http.Client{Timeout: time.Second}
	}

	timeout := r.readyTimeout
	if timeout <= 0 {
		timeout = defaultGuardReadyTimeout
	}

	interval := r.readyInterval
	if interval <= 0 {
		interval = defaultGuardReadyInterval
	}

	healthURL := session.ProxyListenAddress + guardHealthPath
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		requestContext := context.Background()
		cancel := func() {}
		if client.Timeout > 0 {
			requestContext, cancel = context.WithTimeout(context.Background(), client.Timeout)
		}

		request, err := http.NewRequestWithContext(requestContext, http.MethodGet, healthURL, nil)
		if err != nil {
			cancel()
			return fmt.Errorf("build guard proxy readiness request: %w", err)
		}

		response, err := client.Do(request)
		cancel()
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("unexpected readiness status: %s", response.Status)
		} else {
			lastErr = err
		}

		if time.Now().Add(interval).After(deadline) {
			break
		}

		time.Sleep(interval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("guard proxy did not become ready")
	}

	return fmt.Errorf("wait for guard proxy readiness: %w", lastErr)
}
