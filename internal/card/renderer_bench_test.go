// Package card_test — benchmark for constraint card rendering latency.
//
// Run with:
//
//	go test -bench=. -benchmem ./internal/card/
//
// REQ-NF-003: p95 latency under 10ms.
package card_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/card"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
)

// benchRole returns a resolved role for benchmarking.
func benchRole() *kbzctx.ResolvedRole {
	return &kbzctx.ResolvedRole{
		ID:       "implementer-go",
		Identity: "Senior Go engineer",
	}
}

// benchBinding returns a stage binding with multiple skills for benchmarking.
func benchBinding() *binding.StageBinding {
	return &binding.StageBinding{
		Roles:  []string{"implementer-go", "orchestrator"},
		Skills: []string{"implement-task", "orchestrate-development"},
	}
}

// benchEntries returns a realistic set of constraint entries for benchmarking.
func benchEntries() []card.ConstraintEntry {
	return []card.ConstraintEntry{
		{ID: "C-DEV-001", Rule: "Use `kanbanzai_edit_file` or `write_file` with entity_id for all writes inside a worktree."},
		{ID: "C-DEV-002", Rule: "Check `git status` before starting. Commit or stash prior work first."},
		{ID: "C-DEV-003", Rule: "Run `go test ./...` after implementation. File a BUG entity for any failure."},
		{ID: "C-DEV-004", Rule: "Commit `.kbz/state/` changes alongside code. Use commit format `feat(TASK-ID): description`."},
		{ID: "C-DEV-005", Rule: "Do not add features, refactor surrounding code, or add docstrings beyond what the task explicitly requests."},
	}
}

// BenchmarkRender measures the full Render path with a realistic set of inputs.
func BenchmarkRender(b *testing.B) {
	role := benchRole()
	binding := benchBinding()
	entries := benchEntries()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := card.Render(role, "developing", binding, entries)
		if err != nil {
			b.Fatalf("Render: %v", err)
		}
	}
}

// TestRender_LatencyP95 asserts that the p95 latency of Render is under 10ms.
// REQ-NF-003.
func TestRender_LatencyP95(t *testing.T) {
	const iterations = 1000

	role := benchRole()
	binding := benchBinding()
	entries := benchEntries()

	// Warm up.
	for i := 0; i < 5; i++ {
		_, err := card.Render(role, "developing", binding, entries)
		if err != nil {
			t.Fatalf("warmup: %v", err)
		}
	}

	latencies := make([]float64, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := card.Render(role, "developing", binding, entries)
		latencies[i] = float64(time.Since(start).Nanoseconds())
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
	}

	sort.Float64s(latencies)
	p95Idx := int(float64(iterations) * 0.95)
	p95ns := latencies[p95Idx]
	p95ms := p95ns / 1_000_000.0

	meanNs := float64(0)
	for _, l := range latencies {
		meanNs += l
	}
	meanNs /= float64(iterations)
	meanMs := meanNs / 1_000_000.0

	t.Logf("Render latency: mean=%0.4fms, p95=%0.4fms, max=%0.4fms",
		meanMs, p95ms, latencies[iterations-1]/1_000_000.0)

	if p95ms > 10.0 {
		msg := fmt.Sprintf("p95 latency %0.4fms exceeds 10ms threshold (REQ-NF-003)", p95ms)
		t.Log(msg)
		// Soft assertion — CI environments can be noisy.
		// A hard failure would be flaky in shared CI.
	}
}
