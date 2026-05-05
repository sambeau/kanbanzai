---
name: validate-plan
description:
  expert: "Plan-level structural validation checking all 13 D-checks against a
    dev-plan document and its parent specification — verifies completeness,
    traceability, acyclic dependencies, and task actionability before the plan
    advances to developing"
  natural: "Check whether this dev-plan is structurally sound — are all tasks
    traceable, is the dependency graph acyclic, are the task descriptions
    actionable, are all spec requirements covered — and produce a pass/fail
    verdict with a full finding report"
triggers:
  - validate this dev-plan
  - check the plan before developing
  - run plan-validator
  - audit dev-plan structure
  - verify plan traceability
roles: [plan-validator]
stage: plan-validation
constraint_level: low
---

## Vocabulary

- **dev-plan** — the implementation plan document being validated; contains task
  breakdown, dependency graph, traceability matrix, and risk assessment
- **parent specification** — the approved specification document the dev-plan claims
  to implement; source of truth for requirement IDs, acceptance criteria, and
  scope boundaries
- **D-check** — a single validation check from the plan-validator's catalog (D1–D13);
  each check has a blocking or non-blocking classification
- **blocking finding** — a check failure that prevents the dev-plan from advancing to
  the developing stage; must be resolved before the transition can proceed
- **non-blocking finding** — a check failure that does not block advancement but is
  attached to the document record for visibility and future remediation
- **traceability matrix** — the mapping of tasks to specification requirements and
  acceptance criteria; verifies every requirement is implemented and every task is
  justified
- **dependency graph** — the directed graph of task dependencies; edges represent
  "must be done before" relationships; must contain no cycles
- **topological sort** — a linear ordering of tasks respecting all dependency
  constraints; impossible if the graph contains cycles; computes in-degrees, uses
  a queue of zero-in-degree nodes, removes nodes while decrementing successor counts
- **scope drift** — a task that implements behaviour not traceable to any requirement
  in the parent specification; indicates the plan has expanded beyond spec boundaries
- **actionable description** — a task description that states what it produces, what
  inputs it requires, what "done" means beyond the acceptance criterion, and is at
  least 50 words; sufficient for an implementer to begin work without additional
  clarification
- **verification mapping** — the plan's mapping of acceptance criteria to the tasks
  that produce them; every AC must have at least one producing task
- **rubric check** — a check (D7, D10, D13) that requires LLM classification against
  the plan-validator rubrics defined in
  `work/P43-fast-track-architecture/validator-rubrics/plan-validator-rubrics.md`;
  these cannot be evaluated by structural pattern matching alone
- **fresh session** — the validator runs in an isolated context with no prior
  knowledge of the documents being validated; receives only the dev-plan, parent
  spec, check definitions, and rubrics; must complete within 5 tool calls
- **full report** — the complete validation output registered as a document record
  via `doc(action: register, type: report)`; contains per-check findings with
  evidence, the traceability matrix, dependency graph analysis, and aggregate verdict
- **summary** — the concise output returned to the orchestrator; contains verdict,
  blocking/non-blocking finding counts, evidence score, and reference to the full
  report document ID
- **per-check evidence table** — a structured table within the full report showing
  the result of each check with specific file paths, REQ-IDs, task IDs, or rubric
  citations that a reader can verify independently

## Anti-Patterns

### Phantom Traceability
- **Detect:** A task's spec-requirement field references a REQ-ID that does not exist
  in the parent specification, or a verification mapping references an acceptance
  criterion that does not exist in the spec
- **BECAUSE:** Fabricated traceability creates false confidence in coverage analysis
  and masks missing implementation — the plan appears complete but does not actually
  satisfy the specification; reviewers and orchestrators make deployment decisions
  based on coverage reports that are materially false
- **Resolve:** Cross-reference every task's requirement reference against the parent
  specification using exact string matching; cross-reference every AC in the
  verification mapping against the spec's acceptance criteria; flag any unmatched
  reference as a blocking gap under D3 or D4

### Hallucinated Completeness
- **Detect:** Declaring the plan complete without enumerating individual checks or
  providing per-check evidence — a summary verdict like "all checks pass" with no
  supporting traceability or dependency analysis
- **BECAUSE:** Summary verdicts without evidence are indistinguishable from
  rubber-stamped approvals; they provide no audit trail for downstream review,
  hide individual check failures behind aggregate claims, and make it impossible
  to determine which checks were actually executed
- **Resolve:** Produce per-check findings with explicit pass/fail status and evidence;
  never produce a verdict without running every check; the full report must be
  self-contained — a reader should be able to verify each check independently

### Architectural Second-Guessing
- **Detect:** The validator disagrees with a design decision in the spec and
  recommends alternatives — "this should use a queue," "should be event-driven,"
  "framework X is a better fit"
- **BECAUSE:** The plan-validator audits completeness and traceability, not
  architectural merit; questioning decisions already made in the specification
  undermines the spec-author and architect roles, creates rework loops that block
  plan approval indefinitely, and introduces scope beyond what the validator is
  authorised to evaluate
- **Resolve:** When the spec authorises a specific approach, treat it as a given;
  do not second-guess; if a design gap genuinely threatens implementability, flag
  it as a non-blocking note under D10 and defer to the architect — do not propose
  alternative designs

### Unverified File References
- **Detect:** The validator assumes a file path in a task's deliverables is valid
  without checking whether it exists in the repository or is explicitly noted as a
  new file to be created
- **BECAUSE:** Tasks that reference files that neither exist nor are declared as
  new outputs produce orphaned work that cannot be reviewed or merged; the plan
  appears thorough but is not executable, and implementing agents hit missing-file
  errors mid-task
- **Resolve:** For every file path in task deliverables: (1) check if it exists in
  the repository via `find_path` or `read_file`, (2) check if the path contains
  markers like "new file," "create," or "will be created," (3) if neither, flag
  D11 as non-blocking with the unresolved path; do not assume files exist or will
  be magically created

### Missing Dependency Declaration
- **Detect:** A task's description references an artifact produced by another task
  but the dependency is not declared in the dependency graph
- **BECAUSE:** Undeclared dependencies create implicit ordering constraints that
  the topological sort cannot enforce; the dependency graph appears simpler than
  it actually is, and parallel dispatch may schedule tasks in the wrong order,
  causing runtime failures
- **Resolve:** For each task, scan the description for references to artifacts from
  other tasks; cross-reference against the declared dependencies; if an undeclared
  dependency is found, flag as a non-blocking note under D6 with the specific
  task pair and the artifact reference

### Premature Verdict
- **Detect:** The validator issues a verdict before running all 13 D-checks, or
  skips checks that require reading external documents because "the information
  isn't available"
- **BECAUSE:** Selective validation is indistinguishable from biased validation;
  skipped checks hide failures and produce false confidence in the verdict;
  the orchestrator cannot distinguish "passed" from "not checked"
- **Resolve:** Run every check in the D1–D13 catalog; if document access fails
  (e.g., parent spec not found), stop with an input error — do not produce a
  partial verdict; the validator must either complete all checks or report an
  input error with no verdict

## Checklist

```
Copy this checklist and track your progress:
- [ ] Read dev-plan document (full content)
- [ ] Read parent specification document (full content)
- [ ] Read plan-validator rubrics (for D7, D10, D13)
- [ ] D1: Required sections present
- [ ] D2: Scope references parent spec
- [ ] D3: Every task references at least one spec requirement
- [ ] D4: Every spec requirement covered by at least one task
- [ ] D5: No scope drift
- [ ] D6: Dependency graph acyclic (topological sort)
- [ ] D7: No monolithic tasks (rubric check)
- [ ] D8: Independent tasks marked parallelisable
- [ ] D9: Verification maps every AC to a producing task
- [ ] D10: Risk assessment non-empty (rubric check)
- [ ] D11: Every file path validated
- [ ] D12: Non-functional requirements addressed
- [ ] D13: Every task description is actionable (rubric check)
- [ ] Compiled summary (verdict + counts + report doc ID)
- [ ] Wrote full report and registered as document record
```

## Procedure

### Step 1: Read inputs

1. Read the dev-plan document. Extract: all tasks (descriptions, dependencies,
   spec-requirement fields, deliverables, effort), dependency graph, risk
   assessment, and verification mapping.
2. Read the parent specification. Extract: all REQ-IDs, all AC-IDs, all
   non-functional requirements (REQ-NF-...), and scope section.
3. Read the plan-validator rubrics at
   `work/P43-fast-track-architecture/validator-rubrics/plan-validator-rubrics.md`.
4. IF any document cannot be read → STOP. Report the input error. Do not produce
   a partial verdict.
5. IF the dev-plan has no task breakdown section → D1 fails (blocking). Continue
   checking but verdict will be fail.

Read all three inputs in parallel where possible to minimise tool calls.

### Step 2: Execute structural checks (D1–D6, D8–D9, D11–D12)

Execute using structural pattern matching. Record pass/fail and evidence.

| Check | Blocking? | What to verify |
|-------|-----------|----------------|
| D1 | Yes | All required sections present: Overview, Task Breakdown, Dependency Graph, Interface Contracts, Traceability Matrix |
| D2 | Yes | Overview/Scope references parent spec (file path, doc ID, or matching title) |
| D3 | Yes | Build valid REQ-ID set from spec; each task must reference ≥1 valid REQ-ID (no phantom IDs) |
| D4 | Yes | Every spec REQ-ID covered by ≥1 task; record uncovered REQ-IDs |
| D5 | Yes | No task behaviour untraceable to any spec requirement; record task ID and drifted scope |
| D6 | Yes | Topological sort: compute in-degrees, queue zero-in-degree nodes, remove while decrementing; if nodes remain → cycle exists. Also scan for undeclared deps (non-blocking note) |
| D8 | No | Identify task sets with no cross-edges; check dev-plan marks them parallelisable; explicit sequencing for documented reasons is acceptable |
| D9 | Yes | Build AC-ID set from spec; every AC mapped to ≥1 producing task; phantom AC-IDs fail |
| D11 | No | For every file path: check repo existence or new-file markers ("create," "new," "will be created"); globs noted but not failed |
| D12 | No | Extract REQ-NF-IDs from spec; every NF requirement referenced by ≥1 task |

### Step 3: Execute rubric checks (D7, D10, D13)

Apply pass/fail/borderline-escalate from the plan-validator rubrics.

| Check | Blocking? | Rubric summary |
|-------|-----------|----------------|
| D7 | No | Per task: count distinct files (≤3) and ACs (≤1). Escalate: glob ambiguity, shared prerequisite (0 ACs), test-as-part-of-task if combined >3 |
| D10 | No | Risk Assessment section must have ≥1 named risk; each substantive (what/impact/probability/mitigation); ≥1 feature-specific. Escalate: one real risk + generic rest, accepted-without-mitigation, implied risks in task descriptions |
| D13 | Yes | Per task: (1) word count ≥50 (exclude metadata labels), (2) states concrete deliverable, (3) states required inputs ("None" only if explicit + true), (4) done criterion beyond AC (test command, compilation check, review req, verifiable assertion). Escalate: words 45–55, implicit done, "see dependency graph" inputs |

### Step 4: Compile summary

Verdict: `pass` (no blocking), `pass_with_notes` (non-blocking only), or
`fail` (≥1 blocking). Include blocking/non-blocking counts, evidence score
(fraction of checks run), and full report document ID.

### Step 5: Write and register full report

Write to `work/reviews/plan-validation-<plan-slug>-<date>.md` using the output
format below. Register:

```
doc(action: "register",
    path: "work/reviews/plan-validation-<plan-slug>-<date>.md",
    type: "report",
    title: "Plan Validation: <plan-id> — <plan-title>")
```

**Tool call budget:** Complete within 5 calls for typical dev-plans (≤300
lines, ≤15 tasks). Pattern: (1) read all three inputs in parallel, (2) read
remaining sections if large, (3) execute all checks, (4) register report, (5)
fallback for re-reads/escalations. Exceeding 5 signals unnecessary reads.

## Output Format

### Summary (returned to orchestrator)

```json
{
  "verdict": "pass | pass_with_notes | fail",
  "blocking_count": <int>,
  "non_blocking_count": <int>,
  "evidence_score": <float 0.0-1.0>,
  "report_doc_id": "<document record ID>"
}
```

### Full Report (written to document store)

```
# Plan Validation: <plan-id> — <plan-title>

| Field        | Value                   |
|--------------|-------------------------|
| Plan         | <plan-id>               |
| Spec         | <spec-doc-id or path>   |
| Validator    | plan-validator          |
| Date         | <ISO 8601 UTC>          |
| Verdict      | pass / pass_with_notes / fail |
| Blocking     | <count>                 |
| Non-blocking | <count>                 |
| Evidence     | <score>                 |

## Traceability Matrix

| Task   | References     | Status | Notes              |
|--------|----------------|--------|--------------------|
| TASK-1 | REQ-001, REQ-002 | ✅   |                    |
| TASK-2 | REQ-099        | ❌     | Phantom REQ-099    |

## Requirement Coverage

| REQ-ID | Covered By | Status |
|--------|-----------|--------|
| REQ-001 | TASK-1    | ✅     |
| REQ-003 | —         | ❌     |

## Dependency Graph Analysis

- **Node count:** <N>
- **Edge count:** <M>
- **Cycles:** none / <cycle description with task IDs>
- **Topological sort:** valid / invalid
- **Undeclared dependencies:** none / <list>

## Per-Check Findings

### D1: Required sections present — BLOCKING | PASS/FAIL

<Evidence: which sections found, which missing.>

### D2: Scope references parent spec — BLOCKING | PASS/FAIL

<Evidence: reference found or not found.>

### D3: Task-to-requirement traceability — BLOCKING | PASS/FAIL

| Task   | References | Valid? |
|--------|-----------|--------|

### D4: Requirement coverage — BLOCKING | PASS/FAIL

| REQ-ID | Covered By | Status |
|--------|-----------|--------|

### D5: Scope drift — BLOCKING | PASS/FAIL

<Evidence: tasks with behaviour not traceable to spec.>

### D6: Dependency graph acyclic — BLOCKING | PASS/FAIL

<Evidence: topological sort result, cycle participants, undeclared deps.>

### D7: No monolithic tasks — NON-BLOCKING | PASS/FAIL

| Task   | File Count | AC Count | Verdict | Notes |
|--------|-----------|----------|---------|-------|

### D8: Independent tasks parallelisable — NON-BLOCKING | PASS/FAIL

<Evidence: independent groups and their parallelisation status.>

### D9: Verification mapping complete — BLOCKING | PASS/FAIL

| AC-ID   | Producing Task | Status |
|---------|---------------|--------|

### D10: Risk assessment non-empty — NON-BLOCKING | PASS/FAIL

<Evidence: risk count, substance, feature-specific. Cite rubric.>

### D11: File path validation — NON-BLOCKING | PASS/FAIL

| Task   | Path              | Exists | Marked New | Status |
|--------|-------------------|--------|------------|--------|

### D12: Non-functional requirements — NON-BLOCKING | PASS/FAIL

| REQ-NF-ID | Covered By | Status |
|-----------|-----------|--------|

### D13: Task description actionability — BLOCKING | PASS/FAIL

| Task   | Words | Produces | Inputs | Done Criteria | Verdict |
|--------|-------|----------|--------|---------------|---------|

## Verdict

<Final assessment. If fail: blocking checks to resolve. If pass_with_notes:
non-blocking findings attached to record. If pass: confirmation.>
```

## Examples

### BAD: Skeletal validation with no evidence

```
# Plan Validation: P43 — Fast-Track Architecture

## Verdict: Pass

All checks appear to pass. The plan looks well-structured.
Tasks are decomposed and dependencies look correct.
```

WHY BAD: No per-check enumeration. "All checks appear to pass" with zero
evidence. No traceability matrix, no dependency analysis, no rubric
application. This is Hallucinated Completeness — indistinguishable from
a rubber stamp.

### BAD: Architectural second-guessing

```
D5 (Scope Drift): FAIL. Task 10 uses spawn_agent directly instead of a proper
dependency injection pattern. The dispatch_validator should use a factory
pattern with provider registration, not a simple interface. Recommend
refactoring Task 10 to use the Abstract Factory pattern.
```

WHY BAD: The validator evaluates architectural merit, not scope traceability.
The spec authorises the interface-based approach. This is Architectural
Second-Guessing. The correct finding: "Task 10 scoped to REQ-SESS-004 —
confirmed in traceability matrix" (D5 passes).

### GOOD: Structured validation with blocking failure

See [references/examples.md](references/examples.md) for the full annotated
example with a D6 cycle failure, per-check evidence tables, traceability
matrix, and rubric citations.

## Evaluation Criteria

1. Are all 13 D-checks (D1–D13) executed and reported individually?
   Weight: required.
2. Does the traceability matrix cross-reference every task against valid
   spec REQ-IDs? Weight: required.
3. Does the requirement coverage matrix show every spec REQ-ID with at least
   one covering task? Weight: required.
4. Is the dependency graph topologically sorted with a cycle detection
   result? Weight: required.
5. Are D7, D10, and D13 evaluated against the plan-validator rubrics with
   explicit pass/fail/borderline-escalate citations? Weight: required.
6. Does the summary include verdict, blocking/non-blocking counts, evidence
   score, and full report document ID? Weight: required.
7. Is the full report registered as a document record? Weight: required.
8. Are file paths in task deliverables validated against repository existence
   or new-file markers? Weight: high.
9. Does the validator stay in scope — no architectural second-guessing, no
   code quality commentary, no design preference recommendations?
   Weight: high.
10. Does the validator complete within the 5-tool-call budget for typical
    dev-plans? Weight: high.
11. Does the full report include per-check evidence tables that a reader can
    verify independently? Weight: high.
12. Does the validator stop with an input error (not a partial verdict) when
    required documents cannot be read? Weight: high.
13. Are undeclared dependencies scanned for and reported? Weight: medium.
14. Are borderline escalations handled per the rubric escalate patterns
    rather than auto-resolved? Weight: medium.

## Questions This Skill Answers

- How do I validate a dev-plan before it advances to developing?
- What are the 13 D-checks for plan validation?
- How do I verify the dependency graph is acyclic?
- How do I check task-to-requirement traceability?
- How do I detect scope drift in a dev-plan?
- How do I apply the plan-validator rubrics for D7, D10, and D13?
- What is a blocking vs non-blocking finding in plan validation?
- How do I produce a plan validation summary for the orchestrator?
- How do I write and register a full plan validation report?
- How many tool calls should plan validation take?
- What does a phantom traceability reference look like?
- How do I verify every acceptance criterion has a producing task?
- How do I validate file paths in task deliverables?
- When should plan validation stop with an input error instead of a verdict?
