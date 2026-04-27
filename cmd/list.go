package cmd

import (
	"github.com/spf13/cobra"
)

var (
	listFilter      string
	listCluster     string
	listNamespace   string
	listCurrentOnly bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts",
	Long:  "Display all available contexts with cluster details.",
	Example: `  kubecfg list
  kubecfg list --current
  kubecfg list --filter prod
  kubecfg list --cluster eks`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		contexts, err := service.ListContexts(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		if len(contexts) == 0 {
			printInfo("No contexts found. Add a context with 'kubecfg add'")
			return
		}

		filters := contextFilters{
			query:       listFilter,
			cluster:     listCluster,
			namespace:   listNamespace,
			currentOnly: listCurrentOnly,
		}

		contexts = filterContexts(contexts, filters)
		if len(contexts) == 0 {
			if filters.active() {
				printInfo("No contexts matched the provided filters")
				return
			}

			printInfo("No contexts found. Add a context with 'kubecfg add'")
			return
		}

		printContextTable(contexts)
	},
}

func init() {
	listCmd.Flags().StringVarP(&listFilter, "filter", "f", "", "filter contexts by name, cluster, server, or namespace")
	listCmd.Flags().StringVar(&listCluster, "cluster", "", "filter by cluster name")
	listCmd.Flags().StringVar(&listNamespace, "namespace", "", "filter by namespace")
	listCmd.Flags().BoolVar(&listCurrentOnly, "current", false, "show only the current context")
	rootCmd.AddCommand(listCmd)
}
