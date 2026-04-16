package healthservice

import (
	"context"
	"fmt"
	"sort"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
)

// Service orchestrates health checks for kubeconfig contexts.
type Service struct {
	checker        health.Checker
	cache          health.Cache
	repo           domain.KubeConfigRepository
	kubeconfigPath string
}

// New returns a Service wired with the provided checker, cache, repository, and kubeconfig path.
func New(checker health.Checker, cache health.Cache, repo domain.KubeConfigRepository, kubeconfigPath string) *Service {
	return &Service{
		checker:        checker,
		cache:          cache,
		repo:           repo,
		kubeconfigPath: kubeconfigPath,
	}
}

// CheckContext runs a live health check for a single context and stores the result in the cache.
func (s *Service) CheckContext(ctx context.Context, contextName string) (health.Result, error) {
	result := s.checker.Check(ctx, contextName)
	s.cache.Set(result)
	return result, nil
}

// CheckAll runs live health checks for all contexts in the kubeconfig, caches every result,
// and returns them sorted by context name.
func (s *Service) CheckAll(ctx context.Context) ([]health.Result, error) {
	names, err := s.contextNames()
	if err != nil {
		return nil, err
	}

	results := s.checker.CheckAll(ctx, names)
	for _, r := range results {
		s.cache.Set(r)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ContextName < results[j].ContextName
	})

	return results, nil
}

// GetCached returns the cached health result for a context.
// If no cached entry exists it returns a Result with StatusUnknown.
func (s *Service) GetCached(contextName string) health.Result {
	if result, ok := s.cache.Get(contextName); ok {
		return result
	}
	return health.Result{
		ContextName: contextName,
		Status:      health.StatusUnknown,
	}
}

// GetAllCached returns cached health results for every context in the kubeconfig.
// Contexts without a cache entry receive StatusUnknown. Results are sorted by context name.
func (s *Service) GetAllCached(ctx context.Context) ([]health.Result, error) {
	names, err := s.contextNames()
	if err != nil {
		return nil, err
	}

	results := make([]health.Result, 0, len(names))
	for _, name := range names {
		results = append(results, s.GetCached(name))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ContextName < results[j].ContextName
	})

	return results, nil
}

func (s *Service) contextNames() ([]string, error) {
	cfg, err := s.repo.Load(s.kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	names := make([]string, 0, len(cfg.Contexts))
	for _, ctx := range cfg.Contexts {
		names = append(names, ctx.Name)
	}
	return names, nil
}
