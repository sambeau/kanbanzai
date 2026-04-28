| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T01:44:37Z           |
| Status | Draft                          |
| Author | architect                      |

## Scope

This plan implements the specification defined in
`work/P38-plans-and-batches/P38-spec-p38-f2-plan-entity-data-model-lifecycle.md`
(DOC `FEAT-01KQ7YQKBDNAP/spec-p38-spec-p38-f2-plan-entity-data-model-lifecycle`).
It covers the new recursive `Plan` struct with nesting (`parent`, `order`) and
inter-plan dependency fields (`depends_on`), the planning-oriented lifecycle
(`idea` → `shaping` → `ready` → `active` → `done`), cycle detection, and CRUD
operations wired through the entity service and MCP entity tool (REQ-001 through
REQ-022, AC-001 through AC-017).

This plan does **not** cover: the batch entity rename (P38-F3), feature display
ID updates (P38-F4), recursive progress rollup (P38-F5), status dashboard plan
trees (P38-F6), `depends_on` enforcement as lifecycle gates (deferred), document
gate inheritance (P38-F4), or migration of existing state files (P38-F8).

## Task Breakdown

### Task 1: Define New Plan Data Model and PlanStatus Constants

- **Description:** Introduce the new `Plan` struct in `internal/model/entities.go`
  alongside the existing `Plan` struct (which becomes `Batch` in P38-F3). Define the
  `PlanStatus` type with seven distinct lifecycle constants (`idea`, `shaping`,
  `ready`, `active`, `done`, `superseded`, `cancelled`). Register the new entity
  kind `EntityKindPlan` (or reuse/scope the existing one if the rename has not yet
  occurred — this task clarifies the approach). All fields match REQ-001 through
  REQ-005: `id`, `slug`, `name`, `status`, `summary`, `parent`, `design`,
  `depends_on`, `order`, `tags`, `created`, `created_by`, `updated`, `supersedes`,
  `superseded_by`. The struct must exclude `next_feature_seq`.
- **Deliverable:** Modified `internal/model/entities.go` with the new `Plan` struct,
  `PlanStatus` type, and accompanying constants. Both old and new plan structs
  compile side by side.
- **Depends on:** None within this feature. Externally depends on P38-F1 for
  `plan_prefixes` config registry (used by Task 3 for ID generation).
- **Effort:** Medium.
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006,
  REQ-008, AC-001, AC-002, AC-003, AC-006.

### Task 2: Implement Plan Lifecycle Validation

- **Description:** Add the new plan lifecycle transition rules to
  `internal/validate/lifecycle.go`. Register entry state (`idea`) and terminal
  states (`superseded`, `cancelled`) in the existing validation tables. Define the
  allowed transitions per REQ-010 and rejection of all other transitions per
  REQ-012. Implement atomic validation — reject before any mutation occurs
  (REQ-NF-003). The new plan lifecycle is an independent implementation that does
  not modify the existing plan lifecycle code (which becomes the batch lifecycle).
- **Deliverable:** Modified `internal/validate/lifecycle.go` with new plan lifecycle
  tables and transition validation. Unit tests for the new lifecycle rules baked
  into `internal/validate/lifecycle_test.go`.
- **Depends on:** Task 1 (PlanStatus constants must be defined).
- **Effort:** Medium.
- **Spec requirement:** REQ-008, REQ-009, REQ-010, REQ-011, REQ-012, REQ-NF-003,
  AC-006, AC-007, AC-008, AC-009, AC-017.

### Task 3: Implement Plan CRUD in Entity Service

- **Description:** Add `CreatePlan`, `GetPlan`, `UpdatePlan`, `UpdatePlanStatus`,
  and `ListPlans` methods to the entity service (`internal/service/entities.go`).
  `CreatePlan` resolves the plan prefix via `plan_prefixes` config registry and
  sequence counter (from P38-F1). `CreatePlan` and `UpdatePlan` validate parent
  existence (REQ-014) and detect cycles in the parent chain (REQ-015) — walk the
  ancestor chain upward in O(depth) time (REQ-NF-004). `UpdatePlanStatus` delegates
  to the lifecycle validator from Task 2 and includes auto-advance support (REQ-013)
  following the existing `entityTransitionAction` pattern. State files are written
  to `.kbz/state/plans/{id}.yaml` using existing YAML conventions (REQ-007,
  REQ-NF-002). The existing plan CRUD must continue functioning unchanged
  (REQ-NF-001, AC-016).
- **Deliverable:** Modified `internal/service/entities.go` with plan CRUD methods
  and cycle detection logic.
- **Depends on:** Task 1 (Plan struct), Task 2 (lifecycle validation).
- **Effort:** Large.
- **Spec requirement:** REQ-007, REQ-013, REQ-014, REQ-015, REQ-016, REQ-017,
  REQ-018, REQ-019, REQ-020, REQ-021, REQ-NF-001, REQ-NF-002, REQ-NF-004,
  AC-004, AC-005, AC-010, AC-011, AC-012, AC-016.

### Task 4: Wire Plan Type into Entity MCP Tool

- **Description:** Extend `internal/mcp/entity_tool.go` to handle `type: "plan"`
  for create, get, list, update, and transition actions. Plan type is inferred from
  the `P{n}` prefix pattern (existing `IsPlanID` function). The create action accepts
  parent, slug, name, summary, design, depends_on, order, and tags. The transition
  action validates against the new plan lifecycle and reports side effects
  (auto-advance signals). List supports optional parent filter for child-plan queries.
  Document expectations (REQ-023) are returned as guidance metadata in tool responses
  but not enforced.
- **Deliverable:** Modified `internal/mcp/entity_tool.go` with plan type routing
  wired to the entity service methods from Task 3.
- **Depends on:** Task 3 (CRUD operations).
- **Effort:** Medium.
- **Spec requirement:** REQ-022, REQ-023, AC-013, AC-014, AC-015.

### Task 5: Comprehensive Unit and Integration Tests

- **Description:** Write tests covering all acceptance criteria not already covered
  by unit tests in Tasks 2–4. Includes: YAML marshal/unmarshal round-trip for the
  new Plan struct, all valid and invalid lifecycle transitions, parent-reference
  validation (nonexistent parent, self-reference, 3-level cycle), deep nesting
  (depth 5), prefix-based ID assignment, entity tool round-trips, and coexistence
  with existing batch (current plan) operations.
- **Deliverable:** Test files:
  `internal/model/entities_test.go` (data model),
  `internal/service/entities_test.go` (CRUD and nesting),
  `internal/mcp/entity_tool_test.go` (MCP integration).
- **Depends on:** Task 2 (lifecycle tests extend validation tests), Task 3 (CRUD
  tests), Task 4 (MCP integration tests). Can begin in parallel with Task 4 once
  Task 3 is complete.
- **Effort:** Medium.
- **Spec requirement:** AC-001 through AC-017 (all acceptance criteria).

## Dependency Graph

    Task 1 (no internal dependencies; external: P38-F1 for prefix registry)
    Task 2 → depends on Task 1
    Task 3 → depends on Task 1, Task 2
    Task 4 → depends on Task 3
    Task 5 → depends on Task 2, Task 3, Task 4

    Parallel groups: [Task 1] → [Task 2] → [Task 3] → [Task 4, Task 5*]
    * Task 5 can begin once Task 3 is complete, parallel with Task 4.

    Critical path: Task 1 → Task 2 → Task 3 → Task 4
    (Task 5 is not on the critical path if it runs concurrently with Task 4)

## Risk Assessment

### Risk: Namespace Collision with Existing Plan Struct

- **Probability:** Medium.
- **Impact:** High — compilation failures across all packages that reference
  `model.Plan`.
- **Mitigation:** The new plan struct must use a distinct type name (e.g. the
  new struct is `Plan` and the existing is renamed to `Batch` in P38-F3; during
  the transition both coexist under different names). If the rename hasn't
  happened yet, use a temporary disambiguated name or coordinate with P38-F3
  sequencing. Verify compilation of full codebase after Task 1.
- **Affected tasks:** Task 1, Task 3.

### Risk: P38-F1 Plan Prefix Registry Not Ready

- **Probability:** Low.
- **Impact:** High — `CreatePlan` cannot resolve plan prefixes or sequence numbers.
- **Mitigation:** Task 3's `CreatePlan` depends on the `plan_prefixes` registry
  from P38-F1. If F1 is incomplete, stub the prefix resolution with the existing
  `Config.Prefixes` and `NextPlanNumber` which are already functional (the existing
  plan already uses them). The F1 dependency is for the *independent* sequence
  counter; the existing sequential counter suffices as a fallback.
- **Affected tasks:** Task 3.

### Risk: Cycle Detection Edge Cases at Depth

- **Probability:** Low.
- **Impact:** Medium — a cycle at depth 10+ could cause performance issues or
  stack overflow if implemented recursively.
- **Mitigation:** Implement iteration (not recursion) for the ancestor walk.
  Benchmark with depth 100 to confirm O(depth) performance. Add a test for
  deep-but-acyclic chains.
- **Affected tasks:** Task 3, Task 5.

### Risk: Existing Plan Operations Break After Changes

- **Probability:** Low.
- **Impact:** High — blocking regression on all existing workflow commands.
- **Mitigation:** The new plan struct and lifecycle are additive — they do not
  modify the existing `Plan` struct or its lifecycle tables. The existing plan
  type continues to use the old `Plan` struct and old lifecycle constants until
  P38-F3 renames it. Run the full existing test suite after Tasks 1–4 to catch
  regressions. AC-016 explicitly tests this.
- **Affected tasks:** Task 1, Task 3, Task 5.

## Verification Approach

| Acceptance Criterion | Method | Producing Task |
|---|---|---|
| AC-001: Plan struct exists with all fields, YAML-serialisable | Unit test | Task 5 |
| AC-002: Plan created with optional parent + order fields | Unit test | Task 5 |
| AC-003: Plan struct has no `next_feature_seq` | Compile-time check | Task 1 |
| AC-004: Plan ID matches `P{n}-{slug}` pattern | Unit test | Task 5 |
| AC-005: Plan state file at `.kbz/state/plans/{id}.yaml` | Unit test | Task 5 |
| AC-006: All 7 status constants defined, new plan starts `idea` | Unit test | Task 2 |
| AC-007: All valid forward/backward transitions succeed | Unit test | Task 2 |
| AC-008: Terminal transitions from all non-terminal states | Unit test | Task 2 |
| AC-009: Invalid transitions fail with descriptive error | Unit test | Task 2 |
| AC-010: Nonexistent parent rejected on create | Unit test | Task 5 |
| AC-011: Cycle detection rejects P1→P2→P3→P1 | Unit test | Task 5 |
| AC-012: Depth-5 plan tree without errors | Unit test | Task 5 |
| AC-013: MCP entity create for plan type | Integration test | Task 4 |
| AC-014: MCP entity get + transition for plan type | Integration test | Task 4 |
| AC-015: MCP entity list with parent filter | Integration test | Task 4 |
| AC-016: Existing batch CRUD works after plan changes | Regression test | Task 5 |
| AC-017: Invalid transition does not mutate state on disk | Unit test | Task 2 |
