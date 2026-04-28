# Layer 1 / Track A Implementation Review

- Date: 2026-03-19
- Scope: Core data model and file representation (Layer 1, Track A)
- Reviewer: AI agent
- Status: Review complete — issues found

---

## 1. Purpose

This report examines the implementation of Layer 1 (core data model and storage) and Track A (core model and file representation) from the Phase 1 implementation plan. It verifies:

- completeness against the implementation plan's Track A task list
- conformance to the Phase 1 specification
- conformance to accepted decisions (P1-DEC-006 through P1-DEC-010)
- code quality, idiom, and style per AGENTS.md
- test coverage and test quality

References:
- `work/plan/phase-1-implementation-plan.md` §6.1, §7.1
- `work/spec/phase-1-specification.md` §7–§14
- `work/plan/phase-1-decision-log.md` P1-DEC-006 through P1-DEC-010

---

## 2. Implementation Inventory

The following packages exist:

| Package | Files | Tests | Purpose |
|---|---|---|---|
| `cmd/kanbanzai` | `main.go` | `main_test.go` | CLI entry point |
| `internal/core` | `paths.go` | — | Instance root and state path constants |
| `internal/model` | `entities.go` | — | Entity type definitions (structs and interfaces) |
| `internal/id` | `allocator.go` | `allocator_test.go` | ID allocation and validation |
| `internal/storage` | `entity_store.go` | `entity_store_test.go` | File I/O and canonical YAML serialization |
| `internal/validate` | `lifecycle.go` | `lifecycle_test.go` | Lifecycle state machines and transition validation |
| `internal/service` | `entities.go` | `entities_test.go` | Entity CRUD operations |

All tests pass with `-race`. `go vet ./...` reports no issues.

---

## 3. Track A Task Checklist

Track A tasks from `phase-1-implementation-plan.md` §7.1:

| Task | Status | Notes |
|---|---|---|
| Define internal entity types for Epic, Feature, Task, Bug, Decision | ⚠️ Partial | Types are defined in `internal/model` but are dead code — see §5.1 |
| Define canonical serialization model | ✅ Done | `MarshalCanonicalYAML` / `UnmarshalCanonicalYAML` in storage |
| Define deterministic field order | ⚠️ Deviation | Alphabetical sort, not schema-defined — see §5.3 |
| Define file naming conventions | ✅ Done | `ID-slug.yaml` |
| Define path conventions for each entity type | ✅ Done | `.kbz/state/<type>s/` |
| Implement load/save operations | ✅ Done | `EntityStore.Write` and `EntityStore.Load` |
| Implement normalization of timestamps and basic field formatting | ✅ Done | RFC 3339 timestamps, slug normalization |

Track A outputs:

| Output | Status | Notes |
|---|---|---|
| Stable read/write core | ✅ Done | Working end-to-end |
| Example files for each entity type | ❌ Missing | No example/fixture files exist in the repository |

---

## 4. Specification Compliance — What Is Correct

The following are correctly implemented per the spec and accepted decisions.

### 4.1 Entity types (spec §7)

All five Phase 1 entity types (Epic, Feature, Task, Bug, Decision) are defined and supported through the service layer for create, read, list, and status-update operations.

### 4.2 Required fields (spec §9 / P1-DEC-009)

All minimum required fields from spec §9 are present on every entity struct and populated correctly at creation time. The three-way field classification from P1-DEC-009 is implemented:

- System-generated fields (`id`, `status`, `created`/`reported`/`date`) are auto-populated.
- Defaultable Bug fields (`severity`, `priority`, `type`) default to `medium`/`medium`/`implementation-defect`.
- Caller-must-supply fields are validated as required.

### 4.3 ID allocation (P1-DEC-007)

The `internal/id` package correctly implements:

- `E-NNN` for Epics
- `FEAT-NNN` for Features
- `BUG-NNN` for Bugs
- `DEC-NNN` for Decisions
- `FEAT-NNN.N` for Tasks (feature-local sub-IDs)

Allocation scans existing IDs and increments the highest sequence number. Validation enforces prefix correctness and structural format. ID sorting is canonical.

### 4.4 Lifecycle transitions (P1-DEC-010)

The `internal/validate` package implements the exact exhaustive transition tables from P1-DEC-010. All four general rules are enforced:

1. Entry states are system-enforced — new entities always start in their entry state.
2. Terminal states are irreversible.
3. Self-transitions are illegal.
4. Unknown states are rejected.

The `cannot-reproduce → triaged` reopening path for Bug is correctly included. `cannot-reproduce` is correctly classified as non-terminal.

### 4.5 File layout (P1-DEC-006)

State files are stored under `.kbz/state/` with per-type subdirectories (`epics/`, `features/`, `tasks/`, `bugs/`, `decisions/`). Filenames follow the `ID-slug.yaml` convention.

### 4.6 YAML serialization (P1-DEC-008)

The implementation produces:

- Block style only
- Deterministic key order (alphabetical — see §5.3 for the deviation)
- No anchors, aliases, or tags
- Quoted strings only when required
- Scalar booleans and nulls handled correctly

Round-trip tests (write → read → compare) exist and pass.

### 4.7 One file per entity (spec §14.1)

Each entity is stored in its own file. Implemented correctly.

---

## 5. Issues Found

### 5.1 `model` package is dead code

**Severity:** Structural concern
**Spec reference:** Track A task "define internal entity types"

The `internal/model/entities.go` file defines typed Go structs for all five entities with an `Entity` interface. However, nothing in the codebase uses these types. The service layer works entirely with `map[string]any`, and the storage layer uses `EntityRecord` with `Fields map[string]any`.

The types are defined but never used. The Entity interface (`GetKind()`, `GetID()`, `GetSlug()`) is never called. Coverage is 0%.

**Consequence:** Entity creation has no compile-time field validation. Typos in field names or type mismatches in values would only be caught by tests that happen to check for them. The typed structs should either be integrated into the service/storage pipeline or removed until needed.

### 5.2 Duplicate entity-kind constants

**Severity:** Code quality / DRY violation

There are three independent sets of entity-type constants:

- `model.EntityKind` — `EntityKindEpic`, `EntityKindFeature`, etc.
- `validate.EntityKind` — `EntityEpic`, `EntityFeature`, etc.
- `id.EntityType` — `EntityEpic`, `EntityFeature`, etc.

All three are string aliases of the same values, defined independently. This creates a maintenance risk — if someone modifies one set without updating the others, the system silently diverges. The `service` package contains a `validateKindForType()` bridge function that would be unnecessary with a single canonical definition.

### 5.3 YAML field order is alphabetical, not schema-defined

**Severity:** Specification deviation
**Spec reference:** P1-DEC-008

P1-DEC-008 specifies "deterministic **schema-defined** key ordering." The implementation sorts keys alphabetically in `MarshalCanonicalYAML`. This is deterministic, but it is not schema-defined — it does not reflect the logical entity structure.

Compare the spec §9.1 Epic field order vs. alphabetical sort:

| Spec order | Alphabetical sort |
|---|---|
| `id` | `created` |
| `slug` | `created_by` |
| `title` | `id` |
| `status` | `slug` |
| `summary` | `status` |
| `created` | `summary` |
| `created_by` | `title` |

P1-DEC-008 follow-up explicitly calls out: "define the exact schema-defined field order for each phase 1 entity type." This has not been done.

**Recommendation:** Define a per-entity-type field ordering (likely matching the spec §9 listing order) and pass it to the serializer. Alphabetical sort can remain as the fallback for unknown fields.

### 5.4 `time.Now().UTC` initialization bug

**Severity:** Bug (production impact)

```go
// internal/service/entities.go line 94
now: time.Now().UTC,
```

This captures `time.Now()` once at construction time and binds the `.UTC` method of that specific `Time` value. Every subsequent call to `s.now()` returns the same instant — the moment the service was created — not the current time.

Tests work around this by overwriting `svc.now` with a fixed time, masking the bug. In real CLI use, every entity created in a single command invocation gets the same timestamp, which happens to be acceptable by coincidence (the CLI creates one entity per invocation). But any future use case creating multiple entities through a single service instance will produce identical timestamps.

**Fix:**

```go
now: func() time.Time { return time.Now().UTC() },
```

### 5.5 CLI tests write to the real filesystem

**Severity:** Test isolation violation
**AGENTS.md reference:** "Use `t.TempDir()` for filesystem tests — never write to the working directory."

The CLI tests in `cmd/kanbanzai/main_test.go` invoke `run()` which creates `service.NewEntityService("")`, defaulting to `.kbz/state/` relative to the current working directory. This creates real entity files on disk during testing.

Evidence: a `.kbz/` directory exists inside `cmd/kanbanzai/`:

```
cmd/kanbanzai/.kbz/
```

This directory was created by test runs. Tests that create entities (e.g., `TestRunCreate_CreatesEntities`, `TestRunList_PrintsEntityCountAndEntries`) leave behind real files and can interfere with each other across runs.

**Recommendation:** The CLI's `run()` function should accept a root directory parameter (or the service should be injected), and tests should use `t.TempDir()`.

### 5.6 Custom YAML implementation is under-tested

**Severity:** Risk
**Spec reference:** P1-DEC-008

The `internal/storage` package contains a ~300-line hand-written YAML serializer and parser. While it works for current test cases, coverage of critical paths is low:

| Function | Coverage |
|---|---|
| `writeYAMLList` | 47.1% |
| `parseList` | 46.9% |
| `formatScalar` | 37.5% |
| `needsQuotes` | 76.9% |

Specific gaps:

- No test exercises nested maps inside list items via `writeYAMLList`
- No test for the `fmt.Stringer` branch in `formatScalar`
- No test for numeric scalar parsing
- No test for `nil` value serialization
- No test verifying idempotent writes (write the same data twice, compare output byte-for-byte)
- The `quoteString` function maps `\\` to `\\\\`, which double-escapes backslashes — a single backslash in input becomes two backslashes in output

Since no standard YAML library is used (there is not even a `go.sum` file — the project has zero external dependencies), this hand-rolled implementation is the entire YAML layer. It needs significantly more edge-case testing or should be replaced with a real YAML library wrapped in a canonical-output enforcer.

### 5.7 `go.sum` is missing

**Severity:** Minor
**AGENTS.md reference:** "Commit `go.sum` with `go.mod`"

The project has `go.mod` but no `go.sum`. Currently there are no external dependencies, so this is harmless, but `go mod tidy` should be run and the result committed to establish the practice.

### 5.8 No tests for `internal/core`

**Severity:** Minor

The `core` package has 0% coverage. It contains only `RootPath()` and `StatePath()`, which are trivial but define the canonical instance paths that the rest of the system depends on. A simple test locking down the expected return values would prevent accidental changes.

### 5.9 `cannot-reproduce → triaged` transition lacks a dedicated test

**Severity:** Minor
**Spec reference:** P1-DEC-010

P1-DEC-010 explicitly highlights `cannot-reproduce` as a special near-terminal state. The test suite verifies `IsTerminalState(EntityBug, "cannot-reproduce") == false` and `CanTransition(EntityBug, "cannot-reproduce", "triaged") == true`, which is good. However, there is no integration-level test (in the service layer) that exercises the full flow: create a bug → transition to `triaged` → transition to `cannot-reproduce` → reopen to `triaged`. Given the spec emphasis, this deserves an explicit end-to-end test.

### 5.10 Decision `status` field is not in spec §9.5 minimum fields

**Severity:** Observation (not a defect)

Spec §9.5 lists the Decision minimum required fields as: `id`, `slug`, `summary`, `rationale`, `decided_by`, `date`. It does not list `status`. However, spec §10.5 defines Decision lifecycle states (`proposed`, `accepted`, `rejected`, `superseded`), and P1-DEC-010 defines entry state `proposed`.

The implementation adds `status: "proposed"` to every Decision at creation time. This is the correct behavior — the lifecycle requires a status field — but the omission from §9.5's minimum field list is a minor spec inconsistency worth noting.

### 5.11 Example files not produced

**Severity:** Minor
**Plan reference:** Track A outputs

Track A specifies "example files for each entity type" as an output. No example or fixture YAML files exist in the repository. The round-trip tests construct maps in code rather than loading from `testdata/` fixtures.

---

## 6. Test Coverage Summary

| Package | Coverage | Assessment |
|---|---|---|
| `internal/validate` | 95.8% | ✅ Excellent — thorough table-driven tests for all lifecycle operations |
| `internal/id` | 83.0% | ✅ Good — covers allocation, validation, sorting, edge cases |
| `cmd/kanbanzai` | 76.6% | ⚠️ Adequate coverage, but tests write to the real filesystem |
| `internal/service` | 73.8% | ⚠️ Adequate — happy paths and some error paths covered |
| `internal/storage` | 69.2% | ⚠️ Low for a critical serialization module |
| `internal/model` | 0.0% | ❌ Dead code — no consumers exist |
| `internal/core` | 0.0% | ⚠️ Trivial code, but untested |
| **Total** | **73.7%** | |

### Testing strengths

- Table-driven tests used consistently across all test files
- `t.Parallel()` used throughout
- Tests cover both happy paths and error/rejection cases
- Lifecycle validation tests are comprehensive and well-structured
- The service test helper `newTestEntityService` correctly injects a fixed clock for deterministic timestamps

### Testing gaps

- No tests for the `model` package (it is unused)
- No filesystem-isolation in CLI tests
- YAML serializer edge cases are under-covered (nested list maps, nil values, Stringer interface, numeric scalars, backslash escaping)
- No idempotent-write test (write → write → compare)
- No `testdata/` fixture files for canonical entity YAML
- `parseRecordIdentity` coverage is 38.9% — only the feature and epic paths are exercised through integration; bug, decision, and task paths through this function are not directly tested
- `validateKindForType` coverage is 42.9% — only epic, feature, and bug kinds are exercised

---

## 7. Code Quality Assessment

### Positive aspects

- Clean, idiomatic Go style with consistent formatting
- Good error wrapping: `fmt.Errorf("context: %w", err)` used throughout
- Table-driven tests with descriptive names (`TestFunctionName_Scenario`)
- Clear package separation: `model`, `id`, `validate`, `storage`, `service`, `core`
- The lifecycle validation module (`internal/validate`) is particularly well-designed — clean data tables, clear exported API, comprehensive tests
- The ID allocator correctly handles mixed entity families and ignores invalid IDs during scan
- Slug normalization handles spaces, repeated dashes, and surrounding dashes
- `t.Helper()` not needed because the test helper `newTestEntityService` does not perform assertions

### Areas for improvement

- The `model` types should either be integrated or removed — dead code is confusing
- Entity-type constants should be defined once in a shared location
- The YAML serializer needs more tests or should use a standard library
- CLI tests need test isolation via dependency injection or temp directories
- The `time.Now().UTC` method-reference bug should be fixed
- YAML field ordering should be schema-defined per P1-DEC-008

---

## 8. Recommended Actions

Ordered by priority:

1. **Fix the `time.Now().UTC` bug** — Change to `func() time.Time { return time.Now().UTC() }` in `NewEntityService`. Low effort, real bug.

2. **Fix CLI test isolation** — Inject the root directory into the CLI path so tests use `t.TempDir()`. Remove the `cmd/kanbanzai/.kbz/` directory created by previous test runs.

3. **Implement schema-defined field ordering** — Define per-entity-type field order lists matching spec §9 and pass them to `MarshalCanonicalYAML`. This is a spec requirement, not a nice-to-have.

4. **Resolve the `model` package** — Either integrate the typed structs into the service/storage pipeline (preferred — gives compile-time field safety) or remove them to avoid dead code. If integrating, unify the three sets of entity-type constants.

5. **Improve YAML serializer test coverage** — Add tests for nested maps in lists, nil values, numeric scalars, backslash handling, Stringer interface, and idempotent writes. Target ≥85% coverage for `internal/storage`.

6. **Add example entity files** — Create `testdata/` fixtures for each entity type, as specified by Track A outputs. Use these in round-trip tests.

7. **Run `go mod tidy` and commit `go.sum`** — Establishes the practice for when external dependencies arrive.

8. **Add basic `internal/core` tests** — Lock down `RootPath()` and `StatePath()` return values.

---

## 9. Conclusion

The Layer 1 / Track A implementation provides a working foundation. The core mechanics are sound: entities can be created, stored, loaded, listed, and transitioned through validated lifecycle states. ID allocation matches the accepted strategy. The YAML serializer produces deterministic output. The lifecycle validation is thorough and well-tested.

The main structural issues are the dead `model` types, the alphabetical-rather-than-schema-defined YAML field ordering, and the CLI test isolation problem. The `time.Now().UTC` method-reference bug is a real defect that should be fixed immediately. The custom YAML implementation is a risk area that needs more testing.

None of these issues block forward progress on Layer 2, but items 1–3 in the recommended actions should be addressed before building on this foundation, as they represent a real bug, a test hygiene problem, and a spec deviation respectively.

---

## 10. Supplemental Follow-Up Notes

This section supplements the original review text without modifying its findings. It records what was done after the review, what was not done, and why any deviations from the recommendations were taken.

### 10.1 Implemented follow-up items

The following review recommendations were implemented after the original review was written:

- **5.4 / Recommendation 1:** Fixed the `time.Now().UTC` initialization bug by changing the service clock initialization to call `time.Now().UTC()` on each invocation.
- **5.5 / Recommendation 2:** Fixed CLI test isolation so tests no longer write to the real working directory and instead use isolated temporary state.
- **5.3 / Recommendation 3:** Implemented schema-defined field ordering for canonical YAML output, with alphabetical fallback retained for unknown fields.
- **5.1 / Recommendation 4:** Integrated the typed `internal/model` structs into the service/storage pipeline instead of leaving them unused.
- **5.8 / Recommendation 8:** Added basic `internal/core` tests to lock down canonical path behavior.
- **5.9:** Added an explicit service-level bug reopening test covering `reported -> triaged -> cannot-reproduce -> triaged`.
- **5.11 / Recommendation 6:** Added example entity files as canonical fixture files under `testdata/entities/` and wired them into storage tests.

### 10.2 Partially implemented items

#### 5.6 / Recommendation 5 — YAML serializer coverage

This recommendation was **partially implemented**.

Work completed:

- Added tests for nested maps inside list items
- Added tests for `nil` value serialization
- Added tests for numeric scalar parsing
- Added tests for the `fmt.Stringer` path
- Added an idempotent write test
- Added fixture-driven round-trip and load tests
- Added a round-trip test for backslash-containing strings

What was **not** done:

- The custom YAML implementation was **not** replaced with a standard YAML library
- Production escaping behavior was **not** changed as part of the backslash concern

Reason for deviation:

- The recommendation included two possible directions: improve tests, or replace the YAML layer with a library plus canonical-output enforcement.
- The smaller and more directly responsive path was taken: add targeted coverage around the current implementation.
- Replacing the serializer would have been a materially broader architectural change than necessary to address the immediate review findings.
- The backslash concern was treated conservatively because changing escaping semantics without a clearer canonical-YAML requirement would have been a speculative behavior change. Instead, round-trip behavior was tested directly.

#### 5.7 / Recommendation 7 — `go mod tidy` and `go.sum`

This recommendation was **partially implemented**.

Work completed:

- Ran `go mod tidy`

What was **not** done:

- No `go.sum` file was added

Reason for deviation:

- The recommendation said to run `go mod tidy` and commit `go.sum`
- That was only partially possible because the module currently has no external dependencies requiring checksum entries
- As a result, `go mod tidy` completed successfully but did not generate a `go.sum` file

### 10.3 Decision `status` inconsistency

#### 5.10 — Spec inconsistency

This item was **not addressed in code or docs** as part of the implementation follow-up.

Reason:

- The review itself correctly classifies this as an observation rather than a defect
- The inconsistency is between the spec's minimum-field listing and the lifecycle requirements
- Resolving it belongs in specification/documentation maintenance, not in a silent implementation change

### 10.4 Supplemental note on example files

The recommendation asked for "example files for each entity type." The implementation used `testdata/entities/` fixture files.

Reason for this form of implementation:

- It satisfies the Track A output requirement
- It also makes the files executable test assets rather than passive examples
- This keeps the repository tidy and ensures the examples remain verified by tests

### 10.5 Remaining follow-up work

The following work remains reasonable follow-up based on the original review:

1. Review the custom YAML implementation more deeply, especially escaping semantics and parser edge cases
2. Add more direct unit coverage for helper paths such as `parseRecordIdentity` and `validateKindForType`
3. Resolve the Decision `status` field inconsistency in the spec/docs

---

## 11. Verification of Follow-Up Implementation

- Date: 2026-03-19
- Purpose: Independent verification that the follow-up actions described in §10 were actually implemented.

All tests pass. `go vet ./...` is clean. Race detector finds no issues. Overall statement coverage is 74.4%.

### 11.1 Recommendation 1 — `time.Now().UTC` bug fix

**Status: ✅ Verified.**

`NewEntityService` (line 96–98 of `entities.go`) now reads:

```go
now: func() time.Time {
    return time.Now().UTC()
},
```

This is the correct form — a closure that calls `time.Now().UTC()` fresh on every invocation. The original bug (capturing a method reference to a single frozen `Time` value) is fixed.

### 11.2 Recommendation 2 — CLI test isolation

**Status: ⚠️ Mostly verified. One residual artifact remains.**

The CLI tests no longer hit the real filesystem. The implementation introduced a `dependencies` struct with a `newEntityService` factory and a `fakeEntityService` for tests. No call to `service.NewEntityService("")` exists anywhere in the test file. Tests use `testDependencies()` / `testDependenciesWithService()` helpers that wire up the fake — no filesystem access occurs during testing.

This is arguably a better fix than the review recommended (`t.TempDir()` injection) because it eliminates filesystem access entirely rather than redirecting it.

**Residual issue:** The directory `cmd/kanbanzai/.kbz/` still exists on disk. It is empty, not git-tracked (matched by the `.gitignore` `kanbanzai` line), and harmless — but it is leftover pollution from a previous test run before the fix was applied. It should be deleted.

**Also noted:** The production CLI still hardcodes `""` as the root argument at all four call sites (`runCreate`, `runGet`, `runList`, `runUpdate`). This is fine for Phase 1 (the CLI always operates on `.kbz/state/` relative to the working directory), but it means the `newEntityService` factory's `root string` parameter is never used in production. This is an obvious extension point for a future `--root` flag or `KBZ_ROOT` env var, not a defect.

### 11.3 Recommendation 3 — Schema-defined field ordering

**Status: ✅ Verified.**

`MarshalCanonicalYAML` now accepts `entityType string` as its first parameter and delegates to `orderedKeys()`, which consults `fieldOrderForEntityType()` — a switch statement returning a hardcoded ordered slice per entity type.

Per-entity field orders match the spec §9 listing:

| Entity | Leading fields |
|---|---|
| Epic | `id`, `slug`, `title`, `status`, `summary`, `created`, `created_by`, `features` |
| Feature | `id`, `slug`, `epic`, `status`, `summary`, `created`, `created_by`, `spec`, `plan`, `tasks`, `decisions`, `branch`, `supersedes`, `superseded_by` |
| Task | `id`, `feature`, `slug`, `summary`, `status`, `assignee`, `depends_on`, `files_planned`, `started`, `completed`, `verification` |
| Bug | `id`, `slug`, `title`, `status`, `severity`, `priority`, `type`, `reported_by`, `reported`, `observed`, `expected`, ... (14 optional fields follow) |
| Decision | `id`, `slug`, `summary`, `rationale`, `decided_by`, `date`, `affects`, `supersedes`, `superseded_by` |

Alphabetical fallback is retained for:
- Unknown fields on a known entity type (appended after schema-ordered fields, sorted alphabetically)
- Unknown entity types (all fields sorted alphabetically — same as the original behavior)
- Nested sub-maps within any entity (always alphabetical; schema ordering applies only to the top level)

The P1-DEC-008 deviation identified in §5.3 is resolved.

### 11.4 Recommendation 4 — Model package integration

**Status: ⚠️ Partially verified. Write path integrated; read path still uses raw maps.**

The `internal/model` package is now imported by four packages (`id`, `service`, `storage`, `validate`). The service layer constructs typed model structs (e.g., `model.Epic{}`) on the creation path, then converts them to `map[string]any` via helper functions (`epicFields()`, `featureFields()`, etc.) for storage.

However, the read path (Get, List, UpdateStatus) still operates entirely on `map[string]any`. The `CreateResult` and `GetResult` types carry `State map[string]any`, not model structs. `UpdateStatus` mutates the raw map directly (`record.Fields["status"] = nextStatus`). There is no deserialization from YAML back into model structs.

This is a pragmatic halfway point: creation benefits from compile-time field safety, but reads do not. The follow-up notes don't claim full integration — they say "integrated the typed `internal/model` structs into the service/storage pipeline," which is accurate for the write side.

**Duplicate constants (§5.2):** Reduced from three independent sets to one authoritative source plus one alias layer. `model.EntityKind` is the canonical type. `validate` re-exports the constants via a type alias (`type EntityKind = model.EntityKind`) for convenience. `id` uses `model.EntityKind` directly. This is acceptable — the type alias ensures type compatibility, so there is no divergence risk.

**Model coverage:** The `internal/model` package still shows 0% coverage because the model structs have no methods with branching logic — only trivial getters (`GetKind`, `GetID`, `GetSlug`). These are exercised indirectly through the service layer, but since they're in a separate package, the coverage tool doesn't attribute them. This is not a concern.

### 11.5 Recommendation 5 — YAML serializer test coverage

**Status: ✅ Verified.**

The following tests were confirmed present in `entity_store_test.go`:

| Claimed test | Found | Test function |
|---|---|---|
| Nested maps inside list items | ✅ | `TestMarshalCanonicalYAML_NestedMapsInsideListItems` |
| Nil value serialization | ✅ | `TestMarshalCanonicalYAML_SerializesNilValue` |
| Numeric scalar parsing | ✅ | `TestUnmarshalCanonicalYAML_ParsesNumericScalars` |
| `fmt.Stringer` path | ✅ | `TestMarshalCanonicalYAML_UsesStringer` |
| Idempotent write | ✅ | `TestMarshalCanonicalYAML_IdempotentWrite` |
| Backslash round-trip | ✅ | `TestMarshalCanonicalYAML_BackslashEscapingRoundTrip` |

Coverage for `internal/storage` improved from 69.2% to 75.4%. Key function improvements:

| Function | Before | After |
|---|---|---|
| `writeYAMLList` | 47.1% | 94.1% |
| `writeYAMLField` | 83.3% | 96.7% |
| `parseScalar` | 83.3% | 87.5% |

Remaining low-coverage functions: `parseList` (46.9%), `formatScalar` (50.0%). These are noted in §10.5 as remaining follow-up work.

### 11.6 Recommendation 6 — Example entity files

**Status: ✅ Verified.**

Five fixture files exist under `testdata/entities/`:

- `epic.yaml`
- `feature.yaml`
- `task.yaml`
- `bug.yaml`
- `decision.yaml`

These are exercised by `TestCanonicalYAML_FixturesRoundTrip` and `TestEntityStore_Load_FixtureFiles` in the storage test suite, making them active test assets rather than passive examples. The Track A output requirement is satisfied.

### 11.7 Recommendation 7 — `go mod tidy` and `go.sum`

**Status: ✅ Verified (correctly not done).**

`go mod tidy` was run, but no `go.sum` was generated because the module has zero external dependencies. Go does not create a `go.sum` file when there are no checksums to record. The follow-up notes correctly explain this.

### 11.8 Recommendation 8 — `internal/core` tests

**Status: ✅ Verified.**

`internal/core/paths_test.go` exists. Coverage is 100.0%.

### 11.9 §5.9 — Cannot-reproduce reopening test

**Status: ✅ Verified.**

`TestEntityService_UpdateStatus_ReopensCannotReproduceBug` exists in `entities_test.go`. It exercises the full flow:

1. Create bug (status: `reported`)
2. Transition to `triaged`
3. Transition to `cannot-reproduce`
4. Reopen to `triaged`
5. Verify persisted status via `Get()`

This satisfies the P1-DEC-010 emphasis on the `cannot-reproduce` near-terminal special case.

### 11.10 Updated coverage summary

| Package | Before | After | Change |
|---|---|---|---|
| `internal/validate` | 95.8% | 88.5% | ↓ (new `EntryStateOrPanic` at 0% — untested convenience function) |
| `internal/id` | 83.0% | 83.0% | — |
| `cmd/kanbanzai` | 76.6% | 76.3% | — |
| `internal/storage` | 69.2% | 75.4% | ↑ improved |
| `internal/service` | 73.8% | 70.3% | ↓ (new code paths from model integration not fully covered) |
| `internal/core` | 0.0% | 100.0% | ↑ tests added |
| `internal/model` | 0.0% | 0.0% | — (no branching logic to cover) |
| **Total** | **73.7%** | **74.4%** | ↑ |

Note: `validate` dropped from 95.8% to 88.5% because a new function `EntryStateOrPanic` was added but has no tests. This is a convenience wrapper that panics on unknown entity kinds — it's used in production code paths but never exercised with an invalid kind in tests. A test for the panic path would restore coverage.

### 11.11 Remaining items

The following items from the original review are still open:

1. **Delete `cmd/kanbanzai/.kbz/`** — empty leftover directory from pre-fix test runs.
2. **Complete model integration on the read path** — Get/List/UpdateStatus still use `map[string]any`. Not blocking but leaves the read path without compile-time field safety.
3. **Improve `parseList` coverage** (46.9%) and **`formatScalar` coverage** (50.0%) in the YAML serializer.
4. **Add a test for `EntryStateOrPanic`** with an invalid kind to restore `validate` coverage.
5. **Improve `parseRecordIdentity` coverage** (38.9%) and **`validateKindForType` coverage** (42.9%) in the service layer.
6. **Resolve the Decision `status` spec inconsistency** (§5.10) in specification/documentation.
7. **Review YAML escaping semantics** more deeply — the backslash round-trip test was added but production escaping behavior was not changed.

None of these block Layer 2 work. Items 1 and 4 are trivial. Items 3 and 5 are incremental test improvements. Items 2 and 6 are design decisions that can be deferred.

---

## 12. Fixes Applied

- Date: 2026-03-19
- Purpose: Address remaining items from §11.11.

### 12.1 Items resolved

| Item | Action taken | Result |
|---|---|---|
| 1. Delete `cmd/kanbanzai/.kbz/` | Deleted the empty leftover directory | Directory removed; `cmd/kanbanzai/` now contains only `main.go` and `main_test.go` |
| 3. `parseList` coverage | Added `TestUnmarshalCanonicalYAML_ParsesBareListItems`, `TestUnmarshalCanonicalYAML_RejectsInvalidListItem`, `TestUnmarshalCanonicalYAML_RejectsUnexpectedListIndentation` | `parseList`: 46.9% → 96.9% |
| 3. `formatScalar` coverage | Added `TestMarshalCanonicalYAML_FormatsIntegerValues`, `TestMarshalCanonicalYAML_FormatsFloatValues`, `TestMarshalCanonicalYAML_FormatsBooleanFalse`, `TestMarshalCanonicalYAML_FormatsUnknownType` | `formatScalar`: 50.0% → 81.8% |
| 4. `EntryStateOrPanic` test | Added `TestEntryStateOrPanic_ReturnsEntryState` and `TestEntryStateOrPanic_PanicsOnUnknownKind` | `EntryStateOrPanic`: 0% → 100%; `validate` package: 88.5% → 96.2% |
| 5. `validateKindForType` coverage | Added `TestValidateKindForType` with subtests for all 5 entity types plus unknown | `validateKindForType`: 42.9% → 100% |
| 5. `parseRecordIdentity` coverage | Added `TestParseRecordIdentity` with subtests for all 5 entity types plus error cases | `parseRecordIdentity`: 38.9% → 88.9% |

### 12.2 Updated coverage summary

| Package | §11 | §12 | Change |
|---|---|---|---|
| `internal/validate` | 88.5% | 96.2% | ↑ `EntryStateOrPanic` now tested |
| `internal/storage` | 75.4% | 84.2% | ↑ `parseList` and `formatScalar` now tested |
| `internal/service` | 70.3% | 74.9% | ↑ `validateKindForType` and `parseRecordIdentity` now tested |
| `internal/id` | 83.0% | 83.0% | — |
| `cmd/kanbanzai` | 76.3% | 76.3% | — |
| `internal/core` | 100.0% | 100.0% | — |
| `internal/model` | 0.0% | 0.0% | — (no branching logic) |
| **Total** | **74.4%** | **79.2%** | ↑ |

All tests pass with `-race`. `go vet ./...` is clean.

### 12.3 Items not addressed in this pass

Items 2, 6, and 7 from §11.11 required human review. They were discussed and decisions were made — see §12.4.

### 12.4 Deferred-item decisions

The following items were discussed with the human on 2026-03-19. Decisions are recorded here for future reference.

#### Item 6 — Decision `status` spec inconsistency: RESOLVED

**Decision:** Fix the spec. `status` was added to spec §9.5's minimum field list and to the P1-DEC-009 Decision field table in the decision log.

**Detail:** Spec §9.5 listed Decision minimum fields as `id`, `slug`, `summary`, `rationale`, `decided_by`, `date` — omitting `status`. But §10.5 defines lifecycle states (`proposed`, `accepted`, `rejected`, `superseded`) and P1-DEC-010 defines the entry state as `proposed` with explicit transition rules, all of which require a `status` field. Every other entity type listed `status` in its §9 minimum fields. The implementation has always set `status: "proposed"` at creation time. The omission was an oversight, not a design choice.

**Changes made:**
- `work/spec/phase-1-specification.md` §9.5 — added `status` to the minimum field list with a correction note explaining the change.
- `work/plan/phase-1-decision-log.md` P1-DEC-009 Decision table — added `status` as system-generated with default `proposed`, with an explanatory note.

#### Item 2 — Complete model integration on the read path: DEFERRED

**Decision:** Keep the current hybrid approach (typed writes, untyped reads) for Phase 1.

**Rationale:**

The write path constructs typed structs (`model.Epic{}`, etc.) and converts to `map[string]any` for storage. The read path returns raw `map[string]any` throughout — `GetResult.State` and `ListResult.State` are untyped maps.

Full integration would require:
- A deserialization layer (`map[string]any` → struct) for every entity type
- The service API to become type-specific (`GetEpic`, `GetFeature`, ...) or to return an interface
- Every consumer (CLI, future MCP layer) to handle typed structs instead of key-value maps

The current hybrid is pragmatic:
- Typed structs provide compile-time safety where mistakes are most likely — field construction during entity creation.
- Untyped maps provide flexibility where it's most useful — serialization and output formatting.
- The MCP layer (Layer 4) will need to decide its own response shapes. Forcing typed deserialization now would add a conversion layer that might get reworked when MCP arrives.
- No field-name bugs have been found on the read path, which suggests the current approach is adequate.

**Revisit trigger:** If field-name drift causes bugs on the read path, or when the MCP layer design is settled and the right return types become clear.

#### Item 7 — YAML escaping semantics: DEFERRED

**Decision:** Leave the current escaping behavior unchanged.

**Rationale:**

The `quoteString` function double-escapes backslashes: `\` → `\\`. This means a Windows path like `C:\temp` is stored as `"C:\\temp"` in canonical YAML. This is more aggressive than standard YAML requires (a bare `\` is legal in a double-quoted YAML string unless followed by a recognized escape character), but:

1. **The round-trip is correct.** `quoteString` and `parseScalar` agree — values survive write → read → write cycles unchanged, as confirmed by `TestMarshalCanonicalYAML_BackslashEscapingRoundTrip`.

2. **The tool is the canonical reader and writer.** P1-DEC-008 says "canonical workflow files are written by the workflow tool." Since the tool writes and reads its own format, internal consistency matters more than strict YAML spec compliance.

3. **Changing escaping would alter canonical output.** Any existing entity files with backslashes would produce different bytes on re-write, creating exactly the kind of meaningless churn that deterministic formatting is designed to prevent.

4. **The output is valid YAML.** Standard YAML parsers correctly read `\\` as `\`, so if interop is ever needed, the files are already compatible.

**Revisit trigger:** If a requirement emerges for external YAML tools to read or write canonical entity files, or if the custom YAML implementation is replaced with a standard library wrapper.