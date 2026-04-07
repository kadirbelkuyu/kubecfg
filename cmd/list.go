package cmd

import (
	"fmt"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/ui"
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
			printInfo("No contexts found. Add a context with 'kubecfg add'")
			return
		}

		var output strings.Builder

		_, _ = fmt.Fprintf(&output, "  %s  %-30s  %-25s  %-40s  %-20s\n",
			ui.Header(""),
			ui.Header("NAME"),
			ui.Header("CLUSTER"),
			ui.Header("SERVER"),
			ui.Header("NAMESPACE"))

		_, _ = fmt.Fprintf(&output, "  %s\n",
			strings.Repeat("─", 120))

		for _, ctx := range contexts {
			current := "  "
			nameStyle := ctx.Name
			if ctx.Current {
				current = ui.CurrentIndicator()
				nameStyle = ui.ContextName(ctx.Name)
			}

			namespace := ctx.Namespace
			if namespace == "" {
				namespace = "default"
			}

			_, _ = fmt.Fprintf(&output, "  %s  %-30s  %-25s  %-40s  %-20s\n",
				current,
				nameStyle,
				ui.Cluster(ctx.Cluster),
				ui.Server(ctx.Server),
				ui.Namespace(namespace))
		}

		fmt.Println(output.String())
		fmt.Printf("\n%s\n", ui.Info(fmt.Sprintf("Total contexts: %d", len(contexts))))
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
