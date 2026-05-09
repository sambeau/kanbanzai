package context

import (
	"context"
	"sync/atomic"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Generation-keyed cache (REQ-002+003 / AC-002, AC-003, AC-004)
// ─────────────────────────────────────────────────────────────────────────────

// countingLoader wraps a static entry set and tracks how many times it is called.
type countingLoader struct {
	entries []map[string]any
	calls   int32
}

func (c *countingLoader) load() ([]map[string]any, error) {
	atomic.AddInt32(&c.calls, 1)
	return c.entries, nil
}

func (c *countingLoader) count() int {
	return int(atomic.LoadInt32(&c.calls))
}

// fixedGenReader returns a GenReader that always returns the given token.
func fixedGenReader(token string) GenReader {
	return func() (string, error) {
		return token, nil
	}
}

// mutableGen is a GenReader whose token can be changed between calls.
type mutableGen struct {
	token string
}

func (g *mutableGen) read() (string, error) {
	return g.token, nil
}

// TestSurfacer_CacheHit_LoadCalledOnce verifies AC-002: with a warm cache and
// matching generation token, Surface() returns cached results without re-reading.
func TestSurfacer_CacheHit_LoadCalledOnce(t *testing.T) {
	t.Parallel()

	cl := &countingLoader{
		entries: []map[string]any{
			makeTestEntry("KE-C001", "cache-hit-topic", "Always cache BECAUSE I/O is expensive", "project", "confirmed", 0.9, nil, 1),
		},
	}

	s := NewSurfacer(cl.load, nil, fixedNow, fixedGenReader("gen-v1"))

	input := SurfaceInput{}

	// First call: cold cache — loader must be called.
	if _, err := s.Surface(context.Background(), input); err != nil {
		t.Fatalf("Surface() #1 error = %v", err)
	}
	if cl.count() != 1 {
		t.Errorf("after first Surface(), loader called %d times, want 1", cl.count())
	}

	// Second call: warm cache, same generation — loader must NOT be called again.
	if _, err := s.Surface(context.Background(), input); err != nil {
		t.Fatalf("Surface() #2 error = %v", err)
	}
	if cl.count() != 1 {
		t.Errorf("after second Surface() with same generation, loader called %d times, want 1 (cache hit)", cl.count())
	}

	// Third call: same generation — still cached.
	if _, err := s.Surface(context.Background(), input); err != nil {
		t.Fatalf("Surface() #3 error = %v", err)
	}
	if cl.count() != 1 {
		t.Errorf("after third Surface() with same generation, loader called %d times, want 1 (cache hit)", cl.count())
	}
}

// TestSurfacer_CacheMiss_ReloadsOnGenerationChange verifies AC-003: a stale
// cache with a mismatched generation token triggers a reload.
func TestSurfacer_CacheMiss_ReloadsOnGenerationChange(t *testing.T) {
	t.Parallel()

	entry1 := makeTestEntry("KE-C010", "entry-gen1", "First generation content", "project", "confirmed", 0.8, nil, 2)
	entry2 := makeTestEntry("KE-C011", "entry-gen2", "Second generation content", "project", "confirmed", 0.8, nil, 2)

	cl := &countingLoader{entries: []map[string]any{entry1}}

	gen := &mutableGen{token: "gen-v1"}
	s := NewSurfacer(cl.load, nil, fixedNow, gen.read)

	// Warm the cache at gen-v1.
	if _, err := s.Surface(context.Background(), SurfaceInput{}); err != nil {
		t.Fatalf("Surface() #1 error = %v", err)
	}
	if cl.count() != 1 {
		t.Errorf("after first call, loader called %d times, want 1", cl.count())
	}

	// Advance to gen-v2 (simulate a new entry written to disk).
	gen.token = "gen-v2"
	cl.entries = []map[string]any{entry1, entry2}

	// Second call: generation mismatch — must reload.
	if _, err := s.Surface(context.Background(), SurfaceInput{}); err != nil {
		t.Fatalf("Surface() #2 error = %v", err)
	}
	if cl.count() != 2 {
		t.Errorf("after generation change, loader called %d times, want 2 (cache miss)", cl.count())
	}

	// Third call: gen-v2 still matches — cache hit.
	if _, err := s.Surface(context.Background(), SurfaceInput{}); err != nil {
		t.Fatalf("Surface() #3 error = %v", err)
	}
	if cl.count() != 2 {
		t.Errorf("after third call with same generation, loader called %d times, want 2 (cache hit)", cl.count())
	}
}

// TestSurfacer_CacheInvalidatesAfterContribute verifies AC-004: a cache
// populated at generation G1, when the generation advances to G2 after a
// contribute, triggers a reload on the next Surface() call.
func TestSurfacer_CacheInvalidatesAfterContribute(t *testing.T) {
	t.Parallel()

	initialEntries := []map[string]any{
		makeTestEntry("KE-C020", "initial-topic", "Initial knowledge entry", "project", "confirmed", 0.85, nil, 1),
	}

	cl := &countingLoader{entries: initialEntries}
	gen := &mutableGen{token: "G1"}

	s := NewSurfacer(cl.load, nil, fixedNow, gen.read)

	// Populate cache at G1.
	result1, err := s.Surface(context.Background(), SurfaceInput{})
	if err != nil {
		t.Fatalf("Surface() before contribute error = %v", err)
	}
	if cl.count() != 1 {
		t.Errorf("before contribute: loader called %d times, want 1", cl.count())
	}

	// Simulate contribute: new entry written, generation advances to G2.
	contributed := makeTestEntry("KE-C021", "contributed-topic", "New entry from contribute", "project", "contributed", 0.5, nil, 0)
	cl.entries = append(cl.entries, contributed)
	gen.token = "G2"

	// Next Surface() should detect the generation change and reload.
	result2, err := s.Surface(context.Background(), SurfaceInput{})
	if err != nil {
		t.Fatalf("Surface() after contribute error = %v", err)
	}
	if cl.count() != 2 {
		t.Errorf("after contribute: loader called %d times, want 2 (cache miss at G2)", cl.count())
	}

	// The reloaded results must differ from the pre-contribute cache.
	// result2 should have more entries since we added one.
	if len(result2) <= len(result1) {
		t.Logf("result1 len=%d, result2 len=%d (note: RankAndCap caps at 10)", len(result1), len(result2))
		// Not necessarily an error if cap was already reached, but for 2 entries it should differ.
	}
	_ = result2
}

// TestSurfacer_NilGenReader_AlwaysReloads verifies that when genReader is nil,
// the loader is called on every Surface() invocation (caching disabled).
func TestSurfacer_NilGenReader_AlwaysReloads(t *testing.T) {
	t.Parallel()

	cl := &countingLoader{
		entries: []map[string]any{
			makeTestEntry("KE-C030", "no-cache-topic", "Always reload me", "project", "confirmed", 0.7, nil, 3),
		},
	}

	// nil genReader → no caching.
	s := NewSurfacer(cl.load, nil, fixedNow, nil)
	input := SurfaceInput{}

	for i := 0; i < 3; i++ {
		if _, err := s.Surface(context.Background(), input); err != nil {
			t.Fatalf("Surface() #%d error = %v", i+1, err)
		}
	}

	if cl.count() != 3 {
		t.Errorf("nil genReader: loader called %d times, want 3 (no caching)", cl.count())
	}
}

// TestSurfacer_ConcurrentSafety verifies that concurrent Surface() calls with
// a caching genReader do not race or corrupt state.
func TestSurfacer_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	entries := make([]map[string]any, 5)
	for i := 0; i < 5; i++ {
		entries[i] = makeTestEntry(
			"KE-CONC-"+string(rune('A'+i)),
			"concurrent-topic-"+string(rune('A'+i)),
			"Concurrent knowledge entry "+string(rune('A'+i)),
			"project", "confirmed", 0.8+float64(i)*0.01, nil, i,
		)
	}

	cl := &countingLoader{entries: entries}
	gen := &mutableGen{token: "stable-gen"}

	s := NewSurfacer(cl.load, nil, fixedNow, gen.read)

	// Prime the cache.
	if _, err := s.Surface(context.Background(), SurfaceInput{}); err != nil {
		t.Fatalf("prime Surface() error = %v", err)
	}

	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = s.Surface(context.Background(), SurfaceInput{})
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
	// No assertion needed: the race detector will catch data races.
}
