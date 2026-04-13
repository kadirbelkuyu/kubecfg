package application

import (
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

// --- GetPolicy ---

func TestPolicyServiceGetPolicyBuiltinFallback(t *testing.T) {
	svc := NewPolicyService(nil)

	for _, name := range []string{"prod", "staging", "debug"} {
		t.Run(name, func(t *testing.T) {
			p, err := svc.GetPolicy(name)
			if err != nil {
				t.Fatalf("GetPolicy(%q) unexpected error: %v", name, err)
			}
			if p.Name != name {
				t.Fatalf("Name = %q, want %q", p.Name, name)
			}
		})
	}
}

func TestPolicyServiceGetPolicyUserOverride(t *testing.T) {
	profiles := map[string]config.ProfileConfig{
		"prod": {
			Readonly:    false,
			Description: "custom prod — write allowed",
		},
	}
	svc := NewPolicyService(profiles)

	p, err := svc.GetPolicy("prod")
	if err != nil {
		t.Fatalf("GetPolicy(prod) unexpected error: %v", err)
	}
	if p.Readonly {
		t.Fatal("user-defined prod should NOT be readonly")
	}
	if p.Description != "custom prod — write allowed" {
		t.Fatalf("Description = %q, unexpected", p.Description)
	}
}

func TestPolicyServiceGetPolicyUserCustomProfile(t *testing.T) {
	profiles := map[string]config.ProfileConfig{
		"ci": {
			Readonly:         true,
			BlockedResources: []string{"secrets"},
		},
	}
	svc := NewPolicyService(profiles)

	p, err := svc.GetPolicy("ci")
	if err != nil {
		t.Fatalf("GetPolicy(ci) unexpected error: %v", err)
	}
	if !p.Readonly {
		t.Fatal("ci profile should be Readonly")
	}
}

func TestPolicyServiceGetPolicyUnknown(t *testing.T) {
	svc := NewPolicyService(nil)
	_, err := svc.GetPolicy("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile, got nil")
	}
}

func TestPolicyServiceGetPolicyEmptyName(t *testing.T) {
	svc := NewPolicyService(nil)
	_, err := svc.GetPolicy("")
	if err == nil {
		t.Fatal("expected error for empty profile name, got nil")
	}
}

// --- ListPolicies ---

func TestPolicyServiceListPoliciesContainsAllBuiltins(t *testing.T) {
	svc := NewPolicyService(nil)
	policies := svc.ListPolicies()

	names := make(map[string]bool, len(policies))
	for _, p := range policies {
		names[p.Name] = true
	}

	for _, expected := range []string{"prod", "staging", "debug"} {
		if !names[expected] {
			t.Fatalf("ListPolicies() missing builtin %q", expected)
		}
	}
}

func TestPolicyServiceListPoliciesUserOverridesBuiltin(t *testing.T) {
	profiles := map[string]config.ProfileConfig{
		"prod": {Readonly: false, Description: "custom"},
	}
	svc := NewPolicyService(profiles)
	policies := svc.ListPolicies()

	for _, p := range policies {
		if p.Name == "prod" {
			if p.Readonly {
				t.Fatal("user-defined prod should NOT be Readonly in list")
			}
			return
		}
	}
	t.Fatal("prod not found in ListPolicies()")
}

func TestPolicyServiceListPoliciesUserCustomAppearsAlongsideBuiltins(t *testing.T) {
	profiles := map[string]config.ProfileConfig{
		"myprofile": {Readonly: true, Description: "custom profile"},
	}
	svc := NewPolicyService(profiles)
	policies := svc.ListPolicies()

	names := make(map[string]bool, len(policies))
	for _, p := range policies {
		names[p.Name] = true
	}

	if !names["myprofile"] {
		t.Fatal("user-defined profile should appear in ListPolicies()")
	}
	if !names["staging"] {
		t.Fatal("builtin staging should still appear in ListPolicies()")
	}
}

func TestPolicyServiceListPoliciesSortedByName(t *testing.T) {
	svc := NewPolicyService(nil)
	policies := svc.ListPolicies()

	for i := 1; i < len(policies); i++ {
		if policies[i].Name < policies[i-1].Name {
			t.Fatalf("ListPolicies() not sorted: %q comes after %q", policies[i].Name, policies[i-1].Name)
		}
	}
}

// --- ValidatePolicy ---

func TestPolicyServiceValidatePolicyValid(t *testing.T) {
	svc := NewPolicyService(nil)
	err := svc.ValidatePolicy(&domain.Policy{Name: "valid", WarnContextPatterns: []string{"^prod-.*$"}})
	if err != nil {
		t.Fatalf("ValidatePolicy() unexpected error: %v", err)
	}
}

func TestPolicyServiceValidatePolicyEmptyName(t *testing.T) {
	svc := NewPolicyService(nil)
	err := svc.ValidatePolicy(&domain.Policy{Name: ""})
	if err == nil {
		t.Fatal("ValidatePolicy() should fail for empty name")
	}
}

func TestPolicyServiceValidatePolicyInvalidPattern(t *testing.T) {
	svc := NewPolicyService(nil)
	err := svc.ValidatePolicy(&domain.Policy{Name: "x", WarnContextPatterns: []string{"[invalid"}})
	if err == nil {
		t.Fatal("ValidatePolicy() should fail for invalid regex")
	}
}

// --- MergeWithSession ---

func TestPolicyServiceMergeWithSessionReadonly(t *testing.T) {
	svc := NewPolicyService(nil)
	policy := &domain.Policy{Name: "prod", Readonly: true}
	original := &domain.Session{ID: "s1", Readonly: false, PolicyName: "old"}

	merged := svc.MergeWithSession(policy, original)

	if merged == original {
		t.Fatal("MergeWithSession should return a copy, not the same pointer")
	}
	if !merged.Readonly {
		t.Fatal("merged session should be Readonly")
	}
	if merged.PolicyName != "prod" {
		t.Fatalf("PolicyName = %q, want prod", merged.PolicyName)
	}
	if merged.Mode != domain.GuardModeReadonly {
		t.Fatalf("Mode = %q, want %q", merged.Mode, domain.GuardModeReadonly)
	}
	// original must not be mutated
	if original.Readonly {
		t.Fatal("original session must not be mutated")
	}
	if original.PolicyName != "old" {
		t.Fatal("original PolicyName must not be mutated")
	}
}

func TestPolicyServiceMergeWithSessionNonReadonly(t *testing.T) {
	svc := NewPolicyService(nil)
	policy := &domain.Policy{Name: "staging", Readonly: false}
	session := &domain.Session{ID: "s1"}

	merged := svc.MergeWithSession(policy, session)

	if merged.Mode != domain.GuardModePolicy {
		t.Fatalf("Mode = %q, want %q", merged.Mode, domain.GuardModePolicy)
	}
	if merged.Readonly {
		t.Fatal("merged staging session should not be Readonly")
	}
}

func TestPolicyServiceMergeWithSessionNilPolicy(t *testing.T) {
	svc := NewPolicyService(nil)
	session := &domain.Session{ID: "s1"}

	result := svc.MergeWithSession(nil, session)
	if result != session {
		t.Fatal("nil policy should return original session unchanged")
	}
}

func TestPolicyServiceMergeWithSessionNilSession(t *testing.T) {
	svc := NewPolicyService(nil)
	policy := &domain.Policy{Name: "prod"}

	result := svc.MergeWithSession(policy, nil)
	if result != nil {
		t.Fatal("nil session should return nil")
	}
}
