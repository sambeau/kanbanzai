# Implementation Plan: Go Standard Library Modernisation (P67)

| Field  | Value                                   |
|--------|-----------------------------------------|
| Date   | 2026-05-12                              |
| Status | approved |
| Author | orchestrator (sambeau)                  |

## Overview

This plan modernises the kanbanzai codebase to use Go standard library packages
introduced in Go 1.21 (`slices`, `cmp`, `log/slog`) and consolidates two classes of
hand-rolled helper functions with direct stdlib calls. The work is purely mechanical:
no business logic changes, no API surface changes, and no new dependencies.

Four workstreams execute in sequence by cohort (C/D first, then A, then B) to avoid
merge conflicts on shared files. All nine tasks are defined and scoped to disjoint
file sets within each cohort.

## Scope

This plan implements the requirements defined in
`work/P67-stdlib-modernisation/P67-spec-stdlib-modernisation.md`
(`P67-stdlib-modernisation/spec-p67-spec-stdlib-modernisation`, approved 2026-05-12).

It covers four independent workstreams across the kanbanzai codebase:

- **Workstream A (B72-F1):** Replace `"sort"` with `"slices"` (and `"cmp"` where needed)
  across 35 non-test files.
- **Workstream B (B72-F2):** Replace `"log"` with `"log/slog"` and configure a global
  structured-logging handler at two entry points across 16 files.
- **Workstream C (B72-F3):** Delete three hand-rolled helper functions and replace their
  call-sites inline with stdlib equivalents (`slices.Contains`, `strings.TrimSuffix`,
  `bytes.Contains`).
- **Workstream D (B72-F4):** Delete the private `atomicWriteFile` helper in
  `internal/context/refresh.go` and replace its two call-sites with
  `fsutil.WriteFileAtomic`.

**Out of scope:** No test files (`*_test.go`) may be modified. No exported signatures
may change. `go.mod` and `go.sum` must remain unmodified.

---

## Task Breakdown

### Task A1: sort→slices — internal/service/ (11 files)

- **Description:** Replace all `sort.*` calls with `slices.*` / `cmp.*` equivalents in
  the 11 service-layer files.
- **Deliverable:** `internal/service/` builds cleanly with `sort` import removed from
  all 11 files; `go test ./internal/service/...` passes.
- **Depends on:** None (independent).
- **Effort:** Large (highest density of `sort.Slice` calls with multi-field comparators).
- **Spec requirement:** FR-001, FR-002, AC-001, AC-003.
- **Files:**
  `internal/service/doc_audit.go`, `internal/service/doc_validate.go`,
  `internal/service/documents.go`, `internal/service/entities.go`,
  `internal/service/incidents.go`, `internal/service/knowledge.go`,
  `internal/service/migration.go`, `internal/service/queries.go`,
  `internal/service/queue.go`, `internal/service/retro_synthesis.go`,
  `internal/service/retro.go`

### Task A2: sort→slices — internal/mcp/ (3 files)

- **Description:** Replace all `sort.*` calls with `slices.*` in the three MCP-layer
  files.
- **Deliverable:** `internal/mcp/assembly.go`, `internal/mcp/entity_tool.go`,
  `internal/mcp/next_tool.go` build cleanly with `sort` import removed; tests pass.
- **Depends on:** None (independent).
- **Effort:** Small.
- **Spec requirement:** FR-001, FR-002, AC-001.
- **Files:**
  `internal/mcp/assembly.go`, `internal/mcp/entity_tool.go`, `internal/mcp/next_tool.go`

### Task A3: sort→slices — internal/knowledge/ (5 files)

- **Description:** Replace all `sort.*` calls with `slices.*` in the five knowledge
  files.
- **Deliverable:** `internal/knowledge/` builds cleanly; tests pass.
- **Depends on:** None (independent).
- **Effort:** Medium.
- **Spec requirement:** FR-001, FR-002, AC-001.
- **Files:**
  `internal/knowledge/cap_tracker.go`, `internal/knowledge/compact.go`,
  `internal/knowledge/links.go`, `internal/knowledge/score.go`,
  `internal/knowledge/surface.go`

### Task A4: sort→slices — remaining 16 files

- **Description:** Replace all `sort.*` calls with `slices.*` in the remaining files
  spanning `actionlog`, `binding`, `card`, `checkpoint`, `cleanup`, `cli/status`,
  `context`, `health`, `registry`, `storage`, `validate`, and `worktree`.
- **Deliverable:** All 16 files build cleanly; tests pass.
- **Depends on:** None (independent).
- **Effort:** Large (16 files, varied patterns).
- **Spec requirement:** FR-001, FR-002, AC-001.
- **Files:**
  `internal/actionlog/metrics.go`, `internal/binding/gen/main.go`,
  `internal/binding/registry.go`, `internal/binding/validate.go`,
  `internal/card/constraint_registry.go`, `internal/checkpoint/checkpoint.go`,
  `internal/cleanup/list.go`, `internal/cli/status/plain.go`,
  `internal/context/assemble.go`, `internal/health/format.go`,
  `internal/registry/extractor.go`, `internal/registry/render.go`,
  `internal/storage/entity_store.go`, `internal/validate/lifecycle.go`,
  `internal/worktree/store.go`

  *Note: 15 files listed here (binding/gen/main.go is the 16th).*

### Task B1: slog entry-point configuration

- **Description:** Add `slog.SetDefault(...)` in `cmd/kbz/main.go` (as first statement
  in `main()`) and in `internal/mcp/server.go` (before first tool registration).
  Replace `"log"` import with `"log/slog"` in both files. Add `"os"` import to
  `cmd/kbz/main.go` if not already present.
- **Deliverable:** Both entry-point files compile; the structured handler is configured
  before any log call sites are reached.
- **Depends on:** None (independent baseline for B2 and B3).
- **Effort:** Small.
- **Spec requirement:** FR-003, FR-005, AC-002.

### Task B2: slog migration — internal/mcp/ (11 files)

- **Description:** Replace all `log.Printf`/`log.Println`/`log.Print` calls with
  structured `slog.*` calls in the 11 mcp-layer files. Apply the level-mapping convention:
  `WARNING:` prefix → `slog.Warn`, `ERROR:` prefix → `slog.Error`, unqualified →
  `slog.Info`. `[component]` tag prefix becomes a `"component"` key-value attribute.
- **Deliverable:** All 11 mcp files compile with `"log"` removed; structured calls
  use correct levels; tests pass.
- **Depends on:** Task B1 (entry-point handler must exist first).
- **Effort:** Large (11 files, translation of 74 call-sites total across B2+B3).
- **Spec requirement:** FR-003, FR-004, AC-002, AC-004.
- **Files:**
  `internal/mcp/checkpoint_tool.go`, `internal/mcp/decompose_tool.go`,
  `internal/mcp/doc_tool.go`, `internal/mcp/entity_tool.go`,
  `internal/mcp/finish_tool.go`, `internal/mcp/handler.go`,
  `internal/mcp/handoff_tool.go`, `internal/mcp/merge_tool.go`,
  `internal/mcp/server.go`

  *Note: server.go is shared with B1 but B1 only touches the SetDefault line.*

### Task B3: slog migration — remaining 6 files

- **Description:** Replace all `log.*` calls with `slog.*` calls in
  `internal/context/surfacer.go`, `internal/docint/store.go`,
  `internal/gate/registry_cache.go`, `internal/merge/gates.go`,
  `internal/service/documents.go`, `internal/service/entities.go`.
- **Deliverable:** All 6 files compile with `"log"` removed; correct slog levels used.
- **Depends on:** Task B1 (entry-point handler must exist first).
- **Effort:** Medium.
- **Spec requirement:** FR-003, FR-004, AC-002, AC-004.

### Task C1: Point fixes — delete three hand-rolled helpers

- **Description:**
  1. `internal/docint/concepts.go`: delete `stringSliceContains` helper (7-line for-loop)
     and replace its call-site with `slices.Contains(slice, s)`; add `"slices"` import.
  2. `internal/context/surfacer.go`: delete `trimTrailingSlash` helper (6-line guard)
     and inline `strings.TrimSuffix(s, "/")` at its call-site; no new import needed.
  3. `internal/kbzdoctor/doctor.go`: delete `containsMarker` helper (12-line scanner
     chain) and replace call-site with `bytes.Contains(data, []byte(marker))`; remove
     `"bufio"` import.
- **Deliverable:** Three files compile; all helpers are gone; no behaviour change.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** FR-006, AC-005.

### Task D1: Atomic-write consolidation

- **Description:** In `internal/context/refresh.go`, delete the private
  `atomicWriteFile` function and replace its two call-sites with
  `fsutil.WriteFileAtomic(path, data, 0o644)`. Add import
  `"github.com/sambeau/kanbanzai/internal/fsutil"`.
- **Deliverable:** `internal/context/refresh.go` compiles; `atomicWriteFile` no longer
  exists; tests pass; no other files changed.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** FR-007, AC-006.

---

## Dependency Graph

```
Task A1  (no dependencies)  ─┐
Task A2  (no dependencies)  ─┤
Task A3  (no dependencies)  ─┤── can all run in parallel
Task A4  (no dependencies)  ─┘

Task B1  (no dependencies)   ── must land first in Workstream B
Task B2  → depends on B1     ─┐── can run in parallel after B1
Task B3  → depends on B1     ─┘

Task C1  (no dependencies)
Task D1  (no dependencies)
```

**Parallel groups:**
- Wave 1: `[C1, D1]` — small, no shared files, establish clean green baseline
- Wave 2: `[A1, A2, A3, A4]` — independent tasks in disjoint file sets
- Wave 3: `[B1]` — entry-point baseline (must land before B2/B3)
- Wave 4: `[B2, B3]` — call-site translation, can run in parallel

**Critical path:** C1 → A1 → B1 → B2 (longest dependency-free chain through merge
order; true critical path is through the merge schedule rather than task dependencies).

**File-overlap note:** Workstreams A and B both touch `internal/mcp/entity_tool.go`,
`internal/service/documents.go`, and `internal/service/entities.go`. Workstream C and
Workstream B both touch `internal/context/surfacer.go`. The merge order (C → D → A →
B) serialises these at the branch level, so parallel implementation in separate worktrees
is safe; the overlap is resolved at merge time.

---

## Merge Schedule

Features must be merged in the following order to avoid conflicts on shared files:

**Cohort 1 (C and D — can merge in any order):**
- B72-F3 (ws-c-point-fixes)
- B72-F4 (ws-d-atomic-write)

**Cohort 2 (A tasks — can merge in any order after Cohort 1):**
- B72-F1 (ws-a-sort-to-slices): merge A1–A4 within a single branch, or merge the
  feature branch once all A tasks are done.

**Cohort 3 (B — must merge after Cohort 2):**
- B72-F2 (ws-b-log-to-slog): B1 task must land before B2/B3 within the branch.

`conflict(action: "check", feature_ids: ["FEAT-01KREH8S3WFC4","FEAT-01KREH8TF8JZ8"])` —
Cohort 1 verified safe (disjoint files).

---

## Interface Contracts

No new interface contracts are introduced by this plan. All changes are internal
implementation replacements with identical external behaviour:

| Package | Existing contract | Post-migration contract |
|---------|-------------------|-------------------------|
| `sort.Strings(s)` | Sorts `[]string` in ascending order, in-place | `slices.Sort(s)` — identical behaviour |
| `sort.Slice(s, less)` | Sorts `s` using `less(i,j) bool` comparator | `slices.SortFunc(s, cmp func(a,b T) int)` — identical result, comparator contract changes from bool to int |
| `sort.SliceStable(s, less)` | Stable sort using `less(i,j) bool` | `slices.SortStableFunc(s, cmp)` — stability preserved, comparator contract changes |
| `log.Printf(...)` | Unstructured stderr logging | `slog.Info/Warn/Error(...)` — structured key-value logging, same destination |
| `atomicWriteFile(path, data, perm)` | Private atomic write | `fsutil.WriteFileAtomic(path, data, 0o644)` — same semantics, fixed 0o644 permission |

## Traceability Matrix

| Spec Requirement | Task(s) | Acceptance Criterion |
|------------------|---------|----------------------|
| FR-001: Replace `sort` with `slices` (non-stable) | A1, A2, A3, A4 | AC-001, AC-003 |
| FR-002: Replace `sort` with `slices`+`cmp` (stable) | A1, A2, A3, A4 | AC-001, AC-003 |
| FR-003: Configure `slog` global handler at entry points | B1 | AC-002 |
| FR-004: Translate `log.*` call-sites to `slog.*` | B2, B3 | AC-002, AC-004 |
| FR-005: `slog.SetDefault` before any log call | B1 | AC-002 |
| FR-006: Delete hand-rolled helpers (contains, trim, scanner) | C1 | AC-005 |
| FR-007: Replace `atomicWriteFile` with `fsutil.WriteFileAtomic` | D1 | AC-006 |
| NFR-001: `go build ./...` passes after each workstream | All | AC-007 |
| NFR-002: `go test ./...` passes after each workstream | All | AC-008 |
| NFR-003: `go.mod` and `go.sum` unchanged | All | AC-009 |
| NFR-004: No `*_test.go` files modified | All | AC-010 |

## Risk Assessment

### Risk: Comparator contract change breaks sort stability

- **Probability:** Medium
- **Impact:** High (silent behavioural regression — ordering may change)
- **Mitigation:** Every `sort.SliceStable` call-site must become
  `slices.SortStableFunc` (never `slices.SortFunc`). Implementers must grep for
  `SliceStable` in their task file set and confirm zero instances remain after migration.
  Reviewers must flag any `SliceStable → SortFunc` substitution as a blocking finding.
- **Affected tasks:** A1, A2, A3, A4.

### Risk: slog level mis-mapping produces silent information loss

- **Probability:** Medium
- **Impact:** Medium (operational: warnings/errors silently demoted to Info)
- **Mitigation:** The level-mapping table in this plan is normative. Implementers must
  apply it mechanically. Reviewers must audit each call-site translation for correct
  level selection.
- **Affected tasks:** B2, B3.

### Risk: `cmp` package added where not needed

- **Probability:** Low
- **Impact:** Low (unnecessary import — caught by `go build`)
- **Mitigation:** Add `"cmp"` import only in files where `sort.Slice` or
  `sort.SliceStable` is being replaced. Files that only use `sort.Strings`,
  `sort.Ints`, or `sort.Float64s` do not need `"cmp"`.
- **Affected tasks:** A1, A2, A3, A4.

### Risk: go.mod modified by adding a new import path

- **Probability:** Low
- **Impact:** High (blocks merge gate)
- **Mitigation:** `"slices"`, `"cmp"`, and `"log/slog"` are all part of the Go standard
  library (Go 1.21+) and require no `go get`. The `fsutil` package is already in the
  monorepo. Post-implementation, run `git diff HEAD go.mod go.sum` and confirm no output.
- **Affected tasks:** All.

### Risk: Test files inadvertently modified

- **Probability:** Low
- **Impact:** Medium (out-of-scope change, blocked by review)
- **Mitigation:** Implementers must not touch any `*_test.go` file. Reviewers must
  verify `get_files` on the PR contains no test files.
- **Affected tasks:** All.

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001: No `"sort"` imports remain in non-test files | `go build ./...` exits 0; `grep -r '"sort"' --include="*.go" --exclude="*_test.go" .` returns empty | A1, A2, A3, A4 |
| AC-002: No `"log"` imports remain in non-test files | `go build ./...` exits 0; `grep -r '"log"' --include="*.go" --exclude="*_test.go" .` returns empty | B1, B2, B3 |
| AC-003: `sort.SliceStable` → `slices.SortStableFunc` only | Code inspection: `grep -r "SortFunc" --include="*.go"` reviewed for any former `SliceStable` sites | A1, A2, A3, A4 |
| AC-004: slog levels correctly mapped | Code inspection: reviewers verify level-mapping table applied at each call-site | B2, B3 |
| AC-005: Three helpers deleted | `grep -r "stringSliceContains\|trimTrailingSlash\|containsMarker" --include="*.go"` returns empty | C1 |
| AC-006: `atomicWriteFile` deleted | `grep -r "atomicWriteFile" --include="*.go"` returns empty | D1 |
| AC-007: `go build ./...` passes after each workstream merge | CI gate: run `go build ./...` in PR check | All |
| AC-008: `go test ./...` passes after each workstream merge | CI gate: run `go test ./...` in PR check | All |
| AC-009: `go.mod` and `go.sum` unchanged | `git diff HEAD go.mod go.sum` produces no output | All |
| AC-010: No `*_test.go` files modified | PR file list contains no test files | All |
