# Review: P7 — Developer Experience

| Field        | Value                                                        |
|--------------|--------------------------------------------------------------|
| Plan         | P7-developer-experience                                      |
| Features     | FEAT-01KMT-40GZSMHB (server-info-tool)                      |
|              | FEAT-01KMT-40KKZZR5 (human-friendly-id-display)             |
|              | FEAT-01KMT-40P0AGS7 (review-naming-and-folder-conventions)  |
| Specs        | `work/spec/server-info-tool.md`                             |
|              | `work/spec/human-friendly-id-display.md`                    |
|              | `work/spec/review-naming-and-folder-conventions.md`         |
| Reviewer     | Claude Sonnet 4.6                                            |
| Date         | 2026-03-28T14:07:28Z                                        |
| Status       | Findings present — remediation required before closing      |

---

## 1. Summary

P7 delivered three improvements from the P6 retrospective:

1. **`server_info` MCP tool** — exposes build metadata, binary path, install record,
   and in-sync status to eliminate the stale-binary problem.
2. **Human-friendly ID display** — surfaces `display_id` (split form) as primary in
   tool responses, always shows slug alongside the ID, adds optional `label` field on
   features and tasks with a list filter.
3. **Review naming and folder conventions** — establishes `work/reviews/` as the
   canonical destination for review artifacts, migrates five existing reports, and
   updates the code review SKILL.

The implementation is substantially correct and all tests pass with the race detector
enabled. Several issues require remediation: two are critical (a gitignore gap that can
cause machine-specific state to be committed, and an incomplete Makefile), one is
significant (AGENTS.md project status is stale), and the rest are minor test coverage
gaps and display-format deviations.

---

## 2. Spec Conformance

### 2.1 FEAT-01KMT-40GZSMHB — `server_info` Tool

| AC | Description | Status |
|----|-------------|--------|
| AC-01–03 | `internal/buildinfo` package with four `var` declarations, correct defaults | ✅ Pass |
| AC-04–07 | `Makefile` with `build` and `install` targets, correct ldflags | ⚠️ **Partial** — see §3.1 |
| AC-08–09 | `internal/install` package, `WriteRecord`/`ReadRecord` signatures | ✅ Pass |
| AC-10 | Canonical YAML field order | ✅ Pass (verified by `TestWriteRecord_FieldOrder`) |
| AC-11 | Atomic write via temp-rename | ✅ Pass (uses `fsutil.WriteFileAtomic`) |
| AC-12 | Written to `<root>/.kbz/last-install.yaml` | ✅ Pass |
| AC-13 | File excluded from git | ❌ **Fail** — see §3.2 |
| AC-14–17 | `kbz install-record write` CLI subcommand | ✅ Pass |
| AC-18–20 | `server_info` tool registered, no args | ✅ Pass |
| AC-19 | Registered unconditionally regardless of `mcp.groups` | ⚠️ **Formal deviation** — see §3.3 |
| AC-21–24 | Response fields, fresh disk read, filesystem error handling | ✅ Pass |
| AC-25 | Total tool count | ⚠️ **Spec count wrong** — see §3.4 |
| AC-26–31 | Post-merge install step, side effects, restart notice | ✅ Pass |
| AC-32–33 | Graceful degradation without ldflags | ✅ Pass |
| AC-34–38 | Tests | ⚠️ **Partial** — see §3.5 |

### 2.2 FEAT-01KMT-40KKZZR5 — Human-Friendly ID Display

| AC | Description | Status |
|----|-------------|--------|
| AC-01–04 | `Label` field on Feature and Task; 24-char limit; empty = absent | ✅ Pass |
| AC-05–08 | YAML field order (`slug → label → status`); no spurious `label:` key | ✅ Pass |
| AC-09–10 | Split and unsplit ID forms accepted on all tool actions | ✅ Pass |
| AC-11–17 | `display_id` as primary in entity, status, next, handoff, finish responses | ⚠️ **Partial** — see §3.6 |
| AC-18–21 | Slug shown alongside display ID; label display format | ⚠️ **Partial** — see §3.7 |
| AC-22–25 | Label create/update/clear/get | ✅ Pass |
| AC-26–27 | Label list filter | ✅ Pass (implemented; see §3.8 for missing test) |
| AC-28–30 | Status dashboard conditional label column via `has_labels` | ✅ Pass |
| AC-31–33 | Backward compatibility | ✅ Pass |

### 2.3 FEAT-01KMT-40P0AGS7 — Review Naming and Folder Conventions

| AC | Description | Status |
|----|-------------|--------|
| AC-01 | `.skills/code-review.md` updated with `work/reviews/` and filename format | ✅ Pass |
| AC-02 | `bootstrap-workflow.md` placement table includes `work/reviews/` row | ❌ **Fail** — see §3.9 |
| AC-03 | `bootstrap-workflow.md` clarifies `work/reports/` excludes review artifacts | ⚠️ **Partial** — see §3.9 |
| AC-04 | `AGENTS.md` repository structure lists `work/reviews/` | ✅ Pass |
| AC-05 | Five review reports migrated from `work/reports/` to `work/reviews/` | ✅ Pass |
| AC-06 | `track-c-batch-operations-review.md` renamed | ✅ Pass |
| AC-07 | Document state records updated to new paths and hashes | ✅ Pass (verified by passing `doc validate`) |
| AC-08–10 | No spurious file renames; no Go code changes | ✅ Pass |

---

## 3. Findings

### 3.1 Makefile `install` target does not write the install record [CRITICAL]

**Spec:** AC-06 — "The `install` target, after building, calls `kbz install-record write`
(or equivalent) to write `.kbz/last-install.yaml`."

The Makefile `install` target still contains:

```kanbanzai/Makefile#L20-24
install: build
	go install -ldflags "$(LDFLAGS)" ./cmd/kanbanzai
	# TODO: run install-record write --by makefile once CLI subcommand exists
```

The CLI subcommand (`kbz install-record write`) was implemented as part of this very
feature. The TODO was never resolved. The install record will never be written when
`make install` is run, defeating the stated goal of this feature: confirming server
currency via a single `server_info` call.

**Remediation:** Replace the TODO comment with the actual command call:

```/dev/null/Makefile.diff#L1-4
install: build
	go install -ldflags "$(LDFLAGS)" ./cmd/kanbanzai
	kbz install-record write --by makefile
```

### 3.2 `.kbz/last-install.yaml` not excluded from git [CRITICAL]

**Spec:** AC-13 — "`.kbz/last-install.yaml` is either already in `.gitignore`
(covered by an existing `.kbz/` exclusion) or an exclusion is explicitly added.
The file must not be committed."

The `.gitignore` has entries for `.kbz/cache/` and `.kbz/local.yaml` but **not** for
`.kbz/last-install.yaml`. The `.kbz/` directory as a whole is not excluded (workflow
state files in `.kbz/state/` are committed), so the install record file has no
protection.

Although `last-install.yaml` does not currently exist on disk (no install has been
run), as soon as `make install` or `kbz install-record write` is invoked, the file
will appear as an untracked file and could be accidentally staged and committed.
Machine-specific state (binary path, local SHA) committed to the shared repository
would be misleading and incorrect.

**Remediation:** Add to `.gitignore`:

```/dev/null/.gitignore.diff#L1-1
.kbz/last-install.yaml
```

### 3.3 `server_info` registered conditionally, not unconditionally [Minor spec deviation]

**Spec:** AC-19 — "`server_info` belongs to the `core` group and is registered
unconditionally — it is present regardless of the `mcp.groups` or `mcp.preset`
configuration."

`server_info` is registered inside `if groups[config.GroupCore]` in `server.go`. In
practice this is harmless because `GroupCore` cannot be disabled (the config enforcer
overrides any attempt to set `core: false`). But the spec explicitly requires
unconditional registration, and the current structure would silently drop `server_info`
if the core-enforcement logic were ever removed.

**Remediation:** Move `mcpServer.AddTools(ServerInfoTool()...)` outside the
`if groups[config.GroupCore]` block, registering it unconditionally before all group
conditionals.

### 3.4 Spec tool count is wrong (spec artefact, not implementation bug)

**Spec:** AC-25 — "The total number of registered MCP tools across all groups is 21."

The test correctly asserts **22** tools. This is because the `retro` tool was added
(in P5 Phase 2) after the server_info spec was written. Before P7, the count was
already 21 (not 20 as the spec assumed). After adding `server_info` the count is 22.

The implementation and the test are both correct. The spec's count was stale when
written. No code change is required, but the spec should be annotated to reflect the
actual outcome.

### 3.5 `handleServerInfo` has no direct unit test [Minor]

**Spec:** AC-37 — "The `server_info` handler has a unit test confirming `in_sync`
is: `true` when SHAs match, `false` when they differ, `null` when either is unknown."

The `TestDeriveInSync` and `TestDeriveGitSHAShort` tests cover the helper functions
thoroughly. However, there is no test that exercises `handleServerInfo` end-to-end —
confirming the full response shape (all nine top-level fields), the `install_record`
null-vs-populated cases, and the error path when `ReadRecord` returns a non-ErrNotExist
error.

**Remediation:** Add a `TestHandleServerInfo` test in
`internal/mcp/server_info_tool_test.go` that sets up a temp directory, optionally
writes an install record, and verifies the complete response shape.

### 3.6 `finish` and `handoff` do not include `entity_ref` or combined slug display [Minor]

**Spec:** AC-14 (handoff uses `display_id`), AC-15 (finish uses `display_id`),
AC-19 ("wherever a task ID appears in a tool response, the entity's slug is shown
in parentheses immediately after the display ID").

Both `finish_tool.go` and `handoff_tool.go` include `display_id` correctly. However
neither includes an `entity_ref` field (the `display_id (slug)` combined form) or
exposes the slug at the top level of the completion/handoff response. Consumers must
know to look up the slug separately, which is exactly the friction these ACs were
designed to avoid.

`next_tool.go` does better — it includes both `display_id` and `slug` as peer fields —
but still omits `entity_ref`.

**Remediation:** Add a top-level `entity_ref` field (formatted as
`id.FormatEntityRef(displayID, slug, label)`) to the `finish` completion summary and
the `handoff` response header, alongside `display_id`.

### 3.7 Display Convention Reference — verbose `entity get` format not implemented [Minor]

**Spec:** Display Convention Reference table — "Verbose (`entity get`) with label:
`FEAT-01KMR-X1SEQV49 · label: G · policy-and-documentation-updates`"

The implementation uses `entity_ref` (`FEAT-01KMR-X1SEQV49 (G slug)`) uniformly in
all response contexts, including `entity get`. The `·`-separated verbose format
specified in the convention table is not implemented. The current format is functional
and includes all information, but it differs from the spec.

Additionally, the spec example for label display shows an abbreviated slug
(`policy-docs`), while the implementation always passes the full slug to
`FormatEntityRef`. The abbreviated-slug behaviour is not implemented.

This is a low-priority deviation since the response already contains `label`, `slug`,
and `display_id` as separate fields for consumers that need them. The `entity_ref`
string is a convenience display value and its exact format is unlikely to break
callers.

**Remediation (optional):** Either implement the verbose format for `entity get`
responses, or update the spec's Display Convention Reference table to reflect the
uniform `entity_ref` format that was actually adopted.

### 3.8 Missing test: `ListEntitiesFiltered` with label filter [Minor]

The label filter is correctly wired in `internal/service/queries.go` (lines 173–176),
but `internal/service/queries_test.go` has no `TestListEntitiesFiltered_ByLabel` test.
The storage round-trip label tests in `internal/storage/entity_store_test.go` do not
cover the query-layer filter path.

**Remediation:** Add `TestListEntitiesFiltered_ByLabel` to
`internal/service/queries_test.go`, mirroring the pattern of the existing
`_ByStatus` and `_ByTags` tests.

### 3.9 `bootstrap-workflow.md` document placement table incomplete [Minor]

**Spec:** AC-02 — "`work/bootstrap/bootstrap-workflow.md` document placement table
includes a row for `work/reviews/`..." AC-03 — "...clarifies that `work/reports/`
is for general-purpose reports and does not include review lifecycle artifacts."

The document placement table (line ~185 in the file) has been updated in prose
(lines 141–142 correctly distinguish `work/reviews/` from `work/reports/`), but the
**table itself** still shows:

```work/bootstrap/bootstrap-workflow.md#L185-185
| `work/reports/` | `report` | Review reports, audit reports, post-implementation reviews |
```

The table is missing a `work/reviews/` row and the `work/reports/` row still
incorrectly lists "Review reports" as its content.

**Remediation:** Update the table to add a `work/reviews/` row and remove "Review
reports" from the `work/reports/` description:

| Location | Type | Notes |
|----------|------|-------|
| `work/reviews/` | `report` | Review reports produced by the formal `reviewing` lifecycle gate; one file per reviewed feature or bug |
| `work/reports/` | `report` | General-purpose reports: retrospectives, friction analyses, audit findings, research outputs, progress reports |

### 3.10 `p8-decompose-reliability-review.md` violates naming convention [Minor / Post-P7 regression]

`work/reviews/p8-decompose-reliability-review.md` (created during P8) does not follow
the `review-{id}-{slug}.md` convention established by P7. The expected name would be
`review-P8-decompose-reliability.md` (or a feature-level name if the review was
against a feature entity).

This is a P8 artefact, not a P7 implementation bug, but it represents the first
regression against the convention this feature established.

**Remediation:** Rename `p8-decompose-reliability-review.md` to
`review-P8-decompose-reliability.md` and update the document state record path
accordingly.

### 3.11 AGENTS.md Project Status is stale [Significant]

`AGENTS.md` contains two inaccuracies introduced by P7:

1. **Tool count:** The Kanbanzai 2.0 paragraph still reads "The 2.0 MCP server
   registers exactly 20 tools across 7 groups." The actual count is now 22.

2. **P7 and P8 not mentioned:** The Project Status section ends with P6 and
   Kanbanzai 2.0. P7 (complete) and P8 (complete) are not mentioned. Future agents
   reading AGENTS.md will not know these plans were executed or what they delivered.

AGENTS.md is the first document agents read before any task. Stale information here
degrades the quality of every agent interaction.

**Remediation:** Add a paragraph for P7 and P8 to the Project Status section, and
update the tool count in the Kanbanzai 2.0 paragraph from 20 to 22.

### 3.12 Dead code in `handleServerInfo` [Trivial]

`internal/mcp/server_info_tool.go` lines 67–73:

```kanbanzai/internal/mcp/server_info_tool.go#L65-74
rec, err := install.ReadRecord(".")
if err != nil {
    // File-not-found returns nil, nil from ReadRecord.
    // Any other filesystem error is a real problem.
    if !errors.Is(err, os.ErrNotExist) {
        return nil, err
    }
    // Shouldn't happen (ReadRecord handles ErrNotExist), but be safe.
    rec = nil
}
```

`ReadRecord` already converts `ErrNotExist` to `nil, nil` internally, so the
`errors.Is(err, os.ErrNotExist)` branch can never be true. The outer `if err != nil`
is the only reachable path when `err != nil`, and it correctly returns the error.
The inner check and the `rec = nil` assignment are dead.

The comment `// Shouldn't happen` is correct, which makes the dead-code guard
redundant. Simplify to:

```/dev/null/server_info_tool.go.suggested#L1-4
rec, err := install.ReadRecord(".")
if err != nil {
    return nil, err
}
```

---

## 4. Code Quality Assessment

### 4.1 `internal/buildinfo`

Clean. No imports, pure var declarations with correct defaults. The test covers all
four variables. Exactly as specified.

### 4.2 `internal/install`

Well-structured. Uses `fsutil.WriteFileAtomic` correctly. `ReadRecord` has the right
`nil, nil` contract for missing files. Field order in the struct matches the canonical
YAML order required by the spec.

One minor observation: `gopkg.in/yaml.v3` marshals `time.Time` with sub-second
precision (e.g. `2026-03-28T14:06:44.141572Z`) rather than the bare RFC 3339 format
most of the codebase uses. This is valid and round-trips correctly, but deviates from
the project's pattern of explicit RFC 3339 strings with second precision. The
`server_info` response normalises this by formatting `InstalledAt` with
`time.Format("2006-01-02T15:04:05Z07:00")`, so external consumers always see
second-precision RFC 3339. No change needed, but worth noting.

`TestWriteRecord_FieldOrder` is a good test. `TestWriteRecord_AtomicWrite` is labelled
as testing atomicity but only confirms the file exists and has valid content after the
write — it does not actually test the atomic rename semantics. This is fine since the
atomic behaviour is delegated to `fsutil` (which has its own tests), but the test name
is slightly misleading.

### 4.3 `internal/mcp/server_info_tool.go`

Logic is clear and correct. `deriveGitSHAShort` and `deriveInSync` are well-extracted
helpers with complete test coverage. The response shape matches the spec.

The `os.ErrNotExist` dead code noted in §3.12 is the only issue.

### 4.4 `internal/mcp/post_merge_install.go`

The implementation correctly handles all four outcomes: opt-out, no `main.go`, success,
and failure. The `install_complete` side effect includes `git_sha`, `binary_path`, and
a restart instruction. The `install_failed` side effect preserves the error message.

One concern: the post-merge install step resolves the binary path via `$GOBIN` and
`$GOPATH/bin`, which is the right heuristic. However, if neither environment variable
is set and the user has a non-standard Go home, the path will resolve to
`~/go/bin/kanbanzai` even if the binary landed elsewhere. This is acceptable given the
alternative complexity, and the install record written will reflect the actual path
only if `kbz install-record write` is called later by the Makefile.

### 4.5 `internal/id/display.go`

`NormalizeID`, `FormatFullDisplay`, `FormatEntityRef`, and `StripBreakHyphens` are
all well-implemented with comprehensive tests. The split-form detection regex is
correct.

### 4.6 Storage label integration

The canonical field order placing `label` between `slug` and `status` for features,
and between `slug` and `summary` for tasks, is correctly implemented.

Note that for tasks, the field order is `id → parent_feature → slug → label → summary
→ status`, which puts `label` before `summary`, not immediately before `status`. The
spec says "label appears after slug and before status" which is technically satisfied,
but `summary` appearing between `label` and `status` may be surprising. This is a
reasonable layout choice but reviewers should confirm it was intentional.

---

## 5. Test Coverage Summary

| Package / Test | Coverage | Notes |
|----------------|----------|-------|
| `internal/buildinfo` | Complete | All four default vars tested |
| `internal/install` | Good | Round-trip, missing file, field order, atomic write. AC-37 in-sync logic covered via helpers. |
| `internal/id` | Complete | `NormalizeID`, `FormatEntityRef`, `FormatFullDisplay`, `FormatShortDisplay` all covered |
| `internal/validate` — label | Good | `ValidateLabel`, `TestLabelMaxLength` present |
| `internal/storage` — label | Good | `TestFeatureLabelRoundTrip`, `TestFeatureNoLabelOmitted`, `TestTaskLabelRoundTrip` present |
| `internal/service` — label filter | **Gap** | No `TestListEntitiesFiltered_ByLabel` |
| `internal/mcp` — server_info handler | **Gap** | Only helpers tested; no full handler test |
| `internal/mcp` — label in status | **Gap** | `has_labels` conditional column not tested |
| `internal/mcp` — label CRUD via entity tool | **Gap** | No test for set/change/clear via `entity update` at MCP layer |
| `cmd/kanbanzai` — install-record write | **Gap** | No CLI integration test |
| All tests race detector | ✅ Pass | `go test -race ./...` green |

---

## 6. Documentation Review

### AGENTS.md
- ✅ Repository structure lists `work/reviews/` (line ~91)
- ❌ Project Status paragraph says "exactly 20 tools" — should be 22
- ❌ P7 and P8 completions not mentioned

### `work/bootstrap/bootstrap-workflow.md`
- ✅ Lines 141–142 correctly distinguish `work/reviews/` from `work/reports/`
- ❌ Document placement table missing `work/reviews/` row (AC-02)
- ❌ Document placement table `work/reports/` row still says "Review reports" (AC-03)

### `.skills/code-review.md`
- ✅ Updated to specify `work/reviews/` destination and `review-{id}-{slug}.md` format
- ✅ Includes concrete example filename

### `work/spec/server-info-tool.md` and `work/spec/human-friendly-id-display.md`
- Both still marked `Status: Draft`. These should be updated to `Approved` now that
  implementation is complete and verified.

---

## 7. Verdict

### Must Fix Before Closing P7

| # | Finding | Severity |
|---|---------|----------|
| 3.2 | `.kbz/last-install.yaml` not in `.gitignore` | **Critical** |
| 3.1 | Makefile `install` target does not call `kbz install-record write` | **Critical** |
| 3.11 | AGENTS.md Project Status stale (tool count wrong, P7/P8 not mentioned) | **Significant** |
| 3.9 | `bootstrap-workflow.md` document placement table incomplete | **Minor** |

### Should Fix (Follow-up Acceptable)

| # | Finding | Severity |
|---|---------|----------|
| 3.5 | No direct unit test for `handleServerInfo` | Minor |
| 3.6 | `finish` and `handoff` missing `entity_ref` / slug in combined form | Minor |
| 3.8 | No `TestListEntitiesFiltered_ByLabel` service test | Minor |
| 3.3 | `server_info` registered conditionally rather than unconditionally | Minor |
| 3.10 | `p8-decompose-reliability-review.md` naming regression | Minor |

### Low Priority / Informational

| # | Finding | Severity |
|---|---------|----------|
| 3.4 | Spec tool count was wrong before it was written (artefact) | Info |
| 3.7 | Verbose display format differs from spec's Convention Reference table | Low |
| 3.12 | Dead code in `handleServerInfo` error path | Trivial |