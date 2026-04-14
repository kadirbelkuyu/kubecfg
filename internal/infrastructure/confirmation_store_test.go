package infrastructure

import (
	"os"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

func TestFileConfirmationStoreCreateAndRead(t *testing.T) {
	dir := t.TempDir()
	store := NewFileConfirmationStore(dir)

	pending := &domain.PendingConfirmation{
		ID:        "abc-123",
		SessionID: "session-1",
		Method:    "DELETE",
		Resource:  "/api/v1/nodes/node-1",
		CreatedAt: time.Now().UTC(),
		Decision:  domain.ConfirmDecisionPending,
	}

	if err := store.Create(pending); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.Read("abc-123")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.ID != pending.ID {
		t.Errorf("ID = %q, want %q", got.ID, pending.ID)
	}
	if got.Method != pending.Method {
		t.Errorf("Method = %q, want %q", got.Method, pending.Method)
	}
	if got.Decision != domain.ConfirmDecisionPending {
		t.Errorf("Decision = %q, want pending", got.Decision)
	}
}

func TestFileConfirmationStoreDecide(t *testing.T) {
	dir := t.TempDir()
	store := NewFileConfirmationStore(dir)

	pending := &domain.PendingConfirmation{
		ID:       "decide-test",
		Method:   "DELETE",
		Resource: "/api/v1/nodes/node-1",
		Decision: domain.ConfirmDecisionPending,
	}

	if err := store.Create(pending); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Decide("decide-test", domain.ConfirmDecisionApproved); err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	got, err := store.Read("decide-test")
	if err != nil {
		t.Fatalf("Read() after Decide() error = %v", err)
	}
	if got.Decision != domain.ConfirmDecisionApproved {
		t.Errorf("Decision = %q, want approved", got.Decision)
	}
}

func TestFileConfirmationStoreListPending(t *testing.T) {
	dir := t.TempDir()
	store := NewFileConfirmationStore(dir)

	for i, decision := range []domain.ConfirmDecision{
		domain.ConfirmDecisionPending,
		domain.ConfirmDecisionApproved,
		domain.ConfirmDecisionPending,
	} {
		p := &domain.PendingConfirmation{
			ID:       string(rune('a' + i)),
			Decision: decision,
		}
		if err := store.Create(p); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	pending, err := store.ListPending()
	if err != nil {
		t.Fatalf("ListPending() error = %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("ListPending() count = %d, want 2", len(pending))
	}
}

func TestFileConfirmationStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileConfirmationStore(dir)

	p := &domain.PendingConfirmation{ID: "delete-me", Decision: domain.ConfirmDecisionPending}
	if err := store.Create(p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Delete("delete-me"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := store.Read("delete-me"); err == nil {
		t.Fatal("Read() after Delete() should return error")
	}
}

func TestFileConfirmationStoreDeleteMissing(t *testing.T) {
	store := NewFileConfirmationStore(t.TempDir())

	if err := store.Delete("missing"); err != nil {
		t.Fatalf("Delete() on missing confirmation error = %v", err)
	}
}

func TestFileConfirmationStoreListPendingEmptyDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.Remove(dir)
	store := NewFileConfirmationStore(dir)

	pending, err := store.ListPending()
	if err != nil {
		t.Fatalf("ListPending() on missing dir error = %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("ListPending() on missing dir = %d, want 0", len(pending))
	}
}

func TestFileConfirmationStoreReadMissing(t *testing.T) {
	store := NewFileConfirmationStore(t.TempDir())

	if _, err := store.Read("missing"); err == nil {
		t.Fatal("Read() on missing confirmation should return error")
	}
}
