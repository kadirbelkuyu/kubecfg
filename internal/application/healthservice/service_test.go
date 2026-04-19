package healthservice

import (
	"context"
	"testing"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
)

type stubChecker struct {
	results map[string]health.Result
}

func (s stubChecker) Check(ctx context.Context, contextName string) health.Result {
	return s.results[contextName]
}

func (s stubChecker) CheckAll(ctx context.Context, contextNames []string) []health.Result {
	results := make([]health.Result, 0, len(contextNames))
	for _, name := range contextNames {
		results = append(results, s.results[name])
	}
	return results
}

type stubCache struct {
	values map[string]health.Result
}

func (s *stubCache) Get(contextName string) (health.Result, bool) {
	result, ok := s.values[contextName]
	return result, ok
}

func (s *stubCache) Set(result health.Result) {
	s.values[result.ContextName] = result
}

func (s *stubCache) Invalidate(contextName string) {
	delete(s.values, contextName)
}

func (s *stubCache) InvalidateAll() {
	s.values = make(map[string]health.Result)
}

type stubRepo struct {
	cfg *domain.KubeConfig
}

func (s stubRepo) Load(path string) (*domain.KubeConfig, error) {
	return s.cfg, nil
}

func (s stubRepo) Exists(path string) bool {
	return true
}

func (s stubRepo) GetDefaultPath() string {
	return ""
}

func TestCheckContextStoresCache(t *testing.T) {
	cache := &stubCache{values: make(map[string]health.Result)}
	service := New(
		stubChecker{results: map[string]health.Result{"prod": {ContextName: "prod", Status: health.StatusHealthy}}},
		cache,
		stubRepo{cfg: testKubeConfig()},
		"config",
	)

	result, err := service.CheckContext(context.Background(), "prod")
	if err != nil {
		t.Fatalf("CheckContext() error = %v", err)
	}
	if result.Status != health.StatusHealthy {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusHealthy)
	}
	if _, ok := cache.values["prod"]; !ok {
		t.Fatal("cache missing stored result")
	}
}

func TestCheckAllReturnsResults(t *testing.T) {
	cache := &stubCache{values: make(map[string]health.Result)}
	service := New(
		stubChecker{results: map[string]health.Result{
			"prod":    {ContextName: "prod", Status: health.StatusHealthy},
			"staging": {ContextName: "staging", Status: health.StatusDegraded},
		}},
		cache,
		stubRepo{cfg: testKubeConfig()},
		"config",
	)

	results, err := service.CheckAll(context.Background())
	if err != nil {
		t.Fatalf("CheckAll() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(CheckAll()) = %d, want 2", len(results))
	}
}

func TestGetCachedUnknown(t *testing.T) {
	service := New(stubChecker{}, &stubCache{values: make(map[string]health.Result)}, stubRepo{cfg: testKubeConfig()}, "config")
	result := service.GetCached("prod")

	if result.Status != health.StatusUnknown {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusUnknown)
	}
}

func TestGetCachedValue(t *testing.T) {
	cache := &stubCache{values: map[string]health.Result{"prod": {ContextName: "prod", Status: health.StatusHealthy}}}
	service := New(stubChecker{}, cache, stubRepo{cfg: testKubeConfig()}, "config")
	result := service.GetCached("prod")

	if result.Status != health.StatusHealthy {
		t.Fatalf("Status = %v, want %v", result.Status, health.StatusHealthy)
	}
}

func TestGetAllCachedIncludesUnknown(t *testing.T) {
	cache := &stubCache{values: map[string]health.Result{"prod": {ContextName: "prod", Status: health.StatusHealthy}}}
	service := New(stubChecker{}, cache, stubRepo{cfg: testKubeConfig()}, "config")
	results, err := service.GetAllCached(context.Background())
	if err != nil {
		t.Fatalf("GetAllCached() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("len(GetAllCached()) = %d, want 2", len(results))
	}
	if results[1].Status != health.StatusUnknown {
		t.Fatalf("Status = %v, want %v", results[1].Status, health.StatusUnknown)
	}
}

func testKubeConfig() *domain.KubeConfig {
	cfg := domain.NewKubeConfig()
	cfg.Contexts = append(cfg.Contexts,
		domain.ContextEntry{Name: "prod"},
		domain.ContextEntry{Name: "staging"},
	)
	return cfg
}
