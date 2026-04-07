package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var (
	guardTTL       time.Duration
	guardProxyID   string
	guardProxyPath string
)

var guardCmd = &cobra.Command{
	Use:   "guard",
	Short: "Manage guarded Kubernetes sessions",
	Long:  "Start, stop, and inspect readonly guarded Kubernetes sessions.",
}

var guardStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a readonly guard session",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		session, err := guardService.StartReadonly(application.GuardStartOptions{
			SourcePath: kubeconfigPath,
			TTL:        guardTTL,
		})
		if err != nil {
			printError(err)
			return
		}

		printGuardSession("Guard session started", session, "active")
	},
}

var guardStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the active guard session",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		session, err := guardService.Stop()
		if err != nil {
			if errors.Is(err, domain.ErrGuardSessionNotFound) {
				printInfo("No active guard session")
				return
			}
			printError(err)
			return
		}

		printInfo(fmt.Sprintf("Stopped guard session %s", session.ID))
	},
}

var guardStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current guard session status",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		status, err := guardService.Status()
		if err != nil {
			printError(err)
			return
		}

		if status.Session == nil {
			printInfo("No active guard session")
			return
		}

		printGuardSession("Guard session status", status.Session, status.Health)
	},
}

var guardProxyCmd = &cobra.Command{
	Use:    "proxy",
	Short:  "Run the local guard proxy",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		store := infrastructure.NewSessionFileStore(guardProxyPath)
		session, err := store.Load()
		if err != nil {
			printError(err)
			return
		}

		if guardProxyID != "" && session.ID != guardProxyID {
			printError(fmt.Errorf("guard session id mismatch"))
			return
		}

		proxy, err := infrastructure.NewGuardProxy(session, application.NewReadonlyRequestPolicy())
		if err != nil {
			printError(err)
			return
		}

		if err := proxy.Run(); err != nil {
			printError(err)
		}
	},
}

func init() {
	guardStartCmd.Flags().DurationVar(&guardTTL, "ttl", 0, "guard session ttl, for example 30m or 1h")
	guardProxyCmd.Flags().StringVar(&guardProxyID, "session-id", "", "guard session id")
	guardProxyCmd.Flags().StringVar(&guardProxyPath, "session-file", "", "guard session file path")
	_ = guardProxyCmd.Flags().MarkHidden("session-id")
	_ = guardProxyCmd.Flags().MarkHidden("session-file")

	guardCmd.AddCommand(guardStartCmd)
	guardCmd.AddCommand(guardStopCmd)
	guardCmd.AddCommand(guardStatusCmd)
	guardCmd.AddCommand(guardProxyCmd)
	rootCmd.AddCommand(guardCmd)
}

func printGuardSession(title string, session *domain.Session, health string) {
	if session == nil {
		printInfo("No active guard session")
		return
	}

	remaining := session.Remaining(time.Now())
	if health == "expired" {
		remaining = 0
	}

	var output strings.Builder

	_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n",
		ui.IconInfo, ui.Header(strings.ToUpper(title)))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Session ID:   "), ui.Value(session.ID))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Mode:         "), ui.Value(session.ModeDisplay()))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Health:       "), ui.Value(health))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Context:      "), ui.ContextName(session.TargetContext))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Namespace:    "), ui.Namespace(session.NamespaceDisplay()))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Proxy:        "), ui.Server(session.ProxyListenAddress))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Kubeconfig:   "), ui.Value(session.GeneratedKubeconfigPath))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Started At:   "), ui.Value(session.StartedAt.Format(time.RFC3339)))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Expires At:   "), ui.Value(session.ExpiresAt.Format(time.RFC3339)))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Remaining:    "), ui.Value(formatDuration(remaining)))
	_, _ = fmt.Fprintf(&output, "\n  %s %s\n", ui.Label("Export:       "), ui.Value("export KUBECONFIG="+session.GeneratedKubeconfigPath))

	fmt.Println(output.String())
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "expired"
	}
	return value.Round(time.Second).String()
}
