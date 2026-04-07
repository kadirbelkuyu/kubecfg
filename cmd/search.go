package cmd

import "github.com/spf13/cobra"

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search contexts",
	Long:  "Search contexts by name, cluster, server, or namespace.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contexts, err := service.SearchContexts(kubeconfigPath, args[0])
		if err != nil {
			printError(err)
			return
		}

		if len(contexts) == 0 {
			printInfo("No contexts matched the search query")
			return
		}

		printContextTable(contexts)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
