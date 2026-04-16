package health

import (
	"testing"
	"time"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{name: "unknown", status: StatusUnknown, want: "unknown"},
		{name: "healthy", status: StatusHealthy, want: "healthy"},
		{name: "degraded", status: StatusDegraded, want: "degraded"},
		{name: "unhealthy", status: StatusUnhealthy, want: "unhealthy"},
		{name: "unreachable", status: StatusUnreachable, want: "unreachable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{name: "unknown", status: StatusUnknown, want: "?"},
		{name: "healthy", status: StatusHealthy, want: "✓"},
		{name: "degraded", status: StatusDegraded, want: "~"},
		{name: "unhealthy", status: StatusUnhealthy, want: "✗"},
		{name: "unreachable", status: StatusUnreachable, want: "✗"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.Emoji(); got != tt.want {
				t.Fatalf("Emoji() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHealthThresholds(t *testing.T) {
	if DegradedThreshold <= 0 {
		t.Fatal("DegradedThreshold must be positive")
	}
	if CheckTimeout <= DegradedThreshold {
		t.Fatal("CheckTimeout must be greater than DegradedThreshold")
	}
	if CacheExpiry <= 0 {
		t.Fatal("CacheExpiry must be positive")
	}
	if MaxConcurrency <= 0 {
		t.Fatal("MaxConcurrency must be positive")
	}
	if CacheExpiry < 5*time.Second {
		t.Fatal("CacheExpiry is unexpectedly low")
	}
}
