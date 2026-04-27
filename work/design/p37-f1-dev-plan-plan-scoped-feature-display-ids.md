# P37-F1: Plan-scoped Feature Display IDs — Implementation Plan

| Field  | Value                                                      |
|--------|------------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                       |
| Status | Draft                                                      |
| Author | orchestrator                                               |
| Spec   | work/design/p37-f1-spec-plan-scoped-feature-display-ids.md |

---

## Scope

This plan covers the full implementation of plan-scoped feature display IDs as specified in
`work/design/p37-f1-spec-plan-scoped-feature-display-ids.md` (FEAT-01KQ7JDSVMP4E).

The work adds a `next_feature_seq` counter to the Plan model and a `display_id` field to the
Feature model, wires display ID allocation into `CreateFeature`, adds `P{n}-F{m}` input
resolution across all entity operations, updates CLI and MCP outputs to surface display IDs as
primary human-facing identifiers, and backfills existing features via a migration. Canonical
`FEAT-{TSID13}` identifiers are preserved for all storage filenames and cross-references
throughout.

Out of scope: changes to state filenames, distributed conflict resolution beyond git merge
semantics, and any UI beyond the CLI and MCP tool layer.

---

## Task Breakdown

### T1 — Extend Plan and Feature models

**Description:**
Add `NextFeatureSeq int` (yaml: `next_feature_seq`) to `model.Plan` in
`internal/model/entities.go`. Add `DisplayID string` (yaml: `display_id,omitempty`) to
`model.Feature`. Update `planFields()` in `internal/service/plans.go` to always serialise
`next_feature_seq` (even when zero, to make the field visible in newly-created plan files).
Update `featureFields()` in `internal/service/entities.go` to serialise `display_id` when
non-empty. Update `CreatePlan` in `internal/service/plans.go` to initialise
`entity.NextFeatureSeq = 1` before writing the plan file.

**Deliverable:**
`model.Plan` and `model.Feature` compile with the new fields. A newly-created plan YAML
contains `next_feature_seq: 1`. A feature YAML round-trips `display_id` correctly.

**Depends on:** nothing

**Effort:** Small (2–3 h)

**Spec requirements:** REQ-001, REQ-002, REQ-004

---

### T2 — Allocate display_id in CreateFeature

**Description:**
Update `CreateFeature` in `internal/service/entities.go` to perform the four-step allocation
sequence from REQ-008:

1. Load the parent plan state via `loadPlan`.
2. Read `next_feature_seq` from the plan's state map (default to 1 if absent, for plans
   created before T1 landed).
3. Extract the numeric component of the plan ID using `model.ParsePlanID` and format
   `display_id` as `P{n}-F{seq}`.
4. Write the plan state with `next_feature_seq` incremented to seq+1.
5. If step 4 fails, return an error immediately without writing the feature file.
6. Write the feature file with `display_id` set.

The `parent` field is already required by `validateRequired`; add a clear error message that
names the missing plan as the cause if the plan entity is not found (REQ-007). `CreateFeature`
already validates that the parent entity exists; the error message improvement is the only
change to the validation path.

**Deliverable:**
After `CreateFeature`, the plan file shows `next_feature_seq` incremented by 1 and the feature
file contains `display_id: P{n}-F{m}`. A crash injected between the plan write and the feature
write leaves a gap in the sequence but no duplicate display_id.

**Depends on:** T1

**Effort:** Medium (4–5 h)

**Spec requirements:** REQ-003, REQ-005, REQ-006, REQ-007, REQ-008

---

### T3 — Add P{n}-F{m} input resolution

**Description:**
Add a helper `IsFeatureDisplayID(s string) bool` and
`(s *EntityService) ResolveFeatureDisplayID(displayID string) (resolvedID, resolvedSlug string, err error)`
in `internal/service/entities.go`. `IsFeatureDisplayID` matches the case-insensitive pattern
`P\d+-F\d+` using a compiled `regexp.MustCompile`. `ResolveFeatureDisplayID` normalises the
input to uppercase, then scans the features directory for a YAML file whose `display_id` field
matches. When the entity cache is warm, a secondary display_id index (map[string]string built
during `RebuildCache`) is used to avoid disk I/O (satisfies the 100 ms SLA in REQ-NF-001).

Hook the resolution into three callsites, each immediately before their `ResolvePrefix` call:
- `EntityService.Get` — when entityType is `"feature"` and the input matches the display ID
  pattern, resolve via `ResolveFeatureDisplayID` and replace `entityID` and `slug`.
- `EntityService.UpdateStatus` — same guard.
- `EntityService.UpdateEntity` — same guard.

Also update `EntityService.List` when a caller-supplied ID filter matches the display ID
pattern: resolve it first, then apply the existing filter logic.

`id.NormalizeID` in `internal/id/display.go` already passes plan-style IDs (`P{n}-…`) through
unchanged, so no changes are needed there for the input normalisation path.

**Deliverable:**
`entity get P37-F1`, `entity update P37-F2`, `entity transition P37-F3`, and `entity list`
with display ID filter all resolve correctly. Lowercase `p37-f1` resolves identically to
`P37-F1`. Canonical TSID and break-hyphen TSID inputs continue to work unchanged.

**Depends on:** T1

**Effort:** Medium (4–6 h)

**Spec requirements:** REQ-009, REQ-010, REQ-011, REQ-NF-001, REQ-NF-003, REQ-NF-004

---

### T4 — Update CLI output and MCP responses

**Description:**
**CLI** (`cmd/kanbanzai/entity_cmd.go`): Update `printGetResult`, `printListResults`,
`printCreateResult`, and `printStatusUpdateResult` to check for a `display_id` key in the
result state map. When present and non-empty, print it in place of `id.FormatFullDisplay(result.ID)`
as the primary identifier line. The TSID-derived form may still appear on a secondary `tsid:`
line for diagnostic purposes but must not be the only identifier shown.

**MCP** (`internal/mcp/entity_tool.go` and related handlers): `featureFields()` in
`internal/service/entities.go` will already include `display_id` in the state map once T1 is
complete, so MCP responses that return the raw state map (the common pattern) gain the field
automatically. Verify that the entity tool's `get`, `list`, `create`, and `transition` response
builders do not strip unknown fields from the state map before returning. If any builder
explicitly constructs a whitelist response struct, add `display_id` to it.

**Deliverable:**
CLI `entity get` for a feature with a display_id prints `P{n}-F{m}` as the identifier. MCP
entity tool responses for features include a `display_id` field in the JSON payload.

**Depends on:** T3

**Effort:** Small (2–3 h)

**Spec requirements:** REQ-012, REQ-013

---

### T5 — Migration: backfill display_ids on existing features

**Description:**
Implement the backfill migration in `internal/service/migration.go` (currently empty). Add:

```
func (s *EntityService) MigrateDisplayIDs() error
```

Algorithm:
1. List all plans via `listPlanIDs()`.
2. For each plan, call `List("feature")` and filter to features whose `parent` field matches
   the plan ID and whose `display_id` field is absent or empty.
3. Sort the filtered features by `created` timestamp (ascending).
4. For each feature in sorted order, assign `display_id: P{n}-F{seq}` (seq starting at 1 for
   the first feature without a display_id under this plan) and write the updated feature file.
   Use `s.store.Write` directly with the mutated state map.
5. After processing all features for a plan, read the plan file, set `next_feature_seq` to
   (number of features assigned) + 1, and write it back. Features that already had a
   `display_id` before the migration began are counted toward the sequence floor: find the
   highest existing seq number under the plan and set `next_feature_seq` to max(existing_max,
   newly_assigned_count) + 1.
6. Do not rename or move any state files (REQ-NF-005).

Wire `MigrateDisplayIDs` into the existing migration runner (whichever function the server
calls at startup to run outstanding migrations), guarded by a version check or a "migration
already applied" flag consistent with the project's migration convention. Inspect the existing
(empty) `migration.go` and the server startup path to determine the correct wiring point.

**Deliverable:**
Running the migration against a repository with existing features assigns `display_id` values
in creation-timestamp order within each plan. Plan `next_feature_seq` values are set correctly.
No filenames change.

**Depends on:** T1, T2

**Effort:** Medium (4–6 h)

**Spec requirements:** REQ-014, REQ-015

---

### T6 — Tests

**Description:**
Write tests covering all 20 acceptance criteria. Tests live alongside the code under test using
Go's standard `_test.go` convention. New test helpers (fixtures, fake stores) should be added
to existing test helper files rather than new files where possible.

Specific test cases required:

| Test | AC | Method | Location |
|------|----|--------|----------|
| CreatePlan sets next_feature_seq: 1 | AC-001 | Unit | `internal/service/plans_test.go` |
| CreateFeature increments counter | AC-002 | Unit | `internal/service/entities_test.go` |
| display_id format P37-F3 | AC-003 | Unit | `internal/service/entities_test.go` |
| Crash between plan write and feature write leaves gap | AC-004 | Unit | `internal/service/entities_test.go` |
| CreateFeature with no parent returns error, no file written | AC-005 | Unit | `internal/service/entities_test.go` |
| Both writes observable before CreateFeature returns | AC-006 | Unit | `internal/service/entities_test.go` |
| entity get P24-F3 == entity get FEAT-01KMKRQRRX3CC | AC-007 | Integration | `internal/service/entities_test.go` |
| entity get p24-f3 (lowercase) == entity get P24-F3 | AC-008 | Integration | `internal/service/entities_test.go` |
| entity get by display_id returns full state | AC-009 | Integration | `internal/service/entities_test.go` |
| entity update by display_id updates state on disk | AC-010 | Integration | `internal/service/entities_test.go` |
| entity transition by display_id updates status | AC-011 | Integration | `internal/service/entities_test.go` |
| entity list with display_id filter returns one result | AC-012 | Integration | `internal/service/entities_test.go` |
| MCP entity tool response includes display_id field | AC-013 | Integration | `internal/mcp/entity_tool_test.go` |
| CLI entity get shows P{n}-F{m} as primary identifier | AC-014 | Integration | `cmd/kanbanzai/entity_cmd_test.go` |
| Migration assigns display_ids in timestamp order | AC-015 | Migration | `internal/service/migration_test.go` |
| Migration sets next_feature_seq to count+1 | AC-016 | Migration | `internal/service/migration_test.go` |
| Resolution within 100 ms for 1000-feature fixture | AC-017 | Perf | `internal/service/entities_test.go` |
| Canonical TSID input still works | AC-018 | Integration | `internal/service/entities_test.go` |
| Break-hyphen TSID input still works | AC-019 | Integration | `internal/service/entities_test.go` |
| No filenames changed after migration | AC-020 | Migration | `internal/service/migration_test.go` |

For AC-004, inject a fault by passing a test-double `store` whose `Write` succeeds on the
first call (plan) and returns an error on the second (feature), then assert no feature file
with the expected display_id exists on disk.

For AC-017, generate a fixture with 1000 features spread across 10 plans using a test helper;
measure wall-clock time for `ResolveFeatureDisplayID` with the cache cold and with it warm.

**Depends on:** T1, T2, T3, T4, T5

**Effort:** Large (8–10 h)

**Spec requirements:** all 20 ACs (AC-001 through AC-020)

---

## Dependency Graph

```
T1 (models)
├── T2 (CreateFeature allocation)     depends on T1
├── T3 (P{n}-F{m} resolution)         depends on T1
│   └── T4 (CLI + MCP output)         depends on T3
└── T5 (migration backfill)           depends on T1, T2
    └── T6 (tests)                    depends on T1, T2, T3, T4, T5
```

Execution order for a single developer: T1 → T2 and T3 in parallel → T4 after T3 → T5 after
T2 → T6 after all.

---

## Risk Assessment

### RISK-1 — Write ordering crash leaves orphaned counter advance

If the process crashes after writing the incremented plan counter (step 4 of REQ-008) but
before writing the feature file (step 5), the sequence number is consumed but no feature
holds it. The spec accepts gaps; the risk is that a confused user notices the gap and tries to
"fix" it by manually editing the counter, potentially causing a duplicate. Mitigation: AC-004
explicitly tests this path; add a comment in `CreateFeature` and in the operator documentation
explaining that gaps are intentional and the counter must never be decremented.

### RISK-2 — Migration safety on large repositories

`MigrateDisplayIDs` reads and writes every feature file and every plan file in the repository.
On a repository with hundreds of features the migration could take several seconds and leave
partial state if interrupted mid-run. Mitigation: write features before updating the plan
counter for each plan (so a partial run is re-runnable: features already assigned are skipped,
and the counter is only updated once all features under a plan are done). Add a dry-run flag
(`--dry-run`) to `MigrateDisplayIDs` for operator verification before applying. AC-020 verifies
filename stability.

### RISK-3 — Display ID cache index invalidation

The performance SLA (AC-017, 100 ms for 1000 features) requires a warm cache. If the cache is
stale (e.g. features added by another process since last `RebuildCache`) the display_id index
could miss entries and fall back to a full directory scan, adding latency. Mitigation: the
fallback scan is always correct; document that the cache must be rebuilt after out-of-band
writes. In the longer term, a write-through hook in `cacheUpsertFromResult` should update the
display_id index when a feature is created or updated. Implement the write-through hook in T3
to keep the index consistent for the duration of a server session.

---

## Verification Approach

All 20 acceptance criteria from the specification are mapped to tasks and test cases in the
T6 task table above. The mapping is reproduced here for traceability:

| AC   | Requirement(s)        | Task(s)    | Test method        |
|------|-----------------------|------------|--------------------|
| AC-001 | REQ-001, REQ-002    | T1         | Unit               |
| AC-002 | REQ-003             | T2         | Unit               |
| AC-003 | REQ-004, REQ-005    | T2         | Unit               |
| AC-004 | REQ-006, REQ-NF-002 | T2         | Unit (fault inject)|
| AC-005 | REQ-007             | T2         | Unit               |
| AC-006 | REQ-008             | T2         | Unit               |
| AC-007 | REQ-009             | T3         | Integration        |
| AC-008 | REQ-010             | T3         | Integration        |
| AC-009 | REQ-011 (get)       | T3         | Integration        |
| AC-010 | REQ-011 (update)    | T3         | Integration        |
| AC-011 | REQ-011 (transition)| T3         | Integration        |
| AC-012 | REQ-011 (list)      | T3         | Integration        |
| AC-013 | REQ-012             | T4         | Integration        |
| AC-014 | REQ-013             | T4         | Integration        |
| AC-015 | REQ-014             | T5         | Migration          |
| AC-016 | REQ-015             | T5         | Migration          |
| AC-017 | REQ-NF-001          | T3         | Performance        |
| AC-018 | REQ-NF-003          | T3         | Integration        |
| AC-019 | REQ-NF-004          | T3         | Integration        |
| AC-020 | REQ-NF-005          | T5         | Migration          |