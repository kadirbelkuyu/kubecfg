package application

import "testing"

func TestReadonlyRequestPolicyValidate(t *testing.T) {
	policy := NewReadonlyRequestPolicy()

	tests := []struct {
		name    string
		method  string
		path    string
		wantErr bool
		wantMsg string
	}{
		{name: "get allowed", method: "GET", path: "/api/v1/pods", wantErr: false},
		{name: "head allowed", method: "HEAD", path: "/api/v1/pods", wantErr: false},
		{name: "watch get allowed", method: "GET", path: "/api/v1/pods?watch=1", wantErr: false},
		{name: "post blocked", method: "POST", path: "/api/v1/pods", wantErr: true, wantMsg: "guard readonly mode blocked mutating request: POST /api/v1/pods"},
		{name: "put blocked", method: "PUT", path: "/api/v1/pods/test", wantErr: true, wantMsg: "guard readonly mode blocked mutating request: PUT /api/v1/pods/test"},
		{name: "patch blocked", method: "PATCH", path: "/api/v1/namespaces/default/pods/demo", wantErr: true, wantMsg: "guard readonly mode blocked mutating request: PATCH /api/v1/namespaces/default/pods/demo"},
		{name: "delete blocked", method: "DELETE", path: "/api/v1/pods/test", wantErr: true, wantMsg: "guard readonly mode blocked mutating request: DELETE /api/v1/pods/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(tt.method, tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err.Error() != tt.wantMsg {
				t.Fatalf("Validate() error = %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}
