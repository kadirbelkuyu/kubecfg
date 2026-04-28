package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	appgroupservice "github.com/kadirbelkuyu/kubecfg/internal/application/groupservice"
	"github.com/kadirbelkuyu/kubecfg/internal/application/healthservice"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/groupstore"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/healthcheck"
	"github.com/kadirbelkuyu/kubecfg/internal/tui"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var (
	kubeconfigPath string
	service        *application.Service
	groupService   *appgroupservice.Service
	guardService   *application.GuardService
	policyService  *application.PolicyService
	healthSvc      *healthservice.Service
)

var rootCmd = &cobra.Command{
	Use:   "kubecfg",
	Short: "Manage kubeconfig files",
	Long:  "Manage Kubernetes kubeconfig contexts, namespaces, groups, health checks, and guarded sessions.\n\nRun without arguments to launch the interactive TUI.",
	Example: `  kubecfg
  kubecfg use production
  kubecfg ns kube-system
  kubecfg status
  kubecfg guard start --ttl 30m --profile prod`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		config.Init()
		policyService = application.NewPolicyService(config.GetProfiles())
		repo := infrastructure.NewFileRepository()
		if kubeconfigPath == "" {
			kubeconfigPath = config.GetKubeconfigPath()
		}
		config.SetKubeconfigPath(kubeconfigPath)

		service = application.NewService(
			repo,
			application.WithPreviousContextStore(infrastructure.NewPreviousContextStore(config.GetLastContextPath())),
		)
		healthSvc = healthservice.New(
			healthcheck.New(kubeconfigPath),
			healthcheck.NewCache(),
			repo,
			kubeconfigPath,
		)
		groupService = appgroupservice.NewService(
			groupstore.NewFileStore(config.GetGroupsPath()),
			repo,
			kubeconfigPath,
			appgroupservice.WithPolicyResolver(policyService),
		)
		runtime, err := infrastructure.NewGuardProcessRuntime("", config.GetGuardSessionPath())
		if err != nil {
			printError(err)
			os.Exit(1)
		}
		auditStore := infrastructure.NewAuditFileStore(config.GetAuditPath())
		auditService := application.NewAuditService(auditStore, config.IsAuditEnabled())
		sessionStore := infrastructure.NewSessionFileStore(config.GetGuardSessionPath())
		kubeconfigWriter := infrastructure.NewGuardKubeconfigWriter()
		sessionService := application.NewSessionService(sessionStore, runtime, kubeconfigWriter, auditService)
		guardService = application.NewGuardService(
			repo,
			sessionService,
			kubeconfigWriter,
			runtime,
			auditService,
			filepath.Join(config.GetGuardStateDir(), "guard"),
			config.GetGuardDefaultTTL(),
			application.WithGuardPolicyResolver(policyService),
		)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.RunWithConfig(kubeconfigPath, healthSvc); err != nil {
			printError(err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var code exitCoder
		if errors.As(err, &code) {
			os.Exit(code.ExitCode())
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "", "path to the kubeconfig file")
}

func printError(err error) {
	fmt.Fprintln(os.Stderr, ui.Error(err.Error()))
}

func printSuccess(message string) {
	fmt.Println(ui.Success(message))
}

func printWarning(message string) {
	fmt.Fprintln(os.Stderr, ui.Warning(message))
}

func printInfo(message string) {
	fmt.Println(ui.Info(message))
}
