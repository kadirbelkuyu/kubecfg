package application

import (
	"fmt"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

const defaultAuditRecentLimit = 5

type AuditService struct {
	store   domain.AuditStore
	enabled bool
	now     func() time.Time
}

func NewAuditService(store domain.AuditStore, enabled bool) *AuditService {
	return &AuditService{
		store:   store,
		enabled: enabled,
		now:     time.Now,
	}
}

func (s *AuditService) Record(event domain.AuditEvent) error {
	if s == nil || !s.enabled || s.store == nil {
		return nil
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = s.now().UTC()
	}

	if err := s.store.Append(event); err != nil {
		return fmt.Errorf("append audit event: %w", err)
	}

	return nil
}

func (s *AuditService) Recent(limit int) ([]domain.AuditEvent, error) {
	if s == nil || !s.enabled || s.store == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = defaultAuditRecentLimit
	}

	events, err := s.store.ListRecent(limit)
	if err != nil {
		return nil, fmt.Errorf("list recent audit events: %w", err)
	}

	return events, nil
}
