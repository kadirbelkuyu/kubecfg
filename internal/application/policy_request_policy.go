package application

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type PolicyRequestPolicy struct {
	policy *domain.Policy
}

type PolicyRequestBlockedError struct {
	Method string
	Path   string
	Reason string
}

func (e *PolicyRequestBlockedError) Error() string {
	return fmt.Sprintf("[kubecfg guard] %s policy: %s %s — %s", "policy", e.Method, normalizeGuardPath(e.Path), e.Reason)
}

type ConfirmRequiredError struct {
	Method    string
	Path      string
	Namespace string
	Reason    string
}

func (e *ConfirmRequiredError) Error() string {
	return fmt.Sprintf("[kubecfg guard] confirm required: %s %s — %s", e.Method, normalizeGuardPath(e.Path), e.Reason)
}

func IsConfirmRequired(err error) (*ConfirmRequiredError, bool) {
	var e *ConfirmRequiredError
	return e, errors.As(err, &e)
}

func NewPolicyRequestPolicy(policy *domain.Policy) *PolicyRequestPolicy {
	return &PolicyRequestPolicy{policy: policy}
}

func (p *PolicyRequestPolicy) Validate(method, path string) error {
	m := strings.ToUpper(strings.TrimSpace(method))
	normalPath := normalizeGuardPath(path)

	if p.policy.Readonly {
		switch m {
		case "GET", "HEAD":
		default:
			return &PolicyRequestBlockedError{
				Method: m,
				Path:   normalPath,
				Reason: "prod policy: write operations are disabled",
			}
		}
	}

	for _, verb := range p.policy.BlockedVerbs {
		if m == strings.ToUpper(strings.TrimSpace(verb)) {
			return &PolicyRequestBlockedError{
				Method: m,
				Path:   normalPath,
				Reason: fmt.Sprintf("verb %q is blocked by policy", strings.ToUpper(verb)),
			}
		}
	}

	if p.policy.ConfirmDestructive && m == "DELETE" {
		ns := extractNamespaceFromPath(normalPath)
		if ns == "" {
			return &ConfirmRequiredError{
				Method:    m,
				Path:      normalPath,
				Namespace: ns,
				Reason:    "cluster-wide destructive operation requires confirmation",
			}
		}
	}

	for _, resource := range p.policy.BlockedResources {
		if pathContainsResource(normalPath, resource) {
			return &PolicyRequestBlockedError{
				Method: m,
				Path:   normalPath,
				Reason: fmt.Sprintf("resource %q is blocked by policy", resource),
			}
		}
	}

	if len(p.policy.AllowedNamespaces) > 0 {
		ns := extractNamespaceFromPath(normalPath)
		if ns != "" && !namespaceAllowed(ns, p.policy.AllowedNamespaces) {
			return &PolicyRequestBlockedError{
				Method: m,
				Path:   normalPath,
				Reason: fmt.Sprintf("namespace %q is not in the allowed list", ns),
			}
		}
	}

	return nil
}

func pathContainsResource(path, resource string) bool {
	resourceLower := strings.ToLower(strings.TrimSpace(resource))
	cleanPath := strings.SplitN(path, "?", 2)[0]
	parts := strings.Split(strings.ToLower(cleanPath), "/")
	for _, part := range parts {
		if part == resourceLower {
			return true
		}
	}
	return false
}

func extractNamespaceFromPath(path string) string {
	cleanPath := strings.SplitN(path, "?", 2)[0]
	parts := strings.Split(cleanPath, "/")
	for i, part := range parts {
		if part == "namespaces" && i+1 < len(parts) && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return ""
}

func namespaceAllowed(ns string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(ns, a) {
			return true
		}
	}
	return false
}
