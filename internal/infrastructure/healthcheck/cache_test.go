package healthcheck

import (
	"sync"
	"testing"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
)

func TestCacheGetMissing(t *testing.T) {
	cache := NewCache()

	if _, ok := cache.Get("missing"); ok {
		t.Fatal("Get() = true for missing key")
	}
}

func TestCacheGetExpired(t *testing.T) {
	cache := NewCache()
	cache.entries["prod"] = cacheEntry{
		result:    health.Result{ContextName: "prod", Status: health.StatusHealthy},
		expiresAt: time.Now().Add(-time.Second),
	}

	if _, ok := cache.Get("prod"); ok {
		t.Fatal("Get() = true for expired key")
	}
}

func TestCacheGetValid(t *testing.T) {
	cache := NewCache()
	want := health.Result{ContextName: "prod", Status: health.StatusHealthy}
	cache.Set(want)

	got, ok := cache.Get("prod")
	if !ok {
		t.Fatal("Get() = false, want true")
	}
	if got.ContextName != want.ContextName || got.Status != want.Status {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}
}

func TestCacheSetOverwrites(t *testing.T) {
	cache := NewCache()
	cache.Set(health.Result{ContextName: "prod", Status: health.StatusHealthy})
	cache.Set(health.Result{ContextName: "prod", Status: health.StatusUnhealthy})

	got, ok := cache.Get("prod")
	if !ok {
		t.Fatal("Get() = false, want true")
	}
	if got.Status != health.StatusUnhealthy {
		t.Fatalf("Status = %v, want %v", got.Status, health.StatusUnhealthy)
	}
}

func TestCacheInvalidate(t *testing.T) {
	cache := NewCache()
	cache.Set(health.Result{ContextName: "prod", Status: health.StatusHealthy})
	cache.Invalidate("prod")

	if _, ok := cache.Get("prod"); ok {
		t.Fatal("Get() = true after Invalidate()")
	}
}

func TestCacheInvalidateAll(t *testing.T) {
	cache := NewCache()
	cache.Set(health.Result{ContextName: "prod", Status: health.StatusHealthy})
	cache.Set(health.Result{ContextName: "staging", Status: health.StatusDegraded})
	cache.InvalidateAll()

	if len(cache.entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(cache.entries))
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	cache := NewCache()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func(index int) {
			defer wg.Done()
			cache.Set(health.Result{
				ContextName: "ctx",
				Status:      health.Status(index % 5),
			})
		}(i)

		go func() {
			defer wg.Done()
			_, _ = cache.Get("ctx")
		}()
	}

	wg.Wait()
}
