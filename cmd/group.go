package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	appgroupservice "github.com/kadirbelkuyu/kubecfg/internal/application/groupservice"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var (
	groupCreateContexts    []string
	groupCreateDescription string
	groupCreateColor       string
	groupListWide          bool
	groupDeleteForce       bool
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage context groups",
	Long:  "Create, inspect, and use named groups of kubeconfig contexts.",
	Example: `  kubecfg group create prod --contexts eks-prod,gke-prod --color red
  kubecfg group list
  kubecfg group use prod`,
}

var groupCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a context group",
	Example: `  kubecfg group create prod --contexts eks-prod,gke-prod
  kubecfg group create staging --contexts aks-stage --description "Staging clusters"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runGroupCreate(args[0], groupCreateContexts, groupCreateDescription, groupCreateColor); err != nil {
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

func runGroupCreate(name string, contexts []string, description, color string) error {
	g := domaingroup.Group{
		Name:        strings.TrimSpace(name),
		Description: description,
		Contexts:    append([]string(nil), contexts...),
		Color:       color,
	}

	if err := groupService.Create(g); err != nil {
		return formatGroupError(err, name, "")
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

	if err := service.UseContext(kubeconfigPath, selected, ""); err != nil {
		return err
	}

	printSuccess(fmt.Sprintf("Switched to context \"%s\" (group: %s)", selected, g.Name))
	return nil
}

func runGroupRename(oldName, newName string) error {
	if err := groupService.Rename(oldName, newName); err != nil {
		return formatGroupError(err, oldName, "")
	}

	printSuccess(fmt.Sprintf("Group \"%s\" renamed to \"%s\".", oldName, newName))
	return nil
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
		_, _ = fmt.Fprintf(&output, "  %-20s  %-8s  %-80s\n", ui.Header("NAME"), ui.Header("CONTEXTS"), ui.Header("MEMBERS"))
		_, _ = fmt.Fprintf(&output, "  %s\n", strings.Repeat("─", 116))
		for _, g := range groups {
			_, _ = fmt.Fprintf(&output, "  %-20s  %-8d  %-80s\n", g.Name, len(g.Contexts), strings.Join(g.Contexts, ", "))
		}
		return output.String()
	}

	_, _ = fmt.Fprintf(&output, "  %-20s  %-8s  %-40s\n", ui.Header("NAME"), ui.Header("CONTEXTS"), ui.Header("DESCRIPTION"))
	_, _ = fmt.Fprintf(&output, "  %s\n", strings.Repeat("─", 74))
	for _, g := range groups {
		_, _ = fmt.Fprintf(&output, "  %-20s  %-8d  %-40s\n", g.Name, len(g.Contexts), g.Description)
	}

	return output.String()
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

	rootCmd.AddCommand(groupCmd)
}
