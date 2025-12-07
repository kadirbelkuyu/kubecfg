package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use [context-name]",
	Short: "Switch to a context",
	Long:  "Set the current context. Shows interactive picker if no context specified.",
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

		if err := service.UseContext(kubeconfigPath, contextName); err != nil {
			printError(err)
			return
		}

		printSuccess("Switched to context '" + contextName + "'")
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
