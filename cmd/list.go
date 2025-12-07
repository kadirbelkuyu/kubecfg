package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts",
	Long:  "Display all available contexts with cluster details.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		contexts, err := service.ListContexts(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		if len(contexts) == 0 {
			printSuccess("No contexts found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CURRENT\tNAME\tCLUSTER\tSERVER\tNAMESPACE")

		for _, ctx := range contexts {
			current := ""
			if ctx.Current {
				current = "*"
			}
			namespace := ctx.Namespace
			if namespace == "" {
				namespace = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", current, ctx.Name, ctx.Cluster, ctx.Server, namespace)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
