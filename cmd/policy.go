package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage and inspect guard policy profiles",
	Long:  "List, inspect, validate, and scaffold policy profiles for kubecfg guard sessions.",
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available policy profiles",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		policies := policyService.ListPolicies()
		userProfiles := config.GetProfiles()

		var output strings.Builder
		_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n", ui.IconInfo, ui.Header("POLICY PROFILES"))
		_, _ = fmt.Fprintf(&output, "  %-12s  %-10s  %-8s  %-8s  %s\n",
			ui.Header("NAME"), ui.Header("SOURCE"), ui.Header("READONLY"), ui.Header("CONFIRM"), ui.Header("DESCRIPTION"))
		_, _ = fmt.Fprintf(&output, "  %s\n", strings.Repeat("─", 80))

		for _, p := range policies {
			_, isUser := userProfiles[p.Name]
			badge := ui.BuiltinBadge()
			if isUser {
				badge = ui.UserDefinedBadge()
			}
			name := ui.PolicyProfileName(p.Name)
			desc := p.Description
			if desc == "" {
				desc = "—"
			}
			_, _ = fmt.Fprintf(&output, "  %-12s  %-10s  %-8v  %-8v  %s\n",
				name, badge,
				ui.Value(fmt.Sprintf("%v", p.Readonly)),
				ui.Value(fmt.Sprintf("%v", p.ConfirmDestructive)),
				ui.Value(desc))
		}

		fmt.Println(output.String())
	},
}

var policyShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show full details for a policy profile",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		p, err := policyService.GetPolicy(name)
		if err != nil {
			printError(err)
			return
		}

		userProfiles := config.GetProfiles()
		_, isUser := userProfiles[name]
		badge := ui.BuiltinBadge()
		if isUser {
			badge = ui.UserDefinedBadge()
		}

		var output strings.Builder
		_, _ = fmt.Fprintf(&output, "\n  %s %s %s\n\n",
			ui.IconInfo, ui.Header("POLICY: "+strings.ToUpper(p.Name)), badge)

		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Name:                "), ui.PolicyProfileName(p.Name))
		if p.Description != "" {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Description:         "), ui.Value(p.Description))
		}
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Readonly:            "), ui.Value(fmt.Sprintf("%v", p.Readonly)))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Confirm Destructive: "), ui.Value(fmt.Sprintf("%v", p.ConfirmDestructive)))

		if len(p.BlockedResources) > 0 {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Blocked Resources:   "), ui.Value(strings.Join(p.BlockedResources, ", ")))
		}
		if len(p.BlockedVerbs) > 0 {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Blocked Verbs:       "), ui.Value(strings.Join(p.BlockedVerbs, ", ")))
		}
		if len(p.AllowedNamespaces) > 0 {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Allowed Namespaces:  "), ui.Value(strings.Join(p.AllowedNamespaces, ", ")))
		}
		if len(p.WarnContextPatterns) > 0 {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Warn Patterns:       "), ui.Value(strings.Join(p.WarnContextPatterns, ", ")))
		}

		fmt.Println(output.String())
	},
}

var policyInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a sample ~/.kubecfg/config.yaml",
	Long:  "Writes a commented sample config file. Does not overwrite an existing file.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			printError(fmt.Errorf("get home dir: %w", err))
			return
		}

		dir := filepath.Join(home, ".kubecfg")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			printError(fmt.Errorf("create config directory: %w", err))
			return
		}

		dest := filepath.Join(dir, "config.yaml")
		if _, statErr := os.Stat(dest); statErr == nil {
			printWarning(fmt.Sprintf("config file already exists at %s — not overwriting", dest))
			return
		}

		if err := os.WriteFile(dest, []byte(sampleConfig), 0o600); err != nil {
			printError(fmt.Errorf("write config: %w", err))
			return
		}

		printSuccess(fmt.Sprintf("Created %s", dest))
	},
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate user-defined profiles in ~/.kubecfg/config.yaml",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		userProfiles := config.GetProfiles()
		if len(userProfiles) == 0 {
			printInfo("No user-defined profiles found in config (only builtins available)")
			return
		}

		allOK := true
		for name, cfg := range userProfiles {
			p := configProfileToDomainPolicy(name, cfg)
			if err := policyService.ValidatePolicy(p); err != nil {
				printError(fmt.Errorf("profile %q: %w", name, err))
				allOK = false
			} else {
				printSuccess(fmt.Sprintf("profile %q: ok", name))
			}
		}

		if !allOK {
			os.Exit(1)
		}
	},
}

// configProfileToDomainPolicy converts a config.ProfileConfig to a domain.Policy
// for the purpose of validation and display.
func configProfileToDomainPolicy(name string, cfg config.ProfileConfig) *domain.Policy {
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

const sampleConfig = `# kubecfg configuration
# Place this file at ~/.kubecfg/config.yaml
#
# Profiles defined here override the builtin profiles (prod, staging, debug)
# when matched by name.  Guard sessions will use these rules.

profiles:
  prod:
    description: "Production guard: readonly with critical resources blocked"
    readonly: true
    confirm_destructive: true
    blocked_resources:
      - namespaces
      - nodes
    warn_context_patterns:
      - prod
      - production

  staging:
    description: "Staging guard: write access, destructive ops require confirmation"
    readonly: false
    confirm_destructive: true
    warn_context_patterns:
      - staging
      - stage

  debug:
    description: "Debug guard: full access, warns on production contexts"
    readonly: false
    warn_context_patterns:
      - prod
      - production

sessions:
  default_ttl: 30m

audit:
  enabled: true
  path: ~/.kubecfg/audit.log
`

func init() {
	policyCmd.AddCommand(policyListCmd)
	policyCmd.AddCommand(policyShowCmd)
	policyCmd.AddCommand(policyInitCmd)
	policyCmd.AddCommand(policyValidateCmd)
	rootCmd.AddCommand(policyCmd)
}
