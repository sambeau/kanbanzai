# Orchestrate-Development Examples

Examples of correct and incorrect orchestration patterns. Linked from
`.kbz/skills/orchestrate-development/SKILL.md`.

---

## BAD: Serial dispatch with full output retention

```
Dispatch TASK-301 → wait → receive full output (diffs, tool logs, reasoning traces)
→ keep full output in context → Dispatch TASK-302 → wait → receive full output
→ keep full output in context → ...
→ After 3 tasks: context saturated (85% utilisation)
→ TASK-304 dispatched but quality visibly degrades: wrong file scope, missed logic
→ orchestrator doesn't notice because it's managing 4 tasks' worth of context
```

**Problem:** Serial dispatch wastes parallel capacity. Full output retention saturates context
quickly. By task 4 the orchestration quality has degraded.

---

## BAD: Dispatching with unmet dependencies

```
Ready frontier: TASK-301, TASK-302, TASK-303 (no dependencies)
Dispatch all: TASK-301, TASK-302, TASK-303
TASK-301 done. TASK-302 done. TASK-303 done.
Ready frontier: TASK-304 (depends on TASK-301), TASK-305 (no deps)
Dispatch TASK-304 and TASK-305 in parallel ✓
... (good so far)

Next ready frontier: TASK-306 (depends on TASK-305)
Also in ready frontier: TASK-307 (depends on TASK-999 — NOT DONE!)
→ Dispatch both → TASK-307 fails because TASK-999 doesn't exist yet
```

**Problem:** TASK-307 was dispatched with an unmet dependency (TASK-999). Verify each task's
`depends_on` entries are all done before dispatching.

---

## GOOD: Dependency-respecting parallel dispatch with compaction

```
Feature: FEAT-055 — Webhook delivery system
Phase 1: Read dev-plan. 6 tasks total.

Cycle 1 — Ready frontier: TASK-301 (data model), TASK-302 (config schema)
  No shared files — dispatched in parallel.
  TASK-301 done: Added webhook event model with 4 fields, migration included.
  TASK-302 done: Config schema with retry policy fields, validated by tests.
  [Full outputs discarded, summaries retained]

Cycle 2 — Ready frontier: TASK-303 (dispatcher, depends on 301+302),
  TASK-304 (delivery log, depends on 301)
  No shared files — dispatched in parallel.
  TASK-303 done: Dispatcher with 3 delivery backends. 93% coverage.
  TASK-304 done: Delivery log with query API. 89% coverage.
  [Full outputs discarded, summaries retained]
  Context check: 48% — below threshold, continue.

Cycle 3 — Ready frontier: TASK-305 (retry logic, depends on 303+304)
  TASK-306 (metrics, depends on 303+304)
  No shared files — dispatched in parallel.
  TASK-305 FAILED: Retry policy is unbounded — no max_retries config.
    → Failure classified as recoverable (wrong config path in handoff).
    → Re-dispatched with corrected handoff.
    → TASK-305 done: Retry with exponential backoff, configurable max_retries.
  TASK-306 done: Metrics for webhook delivery. 4 metric types.
  [Full outputs discarded, summaries retained]

Close-out: All 6 tasks done. Feature completion summary written.
           Knowledge curation: confirmed 3 entries, promoted 1.
           Feature transitioned to reviewing. Branch merged and deleted.
```
