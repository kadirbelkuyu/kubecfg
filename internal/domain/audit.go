package domain

import "time"

type AuditEventType string

const (
	AuditEventGuardSessionStarted  AuditEventType = "guard_session_started"
	AuditEventGuardSessionStopped  AuditEventType = "guard_session_stopped"
	AuditEventGuardSessionExpired  AuditEventType = "guard_session_expired"
	AuditEventGuardStartFailed     AuditEventType = "guard_start_failed"
	AuditEventGuardRequestBlocked  AuditEventType = "guard_request_blocked"
	AuditEventGuardRequestAllowed  AuditEventType = "guard_request_allowed"
	AuditEventGuardRequestPending  AuditEventType = "guard_request_pending"
	AuditEventGuardRequestApproved AuditEventType = "guard_request_approved"
	AuditEventGuardRequestDenied   AuditEventType = "guard_request_denied"
)

type AuditEvent struct {
	Timestamp time.Time      `json:"timestamp"`
	Type      AuditEventType `json:"type"`
	SessionID string         `json:"session_id,omitempty"`
	Context   string         `json:"context,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Mode      string         `json:"mode,omitempty"`
	Message   string         `json:"message"`
}

type AuditStore interface {
	Append(event AuditEvent) error
	ListRecent(limit int) ([]AuditEvent, error)
}
