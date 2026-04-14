package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
)

type PreviousContextStore struct {
	path string
}

func NewPreviousContextStore(path string) *PreviousContextStore {
	if strings.TrimSpace(path) == "" {
		path = config.GetLastContextPath()
	}

	return &PreviousContextStore{path: path}
}

func ReadPrevious() (string, error) {
	return NewPreviousContextStore("").ReadPrevious()
}

func WritePrevious(name string) error {
	return NewPreviousContextStore("").WritePrevious(name)
}

func (s *PreviousContextStore) ReadPrevious() (string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}

		return "", fmt.Errorf("read previous context: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func (s *PreviousContextStore) WritePrevious(name string) error {
	if err := os.MkdirAll(filepath.Dir(s.path), dirPermission); err != nil {
		return fmt.Errorf("create previous context directory: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(strings.TrimSpace(name)), filePermission); err != nil {
		return fmt.Errorf("write previous context temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename previous context temp file: %w", err)
	}

	return nil
}
