package application

import (
	"fmt"
	"sort"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

// PolicyService resolves policy profiles, merging user-defined config with builtins.
// User-defined profiles (loaded from ~/.kubecfg/config.yaml) override builtins of the same name.
type PolicyService struct {
	profiles map[string]config.ProfileConfig
}

// NewPolicyService creates a PolicyService backed by the given user-defined profiles.
// Pass config.GetProfiles() as the argument; pass nil to fall back entirely to builtins.
func NewPolicyService(profiles map[string]config.ProfileConfig) *PolicyService {
	return &PolicyService{profiles: profiles}
}

// GetPolicy returns the policy for name, checking user config before builtins.
func (s *PolicyService) GetPolicy(name string) (*domain.Policy, error) {
	if name == "" {
		return nil, fmt.Errorf("policy name is required")
	}
	if cfg, ok := s.profiles[name]; ok {
		return profileConfigToPolicy(name, cfg), nil
	}
	return domain.FindProfile(name)
}

// ListPolicies returns all available policies: user-defined profiles first (overriding
// any builtin with the same name), then remaining builtins, sorted by name.
func (s *PolicyService) ListPolicies() []domain.Policy {
	builtins := domain.BuiltinProfiles()
	seen := make(map[string]bool)
	var result []domain.Policy

	for name, cfg := range s.profiles {
		seen[name] = true
		result = append(result, *profileConfigToPolicy(name, cfg))
	}

	for name, p := range builtins {
		if !seen[name] {
			result = append(result, *p)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// ValidatePolicy delegates to the domain validation.
func (s *PolicyService) ValidatePolicy(p *domain.Policy) error {
	return p.Validate()
}

// MergeWithSession applies the policy's mode/readonly flag onto a copy of the session
// and returns the merged copy. The original session is never mutated.
func (s *PolicyService) MergeWithSession(policy *domain.Policy, session *domain.Session) *domain.Session {
	if session == nil || policy == nil {
		return session
	}
	merged := *session
	merged.PolicyName = policy.Name
	merged.Readonly = policy.Readonly
	if policy.Readonly {
		merged.Mode = domain.GuardModeReadonly
	} else {
		merged.Mode = domain.GuardModePolicy
	}
	return &merged
}

// profileConfigToPolicy converts a config.ProfileConfig into a domain.Policy.
func profileConfigToPolicy(name string, cfg config.ProfileConfig) *domain.Policy {
	return &domain.Policy{
		Name:                name,
		Description:         cfg.Description,
		Readonly:            cfg.Readonly,
		ConfirmDestructive:  cfg.ConfirmDestructive,
		BlockedVerbs:        cfg.BlockedVerbs,
		AllowedNamespaces:   cfg.AllowedNamespaces,
		BlockedResources:    cfg.BlockedResources,
		WarnContextPatterns: cfg.WarnContextPatterns,
	}
}
