---
name: validate-plan
description:
  expert: "Structured dev-plan validation producing a gate-checkable report
    with 13 checks (D1-D13), each classified as blocking or non-blocking,
    and a summary verdict (pass/pass_with_notes/fail) for the orchestrator"
  natural: "Validate a dev-plan document against the 13 quality checks —
    tell me if it passes, has notes, or fails, and produce a full report"
triggers:
  - validate a dev-plan
  - run dev-plan validation
  - check a plan for completeness
  - review a dev-plan against the 13 checks
  - produce a plan validation report
roles: [plan-validator]
stage: dev-planning
constraint_level: low
---

## Vocabulary

- **blocking check** — a structural gate that must pass for the dev-plan to be approved; failure prevents advancement past the dev-planning stage
- **non-blocking check** — a quality signal that does not block advancement but must be reported
- **evidence score** — the fraction of checks (D1-D13) whose evidence was positively identified, regardless of pass/fail outcome
- **dependency graph** — the directed acyclic graph of task dependencies; checked by D6
- **traceability matrix** — the mapping from spec requirements to tasks that implement them; checked by D3, D4, D5
- **verification mapping** — the mapping from acceptance criteria to the task that produces the verified output; checked by D9
- **scope drift** — a task that implements behaviour not traceable to any spec requirement; checked by D5
- **task decomposition granularity** — whether each task is appropriately sized (not monolithic, not trivial); checked by D7
- **parallelisability** — whether independent tasks are annotated as parallelisable; checked by D8
- **file-path existence** — whether every referenced file either exists or is explicitly declared as new; checked by D11

## Anti-Patterns

### Phantom Traceability
- **Detect:** A task claims to implement a spec requirement but the requirement ID does not exist in the parent specification
- **BECAUSE:** Fabricated traceability creates false confidence in coverage and masks missing implementation
- **Resolve:** Cross-reference every task's requirement reference against the parent specification document; flag unmatched references as blocking

### Architectural Second-Guessing
- **Detect:** Recommending architectural alternatives — "this should use a queue", "should be event-driven"
- **BECAUSE:** The plan-validator audits completeness and traceability, not architectural merit
- **Resolve:** Treat the spec's authorised approach as given; if a design gap threatens implementability, flag as non-blocking note

### Unverified File References
- **Detect:** Assuming a file path in task deliverables exists without checking
- **BECAUSE:** Non-existent or undeclared file references produce orphaned work
- **Resolve:** For every file path: check existence or new-file declaration; flag unresolved paths as D11 non-blocking

## Checklist

```
Copy this checklist and track your progress:
- [ ] Read the dev-plan document and parent specification
- [ ] D1: All required sections present (Overview, Task Breakdown, Dependency Graph, Interface Contracts, Traceability Matrix)
- [ ] D2: Scope section references the parent specification document
- [ ] D3: Every task references at least one spec requirement
- [ ] D4: Every spec requirement is covered by at least one task
- [ ] D5: No scope drift — all tasks traceable to spec
- [ ] D6: Dependency graph is acyclic
- [ ] D7 (non-blocking): No monolithic tasks (>3 files or >1 AC)
- [ ] D8 (non-blocking): Independent tasks marked parallelisable
- [ ] D9: Every AC mapped to a task in verification section
- [ ] D10 (non-blocking): Risk assessment is non-empty
- [ ] D11 (non-blocking): All file paths exist or are declared new
- [ ] D12 (non-blocking): Non-functional requirements addressed
- [ ] D13: Every task description is adequate (≥50 words, states inputs/outputs/done)
- [ ] Produce summary verdict and full report
- [ ] Register report via doc(action: "register")
```

## Procedure

### Step 1: Orient

1. Read the dev-plan document fully.
2. Read the parent specification document fully.
3. Note which checks are blocking (D1-D6, D9, D13) and which are non-blocking (D7, D8, D10-D12).

### Step 2: Run structural checks (D1, D2)

Check that the dev-plan has all required sections and references the parent spec.
These are blocking — a missing section or missing spec reference prevents advancement.

### Step 3: Build traceability matrix (D3, D4, D5)

1. Extract every task from the Task Breakdown section.
2. For each task, extract its spec requirement references.
3. For each spec requirement, verify it appears in at least one task.
4. Flag any task that references a non-existent requirement (D3 blocking).
5. Flag any uncovered requirement (D4 blocking).
6. Flag any task with no traceable spec reference (D5 blocking).

### Step 4: Validate dependency graph (D6)

1. Build a directed graph from task dependencies.
2. Check for cycles using depth-first search.
3. A cycle is blocking — the graph must be acyclic.

### Step 5: Evaluate task decomposition (D7, D8, D13)

1. Check each task's file count and AC count (D7 non-blocking).
2. Check parallelisability annotations (D8 non-blocking).
3. Check each task description for adequacy: ≥50 words, states inputs, outputs, and done criteria (D13 blocking).

### Step 6: Validate verification and file paths (D9, D11)

1. Map every AC to the task that verifies it (D9 blocking).
2. Check all file paths exist or are declared new (D11 non-blocking).

### Step 7: Check non-functional and risk coverage (D10, D12)

1. Verify risk assessment has content (D10 non-blocking).
2. Verify non-functional requirements are addressed (D12 non-blocking).

### Step 8: Produce output

1. Write a full report with per-check evidence.
2. Register the report via `doc(action: "register", type: "report", ...)`.
3. Return a summary with verdict, counts, evidence score, and report doc ID.

## Output Format

```
Dev-Plan Validation Report

Document: <dev-plan-path>
Parent Spec: <spec-path>

Verdict: pass | pass_with_notes | fail

Checks:
  D1 (blocking): pass | fail — <evidence>
  D2 (blocking): pass | fail — <evidence>
  D3 (blocking): pass | fail — <task→requirement mapping>
  D4 (blocking): pass | fail — <requirement→task mapping>
  D5 (blocking): pass | fail — <scope drift analysis>
  D6 (blocking): pass | fail — <cycle detection result>
  D7 (non-blocking): pass | fail — <monolithic task analysis>
  D8 (non-blocking): pass | fail — <parallelisability analysis>
  D9 (blocking): pass | fail — <AC→task verification mapping>
  D10 (non-blocking): pass | fail — <risk assessment>
  D11 (non-blocking): pass | fail — <file path validation>
  D12 (non-blocking): pass | fail — <NFR coverage>
  D13 (blocking): pass | fail — <task description adequacy>

Blocking: <count>
Non-blocking: <count>
Evidence score: <0.0–1.0>
Report doc ID: <document-id>
```

## Edge Cases

### Missing parent spec
If the spec reference in the dev-plan cannot be resolved, D2 fails (blocking). Skip D3-D5 and D12 (they require the spec). Continue evaluating D6-D11 and D13 independently.

### Empty task list
If the Task Breakdown section exists but contains no tasks, D1 passes but D3, D4, and D9 all fail (blocking). An empty dev-plan cannot advance.

### Cyclic dependency
If the dependency graph contains a cycle, D6 fails (blocking). Identify the cycle explicitly in the report. The plan cannot advance until the cycle is broken.
