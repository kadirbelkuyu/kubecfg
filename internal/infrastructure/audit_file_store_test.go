package infrastructure

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

func TestAuditFileStoreListRecentReturnsNewestFirst(t *testing.T) {
	store := NewAuditFileStore(filepath.Join(t.TempDir(), "audit.log"))
	events := []domain.AuditEvent{
		{
			Timestamp: time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
			Type:      domain.AuditEventGuardSessionStarted,
			Message:   "started",
		},
		{
			Timestamp: time.Date(2026, 4, 10, 10, 5, 0, 0, time.UTC),
			Type:      domain.AuditEventGuardRequestBlocked,
			Message:   "blocked",
		},
		{
			Timestamp: time.Date(2026, 4, 10, 10, 10, 0, 0, time.UTC),
			Type:      domain.AuditEventGuardSessionStopped,
			Message:   "stopped",
		},
	}

	for _, event := range events {
		if err := store.Append(event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	recent, err := store.ListRecent(2)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}

	if len(recent) != 2 {
		t.Fatalf("ListRecent() count = %d, want 2", len(recent))
	}

	if recent[0].Type != domain.AuditEventGuardSessionStopped || recent[1].Type != domain.AuditEventGuardRequestBlocked {
		t.Fatalf("ListRecent() = %+v", recent)
	}
}
