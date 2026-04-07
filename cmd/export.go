package cmd

import (
	"fmt"

	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export [context-name]",
	Short: "Export a context to a standalone kubeconfig",
	Long:  "Export the specified context to a new kubeconfig file. If no context name is provided, the current context is exported.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextName := ""
		if len(args) == 1 {
			contextName = args[0]
		}

		if exportOutput == "" {
			printError(fmt.Errorf("output file is required (use --output)"))
			return
		}

		if err := service.ExportContext(kubeconfigPath, contextName, exportOutput); err != nil {
			printError(err)
			return
		}

		exportedName := contextName
		if exportedName == "" {
			exportedName = "current context"
		}

		printSuccess(fmt.Sprintf("Exported '%s' to %s", ui.ContextName(exportedName), ui.Value(exportOutput)))
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output kubeconfig file")
	rootCmd.AddCommand(exportCmd)
}
