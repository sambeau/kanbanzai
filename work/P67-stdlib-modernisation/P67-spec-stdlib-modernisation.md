# Specification: Go Standard Library Modernisation

| Field  | Value                                                              |
|--------|--------------------------------------------------------------------|
| Date   | 2026-05-12                                                         |
| Status | approved |
| Author | Claude (Sonnet 4.6)                                                |
| Plan   | P67-stdlib-modernisation                                           |
| Design | `work/P67-stdlib-modernisation/P67-design-stdlib-modernisation.md` |

---

## Problem Statement

This specification implements the design described in
`work/P67-stdlib-modernisation/P67-design-stdlib-modernisation.md`
(`P67-stdlib-modernisation/design-p67-design-stdlib-modernisation`).

The codebase declares `go 1.25.0` but makes no use of the `slices`, `cmp`, or `log/slog`
packages introduced in Go 1.21. All sorting is done through the pre-generics `"sort"` package
(35 non-test files, 52 call-sites); all diagnostic output uses the unstructured `"log"` package
(16 non-test files, 74 call-sites); and three private helper functions in separate packages each
reduce to a single stdlib call they duplicate. A fourth issue — a private `atomicWriteFile`
function in `internal/context/refresh.go` — duplicates `internal/fsutil.WriteFileAtomic` while
omitting the `os.Chmod` step, causing files written by that package to inherit the process umask
rather than the canonical `0o644` mode.

This specification covers the four workstreams defined in the design:

- **Workstream A:** Replace `"sort"` with `"slices"` (and `"cmp"`) across 35 non-test files.
- **Workstream B:** Replace `"log"` with `"log/slog"`, configured at the two binary entry
  points (`cmd/kbz/main.go` and `internal/mcp/server.go`).
- **Workstream C:** Delete `stringSliceContains`, `trimTrailingSlash`, and `containsMarker`
  and replace each with the stdlib call it duplicates.
- **Workstream D:** Delete the private `atomicWriteFile` in `internal/context/refresh.go` and
  replace it with `fsutil.WriteFileAtomic`.

**Out of scope:** test files (`*_test.go`); the boundary cases listed in the design's
"What is explicitly out of scope" section (`deduplicateStrings`, `clamp`, `parseSemver`,
`GenerateTSID13`, `buildinfo`); any change to public API surfaces, data formats, or file
schemas; introduction of new external dependencies.

---

## Requirements

### Functional Requirements

#### Workstream A — `sort` → `slices`

- **REQ-001:** Every call-site using `sort.Strings`, `sort.Ints`, or `sort.Float64s` in the
  35 non-test files listed in the design MUST be replaced with `slices.Sort`.

- **REQ-002:** Every call-site using `sort.Slice` in those files MUST be replaced with
  `slices.SortFunc` with a comparator returning `int`.

- **REQ-003:** Every call-site using `sort.SliceStable` in those files MUST be replaced with
  `slices.SortStableFunc` with a comparator returning `int`.

- **REQ-004:** Every file modified by REQ-001 through REQ-003 MUST remove its `"sort"` import
  and add a `"slices"` import.

- **REQ-005:** Every file modified by REQ-002 or REQ-003 that uses `cmp.Compare` to satisfy
  the `int`-return comparator contract MUST add a `"cmp"` import.

#### Workstream B — `"log"` → `"log/slog"`

- **REQ-006:** Every call-site using `log.Printf`, `log.Println`, or `log.Print` in the 16
  non-test files listed in the design MUST be replaced with the appropriate `slog` call:
  `slog.Warn` for messages containing a `WARNING:` prefix, `slog.Error` for messages
  containing an `ERROR:` prefix, and `slog.Info` for all others.

- **REQ-007:** Every `[component]` tag prefix present in an existing `log.Printf` format
  string MUST become a `"component"` key-value attribute on the replacement `slog` call.
  Inline structured values (error objects, counts, durations) MUST become typed key-value
  attributes rather than being formatted into the message string.

- **REQ-008:** Every file modified by REQ-006 MUST remove its `"log"` import and add a
  `"log/slog"` import.

- **REQ-009:** `cmd/kbz/main.go` MUST call `slog.SetDefault` with a
  `slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})` before any
  other initialisation or flag parsing.

- **REQ-010:** `internal/mcp/server.go` MUST call `slog.SetDefault` with the same handler
  configuration as REQ-009 during server startup, before any MCP tool is registered.

#### Workstream C — Point fixes

- **REQ-011:** The `stringSliceContains` function in `internal/docint/concepts.go` MUST be
  deleted. Its two call-sites MUST be replaced with `slices.Contains`. The `"slices"` import
  MUST be added to that file's import block.

- **REQ-012:** The `trimTrailingSlash` function in `internal/context/surfacer.go` MUST be
  deleted. Its call-sites MUST be replaced with `strings.TrimSuffix(s, "/")`.

- **REQ-013:** The `containsMarker` function in `internal/kbzdoctor/doctor.go` MUST be
  deleted. Its call-sites MUST be replaced with `bytes.Contains(data, []byte(marker))`.
  The `"bufio"` import MUST be removed from that file if it is no longer referenced.

#### Workstream D — Internal atomic-write consolidation

- **REQ-014:** The private `atomicWriteFile` function in `internal/context/refresh.go` MUST
  be deleted.

- **REQ-015:** The two former call-sites of `atomicWriteFile` in `internal/context/refresh.go`
  MUST be replaced with `fsutil.WriteFileAtomic(path, data, 0o644)`.

- **REQ-016:** The `internal/fsutil` package MUST be added to `internal/context/refresh.go`'s
  import block using its full module path.

### Non-Functional Requirements

- **REQ-NF-001:** After the changes in each workstream are applied, `go build ./...` MUST exit
  with code 0 and produce no errors or warnings.

- **REQ-NF-002:** After the changes in each workstream are applied, `go test ./...` MUST exit
  with code 0, with all tests passing and no new test failures introduced relative to the
  baseline on the main branch immediately before that workstream's changes.

- **REQ-NF-003:** After all four workstreams are merged, `go.mod` and `go.sum` MUST be
  unchanged from the baseline. No new external dependencies may be introduced.

- **REQ-NF-004:** Each workstream MUST be contained in a separate PR. A workstream PR MUST
  NOT modify files outside the file set defined for that workstream in the design document.

---

## Constraints

- **No test-file changes.** Files matching `*_test.go` are out of scope for all four
  workstreams. Test files that call `log.Printf` or use `sort.*` are not covered by this
  specification and must not be modified.

- **No API changes.** No exported function signature, type, or package-level variable may
  change as a result of this work.

- **No data-format changes.** File schemas, YAML field names, and on-disk state formats must
  be unchanged.

- **No new external dependencies.** `go.mod` and `go.sum` must remain byte-for-byte identical
  after all workstreams are merged (REQ-NF-003).

- **Stability semantics must be preserved.** Every former `sort.SliceStable` call-site must
  use `slices.SortStableFunc`, not `slices.SortFunc`. Substituting a stable sort with an
  unstable one is a behavioural change and is not permitted.

- **Permission change is intentional (Workstream D).** Files written by
  `internal/context/refresh.go` will move from umask-derived permissions to explicit `0o644`.
  This is a documented and intended behaviour change (see design Decisions section).

- **This specification does NOT cover:**
  - `deduplicateStrings` (`internal/service/decompose.go`) — no order-preserving dedup in stdlib.
  - `clamp` (`internal/knowledge/confidence.go`) — justified named helper.
  - `parseSemver` / `parseSemverParts` — no stdlib semver package.
  - `GenerateTSID13` / `NormalizeTSID` — custom ID format baked into persistent state.
  - `buildinfo` vars — ldflags injection has no direct stdlib equivalent for `Version`.

---

## Acceptance Criteria

**Workstream A:**

- **AC-001 (REQ-001, REQ-002, REQ-003):** Given the 35 files listed in Workstream A of the
  design, when `grep -rn '"sort"'` is run against those files after the workstream is
  complete, then no matches are produced.

- **AC-002 (REQ-004):** Given the same 35 files, when `grep -rn '"slices"'` is run after the
  workstream is complete, then every file that previously imported `"sort"` now imports
  `"slices"`.

- **AC-003 (REQ-002, REQ-005):** Given any file that previously used `sort.Slice` or
  `sort.SliceStable`, when the file is inspected, then every comparator function returns
  `int`, and `"cmp"` is present in the import block if `cmp.Compare` is used.

- **AC-004 (REQ-003):** Given any file that previously used `sort.SliceStable`, when the file
  is inspected, then the replacement call is `slices.SortStableFunc` (not `slices.SortFunc`).

**Workstream B:**

- **AC-005 (REQ-006, REQ-008):** Given the 16 files listed in Workstream B of the design,
  when `grep -rn '"log"'` is run against those files after the workstream is complete
  (excluding `"log/slog"` matches), then no matches are produced.

- **AC-006 (REQ-006):** Given any migrated call-site that previously used a `WARNING:`
  prefix, when the source is inspected, then the replacement is `slog.Warn`. Given any
  migrated call-site with an `ERROR:` prefix, the replacement is `slog.Error`. All others
  use `slog.Info`.

- **AC-007 (REQ-007):** Given any migrated call-site that previously used a `[component]`
  tag prefix, when the source is inspected, then a `"component"` key-value attribute carrying
  the component name is present on the `slog` call, and the component name does not appear in
  the message string.

- **AC-008 (REQ-009):** Given `cmd/kbz/main.go`, when the source is inspected, then
  `slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))`
  (or equivalent) appears before any other non-import code in the main function.

- **AC-009 (REQ-010):** Given `internal/mcp/server.go`, when the source is inspected, then
  `slog.SetDefault` with an equivalent handler is called before any `mcp.NewServer` or tool
  registration call.

**Workstream C:**

- **AC-010 (REQ-011):** Given `internal/docint/concepts.go`, when the source is inspected,
  then the `stringSliceContains` function is absent, `slices.Contains` is used at both former
  call-sites, and `"slices"` is present in the import block.

- **AC-011 (REQ-012):** Given `internal/context/surfacer.go`, when the source is inspected,
  then the `trimTrailingSlash` function is absent and `strings.TrimSuffix(s, "/")` is used at
  both former call-sites.

- **AC-012 (REQ-013):** Given `internal/kbzdoctor/doctor.go`, when the source is inspected,
  then the `containsMarker` function is absent, `bytes.Contains(data, []byte(marker))` is used
  at both former call-sites, and `"bufio"` is absent from the import block.

**Workstream D:**

- **AC-013 (REQ-014, REQ-015, REQ-016):** Given `internal/context/refresh.go`, when the
  source is inspected, then the `atomicWriteFile` function is absent, both former call-sites
  use `fsutil.WriteFileAtomic(path, data, 0o644)`, and the `fsutil` package is imported.

**Non-Functional:**

- **AC-014 (REQ-NF-001):** Given each workstream's changes applied in isolation to the main
  branch, when `go build ./...` is run, then the exit code is 0 and stderr contains no error
  lines.

- **AC-015 (REQ-NF-002):** Given each workstream's changes applied in isolation to the main
  branch, when `go test ./...` is run, then the exit code is 0 and no test that previously
  passed now fails.

- **AC-016 (REQ-NF-003):** Given all four workstreams merged, when `git diff HEAD go.mod
  go.sum` is run, then the output is empty.

- **AC-017 (REQ-NF-004):** Given each workstream's PR diff, when the list of changed files is
  inspected, then every changed file belongs to that workstream's declared file set and no
  file outside that set is modified.

---

## Verification Plan

| Criterion | Method      | Description                                                                                  |
|-----------|-------------|----------------------------------------------------------------------------------------------|
| AC-001    | Inspection  | Run `grep -rn '"sort"'` against the 35 Workstream A files; confirm zero matches.            |
| AC-002    | Inspection  | Run `grep -rn '"slices"'` against the 35 files; confirm every file shows a match.           |
| AC-003    | Inspection  | For each former `sort.Slice`/`sort.SliceStable` file, read the comparator return type.      |
| AC-004    | Inspection  | For each former `sort.SliceStable` file, confirm `slices.SortStableFunc` is used.           |
| AC-005    | Inspection  | Run `grep -rn '"log"'` (excluding `log/slog`) against the 16 Workstream B files; zero hits. |
| AC-006    | Inspection  | For each migrated call-site, verify the slog level matches the former prefix convention.     |
| AC-007    | Inspection  | For each migrated call-site with a former `[tag]` prefix, confirm `"component"` attribute.  |
| AC-008    | Inspection  | Read `cmd/kbz/main.go`; confirm `slog.SetDefault` is the first statement in `main()`.       |
| AC-009    | Inspection  | Read `internal/mcp/server.go`; confirm `slog.SetDefault` precedes tool registration.        |
| AC-010    | Inspection  | Read `internal/docint/concepts.go`; confirm function absent, `slices.Contains` present.     |
| AC-011    | Inspection  | Read `internal/context/surfacer.go`; confirm function absent, `strings.TrimSuffix` present. |
| AC-012    | Inspection  | Read `internal/kbzdoctor/doctor.go`; confirm function absent, `bytes.Contains` present, `bufio` absent. |
| AC-013    | Inspection  | Read `internal/context/refresh.go`; confirm function absent, `fsutil.WriteFileAtomic` present. |
| AC-014    | Test        | Run `go build ./...` after each workstream merge; assert exit code 0.                        |
| AC-015    | Test        | Run `go test ./...` after each workstream merge; assert exit code 0, no regressions.         |
| AC-016    | Test        | Run `git diff HEAD go.mod go.sum` after all workstreams merged; assert empty output.         |
| AC-017    | Inspection  | Review each PR's file list; assert no file outside the workstream's declared set is changed. |
