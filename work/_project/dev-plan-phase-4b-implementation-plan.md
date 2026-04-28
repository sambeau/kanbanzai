# Phase 4b Implementation Plan

| Document | Phase 4b Implementation Plan                         |
|----------|------------------------------------------------------|
| Status   | Draft                                                |
| Created  | 2026-03-25                                           |
| Updated  | 2026-03-25                                           |
| Related  | `work/spec/phase-4b-specification.md`                |
|          | `work/plan/phase-4-scope.md`                         |
|          | `work/plan/phase-4-decision-log.md`                  |
|          | `work/plan/phase-4a-implementation-plan.md`          |

---

## 1. Overview

This document defines the implementation plan for Phase 4b: Self-Managed Capabilities.

Phase 4b delivers six feature tracks, developed under self-management using Phase 4a orchestration tooling:

1. **Document store removal** — Execute P4-DES-007 before any feature work begins
2. **Automatic dependency unblocking** — `StatusTransitionHook` extension; `complete_task` gains `unblocked_tasks`
3. **Feature decomposition** — `decompose_feature`, `decompose_review`, `slice_analysis`; vertical slice guidance
4. **Worker review** — `review_task_output`; `needs-rework` transition; spec-section tracing
5. **Conflict domain analysis** — `conflict_domain_check`; `work_queue --conflict-check` annotation
6. **Incidents and RCA** — `Incident` entity; `RootCauseAnalysis` document type; health check extension

The gate for Phase 5 is: all Phase 4b acceptance criteria verified, all Phase 4a punch list items (PL-1 through PL-5) resolved, `go test -race ./...` clean, no blocking health check errors.

---

## 2. Pre-Implementation: Phase 4a Punch List

Before any Phase 4b feature track begins, the Phase 4a punch list must be cleared. These are known gaps that will obstruct Phase 4b if left unresolved.

| Item | Description | Estimated effort |
|------|-------------|-----------------|
| PL-1 | `update status --type plan` CLI routing — add `plan` case to `parseRecordIdentity` | S |
| PL-2 | Phase 4a CLI commands (§13.1–13.4 of Phase 4a spec) | M–L |
| PL-3 | MCP integration tests for Phase 4a tools; update `TestServer_ListTools` | M |
| PL-4 | Create `phase-4a-progress.md`; tick all §15 acceptance criteria | S |
| PL-5 | Update `AGENTS.md` to reflect Phase 4a completion | S |

These items are tracked in `work/plan/phase-4a-implementation-plan.md` §10.2.

---

## 3. Pre-Implementation: Document Store Removal (P4-DES-007)

Per spec §17.1 and P4-DES-007, the Phase 1 document store is removed before Phase 4b feature work begins. This is not a feature track — it is a prerequisite cleanup.

**Steps:**

| Step | Action |
|------|--------|
| D.1 | Register `work/spec/phase-4b-specification.md` as a Phase 2a document record in `.kbz/state/documents/` |
| D.2 | Register `work/plan/phase-4b-implementation-plan.md` as a Phase 2a document record |
| D.3 | Verify both records can be retrieved via `get_entity` on their DOC IDs |
| D.4 | Remove `internal/document/` package |
| D.5 | Remove `.kbz/docs/` directory |
| D.6 | Remove `doc` CLI command group from `cmd/kanbanzai/main.go` |
| D.7 | Remove `scaffold_document`, `submit_document`, `approve_document`, `list_documents`, `validate_document` MCP tools from `internal/mcp/` |
| D.8 | Update `TestServer_ListTools` expected tool list |
| D.9 | Confirm `go test -race ./...` passes |
| D.10 | Update spec §16.7 acceptance criteria checkboxes |

**Verification gate:** D.3 must succeed before D.4 proceeds. If the Phase 2a document record path fails to retrieve the registered documents, stop and diagnose before removing anything.

---

## 4. Implementation Strategy

### 4.1 Dependency structure

```
Punch list (PL-1 to PL-5) + Document store removal (D.1–D.10)
                              │
              ┌───────────────┼──────────────────────────┐
              │               │                          │
              ▼               ▼                          ▼
   Track A:            Track B:                   Track E:
   Automatic           Feature                    Incidents
   Unblocking          Decomposition              and RCA
   (independent,       (depends on                (independent)
   additive)           doc intelligence)
              │               │
              └───────┬───────┘
                      │
                      ▼
               Track C:
               Worker Review
               (depends on doc
               intelligence + Track A
               for rework_reason clear)
                      │
              ┌───────┴──────────┐
              │                  │
              ▼                  ▼
   Track D:                Track F:
   Conflict Domain         Vertical Slice
   Analysis               Guidance
   (depends on            (builds on
   worktree/git)          Track B)
```

Tracks A and E are independent and can start immediately after prerequisites. Track B (decomposition) and Track C (worker review) both require the document intelligence pipeline from Phase 2a — verify `internal/docint/` is available before starting. Track D (conflict analysis) requires git history access via `internal/git/`. Track F (slice analysis) builds on the decomposition guidance established in Track B.

### 4.2 Self-management note

Phase 4b is the first phase fully developed under self-management. Each feature track should be implemented as one or more dispatched tasks via `dispatch_task`. As each track completes, use `complete_task` with knowledge contributions. Once `review_task_output` is implemented (Track C), use it to review subsequent track completions.

---

## 5. Track A: Automatic Dependency Unblocking

**Goal:** Extend `StatusTransitionHook` so completing a task automatically promotes newly unblocked tasks to `ready`. Extend `complete_task` response with `unblocked_tasks`.

**Spec reference:** §7

| Task | Description | Size |
|------|-------------|------|
| A.1 | Define `DependencyUnblockingHook` in `internal/service/` — signature, inputs, outputs | S |
| A.2 | Implement hook: load tasks with `depends_on` referencing completed task ID; check all deps terminal; transition eligible tasks to `ready` | M |
| A.3 | Wire hook into `StatusTransitionHook` chain in `EntityService.UpdateStatus`; fires on terminal transitions only | S |
| A.4 | Ensure hook failure does not fail the original transition (log warning, continue) | S |
| A.5 | Extend `CompleteResult` and `complete_task` MCP response with `unblocked_tasks` field | S |
| A.6 | Write tests: no dependents (no-op); one task fully unblocked; one task partially unblocked (not promoted); chain A→B→C (completing A unblocks B only) | M |
| A.7 | Round-trip test: `complete_task` response includes empty `unblocked_tasks` array when nothing unblocked | S |
| A.8 | Verify hook fires on `not-planned` and `duplicate` terminal transitions, not just `done` | S |

**Key notes:**
- The hook reads and writes entity state using the existing `EntityService`. No additional locking is needed — the per-request serialisation model applies.
- `queued` and `blocked` tasks are both eligible for promotion. The hook does not distinguish between them.
- The promotion it performs is a system-initiated `queued/blocked → ready` transition. It bypasses the normal dependency enforcement gate (the hook already satisfies the gate by construction) — use `UpdateStatus` with an internal bypass flag or call the store directly to avoid re-running the gate check.

**Verification (spec §16.2):** All 7 acceptance criteria must have passing tests.

---

## 6. Track B: Feature Decomposition

**Goal:** Implement `decompose_feature` and `decompose_review` MCP tools with embedded vertical slice guidance. No tasks are written by either tool — they produce proposals only.

**Spec reference:** §6

| Task | Description | Size |
|------|-------------|------|
| B.1 | Define `DecomposeInput` and `DecomposeResult` types in `internal/service/decompose.go` | S |
| B.2 | Implement spec document loading via the Phase 2a document record path (read `spec` field from Feature, load DOC record, read file at `path`) | M |
| B.3 | Implement decomposition guidance engine: apply the six rules from §6.5; produce a `ProposedTask` list | L |
| B.4 | Implement `decompose_feature` MCP tool wiring in `internal/mcp/decompose_tools.go` | S |
| B.5 | Define `DecomposeReviewInput` and `DecomposeReviewResult` types | S |
| B.6 | Implement review logic: check proposal against spec acceptance criteria; detect oversized tasks, dependency cycles, ambiguous summaries | M |
| B.7 | Implement `decompose_review` MCP tool wiring | S |
| B.8 | Implement `kbz feature decompose <id>` CLI command | M |
| B.9 | Implement `kbz feature decompose <id> --confirm` with `human_checkpoint` before task creation | M |
| B.10 | Write tests for `decompose_feature`: no spec registered (error); spec present, proposal produced; guidance rules applied | M |
| B.11 | Write tests for `decompose_review`: clean proposal (pass); gap finding; oversized finding; cycle finding | M |
| B.12 | Write tests for confirmation path: checkpoint created before tasks written | S |

**Key notes:**
- B.2 is critical and must be validated before B.3. The Phase 2a document record path has been used for state tracking but not yet exercised for spec content retrieval. Test it in isolation first.
- The decomposition guidance engine (B.3) is heuristic. Quality of output depends on spec structure. The implementation should be conservative — better to produce fewer, clearer proposed tasks than to over-decompose.
- `decompose_feature` must never write tasks. Use a compile-time check or a review in code review to enforce this.

**Verification (spec §16.1):** All 12 acceptance criteria must have passing tests.

---

## 7. Track C: Worker Review

**Goal:** Implement `review_task_output`, integrating with the document intelligence section-tracing pipeline. Add `rework_reason` to Task schema. Implement `needs-rework → active` clearance of `rework_reason`.

**Spec reference:** §8

**Dependencies:** Track A (for `rework_reason` clear on `needs-rework → active` transition); `internal/docint/` doc_trace pipeline.

| Task | Description | Size |
|------|-------------|------|
| C.1 | Add `rework_reason` field to `Task` model and storage field order (spec §12.2) | S |
| C.2 | Implement `rework_reason` clearing in `UpdateStatus` when transitioning `needs-rework → active` | S |
| C.3 | Round-trip test for Task with `rework_reason` field | S |
| C.4 | Define `ReviewInput` and `ReviewResult` types in `internal/service/review.go` | S |
| C.5 | Implement task-level check: verify `output_files` exist on disk; check `output_summary` addresses task `summary` | M |
| C.6 | Implement spec-level check: call `doc_trace` on feature spec; match to task slug/summary; produce `spec_gap` warnings | M |
| C.7 | Implement result aggregation and severity mapping: task-level errors are blocking; spec-level findings are warnings | S |
| C.8 | Implement state transitions triggered by review result: `fail → needs-rework` (set `rework_reason`); `pass/warn → needs-review` | M |
| C.9 | Implement `review_task_output` MCP tool wiring in `internal/mcp/review_tools.go` | S |
| C.10 | Implement `kbz task review <id>` CLI command | M |
| C.11 | Write tests: missing file (fail); verification met (pass); no spec registered (pass_with_warnings); spec gap found (warning, not fail) | M |
| C.12 | Write tests: `needs-rework` transition sets `rework_reason`; `active` transition clears it | S |
| C.13 | Write tests: review on already-`done` task returns findings without transition | S |

**Key notes:**
- Spec-level findings are always warnings, never errors (spec §17.4). A review must never fail solely on spec heuristics.
- If `doc_trace` returns no section matches, add `no_spec_sections_found` warning and skip spec-level checks. Do not error.
- The `review_task_output` tool SHOULD be used to review Track D and E implementations once it is working — this is the self-validation loop described in spec §17.7.

**Verification (spec §16.3):** All 8 acceptance criteria must have passing tests.

---

## 8. Track D: Conflict Domain Analysis

**Goal:** Implement `conflict_domain_check` and the optional `conflict_check` annotation on `work_queue`. Use existing `internal/git/` infrastructure for branch-level file history.

**Spec reference:** §9

**Dependencies:** `internal/git/` branch and log utilities from Phase 3.

| Task | Description | Size |
|------|-------------|------|
| D.1 | Define `ConflictCheckInput` and `ConflictCheckResult` types | S |
| D.2 | Implement file overlap dimension: compare `files_planned` lists; report shared files | S |
| D.3 | Implement dependency order dimension: check `depends_on` chains for ordering between the two tasks | S |
| D.4 | Implement architectural boundary dimension: keyword matching on task summaries and spec sections | M |
| D.5 | Implement git-history overlap: for tasks with worktree branches, run `git log --name-only` since branch creation and compare touched files | M |
| D.6 | Implement risk aggregation: combine dimension results into overall risk level | S |
| D.7 | Implement `conflict_domain_check` MCP tool wiring in `internal/mcp/conflict_tools.go` | S |
| D.8 | Extend `WorkQueueItem` with `conflict_risk` and `conflict_with` fields; extend `work_queue` MCP tool with `conflict_check` boolean parameter | M |
| D.9 | Extend `WorkQueue` service method to accept `ConflictCheck bool`; when true, run check against active tasks for each ready item | M |
| D.10 | Implement `kbz queue --conflict-check` CLI flag | S |
| D.11 | Write tests for each dimension: file overlap detected; ordering conflict detected; no conflict (none result) | M |
| D.12 | Write tests for `work_queue --conflict-check`: annotated items; unannotated items when flag not set | S |

**Key notes:**
- D.5 (git history) is best-effort. If no worktree branch exists for a task, skip the git step and use only `files_planned` for file overlap. Never error when git data is unavailable.
- The architectural boundary check (D.4) is heuristic. Keep it simple — keyword matching is sufficient for Phase 4b. Do not attempt semantic analysis.
- `conflict_check` in `work_queue` adds latency proportional to the number of active tasks × ready tasks. For typical project sizes this is negligible, but document the trade-off.

**Verification (spec §16.4):** All 6 acceptance criteria must have passing tests.

---

## 9. Track E: Incidents and RCA

**Goal:** Implement the `Incident` entity with its full lifecycle, the `RootCauseAnalysis` document type, and the `unlinked_resolved_incidents` health check extension.

**Spec reference:** §11

| Task | Description | Size |
|------|-------------|------|
| E.1 | Add `EntityKindIncident` to `internal/model/entities.go`; define `Incident` struct with all fields and canonical field order | M |
| E.2 | Add `INC` prefix to `TypePrefix` and `EntityKindFromPrefix` in `internal/id/allocator.go` | S |
| E.3 | Add `Incident` lifecycle transitions to `internal/validate/lifecycle.go` | M |
| E.4 | Add `incidents` directory support to entity store (directory name `incidents`) | S |
| E.5 | Implement `CreateIncident`, `UpdateIncident`, `GetIncident`, `ListIncidents` in `internal/service/incidents.go` | M |
| E.6 | Implement `incident_create` MCP tool | S |
| E.7 | Implement `incident_update` MCP tool | S |
| E.8 | Implement `incident_list` MCP tool | S |
| E.9 | Implement `incident_link_bug` MCP tool | S |
| E.10 | Register `rca` document type in `internal/model/entities.go` `DocumentType` list | S |
| E.11 | Add `incident_ids` and `severity` required front-matter handling for `rca` document type | S |
| E.12 | Implement `unlinked_resolved_incidents` health check category in `internal/health/` | M |
| E.13 | Add `incidents.rca_link_warn_after_days` configuration (default: 7; 0 disables) | S |
| E.14 | Implement `kbz incident create/list/show` CLI commands | M |
| E.15 | Write tests for Incident lifecycle: valid transitions; invalid transitions; terminal state enforcement | M |
| E.16 | Write tests for `incident_link_bug`: idempotency; bug not found error | S |
| E.17 | Write round-trip serialisation test for Incident in canonical field order | S |
| E.18 | Write tests for health check: incident flagged after threshold; incident not flagged before threshold; check disabled when threshold 0; incident with linked RCA not flagged | M |

**Key notes:**
- E.10 and E.11: The `rca` document type participates in the Phase 2a document record path (path reference, not copied). The existing `submit_document` MCP tool (if retained after the Phase 1 removal) or its replacement handles submission. The `incident_ids` field is stored in the document record's metadata.
- The knowledge contribution on RCA approval (spec §11.3) is the orchestrator's responsibility, not an automatic system action. The approved document and its content are available for the agent to read and contribute from.
- `E.1` defines the struct. Cross-check the field order in §11.2 of the spec carefully before writing the storage layer — changing field order after round-trip tests exist is painful.

**Verification (spec §16.6):** All 11 acceptance criteria must have passing tests.

---

## 10. Track F: Vertical Slice Guidance

**Goal:** Implement `slice_analysis` as a standalone tool. Integrate slice guidance more deeply into `decompose_feature` output. Add slice tagging convention.

**Spec reference:** §10

**Dependencies:** Track B (decomposition) must be complete; shares the spec document loading infrastructure.

| Task | Description | Size |
|------|-------------|------|
| F.1 | Define `SliceAnalysisInput` and `SliceAnalysisResult` types | S |
| F.2 | Implement slice identification: extract outcomes from spec acceptance criteria; group by stack layer touch; produce candidate slices | M |
| F.3 | Implement inter-slice dependency detection: identify ordering constraints between slices | S |
| F.4 | Implement `slice_analysis` MCP tool wiring in `internal/mcp/decompose_tools.go` | S |
| F.5 | Extend `decompose_feature` to call `slice_analysis` internally and include `slices` in the proposal | S |
| F.6 | Document slice tagging convention (`slice:<name>` tag) in the tool description and CLI help text | S |
| F.7 | Write tests for `slice_analysis`: single-slice feature (one outcome); multi-slice feature; inter-slice dependency detected | M |
| F.8 | Write tests for `decompose_feature` with slice context: `slices` field populated in proposal | S |

**Key notes:**
- Slice identification (F.2) is heuristic. It uses heading structure and acceptance criteria from the spec. A simple approach is sufficient — identify major acceptance criteria sections, map each to stack layers by keyword matching in the section text.
- The slice tagging convention (F.6) is advisory. The system enforces nothing — it is guidance for agents creating tasks.

**Verification (spec §16.5):** All 4 acceptance criteria must have passing tests.

---

## 11. Dependency Graph

```
 Prerequisites
 ┌──────────────────────────────────────────────────────┐
 │  Phase 4a Punch List (PL-1 to PL-5)                 │
 │  Document Store Removal (D.1–D.10)                   │
 └──────────────────┬───────────────────────────────────┘
                    │
       ┌────────────┼──────────────────┐
       │            │                  │
       ▼            ▼                  ▼
  Track A:      Track B:           Track E:
  Automatic     Feature            Incidents
  Unblocking    Decomposition      and RCA
       │            │
       │            ▼
       │        Track C:
       └──────► Worker Review
                    │
           ┌────────┴────────┐
           ▼                 ▼
       Track D:          Track F:
       Conflict          Vertical Slice
       Analysis          Guidance
```

---

## 12. Effort Estimates

### 12.1 Size definitions

Consistent with Phase 4a:

| Size | Effort |
|------|--------|
| S | 1–2 hours |
| M | 2–4 hours |
| L | 4–8 hours |

### 12.2 Track estimates

| Track | Tasks | S | M | L | Est. Total |
|-------|-------|---|---|---|------------|
| Prerequisites (punch list + doc removal) | 15 | 9 | 5 | 0 | 13–21 hrs |
| A: Automatic Unblocking | 8 | 4 | 4 | 0 | 12–20 hrs |
| B: Feature Decomposition | 12 | 5 | 5 | 1 | 19–31 hrs |
| C: Worker Review | 13 | 5 | 7 | 0 | 19–30 hrs |
| D: Conflict Domain Analysis | 12 | 5 | 6 | 0 | 17–26 hrs |
| E: Incidents and RCA | 18 | 9 | 8 | 0 | 25–38 hrs |
| F: Vertical Slice Guidance | 8 | 5 | 3 | 0 | 11–16 hrs |
| **Total** | **86** | **42** | **38** | **1** | **116–182 hrs** |

### 12.3 Phase estimate

At 6–8 productive hours per day:

- **Single agent:** 15–30 days
- **Two agents (A+E parallel with B, then C, then D+F):** 9–16 days

---

## 13. Implementation Order

### 13.1 Recommended sequence (single agent)

| Step | Work | Rationale |
|------|------|-----------|
| 1 | Phase 4a punch list (PL-1 to PL-5) | Clear technical debt before adding new code |
| 2 | Document store removal (D.1–D.10) | Clean foundation; validates Phase 2a doc path |
| 3 | Track A: Automatic unblocking | Additive, no new dependencies; improves Phase 4a immediately |
| 4 | Track E: Incidents and RCA | Independent entity track; can be done in any order |
| 5 | Track B: Feature decomposition | Core new capability; needs doc intelligence |
| 6 | Track C: Worker review | Builds on doc intelligence established in B; uses Track A's rework_reason |
| 7 | Track D: Conflict domain analysis | Uses git infrastructure; builds on C's foundation |
| 8 | Track F: Vertical slice guidance | Builds on Track B; shortest track |
| 9 | Validation and Phase 5 gate | Verify all acceptance criteria; update progress docs |

### 13.2 Parallel execution (two agents)

**Agent 1: Core orchestration improvements (A → B → C → F)**
**Agent 2: Independent tracks (E → D)**

**Integration point:** Agent 1 completes Track B before Agent 2 begins Track D (D uses the spec document loading infrastructure established in B for architectural boundary analysis).

---

## 14. Verification Strategy

### 14.1 Test coverage requirements

| Category | Requirement |
|----------|-------------|
| Unit tests | All business logic: decomposition rules, review severity mapping, conflict risk aggregation, incident lifecycle |
| Integration tests | MCP tool end-to-end for each new tool |
| Round-trip tests | Incident entity; Task with `rework_reason`; all new entity fields |
| Self-review | Once `review_task_output` is working, use it to review Track D and F implementation tasks |
| Race detector | `go test -race ./...` passes at every track completion checkpoint |

### 14.2 Verification checkpoints

| Checkpoint | Tracks | What to verify |
|------------|--------|----------------|
| CP0 | Prerequisites | PL-1 to PL-5 resolved; doc removal complete; tests pass |
| CP1 | A | `complete_task` returns `unblocked_tasks`; hook promotes blocked tasks automatically |
| CP2 | E | Incident lifecycle enforced; health check fires on unlinked resolved incidents |
| CP3 | B | `decompose_feature` returns proposal; `decompose_review` finds gaps; no tasks written |
| CP4 | C | `review_task_output` transitions to `needs-rework` on fail; `needs-review` on pass |
| CP5 | D | `conflict_domain_check` detects file overlap and ordering conflicts |
| CP6 | F | `slice_analysis` returns slices; `decompose_feature` includes slice context |
| CP7 | All | All 44 spec §16 acceptance criteria checked; `go test -race ./...` clean; health check 0 errors |

### 14.3 Acceptance criteria coverage

Each of the 44 acceptance criteria from spec §16 must have at least one test. Reference the criterion in the test function comment: `// Verifies §16.2: complete_task response includes unblocked_tasks`.

---

## 15. Risk Mitigations

### 15.1 Document intelligence dependency

**Risk:** Tracks B, C, and F depend on `internal/docint/` section tracing. If doc_trace returns poor results for the project's spec format, review and decomposition quality degrades.

**Mitigation:** Treat doc_trace as a best-effort lookup throughout. Any tool that uses it must handle the case where it returns zero matches gracefully (fall back to task-level checks only, add a warning finding). Never block on doc_trace quality.

### 15.2 Decomposition quality cold start

**Risk:** `decompose_feature` output quality depends on spec structure. Poorly structured specs produce poor proposals.

**Mitigation:** The tool includes a `warnings` field for this purpose. Surface quality warnings to the orchestrator. `decompose_review` provides a second-pass check. The confirmation gate (human checkpoint before writing tasks) ensures a human can catch a bad proposal before tasks are persisted.

### 15.3 Automatic unblocking and infinite loops

**Risk:** A dependency cycle (A depends on B, B depends on A) could cause the unblocking hook to loop. In practice, the Phase 4a dependency cycle health check should catch cycles before they become active, but the hook must be defensive.

**Mitigation:** The hook iterates tasks that depend on the just-completed task. A cycle where A depends on B and B depends on A cannot be triggered by completing A (completing A does not set A back to a terminal state). The hook is safe against cycles in the data. Add a test to confirm.

### 15.4 Phase 1 document store removal breakage

**Risk:** Removing `internal/document/` may break tests or code paths that import it.

**Mitigation:** Step D.1–D.3 validates the replacement path before anything is removed. The removal is done in one commit. Run `go build ./...` before and after. Use `go test -race ./...` as the acceptance gate. If any package outside `internal/document/` imports it, those imports must be updated or removed before the package is deleted.

### 15.5 Scope creep from self-management

**Risk:** Phase 4b is the first phase developed inside the system. The orchestrating agent may be tempted to add features not in the spec.

**Mitigation:** The spec (§3.2 and §3.3) lists explicitly deferred and excluded items. Any addition requires a new decision record and human approval. The `human_checkpoint` mechanism provides the escalation path.

---

## 16. Definition of Done

A track is complete when:

1. All tasks in the track are implemented
2. All unit and integration tests pass
3. Round-trip tests pass for all new YAML records and entity fields
4. `go test -race ./...` passes
5. `go vet ./...` is clean
6. Relevant spec §16 acceptance criteria have passing tests with criterion references in test comments
7. MCP tools follow existing patterns: unknown parameters rejected, missing required parameters produce clear errors
8. CLI commands are consistent with existing UX patterns

Phase 4b is complete when:

1. All six tracks plus prerequisites are complete
2. All 44 acceptance criteria (spec §16) verified by passing tests
3. Phase 4a punch list items PL-1 through PL-5 all resolved
4. `AGENTS.md` updated to reflect Phase 4b completion
5. `go test -race ./...` clean
6. `health_check` reports 0 errors on a clean project instance

---

## 17. Open Items

### 17.1 Questions to resolve during implementation

| Question | Disposition |
|----------|-------------|
| `decompose_feature`: how should the decomposition guidance engine handle specs with no structured acceptance criteria section? | Return a proposal based on heading structure and a warning that acceptance criteria were not found; do not error |
| `review_task_output`: should output_files be required or optional? | Optional — if not provided, task-level check uses verification field and output_summary only; missing files check is skipped |
| `incident_link_bug`: should it verify the BUG ID exists? | Yes — return a clear error if the BUG entity is not found |
| Phase 2a doc removal: which MCP tools replace `submit_document` and `approve_document`? | `import_document` (batch) is already available; a single-document `register_document` tool may be needed — resolve in D.7 |
| RCA knowledge contribution: automatic on approval or manual? | Manual (orchestrator responsibility per spec §11.3); system provides the document content, agent contributes |

### 17.2 Potential Phase 4b.1 scope (defer if needed)

If any track falls behind, these items can be deferred without breaking the core capability:

- `slice_analysis` tool (Track F) — decomposition still works without it; slice guidance is embedded in `decompose_feature`
- Git history dimension in `conflict_domain_check` (D.5) — file overlap and dependency order are sufficient for an initial useful check
- `kbz feature decompose --confirm` CLI path (B.9) — MCP tool confirmation flow is sufficient; CLI is convenience
- `decompose_review` (B.5–B.7) — `decompose_feature` is useful without the second-pass review tool

---

## 18. Summary

Phase 4b delivers six capability tracks and one prerequisite cleanup, totalling approximately 86 implementation tasks.

**Key deliverables:**

- Phase 4a punch list cleared; Phase 1 document store removed (P4-DES-007)
- Automatic dependency unblocking closes the manual polling loop
- Feature decomposition with spec-driven proposals, review, and confirmation gate
- Worker review integrating the document intelligence pipeline
- Conflict domain analysis for safer parallel dispatch decisions
- Vertical slice guidance embedded in decomposition and surfaced as a standalone tool
- Incident entity and RCA document type for operational failure tracking

**Estimated effort:** 116–182 hours (9–16 days with two agents)

**Self-management gate:** Phase 4b is developed inside the system. Each track is a dispatched task. `review_task_output` reviews subsequent tracks once available.

**Gate for Phase 5:** All 44 acceptance criteria verified, punch list cleared, `AGENTS.md` updated, `go test -race ./...` clean.

**Implementation sequence:** Prerequisites → A (unblocking) → E (incidents) → B (decomposition) → C (review) → D (conflict) → F (slices) → validation.