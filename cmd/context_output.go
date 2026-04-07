package cmd

import (
	"fmt"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

type contextFilters struct {
	query       string
	cluster     string
	namespace   string
	currentOnly bool
}

func (f contextFilters) active() bool {
	return f.query != "" || f.cluster != "" || f.namespace != "" || f.currentOnly
}

func filterContexts(contexts []application.ContextInfo, filters contextFilters) []application.ContextInfo {
	if !filters.active() {
		return contexts
	}

	filtered := make([]application.ContextInfo, 0, len(contexts))
	query := strings.ToLower(strings.TrimSpace(filters.query))
	cluster := strings.ToLower(strings.TrimSpace(filters.cluster))
	namespace := strings.ToLower(strings.TrimSpace(filters.namespace))

	for _, ctx := range contexts {
		if filters.currentOnly && !ctx.Current {
			continue
		}

		if cluster != "" && !strings.Contains(strings.ToLower(ctx.Cluster), cluster) {
			continue
		}

		ctxNamespace := ctx.Namespace
		if ctxNamespace == "" {
			ctxNamespace = "default"
		}
		if namespace != "" && !strings.Contains(strings.ToLower(ctxNamespace), namespace) {
			continue
		}

		if query != "" && !matchesContextQuery(ctx, query) {
			continue
		}

		filtered = append(filtered, ctx)
	}

	return filtered
}

func matchesContextQuery(ctx application.ContextInfo, query string) bool {
	fields := []string{
		ctx.Name,
		ctx.Cluster,
		ctx.Server,
		ctx.Namespace,
	}

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}

	return false
}

func printContextTable(contexts []application.ContextInfo) {
	var output strings.Builder

	_, _ = fmt.Fprintf(&output, "  %s  %-30s  %-25s  %-40s  %-20s\n",
		ui.Header(""),
		ui.Header("NAME"),
		ui.Header("CLUSTER"),
		ui.Header("SERVER"),
		ui.Header("NAMESPACE"))

	_, _ = fmt.Fprintf(&output, "  %s\n", strings.Repeat("─", 120))

	for _, ctx := range contexts {
		current := "  "
		nameStyle := ctx.Name
		if ctx.Current {
			current = ui.CurrentIndicator()
			nameStyle = ui.ContextName(ctx.Name)
		}

		namespace := ctx.Namespace
		if namespace == "" {
			namespace = "default"
		}

		_, _ = fmt.Fprintf(&output, "  %s  %-30s  %-25s  %-40s  %-20s\n",
			current,
			nameStyle,
			ui.Cluster(ctx.Cluster),
			ui.Server(ctx.Server),
			ui.Namespace(namespace))
	}

	fmt.Println(output.String())
	fmt.Printf("\n%s\n", ui.Info(fmt.Sprintf("Total contexts: %d", len(contexts))))
}
