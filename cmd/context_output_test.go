package cmd

import (
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
)

func TestFilterContexts(t *testing.T) {
	contexts := []application.ContextInfo{
		{Name: "prod", Cluster: "eks-prod", Namespace: "payments", Server: "https://prod.example.com", Current: true},
		{Name: "staging", Cluster: "eks-staging", Namespace: "default", Server: "https://staging.example.com"},
		{Name: "dev", Cluster: "kind-dev", Namespace: "", Server: "https://dev.example.com"},
	}

	tests := []struct {
		name    string
		filters contextFilters
		want    []string
	}{
		{
			name:    "query matches namespace",
			filters: contextFilters{query: "payments"},
			want:    []string{"prod"},
		},
		{
			name:    "cluster filter matches partial name",
			filters: contextFilters{cluster: "eks"},
			want:    []string{"prod", "staging"},
		},
		{
			name:    "namespace filter treats empty as default",
			filters: contextFilters{namespace: "default"},
			want:    []string{"staging", "dev"},
		},
		{
			name:    "current filter keeps only active context",
			filters: contextFilters{currentOnly: true},
			want:    []string{"prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterContexts(contexts, tt.filters)
			if len(filtered) != len(tt.want) {
				t.Fatalf("filterContexts() length = %d, want %d", len(filtered), len(tt.want))
			}

			for i, ctx := range filtered {
				if ctx.Name != tt.want[i] {
					t.Fatalf("filterContexts()[%d] = %q, want %q", i, ctx.Name, tt.want[i])
				}
			}
		})
	}
}
