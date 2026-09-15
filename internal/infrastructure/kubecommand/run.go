package kubecommand

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

type HTTPError struct {
	StatusCode int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d", e.StatusCode)
}

func Run(ctx context.Context, args ...string) ([]byte, error) {
	// #nosec G204 -- kubectl is invoked directly without a shell and arguments are passed literally.
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return nil, fmt.Errorf("run kubectl: %w", err)
	}
	message := strings.ToLower(string(exitErr.Stderr))
	switch {
	case strings.Contains(message, "gcloud auth login"):
		return nil, errors.New("authentication expired; run gcloud auth login in a terminal, then retry")
	case strings.Contains(message, "getting credentials"), strings.Contains(message, "unauthorized"), strings.Contains(message, "must be logged in"):
		return nil, errors.New("authentication failed; sign in with your cluster credential provider, then retry")
	case strings.Contains(message, "forbidden"):
		return nil, &HTTPError{StatusCode: http.StatusForbidden}
	case strings.Contains(message, "(notfound)"):
		return nil, &HTTPError{StatusCode: http.StatusNotFound}
	case strings.Contains(message, "(serviceunavailable)"):
		return nil, &HTTPError{StatusCode: http.StatusServiceUnavailable}
	case strings.Contains(message, "(internalerror)"):
		return nil, &HTTPError{StatusCode: http.StatusInternalServerError}
	case strings.Contains(message, "no such host"):
		return nil, errors.New("no such host")
	case strings.Contains(message, "connection refused"):
		return nil, errors.New("connection refused")
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return nil, errors.New("i/o timeout")
	case strings.Contains(message, "x509"), strings.Contains(message, "tls"):
		return nil, errors.New("TLS validation failed")
	default:
		return nil, fmt.Errorf("kubectl request failed (exit status %d)", exitErr.ExitCode())
	}
}
