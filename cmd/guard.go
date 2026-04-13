package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var (
	guardTTL        time.Duration
	guardProfile    string
	guardPolicyFlag string // --policy, alias for --profile
	guardProxyID    string
	guardProxyPath  string
)

var guardCmd = &cobra.Command{
	Use:   "guard",
	Short: "Manage guarded Kubernetes sessions",
	Long:  "Start, stop, and inspect readonly guarded Kubernetes sessions.",
}

var guardStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a guarded Kubernetes session",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// --policy is an alias for --profile; --profile takes precedence when both are given.
		profile := guardProfile
		if profile == "" {
			profile = guardPolicyFlag
		}

		session, err := guardService.StartReadonly(application.GuardStartOptions{
			SourcePath: kubeconfigPath,
			TTL:        guardTTL,
			Profile:    profile,
		})
		if err != nil {
			printError(err)
			return
		}

		if profile != "" {
			if p, ferr := policyService.GetPolicy(profile); ferr == nil && p.MatchesContext(session.TargetContext) {
				printWarning(fmt.Sprintf("context %q matches warn patterns for profile %q — proceed with caution", session.TargetContext, profile))
			}
		}

		status, err := guardService.Status()
		if err != nil {
			printError(err)
			return
		}

		printGuardStatus("Guard session started", status)
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

		printGuardStatus("Guard session status", status)
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

		auditStore := infrastructure.NewAuditFileStore(config.GetAuditPath())
		proxy, err := infrastructure.NewGuardProxy(session, policyForSession(session), auditStore)
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
	guardStartCmd.Flags().StringVar(&guardProfile, "profile", "", "policy profile to apply (prod, staging, debug)")
	guardStartCmd.Flags().StringVar(&guardPolicyFlag, "policy", "", "policy profile to apply (alias for --profile)")
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

func policyForSession(session *domain.Session) domain.GuardRequestPolicy {
	if session.PolicyName != "" {
		if p, err := policyService.GetPolicy(session.PolicyName); err == nil {
			return application.NewPolicyRequestPolicy(p)
		}
	}
	return application.NewReadonlyRequestPolicy()
}

func printGuardStatus(title string, status *application.GuardStatus) {
	if status == nil {
		printInfo("Guard status unavailable")
		return
	}

	session := status.Session
	remaining := session.Remaining(time.Now())
	if session == nil || status.Health == "expired" {
		remaining = 0
	}

	var output strings.Builder

	_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n",
		ui.IconInfo, ui.Header(strings.ToUpper(title)))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Health:       "), ui.Value(status.Health))
	if status.Detail != "" {
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Detail:       "), ui.Value(status.Detail))
	}
	if session != nil {
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Session ID:   "), ui.Value(session.ID))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Mode:         "), ui.Value(session.ModeDisplay()))
		if session.PolicyName != "" {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Profile:      "), ui.Value(session.PolicyName))
		}
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Context:      "), ui.ContextName(session.TargetContext))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Namespace:    "), ui.Namespace(session.NamespaceDisplay()))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Proxy:        "), ui.Server(session.ProxyListenAddress))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Kubeconfig:   "), ui.Value(session.GeneratedKubeconfigPath))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Started At:   "), ui.Value(session.StartedAt.Format(time.RFC3339)))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Expires At:   "), ui.Value(session.ExpiresAt.Format(time.RFC3339)))
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Remaining:    "), ui.Value(formatDuration(remaining)))
		_, _ = fmt.Fprintf(&output, "\n  %s %s\n", ui.Label("Export:       "), ui.Value("export KUBECONFIG="+session.GeneratedKubeconfigPath))
	}

	if len(status.RecentEvents) > 0 {
		_, _ = fmt.Fprintf(&output, "\n  %s\n", ui.Label("Recent Events:"))
		for _, event := range status.RecentEvents {
			_, _ = fmt.Fprintf(&output, "  %s %s %s\n",
				ui.Value(event.Timestamp.Format(time.RFC3339)),
				ui.Value(string(event.Type)),
				ui.Value(event.Message),
			)
		}
	}

	fmt.Println(output.String())
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "expired"
	}
	return value.Round(time.Second).String()
}
