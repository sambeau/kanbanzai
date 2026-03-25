# Phase 4a Progress

**Last updated:** 2026-03-28

**Status:** Core implementation complete — all 5 tracks implemented, all MCP tools registered and working. CLI commands (PL-2) and MCP integration tests (PL-3) remain as punch list items.

**Purpose:** Track implementation status of Phase 4a deliverables (Orchestration MVP) against the Phase 4a specification (§15 acceptance criteria), and record remaining work in the punch list.

**Related documents:**

- `work/spec/phase-4a-specification.md` — binding specification
- `work/plan/phase-4a-implementation-plan.md` — implementation plan and work breakdown
- `work/plan/phase-4-scope.md` — Phase 4 scope and planning
- `work/plan/phase-4-decision-log.md` — design decisions
- `work/plan/phase-3-progress.md` — Phase 3 completion status (predecessor)

---

## 1. Implementation Status Summary

All 5 tracks (A–E) are implemented. All MCP tools are registered and working. All tests pass with race detector enabled (`go test -race ./...`). `go vet` is clean.

CLI commands (spec §13) are not yet implemented (PL-2). MCP-level integration tests are not yet written (PL-3).

| Track | Name | Status | Key Files |
|-------|------|--------|-----------|
| A | Estimation | ✅ Complete | `internal/mcp/estimation_tools.go` |
| B | Dependency enforcement | ✅ Complete | `internal/mcp/queue_tools.go` |
| C | Task dispatch and completion | ✅ Complete | `internal/mcp/dispatch_tools.go`, `internal/checkpoint/` |
| D | Context assembly enhancements | ✅ Complete | `internal/context/assemble.go`, `internal/mcp/context_tools.go` |
| E | Health check extensions | ✅ Complete | `internal/health/phase4a.go`, `internal/mcp/phase4a_health.go` |

---

## 2. Acceptance Criteria Status

Tracking against spec §15 acceptance criteria.

### §15.1 Estimation — ✅ Met

- [x] `estimate` field is accepted and stored on Task, Feature, Epic, and Bug entities
- [x] Modified Fibonacci values (0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100) are accepted; all other values are rejected with a validation error listing valid values
- [x] Soft limit warnings are emitted for Task and Bug estimates above 13, and for Feature and Epic estimates above 100; the operation succeeds regardless
- [x] Feature effective estimate rollup follows §6.4 rules: task total if tasks with estimates exist, own estimate otherwise
- [x] Feature progress rollup correctly sums estimates from `done` child tasks only
- [x] Epic effective estimate and progress roll up from Features following §6.4 rules
- [x] `not-planned` and `duplicate` tasks are excluded from all rollup totals and progress
- [x] `estimate_query` returns entity, rollup totals, progress, and delta where applicable
- [x] Delta is shown when both original estimate and computed total are present; omitted when either is absent
- [x] Reference examples are stored as Tier 2 KnowledgeEntry records with tag `estimation-reference` and `ttl_days: 0`
- [x] `estimate_set` response includes current reference examples and scale definitions
- [x] `estimate_reference_remove` correctly retires the reference entry
- [x] Round-trip serialisation (write → read → write → compare) produces identical output for all entities with `estimate` field

### §15.2 Dependency enforcement — ✅ Met

- [x] `queued → ready` transition is blocked if any `depends_on` task is not in `done`, `not-planned`, or `duplicate` state
- [x] The error message names the blocking dependency tasks and their current status
- [x] `work_queue` promotes eligible `queued` tasks to `ready` before returning results
- [x] `work_queue` returns only tasks in `ready` status
- [x] `work_queue` result is ordered: estimate ascending (null last), then age descending
- [x] `work_queue` `role` filter correctly limits results to tasks whose parent feature matches the profile
- [x] `work_queue` includes `promoted_count` and `total_queued` in its response
- [x] `dependency_status` shows all `depends_on` entries with their current status and blocking flag
- [x] A task with no `depends_on` entries is immediately eligible for `queued → ready` promotion

### §15.3 Dispatch and completion — ✅ Met

- [x] `dispatch_task` requires task in `ready` status; returns a clear error if status is `active` (with claimed_at and dispatched_by in the error) or any other non-ready status
- [x] `dispatch_task` atomically transitions task to `active` and sets all four dispatch fields (`claimed_at`, `dispatched_to`, `dispatched_at`, `dispatched_by`) in a single write
- [x] A second `dispatch_task` call on the same task (already `active`) returns the "already claimed" error and does not modify the task
- [x] `dispatch_task` response includes the assembled context packet with all sections
- [x] `dispatch_task` response `context.trimmed` reflects entries that were cut
- [x] `complete_task` requires task in `active` status; returns a clear error otherwise
- [x] `complete_task` transitions task to `done` (default) or `needs-review` when `to_status: needs-review` is specified
- [x] `complete_task` sets `completed` timestamp on the task
- [x] `complete_task` stores `completion_summary` on the task
- [x] Each valid knowledge entry in `knowledge_entries` is processed through the knowledge contribution pipeline
- [x] Duplicate knowledge entries are rejected with a reason and do not block the overall completion
- [x] `complete_task` response lists accepted, rejected, and total knowledge contributions
- [x] `human_checkpoint` creates a CHK record with `status: pending`
- [x] `human_checkpoint_respond` transitions checkpoint to `status: responded` and records `response` and `responded_at`
- [x] `human_checkpoint_respond` on an already-responded checkpoint returns a clear error
- [x] `human_checkpoint_get` returns full checkpoint state including response when responded
- [x] `human_checkpoint_list` with `status: pending` returns only pending checkpoints
- [x] `human_checkpoint_list` with `status: responded` returns only responded checkpoints

### §15.4 Context assembly enhancements — ✅ Met

- [x] `context_assemble` response includes `trimmed` field when entries are cut; `trimmed` is an empty array when nothing is cut
- [x] Each entry in `trimmed` includes `entry_id`, `type`, `topic`, `tier` (for knowledge entries), and `size_bytes`
- [x] `trimmed` entries are ordered by the priority in which they were trimmed (lowest priority first)
- [x] `orchestration_context` parameter is accepted by `context_assemble`
- [x] When `orchestration_context` is provided, its content is included in the context packet as an ephemeral Tier 3 entry
- [x] The `orchestration_context` entry is not written to `.kbz/state/knowledge/`
- [x] Existing `context_assemble` callers without `orchestration_context` or `trimmed` expectations are unaffected

### §15.5 Health checks — ✅ Met

- [x] Dependency cycle health check detects and reports cycles with error severity
- [x] Dependency cycle report includes the task IDs that form the cycle
- [x] Stalled dispatch health check flags active tasks past the configured `stall_threshold_days` with no git activity
- [x] Stalled dispatch report includes `task_id`, `dispatched_at`, `dispatched_to`, and `days_stalled`
- [x] Stalled dispatch check is disabled when `stall_threshold_days: 0`
- [x] Estimation coverage check flags features in `active` or later status with no estimated child tasks
- [x] All three new health check categories appear in `health_check` output under their category names
- [x] Severity levels are correct: `dependency_cycles` = error, `stalled_dispatches` = warning, `estimation_coverage` = warning

### §15.6 Deterministic storage — ✅ Met

- [x] Task records with new fields serialise in the field order defined in §11.1
- [x] Feature, Epic, and Bug records with `estimate` field serialise with `estimate` after `status`
- [x] Checkpoint records serialise in the field order defined in §11.5
- [x] Round-trip (write → read → write → compare) produces identical output for all new record types and field additions
- [x] Checkpoint records with `null` values for `responded_at` and `response` serialise without omitting those fields (explicit nulls)

---

## 3. MCP Tools and CLI

### MCP Tools — ✅ All Phase 4a tools registered

| Category | Tools | Status |
|----------|-------|--------|
| Estimation | `estimate_set`, `estimate_query`, `estimate_reference_add`, `estimate_reference_remove` | ✅ |
| Queue | `work_queue`, `dependency_status` | ✅ |
| Dispatch | `dispatch_task`, `complete_task` | ✅ |
| Checkpoint | `human_checkpoint`, `human_checkpoint_respond`, `human_checkpoint_get`, `human_checkpoint_list` | ✅ |
| Health (Phase 4a) | `dependency_cycles`, `stalled_dispatches`, `estimation_coverage` (categories in `health_check`) | ✅ |
| Context (enhanced) | `context_assemble` (with `orchestration_context` and `trimmed`) | ✅ |

### CLI Commands — ❌ Not implemented (PL-2)

| Command | Subcommands | Status |
|---------|-------------|--------|
| `kbz estimate` | `set`, `show`, `reference add/remove/list` | ❌ |
| `kbz queue` | (default list) | ❌ |
| `kbz task deps` | `<id>` | ❌ |
| `kbz dispatch` | `<id>` | ❌ |
| `kbz task complete` | `<id>` | ❌ |
| `kbz checkpoint` | `list`, `show`, `respond` | ❌ |

---

## 4. Test Status

All tests pass: `go test -race ./... ✅` and `go vet ./... ✅`.

### Test coverage by package

| Package | Unit Tests | MCP Tool Tests | Rating |
|---------|-----------|----------------|--------|
| `checkpoint/` | ✅ Good | N/A | Good |
| `context/` (Phase 4a additions) | ✅ Good (trimming, orchestration_context) | N/A | Good |
| `health/` (Phase 4a additions) | ✅ Good | N/A | Good |
| `mcp/estimation_tools.go` | N/A | ❌ None (PL-3) | Weak |
| `mcp/queue_tools.go` | N/A | ❌ None (PL-3) | Weak |
| `mcp/dispatch_tools.go` | N/A | ❌ None (PL-3) | Weak |
| `mcp/phase4a_health.go` | N/A | ❌ None (PL-3) | Weak |

### Key test gaps (remaining)

- No MCP-level integration tests for any Phase 4a tool — `estimation_tools.go`, `queue_tools.go`, `dispatch_tools.go`, and `phase4a_health.go` have zero test functions (PL-3)
- `TestServer_ListTools` in `server_test.go` does not yet include Phase 4a tool names in its expected list (PL-3)

---

## 5. Punch List

Known gaps to resolve before Phase 4a can be declared fully complete. Tracked as PL-1 through PL-5 in the implementation plan (§10.2).

| # | Description | Status | Notes |
|---|-------------|--------|-------|
| PL-1 | `update status --type plan` fails — `parseRecordIdentity` has no `plan` case | ⬜ Open | Bug fix in `internal/service/entities.go` |
| PL-2 | All Phase 4a CLI commands unimplemented (spec §13.1–13.4) | ⬜ Open | MCP tools provide underlying service calls; CLI wiring needed |
| PL-3 | No MCP-level integration tests for Phase 4a tools; `TestServer_ListTools` not updated | ⬜ Open | Being addressed separately |
| PL-4 | Create `phase-4a-progress.md` | ✅ Done | This document |
| PL-5 | Update `AGENTS.md` to reflect Phase 4a status | ⬜ Open | Blocked on PL-1 through PL-3 |

---

## 6. Deferred to Phase 4b

The following capabilities are explicitly out of scope for Phase 4a:

- Automatic unblocking on task completion (Phase 4b Track A)
- Feature decomposition from design documents (Phase 4b Track B)
- Worker output review and rework cycles (Phase 4b Track C)
- Conflict domain analysis (Phase 4b Track D)
- Incident and RCA entities (Phase 4b Track E)
- Vertical slice guidance (Phase 4b Track F)