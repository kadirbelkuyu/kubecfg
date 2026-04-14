package infrastructure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

const auditRotationSizeBytes = 10 * 1024 * 1024 // 10 MB

type AuditFileStore struct {
	path string
}

func NewAuditFileStore(path string) *AuditFileStore {
	return &AuditFileStore{path: path}
}

func (s *AuditFileStore) Append(event domain.AuditEvent) error {
	if err := os.MkdirAll(filepath.Dir(s.path), dirPermission); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}

	if err := s.rotateIfNeeded(); err != nil {
		return err
	}

	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePermission)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}

	return nil
}

func (s *AuditFileStore) ListRecent(limit int) ([]domain.AuditEvent, error) {
	if limit <= 0 {
		return nil, nil
	}

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	lines := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lines = append(lines, line)
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan audit log: %w", err)
	}

	events := make([]domain.AuditEvent, 0, len(lines))
	for index := len(lines) - 1; index >= 0; index-- {
		var event domain.AuditEvent
		if err := json.Unmarshal([]byte(lines[index]), &event); err != nil {
			return nil, fmt.Errorf("decode audit event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

func (s *AuditFileStore) rotateIfNeeded() error {
	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat audit log: %w", err)
	}

	if info.Size() < auditRotationSizeBytes {
		return nil
	}

	rotated := s.path + "." + time.Now().UTC().Format("20060102-150405")
	if err := os.Rename(s.path, rotated); err != nil {
		return fmt.Errorf("rotate audit log: %w", err)
	}

	return nil
}
