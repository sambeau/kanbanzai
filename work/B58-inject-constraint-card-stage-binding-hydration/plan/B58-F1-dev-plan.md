| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:49:22Z |
| Status | Draft |
| Author | architect |
| Feature | FEAT-01KR3MDJ7AV37 — Constraint card and stage-binding hydration |
| Batch  | B58-inject-constraint-card-stage-binding-hydration |
| Spec   | `work/B58-inject-constraint-card-stage-binding-hydration/B58-F1-spec-constraint-card-stage-binding-hydration.md` |

---

## Scope

This plan implements the requirements defined in the approved spec
`work/B58-inject-constraint-card-stage-binding-hydration/B58-F1-spec-constraint-card-stage-binding-hydration.md`
(FEAT-01KR3MDJ7AV37). It covers six tasks (T1–T6).

**In scope:**
- A typed constraint registry (`internal/card` package) with a loadable YAML file at `.kbz/constraints.yaml`.
- A constraint card renderer that composes a ≤25-line, ≤2500-byte card from role, stage binding, and constraint registry inputs.
- A stage-binding hydration payload extractor that serialises key binding fields.
- Injection of the rendered card and hydration payload into `next` (claim mode) responses.
- Injection of the rendered card and hydration payload into `handoff` responses.
- Golden tests for developing, specifying, dev-planning, and reviewing stage bindings; validation tests; and regression tests for existing response fields.

**Out of scope** (matches spec):
- Rewriting skill markdown or role YAML prose.
- Adding the card to informational responses (`status`, entity `get`).
- P44 provider dispatch, B59 MCP tool invariants, B60 registry tables, B62 runtime discovery wrappers.
- Changes to the `binding` package model — the existing `StageBinding` type is used as-is.

**Interface boundary note:** The `next` tool uses the `assembleContext` path in `internal/mcp/assembly.go`.
The `handoff` tool uses `internal/context/Pipeline.Run()`. The injection points are therefore in
`nextClaimMode` (next_tool.go) and the `handoffTool` handler (handoff_tool.go), not inside the
pipeline internals. This preserves the pipeline's single-responsibility boundary.

---

## Task Breakdown

### Task 1: Constraint Registry — type model, loader, and initial YAML

- **Description:** Define the `ConstraintEntry` type (ID, rule statement, role applicability list,
  stage applicability list) and the `ConstraintRegistry` loader in a new `internal/card` package.
  Create `.kbz/constraints.yaml` with constraint entries that cover the four required stages
  (developing, specifying, dev-planning, reviewing). Include registry validation that returns an
  actionable error naming any missing required field, satisfying REQ-007 at the registry layer.
  Deterministic ordering (REQ-NF-004) is guaranteed by sorting entries on load.
- **Deliverable:**
  - `internal/card/constraint_registry.go` — `ConstraintEntry` type, `ConstraintRegistry` struct,
    `LoadConstraintRegistry(path string) (*ConstraintRegistry, error)` loader, `Select(role, stage string) []ConstraintEntry` method.
  - `.kbz/constraints.yaml` — initial constraint entries for at least the 4 required stage/role pairs,
    each with a stable `id`, a `rule` statement, and `applies_to` (roles, stages) metadata.
  - `internal/card/constraint_registry_test.go` — unit tests for loading, validation errors, and
    `Select` determinism.
- **Depends on:** None
- **Effort:** 3 points
- **Spec requirements:** REQ-001, REQ-NF-004, AC-001 (partial — registry inputs exist)

---

### Task 2: Constraint Card Renderer

- **Description:** Implement a renderer in `internal/card` that produces a compact markdown card
  from three typed inputs: a `*context.ResolvedRole`, a `*binding.StageBinding` (plus stage name),
  and a `[]ConstraintEntry` slice (pre-selected by the registry). The card must include:
  role identity, resolved stage, bound skill names, top operational constraints, and a tool-routing
  reminder. Validate inputs and fail with an actionable error naming any missing required field
  (REQ-007). For an unknown stage, emit an explicit `UNKNOWN STAGE` warning and the manual-load
  instruction (REQ-008). Enforce ≤25 non-empty lines and ≤2500 bytes (REQ-NF-001, REQ-NF-002).
  The renderer must not read SKILL.md files or arbitrary prose — only typed inputs (spec constraint).
- **Deliverable:**
  - `internal/card/renderer.go` — `Renderer` type, `Render(role, binding, stage, entries) (string, error)` function.
  - `internal/card/renderer_test.go` — unit tests covering: normal render, missing role identity
    error, unknown stage warning, line count enforcement, byte count enforcement.
- **Depends on:** Task 1 (uses `ConstraintEntry` type)
- **Effort:** 5 points
- **Spec requirements:** REQ-002, REQ-003, REQ-007, REQ-008, REQ-NF-001, REQ-NF-002, REQ-NF-005,
  AC-001, AC-002, AC-006, AC-007

---

### Task 3: Stage-Binding Hydration Payload

- **Description:** Define a `BindingPayload` struct in `internal/card` that serialises the
  machine-readable stage binding fields needed by REQ-006: role names, skill names, effort budget
  text, human gate flag, prerequisites (documents, tasks), and sub-agent profile (roles, skills,
  topology). Implement `HydrateBinding(stage string, b *binding.StageBinding) BindingPayload`.
  Fields absent in a given binding (e.g. no sub-agents) are omitted from the payload rather than
  zeroed, so consumers see no spurious empty objects.
- **Deliverable:**
  - `internal/card/hydration.go` — `BindingPayload` struct with JSON tags, `HydrateBinding` function.
  - `internal/card/hydration_test.go` — unit tests: full binding, minimal binding (omitted fields),
    nil binding (graceful empty payload).
- **Depends on:** None (uses `binding.StageBinding` from the existing `internal/binding` package)
- **Effort:** 2 points
- **Spec requirements:** REQ-006, AC-005

---

### Task 4: Inject Constraint Card and Stage-Binding Payload into `next` Responses

- **Description:** Wire the renderer and hydrator into `nextClaimMode` in `internal/mcp/next_tool.go`.
  After assembling the existing context, call the renderer with the resolved role (from
  `assembledContext.roleProfile`) and the stage binding (looked up from the binding registry using
  `featureStage`), and the hydrator for the binding payload. Add two new top-level fields to the
  result map: `constraint_card` (rendered markdown string) and `stage_binding` (hydrated payload).
  Position `constraint_card` as the first key in the result map so it appears before `context` in
  the JSON response, satisfying "before the detailed task context" (REQ-004, AC-003).
  All existing fields (`task`, `context`, `reclaimed`) must remain present with unchanged shapes (REQ-010, AC-009).
  If the renderer returns an error (missing data), return an error to the caller — do not silently
  return an empty card.
- **Deliverable:**
  - Modified `internal/mcp/next_tool.go` — `nextClaimMode` produces `constraint_card` and
    `stage_binding` fields; `NextTools` signature updated if new dependencies are needed.
  - Updated `internal/mcp/server.go` wiring if `NextTools` needs new arguments (binding registry,
    constraint registry).
  - `internal/mcp/next_tool_constraint_test.go` — tests: constraint card present and non-empty,
    stage_binding fields correct, existing fields still present.
- **Depends on:** Task 2, Task 3
- **Effort:** 3 points
- **Spec requirements:** REQ-004, REQ-010, AC-003, AC-009 (partial — `next` side)

---

### Task 5: Inject Constraint Card and Stage-Binding Payload into `handoff` Responses

- **Description:** Prepend the rendered constraint card to the `prompt` string assembled by
  `kbzctx.RenderPrompt(result)` in `internal/mcp/handoff_tool.go`. The card is derived from the
  resolved role and stage binding already available in the `PipelineResult` / `PipelineState`.
  Expose the resolved role and binding from the pipeline result (if not already accessible) or call
  the renderer directly after `pipeline.Run()`. Also add `stage_binding` as a new top-level field
  in the `handoff` JSON response. All existing response fields (`task_id`, `display_id`, `entity_ref`,
  `prompt`, `context_metadata`) must remain (REQ-010, AC-009).
- **Deliverable:**
  - Modified `internal/mcp/handoff_tool.go` — `prompt` is prefixed with the card;
    `stage_binding` field added to response JSON; `HandoffTools` signature updated if new arguments needed.
  - Updated `internal/mcp/server.go` wiring if needed.
  - `internal/mcp/handoff_tool_constraint_test.go` — tests: prompt begins with card, stage_binding
    present, existing fields preserved.
- **Depends on:** Task 2, Task 3
- **Effort:** 3 points
- **Spec requirements:** REQ-005, REQ-010, AC-004, AC-009 (partial — `handoff` side)

---

### Task 6: Golden Tests, Validation Suite, and Regression Tests

- **Description:** Build the full test suite for all acceptance criteria not already covered by
  unit tests in Tasks 1–5. Specifically:
  1. **Golden tests (AC-008):** Render the constraint card for developing, specifying, dev-planning,
     and reviewing stage/role pairs using fixture inputs. Store expected output in
     `internal/card/testdata/golden/` files. Tests assert exact byte-for-byte match. Run with
     `-update` flag to regenerate.
  2. **Size enforcement tests (AC-010):** Iterate over all role YAML files and all stage bindings;
     render each combination and assert ≤25 non-empty lines and ≤2500 bytes.
  3. **Determinism test (REQ-NF-004):** Render the same inputs 100 times and assert all outputs are
     identical.
  4. **Latency test (REQ-NF-003):** Render and hydrate in a benchmark (`Benchmark*`) and assert
     p95 < 10ms under `go test -bench`.
  5. **Regression tests (AC-009):** Call `nextClaimMode` and `handoffTool` end-to-end with a fixture
     task and assert that `task`, `context`, `reclaimed`, `task_id`, `display_id`, `entity_ref`,
     `prompt`, `context_metadata` fields are all present in the response.
  6. **Unknown-stage test (AC-007):** Render a card for a task with an unrecognised stage and assert
     the output contains `UNKNOWN STAGE` and the manual-load instruction.
  7. **Validation test (AC-006):** Attempt rendering with a role missing `identity` and assert the
     error message names the missing field.
- **Deliverable:**
  - `internal/card/golden_test.go` — golden tests with fixture data.
  - `internal/card/testdata/golden/` — expected card output files for 4 stages.
  - `internal/card/renderer_bench_test.go` — latency benchmark.
  - `internal/mcp/next_tool_regression_test.go` — next field-presence regression test.
  - `internal/mcp/handoff_tool_regression_test.go` — handoff field-presence regression test.
- **Depends on:** Task 4, Task 5
- **Effort:** 5 points
- **Spec requirements:** REQ-009, AC-006, AC-007, AC-008, AC-009, AC-010, REQ-NF-003, REQ-NF-004

---

## Dependency Graph

```
Task 1 (no dependencies)
Task 3 (no dependencies)

Task 2 → depends on Task 1
Task 4 → depends on Task 2, Task 3
Task 5 → depends on Task 2, Task 3

Task 6 → depends on Task 4, Task 5
```

**Parallel groups:**
- Wave 1 (independent): [Task 1, Task 3]
- Wave 2 (after T1): [Task 2]
- Wave 3 (after T2 + T3): [Task 4, Task 5]
- Wave 4 (after T4 + T5): [Task 6]

**Critical path:** Task 1 → Task 2 → Task 4 → Task 6 (total: 3 + 5 + 3 + 5 = 16 points)

**Parallelisable work:** Task 3 can proceed concurrently with Tasks 1 and 2.
Tasks 4 and 5 can be done in parallel once Task 2 and Task 3 are complete.

---

## Risk Assessment

### Risk: `PipelineResult` does not expose resolved role or binding
- **Probability:** medium
- **Impact:** medium — Task 5 (handoff injection) needs access to the resolved role and binding
  after `pipeline.Run()`. If `PipelineResult` only exposes `Sections` and metadata, the
  implementer must either (a) add fields to `PipelineResult`, or (b) resolve role/binding
  independently in the handoff handler.
- **Mitigation:** Task 5 scoped to accept both approaches. Option (b) is the lower-risk path
  (no pipeline API change required); use the binding registry and role store directly in the
  handoff handler, mirroring the pattern used in `nextClaimMode`.
- **Affected tasks:** Task 5

### Risk: `nextClaimMode` assembly path does not surface resolved role for renderer
- **Probability:** medium
- **Impact:** low — `assembledContext.roleProfile` is a string (not a `*ResolvedRole`), so Task 4
  may need to call the role store directly to get the typed role.
- **Mitigation:** Task 4 uses the `roleStore` already threaded into `nextClaimMode` as a parameter.
  Re-resolve the role from the store; this is a cheap read with no side effects.
- **Affected tasks:** Task 4

### Risk: Constraint YAML entries don't align with role/stage pairs in tests
- **Probability:** low
- **Impact:** medium — Golden tests fail if `.kbz/constraints.yaml` entries don't cover the 4
  required role/stage pairs.
- **Mitigation:** Task 1 writes the YAML entries for all 4 required stages before Task 6 writes
  the golden tests. Golden test fixture files are generated with the `-update` flag on first run,
  making them stable thereafter.
- **Affected tasks:** Task 1, Task 6

### Risk: Byte or line budget exceeded by verbose constraint entries
- **Probability:** low
- **Impact:** low — REQ-NF-001 and REQ-NF-002 enforce hard limits.
- **Mitigation:** The renderer enforces limits at render time and fails if exceeded. Task 2 includes
  explicit size enforcement tests. Constraint entries in `.kbz/constraints.yaml` must be concise
  rule statements (no anti-pattern bodies or examples).
- **Affected tasks:** Task 2

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|---------------------|----------------|
| AC-001 (REQ-001, REQ-002): card generated from typed inputs | Unit test — renderer with fixture inputs, assert no hand-written card fixture | Task 2 |
| AC-002 (REQ-003): developing card names role, stage, skill, constraints | Unit test — golden render for developing/implementer | Task 6 |
| AC-003 (REQ-004): `next` response includes card before task context | Integration test — claim fixture task, assert `constraint_card` is first/present | Task 4 |
| AC-004 (REQ-005): `handoff` prompt begins with card | Integration test — call `handoff`, assert `prompt` begins with card | Task 5 |
| AC-005 (REQ-006): context payload contains stage-binding fields | Unit test — hydrator with fixture binding, assert fields present | Task 3 |
| AC-006 (REQ-007): missing role data → loud error naming missing field | Unit test — renderer with stub role missing `identity` | Task 2, confirmed in Task 6 |
| AC-007 (REQ-008): unknown stage → `UNKNOWN STAGE` warning + manual-load instruction | Unit test — renderer called with unrecognised stage | Task 2, confirmed in Task 6 |
| AC-008 (REQ-009): golden tests match for 4 stages | Golden test suite in `internal/card/testdata/golden/` | Task 6 |
| AC-009 (REQ-010): prior `next` and `handoff` fields still present | Regression test — field-presence assertions before and after injection | Task 6 |
| AC-010 (REQ-NF-001, REQ-NF-002): ≤25 lines, ≤2500 bytes for all role/stage pairs | Size enforcement test — render all combinations in test | Task 6 |
| REQ-NF-003: <10ms p95 latency | Benchmark — `go test -bench` | Task 6 |
| REQ-NF-004: deterministic output | Determinism test — 100 renders of same inputs | Task 6 |
