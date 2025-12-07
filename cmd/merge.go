package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mergeOutput string

var mergeCmd = &cobra.Command{
	Use:   "merge [files...]",
	Short: "Merge kubeconfig files",
	Long:  "Combine multiple kubeconfig files into one.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		outputPath := mergeOutput
		if outputPath == "" {
			outputPath = kubeconfigPath
		}

		if err := service.MergeConfigs(args, outputPath); err != nil {
			printError(err)
			return
		}

		printSuccess(fmt.Sprintf("Merged %d configs into %s", len(args), outputPath))
	},
}

func init() {
	mergeCmd.Flags().StringVarP(&mergeOutput, "output", "o", "", "output file path")
	rootCmd.AddCommand(mergeCmd)
}
