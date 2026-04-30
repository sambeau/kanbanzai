# Retrospective Report: B38 Implementation Session

**Date:** 2026-04-30
**Author:** sambeau (orchestrator)
**Scope:** B38 Plans and Batches — features 7, 8, 9, 10

---

## Summary

Implemented 3 features and resolved 1 bug across a single orchestration session:
- Built `kbz migrate` command (FEAT-01KQ7-JDT511BZ, 5 tasks)
- Added plan/batch prefix registries and ProjectConfig (FEAT-01KQ7-YQK6DDDA, 4 tasks)
- Verified state file migration was already complete (FEAT-01KQ7-YQKT04M7)
- Superseded Full Combined Migration (FEAT-01KQA-GMVQABXH) in favour of running `kbz migrate`

9 tasks completed, 3 features moved to reviewing, B38 is ready for batch review.

---

## What Went Well

### The orchestrator role and skill system worked
The automated context assembly from `next()` + `handoff()` provided exactly the right vocabulary, anti-patterns, and procedural guidance. The orchestrator-development skill's phased approach (read dev-plan → identify frontier → dispatch → monitor → close-out) was natural to follow. The vocabulary terms like "ready frontier" and "context compaction" were precise and useful.

### Feature 7 (kbz migrate) flowed smoothly
The task decomposition was well-scoped: T1 (pure functions) → T2 (report generation) → T3+T4 (execute + cleanup) → T5 (tests). Each task had a clear, testable deliverable with no circular dependencies. The interface contract between T1 and T2 (`resolveMigrateTarget` signature) was explicitly defined in the dev-plan, which eliminated ambiguity.

### Feature 9 (Config Schema) was fast to implement
Simple struct additions, validation, and tests. Four tasks, three commits, all tests passing. The spec was clear with numbered requirements and acceptance criteria that mapped directly to test cases.

### Automated dependency unblocking worked
When T1 completed, T2 automatically transitioned from `queued` to `ready`. Same for T3→T4 and T3→T5. No manual intervention needed. This kept the work flowing without orchestration overhead.

### The entity/status tools stayed reliable
Throughout the session, `entity(action: "get")`, `entity(action: "transition")`, `status(id: ...)`, and `finish(task_id: ...)` all behaved consistently. No stale state, no cache issues affecting task transitions.

---

## What Didn't Go Well

### Worktree file editing is painful
The `edit_file` tool doesn't work with worktrees — it only writes to the main project root. The alternative patterns (python3 -c with triple-escaping, heredocs in sh) are fragile. Triple-escaping Go source code through shell/python layers is error-prone and slow. I resorted to `git checkout --` to recover from a truncated file write. The `write_file` MCP tool with `entity_id` parameter worked reliably once I discovered it, but the workflow is undiscoverable.

### Decompose can't parse `AC-001 (REQ-001):` format
Feature 9's spec used `**AC-001 (REQ-001):**` — bold identifier with a REQ reference in parentheses. `decompose(action: propose)` only recognises three formats: `**AC-NN.**`, numbered lists, and checkboxes. This forced manual task creation via individual `entity(action: create)` calls. This is a known bug (BUG-01KPVGMMP56GC) but it bit us in a real workflow.

### Shell heredoc fails consistently in the terminal tool
The `sh` shell used by the terminal tool rejects heredoc syntax (`<< 'EOF'`). Every attempt to use heredocs for multi-line file writes failed. The only reliable worktree file-writing patterns were `python3 -c` (with painful escaping) or `write_file` with `entity_id`.

### Context budget was tight
At 30K bytes, the `next()` context assembly consistently used 99%+ of the configured budget. Knowledge entries, vocabulary, anti-patterns, procedure steps, and examples all compete for space. Several knowledge entries were trimmed from the context. The 30K budget feels too small for the orchestrator role, which needs the full procedure and vocabulary to function correctly.

### Feature 8 was stuck in limbo
Feature 8 (State File Migration) was in `developing` status with 0 tasks, no dev-plan, and a draft spec. The overrides log showed a convoluted history of being pushed through gates without implementation. The actual work was already done (batches/ directory exists, code reads it), but the entity state didn't reflect this. The feature couldn't be advanced past `reviewing` because it had no tasks to complete — it exists only as a tracking placeholder.

---

## What Should Improve

### Make worktree file writing first-class
The `write_file` MCP tool with `entity_id` parameter should be the documented, recommended way to write files in worktrees. Add it to the `implement-task` skill as the primary pattern. The `write_file` tool itself needs better discoverability — it's not in the default tool subset for the developing stage.

### Expand decompose's AC format recognition
The `**AC-NN (REQ-NN):**` format is common in specs that trace ACs back to requirements. It should be a first-class recognised format alongside `**AC-NN.**`. This is a one-line regex fix with outsized workflow impact.

### Increase context budget for orchestrator role
30K bytes is insufficient for the orchestrator role which needs procedure steps, vocabulary, anti-patterns, examples, and knowledge entries all in context simultaneously. Consider 40-45K for the orchestrator role specifically, or implement more aggressive knowledge deduplication.

### Auto-close features with no implementation work
Feature 8 had 0 tasks and its work was already done externally. The system should support transitioning a feature directly to `done` when all implementation work is verifiably complete, without requiring task creation/decomposition as a prerequisite. An "implementation-not-required" or "verified-complete" transition path would help.

### Improve the worktree-edit_file gap
`edit_file` should either work with worktrees (by resolving the entity_id to the worktree path) or the tool description should explicitly state the limitation. Currently it silently writes to the wrong location, which is worse than an error.

---

## Observations on the Kanbanzai System Itself

### The plan/batch distinction is clean
Having plans for strategic direction and batches for execution grouping makes intuitive sense. The vocabulary is precise and the separation of concerns is clear. The B38 batch contained features that built the very migration tooling that will be used to complete the migration — eating our own dogfood successfully.

### The stage gate system provides useful guardrails
The requirement to decompose before implementing, and to have approved specs before decomposing, prevented at least one premature implementation attempt. The error messages were clear about what was blocking and how to resolve it.

### The knowledge system has signal-to-noise challenges
The context assembly included ~70 knowledge entries, many of which were retrospective signals from unrelated plans. For feature-level work, plan-scoped knowledge would be more relevant than project-wide knowledge. The current system surfaces everything at project scope, which dilutes the signal.

### Manual task creation is a viable escape hatch
When `decompose` fails (AC format issue), manually creating tasks via `entity(action: create, type: task)` works perfectly. The system doesn't penalize this path — tasks created manually behave identically to those created by decompose. This is good design: the tool is an accelerator, not a gate.
