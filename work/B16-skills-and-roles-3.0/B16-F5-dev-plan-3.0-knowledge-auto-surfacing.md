# Implementation Plan: Knowledge Auto-Surfacing

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Feature | FEAT-01KN5-88PEF817 (knowledge-auto-surfacing)                    |
| Spec    | `work/spec/3.0-knowledge-auto-surfacing.md`                        |
| Design  | `work/design/skills-system-redesign-v2.md` §6.3                   |

---

## 1. Overview

This plan decomposes the Knowledge Auto-Surfacing specification into assignable tasks for AI agents. The feature implements the subsystem invoked at step 7 of the context assembly pipeline — it automatically selects, scores, ranks, caps, formats, and returns relevant knowledge entries for inclusion in assembled context.

The existing codebase has:
- `internal/knowledge/` — confidence scoring (`confidence.go`), deduplication (`dedup.go`), compaction (`compact.go`), link tracking (`links.go`), pruning (`prune.go`)
- `internal/service/knowledge.go` — `KnowledgeService` with `List(filters)` returning `[]storage.KnowledgeRecord` (fields map includes `id`, `scope`, `tags`, `status`, `confidence`, timestamps)
- `internal/context/assemble.go` — current assembly logic with `matchesScope()` (scope == "project" || scope == roleID) and `formatKnowledgeEntry()`
- `internal/health/` — `health.go` (Issue, CategoryResult, HealthResult types), `categories.go` (check functions), `check.go` (RunHealthCheck orchestrator), `format.go` (output formatting)
- Context Assembly Pipeline plan — defines the `KnowledgeSurfacer` interface that this feature implements, and the `KnowledgeSurfaceInput`/`KnowledgeSurfaceResult`/`SurfacedEntry`/`ExcludedEntry` types

This feature implements the `KnowledgeSurfacer` interface defined in the Context Assembly Pipeline plan (Task 1), replacing the no-op stub with a real matching engine.

**Scope boundaries (from spec §2.2 — out of scope):**
- Changes to the knowledge entry schema, contribution workflow, or storage format
- The context assembly pipeline itself (this spec defines the subsystem invoked at step 7)
- Manual knowledge querying via `knowledge(action: "list")`
- Knowledge compaction logic (only the recommendation trigger is in scope)
- Token budget estimation of surfaced entries (handled by the pipeline at step 9)

---

## 2. Task Breakdown

### Task 1: Matching Engine — File Path, Tag, and "Always" Matching

**Objective:** Implement the three categorical matching criteria that select candidate knowledge entries from the knowledge base: file path prefix matching against task file paths, tag overlap matching against the resolved role's domain tags, and unconditional inclusion of entries tagged `always` or scoped to `project`. All matched entries are collected into a single deduplicated candidate set.

**Specification references:** FR-001, FR-002, FR-003, FR-011, FR-014, NFR-003

**Input context:**
- `internal/service/knowledge.go` — `KnowledgeService.List()` for loading entries; `KnowledgeFilters` struct
- `internal/knowledge/` — existing helper functions: `GetStatus()`, `GetTier()`
- `internal/context/assemble.go` — existing `matchesScope()` as reference for scope matching (the new implementation extends this with file-path prefix matching)
- Spec §3 FR-001 — scope field is a directory prefix; match when task file path starts with the scope
- Spec §3 FR-002 — tag overlap between entry tags and role domain tags
- Spec §3 FR-003 — entries tagged `always` or scoped `project` always included
- Spec §3 FR-011 — deduplicate across criteria (entry appears once regardless of how many criteria matched)
- Spec §3 FR-014 — exclude entries in `retired` status

**Output artifacts:**
- New file `internal/knowledge/surface.go` — matching functions
- New file `internal/knowledge/surface_test.go` — unit tests:
  - File path: `internal/storage/yaml.go` matches scope `internal/storage/`; `internal/mcp/handoff.go` does not
  - Tag: role tags `[security, go]` match entry tagged `security`; role tags `[testing]` do not
  - Always: entry tagged `always` or scoped `project` matches any task/role
  - Dedup: entry matching by file path AND tag AND `always` appears exactly once
  - Retired filter: `retired` entries excluded; `contributed`, `confirmed`, `disputed` included
  - Empty inputs: no file paths → no file-path matches; no role tags → no tag matches
  - Read-only: knowledge base state unchanged after matching (NFR-003)

**Dependencies:** None — this is the foundational task.

**Interface contract (shared with Task 2 and Task 3):**

```go
// MatchInput contains the parameters for knowledge entry matching.
type MatchInput struct {
    FilePaths []string // task's file paths
    RoleTags  []string // resolved role's domain tags
}

// MatchedEntry is a knowledge entry selected by the matching engine.
type MatchedEntry struct {
    ID          string
    Content     string
    Scope       string
    Tags        []string
    Status      string
    Confidence  float64
    ConfirmedAt time.Time // zero value if never confirmed
    CreatedAt   time.Time
}

// MatchEntries selects candidate entries from the knowledge base.
// Returns a deduplicated set of entries matching any criterion, excluding retired entries.
func MatchEntries(entries []map[string]any, input MatchInput) []MatchedEntry
```

---

### Task 2: Recency-Weighted Scoring and Ranking

**Objective:** Implement the recency-weighted confidence scoring function that combines an entry's confidence score with a recency weight derived from its last confirmation timestamp. Implement the ranking, capping (top 10), within-section ordering (highest score last), and excluded-entry logging.

**Specification references:** FR-004, FR-005, FR-006, FR-007, FR-010, NFR-002

**Input context:**
- `internal/knowledge/confidence.go` — existing `WilsonScore()` function and `clamp()` helper as reference for scoring patterns
- `internal/knowledge/surface.go` (from Task 1) — `MatchedEntry` type with `Confidence` and `ConfirmedAt` fields
- Spec §3 FR-004 — recency weight from last confirmation; unconfirmed entries rank lower
- Spec §3 FR-005 — composite score: confidence is not fully overridden by recency; deterministic
- Spec §3 FR-006 — cap at 10 entries; select top 10 by score
- Spec §3 FR-007 — log excluded entries with ID and topic
- Spec §3 FR-010 — within-section ordering: highest score LAST (recency bias)

**Output artifacts:**
- New file `internal/knowledge/score.go` — scoring and ranking functions
- New file `internal/knowledge/score_test.go` — unit tests:
  - Equal confidence, different recency → more recent ranks higher
  - High-confidence stale entry can outscore low-confidence recent entry
  - Unconfirmed (`contributed`) entry ranks lower than confirmed entry at same confidence
  - Deterministic: same inputs → same scores, same order
  - 15 matches → 10 surfaced, 5 excluded with IDs and topics
  - 8 matches → 8 surfaced, no exclusion log
  - 0 matches → empty result
  - Ordering: last entry in result has highest score; first has lowest
- Benchmark test for scoring 500 entries (NFR-001: < 500ms)

**Dependencies:** Task 1 (matching engine produces `[]MatchedEntry`)

**Interface contract (shared with Task 3):**

```go
// ScoredEntry extends MatchedEntry with the computed composite score.
type ScoredEntry struct {
    MatchedEntry
    Score float64 // recency-weighted confidence, used for ranking
}

// ScoreEntry computes the recency-weighted confidence score for a single entry.
// The now parameter enables deterministic testing.
func ScoreEntry(entry MatchedEntry, now time.Time) float64

// RankAndCap scores all matched entries, sorts by score ascending (lowest first,
// highest last for recency-bias ordering), caps at maxEntries, and returns
// the surfaced entries and any excluded entries.
func RankAndCap(entries []MatchedEntry, now time.Time, maxEntries int) (surfaced []ScoredEntry, excluded []ExcludedEntry)

// ExcludedEntry records an entry that was matched but excluded due to the cap.
type ExcludedEntry struct {
    ID    string
    Topic string
}
```

---

### Task 3: Output Formatting and KnowledgeSurfacer Implementation

**Objective:** Implement the "Always/Never X BECAUSE Y" output formatting for surfaced entries, the section omission logic when no entries match, and the concrete `KnowledgeSurfacer` implementation that wires matching, scoring, and formatting into the interface defined by the Context Assembly Pipeline. This is the integration task that produces the component the pipeline calls at step 7.

**Specification references:** FR-008, FR-009, FR-010, FR-013, NFR-002, NFR-004

**Input context:**
- `internal/knowledge/surface.go` (from Task 1) — `MatchEntries()` function
- `internal/knowledge/score.go` (from Task 2) — `RankAndCap()` function, `ScoredEntry` type
- Context Assembly Pipeline plan, Task 1 — `KnowledgeSurfacer` interface, `KnowledgeSurfaceInput`, `KnowledgeSurfaceResult`, `SurfacedEntry` types
- `internal/context/assemble.go` — existing `formatKnowledgeEntry()` as reference for entry formatting patterns
- Spec §3 FR-008 — format as "Always/Never X BECAUSE Y"; entries without BECAUSE rendered with directive only
- Spec §3 FR-009 — position 8 in assembled context (handled by the pipeline, but this task ensures the surfacer returns data suitable for that position)
- Spec §3 FR-013 — zero entries → empty result (no placeholder or heading)
- Spec §3 NFR-004 — missing/empty knowledge base → empty result, no error

**Output artifacts:**
- New file `internal/knowledge/format.go` — directive formatting function
- New file `internal/knowledge/format_test.go` — unit tests:
  - Entry with "Always X BECAUSE Y" content rendered verbatim
  - Entry with "Never use X" content rendered with directive only (no empty BECAUSE clause)
  - Entry with rationale in metadata but no BECAUSE in content → BECAUSE appended
- New file `internal/knowledge/surfacer.go` — concrete `KnowledgeSurfacer` implementation
- New file `internal/knowledge/surfacer_test.go` — integration tests:
  - Full pipeline: entries loaded → matched → scored → capped → formatted → returned
  - Zero matches → empty `Entries` slice, no error
  - Missing knowledge base directory → empty result, no error
  - Empty knowledge base → empty result, no error
  - Excluded entries appear in `Excluded` field with IDs and topics
  - Result is deterministic: two identical calls produce identical output (NFR-002)
- Replace `internal/context/surfacer_stub.go` (the no-op stub from Context Assembly Pipeline Task 5) — the real implementation now satisfies the `KnowledgeSurfacer` interface

**Dependencies:** Task 1 (matching), Task 2 (scoring/ranking)

**Interface contract with Context Assembly Pipeline:** The surfacer implements the `KnowledgeSurfacer` interface from the pipeline plan:

```go
// Surfacer implements the KnowledgeSurfacer interface from the context assembly pipeline.
type Surfacer struct {
    knowledgeSvc *service.KnowledgeService
    capTracker   *CapTracker // optional, nil-safe
    now          func() time.Time // injectable for deterministic testing
}

func NewSurfacer(knowledgeSvc *service.KnowledgeService, capTracker *CapTracker, now func() time.Time) *Surfacer

// Surface implements KnowledgeSurfacer.Surface.
// Loads all non-retired entries, matches by file path/tag/always criteria,
// scores and ranks, caps at 10, formats as directives, and returns the result.
func (s *Surfacer) Surface(input KnowledgeSurfaceInput) (*KnowledgeSurfaceResult, error)
```

---

### Task 4: Health Compaction Recommendation

**Objective:** Extend the `health` tool to flag a compaction recommendation when the auto-surfacing cap (10 entries) is routinely exceeded. "Routinely" means 3 or more consecutive assemblies for the same scope where the cap was hit. This requires a lightweight persistent counter per scope and a new health check category.

**Specification references:** FR-012

**Input context:**
- `internal/health/health.go` — `Issue`, `CategoryResult`, `HealthResult` types
- `internal/health/categories.go` — existing check functions (`CheckKnowledgeStaleness`, `CheckKnowledgeTTL`, etc.) as patterns for new check functions
- `internal/health/check.go` — `RunHealthCheck()` orchestrator and `CheckOptions` struct
- `internal/health/format.go` — `FormatHealthResult()` for output
- Spec §3 FR-012 — 3 consecutive cap-hits for same scope → recommendation; below-cap assembly resets counter

**Output artifacts:**
- New file `internal/knowledge/cap_tracker.go` — persistent counter per scope for consecutive cap-hits
- New file `internal/knowledge/cap_tracker_test.go` — unit tests:
  - 3 consecutive cap-hits → tracker reports scope as needing compaction
  - 2 cap-hits followed by below-cap → counter resets, no recommendation
  - Multiple independent scopes tracked separately
  - Counter persists across invocations (file-based or state-based)
- New function in `internal/health/categories.go` — `CheckKnowledgeCapSaturation()` that reads the cap tracker and produces warnings
- Modified `internal/health/check.go` — wire `CheckKnowledgeCapSaturation` into `RunHealthCheck()`
- Tests for the new health check category

**Dependencies:** Task 3 (the surfacer must record cap-hit events for the tracker to consume)

**Interface contract with Task 3:** The surfacer must call the cap tracker after each surfacing operation:

```go
// CapTracker records cap-hit events per scope for health compaction recommendations.
type CapTracker struct {
    stateRoot string // path to .kbz/state/ or equivalent
}

func NewCapTracker(stateRoot string) *CapTracker

// RecordAssembly records whether the cap was hit for a given scope.
// If capHit is false, the consecutive counter for that scope resets to 0.
func (t *CapTracker) RecordAssembly(scope string, capHit bool) error

// ScopesNeedingCompaction returns scopes with 3+ consecutive cap-hits.
func (t *CapTracker) ScopesNeedingCompaction() ([]ScopeCompactionInfo, error)

type ScopeCompactionInfo struct {
    Scope           string
    ConsecutiveHits int
}

// CheckKnowledgeCapSaturation checks for scopes that routinely exceed the
// knowledge auto-surfacing cap and recommends compaction.
func CheckKnowledgeCapSaturation(tracker *CapTracker) CategoryResult
```

---

## 3. Dependency Graph

```
Task 1: Matching Engine (file path, tag, always, dedup, retired filter)
  │
  ▼
Task 2: Recency-Weighted Scoring and Ranking (score, cap, order, exclusion log)
  │
  ▼
Task 3: Output Formatting and KnowledgeSurfacer (format, integration, stub replacement)
  │
  ▼
Task 4: Health Compaction Recommendation (cap tracker, health check category)
```

**Parallelism:** This feature has a linear dependency chain — each task builds on the previous one. However, Task 4 (health) is loosely coupled and could begin implementation of the `CapTracker` and health check function in parallel with Task 3, provided the interface contract for `RecordAssembly` is agreed upon. The practical parallelism opportunity is:

1. Task 1 (serial — foundational matching)
2. Task 2 (serial — requires Task 1's `MatchedEntry` type)
3. Task 3 + Task 4 `CapTracker` struct (partial parallel — Task 4's tracker can be built against the interface contract while Task 3 integrates the surfacer)
4. Task 4 health wiring (serial — requires Task 3 to call `RecordAssembly`)

**Execution order:**
1. Task 1
2. Task 2
3. Tasks 3 and 4 (Task 4's `CapTracker` implementation can overlap with Task 3; Task 4's health check wiring completes after Task 3)

---

## 4. Interface Contracts

### 4.1 Matching Engine → Scoring (Task 1 → Task 2)

Task 1 produces `[]MatchedEntry` containing all deduplicated candidates with their raw confidence scores and timestamps. Task 2 consumes this slice to compute composite scores. The `MatchedEntry` struct is the handoff type:

```go
type MatchedEntry struct {
    ID          string
    Content     string
    Scope       string
    Tags        []string
    Status      string
    Confidence  float64
    ConfirmedAt time.Time
    CreatedAt   time.Time
}
```

### 4.2 Scoring → Surfacer (Task 2 → Task 3)

Task 2 produces `[]ScoredEntry` (sorted ascending by score — lowest first, highest last for recency-bias ordering) and `[]ExcludedEntry`. Task 3 formats the scored entries into directive strings and wraps them in the `KnowledgeSurfaceResult` type from the pipeline.

### 4.3 Surfacer → Context Assembly Pipeline (Task 3 → Pipeline Task 5)

The surfacer implements the `KnowledgeSurfacer` interface defined in the Context Assembly Pipeline plan (Task 1). The pipeline injects the surfacer and calls `Surface()` at step 7. The surfacer replaces `internal/context/surfacer_stub.go`.

```go
// The pipeline calls:
result, err := surfacer.Surface(KnowledgeSurfaceInput{
    FilePaths: taskFilePaths,
    RoleTags:  resolvedRole.Tags,
})
// result.Entries goes to position 8; result.Excluded goes to diagnostics.
```

### 4.4 Surfacer → Cap Tracker (Task 3 → Task 4)

After each surfacing operation, the surfacer calls `CapTracker.RecordAssembly()` with the dominant scope and whether the cap was hit. The tracker is an optional dependency — if nil, the surfacer skips recording (graceful degradation).

### 4.5 Cap Tracker → Health Check (Task 4)

`CheckKnowledgeCapSaturation()` reads from the `CapTracker` and produces `CategoryResult` issues. It is wired into `RunHealthCheck()` alongside existing knowledge health checks.

---

## 5. Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| FR-001 (File path matching) | Task 1 |
| FR-002 (Tag matching) | Task 1 |
| FR-003 (Explicit "always" entries) | Task 1 |
| FR-004 (Recency weighting) | Task 2 |
| FR-005 (Recency-weighted confidence score) | Task 2 |
| FR-006 (Entry cap — 10 max) | Task 2 |
| FR-007 (Excluded entry logging) | Task 2 |
| FR-008 (Output formatting — "Always/Never X BECAUSE Y") | Task 3 |
| FR-009 (Position 8 in assembled context) | Task 3 |
| FR-010 (Within-section ordering — highest last) | Task 2 |
| FR-011 (Deduplication across criteria) | Task 1 |
| FR-012 (Health compaction recommendation) | Task 4 |
| FR-013 (Omission when no entries match) | Task 3 |
| FR-014 (Filtering of retired entries) | Task 1 |
| NFR-001 (Surfacing latency < 500ms) | Task 2 (benchmark), Task 3 (benchmark) |
| NFR-002 (Deterministic results) | Task 2, Task 3 |
| NFR-003 (Backward compatibility — read-only) | Task 1 |
| NFR-004 (Graceful degradation) | Task 3 |