# Plan Validator Rubrics: D7, D10, and D13

This file defines concrete rubrics for the three plan-validator checks that rely on LLM
classification rather than structural pattern matching. These rubrics MUST be consulted
by the plan-validator agent when evaluating D7 (monolithic task detection), D10 (risk
assessment non-empty), and D13 (actionable task description). Verdicts that cannot be
clearly classified as pass or fail against these rubrics MUST be escalated.

---

## D7: No Monolithic Tasks

**Check definition (from plan-validator role):**
No monolithic tasks — no task touches more than 3 files or addresses more than 1
acceptance criterion.

**Classification:** Non-blocking.

### Pass Definition

A task passes D7 when it satisfies BOTH conditions:

1. **File count ≤ 3.** The task's deliverables list (explicitly stated or implied by
   the task description) references at most 3 distinct files. "Files" means source files
   in the repository — test files are counted. Configuration files, templates, and
   documentation files are counted if the task explicitly lists them as deliverables.

2. **AC count ≤ 1.** The task maps to at most 1 acceptance criterion from the parent
   specification. A task that implements a shared prerequisite for multiple ACs (e.g.,
   "add the `Bypassable` field to the struct" which enables several ACs downstream) is
   counted as addressing the first AC it directly produces. The rule is: each task
   should be accountable for producing ONE verifiable outcome, not several.

A task that exceeds either threshold (more than 3 files OR more than 1 AC) triggers a D7
finding. The finding is non-blocking — the plan can proceed — but it signals that the task
may be too large for a single agent to complete reliably.

#### Positive Examples (Pass)

1. **B31-F1 Task 2** (backfill Bypassable on existing gates):
   > Update every existing `Check()` method in `internal/merge/gates.go` to explicitly
   > set `Bypassable: true` in the initial `GateResult` struct literal. Also scan
   > `internal/merge/*_test.go` for any `GateResult{…}` struct literals...
   >
   > **Deliverable:** `internal/merge/gates.go` with all 7 gates returning
   > `Bypassable: true`; any affected test files updated.

   **Why this passes:** Touches 2 files (`gates.go` and `*_test.go`). The glob
   `internal/merge/*_test.go` may expand to multiple test files, but conceptually this
   is one deliverable category (test files), and the scope is ≤ 3 distinct files in
   practice. Maps to 1 AC (the bypassable backfill).

2. **B31-F1 Task 4** (implement ReviewReportExistsGate):
   > Add the `ReviewReportExistsGate` struct and its `Gate` interface implementation
   > to `internal/merge/gates.go`...
   >
   > **Deliverable:** `internal/merge/gates.go` with `ReviewReportExistsGate` fully
   > implemented.

   **Why this passes:** Single file (1 file). Single AC (gate implementation). Well
   within both thresholds.

3. **B32-F1 Task 7** (heading patterns for suggested_classifications):
   > Add Problem and Motivation, Decisions, Design, Overview/Summary,
   > Requirements/Goals, Risk/Risks, Definition/Glossary heading patterns to
   > suggested_classifications.
   >
   > **Deliverable:** `internal/mcp/doc_intel_tool.go` updated.

   **Why this passes:** Single file, single concern (heading pattern expansion). Maps
   to 1 AC (REQ-106 heading coverage).

#### Negative Examples (Fail)

1. **Hypothetical monolithic task:**
   > Implement the entire merge gate pipeline: add `Bypassable` to `GateResult`,
   > update all 7 gates, create `DocService` interface, implement
   > `ReviewReportExistsGate`, wire into `DefaultGates()`, thread `DocSvc` through
   > merge tool, add unit tests for everything.
   >
   > **Deliverables:** `gate.go`, `gates.go`, `checker.go`, `merge_tool.go`,
   > `server.go`, `gates_test.go`, `merge_tool_test.go`.

   **Why this fails:** 7 files, covers at least 3 ACs (gate implementation, wiring,
   threading). This is essentially the entire feature in one task.

2. **Hypothetical multi-AC task:**
   > Implement all enrichment features from the spec: concepts_suggested derivation,
   > suggested_classifications expansion, and stop-word filtering.
   >
   > **Deliverable:** `internal/mcp/doc_intel_tool.go`.

   **Why this fails:** Even though it touches only 1 file, it maps to at least 3
   distinct ACs (concepts_suggested field, heading patterns, stop-word filtering).

3. **Hypothetical file-sprawl task:**
   > Add validation logging across the codebase.
   >
   > **Deliverables:** `internal/merge/gate.go`, `internal/merge/gates.go`,
   > `internal/merge/checker.go`, `internal/mcp/merge_tool.go`,
   > `internal/mcp/server.go`, `internal/docint/*.go`.

   **Why this fails:** 5+ files, and the glob implies potentially many more. The task
   is unbounded.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **Glob ambiguity.** A task's deliverables include a glob pattern (e.g.,
  `internal/merge/*_test.go` or `internal/docint/*.go`). If the glob could expand to
  more than 3 files in practice, escalate with a note asking the human to enumerate
  the specific files or split the task.

- **Shared prerequisite tasks.** A task adds infrastructure used by multiple downstream
  ACs (e.g., "add the `DocService` interface to `GateContext`"). This task doesn't
  directly produce any AC — it enables others. Count it as addressing 0 ACs for D7
  purposes (passes the AC threshold), but escalate if the file count exceeds 2 — the
  task may be trying to do too much structural work at once.

- **Test-as-part-of-task.** A task's description says "write unit tests for X" as part
  of the same task that implements X. If X itself spans 3 files and the tests add 2
  more, the combined task touches 5 files. Escalate: the plan author should consider
  whether tests should be a separate task.

---

## D10: Risk Assessment Is Non-Empty

**Check definition (from plan-validator role):**
Risk assessment section is non-empty and identifies at least one risk.

**Classification:** Non-blocking.

### Pass Definition

The Risk Assessment section passes D10 when it:

1. **Is present.** The dev-plan document has a section named "Risk Assessment" (or
   substantially equivalent — "Risks", "Risk Analysis").
2. **Is non-empty.** The section contains at least one named risk entry.
3. **Each risk is substantive.** Each risk entry identifies at minimum: (a) what could
   go wrong, (b) a probability or likelihood, (c) an impact if it occurs, and (d) a
   mitigation strategy or why the risk is accepted. A risk that says only
   "Low risk — nothing to worry about" does NOT count.
4. **At least one risk is real.** At least one identified risk is specific to the
   feature — it references a concrete concern (a file path, a dependency, a test
   interaction). Generic risks like "schedule risk" or "requirements may change" do
   NOT satisfy this alone; they must be accompanied by at least one feature-specific
   risk.

A risk assessment that lists only generic project-management risks without any
feature-specific risks triggers D10.

#### Positive Examples (Pass)

1. **B31-F1 Risk: Go zero-value breaks existing gate tests:**
   > **Probability:** High. **Impact:** Medium (CI failure, not a logic error).
   > **Mitigation:** Task 2 explicitly requires scanning
   > `internal/merge/*_test.go` for `GateResult{}` literals and adding
   > `Bypassable: true`. Run `go test ./internal/merge/…` after Task 1 is committed
   > to surface failures before proceeding to later tasks.
   > **Affected tasks:** T1, T2, T8.

   **Why this passes:** Specific technical risk with probability, impact, concrete
   mitigation, and affected task references. Feature-specific.

2. **B31-F1 Risk: DocService interface mismatch:**
   > **Probability:** Medium. **Impact:** Medium (compilation failure; no logic risk).
   > **Mitigation:** The adapter shim in Task 6 is the only coupling point... Verify
   > compilation after Task 6 before writing tests.
   > **Affected tasks:** T3, T6.

   **Why this passes:** Identifies a specific interface compatibility risk with
   concrete mitigation steps and affected tasks.

3. **B31-F1 Risk: Exact REQ-007 message wording drifts between gate and merge tool:**
   > **Probability:** Low. **Impact:** Medium (AC-007 inspection test will fail;
   > user-facing message is wrong).
   > **Mitigation:** The full REQ-007 message string is produced once... Define the
   > message as a named constant or package-level var in `gates.go` so that both the
   > gate and the test reference the same source.

   **Why this passes:** Specific, traceable to a requirement ID, with a concrete
   implementation-level mitigation.

#### Negative Examples (Fail)

1. **Empty risk assessment:**
   > ## Risk Assessment
   >
   > This is a small change. No significant risks.

   **Why this fails:** No risks identified. "No significant risks" is not a risk
   identification — it's a refusal to identify risks.

2. **Generic-only risks:**
   > ## Risk Assessment
   >
   > **Schedule risk:** The feature may take longer than expected.
   > **Requirement change risk:** Requirements may change during implementation.

   **Why this fails:** Both risks are generic and could apply to any feature. Neither
   references a specific file, dependency, test, or technical concern. No feature-
   specific risk is present.

3. **Stub risks without substance:**
   > ## Risk Assessment
   >
   > - **Risk 1:** Something might break. Low risk. We'll fix it if it does.
   > - **Risk 2:** Tests might fail. Low risk. We'll fix them.

   **Why this fails:** No probability assessment, no concrete impact description, no
   mitigation beyond "we'll fix it." Neither risk is substantive.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **One real risk, rest generic.** The assessment has one feature-specific risk and
  several generic ones. Does this pass D10? The rubric says "at least one risk is
  real," so this technically passes — but if the one real risk is trivial (e.g., "a
  typo in the error message") and the task complexity warrants more analysis, escalate
  with a note that the risk depth is shallow.

- **Risk accepted without mitigation.** A risk entry says "Accepted — no mitigation
  planned" with a clear rationale (e.g., "one-line change, rollback is trivial").
  This is valid risk management — not every risk needs mitigation. But if the
  rationale is missing or unconvincing, escalate.

- **Implied risks in task descriptions.** The task breakdown itself describes
  mitigations (e.g., "run tests after Task 1") without a corresponding Risk Assessment
  entry. The plan author may have done the risk analysis but not written it down.
  Escalate with a note pointing out the gap — the mitigation exists but the risk
  should be explicitly stated in the Risk Assessment section.

---

## D13: Every Task Has an Actionable Description

**Check definition (from plan-validator role):**
Every task has an actionable description ≥ 50 words, states what it produces, what
inputs it requires, and what "done" means beyond the AC.

**Classification:** Blocking.

### Pass Definition

A task description passes D13 when it satisfies ALL of:

1. **Word count ≥ 50.** The task description body (excluding metadata like "Task ID,"
   "Depends on," "Effort," and "Deliverable" labels — but including the description
   prose) contains at least 50 words. Count words by splitting on whitespace.

2. **States what it produces.** The description names at least one concrete deliverable.
   This can be a file path, a code artifact, a test suite, a document — something a
   reviewer can point at and say "that was produced." The "Deliverable:" line alone
   counts if it is specific.

3. **States what inputs it requires.** The description identifies what the implementer
   needs before starting: a completed predecessor task, a specific design section, a
   spec requirement ID, or an existing file to modify. "None" is acceptable only if
   explicitly stated and genuinely true (e.g., the first task in a dependency chain).

4. **States what "done" means beyond the AC.** The description includes a completion
   criterion that goes beyond the parent spec's acceptance criterion. This can be:
   - A specific test command that must pass (e.g., `go test ./internal/merge/...`)
   - A compilation check (e.g., "project builds without errors")
   - A review requirement (e.g., "code reviewed by a second developer")
   - A specific assertion about the deliverable (e.g., "all 7 gates return
     `Bypassable: true`")

   The "done" criterion must be verifiable by the implementer without human judgment.

A task that meets all four criteria passes D13. A task that fails any one criterion
fails D13, which is a blocking finding — the plan cannot advance to `developing` until
the task description is remediated.

#### Positive Examples (Pass)

1. **B31-F1 Task 2** (backfill Bypassable):
   > Update every existing `Check()` method in `internal/merge/gates.go` to
   > explicitly set `Bypassable: true` in the initial `GateResult` struct literal.
   > The seven gates are: `EntityDoneGate`, `TasksCompleteGate`,
   > `VerificationExistsGate`, `VerificationPassedGate`, `BranchNotStaleGate`,
   > `NoConflictsGate`, `HealthCheckCleanGate`. Each requires a single-line addition.
   > Also scan `internal/merge/*_test.go` for any `GateResult{…}` struct literals
   > constructed without `Bypassable` and add `Bypassable: true` to preserve test
   > intent; missing the field would cause tests constructing expected `GateResult`
   > values to fail after Task 1.

   **Why this passes:** ~110 words. **Produces:** updated `gates.go` and test files.
   **Inputs:** Task 1 (depends on). **Done beyond AC:** specifies running `go test`
   and verifies that test failures are surfaced. Task explicitly names all 7 gates
   and describes the test scan.

2. **B31-F1 Task 4** (implement ReviewReportExistsGate):
   > Add the `ReviewReportExistsGate` struct and its `Gate` interface
   > implementation to `internal/merge/gates.go`:
   > - `Name()` returns `"review_report_exists"`.
   > - `Severity()` returns `GateSeverityBlocking`.
   > - `Check(ctx GateContext) GateResult`:
   >   1. Read `ctx.Entity["status"]`. If it is not `"reviewing"`, return Pass
   >      immediately...
   > The `Message` field MUST contain the exact wording from REQ-007...

   **Why this passes:** ~110 words. **Produces:** `gates.go` with full gate
   implementation. **Inputs:** Tasks 1 and 3 (depends on). **Done beyond AC:**
   gate passes its own unit tests, message wording matches REQ-007 exactly.

3. **B31-F1 Task 7** (unit tests for ReviewReportExistsGate):
   > Add test functions to `internal/merge/gates_test.go` covering:
   > - Gate skips when entity status is not `"reviewing"` — AC-005.
   > - Gate returns Pass when `DocSvc.ListDocuments` returns ≥ 1 report...
   > Use a table-driven test with a stub `DocService` implementation local to the
   > test file. Do not use `DefaultGates()`; test the gate struct directly.

   **Why this passes:** ~85 words. **Produces:** test file with specific test cases.
   **Inputs:** Tasks 4, 5 (depends on). **Done beyond AC:** table-driven tests,
   stub `DocService`, all ACs covered, `go test` passes.

#### Negative Examples (Fail)

1. **"Implement the ReviewReportExistsGate."**

   **Why this fails:** ~4 words (far below 50). Does not state what it produces
   (which file?), what inputs it needs, or what "done" means. Completely
   non-actionable.

2. **"Write tests for the merge tool changes. See spec for what to test."**

   **Why this fails:** ~14 words. Does not state which test file, which ACs are
   covered, or what "done" means. Delegates understanding to the reader ("see spec")
   — the task description should be self-contained enough to act on.

3. **"Update the code to support the new bypassable field. Modify the struct, update
   the gates, and make sure everything still works."**

   **Why this fails:** ~25 words. Vague about what is produced ("update the code" —
   which files?), no inputs listed, "make sure everything still works" is not a
   verifiable "done" criterion — it requires the implementer to guess what "works"
   means.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **Word count is borderline (45–55 words).** The 50-word threshold is a heuristic.
  A 48-word description that is extremely specific may be more actionable than a
  55-word description that is vague. If the word count is within 10 words of the
  threshold, evaluate specificity: does the description give the implementer
  everything they need to start work? If yes, pass. If uncertain, escalate with the
  word count and your specificity assessment.

- **Done criterion is implicit.** The "done" criterion is "task produces the
  deliverable" — which is just restating what the task produces. This doesn't add
  new information. The "done" criterion must add something beyond "the thing exists."
  But if the deliverable itself implies a verifiable check (e.g., "file compiles
  without errors" is implicit when the deliverable is a Go file), escalate rather
  than failing — the implicit criterion may be sufficient.

- **Inputs are "see dependency graph."** The task says "Depends on: Task 4" but does
  not say WHAT from Task 4 is needed (a struct? a function? a test pattern?). If the
  dependency is listed but not described, escalate — the human should clarify what
  the implementer needs from the predecessor.
