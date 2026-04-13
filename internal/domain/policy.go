package domain

import (
	"fmt"
	"regexp"
	"strings"
)

type PolicyProfile = string

const (
	PolicyProfileProd    PolicyProfile = "prod"
	PolicyProfileStaging PolicyProfile = "staging"
	PolicyProfileDebug   PolicyProfile = "debug"
)

type Policy struct {
	Name                string
	Description         string
	Readonly            bool
	ConfirmDestructive  bool
	BlockedVerbs        []string
	AllowedNamespaces   []string
	BlockedResources    []string
	WarnContextPatterns []string
}

func (p *Policy) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("policy name is required")
	}
	for _, pattern := range p.WarnContextPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid warn context pattern %q: %w", pattern, err)
		}
	}
	return nil
}

func (p *Policy) MatchesContext(contextName string) bool {
	for _, pattern := range p.WarnContextPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(contextName) {
			return true
		}
	}
	return false
}

func BuiltinProfiles() map[PolicyProfile]*Policy {
	return map[PolicyProfile]*Policy{
		PolicyProfileProd: {
			Name:                PolicyProfileProd,
			Description:         "Production guard: readonly access with secrets blocked",
			Readonly:            true,
			ConfirmDestructive:  true,
			BlockedVerbs:        []string{},
			AllowedNamespaces:   []string{},
			BlockedResources:    []string{"secrets"},
			WarnContextPatterns: []string{"prod", "production"},
		},
		PolicyProfileStaging: {
			Name:                PolicyProfileStaging,
			Description:         "Staging guard: write access with destructive operations blocked",
			Readonly:            false,
			ConfirmDestructive:  true,
			BlockedVerbs:        []string{},
			AllowedNamespaces:   []string{},
			BlockedResources:    []string{},
			WarnContextPatterns: []string{"staging", "stage"},
		},
		PolicyProfileDebug: {
			Name:                PolicyProfileDebug,
			Description:         "Debug guard: full access, warns on production contexts",
			Readonly:            false,
			ConfirmDestructive:  false,
			BlockedVerbs:        []string{},
			AllowedNamespaces:   []string{},
			BlockedResources:    []string{},
			WarnContextPatterns: []string{"prod", "production"},
		},
	}
}

func FindProfile(name PolicyProfile) (*Policy, error) {
	profiles := BuiltinProfiles()
	p, ok := profiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown policy profile %q: available profiles are prod, staging, debug", name)
	}
	return p, nil
}
