package application

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/config"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type KubeconfigSourceInfo struct {
	Path           string
	Directory      string
	Name           string
	Active         bool
	ContextCount   int
	CurrentContext string
	Error          string
}

func (s *Service) ListKubeconfigSources(activePath string, dirs []string) []KubeconfigSourceInfo {
	candidates := discoverKubeconfigCandidates(activePath, dirs)
	sources := make([]KubeconfigSourceInfo, 0, len(candidates))
	normalizedActive := config.ExpandPath(activePath)

	for _, path := range candidates {
		source := KubeconfigSourceInfo{
			Path:      path,
			Directory: filepath.Dir(path),
			Name:      filepath.Base(path),
			Active:    path == normalizedActive,
		}

		cfg, err := s.repo.Load(path)
		if err != nil {
			source.Error = err.Error()
			if source.Active {
				sources = append(sources, source)
			}
			continue
		}
		if len(cfg.Contexts) == 0 && !source.Active {
			continue
		}

		source.ContextCount = len(cfg.Contexts)
		source.CurrentContext = cfg.CurrentContext
		sources = append(sources, source)
	}

	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].Active != sources[j].Active {
			return sources[i].Active
		}
		if sources[i].Directory != sources[j].Directory {
			return sources[i].Directory < sources[j].Directory
		}
		return sources[i].Name < sources[j].Name
	})

	return sources
}

func discoverKubeconfigCandidates(activePath string, dirs []string) []string {
	seen := make(map[string]struct{})
	candidates := make([]string, 0)

	add := func(path string) {
		path = config.ExpandPath(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	add(activePath)

	for _, dir := range dirs {
		dir = config.ExpandPath(dir)
		if dir == "" {
			continue
		}

		info, err := os.Stat(dir)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			add(dir)
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || shouldSkipKubeconfigCandidate(entry.Name()) {
				continue
			}
			add(filepath.Join(dir, entry.Name()))
		}
	}

	return candidates
}

func shouldSkipKubeconfigCandidate(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	if strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".lock") {
		return true
	}
	if strings.Contains(name, ".backup.") {
		return true
	}
	return false
}

func (s *Service) ValidateKubeconfigSource(path string) error {
	cfg, err := s.repo.Load(path)
	if err != nil {
		return err
	}
	if len(cfg.Contexts) == 0 {
		return domain.ErrInvalidConfig
	}
	return nil
}
