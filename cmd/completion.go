package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
)

var completionCmd = &cobra.Command{
	Use:                   "completion [bash|zsh|fish|powershell]",
	Short:                 "Generate shell completion scripts",
	Args:                  cobra.ExactArgs(1),
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	ValidArgsFunction:     cobra.FixedCompletions([]string{"bash", "zsh", "fish", "powershell"}, cobra.ShellCompDirectiveNoFileComp),
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell %q", args[0])
		}
	},
}

func init() {
	useCmd.ValidArgsFunction = completeContextNames(true)
	removeCmd.ValidArgsFunction = completeContextNames(false)
	exportCmd.ValidArgsFunction = completeContextNames(false)
	renameCmd.ValidArgsFunction = renameContextCompletion
	nsCmd.ValidArgsFunction = namespaceArgCompletion
	statusCmd.ValidArgsFunction = completeContextNames(false)

	_ = currentCmd.RegisterFlagCompletionFunc("context", completeContextNames(false))
	_ = useCmd.RegisterFlagCompletionFunc("namespace", completeNamespaceFlag)

	rootCmd.AddCommand(completionCmd)
}

func completeContextNames(includePrevious bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		contexts, err := loadContextNames()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results := make([]string, 0, len(contexts)+1)
		if includePrevious {
			previous, err := infrastructure.ReadPrevious()
			if err == nil && previous != "" && strings.HasPrefix(previous, toComplete) {
				results = append(results, "-\tswitch to previous context")
			}
		}

		for _, contextName := range contexts {
			if strings.HasPrefix(contextName, toComplete) {
				results = append(results, contextName)
			}
		}

		return results, cobra.ShellCompDirectiveNoFileComp
	}
}

func renameContextCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completeContextNames(false)(cmd, args, toComplete)
}

func completeNamespaceFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	contextName := ""
	if len(args) > 0 {
		contextName = args[0]
	}

	return completeNamespaces(contextName)(cmd, args, toComplete)
}

func namespaceArgCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completeNamespaces("")(cmd, args, toComplete)
}

func completeNamespaces(contextName string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client := infrastructure.NewKubernetesClient(resolvedKubeconfigPath())
		namespaces, err := client.ListNamespacesForContext(contextName, 2*time.Second)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		results := make([]string, 0, len(namespaces))
		for _, namespace := range namespaces {
			if strings.HasPrefix(namespace, toComplete) {
				results = append(results, namespace)
			}
		}

		return results, cobra.ShellCompDirectiveNoFileComp
	}
}

func loadContextNames() ([]string, error) {
	repo := infrastructure.NewFileRepository()
	cfg, err := repo.Load(resolvedKubeconfigPath())
	if err != nil {
		return nil, err
	}

	return cfg.ContextNames(), nil
}

func resolvedKubeconfigPath() string {
	if strings.TrimSpace(kubeconfigPath) != "" {
		return kubeconfigPath
	}

	config.Init()
	return config.GetKubeconfigPath()
}
