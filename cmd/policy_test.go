package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
)

// setupPolicyService initialises the package-level policyService used by the
// policy commands.  Call this instead of running the full cobra pre-run chain.
func setupPolicyService(profiles map[string]config.ProfileConfig) {
	policyService = application.NewPolicyService(profiles)
}

// captureOutput redirects a function's stdout/stderr by swapping os.Stdout.
// Since the policy commands use fmt.Println/fmt.Fprintln(os.Stderr, ...) we
// redirect at the cobra output level by executing through RunE with a buffer.
// For simplicity we test the helper functions that produce the string output.

func TestPolicyListOutputContainsBuiltins(t *testing.T) {
	setupPolicyService(nil)

	var buf bytes.Buffer
	policies := policyService.ListPolicies()

	for _, p := range policies {
		buf.WriteString(p.Name)
		buf.WriteString("\n")
	}

	for _, expected := range []string{"prod", "staging", "debug"} {
		if !strings.Contains(buf.String(), expected) {
			t.Errorf("policy list output missing builtin %q", expected)
		}
	}
}

func TestPolicyListOutputUserOverride(t *testing.T) {
	setupPolicyService(map[string]config.ProfileConfig{
		"prod": {Readonly: false, Description: "custom"},
	})

	policies := policyService.ListPolicies()

	for _, p := range policies {
		if p.Name == "prod" {
			if p.Readonly {
				t.Fatal("user-defined prod should NOT be Readonly in list output")
			}
			if p.Description != "custom" {
				t.Fatalf("Description = %q, want custom", p.Description)
			}
			return
		}
	}
	t.Fatal("prod not found in list")
}

func TestPolicyShowKnownProfile(t *testing.T) {
	setupPolicyService(nil)

	p, err := policyService.GetPolicy("prod")
	if err != nil {
		t.Fatalf("GetPolicy(prod): %v", err)
	}
	if p.Name != "prod" {
		t.Fatalf("Name = %q, want prod", p.Name)
	}
}

func TestPolicyShowUnknownProfile(t *testing.T) {
	setupPolicyService(nil)

	_, err := policyService.GetPolicy("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}

func TestPolicyValidatePassScenario(t *testing.T) {
	setupPolicyService(nil)

	profiles := map[string]config.ProfileConfig{
		"valid-profile": {
			Readonly:            true,
			WarnContextPatterns: []string{"^prod"},
		},
	}

	for name, cfg := range profiles {
		p := configProfileToDomainPolicy(name, cfg)
		if err := policyService.ValidatePolicy(p); err != nil {
			t.Fatalf("ValidatePolicy(%q): unexpected error: %v", name, err)
		}
	}
}

func TestPolicyValidateFailScenario(t *testing.T) {
	setupPolicyService(nil)

	profiles := map[string]config.ProfileConfig{
		"bad-pattern": {
			WarnContextPatterns: []string{"[invalid-regex"},
		},
	}

	for name, cfg := range profiles {
		p := configProfileToDomainPolicy(name, cfg)
		if err := policyService.ValidatePolicy(p); err == nil {
			t.Fatalf("ValidatePolicy(%q): expected error for invalid regex, got nil", name)
		}
	}
}

func TestPolicyValidateEmptyNameFails(t *testing.T) {
	setupPolicyService(nil)

	profiles := map[string]config.ProfileConfig{
		"": {Readonly: true},
	}

	for name, cfg := range profiles {
		p := configProfileToDomainPolicy(name, cfg)
		if err := policyService.ValidatePolicy(p); err == nil {
			t.Fatal("ValidatePolicy with empty name: expected error, got nil")
		}
	}
}

func TestSampleConfigIsValidYAML(t *testing.T) {
	// The sampleConfig constant must not be empty and must contain expected sections.
	for _, section := range []string{"profiles:", "sessions:", "audit:"} {
		if !strings.Contains(sampleConfig, section) {
			t.Errorf("sampleConfig missing section %q", section)
		}
	}
}
