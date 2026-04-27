package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	appgroupservice "github.com/kadirbelkuyu/kubecfg/internal/application/groupservice"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var (
	groupCreateContexts    []string
	groupCreateDescription string
	groupCreateColor       string
	groupCreatePolicy      string
	groupListWide          bool
	groupDeleteForce       bool
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage context groups",
	Long:  "Create, inspect, and use named groups of kubeconfig contexts.",
	Example: `  kubecfg group create prod --contexts eks-prod,gke-prod --color red
  kubecfg group create prod --contexts eks-prod,gke-prod --policy prod
  kubecfg group list
  kubecfg group use prod`,
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a context group",
	Example: `  kubecfg group create prod --contexts eks-prod,gke-prod
  kubecfg group create staging --contexts aks-stage --description "Staging clusters"
  kubecfg group create prod --contexts eks-prod,gke-prod --policy prod`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupCreate(args[0], groupCreateContexts, groupCreateDescription, groupCreateColor, groupCreatePolicy); err != nil {
			printError(err)
		}
	},
}

var groupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List context groups",
	Example: `  kubecfg group list
  kubecfg group list --wide`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupList(groupListWide); err != nil {
			printError(err)
		}
	},
}

var groupShowCmd = &cobra.Command{
	Use:     "show <name>",
	Short:   "Show a context group",
	Example: `  kubecfg group show prod`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupShow(args[0]); err != nil {
			printError(err)
		}
	},
}

var groupAddCmd = &cobra.Command{
	Use:     "add <group-name> <context-name>",
	Short:   "Add a context to a group",
	Example: `  kubecfg group add prod eks-prod`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupAdd(args[0], args[1]); err != nil {
			printError(err)
		}
	},
}

var groupRemoveCmd = &cobra.Command{
	Use:     "remove <group-name> <context-name>",
	Short:   "Remove a context from a group",
	Example: `  kubecfg group remove prod old-prod`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupRemove(args[0], args[1]); err != nil {
			printError(err)
		}
	},
}

var groupDeleteCmd = &cobra.Command{
	Use:     "delete <name>",
	Short:   "Delete a context group",
	Example: `  kubecfg group delete prod --force`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupDelete(args[0], groupDeleteForce); err != nil {
			printError(err)
		}
	},
}

var groupUseCmd = &cobra.Command{
	Use:     "use <name>",
	Short:   "Switch to a context from a group",
	Example: `  kubecfg group use prod`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupUse(args[0]); err != nil {
			printError(err)
		}
	},
}

var groupRenameCmd = &cobra.Command{
	Use:     "rename <old-name> <new-name>",
	Short:   "Rename a context group",
	Example: `  kubecfg group rename prod production`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupRename(args[0], args[1]); err != nil {
			printError(err)
		}
	},
}

var groupSetPolicyCmd = &cobra.Command{
	Use:     "set-policy <group-name> <policy-name>",
	Short:   "Bind a policy profile to a context group",
	Example: `  kubecfg group set-policy prod prod`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupSetPolicy(args[0], args[1]); err != nil {
			printError(err)
		}
	},
}

var groupUnsetPolicyCmd = &cobra.Command{
	Use:     "unset-policy <group-name>",
	Short:   "Remove a policy binding from a context group",
	Example: `  kubecfg group unset-policy prod`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupUnsetPolicy(args[0]); err != nil {
			printError(err)
		}
	},
}

func runGroupCreate(name string, contexts []string, description, color, policy string) error {
	g := domaingroup.Group{
		Name:        strings.TrimSpace(name),
		Description: description,
		Policy:      policy,
		Contexts:    append([]string(nil), contexts...),
		Color:       color,
	}

	if err := groupService.Create(g); err != nil {
		return formatGroupError(err, name, "")
	}

	if strings.TrimSpace(policy) != "" {
		printSuccess(fmt.Sprintf("Group \"%s\" created with %d contexts and policy \"%s\".", g.Name, len(g.Contexts), strings.TrimSpace(policy)))
		return nil
	}

	printSuccess(fmt.Sprintf("Group \"%s\" created with %d contexts.", g.Name, len(g.Contexts)))
	return nil
}

func runGroupList(wide bool) error {
	groups, err := groupService.List()
	if err != nil {
		return err
	}

	if len(groups) == 0 {
		printInfo("No groups found. Create one with 'kubecfg group create'.")
		return nil
	}

	fmt.Print(renderGroupTable(groups, wide))
	return nil
}

func runGroupShow(name string) error {
	g, missing, err := groupService.Resolve(name)
	if err != nil {
		return formatGroupError(err, name, "")
	}

	fmt.Print(renderGroupDetails(g, missing))
	return nil
}

func runGroupAdd(groupName, contextName string) error {
	if err := groupService.AddContext(groupName, contextName); err != nil {
		return formatGroupError(err, groupName, contextName)
	}

	printSuccess(fmt.Sprintf("Context \"%s\" added to group \"%s\".", contextName, groupName))
	return nil
}

func runGroupRemove(groupName, contextName string) error {
	if err := groupService.RemoveContext(groupName, contextName); err != nil {
		return formatGroupError(err, groupName, contextName)
	}

	printSuccess(fmt.Sprintf("Context \"%s\" removed from group \"%s\".", contextName, groupName))
	return nil
}

func runGroupDelete(name string, force bool) error {
	if !force {
		return fmt.Errorf("group deletion requires --force. Re-run \"kubecfg group delete %s --force\"", name)
	}

	if err := groupService.Delete(name); err != nil {
		return formatGroupError(err, name, "")
	}

	printSuccess(fmt.Sprintf("Group \"%s\" deleted. (This did not affect your kubeconfig.)", name))
	return nil
}

func runGroupUse(name string) error {
	g, missing, err := groupService.Resolve(name)
	if err != nil {
		return formatGroupError(err, name, "")
	}

	printMissingGroupContexts(g.Name, missing)

	contexts, err := service.ListContexts(kubeconfigPath)
	if err != nil {
		return err
	}

	available := groupContextInfos(g, contexts, missing)
	if len(available) == 0 {
		return fmt.Errorf("group %q has no contexts present in your kubeconfig. Remove stale members with \"kubecfg group remove %s <context-name>\" or update ~/.kubecfg/groups.yaml", g.Name, g.Name)
	}

	selected := available[0].Name
	if len(available) > 1 {
		selected = selectContextInteractiveWithHeader(available, fmt.Sprintf("Group: %s", g.Name))
		if selected == "" {
			return nil
		}
	}

	var guardSession *domain.Session
	var previousPolicy string
	var hadActiveGuard bool
	if g.Policy != "" {
		guardSession, previousPolicy, hadActiveGuard, err = startGroupGuard(g, selected)
		if err != nil {
			return err
		}
	}

	if err := service.UseContext(kubeconfigPath, selected, ""); err != nil {
		if guardSession != nil {
			_, _ = guardService.Stop()
		}
		return err
	}

	printSuccess(fmt.Sprintf("Switched to context \"%s\" (group: %s)", selected, g.Name))
	if guardSession != nil {
		if hadActiveGuard && previousPolicy != guardSession.PolicyName {
			printInfo(fmt.Sprintf("Guard policy changed from %s to %s because group %s requires it", previousPolicy, guardSession.PolicyName, g.Name))
		}
		printSuccess(fmt.Sprintf("Guard activated with policy %s", guardSession.PolicyName))
	}
	return nil
}

func runGroupRename(oldName, newName string) error {
	if err := groupService.Rename(oldName, newName); err != nil {
		return formatGroupError(err, oldName, "")
	}

	printSuccess(fmt.Sprintf("Group \"%s\" renamed to \"%s\".", oldName, newName))
	return nil
}

func runGroupSetPolicy(groupName, policyName string) error {
	if err := groupService.SetPolicy(groupName, policyName); err != nil {
		return formatGroupError(err, groupName, "")
	}

	printSuccess(fmt.Sprintf("Policy \"%s\" bound to group \"%s\".", strings.TrimSpace(policyName), groupName))
	return nil
}

func runGroupUnsetPolicy(groupName string) error {
	if err := groupService.UnsetPolicy(groupName); err != nil {
		return formatGroupError(err, groupName, "")
	}

	printSuccess(fmt.Sprintf("Policy binding removed from group \"%s\".", groupName))
	return nil
}

func startGroupGuard(g domaingroup.Group, contextName string) (*domain.Session, string, bool, error) {
	status, err := guardService.Status()
	if err != nil {
		return nil, "", false, err
	}
	previousPolicy, wasActive := activeGuardPolicy(status)

	session, err := guardService.StartReadonly(application.GuardStartOptions{
		SourcePath:    kubeconfigPath,
		Profile:       g.Policy,
		TargetContext: contextName,
		ReplaceActive: true,
	})
	if err != nil {
		return nil, "", false, fmt.Errorf("activate guard for group %q with policy %q: %w", g.Name, g.Policy, err)
	}
	return session, previousPolicy, wasActive, nil
}

func activeGuardPolicy(status *application.GuardStatus) (string, bool) {
	if status == nil || !status.Active || status.Session == nil {
		return "", false
	}
	if status.Session.PolicyName != "" {
		return status.Session.PolicyName, true
	}
	return "readonly", true
}

func formatGroupError(err error, groupName, contextName string) error {
	var missingErr appgroupservice.MissingContextsError
	if errors.As(err, &missingErr) {
		if len(missingErr.Names) == 1 {
			return fmt.Errorf("context \"%s\" is not in your kubeconfig; run \"kubecfg list\" to inspect available contexts", missingErr.Names[0])
		}

		return fmt.Errorf("these contexts are not in your kubeconfig: %s; run \"kubecfg list\" to inspect available contexts", strings.Join(missingErr.Names, ", "))
	}

	switch {
	case errors.Is(err, domaingroup.ErrGroupAlreadyExists):
		return fmt.Errorf("group \"%s\" already exists; run \"kubecfg group list\" to inspect existing groups", groupName)
	case errors.Is(err, domaingroup.ErrGroupNotFound):
		return fmt.Errorf("group \"%s\" was not found; run \"kubecfg group list\" to see defined groups", groupName)
	case errors.Is(err, domaingroup.ErrEmptyContextList):
		if contextName != "" {
			return fmt.Errorf("group \"%s\" must contain at least one context; delete the group with \"kubecfg group delete %s --force\" if you want to remove it entirely", groupName, groupName)
		}
		return fmt.Errorf("group \"%s\" must contain at least one context", groupName)
	default:
		return err
	}
}

func renderGroupTable(groups []domaingroup.Group, wide bool) string {
	var output strings.Builder

	if wide {
		_, _ = fmt.Fprintf(&output, "  %-20s  %-8s  %-12s  %-80s\n", ui.Header("NAME"), ui.Header("CONTEXTS"), ui.Header("POLICY"), ui.Header("MEMBERS"))
		_, _ = fmt.Fprintf(&output, "  %s\n", strings.Repeat("─", 130))
		for _, g := range groups {
			_, _ = fmt.Fprintf(&output, "  %-20s  %-8d  %-12s  %-80s\n", g.Name, len(g.Contexts), displayGroupPolicy(g.Policy), strings.Join(g.Contexts, ", "))
		}
		return output.String()
	}

	_, _ = fmt.Fprintf(&output, "  %-20s  %-8s  %-12s  %-40s\n", ui.Header("NAME"), ui.Header("CONTEXTS"), ui.Header("POLICY"), ui.Header("DESCRIPTION"))
	_, _ = fmt.Fprintf(&output, "  %s\n", strings.Repeat("─", 88))
	for _, g := range groups {
		_, _ = fmt.Fprintf(&output, "  %-20s  %-8d  %-12s  %-40s\n", g.Name, len(g.Contexts), displayGroupPolicy(g.Policy), g.Description)
	}

	return output.String()
}

func displayGroupPolicy(policy string) string {
	if strings.TrimSpace(policy) == "" {
		return "-"
	}
	return policy
}

func renderGroupDetails(g domaingroup.Group, missing []string) string {
	var output strings.Builder
	missingSet := make(map[string]struct{}, len(missing))
	for _, name := range missing {
		missingSet[name] = struct{}{}
	}

	_, _ = fmt.Fprintf(&output, "Group: %s\n", g.Name)
	_, _ = fmt.Fprintf(&output, "Description: %s\n", g.Description)
	_, _ = fmt.Fprintf(&output, "Color: %s\n", g.Color)
	_, _ = fmt.Fprintf(&output, "Policy: %s\n", displayGroupPolicy(g.Policy))
	_, _ = fmt.Fprintf(&output, "Contexts (%d):\n", len(g.Contexts))

	for _, contextName := range g.Contexts {
		if _, missing := missingSet[contextName]; missing {
			_, _ = fmt.Fprintf(&output, "  %s %s (not found in kubeconfig)\n", ui.IconCross, contextName)
			continue
		}

		_, _ = fmt.Fprintf(&output, "  %s %s (reachable)\n", ui.IconCheck, contextName)
	}

	return output.String()
}

func groupContextInfos(g domaingroup.Group, contexts []application.ContextInfo, missing []string) []application.ContextInfo {
	missingSet := make(map[string]struct{}, len(missing))
	for _, name := range missing {
		missingSet[name] = struct{}{}
	}

	byName := make(map[string]application.ContextInfo, len(contexts))
	for _, context := range contexts {
		byName[context.Name] = context
	}

	filtered := make([]application.ContextInfo, 0, len(g.Contexts))
	for _, contextName := range g.Contexts {
		if _, isMissing := missingSet[contextName]; isMissing {
			continue
		}
		context, ok := byName[contextName]
		if !ok {
			continue
		}
		filtered = append(filtered, context)
	}

	return filtered
}

func printMissingGroupContexts(groupName string, missing []string) {
	for _, contextName := range missing {
		printWarning(fmt.Sprintf("Context \"%s\" is not in your kubeconfig. Run \"kubecfg group remove %s %s\" to clean up.", contextName, groupName, contextName))
	}
}

func init() {
	groupCreateCmd.Flags().StringSliceVar(&groupCreateContexts, "contexts", nil, "comma-separated context names")
	groupCreateCmd.Flags().StringVar(&groupCreateDescription, "description", "", "group description")
	groupCreateCmd.Flags().StringVar(&groupCreateColor, "color", "", "TUI color hint: red|yellow|green|blue|cyan|magenta")
	groupCreateCmd.Flags().StringVar(&groupCreatePolicy, "policy", "", "policy profile to activate when using the group")
	_ = groupCreateCmd.MarkFlagRequired("contexts")

	groupListCmd.Flags().BoolVar(&groupListWide, "wide", false, "show group members")
	groupDeleteCmd.Flags().BoolVar(&groupDeleteForce, "force", false, "delete without confirmation prompt")

	groupCmd.AddCommand(groupCreateCmd)
	groupCmd.AddCommand(groupListCmd)
	groupCmd.AddCommand(groupShowCmd)
	groupCmd.AddCommand(groupAddCmd)
	groupCmd.AddCommand(groupRemoveCmd)
	groupCmd.AddCommand(groupDeleteCmd)
	groupCmd.AddCommand(groupUseCmd)
	groupCmd.AddCommand(groupRenameCmd)
	groupCmd.AddCommand(groupSetPolicyCmd)
	groupCmd.AddCommand(groupUnsetPolicyCmd)

	rootCmd.AddCommand(groupCmd)
}
