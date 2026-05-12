# Orchestrator Prompt: Go Standard Library Modernisation (P67)

## Your role

You are an **orchestrator** working on plan `P67-stdlib-modernisation` in the kanbanzai
repository at `/Users/samphillips/Dev/kanbanzai`.

Your job is to carry this plan from **dev-planning → implementation → review** across four
independent workstreams. You will write the dev-plan, create features and tasks, dispatch
implementer sub-agents, and drive each workstream through to a merged and verified state.

Read `.kbz/roles/orchestrator.yaml` and `.kbz/stage-bindings.yaml` before you begin.
Follow the `orchestrate-development` skill at `.kbz/skills/orchestrate-development/SKILL.md`
throughout.

---

## Plan context

**Plan:** `P67-stdlib-modernisation` — Go Standard Library Modernisation  
**Status:** `shaping` — specification is approved, dev-planning not yet started

The three documents you need are already in `work/P67-stdlib-modernisation/`:

| Document | Path | Status |
|----------|------|--------|
| Research | `work/P67-stdlib-modernisation/P67-research-stdlib-modernisation.md` | Approved |
| Design   | `work/P67-stdlib-modernisation/P67-design-stdlib-modernisation.md`   | Approved |
| Spec     | `work/P67-stdlib-modernisation/P67-spec-stdlib-modernisation.md`     | Draft — **approve this first** |

**First action:** Approve the specification with
`doc(action: "approve", id: "P67-stdlib-modernisation/spec-p67-spec-stdlib-modernisation")`,
then proceed to dev-planning.

---

## What the work is

The codebase declares `go 1.25.0` but does not use any Go 1.21 packages. The work is four
independent workstreams, each touching a disjoint set of files:

### Workstream A — `sort` → `slices` (35 files)

Replace all `"sort"` imports with `"slices"` (and `"cmp"` where comparators are needed)
across 35 non-test files.

**File set** (from the design):
```
internal/actionlog/metrics.go
internal/binding/gen/main.go
internal/binding/registry.go
internal/binding/validate.go
internal/card/constraint_registry.go
internal/checkpoint/checkpoint.go
internal/cleanup/list.go
internal/cli/status/plain.go
internal/context/assemble.go
internal/health/format.go
internal/knowledge/cap_tracker.go
internal/knowledge/compact.go
internal/knowledge/links.go
internal/knowledge/score.go
internal/knowledge/surface.go
internal/mcp/assembly.go
internal/mcp/entity_tool.go
internal/mcp/next_tool.go
internal/registry/extractor.go
internal/registry/render.go
internal/service/doc_audit.go
internal/service/doc_validate.go
internal/service/documents.go
internal/service/entities.go
internal/service/incidents.go
internal/service/knowledge.go
internal/service/migration.go
internal/service/queries.go
internal/service/queue.go
internal/service/retro_synthesis.go
internal/service/retro.go
internal/storage/entity_store.go
internal/validate/lifecycle.go
internal/worktree/store.go
```

**Replacement rules:**

| Old | New | Notes |
|-----|-----|-------|
| `sort.Strings(s)` | `slices.Sort(s)` | Direct drop-in |
| `sort.Ints(s)` | `slices.Sort(s)` | Direct drop-in |
| `sort.Float64s(s)` | `slices.Sort(s)` | Direct drop-in |
| `sort.Slice(s, func(i, j int) bool { … })` | `slices.SortFunc(s, func(a, b T) int { … })` | Comparator changes to `int` return |
| `sort.SliceStable(s, func(i, j int) bool { … })` | `slices.SortStableFunc(s, func(a, b T) int { … })` | Must stay Stable — do NOT use `SortFunc` |

**Critical — comparator contract change:** `slices.SortFunc` requires a comparator
returning `int` (negative/zero/positive), not `bool`. Use `cmp.Compare(a, b)` from the
`"cmp"` package for simple field comparisons. Multi-field comparators use early-return
on non-zero:

```go
// Before
sort.Slice(items, func(i, j int) bool {
    if items[i].Priority != items[j].Priority {
        return items[i].Priority > items[j].Priority
    }
    return items[i].CreatedAt.Before(items[j].CreatedAt)
})

// After
slices.SortFunc(items, func(a, b Item) int {
    if n := cmp.Compare(b.Priority, a.Priority); n != 0 {
        return n
    }
    return a.CreatedAt.Compare(b.CreatedAt)
})
```

**Import changes per file:** remove `"sort"`, add `"slices"`. Add `"cmp"` only where
`sort.Slice`/`sort.SliceStable` is being replaced (not needed for `sort.Strings` etc.).

---

### Workstream B — `"log"` → `"log/slog"` (16 files)

Replace all `"log"` imports with `"log/slog"`, and configure the global logger at the
two binary entry points.

**File set:**
```
cmd/kbz/main.go
internal/context/surfacer.go
internal/docint/store.go
internal/gate/registry_cache.go
internal/mcp/checkpoint_tool.go
internal/mcp/decompose_tool.go
internal/mcp/doc_tool.go
internal/mcp/entity_tool.go
internal/mcp/finish_tool.go
internal/mcp/handler.go
internal/mcp/handoff_tool.go
internal/mcp/merge_tool.go
internal/mcp/server.go
internal/merge/gates.go
internal/service/documents.go
internal/service/entities.go
```

**Entry-point configuration.** Two files need `slog.SetDefault` added:

1. `cmd/kbz/main.go` — add as the first statement in `main()`, before any flag parsing:
   ```go
   slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
       Level: slog.LevelInfo,
   })))
   ```

2. `internal/mcp/server.go` — add during server startup, before any tool is registered.

**Call-site translation rules:**

| Old pattern | New call |
|-------------|----------|
| `log.Printf("[comp] WARNING: msg: %v", err)` | `slog.Warn("msg", "component", "comp", "err", err)` |
| `log.Printf("[comp] ERROR: msg: %v", err)` | `slog.Error("msg", "component", "comp", "err", err)` |
| `log.Printf("[comp] msg: %v", val)` | `slog.Info("msg", "component", "comp", "detail", val)` |
| `log.Printf("WARNING: msg: %v", err)` | `slog.Warn("msg", "err", err)` |

The `[component]` tag prefix becomes a `"component"` key-value attribute. Structured
values (errors, counts, durations) become typed attributes. The `WARNING:`/`ERROR:`
prefix selects the log level; unqualified messages default to `slog.Info`.

**Import change:** replace `"log"` with `"log/slog"`. The entry-point files also need
`"os"` if not already present.

There are **no `log.Fatal` or `log.Panic` calls** in the affected files — all 74
call-sites use `log.Printf`/`log.Println`/`log.Print` only.

---

### Workstream C — Point fixes (3 files)

Three single-file helper functions to delete and replace inline:

| File | Delete | Replace with |
|------|--------|--------------|
| `internal/docint/concepts.go:124` | `stringSliceContains` (7-line for-loop) | `slices.Contains(slice, s)` + add `"slices"` import |
| `internal/context/surfacer.go:190` | `trimTrailingSlash` (6-line guard) | `strings.TrimSuffix(s, "/")` (inline — `"strings"` already imported) |
| `internal/kbzdoctor/doctor.go:279` | `containsMarker` (12-line scanner chain) | `bytes.Contains(data, []byte(marker))` + remove `"bufio"` import |

---

### Workstream D — Internal atomic-write consolidation (1 file)

**File:** `internal/context/refresh.go` only.

Delete the private `atomicWriteFile` function and replace its two call-sites with
`fsutil.WriteFileAtomic(path, data, 0o644)`. Add the `fsutil` package import:
`"github.com/sambeau/kanbanzai/internal/fsutil"`.

The permission change from umask-derived to `0o644` is intentional — these are role/skill
YAML files committed to the repository and must have reproducible permissions.

---

## Dev-planning instructions

Create a **batch** for this plan, then create **four features** — one per workstream.
Use the `write-dev-plan` skill at `.kbz/skills/write-dev-plan/SKILL.md`.

Write a single dev-plan document at
`work/P67-stdlib-modernisation/P67-dev-plan-stdlib-modernisation.md`
covering all four workstreams. Register it with `auto_approve: true`.

**Recommended task granularity for Workstream A:**
35 files is too large for a single task — an implementer with 35 large files will saturate
its context window. Split by package group:

- Task A1: `internal/service/` (11 files — the densest group)
- Task A2: `internal/mcp/` (3 files: `assembly.go`, `entity_tool.go`, `next_tool.go`)
- Task A3: `internal/knowledge/` (5 files)
- Task A4: Remaining files (16 files spanning `actionlog`, `binding`, `card`, `checkpoint`,
  `cleanup`, `cli`, `context`, `health`, `registry`, `storage`, `validate`, `worktree`)

Tasks A1–A4 have no mutual dependencies (disjoint file sets) and can be dispatched in
parallel. Workstreams C and D are each a single small task.

**Workstream B task granularity:**
Split by entry-point responsibility:
- Task B1: Configure `slog.SetDefault` in both entry points (`cmd/kbz/main.go` and
  `internal/mcp/server.go`) — this must land first so the handler exists before call-sites
  are migrated.
- Task B2: Migrate `internal/mcp/` call-sites (11 files)
- Task B3: Migrate remaining call-sites (`internal/context/surfacer.go`,
  `internal/docint/store.go`, `internal/gate/registry_cache.go`,
  `internal/merge/gates.go`, `internal/service/documents.go`,
  `internal/service/entities.go`)

B2 and B3 depend on B1 (entry-point configuration must exist first); B2 and B3 can
then run in parallel.

**Recommended merge order:** C → D → A1–A4 → B1 → B2–B3

Workstreams C and D are tiny, establish a clean green baseline, and validate that
the test suite is healthy before the bulk of the work begins. Workstream A is the
mechanical bulk. Workstream B follows last because it requires the entry-point decision
to be reviewed with full context.

---

## Verification requirements (from the spec)

After **each** workstream is merged to `main`:

1. `go build ./...` must exit 0.
2. `go test ./...` must exit 0 with no new failures.

After **all four** workstreams are merged:

3. `git diff HEAD go.mod go.sum` must produce no output (no dependency changes).

These are non-negotiable gates — do not advance a workstream to reviewing if the build
or test suite is red.

---

## Constraints to enforce during implementation

- **Test files are out of scope.** No `*_test.go` file may be modified.
- **No API changes.** No exported signature may change.
- **Stability must be preserved.** Every `sort.SliceStable` call-site must become
  `slices.SortStableFunc`, never `slices.SortFunc`. This is a correctness requirement,
  not style — substituting unstable for stable is a behavioural change.
- **One PR per workstream.** A workstream PR must not touch files outside its declared
  file set.
- **No new external dependencies.** `go.mod` and `go.sum` must not change.

---

## Workflow state

The plan `P67-stdlib-modernisation` is in `shaping` status. The specification
(`P67-stdlib-modernisation/spec-p67-spec-stdlib-modernisation`) is in `draft` and
needs your approval before dev-planning can begin.

After approving the spec, create a batch under P67, create the four feature entities
(one per workstream), write the dev-plan, decompose into tasks with
`decompose(feature_id: "FEAT-xxx")`, then proceed to development following the
`orchestrate-development` skill.

Use `implementer-go` as the role when dispatching sub-agents with `handoff`. All
implementation work must stay in feature worktrees — do not commit to `main` directly.

---

## Key files for implementers

Sub-agents implementing Workstream A and B will need to read the files they are
changing carefully before editing. Pass the relevant section of this prompt as
context in the `handoff` `instructions` parameter — specifically the replacement
mapping tables and the comparator/slog call-site translation rules. Do not assume
sub-agents will infer the correct `int`-return comparator pattern or the `slog`
level-mapping convention without it being stated explicitly in their handoff.

---

## References

| Document | Path |
|----------|------|
| Research | `work/P67-stdlib-modernisation/P67-research-stdlib-modernisation.md` |
| Design   | `work/P67-stdlib-modernisation/P67-design-stdlib-modernisation.md` |
| Spec     | `work/P67-stdlib-modernisation/P67-spec-stdlib-modernisation.md` |
| Orchestrate-development skill | `.kbz/skills/orchestrate-development/SKILL.md` |
| Write-dev-plan skill | `.kbz/skills/write-dev-plan/SKILL.md` |
| Agents skill | `.agents/skills/kanbanzai-agents/SKILL.md` |
| Implementer-go role | `.kbz/roles/implementer-go.yaml` |
