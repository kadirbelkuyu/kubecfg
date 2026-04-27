package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print version information",
	Long:    "Display the version of kubecfg.",
	Example: `  kubecfg version`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("kubecfg version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
