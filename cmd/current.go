package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current context",
	Long:  "Display the active context name.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		contexts, err := service.ListContexts(kubeconfigPath)
		if err != nil {
			printError(err)
			return
		}

		target, found := resolveCurrentTarget(contexts, currentContextFlag)
		if !found {
			if currentContextFlag != "" {
				printError(fmt.Errorf("context %q not found", currentContextFlag))
				return
			}
			printWarning("No current context set")
			return
		}

		status, err := guardService.Status()
		if err != nil {
			printError(err)
			return
		}

		fmt.Println(renderCurrentContext(target, status, currentContextFlag == ""))
	},
}

var currentContextFlag string

func resolveCurrentTarget(contexts []application.ContextInfo, contextName string) (application.ContextInfo, bool) {
	if contextName != "" {
		for _, ctx := range contexts {
			if ctx.Name == contextName {
				return ctx, true
			}
		}
		return application.ContextInfo{}, false
	}

	for _, ctx := range contexts {
		if ctx.Current {
			return ctx, true
		}
	}

	return application.ContextInfo{}, false
}

func renderCurrentContext(ctx application.ContextInfo, status *application.GuardStatus, isCurrent bool) string {
	var output strings.Builder
	title := "CONTEXT DETAILS"
	if isCurrent {
		title = "CURRENT CONTEXT"
	}

	_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n",
		ui.IconContext, ui.Header(title))

	_, _ = fmt.Fprintf(&output, "  %s %s\n",
		ui.Label("Context:  "), ui.ContextName(ctx.Name))

	_, _ = fmt.Fprintf(&output, "  %s %s\n",
		ui.Label("Cluster:  "), ui.Cluster(ctx.Cluster))

	namespace := ctx.Namespace
	if namespace == "" {
		namespace = "default"
	}
	_, _ = fmt.Fprintf(&output, "  %s %s\n",
		ui.Label("Namespace:"), ui.Namespace(namespace))

	_, _ = fmt.Fprintf(&output, "  %s %s\n",
		ui.Label("Server:   "), ui.Server(ctx.Server))

	_, _ = fmt.Fprintf(&output, "  %s %s\n",
		ui.Label("Guard:    "), ui.Value(guardStatusLabel(status, ctx.Name)))

	return output.String()
}

func guardStatusLabel(status *application.GuardStatus, contextName string) string {
	if status == nil || !status.Active || status.Session == nil || status.Session.TargetContext != contextName {
		return "inactive"
	}

	return fmt.Sprintf("%s (%s remaining)", status.Health, status.Remaining.Round(time.Second))
}

func init() {
	currentCmd.Flags().StringVar(&currentContextFlag, "context", "", "show details for a specific context without switching")
	rootCmd.AddCommand(currentCmd)
}
