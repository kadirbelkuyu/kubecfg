package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mergeOutput string
var mergeConflictStrategy string

var mergeCmd = &cobra.Command{
	Use:   "merge [files...]",
	Short: "Merge kubeconfig files",
	Long:  "Combine multiple kubeconfig files into one.\nChoose how name conflicts are handled with --on-conflict: skip, overwrite, rename, fail.",
	Example: `  kubecfg merge prod.yaml staging.yaml --output merged.yaml
  kubecfg merge east.yaml west.yaml --on-conflict rename`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		outputPath := mergeOutput
		if outputPath == "" {
			outputPath = kubeconfigPath
		}

		if err := service.MergeConfigs(args, outputPath, mergeConflictStrategy); err != nil {
			printError(err)
			return
		}

		printSuccess(fmt.Sprintf("Merged %d configs into %s using '%s' strategy", len(args), outputPath, mergeConflictStrategy))
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&mergeOutput, "output", "o", "", "output kubeconfig file")
	mergeCmd.Flags().StringVar(&mergeConflictStrategy, "on-conflict", "skip", "conflict strategy: skip, overwrite, rename, fail")
	rootCmd.AddCommand(mergeCmd)
}
