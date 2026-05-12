---
# kanbanzai-managed: true
# kanbanzai-version: dev
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

- **parent batch** — the batch entity that owns the feature under review;
  reviews are scoped to a single feature but the batch provides context
  for cross-feature dependencies and aggregate delivery status
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
- [ ] Classified unclassified feature documents (or confirmed context budget insufficient)
- [ ] Grouped files into review units by functional boundary
- [ ] Selected specialist reviewers based on change scope
- [ ] Dispatched each sub-agent with review-code skill + reviewer role + scope
- [ ] Received structured output from every dispatched sub-agent
- [ ] Validated that every sub-agent output contains per-dimension evidence
- [ ] Deduplicated overlapping findings across reviewers
- [ ] Classified aggregate findings (blocking vs non-blocking)
- [ ] Produced aggregate verdict with routing decision
- [ ] Routed blocking findings to remediation OR routed to approval
- [ ] Wrote review document and registered with doc()
- [ ] Created human checkpoint for ambiguous or high-stakes findings (if applicable)
- [ ] Managed remediation cycle within iteration cap (if rejected)
- [ ] Post-review merge: verified all tasks terminal
- [ ] Post-review merge: transitioned feature to merging
- [ ] Post-review merge: ran merge check and execute
- [ ] Post-review merge: verified merge ancestry (git merge-base --is-ancestor)
- [ ] Post-review merge: transitioned to verifying, ran build and tests, transitioned to done (or needs-rework on failure)
```

## Procedure

### Step 1: Verify prerequisites

1a. Confirm the feature is in `reviewing` status.
1b. Classify unclassified feature documents. Call `doc_intel(action: "pending")` and
    filter results for documents owned by the feature under review. For each unclassified
    document, process in priority order (specification → design → dev-plan):
    1. Call `doc_intel(action: "guide", id: "DOC-xxx")` to get the section outline
       and content hash.
    2. Read the sections needed to understand the document content.
    3. Call `doc_intel(action: "classify", id: "DOC-xxx", content_hash: "...", ...)`
       to submit the classifications.

    **Classification is NOT a blocking prerequisite.** If context budget is exhausted,
    MUST proceed with reviewing anyway.

    **Rationale:** Reviewer sub-agents use `doc_intel` to navigate documents. Layer 3
    classification enables role-based search and produces richer guides. An unclassified
    corpus forces reviewers to fall back to structural navigation only, missing decision
    and rationale fragments that classification would have surfaced.
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
     Proceed to the Post-Review Merge phase below.
   - `approved` → proceed to the Post-Review Merge phase below.
3. IF this is a re-review (review cycle count > 1), verify that previously
   blocking findings have been resolved. Do not approve if prior blocking
   findings were not addressed.

### Step 7: Write review document

1. Write collated findings to the canonical review path. Consult
   `.agents/skills/kanbanzai-documents/SKILL.md` § "Document Types
   and Locations" for the filename template.

   For a batch-scoped feature review:
   `work/{BatchID}-{batch-slug}/{feature-id}-review-{slug}.md`

   Where `{feature-id}` is the full feature entity ID
   (e.g., `FEAT-01KMRX1SEQV49`) and `{slug}` is the feature slug.
2. The document must contain: summary verdict, per-dimension verdicts,
   blocking findings with locations, non-blocking findings, and review
   unit breakdown showing the dispatch scope for each sub-agent.
3. Register the document:
   `doc(action: "register", owner: "<feature-id>", path: "<report-path>", type: "report", title: "Review: <feature-slug>")`

### Remediation Phase (when verdict is rejected)

Enter this phase only when the aggregate verdict is `rejected`.

1. Transition the feature to `needs-rework`:
   `entity(action: "transition", id: "<feature-id>", status: "needs-rework")`
2. Create remediation tasks as children of the feature — one per blocking
   finding or logical group of related findings:
   `entity(action: "create", type: "task", parent_feature: "<feature-id>")`
3. Before dispatching tasks in parallel, check for file overlap:
   `conflict(action: "check", task_ids: [...])`
4. Dispatch tasks through the normal workflow:
   `next(id: "<task-id>")` to claim and activate.
5. After remediation tasks complete, re-review ONLY affected sections — not
   the entire feature. Spawn sub-agents for affected review units only.
6. If re-review passes: transition feature to `reviewing` status (the
   approved verdict state) and proceed to the Post-Review Merge phase
   below. If new blocking findings: repeat from step 1 of this phase.
7. **Iteration cap:** Maximum 3 remediation-re-review cycles. If issues
   persist after 3 cycles, escalate to human via
   `checkpoint(action: "create")`.

### Human Checkpoint Integration

Create a human checkpoint (`checkpoint(action: "create")`) when:

1. **Ambiguous findings** — findings that are not clearly blocking or
   non-blocking; the orchestrator cannot make a confident routing decision.
2. **High-stakes features** — when the feature is critical and final
   approval should be explicit. Pass the review document summary in the
   checkpoint context.
3. **Dimension disagreement** — when review dimensions produce conflicting
   signals (e.g., spec conformance passes but implementation quality fails).

For each scenario, include in the checkpoint context:
- The aggregate verdict and per-dimension verdicts
- A summary of the contentious findings
- The recommended action and why the orchestrator is uncertain

Wait for the human response before proceeding. Do not dispatch remediation
or transition feature state while a checkpoint is pending.

### Post-Review Merge (when verdict is approved or approved_with_followups)

Enter this phase only when the aggregate verdict is `approved` or
`approved_with_followups` and a PR exists for the feature.

The `merging` stage gate is `auto` — no human checkpoint is required, but the
orchestrator must complete all five steps before the feature reaches `done`.

**Step 1: Verify all tasks are terminal**

Confirm every task under the feature is in a terminal state — `done`,
`not-planned`, or `duplicate`. No task may remain in `ready`, `active`,
`needs-review`, or `needs-rework`.

```
entity(action: "list", type: "task", parent: "<feature-id>")
```

If any task is non-terminal, STOP. Do not proceed to merge. Either complete
the remaining tasks or transition them to a terminal state with justification.

**Step 2: Transition feature to `merging`**

Advance the feature from `reviewing` to `merging`:

```
entity(action: "transition", id: "<feature-id>", status: "merging")
```

This transition requires all tasks to be terminal and at least one approved
report document. If the transition fails, check the gate prerequisites and
resolve before retrying.

**Step 3: Run merge check and execute**

Verify the PR exists and check merge gates:

```
pr(action: "status", entity_id: "<feature-id>")
merge(action: "check", entity_id: "<feature-id>")
```

If `merge(action: "check")` reports blocking gates, resolve them before
proceeding. Common blockers: missing PR, CI failures, unreviewed code.

Execute the merge:

```
merge(action: "execute", entity_id: "<feature-id>")
```

If the merge fails, STOP. Review the failure output, resolve the issue, and
retry from step 3. Do not advance the feature to `verifying` until the merge
succeeds.

**Step 4: Verify merge ancestry**

Confirm the feature branch is an ancestor of main:

```
git merge-base --is-ancestor <feature-branch> main
```

If the command exits non-zero, the merge did not complete successfully. The
worktree record will remain `active` and the feature will stay in `merging`.
Investigate and retry from step 3.

If the command exits zero, the merge is confirmed. The worktree record is
automatically marked `merged`.

**Step 5: Build, test, and transition**

Advance the feature from `merging` to `verifying`:

```
entity(action: "transition", id: "<feature-id>", status: "verifying")
```

Run the build from the repository root:

```
go build ./...
```

If the build fails:

```
entity(action: "transition", id: "<feature-id>", status: "needs-rework",
       reason: "Build failed on main after merge: <build-error-output>")
```

STOP. Report the build failure and do not proceed further.

Run the tests from the repository root using the test tool:

```
test(action: "run")
```

(Which runs `go test ./...` internally.)

If any test fails:

```
entity(action: "transition", id: "<feature-id>", status: "needs-rework",
       reason: "Tests failed on main after merge: <test-failure-output>")
```

STOP. Report the test failure and do not proceed further.

If both build and tests pass:

```
entity(action: "transition", id: "<feature-id>", status: "done")
```

Contribute a knowledge entry summarizing the merge:

```
knowledge(action: "contribute",
    topic: "merge-complete-<feature-id>",
    content: "Feature merged and verified. PR: <pr-url>. Build and tests passed on main.",
    scope: "project")
```

### Context Budget Strategy

The orchestrator and sub-agents have deliberately different context profiles:

**Orchestrator** works at metadata level only (~6–14 KB total): feature
entity state, spec outline, task list with file paths, skill document,
collated findings. The orchestrator never reads source code.

**Sub-agents** hold their review unit's context (~12–30 KB per agent):
reviewer profile, skill document, spec section(s), source files, output
template.

This means: orchestrator context cost is constant regardless of codebase
size; sub-agent context scales with review unit size, not feature size.

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
- How do I coordinate an evidence-backed feature review?
