package domain

import "testing"

func TestPolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{
			name:    "valid policy",
			policy:  Policy{Name: "custom", WarnContextPatterns: []string{"prod"}},
			wantErr: false,
		},
		{
			name:    "missing name",
			policy:  Policy{Name: ""},
			wantErr: true,
		},
		{
			name:    "whitespace name",
			policy:  Policy{Name: "   "},
			wantErr: true,
		},
		{
			name:    "invalid regexp in warn patterns",
			policy:  Policy{Name: "x", WarnContextPatterns: []string{"[invalid"}},
			wantErr: true,
		},
		{
			name:    "valid regexp in warn patterns",
			policy:  Policy{Name: "x", WarnContextPatterns: []string{"^prod-.*$"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolicyMatchesContext(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		contextName string
		want        bool
	}{
		{name: "exact match", patterns: []string{"prod"}, contextName: "prod", want: true},
		{name: "substring match", patterns: []string{"prod"}, contextName: "my-production-cluster", want: true},
		{name: "prefix match", patterns: []string{"^prod"}, contextName: "prod-eu-west", want: true},
		{name: "no match", patterns: []string{"prod"}, contextName: "staging", want: false},
		{name: "multiple patterns first matches", patterns: []string{"prod", "staging"}, contextName: "prod", want: true},
		{name: "multiple patterns second matches", patterns: []string{"prod", "staging"}, contextName: "staging-eu", want: true},
		{name: "empty patterns", patterns: []string{}, contextName: "prod", want: false},
		{name: "nil patterns", patterns: nil, contextName: "prod", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Name: "test", WarnContextPatterns: tt.patterns}
			if got := p.MatchesContext(tt.contextName); got != tt.want {
				t.Fatalf("MatchesContext(%q) = %v, want %v", tt.contextName, got, tt.want)
			}
		})
	}
}

func TestBuiltinProfilesValid(t *testing.T) {
	profiles := BuiltinProfiles()

	expected := []PolicyProfile{PolicyProfileProd, PolicyProfileStaging, PolicyProfileDebug}
	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			p, ok := profiles[name]
			if !ok {
				t.Fatalf("BuiltinProfiles() missing profile %q", name)
			}
			if err := p.Validate(); err != nil {
				t.Fatalf("BuiltinProfiles()[%q].Validate() error = %v", name, err)
			}
			if p.Name != name {
				t.Fatalf("profile Name = %q, want %q", p.Name, name)
			}
		})
	}
}

func TestBuiltinProfilesBehavior(t *testing.T) {
	profiles := BuiltinProfiles()

	t.Run("prod is readonly", func(t *testing.T) {
		if !profiles[PolicyProfileProd].Readonly {
			t.Fatal("prod profile should be Readonly")
		}
	})

	t.Run("prod blocks secrets", func(t *testing.T) {
		p := profiles[PolicyProfileProd]
		found := false
		for _, r := range p.BlockedResources {
			if r == "secrets" {
				found = true
			}
		}
		if !found {
			t.Fatal("prod profile should block secrets resource")
		}
	})

	t.Run("prod warns on prod contexts", func(t *testing.T) {
		p := profiles[PolicyProfileProd]
		if !p.MatchesContext("production") {
			t.Fatal("prod profile should warn on production context")
		}
	})

	t.Run("staging is not readonly", func(t *testing.T) {
		if profiles[PolicyProfileStaging].Readonly {
			t.Fatal("staging profile should not be Readonly")
		}
	})

	t.Run("staging blocks destructive", func(t *testing.T) {
		if !profiles[PolicyProfileStaging].ConfirmDestructive {
			t.Fatal("staging profile should have ConfirmDestructive")
		}
	})

	t.Run("debug is permissive", func(t *testing.T) {
		p := profiles[PolicyProfileDebug]
		if p.Readonly {
			t.Fatal("debug profile should not be Readonly")
		}
		if p.ConfirmDestructive {
			t.Fatal("debug profile should not have ConfirmDestructive")
		}
		if len(p.BlockedVerbs) != 0 {
			t.Fatal("debug profile should have no BlockedVerbs")
		}
		if len(p.BlockedResources) != 0 {
			t.Fatal("debug profile should have no BlockedResources")
		}
	})
}

func TestFindProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{name: "prod found", profile: "prod", wantErr: false},
		{name: "staging found", profile: "staging", wantErr: false},
		{name: "debug found", profile: "debug", wantErr: false},
		{name: "unknown returns error", profile: "unknown", wantErr: true},
		{name: "empty returns error", profile: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := FindProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Fatalf("FindProfile(%q) error = %v, wantErr %v", tt.profile, err, tt.wantErr)
			}
			if !tt.wantErr && p == nil {
				t.Fatalf("FindProfile(%q) returned nil policy without error", tt.profile)
			}
		})
	}
}
