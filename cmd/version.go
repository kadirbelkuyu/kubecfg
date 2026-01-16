package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Display the version of kubecfg.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kubecfg version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
