package infrastructure

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type GuardProcessRuntime struct {
	binaryPath  string
	sessionPath string
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
		binaryPath:  binaryPath,
		sessionPath: sessionPath,
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
