# Design: Go Standard Library Modernisation

| Field  | Value                              |
|--------|------------------------------------|
| Date   | 2026-05-12                         |
| Status | approved |
| Author | Claude (Sonnet 4.6)                |
| Plan   | P67-stdlib-modernisation           |
| Source | P67-research-stdlib-modernisation  |

---

## Overview

This design covers the replacement of hand-rolled helpers and legacy standard library packages
with their modern Go equivalents across the kanbanzai codebase. The work is scoped to four
independent workstreams: (A) migrating `"sort"` to `"slices"` across 35 files, (B) migrating
`"log"` to `"log/slog"` across 16 files, (C) deleting three single-file helper functions that
reduplicate one stdlib call each, and (D) consolidating a duplicated atomic-write helper. All
changes are purely mechanical — no public API surfaces, data formats, or feature behaviours
are altered.

---

## Goals and Non-Goals

**Goals:**
- Replace all `"sort"` imports with `"slices"` (and `"cmp"` where comparators are needed).
- Replace all `"log"` imports with `"log/slog"`, configured at the two binary entry points.
- Delete `stringSliceContains`, `trimTrailingSlash`, and `containsMarker` in favour of the
  stdlib calls they wrap.
- Consolidate the duplicate `atomicWriteFile` in `internal/context/refresh.go` onto
  `internal/fsutil.WriteFileAtomic`.
- Ensure `go build ./...` and the full test suite pass after each workstream is merged.

**Non-Goals:**
- Replacing any custom code that has no direct stdlib equivalent (`deduplicateStrings`,
  `clamp`, `parseSemver`, `GenerateTSID13`, `buildinfo`).
- Introducing a custom `slog.Handler` abstraction or per-request logging context.
- Migrating test files — test helpers that call `log.Printf` are out of scope.
- Any performance tuning or algorithmic changes beyond what the API substitution provides.
- Changing any public API surface or data model.

---

## Problem and Motivation

The kanbanzai codebase systematically bypasses modern Go standard library packages that have
been available since Go 1.21 — despite declaring `go 1.25.0` in `go.mod`. Two patterns
dominate the problem surface:

**Pattern 1 — `sort` instead of `slices` (35 files, 52 call-sites).**
The entire codebase uses the pre-generics `"sort"` package, which requires index-based
`func(i, j int) bool` comparators that operate by side-effect on the original slice. The
`"slices"` package, stable since Go 1.21, provides typed, value-based comparators that are
easier to read and less error-prone. The codebase imports `"sort"` in 35 non-test files and
does not import `"slices"` anywhere.

**Pattern 2 — `"log"` instead of `"log/slog"` (16 files, 74 call-sites).**
All diagnostic output goes through the unstructured `log.Printf` family. Structure is hand-
encoded in format strings using ad-hoc conventions (`[component] WARNING: key: value`), making
log output inconsistent, non-machine-parseable, and resistant to log-level filtering. The
`log/slog` package, stable since Go 1.21, provides structured, levelled logging with no added
dependency. The codebase does not import `"log/slog"` anywhere.

**Pattern 3 — hand-rolled single-line helpers (3 call-sites).**
Three functions exist whose entire bodies reduce to one stdlib call:
- `stringSliceContains` in `internal/docint/concepts.go` reimplements `slices.Contains`.
- `trimTrailingSlash` in `internal/context/surfacer.go` reimplements `strings.TrimSuffix(s, "/")`.
- `containsMarker` in `internal/kbzdoctor/doctor.go` reimplements `bytes.Contains` using a
  scanner chain that unnecessarily converts `[]byte` → `string` → `io.Reader`.

**Pattern 4 — internal atomic-write duplication.**
`internal/context/refresh.go` contains a private `atomicWriteFile` function that is a
functionally equivalent but structurally divergent copy of `internal/fsutil.WriteFileAtomic`.
The copy omits the `os.Chmod` step. This creates an inconsistency: files written via `refresh`
inherit the process umask rather than the explicit `0644` mode used everywhere else in the
codebase.

**Impact of inaction.** The gap between the declared Go version and actual stdlib usage will
widen with each release. The `sort` API is not deprecated, but the `slices` API is the idiom
expected by Go engineers on 1.21+: new contributors will write `slices` and code reviewers
will flag `sort.Slice` as a style inconsistency. The unstructured log output is a practical
problem today: the MCP server emits diagnostics that cannot be filtered by level or parsed
by tooling.

---

## Design

The work is organised into four independent workstreams. They touch non-overlapping file sets
and can be implemented in parallel or in any order. Each workstream is bounded to a pure
mechanical replacement with no semantic change to behaviour.

### Workstream A — `sort` → `slices`

**Scope:** All 35 non-test files currently importing `"sort"`.

**Replacement mapping:**

| Old call | New call | Notes |
|----------|----------|-------|
| `sort.Strings(s)` | `slices.Sort(s)` | Direct drop-in |
| `sort.Ints(s)` | `slices.Sort(s)` | Direct drop-in |
| `sort.Float64s(s)` | `slices.Sort(s)` | Direct drop-in |
| `sort.Slice(s, func(i, j int) bool { … })` | `slices.SortFunc(s, func(a, b T) int { … })` | Comparator signature changes |
| `sort.SliceStable(s, func(i, j int) bool { … })` | `slices.SortStableFunc(s, func(a, b T) int { … })` | Comparator signature changes |

The comparator signature change is the only non-trivial part. `slices.SortFunc` requires a
function returning `int` (negative/zero/positive), not `bool`. The `"cmp"` package (Go 1.21)
provides `cmp.Compare(a, b)` which returns the correct `int` contract for any ordered type.
Multi-field comparators follow the same pattern with early-return on non-zero:

```
// Old
sort.Slice(items, func(i, j int) bool {
    if items[i].Priority != items[j].Priority {
        return items[i].Priority > items[j].Priority
    }
    return items[i].CreatedAt.Before(items[j].CreatedAt)
})

// New
slices.SortFunc(items, func(a, b Item) int {
    if n := cmp.Compare(b.Priority, a.Priority); n != 0 {
        return n
    }
    return a.CreatedAt.Compare(b.CreatedAt)
})
```

**Import changes per file:** remove `"sort"`, add `"slices"`. Add `"cmp"` only in files that
use `sort.Slice`/`sort.SliceStable` with value comparisons (not needed for the
`sort.Strings`/`sort.Float64s` call-sites which become `slices.Sort`).

**Failure modes:** None. `slices.SortFunc` and `sort.Slice` both use an introsort-based
algorithm. Stability semantics are preserved by using `slices.SortStableFunc` wherever
`sort.SliceStable` was used.

**Verification:** `go build ./...` must pass after each file. The full test suite must pass
after all files in the workstream are complete.

---

### Workstream B — `"log"` → `"log/slog"`

**Scope:** 16 non-test files currently importing `"log"`. Two entry points.

**Handler configuration.** The existing `log.Printf` calls emit plain-text lines to stderr.
The replacement uses `slog`'s default text handler, which produces structured key=value output
to stderr at the same destination. No custom handler abstraction is introduced. The global
`slog` logger is configured at the two binary entry points, not buried in library packages:

- `cmd/kbz/main.go` — CLI entry point: configure before `flag.Parse()`.
- `internal/mcp/server.go` — MCP server entry point: configure in server startup, before any
  tool is registered.

Configuration at each entry point:

```
// Production default — text output to stderr
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
})))
```

Library packages (`internal/mcp/`, `internal/service/`, etc.) call `slog.Info`, `slog.Warn`,
`slog.Error` on the package-level default logger — no logger is threaded through function
arguments.

**Call-site translation rules.**

The existing ad-hoc format-string structure maps cleanly to slog attributes:

| Old pattern | New call |
|-------------|----------|
| `log.Printf("[comp] WARNING: msg: %v", err)` | `slog.Warn("msg", "component", "comp", "err", err)` |
| `log.Printf("[comp] ERROR: msg: %v", err)` | `slog.Error("msg", "component", "comp", "err", err)` |
| `log.Printf("[comp] msg: %v", val)` | `slog.Info("msg", "component", "comp", "detail", val)` |
| `log.Printf("WARNING: msg: %v", err)` | `slog.Warn("msg", "err", err)` |

The `[component]` tag prefix becomes a `"component"` key-value attribute. The `WARNING:`/
`ERROR:` prefixes select the log level. Unqualified messages default to `Info`. Structured
values (counts, durations) become typed attributes rather than being formatted inline in the
message string.

**Import changes per file:** replace `"log"` with `"log/slog"`. The two entry-point files
additionally add `"os"` if not already present.

**Failure modes:** `slog.Warn` and `slog.Error` do not terminate the process (unlike
`log.Fatal`). There are no `log.Fatal` or `log.Panic` calls in the affected files — all
74 call-sites use `log.Printf`/`log.Println`/`log.Print` only, so this is safe.

**Verification:** `go build ./...` must pass. The MCP server and `kbz` CLI must start
successfully and emit structured output to stderr. The full test suite must pass.

---

### Workstream C — Point fixes (Findings 3, 4, 5)

**Scope:** Three files, each a one-line replacement.

| File | Old | New |
|------|-----|-----|
| `internal/docint/concepts.go` | `stringSliceContains(slice, s)` | `slices.Contains(slice, s)` + delete the 7-line helper |
| `internal/context/surfacer.go` | `trimTrailingSlash(s)` | `strings.TrimSuffix(s, "/")` + delete the 6-line helper |
| `internal/kbzdoctor/doctor.go` | `containsMarker(data, marker)` | `bytes.Contains(data, []byte(marker))` + delete the 12-line helper; remove `bufio` import |

`internal/docint/concepts.go` will also need `"slices"` added to its import block. The
`"strings"` import in `internal/context/surfacer.go` is already present.

**Failure modes:** None. `slices.Contains` and `bytes.Contains` have the same semantics as
their hand-rolled counterparts for the inputs in question.

**Verification:** `go build ./...` must pass. Existing tests in `internal/docint/` and
`internal/kbzdoctor/` must pass without modification.

---

### Workstream D — Internal atomic-write consolidation

**Scope:** `internal/context/refresh.go` only.

`refresh.go` defines a private `atomicWriteFile(path string, data []byte) error` that
duplicates `fsutil.WriteFileAtomic(path string, data []byte, perm os.FileMode) error`. The
difference is that the local copy omits `os.Chmod`, relying on the process umask for file
permissions.

The fix is to delete the local copy and replace its two call-sites with
`fsutil.WriteFileAtomic(path, data, 0o644)`, matching the explicit mode used throughout the
rest of the codebase. The `fsutil` package is already imported in most sibling packages; it
will be added to `refresh.go`'s import block.

**Failure modes:** Files previously written by `atomicWriteFile` will now receive an explicit
`0o644` chmod. On systems where the process umask was already producing `0o644`, there is no
observable change. On systems with a restrictive umask (e.g. `0o077`), files will become
world-readable where they were not before — which is the correct behaviour for role and skill
YAML files that are committed to the repository.

**Verification:** `go build ./...` must pass. Existing tests in `internal/context/` must pass.

---

### What is explicitly out of scope

The following boundary cases identified in the audit are **not** addressed in this design:

- `deduplicateStrings` (`internal/service/decompose.go`) — no stdlib order-preserving dedup
  equivalent; justified to leave as-is.
- `clamp` (`internal/knowledge/confidence.go`) — domain-specific named helper; the readability
  argument for keeping it outweighs the cosmetic benefit of inlining.
- `parseSemver` / `parseSemverParts` — no stdlib semver package; justified.
- `GenerateTSID13` / `NormalizeTSID` — custom ID format baked into persistent state; a
  data-model migration is out of scope.
- `buildinfo` vars — the `Version` field has no runtime/debug equivalent; `GitSHA`/`Dirty`
  could migrate but the benefit is marginal and it is deferred.

---

## Alternatives Considered

### Alternative A — Opportunistic migration (migrate only when files are otherwise touched)

Migrate `sort` → `slices` and `log` → `log/slog` gradually, one file at a time, only when a
file is already being modified for another reason.

**Easier:** Zero dedicated effort; no risk of merge conflicts on unrelated branches.

**Harder:** The codebase remains inconsistent for an indefinite period. New code has no clear
canonical example to follow. Code review discussions about which API to use recur on every PR.
The `atomicWriteFile` duplication continues to diverge silently.

**Rejected** because the audit has already done the hard work of enumerating all call-sites.
The mechanical nature of the change makes a focused batch the lowest-total-effort path.

---

### Alternative B — Do nothing (status quo)

Accept the existing code as correct and close P67 without changes.

**Easier:** No engineering time spent; no regression risk.

**Harder:** The gap between declared Go version and actual stdlib usage widens. The MCP
server's unstructured logs cannot be filtered or parsed. Future contributors write `slices`
and create inconsistency. The `atomicWriteFile` permission inconsistency remains a latent bug
on systems with non-standard umasks.

**Rejected** because the MCP server diagnostic value of structured logging is concrete and
immediate, and the `sort` → `slices` migration eliminates an entire class of index-arithmetic
comparator bugs at zero cost.

---

### Alternative C — Single omnibus PR across all workstreams

Implement all four workstreams in one branch, one PR, one review.

**Easier:** One PR to review and merge; no coordination overhead between workstreams.

**Harder:** A 50-file diff is harder to review meaningfully. A test regression in any one file
blocks the entire migration. Parallel implementation is not possible if workstreams share a
branch.

**Rejected** in favour of the four-workstream design, which allows each workstream to be
reviewed, merged, and verified independently. Workstreams A and C can be parallelised;
Workstream B (slog) should follow A to avoid a mixed log/slog state during review.

---

### Alternative D — Thread `*slog.Logger` through function arguments (for Workstream B)

Rather than using the package-level default logger, pass a `*slog.Logger` as a parameter
to every function that currently calls `log.Printf`.

**Easier:** Testable in isolation; each call-site can use a different handler in tests.

**Harder:** Requires changing function signatures across 16 files and all their callers.
This is a much larger diff than the log-call replacement alone, and the benefit is marginal
for diagnostic logging (as opposed to application logging where per-request context matters).

**Rejected.** The existing code uses the default global `log` logger without any injection;
the natural equivalent is `slog`'s global default. Per-function injection can be added later
if operational need arises.

---

## Decisions

**Decision:** Use `slices.SortFunc` with `cmp.Compare` for all multi-field comparators.
**Context:** `slices.SortFunc` requires an `int`-returning comparator; existing code uses
`sort.Slice` with a `bool`-returning comparator. `cmp.Compare` (Go 1.21) returns the correct
`int` contract for any ordered type and is the idiomatic bridge.
**Rationale:** `cmp.Compare` is already in the standard library and handles the comparison
contract uniformly. Rolling a per-file helper (e.g. `func less(a, b int) int`) would
re-introduce the NIH pattern we are trying to eliminate.
**Consequences:** Every file in Workstream A that uses `sort.Slice` will need both `"slices"`
and `"cmp"` in its import block. Files that only use `sort.Strings`/`sort.Float64s` need only
`"slices"`.

---

**Decision:** Use `slog`'s default text handler configured at binary entry points; no custom
handler abstraction.
**Context:** The codebase has two binary entry points (`cmd/kbz/main.go`,
`internal/mcp/server.go`) and no existing logging infrastructure. Introducing a custom handler
type would add abstraction with no current consumer.
**Rationale:** The stdlib text handler produces output that is human-readable at a terminal
and parseable by common log tools. Configuration at entry points (not library packages) is the
stdlib-recommended pattern. Adding a custom abstraction now would be premature — the need for
a JSON handler or log-level flag can be addressed when an operational requirement surfaces.
**Consequences:** All library packages call the package-level `slog.*` functions, which
delegate to whatever handler the entry point has configured. Tests that exercise these packages
will produce `slog` output on stderr unless the test configures the default handler to
`io.Discard` — this is a one-line setup that should be added to affected package test helpers.

---

**Decision:** Fix the `atomicWriteFile` permission inconsistency by adopting `0o644` (Workstream D).
**Context:** The local copy in `refresh.go` inherited the process umask; all other atomic
writes in the project use an explicit `0o644`. The files written by `refresh.go` (role and
skill YAML files with updated `last_verified` timestamps) are committed to the repository.
**Rationale:** Committed files should have reproducible permissions regardless of the
developer's umask. `0o644` is the conventional permission for text files committed to a
repository.
**Consequences:** On systems with a restrictive umask, role and skill YAML files written by
`RefreshRoleLastVerified` and `RefreshSkillLastVerified` will become group- and world-readable
where they were not before. This is the correct outcome.

---

**Decision:** Implement as four independent workstreams, not one omnibus change.
**Context:** The four workstreams touch non-overlapping file sets. Workstream A (35 files) and
Workstream C (3 files) are purely mechanical; Workstream B (16 files) requires an
architectural choice at entry points; Workstream D (1 file) is a tiny cleanup.
**Rationale:** Independent workstreams can be reviewed and merged in isolation. A regression
in Workstream A does not block Workstream C. Parallelising A and C reduces wall-clock time.
Workstream B benefits from reviewing A first to establish the `"slices"` + `"cmp"` precedent
before reviewers see `"log/slog"`.
**Consequences:** Four PRs instead of one. Recommended merge order: C → D → A → B. Workstreams
C and D are trivially small and establish a clean baseline; A is the mechanical bulk; B
requires the entry-point handler decision to be reviewed last, with the widest context.

---

## Dependencies

**No new external dependencies are introduced.** All replacements use packages that are part
of the Go standard library and are already available at the project's declared toolchain
version of `go 1.25.0`:

| Package | Type | Available since | Already in go.mod? |
|---------|------|----------------|--------------------|
| `"slices"` | stdlib | Go 1.21 | Not needed — stdlib |
| `"cmp"` | stdlib | Go 1.21 | Not needed — stdlib |
| `"log/slog"` | stdlib | Go 1.21 | Not needed — stdlib |
| `"bytes"` | stdlib | Go 1 | Not needed — stdlib |
| `"strings"` | stdlib | Go 1 | Not needed — stdlib |

`go.mod` and `go.sum` require no changes. The `internal/fsutil` package used in Workstream D
is already a first-party package within this module.
