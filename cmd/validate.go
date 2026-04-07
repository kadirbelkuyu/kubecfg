package cmd

import (
	"fmt"
	"os"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:           "validate [file]",
	Short:         "Validate a kubeconfig file",
	Long:          "Validate the kubeconfig structure and report broken references, duplicate entries, and missing required fields.",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := kubeconfigPath
		if len(args) == 1 {
			targetPath = args[0]
		}

		report, err := service.ValidateConfig(targetPath)
		if err != nil {
			printError(fmt.Errorf("failed to validate %s: %w", targetPath, err))
			return err
		}

		fmt.Println()
		fmt.Printf("  %s %s\n\n", ui.IconInfo, ui.Header("VALIDATION REPORT"))
		fmt.Printf("  %s %s\n", ui.Label("Path:     "), ui.Value(report.Path))
		fmt.Printf("  %s %s\n", ui.Label("Contexts: "), ui.Value(fmt.Sprintf("%d", report.ContextCount)))
		fmt.Printf("  %s %s\n", ui.Label("Clusters: "), ui.Value(fmt.Sprintf("%d", report.ClusterCount)))
		fmt.Printf("  %s %s\n", ui.Label("Users:    "), ui.Value(fmt.Sprintf("%d", report.UserCount)))

		if report.IsValid() {
			fmt.Printf("\n%s\n", ui.Success("Validation passed"))
			return nil
		}

		fmt.Fprintf(os.Stderr, "\n%s\n", ui.Error(fmt.Sprintf("Validation failed with %d issue(s)", len(report.Issues))))
		for _, issue := range report.Issues {
			fmt.Fprintf(os.Stderr, "  - %s\n", issue.String())
		}

		return domain.ErrInvalidConfig
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
