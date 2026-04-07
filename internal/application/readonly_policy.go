package application

import (
	"fmt"
	"strings"
)

type ReadonlyRequestPolicy struct{}

type MutatingRequestBlockedError struct {
	Method string
	Path   string
}

func NewReadonlyRequestPolicy() *ReadonlyRequestPolicy {
	return &ReadonlyRequestPolicy{}
}

func (p *ReadonlyRequestPolicy) Validate(method, path string) error {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "HEAD":
		return nil
	default:
		return &MutatingRequestBlockedError{
			Method: strings.ToUpper(strings.TrimSpace(method)),
			Path:   normalizeGuardPath(path),
		}
	}
}

func (e *MutatingRequestBlockedError) Error() string {
	return fmt.Sprintf("guard readonly mode blocked mutating request: %s %s", e.Method, normalizeGuardPath(e.Path))
}

func normalizeGuardPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	return trimmed
}
