package cmd

import (
	"fmt"
	"os"

	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var namespaceFlag string

var useCmd = &cobra.Command{
	Use:   "use [context-name]",
	Short: "Switch to a context",
	Long:  "Set the current context and optionally set the namespace. Shows interactive picker if no context specified.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var contextName string

		if len(args) == 1 {
			contextName = args[0]
		} else {
			contexts, err := service.ListContexts(kubeconfigPath)
			if err != nil {
				printError(err)
				return
			}

			if len(contexts) == 0 {
				printError(fmt.Errorf("no contexts available"))
				return
			}

			var items []string
			var currentIdx int
			for i, ctx := range contexts {
				items = append(items, ctx.Name)
				if ctx.Current {
					currentIdx = i
				}
			}

			prompt := promptui.Select{
				Label:     "Select context",
				Items:     items,
				CursorPos: currentIdx,
				Size:      10,
				Templates: &promptui.SelectTemplates{
					Label:    "{{ . }}",
					Active:   "\033[33m▸ {{ . }}\033[0m",
					Inactive: "  {{ . }}",
					Selected: "\033[32m✓ {{ . }}\033[0m",
				},
				HideHelp: true,
				Stdout:   os.Stderr,
			}

			_, result, err := prompt.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					return
				}
				printError(err)
				return
			}
			contextName = result
		}

		namespace := namespaceFlag
		if namespace == "" && !cmd.Flags().Changed("namespace") {
			namespace = selectNamespaceForContext(contextName)
		}

		if err := service.UseContext(kubeconfigPath, contextName, namespace); err != nil {
			printError(err)
			return
		}

		if namespace != "" {
			printSuccess(fmt.Sprintf("Switched to context '%s' with namespace '%s'", contextName, namespace))
		} else {
			printSuccess("Switched to context '" + contextName + "'")
		}
	},
}

func selectNamespaceForContext(contextName string) string {
	currentNs, _ := service.GetContextNamespace(kubeconfigPath, contextName)

	k8sClient := infrastructure.NewKubernetesClient(kubeconfigPath)
	namespaces, err := k8sClient.ListNamespaces()
	if err != nil {
		return currentNs
	}

	if len(namespaces) == 0 {
		return currentNs
	}

	skipOption := "[Skip - keep current]"
	items := append([]string{skipOption}, namespaces...)

	var currentIdx int
	for i, ns := range items {
		if ns == currentNs {
			currentIdx = i
			break
		}
	}

	prompt := promptui.Select{
		Label:     "Select namespace",
		Items:     items,
		CursorPos: currentIdx,
		Size:      10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "\033[33m▸ {{ . }}\033[0m",
			Inactive: "  {{ . }}",
			Selected: "\033[32m✓ {{ . }}\033[0m",
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
	useCmd.Flags().StringVarP(&namespaceFlag, "namespace", "n", "", "set the namespace for the context")
	rootCmd.AddCommand(useCmd)
}
