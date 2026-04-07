package infrastructure

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"

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
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, filePermission)
	if err != nil {
		return fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer func() {
		_ = devNull.Close()
	}()

	cmd := exec.Command(r.binaryPath, "guard", "proxy", "--session-file", r.sessionPath, "--session-id", session.ID)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start guard proxy process: %w", err)
	}

	session.ProxyPID = cmd.Process.Pid

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("detach guard proxy process: %w", err)
	}

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
