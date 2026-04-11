package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
	"github.com/spf13/cobra"
)

var auditLimit int

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Inspect kubecfg audit events",
}

var auditTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Show recent audit events",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		auditStore := infrastructure.NewAuditFileStore(config.GetAuditPath())
		auditService := application.NewAuditService(auditStore, config.IsAuditEnabled())

		events, err := auditService.Recent(auditLimit)
		if err != nil {
			printError(err)
			return
		}

		if len(events) == 0 {
			printInfo("No audit events")
			return
		}

		var output strings.Builder
		_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n", ui.IconInfo, ui.Header("RECENT AUDIT EVENTS"))
		for _, event := range events {
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Time:   "), ui.Value(event.Timestamp.Format(time.RFC3339)))
			_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Type:   "), ui.Value(string(event.Type)))
			if event.SessionID != "" {
				_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Session:"), ui.Value(event.SessionID))
			}
			if event.Context != "" {
				_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Context:"), ui.ContextName(event.Context))
			}
			if event.Namespace != "" {
				_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("NS:    "), ui.Namespace(event.Namespace))
			}
			_, _ = fmt.Fprintf(&output, "  %s %s\n\n", ui.Label("Msg:   "), ui.Value(event.Message))
		}

		fmt.Println(output.String())
	},
}

func init() {
	auditTailCmd.Flags().IntVar(&auditLimit, "limit", 10, "number of recent audit events to show")
	auditCmd.AddCommand(auditTailCmd)
	rootCmd.AddCommand(auditCmd)
}
