package cmd

import (
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current context",
	Long:  "Display the active context name.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		contexts, err := service.ListContexts(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		for _, ctx := range contexts {
			if ctx.Current {
				printSuccess(ctx.Name)
				return
			}
		}

		printSuccess("No current context set")
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
