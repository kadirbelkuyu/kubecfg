package group

import (
	"fmt"
	"regexp"
	"strings"
)

var groupNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

var allowedColors = map[string]struct{}{
	"red":     {},
	"yellow":  {},
	"green":   {},
	"blue":    {},
	"cyan":    {},
	"magenta": {},
}

// Group is a named collection of context names.
type Group struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Policy      string   `yaml:"policy,omitempty"`
	Contexts    []string `yaml:"contexts"`
	Color       string   `yaml:"color,omitempty"`
}

// Validate checks whether the group is structurally valid.
func (g Group) Validate() error {
	name := strings.TrimSpace(g.Name)
	if !groupNamePattern.MatchString(name) {
		return ErrInvalidGroupName
	}

	if len(g.Contexts) == 0 {
		return ErrEmptyContextList
	}

	for _, contextName := range g.Contexts {
		if strings.TrimSpace(contextName) == "" {
			return fmt.Errorf("group contains an empty context name")
		}
	}

	if color := strings.TrimSpace(g.Color); color != "" {
		if _, ok := allowedColors[color]; !ok {
			return fmt.Errorf("invalid group color %q", g.Color)
		}
	}

	return nil
}
