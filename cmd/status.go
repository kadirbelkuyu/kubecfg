package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/application/healthservice"
	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/healthcheck"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var suggestions = map[string]string{
	"connection refused": "Is the cluster running locally? (e.g. minikube start / kind create cluster)",
	"TLS":                "Certificate may be expired or the CA has changed. Re-download the kubeconfig.",
	"no such host":       "DNS resolution failed. Check your VPN or network connection.",
	"i/o timeout":        "The API server is not responding. Check firewall rules or VPN.",
}

var (
	statusContext string
	statusWatch   bool
	statusTimeout time.Duration
	statusOutput  string
)

var newStatusHealthServiceFn = newStatusHealthService

var statusCmd = &cobra.Command{
	Use:           "status [context-name]",
	Short:         "Check context health",
	Long:          "Check one or more Kubernetes contexts and report API server reachability.",
	Example:       "  kubecfg status\n  kubecfg status production\n  kubecfg status --watch\n  kubecfg status --output json",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := statusContext
		if len(args) == 1 {
			target = args[0]
		}

		if strings.TrimSpace(statusOutput) == "" {
			statusOutput = "table"
		}
		if statusTimeout <= 0 {
			statusTimeout = health.CheckTimeout
		}

		statusHealthSvc := newStatusHealthServiceFn(statusTimeout)
		if target != "" {
			return runSingleStatusCheck(statusHealthSvc, target, statusOutput)
		}

		if statusWatch {
			return runStatusWatch(statusHealthSvc, statusOutput)
		}

		return runStatusOnce(statusHealthSvc, statusOutput)
	},
}

func init() {
	statusCmd.ValidArgsFunction = completeContextNames(false)
	statusCmd.Flags().StringVar(&statusContext, "context", "", "check a single context instead of all")
	statusCmd.Flags().BoolVar(&statusWatch, "watch", false, "repeat checks every 10 seconds until interrupted")
	statusCmd.Flags().DurationVar(&statusTimeout, "timeout", health.CheckTimeout, "per-check timeout")
	statusCmd.Flags().StringVar(&statusOutput, "output", "table", "output format: table, json, yaml")
	_ = statusCmd.RegisterFlagCompletionFunc("context", completeContextNames(false))
	_ = statusCmd.RegisterFlagCompletionFunc("output", cobra.FixedCompletions([]string{"table", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp))
	rootCmd.AddCommand(statusCmd)
}

func newStatusHealthService(timeout time.Duration) *healthservice.Service {
	repo := infrastructure.NewFileRepository()
	return healthservice.New(
		healthcheck.NewWithTimeout(resolvedKubeconfigPath(), timeout),
		healthcheck.NewCache(),
		repo,
		resolvedKubeconfigPath(),
	)
}

func runSingleStatusCheck(statusHealthSvc *healthservice.Service, contextName, output string) error {
	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout+time.Second)
	defer cancel()

	contexts, err := service.ListContexts(resolvedKubeconfigPath())
	if err != nil {
		return err
	}

	info, found := findContextInfo(contexts, contextName)
	if !found {
		return fmt.Errorf("context %q not found", contextName)
	}

	result, err := statusHealthSvc.CheckContext(ctx, contextName)
	if err != nil {
		return err
	}

	if output == "json" || output == "yaml" {
		return writeStructuredOutput(output, result)
	}

	fmt.Println(renderSingleStatus(info, result))
	if exitCodeForResults([]health.Result{result}) != 0 {
		return exitError{code: 1}
	}
	return nil
}

func runStatusOnce(statusHealthSvc *healthservice.Service, output string) error {
	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout+time.Second)
	defer cancel()

	contexts, err := service.ListContexts(resolvedKubeconfigPath())
	if err != nil {
		return err
	}

	results := runChecksForContexts(ctx, statusHealthSvc, contextNamesFromInfo(contexts))

	if output == "json" || output == "yaml" {
		sort.Slice(results, func(i, j int) bool {
			return results[i].ContextName < results[j].ContextName
		})
		if err := writeStructuredOutput(output, results); err != nil {
			return err
		}
	} else {
		fmt.Print(renderStatusTable(results))
	}

	if exitCodeForResults(results) != 0 {
		return exitError{code: 1}
	}
	return nil
}

func runStatusWatch(statusHealthSvc *healthservice.Service, output string) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), statusTimeout+time.Second)
		results, err := statusHealthSvc.CheckAll(ctx)
		cancel()
		if err != nil {
			return err
		}

		if output == "json" || output == "yaml" {
			if err := writeStructuredOutput(output, results); err != nil {
				return err
			}
		} else {
			fmt.Print("\033[H\033[2J")
			fmt.Print(renderStatusTable(results))
		}

		<-ticker.C
	}
}

func runChecksForContexts(ctx context.Context, statusHealthSvc *healthservice.Service, names []string) []health.Result {
	resultsCh := make(chan health.Result, len(names))
	var wg sync.WaitGroup
	sem := make(chan struct{}, health.MaxConcurrency)

	for _, name := range names {
		wg.Add(1)
		go func(contextName string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				resultsCh <- health.Result{ContextName: contextName, Status: health.StatusUnknown}
				return
			}
			defer func() { <-sem }()

			result, _ := statusHealthSvc.CheckContext(ctx, contextName)
			resultsCh <- result
		}(name)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	results := make([]health.Result, 0, len(names))
	if statusOutput == "table" {
		fmt.Print(renderStatusHeader())
	}
	for result := range resultsCh {
		results = append(results, result)
		if statusOutput == "table" {
			fmt.Print(renderStatusRow(result))
		}
	}
	if statusOutput == "table" {
		fmt.Println()
	}

	return results
}

func writeStructuredOutput(output string, value any) error {
	switch output {
	case "json":
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
		fmt.Println(string(data))
		return nil
	case "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode yaml: %w", err)
		}
		fmt.Print(string(data))
		return nil
	case "table":
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", output)
	}
}

func renderStatusHeader() string {
	return fmt.Sprintf("  %-30s  %-14s  %-10s  %-12s\n  %s\n",
		ui.Header("CONTEXT"),
		ui.Header("STATUS"),
		ui.Header("LATENCY"),
		ui.Header("CHECKED"),
		strings.Repeat("─", 78),
	)
}

func renderStatusTable(results []health.Result) string {
	sort.Slice(results, func(i, j int) bool {
		return results[i].ContextName < results[j].ContextName
	})

	var output strings.Builder
	output.WriteString(renderStatusHeader())
	for _, result := range results {
		output.WriteString(renderStatusRow(result))
	}
	output.WriteString("\n")
	return output.String()
}

func renderStatusRow(result health.Result) string {
	return fmt.Sprintf("  %-30s  %-14s  %-10s  %-12s\n",
		result.ContextName,
		renderStatusValue(result),
		renderLatency(result),
		renderChecked(result),
	)
}

func renderSingleStatus(info application.ContextInfo, result health.Result) string {
	var output strings.Builder

	_, _ = fmt.Fprintf(&output, "\n  %s %s\n\n", ui.IconInfo, ui.Header("CONTEXT HEALTH"))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Context:  "), ui.ContextName(info.Name))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Server:   "), ui.Server(info.Server))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Status:   "), ui.Value(renderStatusValue(result)))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Latency:  "), ui.Value(renderLatency(result)))
	_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Checked:  "), ui.Value(result.CheckedAt.Format("2006-01-02 15:04:05")))

	if result.Error != "" {
		_, _ = fmt.Fprintf(&output, "  %s %s\n", ui.Label("Error:    "), ui.Error(result.Error))
		if suggestion := suggestionForError(result.Error); suggestion != "" {
			_, _ = fmt.Fprintf(&output, "\n  %s %s\n", ui.Label("Suggestion:"), ui.Value(suggestion))
		}
	}

	return output.String()
}

func renderStatusValue(result health.Result) string {
	return result.Status.Emoji() + " " + result.Status.String()
}

func renderLatency(result health.Result) string {
	if result.Latency <= 0 {
		return "—"
	}
	if result.Latency < time.Second {
		return result.Latency.Round(time.Millisecond).String()
	}
	return result.Latency.Round(100 * time.Millisecond).String()
}

func renderChecked(result health.Result) string {
	if result.CheckedAt.IsZero() {
		return "—"
	}
	if time.Since(result.CheckedAt) < 2*time.Second {
		return "just now"
	}
	return result.CheckedAt.Format("15:04:05")
}

func exitCodeForResults(results []health.Result) int {
	for _, result := range results {
		if result.Status == health.StatusUnhealthy || result.Status == health.StatusUnreachable {
			return 1
		}
	}
	return 0
}

func suggestionForError(value string) string {
	for match, suggestion := range suggestions {
		if strings.Contains(value, match) {
			return suggestion
		}
	}
	return ""
}

func findContextInfo(contexts []application.ContextInfo, name string) (application.ContextInfo, bool) {
	for _, contextInfo := range contexts {
		if contextInfo.Name == name {
			return contextInfo, true
		}
	}
	return application.ContextInfo{}, false
}

func contextNamesFromInfo(contexts []application.ContextInfo) []string {
	names := make([]string, 0, len(contexts))
	for _, contextInfo := range contexts {
		names = append(names, contextInfo.Name)
	}
	return names
}
