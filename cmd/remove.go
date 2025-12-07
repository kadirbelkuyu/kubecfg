package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:   "remove [context-name]",
	Short: "Remove a context",
	Long:  "Delete a context and its associated entries.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextName := args[0]

		if !removeForce {
			fmt.Printf("Remove context '%s'? [y/N]: ", contextName)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				printSuccess("Cancelled")
				return
			}
		}

		if err := service.RemoveContext(kubeconfigPath, contextName); err != nil {
			printError(err)
			return
		}

		printSuccess("Context '" + contextName + "' removed successfully")
	},
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "skip confirmation")
	rootCmd.AddCommand(removeCmd)
}
