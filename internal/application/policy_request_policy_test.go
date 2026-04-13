package application

import (
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

func TestPolicyRequestPolicyProd(t *testing.T) {
	prod, _ := domain.FindProfile(domain.PolicyProfileProd)
	policy := NewPolicyRequestPolicy(prod)

	tests := []struct {
		name    string
		method  string
		path    string
		wantErr bool
		wantMsg string
	}{
		{name: "GET allowed", method: "GET", path: "/api/v1/pods", wantErr: false},
		{name: "HEAD allowed", method: "HEAD", path: "/api/v1/pods", wantErr: false},
		{name: "POST blocked (readonly)", method: "POST", path: "/api/v1/pods", wantErr: true},
		{name: "DELETE blocked (readonly)", method: "DELETE", path: "/api/v1/pods/test", wantErr: true},
		{name: "GET secrets blocked", method: "GET", path: "/api/v1/namespaces/default/secrets", wantErr: true},
		{name: "GET secrets subpath blocked", method: "GET", path: "/api/v1/namespaces/default/secrets/my-secret", wantErr: true},
		{name: "GET pods allowed", method: "GET", path: "/api/v1/namespaces/default/pods", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(tt.method, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%q, %q) error = %v, wantErr %v", tt.method, tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRequestPolicyStaging(t *testing.T) {
	staging, _ := domain.FindProfile(domain.PolicyProfileStaging)
	policy := NewPolicyRequestPolicy(staging)

	tests := []struct {
		name    string
		method  string
		path    string
		wantErr bool
	}{
		{name: "GET allowed", method: "GET", path: "/api/v1/pods", wantErr: false},
		{name: "POST allowed", method: "POST", path: "/api/v1/pods", wantErr: false},
		{name: "PUT allowed", method: "PUT", path: "/api/v1/pods/test", wantErr: false},
		{name: "PATCH allowed", method: "PATCH", path: "/api/v1/pods/test", wantErr: false},
		{name: "DELETE blocked (destructive)", method: "DELETE", path: "/api/v1/pods/test", wantErr: true},
		{name: "GET secrets allowed in staging", method: "GET", path: "/api/v1/namespaces/default/secrets", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(tt.method, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%q, %q) error = %v, wantErr %v", tt.method, tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRequestPolicyDebug(t *testing.T) {
	debug, _ := domain.FindProfile(domain.PolicyProfileDebug)
	policy := NewPolicyRequestPolicy(debug)

	tests := []struct {
		name    string
		method  string
		path    string
		wantErr bool
	}{
		{name: "GET allowed", method: "GET", path: "/api/v1/pods", wantErr: false},
		{name: "POST allowed", method: "POST", path: "/api/v1/pods", wantErr: false},
		{name: "DELETE allowed", method: "DELETE", path: "/api/v1/pods/test", wantErr: false},
		{name: "GET secrets allowed", method: "GET", path: "/api/v1/namespaces/default/secrets", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(tt.method, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%q, %q) error = %v, wantErr %v", tt.method, tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRequestPolicyBlockedVerbs(t *testing.T) {
	p := &domain.Policy{
		Name:         "custom",
		BlockedVerbs: []string{"POST", "PUT"},
	}
	policy := NewPolicyRequestPolicy(p)

	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{name: "GET allowed", method: "GET", wantErr: false},
		{name: "DELETE allowed", method: "DELETE", wantErr: false},
		{name: "POST blocked", method: "POST", wantErr: true},
		{name: "PUT blocked", method: "PUT", wantErr: true},
		{name: "PATCH allowed", method: "PATCH", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(tt.method, "/api/v1/pods")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(%q) error = %v, wantErr %v", tt.method, err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRequestPolicyAllowedNamespaces(t *testing.T) {
	p := &domain.Policy{
		Name:              "ns-restricted",
		AllowedNamespaces: []string{"production", "monitoring"},
	}
	policy := NewPolicyRequestPolicy(p)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "allowed namespace production", path: "/api/v1/namespaces/production/pods", wantErr: false},
		{name: "allowed namespace monitoring", path: "/api/v1/namespaces/monitoring/pods", wantErr: false},
		{name: "blocked namespace default", path: "/api/v1/namespaces/default/pods", wantErr: true},
		{name: "blocked namespace kube-system", path: "/api/v1/namespaces/kube-system/secrets", wantErr: true},
		{name: "no namespace in path always allowed", path: "/api/v1/nodes", wantErr: false},
		{name: "namespace list endpoint allowed", path: "/api/v1/namespaces", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate("GET", tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(GET, %q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRequestPolicyBlockedResources(t *testing.T) {
	p := &domain.Policy{
		Name:             "no-secrets",
		BlockedResources: []string{"secrets", "configmaps"},
	}
	policy := NewPolicyRequestPolicy(p)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "pods allowed", path: "/api/v1/namespaces/default/pods", wantErr: false},
		{name: "secrets blocked", path: "/api/v1/namespaces/default/secrets", wantErr: true},
		{name: "specific secret blocked", path: "/api/v1/namespaces/default/secrets/my-token", wantErr: true},
		{name: "configmaps blocked", path: "/api/v1/namespaces/default/configmaps", wantErr: true},
		{name: "query params stripped", path: "/api/v1/namespaces/default/secrets?watch=1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate("GET", tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate(GET, %q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestPolicyRequestBlockedErrorMessage(t *testing.T) {
	p := &domain.Policy{
		Name:               "test",
		ConfirmDestructive: true,
	}
	policy := NewPolicyRequestPolicy(p)

	err := policy.Validate("DELETE", "/api/v1/pods/mypod")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	blocked, ok := err.(*PolicyRequestBlockedError)
	if !ok {
		t.Fatalf("expected *PolicyRequestBlockedError, got %T", err)
	}

	if blocked.Method != "DELETE" {
		t.Fatalf("Method = %q, want DELETE", blocked.Method)
	}

	if blocked.Path != "/api/v1/pods/mypod" {
		t.Fatalf("Path = %q, want /api/v1/pods/mypod", blocked.Path)
	}

	if blocked.Reason != "destructive operation requires confirmation" {
		t.Fatalf("Reason = %q, unexpected", blocked.Reason)
	}
}

func TestPathContainsResource(t *testing.T) {
	tests := []struct {
		path     string
		resource string
		want     bool
	}{
		{"/api/v1/namespaces/default/secrets", "secrets", true},
		{"/api/v1/namespaces/default/secrets/my-token", "secrets", true},
		{"/api/v1/namespaces/default/pods", "secrets", false},
		{"/api/v1/secrets", "secrets", true},
		{"/api/v1/namespaces/default/secrets?watch=1", "secrets", true},
		{"/api/v1/namespaces/default/Secrets", "secrets", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := pathContainsResource(tt.path, tt.resource); got != tt.want {
				t.Fatalf("pathContainsResource(%q, %q) = %v, want %v", tt.path, tt.resource, got, tt.want)
			}
		})
	}
}

func TestExtractNamespaceFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/namespaces/default/pods", "default"},
		{"/api/v1/namespaces/kube-system/secrets", "kube-system"},
		{"/api/v1/nodes", ""},
		{"/api/v1/namespaces", ""},
		{"/apis/apps/v1/namespaces/production/deployments", "production"},
		{"/api/v1/namespaces/default/pods?watch=1", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := extractNamespaceFromPath(tt.path); got != tt.want {
				t.Fatalf("extractNamespaceFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
