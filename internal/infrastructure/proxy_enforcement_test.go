package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

// newEnforcementHandler builds an HTTP handler that mirrors the guard proxy's
// enforcement middleware: validate the request via policy, return 403 on
// rejection, 200 on pass-through.
func newEnforcementHandler(policy domain.GuardRequestPolicy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := policy.Validate(r.Method, r.URL.RequestURI()); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

// --- BlockedResources enforcement ---

func TestProxyBlockedResourcesBlocked(t *testing.T) {
	policy := application.NewPolicyRequestPolicy(&domain.Policy{
		Name:             "no-secrets",
		BlockedResources: []string{"secrets", "configmaps"},
	})
	srv := httptest.NewServer(newEnforcementHandler(policy))
	defer srv.Close()

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/api/v1/namespaces/default/secrets", http.StatusForbidden},
		{"GET", "/api/v1/namespaces/default/secrets/my-token", http.StatusForbidden},
		{"GET", "/api/v1/namespaces/default/configmaps", http.StatusForbidden},
		{"POST", "/api/v1/namespaces/default/secrets", http.StatusForbidden},
		{"GET", "/api/v1/namespaces/default/pods", http.StatusOK},
		{"GET", "/api/v1/namespaces/default/services", http.StatusOK},
	}

	client := srv.Client()
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

// --- AllowedNamespaces enforcement ---

func TestProxyAllowedNamespacesEnforcement(t *testing.T) {
	policy := application.NewPolicyRequestPolicy(&domain.Policy{
		Name:              "ns-restricted",
		AllowedNamespaces: []string{"production", "monitoring"},
	})
	srv := httptest.NewServer(newEnforcementHandler(policy))
	defer srv.Close()

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/api/v1/namespaces/production/pods", http.StatusOK},
		{"GET", "/api/v1/namespaces/monitoring/pods", http.StatusOK},
		{"GET", "/api/v1/namespaces/default/pods", http.StatusForbidden},
		{"GET", "/api/v1/namespaces/kube-system/secrets", http.StatusForbidden},
		// cluster-scoped resources (no namespace in path) are always allowed
		{"GET", "/api/v1/nodes", http.StatusOK},
	}

	client := srv.Client()
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })
			if resp.StatusCode != tt.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}

// --- Policy-aware error messages ---

func TestProxyErrorMessagesDescriptive(t *testing.T) {
	tests := []struct {
		name           string
		policy         *domain.Policy
		method         string
		path           string
		wantSubstrings []string
	}{
		{
			name:           "readonly blocks POST",
			policy:         &domain.Policy{Name: "p", Readonly: true},
			method:         "POST",
			path:           "/api/v1/pods",
			wantSubstrings: []string{"POST", "write operations are disabled"},
		},
		{
			name:           "blocked resource message",
			policy:         &domain.Policy{Name: "p", BlockedResources: []string{"secrets"}},
			method:         "GET",
			path:           "/api/v1/namespaces/default/secrets",
			wantSubstrings: []string{"secrets", "blocked"},
		},
		{
			name:           "namespace not allowed message",
			policy:         &domain.Policy{Name: "p", AllowedNamespaces: []string{"prod"}},
			method:         "GET",
			path:           "/api/v1/namespaces/default/pods",
			wantSubstrings: []string{"default", "allowed"},
		},
		{
			name:           "destructive confirm message",
			policy:         &domain.Policy{Name: "p", ConfirmDestructive: true},
			method:         "DELETE",
			path:           "/api/v1/nodes/node-1",
			wantSubstrings: []string{"destructive", "confirmation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			newEnforcementHandler(application.NewPolicyRequestPolicy(tt.policy)).ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", rr.Code)
			}

			body := rr.Body.String()
			for _, sub := range tt.wantSubstrings {
				if !containsCI(body, sub) {
					t.Errorf("response body %q does not contain %q", body, sub)
				}
			}
		})
	}
}

// --- Readonly enforcement ---

func TestProxyReadonlyBlocksMutating(t *testing.T) {
	policy := application.NewPolicyRequestPolicy(&domain.Policy{
		Name:     "readonly",
		Readonly: true,
	})
	srv := httptest.NewServer(newEnforcementHandler(policy))
	defer srv.Close()

	allowed := []string{"GET", "HEAD"}
	blocked := []string{"POST", "PUT", "PATCH", "DELETE"}

	client := srv.Client()

	for _, method := range allowed {
		t.Run(method+" allowed", func(t *testing.T) {
			req, _ := http.NewRequest(method, srv.URL+"/api/v1/pods", nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: status = %d, want 200", method, resp.StatusCode)
			}
		})
	}

	for _, method := range blocked {
		t.Run(method+" blocked", func(t *testing.T) {
			req, _ := http.NewRequest(method, srv.URL+"/api/v1/pods", nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s: status = %d, want 403", method, resp.StatusCode)
			}
		})
	}
}

// containsCI is a simple case-insensitive substring check.
func containsCI(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	sl, subl := []rune(s), []rune(sub)
	toLower := func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return r
	}
	for i := range sl {
		if i+len(subl) > len(sl) {
			return false
		}
		match := true
		for j, r := range subl {
			if toLower(sl[i+j]) != toLower(r) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
