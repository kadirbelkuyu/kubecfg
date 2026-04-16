package groupservice

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	domaingroup "github.com/kadirbelkuyu/kubecfg/internal/domain/group"
)

type replaceAllStore interface {
	ReplaceAll(groups []domaingroup.Group) error
}

type MissingContextsError struct {
	Names []string
}

func (e MissingContextsError) Error() string {
	if len(e.Names) == 1 {
		return fmt.Sprintf("context %q is not in your kubeconfig", e.Names[0])
	}

	return fmt.Sprintf("contexts %q are not in your kubeconfig", strings.Join(e.Names, ", "))
}

type Service struct {
	store          domaingroup.Store
	kubeRepo       domain.KubeConfigRepository
	kubeconfigPath string
}

func NewService(store domaingroup.Store, kubeRepo domain.KubeConfigRepository, kubeconfigPath string) *Service {
	path := strings.TrimSpace(kubeconfigPath)
	if path == "" && kubeRepo != nil {
		path = kubeRepo.GetDefaultPath()
	}

	return &Service{
		store:          store,
		kubeRepo:       kubeRepo,
		kubeconfigPath: path,
	}
}

func (s *Service) Create(g domaingroup.Group) error {
	g = normalizeGroup(g)
	if err := g.Validate(); err != nil {
		return err
	}

	if _, err := s.store.Get(g.Name); err == nil {
		return domaingroup.ErrGroupAlreadyExists
	} else if !errors.Is(err, domaingroup.ErrGroupNotFound) {
		return fmt.Errorf("get group %q: %w", g.Name, err)
	}

	missing, err := s.missingContexts(g.Contexts)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return MissingContextsError{Names: missing}
	}

	if err := s.store.Save(g); err != nil {
		return fmt.Errorf("save group %q: %w", g.Name, err)
	}

	return nil

}

func (s *Service) AddContext(groupName, contextName string) error {
	contextName = strings.TrimSpace(contextName)
	if err := s.ensureContextExists(contextName); err != nil {
		return err
	}

	g, err := s.Get(groupName)
	if err != nil {
		return err
	}

	for _, existing := range g.Contexts {
		if existing == contextName {
			return nil
		}
	}

	g.Contexts = append(g.Contexts, contextName)
	if err := s.store.Save(g); err != nil {
		return fmt.Errorf("save group %q: %w", g.Name, err)
	}

	return nil

}

func (s *Service) RemoveContext(groupName, contextName string) error {
	g, err := s.Get(groupName)
	if err != nil {
		return err
	}

	index := -1
	for i, existing := range g.Contexts {
		if existing == contextName {
			index = i
			break
		}
	}

	if index < 0 {
		return nil
	}

	if len(g.Contexts) == 1 {
		return domaingroup.ErrEmptyContextList
	}

	g.Contexts = append(g.Contexts[:index], g.Contexts[index+1:]...)
	if err := s.store.Save(g); err != nil {
		return fmt.Errorf("save group %q: %w", g.Name, err)
	}

	return nil
}

func (s *Service) Delete(name string) error {
	if err := s.store.Delete(strings.TrimSpace(name)); err != nil {
		if errors.Is(err, domaingroup.ErrGroupNotFound) {
			return domaingroup.ErrGroupNotFound
		}
		return fmt.Errorf("delete group %q: %w", name, err)
	}

	return nil
}

func (s *Service) List() ([]domaingroup.Group, error) {
	groups, err := s.store.List()
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}

	return groups, nil
}

func (s *Service) Get(name string) (domaingroup.Group, error) {
	g, err := s.store.Get(strings.TrimSpace(name))
	if err != nil {
		if errors.Is(err, domaingroup.ErrGroupNotFound) {
			return domaingroup.Group{}, domaingroup.ErrGroupNotFound
		}
		return domaingroup.Group{}, fmt.Errorf("get group %q: %w", name, err)
	}

	return g, nil
}

func (s *Service) Resolve(name string) (domaingroup.Group, []string, error) {
	g, err := s.Get(name)
	if err != nil {
		return domaingroup.Group{}, nil, err
	}

	missing, err := s.missingContexts(g.Contexts)
	if err != nil {
		return domaingroup.Group{}, nil, err
	}

	return g, missing, nil
}

func (s *Service) Rename(oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)

	groups, err := s.List()
	if err != nil {
		return err
	}

	index := -1
	for i, candidate := range groups {
		if candidate.Name == oldName {
			index = i
			continue
		}

		if candidate.Name == newName {
			return domaingroup.ErrGroupAlreadyExists
		}
	}

	if index < 0 {
		return domaingroup.ErrGroupNotFound
	}

	updated := normalizeGroup(groups[index])
	updated.Name = newName
	if err := updated.Validate(); err != nil {
		return err
	}

	groups[index] = updated
	if writer, ok := s.store.(replaceAllStore); ok {
		if err := writer.ReplaceAll(groups); err != nil {
			return fmt.Errorf("rewrite groups: %w", err)
		}
		return nil
	}

	if err := s.store.Save(updated); err != nil {
		return fmt.Errorf("save group %q: %w", updated.Name, err)
	}

	if err := s.store.Delete(oldName); err != nil {
		return fmt.Errorf("delete group %q: %w", oldName, err)
	}

	return nil
}

func (s *Service) ensureContextExists(contextName string) error {
	missing, err := s.missingContexts([]string{contextName})
	if err != nil {
		return err
	}

	if len(missing) > 0 {
		return MissingContextsError{Names: missing}
	}

	return nil
}

func (s *Service) missingContexts(contextNames []string) ([]string, error) {
	config, err := s.loadKubeConfig()
	if err != nil {
		return nil, err
	}

	existing := make(map[string]struct{}, len(config.Contexts))
	for _, context := range config.Contexts {
		existing[context.Name] = struct{}{}
	}

	missing := make([]string, 0)
	for _, contextName := range contextNames {
		if _, ok := existing[contextName]; ok {
			continue
		}

		missing = append(missing, contextName)
	}

	return missing, nil
}

func (s *Service) loadKubeConfig() (*domain.KubeConfig, error) {
	if s.kubeRepo == nil {
		return nil, fmt.Errorf("kubeconfig repository is required")
	}

	config, err := s.kubeRepo.Load(s.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	return config, nil
}

func normalizeGroup(g domaingroup.Group) domaingroup.Group {
	g.Name = strings.TrimSpace(g.Name)
	g.Description = strings.TrimSpace(g.Description)
	g.Color = strings.TrimSpace(g.Color)

	contexts := make([]string, 0, len(g.Contexts))
	for _, contextName := range g.Contexts {
		contexts = append(contexts, strings.TrimSpace(contextName))
	}
	g.Contexts = contexts

	return g
}
