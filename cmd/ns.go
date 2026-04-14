package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/fzf"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var nsCmd = &cobra.Command{
	Use:   "ns [namespace]",
	Short: "Set namespace for current context",
	Long:  "Set the namespace for the current context. Shows interactive picker if no namespace specified.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var namespace string

		contexts, err := service.ListContexts(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		var currentCtx string
		var currentNs string
		for _, ctx := range contexts {
			if ctx.Current {
				currentCtx = ctx.Name
				currentNs = ctx.Namespace
				break
			}
		}

		if currentCtx == "" {
			printError(fmt.Errorf("no current context set"))
			return
		}

		if len(args) == 1 {
			namespace = args[0]
		} else {
			namespace = selectNamespace(currentNs)
			if namespace == "" {
				return
			}
		}

		if err := service.SetNamespace(kubeconfigPath, namespace); err != nil {
			printError(err)
			return
		}

		printSuccess(fmt.Sprintf("Namespace set to '%s'", ui.Namespace(namespace)))
	},
}

func selectNamespace(currentNs string) string {
	k8sClient := infrastructure.NewKubernetesClient(kubeconfigPath)
	namespaces, err := k8sClient.ListNamespaces()
	if err != nil {
		printWarning(fmt.Sprintf("Could not fetch namespaces from cluster: %v", err))
		return promptNamespaceManual(currentNs)
	}

	if len(namespaces) == 0 {
		return promptNamespaceManual(currentNs)
	}

	if fzf.Available() {
		selected, err := fzf.Select(namespaces, fzf.Options{
			Prompt: "namespace> ",
			Header: "Select Kubernetes namespace",
		})
		if err == nil {
			return selected
		}
		if err == fzf.ErrAborted {
			return ""
		}
	}

	var currentIdx int
	for i, ns := range namespaces {
		if ns == currentNs {
			currentIdx = i
			break
		}
	}

	prompt := promptui.Select{
		Label:     ui.IconNamespace + " Select namespace",
		Items:     namespaces,
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
		if err == promptui.ErrInterrupt {
			return ""
		}
		printError(err)
		return ""
	}

	return result
}

func promptNamespaceManual(currentNs string) string {
	templates := &promptui.PromptTemplates{
		Prompt:  fmt.Sprintf("%s {{ . }} ", ui.IconNamespace),
		Valid:   fmt.Sprintf("%s {{ . }} ", ui.IconNamespace),
		Invalid: fmt.Sprintf("%s {{ . }} ", ui.IconError),
		Success: fmt.Sprintf("%s {{ . }} ", ui.IconCheck),
	}

	prompt := promptui.Prompt{
		Label:     "Enter namespace",
		Default:   currentNs,
		Templates: templates,
	}

	result, err := prompt.Run()
	if err != nil {
		return ""
	}

	return result
}

var nsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current namespace",
	Long:  "Display the namespace of the current context.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		contexts, err := service.ListContexts(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		for _, ctx := range contexts {
			if ctx.Current {
				ns := ctx.Namespace
				if ns == "" {
					ns = "default"
				}
				fmt.Println(ui.Namespace(ns))
				return
			}
		}

		printError(fmt.Errorf("no current context set"))
	},
}

func init() {
	nsCmd.AddCommand(nsCurrentCmd)
	rootCmd.AddCommand(nsCmd)
}
