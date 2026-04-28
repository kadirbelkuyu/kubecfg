package tui

import (
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
)

func TestFormatGroupLineIncludesPolicyBadge(t *testing.T) {
	line := formatGroupLine(domaingroup.Group{
		Name:        "prod",
		Description: "Production clusters",
		Policy:      "prod",
		Contexts:    []string{"prod-eu", "prod-us"},
	})

	for _, want := range []string{"prod", "[2]", "policy:prod", "Production clusters"} {
		if !strings.Contains(line, want) {
			t.Fatalf("formatGroupLine() = %q, want %q", line, want)
		}
	}
}

func TestActiveGuardPolicyNameUsesReadonlyFallback(t *testing.T) {
	status := &application.GuardStatus{
		Active:  true,
		Session: &domain.Session{},
	}

	policy, active := activeGuardPolicyName(status)
	if !active {
		t.Fatal("active = false, want true")
	}
	if policy != "readonly" {
		t.Fatalf("policy = %q, want readonly", policy)
	}
}
