# Retrospective: B36 Review Fixes Implementation

**Author:** sambeau (agent implementation session)
**Date:** 2026-04-30
**Scope:** 10 blocking review fixes across F2, F3, F4 in B36-kbz-cli-and-status

---

## What Went Well

### Good: Dev-plan amendment before task creation was clean
The dev-plan amendment workflow was smooth. The three dev-plans were easy to amend
with targeted edits. The review document was precise about what code was wrong,
where it lived, and what the corrected behaviour should be.

### Good: Entity lifecycle system caught state issues
The gate system correctly identified when features could not advance because a
task was in a non-terminal state. Even though the stale-cache issue required
overrides, the gates themselves were working as designed.

### Good: Single-file edits with build-and-test feedback loop were fast
All 10 fixes were concentrated in a small number of files, so the
edit-compile-test cycle was tight.

---

## What Did Not Go Well

### Bad: State inconsistency between entity get and entity list
TASK-01KQFT5G5XMS7 was marked done via finish(), entity get confirmed status: done,
but entity list and the feature transition gate both saw it as active. This
required two override transitions. The finish tool returned success but the state
was not consistently propagated.

### Bad: write_file tool failures
Attempts to use write_file failed with a directory-not-found error. The tool
appeared to misinterpret the project root path. Switching to edit_file resolved it.

### Bad: edit_file silently deleted content on partial match
A multi-edit call where some old_text patterns did not match caused non-matching
edits to be silently skipped while matching ones applied, including deletions
without replacements. This corrupted the F4 dev-plan. Required git checkout to
restore and redo edits one at a time.

### Bad: Dependency interface discovery required trial-and-error
When implementing the FR-007 fallback probe, I assumed deps.newDocumentService
took one argument and returned a service with ListDocuments. Both were wrong.
I had to grep through the codebase mid-implementation to discover correct
signatures. The handoff tool would have assembled this context for a sub-agent.

---

## What to Improve

### Improve: Fix finish state propagation
A finish() call that returns success should guarantee the task is visible as
terminal to all subsequent gate checks. The inconsistency is a correctness bug.

### Improve: Make edit_file atomic on multi-edit calls
Either apply all edits or none. If some old_text patterns do not match, fail the
entire call with a clear error instead of silently applying partial changes.

### Improve: Direct implementation context for orchestrator
When the orchestrator implements tasks directly, there is no handoff-assembled
context packet. A lightweight context assembly mode on next would help.

### Improve: Test assertions should reference spec requirements
Several tests checked for implementation-specific strings rather than
spec-required output. Tests asserting against spec requirements would have
caught the original violations and would not have needed updating during fixes.

---

## Summary

The review-fix-implement-advance cycle worked end-to-end in about 70 tool calls
across 7 tasks and 3 features. Primary friction was state consistency (stale cache
between finish and gate checks) and tool reliability (partial edit_file application,
write_file failures). The dev-plan amendment pattern is solid but assumes tooling
that reliably reflects state changes.
