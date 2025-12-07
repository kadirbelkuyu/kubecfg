package cmd

import (
	"fmt"
	"os"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/spf13/cobra"
)

var (
	kubeconfigPath string
	service        *application.Service
)

var rootCmd = &cobra.Command{
	Use:   "kubecfg",
	Short: "Manage kubeconfig files",
	Long:  "Add, remove, merge and switch between Kubernetes contexts.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		repo := infrastructure.NewFileRepository()
		service = application.NewService(repo)
		if kubeconfigPath == "" {
			kubeconfigPath = service.GetDefaultPath()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "", "path to kubeconfig file")
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

func printSuccess(message string) {
	fmt.Println(message)
}
