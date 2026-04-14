package domain

import "time"

type ConfirmDecision string

const (
	ConfirmDecisionPending  ConfirmDecision = "pending"
	ConfirmDecisionApproved ConfirmDecision = "approved"
	ConfirmDecisionDenied   ConfirmDecision = "denied"
)

type PendingConfirmation struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	Method    string          `json:"method"`
	Resource  string          `json:"resource"`
	Namespace string          `json:"namespace"`
	CreatedAt time.Time       `json:"created_at"`
	Decision  ConfirmDecision `json:"decision"`
}

type ConfirmationStore interface {
	Create(pending *PendingConfirmation) error
	Read(id string) (*PendingConfirmation, error)
	Decide(id string, decision ConfirmDecision) error
	ListPending() ([]*PendingConfirmation, error)
	Delete(id string) error
}
