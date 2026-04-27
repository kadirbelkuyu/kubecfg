package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove [context-name]",
	Short: "Remove a context",
	Long:  "Delete a context and its associated entries.",
	Example: `  kubecfg remove old-dev
  kubecfg remove old-dev --force`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextName := args[0]

		if !removeForce {
			fmt.Printf("%s Remove context '%s'? [y/N]: ", ui.IconWarning, ui.ContextName(contextName))
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				printSuccess("Canceled")
				return
			}
		}

		if err := service.RemoveContext(kubeconfigPath, contextName); err != nil {
			printError(err)
			return
		}

		printSuccess(fmt.Sprintf("Context '%s' removed successfully", ui.ContextName(contextName)))
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}
