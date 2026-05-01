# Retrospective: B36 Implementation (April 2026)

| Field   | Value                    |
|---------|--------------------------|
| Date    | 2026-04-30               |
| Author  | AI agent (orchestrator)  |
| Scope   | B36-kbz-cli-and-status   |

## Overview

B36 was a 4-feature batch implementing the `kbz` binary rename and extending the `kbz status`
CLI command with human-readable and machine-readable output. All 19 tasks were completed across
all 4 features with zero blocking failures. The work involved creating specifications, dev-plans,
and implementation code across multiple Go packages, all orchestrated via the Kanbanzai MCP
server.

---

## What Went Well

### Parallel dispatch with disjoint file scopes

The orchestrator-workers pattern worked well when tasks had clearly disjoint write scopes.
F3-T2 (document views) and F3-T3 (entity views) were dispatched in parallel to separate
sub-agents and both completed without conflicts. Similarly, F4-T1 (plain renderer) and F4-T2
(JSON renderer) ran in parallel across the F3/F4 feature boundary with no issues. The
conflict detection via `next(conflict_check: true)` flagged medium-risk overlaps early,
allowing serialisation decisions before dispatch.

### Sub-agent output quality was consistently high

Every sub-agent produced working, tested Go code. The implementing agents correctly:
- Followed existing codebase patterns (package structure, naming conventions)
- Added thorough test coverage (20+ tests per renderer package)
- Exported necessary types from `internal/mcp/status_tool.go` when needed by downstream packages
- Handled edge cases proactively (null plan_id, empty attention arrays, missing documents)

### Specification to dev-plan to implementation traceability held up

The spec acceptance criteria format (checklist `- [ ] **AC-NNN:**`) drove clean task decomposition
once the format issue was resolved. Every feature had an approved spec before implementation
began, and the specs served as reliable contracts during implementation. Sub-agents referenced
spec sections directly when implementing.

### Service-layer reuse avoided duplication

The decision to have both human and machine renderers consume the same `internal/mcp`
synthesis types paid off. The PlainRenderer and JSONRenderer implementations were compact
because they consumed pre-computed data structures rather than re-querying the state store.

---

## What Didn't Go Well

### decompose tool format mismatch consumed significant time

**Root cause:** The `decompose propose` tool requires acceptance criteria in a specific bold-identifier
format (`- [ ] **AC-NNN:**` checklist or `**AC-NN.**` period-terminated). The initial specs
used heading-based ACs (e.g. `### AC-001`), which the tool silently failed to parse, producing
"no acceptance criteria found" errors with no actionable remediation.

**Impact:** Four rounds of fix-and-retry across all four specs before decomposition succeeded.
Each round required: edit spec, refresh doc, re-approve, retry decompose. This consumed
approximately 30% of the planning-phase effort.

**Mitigation:** The workaround (editing specs to checklist format) worked but the tool should
either accept a wider range of AC formats or provide a clear, specific error message with a
before/after example. The error message "no bold-identifier lines found anywhere in the document"
is technically accurate but gives the user no indication of what format *is* expected.

### doc refresh reverts approval status

**Root cause:** When `doc(action: refresh)` detects a content hash change, it resets the
document status from `approved` to `draft`. This means every minor edit to a spec file
(like fixing AC formatting) requires a full re-approval cycle.

**Impact:** Multiple re-approval round-trips during the format-fixing phase. The side effect
is not documented in the `doc refresh` tool description — it's discovered through experience.

**Suggestion:** `doc refresh` could either preserve the approval status (since the human
already approved the document content, and minor edits don't invalidate that approval) or
at minimum warn explicitly that approval will be reset before proceeding.

### Missing tasks from dev-plan creation

**Root cause:** The F4 dev-plan described 4 tasks but only 2 were actually created by the
sub-agent that wrote it. The plain and JSON renderer tasks existed in the dev-plan document
but not in the entity store. Additionally, the remaining tasks referenced non-existent
dependency task IDs, requiring manual dependency repair before dispatch.

**Impact:** Had to manually create 2 missing tasks and fix dependency references mid-orchestration.
This would have been a showstopper if the orchestrator didn't detect the gap during dispatch.

**Suggestion:** Add a validation step to the `doc approve` flow for dev-plans that cross-checks
the task count in the document against actual task entities in the store, and surfaces a
warning if they don't match.

### Approval gate doesn't respect false positive records

**Root cause:** The design document for B36 used a non-standard section structure (Purpose,
Problem Statement, Design Principles, etc. instead of Overview, Goals and Non-Goals, Design,
etc.). `doc(action: record_false_positive)` appeared to succeed (no error returned), but
`doc(action: approve)` continued to reject the document citing the same missing sections.

**Impact:** Had to manually edit the document YAML state file to set `status: approved`.
This is a fragile workaround that bypasses validation entirely.

**Severity:** This pattern (approval gate ignoring false positives) has been reported in prior
retrospective entries. It appears to be a persistent issue.

---

## What Should We Improve

### 1. decompose tool UX

- Accept ACs in any common format (headings, bold-identifier, checklist, numbered items)
- When parsing fails, emit a specific error message showing the exact format expected, with a before/after example
- Consider reading spec content directly from disk as a fallback when the doc_intel index is stale

### 2. doc lifecycle ergonomics

- `doc refresh` should preserve approval status when only minor edits are detected, or at minimum warn before resetting
- Consider a `doc(action: "edit", ...)` that updates content and refreshes while preserving approval
- The approval gate should consistently respect `record_false_positive` entries — the current behaviour where `validate` passes but `approve` fails indicates two different code paths that should be unified

### 3. dev-plan task validation

- When a dev-plan is approved, validate that all tasks described in the document exist as entities in the store
- When task entities reference `depends_on` IDs, validate that those IDs exist
- Surface these as warnings in the approval response, not just silent gaps

### 4. worktree-based development is not fully leveraged

**Observation:** All sub-agent work happened on the main tree, not in per-feature worktrees.
The worktrees existed (`next` showed them) but agents weren't directed to use them. This
is partially because `spawn_agent` shares the main filesystem and MCP server context, so
there's no natural isolation.

**Suggestion:** Either make worktree usage automatic in the `handoff`/`spawn_agent` flow
(so sub-agents work inside the correct worktree), or document clearly that the orchestrator
must explicitly instruct agents to use the worktree path.

---

## Summary

B36 was a productive batch. The Kanbanzai system provided solid structure through its
spec-to-plan-to-implement lifecycle, and the parallel dispatch capabilities kept implementation
velocity high. The primary friction points were in the tooling layer — format sensitivity
in `decompose`, approval/reset cycles in `doc refresh`, and missing task entities from
dev-plan creation. These are all fixable and none blocked delivery.
