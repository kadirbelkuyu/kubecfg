package health

import "time"

type Status int

const (
	StatusUnknown Status = iota
	StatusHealthy
	StatusDegraded
	StatusUnhealthy
	StatusUnreachable
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusUnreachable:
		return "unreachable"
	default:
		return "unknown"
	}
}

func (s Status) Emoji() string {
	switch s {
	case StatusHealthy:
		return "✓"
	case StatusDegraded:
		return "~"
	case StatusUnhealthy, StatusUnreachable:
		return "✗"
	default:
		return "?"
	}
}

type Result struct {
	ContextName string        `json:"contextName" yaml:"contextName"`
	Status      Status        `json:"status" yaml:"status"`
	Latency     time.Duration `json:"latency" yaml:"latency"`
	Error       string        `json:"error,omitempty" yaml:"error,omitempty"`
	CheckedAt   time.Time     `json:"checkedAt" yaml:"checkedAt"`
}

const (
	DegradedThreshold = 800 * time.Millisecond
	CheckTimeout      = 4 * time.Second
	CacheExpiry       = 30 * time.Second
	MaxConcurrency    = 10
)
