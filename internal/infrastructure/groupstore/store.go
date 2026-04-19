package groupstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
)

const (
	filePermission = 0600
	dirPermission  = 0700
)

type Store = domaingroup.Store

type FileStore struct {
	path string
}

type fileData struct {
	Groups []domaingroup.Group `yaml:"groups"`
}

func NewFileStore(path string) *FileStore {
	if strings.TrimSpace(path) == "" {
		path = config.GetGroupsPath()
	}

	return &FileStore{path: path}
}

func (s *FileStore) List() ([]domaingroup.Group, error) {
	data, err := s.read()
	if err != nil {
		return nil, err
	}

	return append([]domaingroup.Group(nil), data.Groups...), nil
}

func (s *FileStore) Get(name string) (domaingroup.Group, error) {
	data, err := s.read()
	if err != nil {
		return domaingroup.Group{}, err
	}

	for _, candidate := range data.Groups {
		if candidate.Name == name {
			return candidate, nil
		}
	}

	return domaingroup.Group{}, domaingroup.ErrGroupNotFound
}

func (s *FileStore) Save(g domaingroup.Group) error {
	if err := g.Validate(); err != nil {
		return err
	}

	data, err := s.read()
	if err != nil {
		return err
	}

	for index := range data.Groups {
		if data.Groups[index].Name == g.Name {
			data.Groups[index] = g
			return s.write(data)
		}
	}

	data.Groups = append(data.Groups, g)
	return s.write(data)
}

func (s *FileStore) Delete(name string) error {
	data, err := s.read()
	if err != nil {
		return err
	}

	for index := range data.Groups {
		if data.Groups[index].Name != name {
			continue
		}

		data.Groups = append(data.Groups[:index], data.Groups[index+1:]...)
		return s.write(data)
	}

	return domaingroup.ErrGroupNotFound
}

func (s *FileStore) ReplaceAll(groups []domaingroup.Group) error {
	return s.write(fileData{Groups: append([]domaingroup.Group(nil), groups...)})
}

func (s *FileStore) read() (fileData, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileData{Groups: []domaingroup.Group{}}, nil
		}

		return fileData{}, fmt.Errorf("read groups file: %w", err)
	}

	if len(data) == 0 {
		return fileData{Groups: []domaingroup.Group{}}, nil
	}

	var document fileData
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fileData{}, fmt.Errorf("decode groups file: %w", err)
	}

	if document.Groups == nil {
		document.Groups = []domaingroup.Group{}
	}

	for _, candidate := range document.Groups {
		if err := candidate.Validate(); err != nil {
			return fileData{}, fmt.Errorf("validate group %q: %w", candidate.Name, err)
		}
	}

	return document, nil
}

func (s *FileStore) write(data fileData) error {
	if data.Groups == nil {
		data.Groups = []domaingroup.Group{}
	}

	if err := os.MkdirAll(filepath.Dir(s.path), dirPermission); err != nil {
		return fmt.Errorf("create groups directory: %w", err)
	}

	payload, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode groups file: %w", err)
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, filePermission); err != nil {
		return fmt.Errorf("write groups temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit groups file: %w", err)
	}

	return nil
}
