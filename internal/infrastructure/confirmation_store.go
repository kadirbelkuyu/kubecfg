package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type FileConfirmationStore struct {
	dir string
}

func NewFileConfirmationStore(dir string) *FileConfirmationStore {
	return &FileConfirmationStore{dir: dir}
}

func (s *FileConfirmationStore) Create(pending *domain.PendingConfirmation) error {
	if err := os.MkdirAll(s.dir, dirPermission); err != nil {
		return fmt.Errorf("create confirmations dir: %w", err)
	}

	data, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("encode confirmation: %w", err)
	}

	path := s.path(pending.ID)
	if err := os.WriteFile(path, data, filePermission); err != nil {
		return fmt.Errorf("write confirmation: %w", err)
	}

	return nil
}

func (s *FileConfirmationStore) Read(id string) (*domain.PendingConfirmation, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("confirmation %q not found", id)
		}
		return nil, fmt.Errorf("read confirmation: %w", err)
	}

	var pending domain.PendingConfirmation
	if err := json.Unmarshal(data, &pending); err != nil {
		return nil, fmt.Errorf("decode confirmation: %w", err)
	}

	return &pending, nil
}

func (s *FileConfirmationStore) Decide(id string, decision domain.ConfirmDecision) error {
	pending, err := s.Read(id)
	if err != nil {
		return err
	}

	pending.Decision = decision

	data, err := json.Marshal(pending)
	if err != nil {
		return fmt.Errorf("encode confirmation: %w", err)
	}

	tmp := s.path(id) + ".tmp"
	if err := os.WriteFile(tmp, data, filePermission); err != nil {
		return fmt.Errorf("write confirmation update: %w", err)
	}

	if err := os.Rename(tmp, s.path(id)); err != nil {
		return fmt.Errorf("commit confirmation update: %w", err)
	}

	return nil
}

func (s *FileConfirmationStore) ListPending() ([]*domain.PendingConfirmation, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read confirmations dir: %w", err)
	}

	var pending []*domain.PendingConfirmation
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		p, err := s.Read(id)
		if err != nil {
			continue
		}

		if p.Decision == domain.ConfirmDecisionPending {
			pending = append(pending, p)
		}
	}

	return pending, nil
}

func (s *FileConfirmationStore) Delete(id string) error {
	if err := os.Remove(s.path(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete confirmation: %w", err)
	}
	return nil
}

func (s *FileConfirmationStore) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}
