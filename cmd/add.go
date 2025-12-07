package cmd

import (
	"github.com/spf13/cobra"
)

var addName string

var addCmd = &cobra.Command{
	Use:   "add [file]",
	Short: "Add a kubeconfig file",
	Long:  "Import a kubeconfig file with a custom context name.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sourcePath := args[0]

		if addName == "" {
			printError(ErrNameRequired)
			return
		}

		if err := service.AddConfig(sourcePath, kubeconfigPath, addName); err != nil {
			printError(err)
			return
		}

		printSuccess("Context '" + addName + "' added successfully")
	},
}

func init() {
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "name for the new context (required)")
	addCmd.MarkFlagRequired("name")
	rootCmd.AddCommand(addCmd)
}
