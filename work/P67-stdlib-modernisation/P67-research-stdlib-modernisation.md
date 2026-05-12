# Go Standard Library / Well-Known Library Audit: kanbanzai

**Plan:** P67-stdlib-modernisation  
**Status:** Research complete — ready for design  
**Go version:** 1.25.0 (no version bump required for any finding)

---

## Summary

Searched 576 `.go` files (excluding `.worktrees/`, `vendor/`, `.kbz/`). Found **14 candidates**
across the `internal/` tree: **5 STDLIB**, **0 WELL-KNOWN**, **7 BOUNDARY**, **2 FALSE_POSITIVE**.

The dominant theme is the complete absence of the `slices` package despite `go 1.25.0` in
`go.mod` — the codebase uses `"sort"` in 35 non-test files with no `"slices"` imports anywhere.
A secondary theme is pervasive use of the pre-structured `"log"` package across the MCP and
service layers, with no `"log/slog"` adoption. Three smaller point findings involve hand-rolled
helpers that each reduce to a single stdlib call.

---

## STDLIB Replacements (priority)

### Finding 1 — `sort` → `slices` (35 files, 52 call-sites)

| Field | Detail |
|-------|--------|
| **Files** | 35 non-test files in `internal/` (see list below) |
| **Symbols** | `sort.Slice`, `sort.SliceStable`, `sort.Strings`, `sort.Float64s` |
| **Current** | `"sort"` package with callback-style `func(i, j int) bool` comparators |
| **Replacement** | `slices.SortFunc`, `slices.SortStableFunc`, `slices.Sort` from `"slices"` |
| **Since** | Go 1.21 |
| **Rationale** | Largest single refactoring surface. Every `sort.Strings(s)` collapses to `slices.Sort(s)`; every `sort.Slice(s, less)` becomes `slices.SortFunc(s, cmp)` with a typed comparator. No feature gap. `"slices"` is not imported anywhere in the codebase despite the project targeting Go 1.25. |

**Migration pattern:**

```go
// Before
sort.Strings(names)
sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })

// After
slices.Sort(names)
slices.SortFunc(items, func(a, b Item) int { return cmp.Compare(b.Score, a.Score) })
```

`sort.SliceStable` → `slices.SortStableFunc`; `sort.Float64s` → `slices.Sort`.
The `cmp.Compare` helper (Go 1.21) handles the `int`-return comparator contract.

**Files importing `"sort"` (non-test):**

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

---

### Finding 2 — `"log"` → `"log/slog"` (16 files, 74 call-sites)

| Field | Detail |
|-------|--------|
| **Files** | 16 non-test files (see list below) |
| **Symbols** | `log.Printf`, `log.Println`, `log.Print` |
| **Current** | Unstructured `Printf`-style diagnostics; structure hand-encoded in format strings e.g. `log.Printf("[server] WARNING: cache warm-up failed: %v", err)` |
| **Replacement** | `slog.Info`, `slog.Warn`, `slog.Error`, `slog.Debug` from `"log/slog"` |
| **Since** | Go 1.21 |
| **Rationale** | The codebase manually encodes structure into format strings. `log/slog` provides first-class key-value pairs, log levels, and handlers without changing call-site effort. Not imported anywhere. |

**Files importing `"log"` (non-test):**

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

**Migration pattern (representative):**

```go
// Before
log.Printf("[server] WARNING: cache warm-up failed (continuing): %v", err)
log.Printf("[server] cache warm-up: loaded %d entities in %s", n, time.Since(start))

// After
slog.Warn("cache warm-up failed", "component", "server", "err", err)
slog.Info("cache warm-up complete", "component", "server", "entities", n, "elapsed", time.Since(start))
```

---

### Finding 3 — `stringSliceContains` → `slices.Contains`

| Field | Detail |
|-------|--------|
| **File** | `internal/docint/concepts.go:124` |
| **Symbol** | `stringSliceContains` |
| **Current** | 7-line for-loop linear scan |
| **Replacement** | `slices.Contains[[]string, string]` from `"slices"` |
| **Since** | Go 1.21 |
| **Rationale** | Verbatim reimplementation of `slices.Contains`. Called in 2 places in the same file. |

```go
// Before
func stringSliceContains(slice []string, s string) bool {
    for _, v := range slice {
        if v == s { return true }
    }
    return false
}

// After (inline, no helper needed)
slices.Contains(slice, s)
```

---

### Finding 4 — `trimTrailingSlash` → `strings.TrimSuffix`

| Field | Detail |
|-------|--------|
| **File** | `internal/context/surfacer.go:190` |
| **Symbol** | `trimTrailingSlash` |
| **Current** | 6-line guard function stripping a trailing `/` |
| **Replacement** | `strings.TrimSuffix(s, "/")` |
| **Since** | stdlib (any version) |
| **Rationale** | Wraps a single stdlib call already in the file's import list. The `strings` package is already imported. |

---

### Finding 5 — `containsMarker` → `bytes.Contains`

| Field | Detail |
|-------|--------|
| **File** | `internal/kbzdoctor/doctor.go:279` |
| **Symbol** | `containsMarker` |
| **Current** | Converts `[]byte` → `string` → `strings.NewReader` → `bufio.Scanner`, then iterates lines calling `strings.Contains` |
| **Replacement** | `bytes.Contains(data, []byte(marker))` |
| **Since** | stdlib (any version) |
| **Rationale** | The marker is a single-line HTML comment that cannot span a newline, so `bytes.Contains` is semantically equivalent. Eliminates the Scanner/Reader/string-conversion chain. The `bufio` import in this file exists solely for this function. |

---

## Well-Known Library Recommendations

**None.** All current third-party dependencies (`gopkg.in/yaml.v3`, `golang.org/x/term`,
`github.com/google/uuid`, `modernc.org/sqlite`, `github.com/jackc/pgx/v5`,
`github.com/mark3labs/mcp-go`) are appropriate for their use cases. No custom
re-implementations exist that should instead use a well-known library.

---

## Boundary Cases (justified, adjacent to stdlib)

| # | File | Symbol | Stdlib touchpoint | Why justified |
|---|------|--------|-------------------|---------------|
| 1 | `internal/fsutil/atomic.go:11` | `WriteFileAtomic` | `os.WriteFile` | No stdlib `WriteFileAtomic`. Temp-rename is the standard idiom; stdlib doesn't surface it as a single function. Correctly modelled as a project utility. |
| 2 | `internal/context/refresh.go:92` | `atomicWriteFile` (private) | `fsutil.WriteFileAtomic` (same project) | **Internal duplication**, not a stdlib gap. `refresh.go` has a local copy without `os.Chmod`. Consolidate to `fsutil.WriteFileAtomic` — no stdlib change required. |
| 3 | `internal/buildinfo/buildinfo.go` | `Version`, `GitSHA`, `BuildTime` vars | `runtime/debug.ReadBuildInfo()` | `debug.ReadBuildInfo()` (Go 1.12+) can provide VCS SHA and dirty flag automatically. However, it cannot inject a human-chosen semantic version string — that still requires `-ldflags`. `GitSHA`/`Dirty` could migrate; `Version` cannot. Low priority. |
| 4 | `internal/service/decompose.go:1457` | `deduplicateStrings` | `slices.Compact` (Go 1.21) | `slices.Compact` only removes **adjacent** duplicates. `deduplicateStrings` is order-preserving. No single stdlib call does this. Justified. |
| 5 | `internal/knowledge/confidence.go:6` | `clamp` | Go 1.21 built-in `min`/`max` | Could be inlined as `min(max(v, lo), hi)`. However the function is 3 lines, named, and documents intent. Trade-off is neutral. |
| 6 | `internal/config/config.go:604`, `internal/kbzinit/compare.go:109` | `parseSemver`, `parseSemverParts` | `golang.org/x/mod/semver` | `golang.org/x/mod/semver` enforces the Go module `v` prefix format. The project needs both `v1.0.0` and `1.0.0` forms. No stdlib semver package exists. Justified. |
| 7 | `internal/id/tsid.go` | `GenerateTSID13`, `NormalizeTSID` | `github.com/google/uuid` v7 | TSID format is baked into all persisted state. UUID v7 is a different 36-char format. Replacement would be a breaking data-model change. Justified. |

---

## Appendix A: Search Methodology

1. **File inventory:** `find . -name '*.go' -not -path './.worktrees/*' -not -path './.kbz/*'` — 576 files.
2. **NIH-density zones:** Scanned `internal/fsutil/`, `internal/id/`, `internal/cache/`, `internal/context/`, `internal/hashvalidate/`, `internal/coordination/`, `internal/knowledge/`, `internal/health/`, `internal/kbzdoctor/` for custom `func` signatures.
3. **Sort/slices audit:** `grep -rn '"sort"'` across all non-test files → 35 files; `grep -rn '"slices"'` → 0 hits.
4. **Log/slog audit:** `grep -rn '"log"'` → 16 files; `grep -rn '"log/slog"'` → 0 hits.
5. **String helper scan:** Grepped for function names containing `Contains`, `HasPrefix`, `HasSuffix`, `Join`, `Split`, `Trim`, `Clamp`, `Min`, `Max`, `Dedup`, `Filter`.
6. **Legacy API scan:** Searched for `io/ioutil`, `math/rand` (non-v2), `sort.Search` — all clean.
7. **Dependency cross-check:** Verified `go.mod` direct/indirect imports against findings.

**Scope excluded:** `.worktrees/`, `_testexternal/`, `*_test.go`, `internal/kbzinit/` embedded content files.

---

## Appendix B: Go Version Note

The project declares `go 1.25.0` in `go.mod`. Every finding in the STDLIB table is available
with no version bump:

| Package | Available since | Version gap? |
|---------|----------------|--------------|
| `slices` | Go 1.21 | **None** |
| `log/slog` | Go 1.21 | **None** |
| `slices.Contains` | Go 1.21 | **None** |
| `strings.TrimSuffix` | Go 1 | **None** |
| `bytes.Contains` | Go 1 | **None** |
| Built-in `min`/`max` | Go 1.21 | **None** (boundary only) |

---

## Appendix C: Retrieval Anchors (for design phase)

- **go.mod version:** `go 1.25.0` — no version bump needed for any finding.
- **Test impact:** Findings 1 and 2 touch ~50 non-test files. Tests for affected packages will compile but should be run to confirm no behavioural regression.
- **Transitive deps:** `github.com/spf13/cast v1.7.1` is in `go.sum` as a transitive-only dep; it is not directly used and is unrelated to these findings.
- **Error behaviour differences:** `slices.SortFunc` requires an `int`-returning comparator (not `bool`); all `sort.Slice` callsites must be updated to return `cmp.Compare(a, b)` or an equivalent expression. This is a mechanical but non-trivial change.
- **`slog` handler:** Adopting `log/slog` will require a decision on the default handler (text vs JSON) and whether to configure it at server startup. The existing `log.Printf` calls use no custom logger — the default global logger is in use throughout.
