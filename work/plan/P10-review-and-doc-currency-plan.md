# P10: Review Workflow and Documentation Currency — Implementation Plan

| Document | P10 Implementation Plan                              |
|----------|------------------------------------------------------|
| Status   | Draft                                                |
| Created  | 2026-03-28T15:31:17Z                                 |
| Plan     | P10-review-and-doc-currency                          |
| Source   | `work/research/post-p9-feedback-analysis.md`         |
| Related  | `.skills/code-review.md` (existing feature-level review SKILL) |
|          | `internal/health/check.go` (health check framework)  |
|          | `internal/mcp/health_tool.go` (AdditionalHealthChecker pattern) |

---

## 1. Purpose

This plan defines the work breakdown, sequencing, and constraints for P10: Review Workflow and Documentation Currency. It addresses two systemic gaps identified in the post-P9 feedback analysis:

1. **The review workflow gap** — plan-level reviews happen outside the Kanbanzai workflow system, so context assembly, nudges, and retro signal capture are structurally unable to help during the activity that most needs them.
2. **Documentation drift** — stale tool references in SKILL files and missing AGENTS.md updates survive across multiple plans without automated detection.

The plan delivers three features (A, B, C) corresponding to the three recommendations (R1, R2, R3) in the feedback analysis.

---

## 2. Outcome

When P10 is complete, the system will:

1. Provide a plan-level review SKILL (`.skills/plan-review.md`) that captures the improvised review procedure as a repeatable, tool-routed checklist.
2. Support review tasks as first-class workflow entities — plan reviews are claimable via `next`, get context assembly, trigger nudges, and capture retro signals through `finish`.
3. Detect stale tool references in `.skills/*.md` and `AGENTS.md` via a new health check category.
4. Detect missing AGENTS.md Project Status and Scope Guard updates when a plan reaches a terminal state.

---

## 3. Planning Principles

### 3.1 Documentation first, implementation second

Feature A (the review SKILL) is pure documentation and has no code dependencies. It should ship first — it's immediately useful even without B or C. It also serves as the procedural definition that Feature B will automate.

### 3.2 Extend existing patterns

Feature B extends the entity lifecycle and `next`/`finish` tools. Feature C extends the `AdditionalHealthChecker` pattern used by Phase 3, 4a, and 4b health checks. No new architectural patterns are needed.

### 3.3 Design before implementation for Feature B

Feature A needs no design document — the content is fully specified in the feedback analysis. Feature C is a straightforward extension of the health check framework and can go directly to spec. Feature B changes the entity lifecycle (adding plan-level review semantics) and needs a design document to resolve how plan review interacts with the existing feature-level `reviewing` state.

### 3.4 Keep it small

The three features are deliberately scoped to close known gaps, not to redesign the review system. Resist scope expansion into review workflow automation, multi-reviewer coordination, or review metrics.

---

## 4. Features

| Feature | Name | Effort | Dependencies |
|---------|------|--------|--------------|
| **A** | Plan-level review SKILL | Small | None |
| **B** | Review tasks as workflow entities | Medium | A (procedural definition) |
| **C** | Documentation currency health check | Medium | None |
| **D** | Plan document naming convention | Small | None |

Features A, C, and D are independent and can be developed in parallel. Feature B depends on A for its procedural definition (the SKILL tells you what the review task should assemble).

---

## 5. Feature A: Plan-Level Review SKILL

### 5.1 Summary

Create `.skills/plan-review.md` — a SKILL document that defines the procedure for reviewing a completed plan. This captures the ad-hoc procedure improvised during P7–P9 reviews and routes the reviewer through Kanbanzai tools.

### 5.2 Approach

The SKILL follows the same structure as `.skills/code-review.md`: purpose, audience, inputs, procedure, structured output, and orchestration notes. The procedure is derived from the post-P9 feedback analysis §6 R1 but expanded into the full SKILL format.

### 5.3 Deliverables

| Deliverable | Path |
|-------------|------|
| Plan review SKILL | `.skills/plan-review.md` |
| AGENTS.md update | Reference to new SKILL in the Key Design Documents table |

### 5.4 Procedure outline

The SKILL must cover at minimum:

1. **Plan discovery** — `status(id: "<plan-id>")` to get the full dashboard: features, tasks, documents, attention items.
2. **Scope verification** — confirm all features are in terminal state; check for `needs-rework`, `blocked`, or `active` items; verify the plan summary matches what was delivered.
3. **Spec conformance** — for each feature, read the spec acceptance criteria and verify against implementation. Use `entity(action: "list", parent: "<plan-id>")` to enumerate features.
4. **Documentation currency** — check AGENTS.md Project Status mentions the plan; check Scope Guard lists it as complete; verify spec document status is Approved.
5. **Cross-cutting checks** — `go test -race ./...`; `health()` for new warnings; check for uncommitted changes.
6. **Retro contribution** — before finishing, contribute retro signals via `finish` (if working through the entity lifecycle) or `knowledge(action: "contribute")`.
7. **Report output** — write findings to `work/reviews/review-<plan-slug>.md` using the established review document format.

### 5.5 Scope exclusions

- The SKILL does not cover feature-level code review (that's `.skills/code-review.md`).
- The SKILL does not define how to create or dispatch review tasks (that's Feature B).
- The SKILL does not automate any checks (that's Feature C for documentation currency).

### 5.6 Acceptance criteria

| # | Criterion |
|---|-----------|
| A.1 | `.skills/plan-review.md` exists and follows the SKILL document structure |
| A.2 | The procedure routes the reviewer through at least `status`, `entity list`, and `health` |
| A.3 | The procedure includes a retro contribution step |
| A.4 | The procedure includes a documentation currency step (manual until Feature C automates it) |
| A.5 | AGENTS.md Key Design Documents table references the new SKILL |

---

## 6. Feature B: Review Tasks as Workflow Entities

### 6.1 Summary

Add a `reviewing` state to the Plan lifecycle, making plan-level reviews a first-class lifecycle gate. This mirrors the feature-level `reviewing` state added in P6, following the same pattern: add the lifecycle gate first, build automation around it later.

### 6.2 Decided approach: Plan lifecycle extension (Option 1)

Add `reviewing` to the plan state machine: `active → reviewing → done`. A plan review is a plan-level activity, so it belongs on the plan lifecycle — not shoehorned into the task model via a synthetic feature.

**Key design decisions:**

1. **Plan lifecycle change.** The plan state machine gains `reviewing` between `active` and `done`. The transition `active → done` is removed (plans must pass through review, just as features must). The current plan lifecycle is `proposed → designing → active → done` (plus `superseded`/`cancelled` terminals). The new lifecycle is `proposed → designing → active → reviewing → done`.

2. **Plan review is independent of feature review.** Feature reviews happen during the feature `developing → reviewing → done` lifecycle. Plan review happens after all features are done. It reviews the aggregate — documentation currency, spec status, AGENTS.md updates, cross-cutting concerns. The two review types are complementary, not overlapping.

3. **Context assembly is `status`, not `next`.** `status(id: "<plan>")` already produces the plan dashboard: feature list, task status, associated documents, attention items. This *is* the context assembly for a plan review. There is no need to extend `next` to handle plans in this phase. The plan review SKILL (Feature A) routes the reviewer through `status` as the entry point.

4. **No automatic review creation.** Transitioning a plan to `reviewing` is a manual action. No nudge or automatic trigger when features complete — that adds complexity and may not suit all workflows.

5. **Deferred: `next`/`finish` for plans.** Making plans claimable via `next` and completable via `finish` (with nudges) is valuable but not required for the lifecycle gate to work. This is future work, following the same incremental pattern as P6: lifecycle gate first, tooling automation later.

### 6.3 Approach

1. Write a specification with acceptance criteria.
2. Add `PlanStatusReviewing` to `internal/model/entities.go`.
3. Update the plan state machine in `internal/validate/` to add the `reviewing` state and transitions: `active → reviewing`, `reviewing → done`, `reviewing → needs-rework` (if we want rework parity with features — to be decided in spec).
4. Remove the direct `active → done` transition.
5. Update `status` dashboard to reflect the new state.
6. Update `advance: true` logic if it applies to plans.
7. Verify existing plans in `done` state are unaffected (the state machine change only affects future transitions).

### 6.4 Deliverables

| Deliverable | Path |
|-------------|------|
| Specification | `work/spec/plan-review-lifecycle.md` |
| Model change | `internal/model/entities.go` (add `PlanStatusReviewing`) |
| State machine update | `internal/validate/` (plan transitions) |
| Tests | Colocated `*_test.go` files |

### 6.5 Acceptance criteria

| # | Criterion |
|---|-----------|
| B.1 | `PlanStatusReviewing` exists in the plan status enum |
| B.2 | Plan state machine allows `active → reviewing` and `reviewing → done` |
| B.3 | Plan state machine does not allow `active → done` (must pass through `reviewing`) |
| B.4 | Plans already in `done` state are unaffected (no migration required) |
| B.5 | `status(id: "<plan>")` correctly displays plans in `reviewing` state |
| B.6 | `entity(action: "transition", id: "<plan>", status: "reviewing")` works |
| B.7 | `go test -race ./...` passes |

---

## 7. Feature C: Documentation Currency Health Check

### 7.1 Summary

Add a new health check category that detects stale references in agent-facing documentation. Two tiers: tool name validation (automated) and plan completion documentation checklist (entity-state-driven).

### 7.2 Approach

Implement as an `AdditionalHealthChecker` (the same pattern used by `Phase3HealthChecker`, `Phase4aHealthChecker`, and `Phase4bHealthChecker`), registered in the MCP server alongside existing checkers. This keeps the health check framework unchanged and follows the established extension pattern.

### 7.3 Tier 1: Tool Name Validation

Scan `.skills/*.md` and `AGENTS.md` for backtick-wrapped identifiers that match known tool name patterns. Compare against the registered MCP tool set. Flag any referenced tool name that isn't in the registry.

**Implementation sketch:**

1. Read `.skills/*.md` and `AGENTS.md` from the repository root.
2. Extract candidate tool names using a pattern like `` `tool_name` `` or `tool(action: ...)` invocations.
3. Compare against the set of tool names from the MCP server's tool registry (available at server construction time).
4. Emit a health warning for each unrecognised tool name, including the file path and line number.

**Design consideration:** The tool registry is available in the MCP server layer (`internal/mcp/`) but the health check framework (`internal/health/`) operates at a lower level. The `AdditionalHealthChecker` closure pattern bridges this gap — the checker is constructed in the MCP layer with access to the tool registry, then passed to the health tool as a closure. This is the same pattern used for worktree and knowledge health checks.

### 7.4 Tier 2: Plan Completion Documentation Checklist

When a plan is in a terminal state (`done`), verify:

1. AGENTS.md Project Status section mentions the plan slug or ID.
2. AGENTS.md Scope Guard section lists the plan as complete.
3. All spec documents associated with the plan's features have status `Approved`.

**Implementation sketch:**

1. List all plans in terminal state.
2. For each, read AGENTS.md and check for slug/ID presence in the Project Status and Scope Guard sections.
3. List features under each terminal plan; for each feature, check associated spec document status.
4. Emit a health warning for each missing mention or non-approved spec.

**Design consideration:** This requires reading AGENTS.md content, which the health system doesn't currently do. The checker will need the repository root path (already available via `CheckOptions.RepoPath`) and a way to enumerate plan-to-feature-to-spec relationships (available via `EntityService` and document records). The `AdditionalHealthChecker` closure can capture these dependencies at construction time.

### 7.5 Deliverables

| Deliverable | Path |
|-------------|------|
| Specification | `work/spec/doc-currency-health-check.md` |
| Health check implementation | `internal/mcp/doc_currency_health.go` |
| Tests | `internal/mcp/doc_currency_health_test.go` |
| MCP server registration | `internal/mcp/server.go` (register the new checker) |

### 7.6 Acceptance criteria

| # | Criterion |
|---|-----------|
| C.1 | Health check detects a tool name in `.skills/*.md` that is not in the MCP tool registry |
| C.2 | Health check detects a tool name in `AGENTS.md` that is not in the MCP tool registry |
| C.3 | Health check does not flag tool names that are in the registry |
| C.4 | Health check detects a plan in `done` state with no mention in AGENTS.md Project Status |
| C.5 | Health check detects a plan in `done` state with no mention in AGENTS.md Scope Guard |
| C.6 | Health check detects a feature spec document that is not in `Approved` status when the parent plan is `done` |
| C.7 | Health check does not flag plans that are not in terminal state |
| C.8 | The new checker is registered via the `AdditionalHealthChecker` pattern |
| C.9 | `go test -race ./...` passes |

---

## 8. Feature D: Plan Document Naming Convention

### 8.1 Summary

Establish a naming convention that prefixes plan document filenames with their P-number, making them discoverable by plan identity rather than requiring knowledge of the slug or phase name.

Example: `review-and-doc-currency-plan.md` → `P10-review-and-doc-currency-plan.md`

### 8.2 Motivation

Plan documents are currently named by slug or phase number, with no visible link to the plan entity ID. Finding "the P9 plan document" requires knowing that P9's slug is `mcp-discoverability` or searching file contents. A P-number prefix makes the mapping immediate and supports tab-completion in the shell.

### 8.3 Convention

Plan document filenames in `work/plan/` follow the pattern: `P{N}-{slug}-{type}.md`

Where `{type}` is one of: `implementation-plan`, `decision-log`, `progress`, `scope`, or other plan-level document types.

Examples:
- `P10-review-and-doc-currency-plan.md`
- `P4-kanbanzai-2.0-implementation-plan.md`
- `P6-workflow-quality-and-review-plan.md`

### 8.4 Scope

| Item | Action |
|------|--------|
| New plan documents | Must use the `P{N}-` prefix convention going forward |
| Existing plan documents (8 implementation plans + supporting docs) | Rename in a single commit, update all internal cross-references, update document records |
| `AGENTS.md` references | Update any file paths that reference renamed documents |
| `bootstrap-workflow.md` | Document the naming convention in the document placement table |

### 8.5 Deliverables

| Deliverable | Description |
|-------------|-------------|
| Renamed files | Existing plan docs renamed with `P{N}-` prefix |
| Updated cross-references | All documents and AGENTS.md that reference renamed files |
| Updated document records | `doc(action: refresh)` on any renamed documents |
| Convention documented | Naming rule added to `bootstrap-workflow.md` |

### 8.6 Acceptance criteria

| # | Criterion |
|---|-----------|
| D.1 | All plan implementation plans in `work/plan/` have a `P{N}-` filename prefix |
| D.2 | All internal cross-references to renamed files are updated |
| D.3 | Document records for renamed files have correct paths |
| D.4 | The naming convention is documented in `bootstrap-workflow.md` |
| D.5 | No broken file references in AGENTS.md or other documents |

---

## 9. Sequencing

```
Phase 1: Documentation (Feature A)
├── Write .skills/plan-review.md
├── Update AGENTS.md key documents table
└── Gate: human approval of SKILL content

Phase 2: Parallel implementation (Features B, C, and D)
├── Track B: Plan review lifecycle
│   ├── Write specification
│   ├── Gate: human approval of spec
│   └── Implement (model, state machine, tests)
│
├── Track C: Health check
│   ├── Write specification
│   ├── Gate: human approval of spec
│   └── Implement
│
└── Track D: Naming convention
    ├── Rename existing plan documents
    ├── Update all cross-references
    └── Document convention in bootstrap-workflow

Phase 3: Integration and review
├── Verify all acceptance criteria
├── Update AGENTS.md (Project Status, Scope Guard)
├── Use the plan review SKILL (Feature A) to review the plan itself
└── Gate: plan review passes
```

Feature A ships first because it's immediately useful and defines the procedure that Feature B will enforce as a lifecycle gate. Features B, C, and D are independent and can be developed in parallel. Feature D is small enough to ship early alongside A. Phase 3 is a natural validation — we use Feature A's SKILL to conduct the plan review, which now passes through Feature B's `reviewing` lifecycle state, and Feature C's health check validates that the documentation was updated.

---

## 9. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Removing `active → done` breaks existing automation or scripts | Plans can't be closed without review | Verify no tooling assumes direct `active → done`; add `reviewing` transition guidance to plan review SKILL |
| Tool name extraction produces false positives | Noisy health output | Use conservative matching (backtick-wrapped names, known tool name patterns); tune in spec |
| AGENTS.md format changes break Tier 2 checks | Health check produces false negatives | Match on plan slug (stable) not section formatting; document expected AGENTS.md structure |
| Plan review SKILL becomes stale (ironic) | Same drift problem we're solving | Feature C Tier 1 will catch stale tool names in SKILL files; Tier 2 catches missing plan updates |
| Renaming plan docs breaks external links or bookmarks | Stale URLs or file references | Single-commit rename with comprehensive cross-reference update; `git log --follow` preserves history |

---

## 10. Out of Scope

These are adjacent concerns that are explicitly excluded from P10:

- **Multi-reviewer coordination** — P10 covers single-reviewer plan review. Multi-reviewer workflows (assigning reviewers, resolving conflicting findings) are a separate concern.
- **Review metrics and dashboards** — tracking review velocity, finding rates, or rework frequency is useful but not needed to close the identified gaps.
- **Automated review orchestration** — P10 makes reviews possible within the workflow system; automating the orchestration (auto-dispatching review sub-agents) is future work.
- **Feature-level review SKILL updates** — `.skills/code-review.md` covers feature review and is orthogonal to plan review. If stale references are found in it, Feature C will detect them, but rewriting the feature review SKILL is not in scope.
- **Cross-project documentation linting** — Feature C checks Kanbanzai-specific documentation (SKILL files, AGENTS.md). General-purpose documentation linting is out of scope.