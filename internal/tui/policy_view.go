package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

// Policy profile accent colors — consistent with internal/ui/styles.go.
var (
	policyProdColor    = lipgloss.Color("#F59E0B") // amber/orange — prod
	policyStagingColor = lipgloss.Color("#EAB308") // yellow       — staging
	policyDebugColor   = lipgloss.Color("#22C55E") // green        — debug
	policyBuiltinColor = lipgloss.Color("#6C7086") // muted gray   — builtin badge
)

func policyNameStyle(name string) lipgloss.Style {
	switch name {
	case "prod":
		return lipgloss.NewStyle().Foreground(policyProdColor).Bold(true)
	case "staging":
		return lipgloss.NewStyle().Foreground(policyStagingColor).Bold(true)
	case "debug":
		return lipgloss.NewStyle().Foreground(policyDebugColor).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	}
}

func renderPolicyName(name string) string {
	return policyNameStyle(name).Render(name)
}

func renderPolicyBadge(isUserDefined bool) string {
	if isUserDefined {
		return lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("[user]")
	}
	return lipgloss.NewStyle().Foreground(policyBuiltinColor).Render("[builtin]")
}

// viewPolicyList renders the policy list panel for the given policies.
// isUserDefined maps profile name → whether it comes from user config.
func viewPolicyList(policies []domain.Policy, isUserDefined map[string]bool, cursor int) string {
	var b strings.Builder

	title := HeaderStyle.Render("≡ Policy Profiles")
	b.WriteString("\n " + title + "\n\n")

	if len(policies) == 0 {
		b.WriteString(" " + DimItemStyle.Render("No policies found") + "\n")
		return b.String()
	}

	maxVisible := 10
	start := 0
	if cursor >= maxVisible {
		start = cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(policies) {
		end = len(policies)
	}

	for i := start; i < end; i++ {
		p := policies[i]
		name := renderPolicyName(p.Name)
		badge := renderPolicyBadge(isUserDefined[p.Name])

		flags := buildPolicyFlags(p)
		flagStr := DimItemStyle.Render(flags)

		var line string
		if i == cursor {
			cur := SelectedItemStyle.Render(IconCurrent)
			line = fmt.Sprintf(" %s %s %s  %s", cur, name, badge, flagStr)
		} else {
			line = fmt.Sprintf("   %s %s  %s", name, badge, flagStr)
		}
		b.WriteString(line + "\n")
	}

	if len(policies) > maxVisible {
		b.WriteString("\n" + DimItemStyle.Render(fmt.Sprintf(" [%d/%d]", cursor+1, len(policies))))
	}

	return b.String()
}

// viewPolicyDetail renders the detail panel for a single policy.
func viewPolicyDetail(p *domain.Policy, isUserDefined bool) string {
	var b strings.Builder

	badge := renderPolicyBadge(isUserDefined)
	title := HeaderStyle.Render("≡ "+strings.ToUpper(p.Name)) + "  " + badge
	b.WriteString("\n " + title + "\n\n")

	field := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			DimItemStyle.Render(fmt.Sprintf("%-22s", label)),
			NormalItemStyle.Render(value)))
	}

	if p.Description != "" {
		field("Description:", p.Description)
	}
	field("Readonly:", fmt.Sprintf("%v", p.Readonly))
	field("Confirm Destructive:", fmt.Sprintf("%v", p.ConfirmDestructive))

	if len(p.BlockedResources) > 0 {
		field("Blocked Resources:", strings.Join(p.BlockedResources, ", "))
	}
	if len(p.BlockedVerbs) > 0 {
		field("Blocked Verbs:", strings.Join(p.BlockedVerbs, ", "))
	}
	if len(p.AllowedNamespaces) > 0 {
		field("Allowed Namespaces:", strings.Join(p.AllowedNamespaces, ", "))
	}
	if len(p.WarnContextPatterns) > 0 {
		field("Warn Patterns:", strings.Join(p.WarnContextPatterns, ", "))
	}

	return b.String()
}

// buildPolicyFlags returns a compact flag string for list display.
func buildPolicyFlags(p domain.Policy) string {
	var flags []string
	if p.Readonly {
		flags = append(flags, "readonly")
	}
	if p.ConfirmDestructive {
		flags = append(flags, "confirm-delete")
	}
	if len(p.BlockedResources) > 0 {
		flags = append(flags, fmt.Sprintf("blocked:%s", strings.Join(p.BlockedResources, ",")))
	}
	if len(p.AllowedNamespaces) > 0 {
		flags = append(flags, fmt.Sprintf("ns:%s", strings.Join(p.AllowedNamespaces, ",")))
	}
	if len(flags) == 0 {
		return "unrestricted"
	}
	return strings.Join(flags, "  ")
}
