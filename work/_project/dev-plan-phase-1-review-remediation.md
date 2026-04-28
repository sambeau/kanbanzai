# Phase 1 Review Remediation Plan

- Status: implementation plan
- Purpose: fix all blocking issues and non-blocking follow-ups identified in the Phase 1 remaining work review
- Date: 2025-07-26
- Based on:
  - Phase 1 remaining work review (Feature Implementation Review Profile)
  - `work/plan/phase-1-remaining-work.md`
  - `work/spec/id-system-specification.md`
  - `work/spec/phase-1-specification.md`
  - `work/spec/bootstrap-specification.md`

---

## 1. Findings Summary

The review identified 3 blocking issues and 7 non-blocking follow-ups.

### Blocking

| # | Finding | Spec reference |
|---|---------|----------------|
| B1 | Prefix resolution not implemented | §14.7 |
| B2 | Display formatting not wired into CLI or MCP output | §14.8 |
| B3 | All bootstrap features in `draft` — need at least one `open` | §9.2 |

### Non-blocking

| # | Finding |
|---|---------|
| N1 | Test fixture migration incomplete — 4 packages still use old-format IDs; no shared test constants |
| N2 | CLI test uses `--feature` flag — should be `--parent_feature` to match production code |
| N3 | P1-DEC-016 has 3 resolved follow-ups not struck through |
| N4 | P1-DEC-006 has no TSID13 filename examples |
| N5 | `NormalizeTSID` only uppercases — doesn't do full Crockford normalization (I/L→1, O→0) |
| N6 | No bug entity in bootstrap state — §9.7 asks for Task, Bug, and Decision creation verification |
| N7 | §9.5 (`validate`) semantics unclear — the `validate` command is a pre-persist check, not a project-wide validator |

### Disposition: N7 (§9.5 semantics)

N7 is not a code or documentation fix — it's an open question about the bootstrap specification's intent. The `validate` command validates a candidate entity before persistence. The `health` command validates project-wide state integrity. The bootstrap spec §9.5 says "`kbz validate` passes", but the command requires `--type` and field flags — it isn't a project-wide assertion.

**Recommendation:** accept `health` as satisfying both §9.4 and §9.5 for bootstrap purposes, and note this interpretation in the bootstrap activation decision record. No code change needed. If you disagree, let me know and I'll add a remediation task.

---

## 2. Work Items

### W1: Prefix Resolution (B1)

**Goal:** implement §14.7 — accept ID prefixes in Get, UpdateStatus, and UpdateEntity; resolve to a unique entity or return clear errors.

**Current state:** every lookup path requires the caller to supply exact `(type, id, slug)`. The only directory scan is `EntityService.List`, which globs `*.yaml` and loads every file. `entityExists` does a lighter glob (`{ID}-*.yaml`) but requires an exact full ID.

**Approach:** add a `ResolvePrefix` method to the service layer that scans filenames without loading YAML, then wire it into the existing Get/Update paths. The prefix input goes through `StripBreakHyphens` and case normalization first.

**Changes:**

| File | Change |
|------|--------|
| `internal/service/entities.go` | Add `ResolvePrefix(entityType, prefix string) (id, slug string, err error)`. Globs `{dir}/*.yaml`, parses each filename with `parseRecordIdentity` to extract `(id, slug)`, matches normalized prefix against normalized ID. Returns: unique match → `(id, slug, nil)`; no match → descriptive error; ambiguous → error listing candidates. |
| `internal/service/entities.go` | Update `Get`, `UpdateStatus`, `UpdateEntity`: when `slug` is empty, call `ResolvePrefix` to resolve `(id, slug)` from the prefix. When `slug` is provided, keep current exact-lookup path. |
| `internal/service/entities_test.go` | Test prefix resolution: exact match, unambiguous prefix, ambiguous prefix (candidates listed), no match, case-insensitive, break-hyphen stripping. Test Get/UpdateStatus/UpdateEntity with prefix-only input (no slug). |
| `internal/mcp/entity_tools.go` | Make `slug` optional (remove `mcp.Required()`) on `get_entity`, `update_status`, `update_entity`. When slug is absent, pass empty string to service — resolution happens there. |
| `internal/mcp/server_test.go` | Add tests for MCP tools with prefix-only ID (no slug). |
| `cmd/kanbanzai/main.go` | Make `--slug` optional for `get`, `update status`, `update fields`. When absent, pass empty string. |
| `cmd/kanbanzai/main_test.go` | Add tests for CLI get/update with prefix-only ID. |

**Acceptance criteria:**

- Unambiguous prefix resolves to the entity — verified for all 5 entity types (epics resolve by slug prefix).
- Ambiguous prefix returns an error listing all candidates with full IDs.
- No match returns a clear "not found" error.
- Input is normalized: break hyphens stripped, case-insensitive.
- Exact `(type, id, slug)` path still works unchanged.

---

### W2: Display Formatting Wiring (B2)

**Goal:** wire `FormatFullDisplay` and `FormatShortDisplay` into CLI output and MCP responses so humans and agents see break-hyphenated IDs.

**Current state:** `FormatFullDisplay`, `FormatShortDisplay`, and `ShortestUniquePrefix` are implemented and tested in `internal/id/display.go` but never called from any output path.

**Approach:** call `FormatFullDisplay` in CLI print functions and MCP JSON responses. Use `FormatShortDisplay` in list output where multiple entities of the same type appear. The canonical (non-hyphenated) form remains in storage and in the `id` field of YAML files — display formatting is output-only.

**Changes:**

| File | Change |
|------|--------|
| `cmd/kanbanzai/main.go` | In `printCreateResult`: format `result.ID` through `id.FormatFullDisplay`. In `printGetResult`: same. In `printListResults`: compute shortest unique prefixes for the result set, use `id.FormatShortDisplay` for the listing, `id.FormatFullDisplay` for detail views. |
| `cmd/kanbanzai/main_test.go` | Update expected output in tests to include break hyphens. |
| `internal/mcp/entity_tools.go` | In `getEntityTool` response: add a `display_id` field with `id.FormatFullDisplay(result.ID)`. In `listEntitiesTool` response: add `display_id` to each entry. In `createEntityTool`/`createTaskTool` responses: add `display_id`. Keep the raw `id` field unchanged for machine consumption. |
| `internal/mcp/server_test.go` | Verify `display_id` field is present and correctly formatted in tool responses. |

**Acceptance criteria:**

- CLI create/get/list output shows break-hyphenated IDs (e.g., `FEAT-01J3K-7MXP3RT5`).
- MCP responses include both `id` (canonical) and `display_id` (formatted).
- Epic IDs pass through unchanged (no break hyphen for slugs).
- No break hyphens appear in stored files.

---

### W3: Bootstrap Feature Status (B3)

**Goal:** transition at least one feature to `open` to satisfy bootstrap spec §9.2.

**Changes:**

Transition two features whose work is actively underway:

```
kbz update status --type feature --id FEAT-01KM8JTBFEJ4Q --slug id-system-migration --status open
kbz update status --type feature --id FEAT-01KM8JTF0MK91 --slug bootstrap-self-hosting --status open
```

Verify with `kbz health` — should still pass.

**Acceptance criteria:**

- At least one feature has `status: open` or `status: in-progress`.
- Health check passes.

---

### W4: Test Fixture Migration (N1)

**Goal:** replace old-format IDs in the 4 remaining packages with TSID13-format IDs and introduce shared test constants.

**Current state:** `internal/id/`, `internal/storage/`, `internal/service/`, `internal/validate/`, and `internal/mcp/` tests are already on TSID13. Four packages still use legacy IDs: `cmd/kanbanzai/`, `internal/cache/`, `internal/document/`, `internal/model/`.

**Changes:**

| File | Change |
|------|--------|
| `internal/testutil/ids.go` (new) | Define well-known test constants: `TestEpicID = "EPIC-TESTEPIC"`, `TestFeatureID = "FEAT-01J3K7MXP3RT5"`, `TestTaskID = "TASK-01J3KZZZBB4KF"`, `TestBugID = "BUG-01J4AR7WHN4F2"`, `TestDecisionID = "DEC-01J3KABCDE7MX"`, `TestDocumentID = "DOC-01J3KDOCTEST01"`. |
| `internal/cache/cache_test.go` | Replace `FEAT-001`, `BUG-001`, `DEC-001`, `E-001`, `E-002`, `FEAT-001.1` with constants from `testutil`. |
| `internal/model/entities_test.go` | Replace `E-001`, `FEAT-001`, `FEAT-002` with constants from `testutil`. Add `EntityKindDocument` to `TestEntityKind_Values`. Add `var _ model.Entity = model.Document{}` if a `model.Document` struct is added (see note below). |
| `internal/document/service_test.go` | Replace `FEAT-001` references with TSID13 equivalents. |
| `internal/document/store_test.go` | Replace `DOC-001`, `FEAT-042` with TSID13 equivalents. |
| `internal/document/validate_test.go` | Replace `FEAT-001`, `FEAT-999` with TSID13 equivalents. |
| `cmd/kanbanzai/main_test.go` | Covered in W5 below. |

**Note on `model.Document`:** the document system uses its own `Document` struct in `internal/document/types.go` with `Meta`/`Body` fields. It does not implement the `model.Entity` interface. This appears to be a deliberate design choice — documents are a separate subsystem with different semantics (body content, MIME type). Adding a `model.Document` struct would create a parallel type with no clear consumer. I recommend leaving this as-is unless you want documents to be queryable through the same entity service paths. If so, let me know and I'll add a work item.

**Acceptance criteria:**

- No old-format IDs (`E-001`, `FEAT-001`, etc.) in test code except intentional legacy-recognition tests.
- Shared constants used consistently.
- Full test suite passes.

---

### W5: CLI Test Modernisation (N2)

**Goal:** fix the CLI test fake to use TSID13 IDs and correct flag names.

**Current state:** `fakeEntityService` in `main_test.go` returns `E-001` for epics and `FEAT-001.1` for tasks. Test assertions check for `E-` prefix and `FEAT-NNN.N` format. The task creation test passes `--feature FEAT-001` instead of `--parent_feature FEAT-001`.

**Changes:**

| File | Change |
|------|--------|
| `cmd/kanbanzai/main_test.go` | Update `newFakeEntityService`: return `EPIC-TESTEPIC` for epics, `FEAT-{TSID}` for features, `TASK-{TSID}` for tasks, `BUG-{TSID}` for bugs, `DEC-{TSID}` for decisions. Update ID format assertions: check for `EPIC-` prefix on epics, `TASK-` prefix + 18-char length on tasks, etc. Fix task creation test: change `--feature` to `--parent_feature`. Use constants from `testutil` (W4). |

**Acceptance criteria:**

- CLI tests use TSID13 IDs throughout.
- Task creation test uses `--parent_feature`.
- All CLI tests pass.

---

### W6: Decision Log Housekeeping (N3, N4)

**Goal:** strike through resolved follow-ups and update filename examples.

**Changes:**

| File | Change |
|------|--------|
| `work/plan/phase-1-decision-log.md` — P1-DEC-016 | Strike through all 3 follow-ups with resolution notes: "Go 1.25.0 — see `go.mod`", "`mcp-go v0.45.0` adopted — see `go.mod`", "Module structure created — see `go.mod`". |
| `work/plan/phase-1-decision-log.md` — P1-DEC-006 | Add TSID13 filename examples to the file layout section, e.g., `FEAT-01J3K7MXP3RT5-profile-editing.yaml`, `EPIC-PHASE1-phase-1-kernel.yaml`. |

**Acceptance criteria:**

- No P1-DEC-016 follow-up appears open when it's resolved in code.
- P1-DEC-006 shows concrete examples of TSID13 filenames.

---

### W7: Crockford Normalisation Fix (N5)

**Goal:** make `NormalizeTSID` perform full Crockford base32 normalization, not just uppercasing.

**Current state:** `NormalizeTSID` calls `ValidateTSID13` (which accepts `I`, `L`, `O` via the decode table) then returns `strings.ToUpper(s)`. This leaves non-canonical characters like `O` in the output.

**Changes:**

| File | Change |
|------|--------|
| `internal/id/tsid.go` | In `NormalizeTSID`: after uppercasing, replace `I` → `1`, `L` → `1`, `O` → `0`. This matches the Crockford spec and ensures round-trip correctness. |
| `internal/id/tsid_test.go` | Add test cases: `"01j3k7mxp3rto"` normalizes to `"01J3K7MXP3RT0"` (O→0); `"01J3K7MXP3RTl"` normalizes to `"01J3K7MXP3RT1"` (l→1); `"01J3K7MXP3RTi"` normalizes to `"01J3K7MXP3RT1"` (i→1). |

**Acceptance criteria:**

- `NormalizeTSID` output contains only canonical Crockford characters (0-9, A-H, J-K, M-N, P-T, V-W, X-Y, Z).
- Ambiguous characters are mapped: `I/i/L/l → 1`, `O/o → 0`.

---

### W8: Bootstrap Bug Verification (N6)

**Goal:** create a bug entity to verify §9.7 end-to-end.

**Changes:**

Create a bug through the CLI, verify it round-trips, then close or remove it:

```
kbz create bug --slug review-test-bug --summary "Verify bug creation for bootstrap §9.7"
kbz health
```

If the tool is working correctly, the bug is created with a `BUG-{TSID}` ID, health passes, and §9.7 is satisfied. The bug can remain in state (it documents a real verification step) or be removed — your call.

**Acceptance criteria:**

- Bug entity created with valid TSID13 ID.
- Health check passes with the new bug in state.
- Task, Bug, and Decision creation all verified (Task and Decision already exist from prior bootstrap).

---

## 3. Dependency Graph

```
W7: Crockford fix ─────────────────────────────── (independent)

W1: Prefix resolution ─────┐
                            ├──► W4: Test fixture migration ──► W5: CLI test modernisation
W2: Display formatting ────┘

W6: Decision log housekeeping ─────────────────── (independent)

W3: Bootstrap feature status ──► W8: Bootstrap bug ── (after code changes settled)
```

W1 and W2 are the core code changes. W4 and W5 depend on them because the test constants and assertions need to reflect the final behavior (display-formatted output, prefix resolution). W7 and W6 are independent. W3 and W8 are workflow-state changes best done after code is settled.

---

## 4. Recommended Execution Order

### Phase A: Independent Fixes (W6, W7)

No dependencies on other items. Can run in parallel with Phase B.

1. **W7** — fix `NormalizeTSID` Crockford substitution, add tests.
2. **W6** — strike through P1-DEC-016 follow-ups, update P1-DEC-006 filename examples.

**Gate:** `go test ./internal/id/...` passes. Decision log is internally consistent.

### Phase B: Core Features (W1, W2)

The two blocking code changes. W1 and W2 can proceed in parallel — they touch different parts of the service/MCP/CLI layers.

3. **W1** — implement `ResolvePrefix`, wire into service → MCP → CLI.
4. **W2** — wire `FormatFullDisplay`/`FormatShortDisplay` into CLI print functions and MCP responses.

**Gate:** `go build ./...` succeeds. New unit tests pass. Manual smoke test: create a feature, get it by prefix (no slug), confirm display-formatted ID in output.

### Phase C: Test Migration (W4, W5)

Depends on Phase B — the test expectations need to reflect display formatting and prefix resolution.

5. **W4** — create `internal/testutil/ids.go`, migrate test fixtures in `cache`, `model`, `document` packages.
6. **W5** — modernise CLI test fake: TSID13 IDs, correct flag names, updated assertions.

**Gate:** `go test ./...` passes with zero failures. `go test -race ./...` passes. No old-format IDs remain outside intentional legacy tests.

### Phase D: Bootstrap Completion (W3, W8)

After all code changes are settled.

7. **W3** — transition features to `open`.
8. **W8** — create bug entity, verify §9.7.

**Gate:** `kbz health` passes. All 8 bootstrap acceptance criteria (§9.1–§9.8) satisfied.

---

## 5. Effort Estimates

| Item | Effort | Notes |
|------|--------|-------|
| W1: Prefix resolution | Medium-large | New service method + wiring through 3 layers + tests |
| W2: Display formatting wiring | Small-medium | Calling existing functions from output paths + updating test expectations |
| W3: Bootstrap feature status | Trivial | Two CLI commands |
| W4: Test fixture migration | Medium | Wide-reaching but mechanical |
| W5: CLI test modernisation | Small-medium | Concentrated in one file but many assertions to update |
| W6: Decision log housekeeping | Small | Text edits only |
| W7: Crockford normalisation | Small | 3-line code change + tests |
| W8: Bootstrap bug verification | Trivial | One CLI command |

**Total estimated effort:** Medium-large. The critical path is W1 → W4/W5. W1 (prefix resolution) is the only item with meaningful design work.

---

## 6. Items Not Included

| Item | Reason |
|------|--------|
| `model.Document` struct implementing `Entity` | Documents are a separate subsystem with different semantics (body content, MIME type, separate store). Adding a parallel `model.Document` would create a type with no consumer. The `EntityKindDocument` constant and `TypePrefix` mapping exist for ID allocation — that's sufficient. If documents should be queryable through the entity service, this is a design decision, not a bug fix. |
| §9.5 (`validate` command) reinterpretation | The `validate` command is a pre-persist candidate validator, not a project-wide state check. The `health` command satisfies the bootstrap spec's intent. This is a spec-interpretation question for the human, noted in the bootstrap activation decision record. |