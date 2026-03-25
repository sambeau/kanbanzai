# Phase 4a Implementation Plan

| Document | Phase 4a Implementation Plan                        |
|----------|-----------------------------------------------------|
| Status   | Draft                                               |
| Created  | 2026-03-25                                          |
| Updated  | 2026-03-25                                          |
| Related  | `work/spec/phase-4a-specification.md`               |
|          | `work/plan/phase-4-scope.md`                        |
|          | `work/plan/phase-4-decision-log.md`                 |
|          | `work/design/estimation-and-progress-design.md`     |

---

## 1. Overview

This document defines the implementation plan for Phase 4a: Orchestration MVP.

Phase 4a delivers the minimum viable orchestration layer across five tracks:

1. **Estimation** — Story point scale, rollup queries, AI calibration references
2. **Dependency Enforcement** — Transition-gated ready queue, dependency status
3. **Task Dispatch and Completion** — Atomic claim, context dispatch, knowledge-contributing completion, human checkpoints
4. **Context Assembly Enhancements** — Trimming visibility, orchestration context injection
5. **Health Check Extensions** — Dependency cycles, stalled dispatches, estimation coverage

The gate for Phase 4b is: Phase 4a complete, validated on at least one real workload, with no blocking health check errors.

---

## 2. Implementation Strategy

### 2.1 Dependency structure

Phase 4a's five tracks have a clear dependency shape:

```
Track A: Estimation          ─────────────────────────────────────┐
Track B: Dependency Enf.     ──────────────────────┐              │
Track D: Context Assembly    ──────────────────┐   │              │
                                               │   │              │
                                               ▼   ▼              │
                                         Track C: Dispatch        │
                                               │                  │
                                               └──────────┐       │
                                                          ▼       ▼
                                                    Track E: Health Checks
```

Tracks A, B, and D are independent of each other and can all start immediately. Track C depends on B (dispatch is only meaningful once the ready queue exists). Track E integrates outputs from all other tracks — it is the last to complete.

### 2.2 Build order within tracks

**Track A (Estimation):**

```
Schema additions (estimate field)
    ↓
Scale validation and soft limits
    ↓
Rollup computation (Feature → Epic)
    ↓
MCP tools (estimate_set, estimate_query, reference management)
    ↓
CLI commands
    ↓
Tests + round-trip verification
```

**Track B (Dependency Enforcement):**

```
Transition validator extension (queued → ready gate)
    ↓
work_queue promotion + sort + filter
    ↓
dependency_status tool
    ↓
CLI commands
    ↓
Tests
```

**Track C (Task Dispatch and Completion):**

```
Task schema additions (dispatch fields + completion_summary)
    ↓
dispatch_task (atomic claim + context assembly)
    ↓
complete_task (transition + knowledge batch)
    ↓
Checkpoint record storage + serialisation
    ↓
Checkpoint MCP tools (create, respond, get, list)
    ↓
CLI commands
    ↓
Tests
```

**Track D (Context Assembly Enhancements):**

```
Trimming tracking instrumentation in context_assemble
    ↓
trimmed response field
    ↓
orchestration_context parameter + ephemeral injection
    ↓
Tests + backwards compatibility verification
```

**Track E (Health Check Extensions):**

```
Dependency cycle detection algorithm
    ↓
Stalled dispatch detection
    ↓
Estimation coverage detection
    ↓
Integration into health_check output + config
    ↓
Tests
```

### 2.3 Parallelism opportunities

| Group | Tracks | Notes |
|-------|--------|-------|
| Group 1 | A, B, D | All independent; start simultaneously |
| Group 2 | C | Starts after B is complete; D can be integrated concurrently |
| Group 3 | E | Starts after A, B, C are complete |

With two agents: Agent 1 runs A → B → C sequentially; Agent 2 runs D then assists with C or E.

With one agent: A → B → D → C → E. Start with A (purely additive, no risk) to validate the schema extension pattern before tackling more complex tracks.

---

## 3. Track Breakdown

### Track A: Estimation

**Goal:** Implement the complete `estimation-and-progress-design.md` design — story point estimates on all entity types, computed rollup queries, and AI calibration reference management.

| Task | Description | Estimate |
|------|-------------|----------|
| A.1 | Add `estimate` field to Task, Feature, Epic, Bug models (`internal/model/entities.go`) | S |
| A.2 | Extend storage field order for all four entity types (`internal/storage/entity_store.go`) | S |
| A.3 | Implement Modified Fibonacci scale constant set and validation function | S |
| A.4 | Implement per-entity-type soft limit warning logic | S |
| A.5 | Implement Feature effective estimate rollup (§6.4 rules, excluding `not-planned`/`duplicate`) | M |
| A.6 | Implement Feature progress rollup (sum of `done` task estimates) | S |
| A.7 | Implement Epic effective estimate rollup (via Feature rollup) | M |
| A.8 | Implement Epic progress rollup | S |
| A.9 | Implement `estimate_set` MCP tool (validate scale, store, emit soft limit warning, include references) | M |
| A.10 | Implement `estimate_query` MCP tool (rollup query, delta computation) | M |
| A.11 | Implement reference example contribution via knowledge pipeline (`ttl_days: 0`, tag `estimation-reference`) | S |
| A.12 | Implement `estimate_reference_add` MCP tool | S |
| A.13 | Implement `estimate_reference_remove` MCP tool | S |
| A.14 | Implement CLI commands: `kbz estimate set`, `kbz estimate show`, `kbz estimate reference add/remove/list` | M |
| A.15 | Write tests: scale validation, soft limit warnings, rollup rules (all exclusion cases) | M |
| A.16 | Write round-trip tests for all four entity types with `estimate` field | S |

**Dependencies:** None (foundational — purely additive to existing entities)

**Key implementation notes:**
- Rollup is computed on read; never stored. Do not add a cache layer.
- `not-planned` and `duplicate` tasks are excluded from both totals and progress.
- Feature delta = task_total − original estimate (not the reverse). Show with sign.
- `ttl_days: 0` exempts reference examples from TTL pruning. The knowledge contribution pipeline must accept and store this value.

**Verification:**
- Unit tests for scale validation (valid values pass, non-scale values rejected)
- Unit tests for all rollup rules, including the three-way Feature effective estimate logic
- Unit tests for `not-planned`/`duplicate` exclusion
- Round-trip tests for each entity type: write entity with `estimate` → read → write → compare
- `estimate_query` returns correct delta when both original and computed totals present; omits delta when either is absent

---

### Track B: Dependency Enforcement

**Goal:** Gate the `queued → ready` transition on resolved dependencies and expose the ready queue as the orchestrator's primary dispatch signal.

| Task | Description | Estimate |
|------|-------------|----------|
| B.1 | Extend `validate.ValidateTransition` to enforce dependency rule on `queued → ready` | M |
| B.2 | Implement descriptive error message listing blocking dependency task IDs and their statuses | S |
| B.3 | Write tests for transition enforcement (blocking deps, resolved deps, no deps, cross-feature deps) | M |
| B.4 | Implement `work_queue` promotion logic: load `queued` tasks, check deps, transition eligible to `ready` | M |
| B.5 | Implement `work_queue` sort: estimate ASC (null last), age DESC, task ID lexicographic tie-break | S |
| B.6 | Implement `work_queue` role filter (optional `role` parameter filters by parent feature profile) | S |
| B.7 | Implement `work_queue` response: `queue` array, `promoted_count`, `total_queued` | S |
| B.8 | Implement `work_queue` MCP tool (wires B.4–B.7) | M |
| B.9 | Implement `dependency_status` MCP tool: loads task, resolves all `depends_on` entries, returns blocking/resolved flags | S |
| B.10 | Implement CLI commands: `kbz queue [--role <profile>]`, `kbz task deps <id>` | S |
| B.11 | Write tests for `work_queue`: promotion side effects, sort order, role filter, `promoted_count` accuracy | M |
| B.12 | Write tests for `dependency_status`: blocking deps, resolved deps, mixed state | S |

**Dependencies:** None (extends existing `validate` package)

**Key implementation notes:**
- `work_queue` is a write-through query. It modifies task state as a side effect. Document this clearly in the MCP tool description.
- The `StatusTransitionHook` fires on every status transition, including `queued → ready`. The current `WorktreeTransitionHook` only acts on `active` transitions, so `work_queue`-triggered promotions are safe — they fire the hook but produce no side effects. Verify this in tests.
- Terminal states that satisfy a dependency: `done`, `not-planned`, `duplicate`.
- A task with an empty `depends_on` list qualifies for promotion immediately.

**Verification:**
- Transition enforcement: attempting `queued → ready` on a task with unresolved deps returns an error naming the blocking tasks
- Transition enforcement: task with all deps terminal can transition `queued → ready`
- `work_queue`: promotes exactly the qualifying `queued` tasks; leaves blocked tasks in `queued`
- `work_queue`: sort order is deterministic across repeated calls
- `work_queue`: `promoted_count` reflects actual promotions in this call
- `StatusTransitionHook` fires on promotions but produces no unintended side effects

---

### Track C: Task Dispatch and Completion

**Goal:** Implement the atomic dispatch/complete loop and the human checkpoint mechanism.

| Task | Description | Estimate |
|------|-------------|----------|
| C.1 | Add `claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by`, `completion_summary` to Task model | S |
| C.2 | Extend Task storage field order (§11.1 of spec) | S |
| C.3 | Implement `dispatch_task` MCP tool: load task, verify `ready` status, transition to `active`, set dispatch fields, call `context_assemble`, return packet | L |
| C.4 | Implement "already claimed" error path in `dispatch_task` (task in `active` state) | S |
| C.5 | Implement `complete_task` MCP tool: verify `active` status, transition, set `completed`, store `completion_summary` | M |
| C.6 | Implement knowledge batch processing in `complete_task`: iterate `knowledge_entries`, call contribution pipeline per entry, collect accepted/rejected results | M |
| C.7 | Define checkpoint record type (`internal/...` package — see implementation notes) | S |
| C.8 | Implement checkpoint YAML serialisation with explicit `null` values for `responded_at` and `response` | S |
| C.9 | Implement checkpoint store: create, get, list, update-to-responded | M |
| C.10 | Implement `human_checkpoint` MCP tool | S |
| C.11 | Implement `human_checkpoint_respond` MCP tool (including "already responded" error) | S |
| C.12 | Implement `human_checkpoint_get` MCP tool | S |
| C.13 | Implement `human_checkpoint_list` MCP tool (with optional `status` filter) | S |
| C.14 | Implement CLI commands: `kbz dispatch <id> --role <profile> --by <identity>`, `kbz task complete <id>` | M |
| C.15 | Implement CLI commands: `kbz checkpoint list/show/respond` | S |
| C.16 | Write tests for `dispatch_task`: successful dispatch, already-claimed refusal, non-ready status error | M |
| C.17 | Write tests for `complete_task`: transition to done, transition to needs-review, knowledge batch (accepted, rejected, duplicate) | M |
| C.18 | Write tests for checkpoint lifecycle: create, get pending, respond, get responded, list with filter | M |
| C.19 | Round-trip tests for Task with dispatch fields and checkpoint records | S |

**Dependencies:** Track B (dispatch requires ready queue to be meaningful; `dispatch_task` requires tasks to be in `ready` status)

**Key implementation notes:**
- `dispatch_task` atomicity relies on Go's per-request serialisation in the single-process MCP server. Two concurrent requests to dispatch the same task will be serialised — the first write wins, the second reads `active` status and returns the "already claimed" error. No additional locking is required for Phase 4a.
- The checkpoint package can sit alongside the worktree package pattern: a lightweight `internal/checkpoint/` package with its own store type, not wired into the full entity machinery. Checkpoint IDs use the `CHK` prefix but are not registered in the prefix allocator — generate with `id.New()` directly and prepend `CHK-`.
- Checkpoint records require explicit `null` for `responded_at` and `response` fields (unlike other optional fields which use `omitempty`). The YAML serialiser needs a special case for checkpoint records.
- `complete_task` knowledge batch processing: errors on individual entries (duplicate, invalid) are collected and returned but do not fail the whole operation. The task transition proceeds regardless.
- `dispatch_task` calls `context_assemble` internally. Pass through the `orchestration_context` string (from Track D) and the `max_bytes` parameter. This means Track D should be integrated before or alongside Track C for the full dispatch response to include `trimmed`.

**Verification:**
- `dispatch_task` success: task transitions to `active`, all four dispatch fields set, context packet returned
- `dispatch_task` already-claimed: returns error with `dispatched_by` and `claimed_at` in message; task state unchanged
- `dispatch_task` non-ready status: returns clear error naming the actual status
- `complete_task` to `done`: task status = `done`, `completed` set, `completion_summary` stored
- `complete_task` to `needs-review`: task status = `needs-review`
- `complete_task` knowledge batch: valid entries contributed, duplicates rejected with reason, task completes regardless
- Checkpoint create → list (pending) → respond → list (responded) → get (with response): full lifecycle
- `human_checkpoint_respond` on already-responded checkpoint: returns error

---

### Track D: Context Assembly Enhancements

**Goal:** Make context trimming visible and allow orchestrators to inject ephemeral handoff notes into dispatched context packets.

| Task | Description | Estimate |
|------|-------------|----------|
| D.1 | Instrument the existing trimming logic in `context_assemble` to capture cut entries as they are dropped | M |
| D.2 | Define `trimmed` entry struct: `entry_id`, `type`, `topic`, `tier`, `size_bytes` | S |
| D.3 | Add `trimmed` array to `context_assemble` response (empty array when nothing cut) | S |
| D.4 | Add `orchestration_context` string parameter to `context_assemble` MCP tool | S |
| D.5 | Implement ephemeral Tier 3 entry creation from `orchestration_context` (built in-memory, not written to store) | M |
| D.6 | Integrate ephemeral entry into context assembly (counts toward byte budget; trimmed if over budget) | S |
| D.7 | Write tests for `trimmed` field: entries cut in correct order, correct metadata in each trimmed entry | M |
| D.8 | Write tests for `orchestration_context`: content appears in packet, entry not written to knowledge store | M |
| D.9 | Verify backwards compatibility: existing `context_assemble` callers without new parameters are unaffected | S |

**Dependencies:** None (extends existing `context_assemble` infrastructure)

**Key implementation notes:**
- The trimming logic currently drops entries silently. The change is to record each dropped entry before removing it from the packet. The ordering of the `trimmed` array must reflect the order of trimming: lowest priority first (Tier 3 lowest confidence first, then Tier 2, then design context).
- The ephemeral Tier 3 entry from `orchestration_context` must never be written to `.kbz/state/knowledge/`. It is assembled into the packet in memory and discarded after the response is returned.
- `trimmed` must be present in all `context_assemble` responses, as an empty array when nothing is cut. Omitting it when empty is acceptable, but an empty array is preferred for consistent response shape.
- Track D is independent but should be integrated before Track C's `dispatch_task` is finalised, since `dispatch_task` embeds a `context_assemble` call and should return the `trimmed` field in its response.

**Verification:**
- `trimmed` is an empty array (or absent) when byte budget is not exceeded
- `trimmed` entries appear in order of trimming priority (lowest first)
- Each `trimmed` entry has correct `entry_id`, `type`, `topic`, `tier`, `size_bytes`
- `orchestration_context` string appears in the assembled packet
- `orchestration_context` content is not present in `.kbz/state/knowledge/` after assembly
- Calling `context_assemble` without `orchestration_context` parameter produces identical output to the current behaviour

---

### Track E: Health Check Extensions

**Goal:** Add three new health check categories covering dependency cycles, stalled dispatches, and estimation coverage.

| Task | Description | Estimate |
|------|-------------|----------|
| E.1 | Implement dependency cycle detection: DFS traversal of `depends_on` chains across all tasks | M |
| E.2 | Implement `dependency_cycles` health check category (error severity) | S |
| E.3 | Implement stalled dispatch detection: load `active` tasks, check `dispatched_at` age, check git activity on parent worktree branch | M |
| E.4 | Implement `stalled_dispatches` health check category (warning severity) | S |
| E.5 | Implement estimation coverage check: load features in `active`+, check child task estimates | S |
| E.6 | Implement `estimation_coverage` health check category (warning severity) | S |
| E.7 | Integrate three new categories into `health_check` MCP tool output | S |
| E.8 | Implement `dispatch.stall_threshold_days` configuration (default: 3; 0 disables stalled check) | S |
| E.9 | Implement `estimation.coverage_warn_at_status` configuration (default: `active`) | S |
| E.10 | Write tests for cycle detection: direct cycle, transitive cycle, no cycle, self-reference | M |
| E.11 | Write tests for stalled dispatch: within threshold (no flag), past threshold (flag), no worktree (time-only check) | M |
| E.12 | Write tests for estimation coverage: feature with estimates (no flag), feature without estimates (flag), draft feature (no flag) | S |

**Dependencies:** Track A (for estimation coverage), Track B (for cycle detection over same `depends_on` data), Track C (for stalled dispatch — needs `dispatched_at` on task schema)

**Key implementation notes:**
- Cycle detection uses DFS with a visited set and a recursion stack. A node in the recursion stack that is revisited indicates a cycle. Report all task IDs in each cycle, not just the detecting pair.
- The stalled dispatch check uses the existing git branch tracking infrastructure from Phase 3 (`internal/git/`). If the parent feature has no associated worktree, skip the git activity check and fall back to the time threshold alone.
- The stall threshold check should be skipped entirely (no issues reported) when `dispatch.stall_threshold_days` is 0.
- Estimation coverage uses the `active`-or-later threshold from config. The Feature lifecycle order (from `validate/lifecycle.go`) determines what counts as "active or later". Features in `draft`, `proposed`, or `ready` are excluded.

**Verification:**
- Cycle check: simple A→B→A cycle reported as error with both task IDs
- Cycle check: no cycles in a clean project produces zero issues
- Stalled dispatch: task dispatched 4 days ago with no git activity flagged (threshold 3); task dispatched 2 days ago not flagged
- Stalled dispatch: check disabled when `stall_threshold_days: 0`
- Estimation coverage: feature in `active` status with all tasks unestimated produces warning
- Estimation coverage: feature in `draft` status not flagged regardless of task estimates
- All three categories present in health check output with correct category names and severities

---

## 4. Dependency Graph

```
                    ┌─────────────────────────────────────────────────┐
                    │   Track A: Estimation                           │
                    │   (start immediately — no dependencies)         │
                    └─────────────────────────────────────────────────┘
                                          │
         ┌────────────────────────────────┼───────────────────────────┐
         │                               │                            │
         ▼                               ▼                            │
┌─────────────────┐             ┌─────────────────┐                   │
│  Track B:       │             │  Track D:       │                   │
│  Dependency     │             │  Context        │                   │
│  Enforcement    │             │  Assembly       │                   │
│  (independent)  │             │  (independent)  │                   │
└────────┬────────┘             └────────┬────────┘                   │
         │                               │                            │
         └───────────────┬───────────────┘                            │
                         │                                            │
                         ▼                                            │
                ┌─────────────────┐                                   │
                │  Track C:       │                                   │
                │  Dispatch and   │                                   │
                │  Completion     │                                   │
                └────────┬────────┘                                   │
                         │                                            │
                         └────────────────────────────────────────────┘
                                          │
                                          ▼
                              ┌───────────────────────┐
                              │  Track E:             │
                              │  Health Check Exts    │
                              │  (integrates all)     │
                              └───────────────────────┘
```

---

## 5. Effort Estimates

### 5.1 Size definitions

| Size | Effort | Description |
|------|--------|-------------|
| S | 1–2 hours | Simple, well-defined; follows existing patterns |
| M | 2–4 hours | Moderate complexity; some design or edge cases |
| L | 4–8 hours | Complex; multiple moving parts or careful atomicity requirements |

### 5.2 Track estimates

| Track | Tasks | S  | M  | L | Est. Total  |
|-------|-------|----|----|---|-------------|
| A: Estimation             | 16 | 9  | 6  | 0 | 21–33 hrs  |
| B: Dependency Enforcement | 12 | 5  | 6  | 0 | 17–27 hrs  |
| C: Dispatch and Completion| 19 | 9  | 8  | 1 | 26–42 hrs  |
| D: Context Assembly       |  9 | 3  | 5  | 0 | 13–20 hrs  |
| E: Health Check Extensions| 12 | 6  | 5  | 0 | 16–24 hrs  |
| **Total**                 | **68** | **32** | **30** | **1** | **93–146 hrs** |

### 5.3 Phase estimate

At 6–8 productive hours per day:

- **Single agent:** 12–24 days
- **Two agents (A+D parallel with B, then C, then E):** 7–14 days

This is smaller than Phase 3 (136–218 hrs), which is consistent with the scope doc's finding that Phase 4 adds approximately five new tools and the heavy lifting was done in Phases 1–3.

---

## 6. Implementation Order

### 6.1 Recommended sequence (single agent)

| Step | Track | Rationale |
|------|-------|-----------|
| 1 | A (Estimation) | Purely additive, no dependencies, validates schema extension pattern |
| 2 | B (Dependency Enforcement) | Independent, foundational for dispatch |
| 3 | D (Context Assembly) | Short track; complete before integrating into dispatch |
| 4 | C (Dispatch and Completion) | Depends on B; integrates D's context_assemble extensions |
| 5 | E (Health Checks) | Depends on all others; final integration |
| 6 | Validation workload | Run Phase 4a tools on a real task before declaring complete |

### 6.2 Parallel execution (two agents)

**Agent 1: Core orchestration path (A → B → C)**

| Step | Track | Notes |
|------|-------|-------|
| 1 | A (Estimation) | Build schema, rollup, tools |
| 2 | B (Dependency Enforcement) | Build enforcement, work_queue |
| 3 | C (Dispatch and Completion) | Build dispatch loop and checkpoints |

**Agent 2: Supporting tracks (D → E)**

| Step | Track | Notes |
|------|-------|-------|
| 1 | D (Context Assembly) | Complete quickly; share output for integration into C |
| 2 | E (Health Checks) | Begin once A and B are done; wait for C if needed for stalled dispatch |

**Integration point:** Agent 2 completes Track D before Agent 1 finalises `dispatch_task` (Track C.3), so that `trimmed` and `orchestration_context` are available in the dispatch response from the start.

---

## 7. Verification Strategy

### 7.1 Test coverage requirements

| Category | Requirement |
|----------|-------------|
| Unit tests | All business logic: scale validation, rollup rules, enforcement rule, cycle detection |
| Integration tests | MCP tool end-to-end: dispatch loop, checkpoint lifecycle, work_queue promotion |
| Round-trip tests | All new entity fields and checkpoint records: write → read → write → compare |
| Backwards compatibility | `context_assemble` without new parameters produces unchanged output |
| Race detector | `go test -race ./...` passes; pay particular attention to `dispatch_task` |

### 7.2 Verification checkpoints

| Checkpoint | Tracks | Verification |
|------------|--------|--------------|
| CP1: Estimation basics | A | estimate field stored/retrieved; scale validated; soft limit warns |
| CP2: Rollup correctness | A | Feature and Epic rollup rules, including exclusions and delta |
| CP3: Reference examples | A | Reference contributed, TTL-exempt, presented at estimation time |
| CP4: Dependency gate | B | Transition blocked with unmet deps; passes with terminal deps |
| CP5: Work queue | B | Promotes eligible tasks; sorted correctly; role filter works |
| CP6: Trimming visibility | D | trimmed array populated on budget overflow; empty when not needed |
| CP7: Orchestration context | D | Ephemeral entry in packet; not in knowledge store |
| CP8: Dispatch atomicity | C | Second dispatch on same task returns "already claimed" error |
| CP9: Complete loop | C | complete_task transitions task, contributes knowledge, handles duplicates |
| CP10: Checkpoint lifecycle | C | Create → list pending → respond → get response → list responded |
| CP11: Health integration | E | All three new categories in health_check output with correct severities |
| CP12: Full orchestration loop | All | work_queue → dispatch_task → complete_task → work_queue; health_check clean |

### 7.3 Acceptance criteria verification

Each of the 44 acceptance criteria from the specification (§15) must have at least one test. Test functions should reference the criterion section: `// Verifies §15.2: queued→ready blocked when depends_on tasks are not terminal`.

The Phase 4a validation workload (§17.7 of spec) is a human-run integration test that verifies the full loop end-to-end. It is not automated but must be completed before Phase 4b begins.

---

## 8. Risk Mitigations

### 8.1 dispatch_task atomicity assumption

**Risk:** If the MCP server is ever run as multiple processes (e.g. during testing with parallel test runners), the single-process atomicity assumption breaks and two orchestrators could claim the same task.

**Mitigation:**
- Document the single-process assumption clearly in code comments on `dispatch_task`.
- In tests, never call `dispatch_task` from multiple goroutines on the same task — test the "already claimed" path sequentially.
- The race detector (`go test -race`) will catch data races if the assumption is violated accidentally.
- If multi-process deployment becomes a requirement in a future phase, introduce file-level locking (the pattern already exists in `internal/fsutil/` from Phase 1).

### 8.2 work_queue write-through mutation

**Risk:** `work_queue` modifies task state (promoting `queued → ready`) as a side effect of a query. This is unexpected for callers who assume query tools are pure reads. It also means frequent `work_queue` calls could produce repeated (no-op) promotion attempts.

**Mitigation:**
- The tool description must explicitly state the promotion side effect.
- Promotion attempts on tasks that are already in `ready` status (or any non-`queued` status) are silently skipped — not an error. The enforcement rule rejects invalid transitions; `work_queue` just attempts the transition for each qualifying candidate.
- In tests, verify that calling `work_queue` twice in succession does not double-promote or produce errors.

### 8.3 Estimation rollup performance

**Risk:** Rollup queries traverse all child entities on every call. For large projects, this could become slow.

**Mitigation:**
- Phase 4a does not add caching. The design principle is compute on read (§2.1 of estimation design).
- For typical project sizes (tens of features, hundreds of tasks), in-memory traversal is fast enough.
- If benchmarks show a problem, cache in the derived SQLite layer in a later phase. Do not pre-optimise.

### 8.4 Checkpoint explicit nulls

**Risk:** The YAML serialiser uses `omitempty` for all optional fields throughout the codebase. Checkpoint records require explicit `null` values for `responded_at` and `response` (to make unresponded state explicit). Adding a per-type exception to the serialiser is a behaviour change that could affect other types if done carelessly.

**Mitigation:**
- Implement explicit nulls for checkpoint records specifically, either via a custom marshaller on the checkpoint type or by using a wrapper field type (e.g. `*string` with explicit nil serialisation).
- Round-trip tests for checkpoints must verify that `responded_at: null` and `response: null` appear in the serialised output, not absent fields.
- Verify in tests that existing entity types are unaffected by this change.

### 8.5 Track D integration timing

**Risk:** If Track D (Context Assembly Enhancements) is completed after `dispatch_task` is already built and tested, integrating `trimmed` and `orchestration_context` into the dispatch response requires revisiting Track C's implementation and tests.

**Mitigation:**
- In single-agent execution, implement Track D (step 3) before Track C (step 4) per the recommended sequence.
- In parallel execution, Agent 2 completes Track D before Agent 1 finalises `dispatch_task` (C.3). The integration point is explicit in §6.2.
- If the ordering is not honoured, `dispatch_task` can be implemented without the new context fields first and extended afterwards — the additions are backwards-compatible.

### 8.6 Scope creep into Phase 4b

**Risk:** While building dispatch and completion, temptation to add decomposition hints, worker review stubs, or automatic unblocking — all Phase 4b features.

**Mitigation:**
- The spec is the contract. Any feature not in §3.1 of the Phase 4a spec requires explicit approval.
- Track E's dependency cycle detection and stalled dispatch detection are health checks, not automatic remediation. No automatic status transitions in Phase 4a beyond those specified.
- Automatic dependency unblocking (Phase 4b's `StatusTransitionHook` extension) must not be added to `complete_task` in Phase 4a.

---

## 9. Definition of Done

A track is complete when:

1. All tasks are implemented
2. All unit and integration tests pass
3. Round-trip tests pass for all new YAML records and entity fields
4. `go test -race ./...` passes
5. `go vet ./...` is clean
6. `go fmt ./...` applied
7. MCP tools follow existing patterns: unknown parameters rejected, missing required parameters produce clear errors, entity-not-found returns a structured error not a panic
8. CLI commands are consistent with existing UX patterns
9. Relevant acceptance criteria (§15 of spec) have passing tests

Phase 4a is complete when:

1. All five tracks are complete
2. All 44 acceptance criteria (§15 of spec) are verified by passing tests
3. The full orchestration loop integration test passes (CP12)
4. No blocking health check errors in a clean project instance
5. The Phase 4a validation workload has been run by a human and confirmed (§17.7 of spec)
6. `AGENTS.md` and `work/plan/phase-4a-progress.md` are updated to reflect completion

---

## 10. Open Items

### 10.1 Questions to resolve during implementation

| Question | Disposition |
|----------|-------------|
| Checkpoint package location: `internal/checkpoint/` alongside `internal/worktree/`, or inside `internal/service/`? | Resolve in Track C.7; prefer `internal/checkpoint/` for symmetry with worktree |
| `estimate_set` on a Feature that already has child task estimates: warn or silent? | Warn: inform the caller that original estimate diverges from task total |
| `work_queue` when zero tasks are in `queued` or `ready`: return empty queue (not error) | Implement empty queue as valid response; not an error |
| Stalled dispatch check: should it check git activity on the branch or on the worktree path? | Branch activity via `git log` since `dispatched_at` on the worktree branch |
| `complete_task` when task has no `dispatched_by` (manually transitioned to active): allow or reject? | Allow — `dispatched_by` is informational; completing a manually-activated task is valid |

### 10.2 Punch list (known gaps to fix before validation)

| # | Gap | Location | Fix |
|---|-----|----------|-----|
| PL-1 | `update status --type plan` fails — `parseRecordIdentity` has no `plan` case, so the generic `UpdateStatus` path cannot resolve Plan filenames (`{id}.yaml`, no slug suffix) | `internal/service/entities.go` `parseRecordIdentity` | Add a `plan` case that calls `model.ParsePlanID` to extract the slug, matching how `entityFileName` writes Plan files |
| PL-2 | All Phase 4a CLI commands unimplemented — `kbz estimate set/show/reference`, `kbz queue`, `kbz task deps`, `kbz dispatch`, `kbz task complete`, `kbz checkpoint list/show/respond` (spec §13.1–13.4) | `cmd/kanbanzai/main.go` | Implement each command group following existing CLI patterns; MCP tools are complete and provide the underlying service calls |
| PL-3 | No MCP-level integration tests for any Phase 4a tool — `dispatch_tools.go`, `estimation_tools.go`, and `queue_tools.go` have zero test functions; `TestServer_ListTools` in `server_test.go` does not include Phase 4a tool names in its expected list | `internal/mcp/` | Add integration tests for the happy path and key error paths of each Phase 4a tool; update `TestServer_ListTools` expected tool list to include all Phase 4a tool names |
| PL-4 | No `phase-4a-progress.md` tracking document; all 44 spec §15 acceptance criteria remain unchecked | `work/plan/` | Create `phase-4a-progress.md` modelled on `phase-3-progress.md`; tick criteria against the existing implementation |
| PL-5 | `AGENTS.md` not updated to reflect Phase 4a status — still reads "Phase 3 is complete" with no mention of Phase 4a | `AGENTS.md` | Update Phase Status section once PL-2–PL-4 are resolved and the §17.7 validation workload is complete |

### 10.3 Potential Phase 4a.1 scope (defer if needed)

If any track falls behind estimates, these items can be deferred without breaking the core orchestration loop:

- `estimate_reference_remove` (A.13) — add/list is sufficient for calibration; remove is convenience
- Role filter in `work_queue` (B.6) — return all ready tasks; filter is a UX improvement
- `kbz dispatch` CLI command (C.14) — MCP tool is sufficient for agent use; CLI is human convenience
- `estimation.coverage_warn_at_status` config (E.9) — hard-code `active` as the threshold for Phase 4a

---

## 11. Summary

Phase 4a implements orchestration support through 5 tracks and 68 tasks.

**Key deliverables:**

- Story point estimates on all entity types with computed rollups and delta surfacing
- Dependency-gated `queued → ready` transition with promoted work queue
- Atomic dispatch (claim + context assembly) and knowledge-contributing completion
- Human checkpoint mechanism for structured orchestrator escalation
- Context trimming visibility and orchestration context injection
- Three new health check categories: cycles, stalled dispatches, estimation coverage

**Estimated effort:** 93–146 hours (7–14 days with two agents)

**Verification:** 12 checkpoints, 44 acceptance criteria, race detector on all tests

**Risks mitigated:** atomicity assumption, write-through query side effects, rollup performance, checkpoint serialisation edge case, integration ordering

**Gate for Phase 4b:** Phase 4a complete, validated on at least one real workload, no blocking health errors. Phase 4b specification written under self-management using Phase 4a tooling.

**Implementation sequence:** A (estimation) → B (dependency enforcement) → D (context assembly) → C (dispatch and completion) → E (health checks) → validation workload.