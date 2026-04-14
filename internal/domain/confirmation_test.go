package domain

import (
	"testing"
	"time"
)

func TestPendingConfirmationDefaults(t *testing.T) {
	p := &PendingConfirmation{
		ID:        "test-id",
		SessionID: "session-id",
		Method:    "DELETE",
		Resource:  "/api/v1/namespaces/production",
		CreatedAt: time.Now(),
		Decision:  ConfirmDecisionPending,
	}

	if p.Decision != ConfirmDecisionPending {
		t.Fatalf("expected pending decision, got %q", p.Decision)
	}
}

func TestConfirmDecisionConstants(t *testing.T) {
	tests := []struct {
		decision ConfirmDecision
		want     string
	}{
		{ConfirmDecisionPending, "pending"},
		{ConfirmDecisionApproved, "approved"},
		{ConfirmDecisionDenied, "denied"},
	}

	for _, tt := range tests {
		if string(tt.decision) != tt.want {
			t.Errorf("decision constant = %q, want %q", tt.decision, tt.want)
		}
	}
}
