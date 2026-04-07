package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var (
	undoBackup string
	undoList   bool
	undoForce  bool
)

var undoCmd = &cobra.Command{
	Use:     "undo",
	Aliases: []string{"rollback"},
	Short:   "Restore kubeconfig from a backup",
	Long:    "Restore the latest kubeconfig backup by default, or list and restore a specific backup.",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if undoList && undoBackup != "" {
			printError(fmt.Errorf("--list and --to cannot be used together"))
			return
		}

		backups, err := service.ListBackups(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		if len(backups) == 0 {
			printWarning("No backups available")
			return
		}

		if undoList {
			printBackupList(backups)
			return
		}

		selected := backups[0]
		if undoBackup != "" {
			backup, found := findBackup(backups, undoBackup)
			if !found {
				printError(domain.ErrBackupNotFound)
				return
			}
			selected = backup
		}

		if !undoForce && !confirmRestore(selected.Name) {
			printSuccess("Cancelled")
			return
		}

		if err := service.RestoreBackup(kubeconfigPath, selected.Path); err != nil {
			printError(err)
			return
		}

		printSuccess(fmt.Sprintf("Restored kubeconfig from '%s'", ui.Value(selected.Name)))
	},
}

func printBackupList(backups []application.BackupInfo) {
	fmt.Println()
	fmt.Printf("  %s %s\n\n", ui.IconInfo, ui.Header("AVAILABLE BACKUPS"))

	for i, backup := range backups {
		marker := " "
		if i == 0 {
			marker = ui.IconCurrent
		}

		createdAt := "unknown"
		if !backup.CreatedAt.IsZero() {
			createdAt = backup.CreatedAt.Format("2006-01-02 15:04:05")
		}

		fmt.Printf("  %s %s  %s\n", marker, ui.Value(createdAt), filepath.Base(backup.Path))
	}

	fmt.Printf("\n%s\n", ui.Info("Use 'kubecfg undo --to <backup-name>' to restore a specific backup"))
}

func findBackup(backups []application.BackupInfo, target string) (application.BackupInfo, bool) {
	for _, backup := range backups {
		if backup.Path == target || backup.Name == target || filepath.Base(backup.Path) == target {
			return backup, true
		}
	}

	return application.BackupInfo{}, false
}

func confirmRestore(backupName string) bool {
	fmt.Printf("%s Restore kubeconfig from '%s'? [y/N]: ", ui.IconWarning, ui.Value(backupName))
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func init() {
	undoCmd.Flags().StringVar(&undoBackup, "to", "", "restore a specific backup by file path or backup name")
	undoCmd.Flags().BoolVar(&undoList, "list", false, "list available backups")
	undoCmd.Flags().BoolVarP(&undoForce, "force", "f", false, "skip confirmation")
	rootCmd.AddCommand(undoCmd)
}
