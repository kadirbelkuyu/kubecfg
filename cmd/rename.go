package cmd

import (
	"fmt"

	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename [old-name] [new-name]",
	Short: "Rename a context",
	Long:  "Change the name of an existing context.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldName := args[0]
		newName := args[1]

		if err := service.RenameContext(kubeconfigPath, oldName, newName); err != nil {
			printError(err)
			return
		}

		printSuccess(fmt.Sprintf("Context '%s' renamed to '%s'", ui.ContextName(oldName), ui.ContextName(newName)))
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
