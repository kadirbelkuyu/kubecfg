package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/ui"
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
				var output strings.Builder

				_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n",
					ui.IconContext, ui.Header("CURRENT CONTEXT"))

				_, _ = fmt.Fprintf(&output, "  %s %s\n",
					ui.Label("Context:  "), ui.ContextName(ctx.Name))

				_, _ = fmt.Fprintf(&output, "  %s %s\n",
					ui.Label("Cluster:  "), ui.Cluster(ctx.Cluster))

				ns := ctx.Namespace
				if ns == "" {
					ns = "default"
				}
				_, _ = fmt.Fprintf(&output, "  %s %s\n",
					ui.Label("Namespace:"), ui.Namespace(ns))

				_, _ = fmt.Fprintf(&output, "  %s %s\n",
					ui.Label("Server:   "), ui.Server(ctx.Server))

				fmt.Println(output.String())
				return
			}
		}

		printWarning("No current context set")
	},
}

func init() {
	rootCmd.AddCommand(currentCmd)
}
