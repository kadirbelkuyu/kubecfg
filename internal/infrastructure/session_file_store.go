package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type SessionFileStore struct {
	path string
}

func NewSessionFileStore(path string) *SessionFileStore {
	return &SessionFileStore{path: path}
}

func (s *SessionFileStore) Load() (*domain.Session, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, domain.ErrGuardSessionNotFound
		}
		return nil, fmt.Errorf("read session file: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("decode session file: %w", err)
	}

	return &session, nil
}

func (s *SessionFileStore) Save(session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session is required")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), dirPermission); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, filePermission); err != nil {
		return fmt.Errorf("write session file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("save session file: %w", err)
	}

	return nil
}

func (s *SessionFileStore) Delete() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove session file: %w", err)
	}
	return nil
}
