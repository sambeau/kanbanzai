---
name: orchestrate-review
description:
  expert: "Review orchestration with adaptive specialist dispatch,
    finding collation, verdict aggregation, and remediation routing
    across review dimensions for feature-level code review"
  natural: "Coordinate a team of code reviewers, collect their findings,
    and decide whether the code is ready to ship"
triggers:
  - orchestrate code review
  - coordinate review team
  - run review for feature
  - dispatch review sub-agents
  - collate review findings
roles: [orchestrator]
stage: reviewing
constraint_level: medium
---

## Vocabulary

- **review unit decomposition** — grouping related files into coherent units
  for independent review; each unit gets its own sub-agent dispatch
- **finding collation** — merging structured findings from multiple sub-agent
  review outputs into a single consolidated list
- **verdict aggregation** — deriving an overall feature verdict from the
  individual per-reviewer, per-dimension outcomes
- **remediation routing** — directing blocking findings to the appropriate
  agent or human for resolution before re-review
- **dispatch protocol** — the sequence of assembling skill + role + scope
  for each sub-agent before spawning it
- **adaptive composition** — selecting the number and type of specialist
  reviewers based on the actual files changed, rather than always dispatching
  a fixed team
- **review cycle count** — the number of review → remediation → re-review
  iterations a feature has undergone; signals whether progress is being made
- **specialist selection criteria** — the decision factors for which reviewer
  roles to dispatch (file types, change scope, risk profile)
- **deduplication pass** — identifying findings from different reviewers that
  describe the same issue at the same location, and collapsing them into one
- **review unit scope** — the set of files, spec sections, and dimensions
  assigned to one sub-agent for a single review pass
- **aggregate verdict** — the final combined verdict for the entire feature
  after all review units are evaluated and findings collated
- **dispatch ceiling** — the maximum number of sub-agents the binding
  registry permits for a single review orchestration (currently 4)
- **re-dispatch** — sending a review unit back to a sub-agent when the
  initial output failed evidence validation

## Anti-Patterns

### Result-Without-Evidence

- **Detect:** A sub-agent returns a review output where one or more
  dimensions lack evidence citations or spec references — the verdict
  is stated but not substantiated
- **BECAUSE:** Sub-agent outputs without per-dimension evidence cannot
  be distinguished from rubber-stamp approvals (MAST FM-3.1). Accepting
  unsubstantiated output propagates the failure to the aggregate verdict
  and defeats the purpose of specialist dispatch.
- **Resolve:** Reject the output. Re-dispatch the review unit to the same
  sub-agent with an explicit instruction to provide per-dimension evidence
  and spec citations. If the second attempt also lacks evidence, escalate
  to a human checkpoint.

### Over-Decomposition

- **Detect:** A feature is split into more review units than the file
  count warrants — typically more than one review unit per 5–8 files, or
  review units containing only 1–2 files when those files are closely
  related to adjacent units
- **BECAUSE:** Each review unit adds dispatch overhead, consumes context
  budget across sub-agents, and increases the deduplication burden. Overly
  granular units also lose cross-file context that reviewers need to assess
  coherence.
- **Resolve:** Merge related files into fewer, larger review units. Group
  by functional boundary (e.g., handler + service + tests for one endpoint)
  rather than by file type.

### Static Team Dispatch

- **Detect:** Every review dispatches all 4 specialist reviewers regardless
  of change scope — a 3-file documentation change gets the same reviewer
  panel as a 40-file service implementation
- **BECAUSE:** Static dispatch wastes context budget and agent time on
  dimensions that have no material to evaluate. Captain Agent research
  shows 15–25% improvement from adaptive team composition over static
  teams because specialists can focus on relevant material.
- **Resolve:** Select reviewers based on what actually changed. If no
  security-relevant code changed, do not dispatch the security reviewer.
  If the change is test-only, dispatch only the testing reviewer. Match
  the panel to the risk profile of the change.

### Premature Approval

- **Detect:** Routing to approval when one or more sub-agent outputs have
  not yet been received, or when a re-dispatch is still pending
- **BECAUSE:** Incomplete collation means the aggregate verdict is based
  on partial information. A missing reviewer output could contain blocking
  findings that would change the verdict.
- **Resolve:** Wait for all dispatched sub-agents to return. Only begin
  verdict aggregation when every dispatch has a corresponding output.

## Checklist

```
Copy this checklist and track your progress:
- [ ] Identified all files changed in the feature
- [ ] Grouped files into review units by functional boundary
- [ ] Selected specialist reviewers based on change scope
- [ ] Dispatched each sub-agent with review-code skill + reviewer role + scope
- [ ] Received structured output from every dispatched sub-agent
- [ ] Validated that every sub-agent output contains per-dimension evidence
- [ ] Deduplicated overlapping findings across reviewers
- [ ] Classified aggregate findings (blocking vs non-blocking)
- [ ] Produced aggregate verdict with routing decision
- [ ] Routed blocking findings to remediation OR routed to approval
```

## Procedure

### Step 1: Verify prerequisites

1. Confirm the feature is in `reviewing` status.
2. Locate the specification document(s) for the feature.
3. Identify all files changed in the feature (use the worktree diff or
   file list from the feature entity).
4. IF the spec is missing or the feature is not in the correct status →
   STOP. Report the missing prerequisite. Do not proceed without a spec
   to review against.

### Step 2: Decompose into review units

1. Group related files into review units by functional boundary — a unit
   should contain files that a reviewer needs to see together to assess
   coherence (e.g., handler + service + repository + tests for one domain).
2. Each review unit gets a label (e.g., `entity-lifecycle`, `storage-layer`,
   `cli-commands`) and a file list.
3. Assign the relevant spec section(s) to each review unit.
4. IF the feature has ≤8 files → a single review unit is likely sufficient.
   Do not split unless files serve clearly independent concerns.

### Step 3: Select specialist reviewers adaptively

Select which reviewer roles to dispatch based on what the files contain.
The dispatch ceiling is 4 sub-agents, but this is a maximum, not a target.

Decision factors:
- **Always dispatch:** `reviewer-conformance` — spec conformance is required
  for every review.
- **Dispatch if production code changed:** `reviewer-quality` — implementation
  quality applies when there is implementation to evaluate.
- **Dispatch if test files changed or test coverage is a spec requirement:**
  `reviewer-testing` — testing adequacy applies when tests are in scope.
- **Dispatch if security-relevant code changed:** `reviewer-security` —
  security review applies to authentication, authorisation, input handling,
  cryptography, or external system integration.

For small features (≤10 files, single concern), 1–2 reviewers are often
sufficient. Match the panel size to the risk and scope of the change.

### Step 4: Dispatch sub-agents

For each selected reviewer, dispatch a sub-agent with:
- **Skill:** `review-code`
- **Role:** The selected reviewer role (e.g., `reviewer-conformance`)
- **Scope:** The review unit file list + relevant spec section(s)

Each sub-agent operates independently. They do not see each other's output.
They each produce a structured review output in the `review-code` format.

### Step 5: Collate and validate findings

1. Collect structured outputs from all dispatched sub-agents.
2. IF any output is missing → wait. Do not aggregate from partial results.
3. Validate each output: does every dimension have an explicit outcome with
   evidence? IF not → reject and re-dispatch (see Result-Without-Evidence).
4. Merge all findings into a single collated list.
5. **Deduplication pass:** Identify findings from different reviewers that
   describe the same issue at the same location. Collapse duplicates into
   a single finding, preserving the highest classification (if one reviewer
   marked it blocking and another non-blocking, keep blocking).

### Step 6: Aggregate verdict and route

1. Derive the aggregate verdict from per-reviewer outcomes:
   - IF any dimension across any reviewer has a `fail` outcome → aggregate
     is `rejected`
   - IF any reviewer has `concern` outcomes but no `fail` → aggregate is
     `approved_with_followups`
   - IF all dimensions across all reviewers are `pass` or `pass_with_notes`
     → aggregate is `approved`
2. Route based on aggregate verdict:
   - `rejected` → produce remediation plan listing each blocking finding
     with its location and spec reference. Route to the implementing agent
     or human for resolution.
   - `approved_with_followups` → list non-blocking findings for follow-up.
     Feature can proceed to done.
   - `approved` → feature can proceed to done.
3. IF this is a re-review (review cycle count > 1), verify that previously
   blocking findings have been resolved. Do not approve if prior blocking
   findings were not addressed.

## Output Format

```
Feature: <feature-id> — <feature-slug>
Review cycle: <N>
Reviewers dispatched: <list of reviewer roles>
Review units: <count>

---

Per-Reviewer Summary:

  Reviewer: <role>
  Review unit: <unit-label>
  Verdict: <per-reviewer verdict>
  Dimensions:
    <dimension>: <outcome>
    <dimension>: <outcome>
  Findings: <count blocking>, <count non-blocking>

  (repeat per reviewer)

---

Collated Findings (deduplicated):

  [B-1] (blocking)
  Dimension: <dimension>
  Location: <file>:<lines>
  Spec ref: <requirement ID>
  Description: <what is wrong and why it violates the spec>
  Reported by: <reviewer role(s)>

  [NB-1] (non-blocking)
  Dimension: <dimension>
  Location: <file>:<lines>
  Description: <observation and recommendation>
  Reported by: <reviewer role(s)>

  (repeat per finding)

---

Aggregate Verdict: <approved | approved_with_followups | rejected>

Remediation Plan (if rejected):
  1. [B-1] — <brief action required> → route to <agent/human>
  2. [B-2] — <brief action required> → route to <agent/human>
```

## Examples

### BAD: Static dispatch with unvalidated output

```
Feature: FEAT-042 — add-user-endpoint
Reviewers dispatched: reviewer-conformance, reviewer-quality,
  reviewer-security, reviewer-testing
Review units: 4

Per-Reviewer Summary:
  Reviewer: reviewer-conformance — Approved
  Reviewer: reviewer-quality — Approved
  Reviewer: reviewer-security — Approved
  Reviewer: reviewer-testing — Approved

Collated Findings: None

Aggregate Verdict: approved
```

WHY BAD: All 4 specialists dispatched for what may be a small change
(no evidence the scope warranted it). No per-dimension outcomes shown.
No evidence citations from any reviewer. The "approved" verdicts are
unsubstantiated — this is result-without-evidence propagated to the
aggregate level. A machine cannot verify what was actually checked.

### GOOD: Adaptive dispatch with validated collation

```
Feature: FEAT-042 — add-user-endpoint
Review cycle: 1
Reviewers dispatched: reviewer-conformance, reviewer-quality
Review units: 1 (user-endpoint: handler.go, service.go, store.go,
  handler_test.go, service_test.go)

---

Per-Reviewer Summary:

  Reviewer: reviewer-conformance
  Review unit: user-endpoint
  Verdict: approved
  Dimensions:
    spec_conformance: pass
      Evidence: AC-1 (user creation, handler.go L22-45),
      AC-2 (validation, service.go L18-33), AC-3 (error response,
      handler.go L47-62) — all criteria verified

  Reviewer: reviewer-quality
  Review unit: user-endpoint
  Verdict: approved_with_followups
  Dimensions:
    implementation_quality: pass_with_notes
      Evidence: Error wrapping with %w throughout. Interface-based
      injection at consumer (service.go L8).
      Finding (non-blocking): handler.go L55 — error response uses
      http.StatusInternalServerError for a validation failure;
      should be http.StatusBadRequest
    test_adequacy: pass
      Evidence: 18 test cases across handler and service. Table-driven.
      Covers happy path, validation failures, duplicate detection.

---

Collated Findings (deduplicated):

  [NB-1] (non-blocking)
  Dimension: implementation_quality
  Location: handler.go:55
  Description: HTTP status code for validation error is 500 instead of
    400. Does not violate spec (AC-3 says "error response" without
    specifying status code) but is a correctness improvement.
  Reported by: reviewer-quality

---

Aggregate Verdict: approved_with_followups

Follow-up items:
  1. [NB-1] — Consider changing status code to 400 for validation errors
```

WHY GOOD: Only 2 reviewers dispatched — security reviewer omitted because
no security-relevant code changed (no auth, no crypto, no external input
beyond the handler's existing framework validation). Single review unit
because the files are one cohesive domain. Every dimension has evidence.
The non-blocking finding is specific with location. Aggregate verdict
correctly reflects the pass_with_notes from the quality reviewer.

## Evaluation Criteria

1. Were specialist reviewers selected based on the actual files changed,
   with a rationale for inclusions and exclusions?
   Weight: required.
2. Was every sub-agent output validated for per-dimension evidence before
   collation?
   Weight: required.
3. Were overlapping findings from different reviewers deduplicated into
   single entries?
   Weight: high.
4. Does the aggregate verdict correctly reflect the per-reviewer outcomes
   (worst outcome propagates)?
   Weight: required.
5. Are blocking findings routed to remediation with specific actions?
   Weight: high.
6. Is the review unit decomposition appropriate for the feature scope
   (not over-decomposed)?
   Weight: medium.
7. Does the output distinguish per-reviewer summaries from the collated
   aggregate?
   Weight: high.

## Questions This Skill Answers

- How do I coordinate a multi-reviewer code review for a feature?
- When should I dispatch fewer than 4 specialist reviewers?
- How do I group files into review units for sub-agent dispatch?
- What do I do when a sub-agent returns a review without evidence?
- How do I deduplicate findings from multiple reviewers?
- How do I decide between remediation routing and approval?
- What does the aggregate review report look like?
- How do I handle a re-review after remediation?