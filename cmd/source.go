package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/fzf"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var sourceCmd = &cobra.Command{
	Use:     "source",
	Aliases: []string{"sources"},
	Short:   "Manage kubeconfig source files",
	Long:    "Manage the kubeconfig file whose contexts kubecfg displays and modifies.",
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered kubeconfig source files",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		printKubeconfigSources(listKubeconfigSources())
	},
}

var sourceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active kubeconfig source file",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		printSuccess(fmt.Sprintf("Active kubeconfig source: %s", ui.Value(config.GetKubeconfigPath())))
	},
}

var sourceUseCmd = &cobra.Command{
	Use:   "use [file|name]",
	Short: "Switch the active kubeconfig source file",
	Long:  "Switch the active kubeconfig source file. Omit the argument to choose from discovered files.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path, err := resolveKubeconfigSourceSelection(args)
		if err != nil {
			printError(err)
			return
		}
		if path == "" {
			return
		}

		if err := sourceService().ValidateKubeconfigSource(path); err != nil {
			printError(fmt.Errorf("invalid kubeconfig source: %w", err))
			return
		}
		if err := config.SetActiveKubeconfigPath(path); err != nil {
			printError(err)
			return
		}
		kubeconfigPath = config.GetKubeconfigPath()

		printSuccess(fmt.Sprintf("Active kubeconfig source: %s", ui.Value(kubeconfigPath)))
	},
}

var sourceDirCmd = &cobra.Command{
	Use:   "dir",
	Short: "Manage directories scanned for kubeconfig files",
}

var sourceDirListCmd = &cobra.Command{
	Use:   "list",
	Short: "List source directories",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		for _, dir := range config.GetKubeconfigSourceDirs() {
			fmt.Println(dir)
		}
	},
}

var sourceDirAddCmd = &cobra.Command{
	Use:   "add [dir]",
	Short: "Add a directory to scan for kubeconfig files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.AddKubeconfigSourceDir(args[0]); err != nil {
			printError(err)
			return
		}
		printSuccess(fmt.Sprintf("Kubeconfig source directory added: %s", ui.Value(config.ExpandPath(args[0]))))
	},
}

var sourceDirRemoveCmd = &cobra.Command{
	Use:     "remove [dir]",
	Aliases: []string{"rm"},
	Short:   "Remove a directory from kubeconfig source scanning",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.RemoveKubeconfigSourceDir(args[0]); err != nil {
			printError(err)
			return
		}
		printSuccess(fmt.Sprintf("Kubeconfig source directory removed: %s", ui.Value(config.ExpandPath(args[0]))))
	},
}

func listKubeconfigSources() []application.KubeconfigSourceInfo {
	return sourceService().ListKubeconfigSources(config.GetKubeconfigPath(), config.GetKubeconfigSourceDirs())
}

func sourceService() *application.Service {
	if service != nil {
		return service
	}

	config.Init()
	return application.NewService(infrastructure.NewFileRepository())
}

func resolveKubeconfigSourceSelection(args []string) (string, error) {
	sources := listKubeconfigSources()
	if len(args) == 0 {
		return selectKubeconfigSourceInteractive(sources), nil
	}

	value := strings.TrimSpace(args[0])
	if value == "" {
		return "", nil
	}

	expanded := config.ExpandPath(value)
	for _, source := range sources {
		if source.Path == expanded || source.Name == value {
			return source.Path, nil
		}
	}

	return expanded, nil
}

func selectKubeconfigSourceInteractive(sources []application.KubeconfigSourceInfo) string {
	available := make([]application.KubeconfigSourceInfo, 0, len(sources))
	for _, source := range sources {
		if source.Error == "" && source.ContextCount > 0 {
			available = append(available, source)
		}
	}
	if len(available) == 0 {
		printWarning("No kubeconfig sources found")
		return ""
	}

	items := make([]string, 0, len(available))
	byLabel := make(map[string]string, len(available))
	var currentIdx int
	for i, source := range available {
		label := fmt.Sprintf("%s (%d contexts)", source.Path, source.ContextCount)
		if source.Active {
			currentIdx = i
			label += " [active]"
		}
		items = append(items, label)
		byLabel[label] = source.Path
	}

	if fzf.Available() {
		selected, err := fzf.Select(items, fzf.Options{
			Prompt: "source> ",
			Header: "Select kubeconfig source",
		})
		if err == nil {
			return byLabel[selected]
		}
		if err == fzf.ErrAborted {
			return ""
		}
	}

	prompt := promptui.Select{
		Label:     "Select kubeconfig source",
		Items:     items,
		CursorPos: currentIdx,
		Size:      10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   fmt.Sprintf("\033[33m%s {{ . }}\033[0m", ui.IconCurrent),
			Inactive: "  {{ . }}",
			Selected: fmt.Sprintf("\033[32m%s {{ . }}\033[0m", ui.IconCheck),
		},
		HideHelp: true,
		Stdout:   os.Stderr,
	}

	_, result, err := prompt.Run()
	if err != nil {
		if err != promptui.ErrInterrupt {
			printError(err)
		}
		return ""
	}

	return byLabel[result]
}

func printKubeconfigSources(sources []application.KubeconfigSourceInfo) {
	if len(sources) == 0 {
		printInfo("No kubeconfig sources found. Add a directory with 'kubecfg source dir add <path>'")
		return
	}

	fmt.Printf("  %-3s  %-50s  %-8s  %-30s  %s\n",
		ui.Header(""),
		ui.Header("PATH"),
		ui.Header("CONTEXTS"),
		ui.Header("CURRENT"),
		ui.Header("STATUS"),
	)
	fmt.Printf("  %s\n", strings.Repeat("─", 112))
	for _, source := range sources {
		active := ""
		if source.Active {
			active = ui.CurrentIndicator()
		}

		status := ui.Success("ok")
		if source.Error != "" {
			status = ui.Error(source.Error)
		}

		current := source.CurrentContext
		if current == "" {
			current = "-"
		}

		fmt.Printf("  %-3s  %-50s  %-8d  %-30s  %s\n",
			active,
			ui.Value(source.Path),
			source.ContextCount,
			current,
			status,
		)
	}
}

func completeKubeconfigSources(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	sources := listKubeconfigSources()
	results := make([]string, 0, len(sources))
	for _, source := range sources {
		if source.Error != "" {
			continue
		}
		if strings.HasPrefix(source.Path, toComplete) || strings.HasPrefix(source.Name, toComplete) {
			results = append(results, source.Path)
		}
	}
	return results, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	sourceUseCmd.ValidArgsFunction = completeKubeconfigSources

	sourceDirCmd.AddCommand(sourceDirListCmd, sourceDirAddCmd, sourceDirRemoveCmd)
	sourceCmd.AddCommand(sourceListCmd, sourceCurrentCmd, sourceUseCmd, sourceDirCmd)
	rootCmd.AddCommand(sourceCmd)
}
