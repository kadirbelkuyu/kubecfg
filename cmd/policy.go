package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage guard policy profiles",
	Long:  "List, inspect, validate, and scaffold policy profiles for kubecfg guard sessions.",
	Example: `  kubecfg policy list
  kubecfg policy show prod
  kubecfg policy init
  kubecfg policy validate`,
}

var policyListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List policy profiles",
	Example: `  kubecfg policy list`,
	Args:    cobra.NoArgs,
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
	Short: "Show policy profile details",
	Example: `  kubecfg policy show prod
  kubecfg policy show staging`,
	Args: cobra.ExactArgs(1),
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
	Use:     "init",
	Short:   "Generate a sample config file",
	Long:    "Write a sample ~/.kubecfg/config.yaml file without overwriting an existing file.",
	Example: `  kubecfg policy init`,
	Args:    cobra.NoArgs,
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
	Use:     "validate",
	Short:   "Validate user-defined profiles",
	Long:    "Validate policy profiles defined in ~/.kubecfg/config.yaml.",
	Example: `  kubecfg policy validate`,
	Args:    cobra.NoArgs,
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

kubeconfig_sources:
  active: ~/.kube/config
  dirs:
    - ~/.kube
    - ~/team-kubeconfigs

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

var policyCreateFrom string

var policyCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a custom policy profile",
	Long:  "Print a new profile snippet derived from an existing profile.",
	Example: `  kubecfg policy create restricted --from prod
  kubecfg policy create qa --from staging`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if name == domain.PolicyProfileProd || name == domain.PolicyProfileStaging || name == domain.PolicyProfileDebug {
			printError(fmt.Errorf("cannot override builtin profile %q — choose a different name", name))
			return
		}

		base, err := policyService.GetPolicy(policyCreateFrom)
		if err != nil {
			printError(fmt.Errorf("base profile: %w", err))
			return
		}

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

		snippet := fmt.Sprintf(`
# Profile: %s (derived from %s)
# Add this under the "profiles:" key in %s

  %s:
    description: "%s (copy of %s)"
    readonly: %v
    confirm_destructive: %v
    blocked_resources: %s
    blocked_verbs: %s
    allowed_namespaces: %s
    warn_context_patterns: %s
`,
			name, policyCreateFrom, dest,
			name,
			base.Description, policyCreateFrom,
			base.Readonly,
			base.ConfirmDestructive,
			yamlStringSlice(base.BlockedResources),
			yamlStringSlice(base.BlockedVerbs),
			yamlStringSlice(base.AllowedNamespaces),
			yamlStringSlice(base.WarnContextPatterns),
		)

		printInfo(fmt.Sprintf("Add the following to %s:\n%s", dest, snippet))
	},
}

func yamlStringSlice(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	var sb strings.Builder
	for _, item := range items {
		_, _ = fmt.Fprintf(&sb, "\n      - %s", item)
	}
	return sb.String()
}

func init() {
	policyCreateCmd.Flags().StringVar(&policyCreateFrom, "from", "staging", "base profile: prod, staging, debug")
	policyCmd.AddCommand(policyListCmd)
	policyCmd.AddCommand(policyShowCmd)
	policyCmd.AddCommand(policyInitCmd)
	policyCmd.AddCommand(policyValidateCmd)
	policyCmd.AddCommand(policyCreateCmd)
	rootCmd.AddCommand(policyCmd)
}
