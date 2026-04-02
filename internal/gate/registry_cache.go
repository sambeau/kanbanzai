package gate

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/sambeau/kanbanzai/internal/binding"
)

// RegistryCache provides concurrency-safe, mtime-based caching for the
// stage-bindings registry file. It re-reads the file only when its
// modification time changes.
type RegistryCache struct {
	mu          sync.RWMutex
	cached      *binding.BindingFile
	cachedMtime time.Time
	path        string
	loaded      bool
}

// NewRegistryCache creates a cache for the binding registry at the given path.
// It does not read the file immediately; the first call to Get triggers loading.
func NewRegistryCache(path string) *RegistryCache {
	return &RegistryCache{path: path}
}

// Get returns the cached BindingFile, re-reading from disk only when the
// file's mtime has changed. If the file does not exist, it returns (nil, nil)
// so the caller can fall back to hardcoded gates. Load errors are logged and
// treated as a missing file (nil, nil).
func (c *RegistryCache) Get() (*binding.BindingFile, error) {
	// Stat the file outside the lock — this is safe because we double-check
	// under the write lock before actually using the result.
	info, statErr := os.Stat(c.path)

	// Fast path: file gone → clear cache and return nil.
	if statErr != nil {
		if os.IsNotExist(statErr) {
			c.mu.Lock()
			c.cached = nil
			c.loaded = false
			c.cachedMtime = time.Time{}
			c.mu.Unlock()
			return nil, nil
		}
		return nil, statErr
	}

	mtime := info.ModTime()

	// Read-lock: check whether a refresh is needed.
	c.mu.RLock()
	if c.loaded && c.cachedMtime.Equal(mtime) {
		result := c.cached
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	// Write-lock: double-check and refresh if still needed.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-stat under the lock — another goroutine may have refreshed already,
	// or the file may have changed again.
	info, statErr = os.Stat(c.path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			c.cached = nil
			c.loaded = false
			c.cachedMtime = time.Time{}
			return nil, nil
		}
		return nil, statErr
	}
	mtime = info.ModTime()

	if c.loaded && c.cachedMtime.Equal(mtime) {
		return c.cached, nil
	}

	bf, errs := binding.LoadBindingFile(c.path)
	if len(errs) > 0 {
		log.Printf("registry_cache: failed to load %s: %v", c.path, errs)
		// Return nil so caller falls back to hardcoded gates.
		// Keep the old cache state so a subsequent stat change can retry.
		return nil, nil
	}

	c.cached = bf
	c.cachedMtime = mtime
	c.loaded = true
	return bf, nil
}

// LookupPrereqs returns the Prerequisites for the given stage, or (nil, false)
// if the registry is unavailable or the stage has no prerequisites.
func (c *RegistryCache) LookupPrereqs(stage string) (*binding.Prerequisites, bool) {
	bf, err := c.Get()
	if err != nil || bf == nil {
		return nil, false
	}

	sb, ok := bf.StageBindings[stage]
	if !ok || sb == nil || sb.Prerequisites == nil {
		return nil, false
	}
	return sb.Prerequisites, true
}

// LookupOverridePolicy returns the override policy for the given stage.
// When the registry is unavailable or the stage has no explicit policy,
// it returns ("agent", false) as the default.
func (c *RegistryCache) LookupOverridePolicy(stage string) (string, bool) {
	bf, err := c.Get()
	if err != nil || bf == nil {
		return "agent", false
	}

	sb, ok := bf.StageBindings[stage]
	if !ok || sb == nil || sb.Prerequisites == nil || sb.Prerequisites.OverridePolicy == "" {
		return "agent", false
	}
	return sb.Prerequisites.OverridePolicy, true
}
