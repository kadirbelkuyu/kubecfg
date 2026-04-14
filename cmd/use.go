package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/fzf"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

const interactiveNamespaceValue = "__interactive__"

var namespaceFlag string

var useCmd = &cobra.Command{
	Use:   "use [context-name]",
	Short: "Switch to a context",
	Long:  "Set the current context and optionally set the namespace.\nUse -n flag without value for interactive namespace selection.\nUse -n <namespace> to set a specific namespace.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runUse(cmd, args); err != nil {
			printError(err)
		}
	},
}

func runUse(cmd *cobra.Command, args []string) error {
	contextName, err := resolveContextName(args)
	if err != nil {
		return err
	}

	if contextName == "" {
		return nil
	}

	namespace := resolveNamespace(cmd, contextName)

	if err := service.UseContext(kubeconfigPath, contextName, namespace); err != nil {
		return err
	}

	printContextSwitchResult(contextName, namespace)
	return nil
}

func resolveContextName(args []string) (string, error) {
	if len(args) == 1 {
		if args[0] == "-" {
			previous, err := infrastructure.ReadPrevious()
			if err != nil {
				return "", err
			}

			if previous == "" {
				return "", fmt.Errorf("no previous context recorded")
			}

			return previous, nil
		}

		return args[0], nil
	}

	contexts, err := service.ListContexts(kubeconfigPath)
	if err != nil {
		return "", err
	}

	if len(contexts) == 0 {
		return "", fmt.Errorf("no contexts available")
	}

	return selectContextInteractive(contexts), nil
}

func selectContextInteractive(contexts []application.ContextInfo) string {
	items := make([]string, 0, len(contexts))
	var currentIdx int
	for i, ctx := range contexts {
		items = append(items, ctx.Name)
		if ctx.Current {
			currentIdx = i
		}
	}

	if fzf.Available() {
		selected, err := fzf.Select(items, fzf.Options{
			Prompt:  "context> ",
			Header:  "Select Kubernetes context",
			Preview: "kubecfg current --context {}",
		})
		if err == nil {
			return selected
		}
		if err == fzf.ErrAborted {
			return ""
		}
	}

	prompt := promptui.Select{
		Label:     ui.IconContext + " Select context",
		Items:     items,
		CursorPos: currentIdx,
		Size:      10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   fmt.Sprintf("\033[33m%s {{ . }}\033[0m", ui.IconCurrent),
			Inactive: "  {{ . }}",
			Selected: fmt.Sprintf("\033[32m%s {{ . }}\033[0m", ui.IconCheck),
		},
		HideHelp: true,
		Stdout:   os.Stderr,
	}

	_, result, err := prompt.Run()
	if err != nil {
		if err != promptui.ErrInterrupt {
			printError(err)
		}
		return ""
	}

	return result
}

func resolveNamespace(cmd *cobra.Command, contextName string) string {
	if !cmd.Flags().Changed("namespace") {
		return ""
	}

	if namespaceFlag == interactiveNamespaceValue {
		return selectNamespaceForContext(contextName)
	}

	return namespaceFlag
}

func printContextSwitchResult(contextName, namespace string) {
	if namespace != "" {
		printSuccess(fmt.Sprintf("Switched to context '%s' with namespace '%s'",
			ui.ContextName(contextName), ui.Namespace(namespace)))
		return
	}
	printSuccess(fmt.Sprintf("Switched to context '%s'", ui.ContextName(contextName)))
}

func selectNamespaceForContext(contextName string) string {
	currentNs, _ := service.GetContextNamespace(kubeconfigPath, contextName)

	k8sClient := infrastructure.NewKubernetesClient(kubeconfigPath)
	namespaces, err := k8sClient.ListNamespacesForContext(contextName, 10*time.Second)
	if err != nil {
		return currentNs
	}

	if len(namespaces) == 0 {
		return currentNs
	}

	skipOption := "[Skip - keep current]"
	items := append([]string{skipOption}, namespaces...)

	if fzf.Available() {
		selected, err := fzf.Select(items, fzf.Options{
			Prompt: "namespace> ",
			Header: "Select Kubernetes namespace",
		})
		if err == nil {
			if selected == skipOption {
				return ""
			}
			return selected
		}
		if err == fzf.ErrAborted {
			return currentNs
		}
	}

	var currentIdx int
	for i, ns := range items {
		if ns == currentNs {
			currentIdx = i
			break
		}
	}

	prompt := promptui.Select{
		Label:     ui.IconNamespace + " Select namespace",
		Items:     items,
		CursorPos: currentIdx,
		Size:      10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   fmt.Sprintf("\033[33m%s {{ . }}\033[0m", ui.IconCurrent),
			Inactive: "  {{ . }}",
			Selected: fmt.Sprintf("\033[32m%s {{ . }}\033[0m", ui.IconCheck),
		},
		HideHelp: true,
		Stdout:   os.Stderr,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return currentNs
	}

	if result == skipOption {
		return ""
	}

	return result
}

func init() {
	useCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", "", "set namespace (use -n without value for interactive selection)")
	useCmd.Flags().Lookup("namespace").NoOptDefVal = interactiveNamespaceValue
	rootCmd.AddCommand(useCmd)
}
