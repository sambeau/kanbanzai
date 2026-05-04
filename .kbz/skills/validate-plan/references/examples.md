# validate-plan: Extended Examples

See `SKILL.md` for the main skill definition and procedure. This file contains
the full GOOD example and additional BAD/GOOD pairs with detailed analysis.

## GOOD: Structured validation with blocking failure

```
# Plan Validation: P43-fast-track-architecture — Fast-Track Architecture

| Field        | Value                         |
|--------------|-------------------------------|
| Plan         | P43-fast-track-architecture   |
| Spec         | P43-spec-fast-track-architecture.md |
| Validator    | plan-validator                |
| Date         | 2026-05-04T15:00:00Z          |
| Verdict      | fail                          |
| Blocking     | 2                             |
| Non-blocking | 1                             |
| Evidence     | 1.0                           |

## Traceability Matrix

| Task      | References            | Status |
|-----------|-----------------------|--------|
| TASK-1    | REQ-TIER-001..003,005,006 | ✅ |
| TASK-2    | REQ-SPEC-001, REQ-SPEC-004 | ✅ |
| TASK-3    | REQ-PLAN-001, REQ-PLAN-004 | ✅ |
| TASK-4    | REQ-RVW-001, REQ-RVW-004   | ✅ |
| TASK-5    | REQ-RUB-001, REQ-RUB-002, REQ-RUB-003 | ✅ |
| TASK-6    | REQ-SPEC-002, REQ-SPEC-003, REQ-SESS-001..003, REQ-NF-001 | ✅ |
| TASK-7    | REQ-PLAN-002, REQ-PLAN-003, REQ-SESS-001..003  | ✅ |
| TASK-8    | REQ-RVW-002, REQ-RVW-003, REQ-SESS-001..003    | ✅ |
| TASK-9    | REQ-INFER-001, REQ-INFER-002, REQ-INFER-003     | ✅ |
| TASK-10   | REQ-SESS-004, REQ-PIPE-002  | ✅ |
| TASK-11   | REQ-TRANS-001 through REQ-TRANS-007              | ✅ |
| TASK-12   | REQ-PIPE-001 through REQ-PIPE-007, REQ-TIER-004  | ✅ |
| TASK-13   | REQ-TIER-003, REQ-PIPE-006, REQ-TRANS-006        | ✅ |
| TASK-14   | All acceptance criteria                          | ✅ |

## Requirement Coverage

| REQ-ID         | Covered By | Status |
|----------------|-----------|--------|
| REQ-TIER-001   | TASK-1    | ✅     |
| REQ-TIER-002   | TASK-1    | ✅     |
| REQ-PLAN-003   | TASK-7    | ✅     |

(All REQ-IDs covered — D4 passes.)

## Per-Check Findings

### D1: Required sections present — BLOCKING | PASS
All five required sections found: Overview, Task Breakdown, Dependency Graph,
Interface Contracts (as "Dependency Graph" with parallel groups annotation),
Traceability Matrix (implicit in task spec-requirement fields).

### D2: Scope references parent spec — BLOCKING | PASS
Overview references `work/P43-fast-track-architecture/P43-spec-fast-track-architecture.md`
with document ID.

### D3: Task-to-requirement traceability — BLOCKING | PASS
All 14 tasks reference valid REQ-IDs from the parent specification.
No phantom references detected. (See traceability matrix above.)

### D4: Requirement coverage — BLOCKING | PASS
All REQ-IDs from the specification are covered by at least one task.
(See requirement coverage table above.)

### D5: Scope drift — BLOCKING | PASS
All task behaviour is traceable to spec requirements. Task 12 mentions
P44 integration but explicitly scopes it as out-of-scope, consistent
with the spec boundary.

### D6: Dependency graph acyclic — BLOCKING | FAIL

**Evidence:** Topological sort failed. Nodes remaining after processing:
Task 6, Task 7, Task 8, Task 12. Cycle detected: Task 6 → Task 12 → Task 6.

The dependency graph declares: Task 12 depends on Task 6, Task 6 depends
on Task 5, Task 5 depends on Task 2. However, Task 12's description states
"Depends on: Task 10, Task 11, Task 6, Task 7, Task 8" and Task 6's
description states "Depends on: Task 12" (circular). This appears to be a
transcription error — Task 12 depends on Task 6 but Task 6 should not
depend on Task 12.

**Undeclared dependencies:** None detected in task body scan.

### D7: No monolithic tasks — NON-BLOCKING | PASS
All 14 tasks within file-count and AC-count thresholds. Worst case: Task 14
touches 2 files (`internal/validate/`, `internal/mcp/`) and maps to 1 AC
(per-criterion verification).

### D8: Independent tasks parallelisable — NON-BLOCKING | PASS
Independent task groups identified and marked: Wave 1 (Tasks 2,3,4 and
Tasks 9,10), Wave 2 (Tasks 6,7,8), Wave 5 (Tasks 13,14).

### D9: Verification mapping complete — BLOCKING | PASS
All AC-IDs from the specification are mapped to producing tasks in the
verification section. No phantom AC references.

### D10: Risk assessment non-empty — NON-BLOCKING | PASS
Five risks identified. All five are substantive with probability, impact,
mitigation, and affected tasks. All five are feature-specific (cite
`internal/validate/`, `gates.go`, `spawn_agent`, etc.). Rubric: clear pass.

### D11: File path validation — NON-BLOCKING | FAIL

| Task   | Path                                    | Exists | Marked New | Status |
|--------|-----------------------------------------|--------|------------|--------|
| TASK-6 | .kbz/skills/validate-spec/SKILL.md     | ❌     | ❌         | ⚠️     |
| TASK-7 | .kbz/skills/validate-plan/SKILL.md     | ❌     | ❌         | ⚠️     |
| TASK-8 | .kbz/skills/validate-review/SKILL.md   | ❌     | ❌         | ⚠️     |

Three task deliverables reference files that do not yet exist and are not
marked as "new file" or "to be created." This is expected for skill-creation
tasks where the file IS the task output. Flagged as non-blocking — the task
descriptions imply these are the files being created.

### D12: Non-functional requirements — NON-BLOCKING | PASS
REQ-NF-001 through REQ-NF-003 all referenced by tasks (Task 6, Task 11,
Task 14).

### D13: Task description actionability — BLOCKING | PASS
All 14 task descriptions exceed 50 words, state what they produce, state
required inputs, and include a verifiable done criterion. Rubric: clear pass
on all tasks. Worst case: Task 1 at 95 words with explicit file targets,
dependency statement ("Depends on: None"), and done criterion
("unit tests pass, config loads without error").

## Verdict

FAIL. One blocking finding: D6 (dependency graph acyclic) — cycle detected
between Task 6 and Task 12. This must be resolved before the plan can advance
to developing. One non-blocking finding: D11 (file path validation) — three
skill files do not yet exist (expected for creation tasks).
```

WHY GOOD: Every check enumerated with explicit pass/fail and evidence.
Traceability matrix cross-references all tasks against spec REQ-IDs.
Requirement coverage matrix shows every REQ-ID is covered. Dependency graph
analysis includes topological sort result, cycle participants, and cycle
diagnosis. Rubric checks (D7, D10, D13) cite rubric classification. File
path validation includes per-path status in a structured table. Summary
provides counts and report reference. A reader can verify each claim
independently.
