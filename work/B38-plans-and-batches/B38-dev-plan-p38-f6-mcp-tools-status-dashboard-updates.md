# P38-F6: MCP Tools and Status Dashboard Updates — Dev Plan

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T01:45:07Z           |
| Status | Draft                          |
| Author | architect                      |

---

## Scope

This dev-plan implements the specification `work/P38-plans-and-batches/P38-spec-p38-f6-mcp-tools-status-dashboard-updates.md` (P38-F6). It covers the MCP tool and status dashboard changes needed to support the plan entity, the renamed batch entity, and recursive progress rollup.

The plan covers:
- Entity tool: `type: "plan"` and `type: "batch"` CRUD, lifecycle enforcement, list with parent filter
- Status tool: project overview with top-level plans and standalone batches, plan dashboard with recursive progress and child-entity summary, batch dashboard with feature statuses and B{n}-F{n} display IDs
- Decompose, next, estimate, knowledge, doc tool updates for plan/batch acceptance
- Terminology sweep across all MCP response strings, error messages, commit messages, and side-effect type names

**Out of scope:** P38-F1 config schema (dependency — must already exist), P38-F2 plan entity lifecycle (dependency — must already exist), P38-F3 batch entity rename (dependency — must already exist), P38-F4 display IDs (dependency — must already exist), P38-F5 recursive rollup (dependency — must already exist), retroactive record migration (P38-F8).

---

## Task Breakdown

### Task 1: Entity Tool — Plan and Batch Type Support

- **Description:** Extend the entity tool to accept `type: "plan"` and `type: "batch"` for create, get, list, update, and transition operations. Wire plan operations to the plan lifecycle (P38-F2). Wire batch operations to batch CRUD (P38-F3 rename). Accept `type: "plan"` as a deprecated synonym for `type: "batch"` during transition. All entity tool responses for batch entities use "batch" terminology in display fields, messages, and commit messages.
- **Deliverable:** Updated `internal/mcp/entity_tool.go` with plan/batch type dispatch, `entityKindFromType`, `entityInferType` extended for batch prefix detection; updated service-layer calls for plan CRUD and batch CRUD; new and updated tests in `entity_tool_test.go`.
- **Depends on:** None (parallel with Task 2)
- **Effort:** Large
- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-007

### Task 2: Status Dashboard — Project Overview with Plans and Standalone Batches

- **Description:** Extend `status()` (no ID) project overview to render top-level plans with statuses and recursive progress, standalone batches (batches with no parent plan), and summary counts (total plans, batches, features, tasks). Update `synthesiseProject` in `status_tool.go`.
- **Deliverable:** Updated `internal/mcp/status_tool.go` — `synthesiseProject` function; new project-overview response shape fields; tests in `status_tool_test.go`.
- **Depends on:** None (parallel with Task 1)
- **Effort:** Medium
- **Spec requirements:** REQ-008

### Task 3: Status Dashboard — Plan Dashboard with Recursive Progress

- **Description:** Extend `status(id: "P{n}-slug")` to render a plan dashboard with the plan's name, status, summary, recursive progress from `ComputePlanRollup`, direct child plans with statuses, direct child batches with statuses, document references, and attention items. Add the child-entity summary string ("3 batches, 2 plans — 65% complete"). Update `synthesisePlan` and add `idTypePlan` handling in `inferIDType` (or add new `idTypePlanEntity` for the new plan entity type).
- **Deliverable:** Updated `internal/mcp/status_tool.go` — new `synthesisePlanEntity` function (or extended `synthesisePlan`), child-plan and child-batch rendering, recursive progress display, child-entity summary; tests in `status_tool_test.go`.
- **Depends on:** Task 2 (shared status_tool.go context)
- **Effort:** Large
- **Spec requirements:** REQ-009, REQ-012, REQ-013

### Task 4: Status Dashboard — Batch Dashboard

- **Description:** Extend `status(id: "B{n}-slug")` to render a batch dashboard equivalent to the pre-P38 plan dashboard: batch name, status, summary, progress from `ComputeBatchRollup`, features belonging to the batch with statuses, tasks ready/active/done counts per feature, document gaps, attention items. Feature display IDs use `B{n}-F{n}` format. Add `idTypeBatch` to `inferIDType`.
- **Deliverable:** Updated `internal/mcp/status_tool.go` — new `synthesiseBatch` function, `idTypeBatch` constant, batch dashboard response shape, B{n}-F{n} display ID rendering; tests in `status_tool_test.go`.
- **Depends on:** Task 2, Task 3 (shared status_tool.go, shared rendering patterns)
- **Effort:** Medium
- **Spec requirements:** REQ-010, REQ-011, REQ-012 (partial)

### Task 5: Decompose, Next, Estimate, Knowledge, and Doc Tool Updates

- **Description:** Update all five tools to accept batch/plan IDs where appropriate:
  - **Decompose** (`decompose_tool.go`): Accept batch ID as `feature_id` parent for features owned by batches.
  - **Next** (`next_tool.go`): Accept batch IDs; context packets use "batch" terminology. Update `nextInferEntityType` for batch prefix.
  - **Estimate** (`estimate_tool.go`): Dispatch batch queries to `ComputeBatchRollup` and plan queries to `ComputePlanRollup`. Update `resolveEntityType` for batch.
  - **Knowledge** (`knowledge_tool.go`): Accept batch ID in `scope` parameter for knowledge entry scoping.
  - **Doc** (`doc_tool.go`): Accept batch ID as `owner` for document registration/listing/gaps.
- **Deliverable:** Updated `decompose_tool.go`, `next_tool.go`, `estimate_tool.go`, `knowledge_tool.go`, `doc_tool.go`; associated test files.
- **Depends on:** Task 1 (entity tool batch/plan ID resolution is the foundation for all tool ID handling)
- **Effort:** Medium
- **Spec requirements:** REQ-014, REQ-015, REQ-016, REQ-017, REQ-018

### Task 6: Terminology Consistency Sweep

- **Description:** Audit and correct all MCP tool response messages, error strings, commit messages, and side-effect type names for plan/batch terminology. Rename `SideEffectPlanAutoAdvanced` to `SideEffectBatchAutoAdvanced` and add new `SideEffectPlanAutoAdvanced` for plan auto-advance. Verify that no tool response uses "plan" to mean "batch" (except the deprecated `type: "plan"` synonym in entity tool dispatch). Includes project-wide grep to confirm zero false "plan" references to batch entities.
- **Deliverable:** Updated `sideeffect.go` (side-effect type constants and any producer code), corrected strings across all MCP tool files; project-wide terminology audit report.
- **Depends on:** Task 1, Task 2, Task 3, Task 4, Task 5 (all tools must have their logic in place before the sweep)
- **Effort:** Small
- **Spec requirements:** REQ-019, REQ-020

### Task 7: Integration and Compatibility Verification

- **Description:** End-to-end integration tests verifying acceptance criteria AC-001 through AC-020. Key focus areas: plan create/transition lifecycle, batch create with both type strings, status dashboard output shapes for all three scopes, B{n}-F{n} display IDs, recursive progress values from rollup functions, backward compatibility (existing client calling `status(batch-id)` receives familiar fields at expected paths), and terminology correctness.
- **Deliverable:** New and extended tests in `internal/mcp/status_tool_test.go`, `entity_tool_test.go`, `estimate_tool_test.go`, `knowledge_tool_test.go`, `doc_tool_test.go`, and `next_tool_test.go`; new `integration_test.go` entries if needed.
- **Depends on:** Task 6 (terminology sweep must be complete to verify AC-018 and AC-019)
- **Effort:** Large
- **Spec requirements:** AC-001 through AC-020

---

## Dependency Graph

```
Task 1 (no dependencies)          Task 2 (no dependencies)
    │                                  │
    │                                  ├── Task 3 (depends on Task 2)
    │                                  │       │
    │                                  │       ├── Task 4 (depends on Task 2, Task 3)
    │                                  │       │
    ├── Task 5 (depends on Task 1) ────┘       │
    │                                  │       │
    └── Task 6 (depends on all) ←─────┴───────┘
            │
            └── Task 7 (depends on Task 6)
```

**Parallel groups:** [Task 1, Task 2], [Task 3, Task 5] (after their respective dependencies)
**Critical path:** Task 2 → Task 3 → Task 4 → Task 6 → Task 7

---

## Risk Assessment

### Risk: ID Type Collision Between Old Plan (Batch) and New Plan Entity

- **Probability:** Medium
- **Impact:** High — Ambiguous ID resolution could route batch IDs to plan handlers or vice versa, breaking all tool dispatch.
- **Mitigation:** P38-F1 config schema provides prefix registries that unambiguously distinguish batch prefixes (single letter + digit) from plan prefixes. The entity tool resolves type from the prefix registry before dispatching. Task 1 must coordinate with P38-F1's prefix registry design.
- **Affected tasks:** Task 1, Task 3, Task 4, Task 5

### Risk: Status Dashboard Response Shape Breakage

- **Probability:** Low
- **Impact:** High — Existing MCP clients calling `status(batch-id)` after P38-F3 rename could receive broken responses if field paths change.
- **Mitigation:** AC-020 explicitly requires backward compatibility. The batch dashboard response shape is the pre-P38 plan dashboard shape with fields at existing paths. Task 7 verifies this directly. Any new fields are additive only.
- **Affected tasks:** Task 4, Task 7

### Risk: Stale Plan/Batch Typing Across Dependent Features

- **Probability:** Medium
- **Impact:** Medium — If P38-F1, P38-F2, P38-F3, or P38-F5 are incomplete or change after this feature starts, tool code may reference functions or types that don't yet exist or have changed signatures.
- **Mitigation:** The dependency chain enforces that P38-F1 through P38-F5 are complete before this feature. Interface contracts are defined in the specification (see §Dependencies). Task 1 through Task 5 reference only the stable service-layer APIs exposed by those features.
- **Affected tasks:** Task 1, Task 3, Task 4, Task 5

### Risk: Terminology Sweep Misses Dynamic Strings

- **Probability:** Medium
- **Impact:** Low — Some "plan" → "batch" terminology may be generated dynamically and missed by a static grep audit.
- **Mitigation:** Task 6 includes both static grep and runtime inspection via tests. Task 7 (AC-018) runs a fresh grep after all changes. Any remaining dynamic strings would only affect log/cli output, not MCP protocol responses.
- **Affected tasks:** Task 6, Task 7

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-001: plan create returns plan ID and status `idea` | Unit test | Task 1 |
| AC-002: batch create via both type strings | Unit test | Task 1 |
| AC-003: plan create response shape | Unit test | Task 1 |
| AC-004: plan lifecycle transitions | Unit test | Task 1 |
| AC-005: list plans/batches with parent filter | Unit test | Task 1 |
| AC-006: batch commit messages and display fields | Code review | Task 6 |
| AC-007: status() project overview | Unit test | Task 2 |
| AC-008: status(P1-...) plan dashboard | Unit test | Task 3 |
| AC-009: status(B24-...) batch dashboard | Unit test | Task 4 |
| AC-010: B{n}-F{n} feature display IDs | Unit test | Task 4 |
| AC-011: rollup function dispatch in status rendering | Unit test | Task 3, Task 4 |
| AC-012: child-entity summary string | Unit test | Task 3 |
| AC-013: decompose under batch-owned feature | Unit test | Task 5 |
| AC-014: next() context packet terminology | Unit test | Task 5 |
| AC-015: estimate query dispatch per entity type | Unit test | Task 5 |
| AC-016: knowledge entry scope round-trip | Unit test | Task 5 |
| AC-017: doc registration under batch owner | Unit test | Task 5 |
| AC-018: zero false "plan" references to batch | Static grep + code review | Task 6, Task 7 |
| AC-019: auto-advance side effect types | Unit test | Task 6 |
| AC-020: existing client compatibility | Integration test | Task 7 |
