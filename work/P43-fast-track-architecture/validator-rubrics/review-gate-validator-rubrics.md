# Review Gate Validator Rubrics: R2 and R3

This file defines concrete rubrics for the two review-gate-validator checks that rely on
LLM classification rather than structural pattern matching. These rubrics MUST be
consulted by the review-gate-validator agent when evaluating R2 (rubber-stamp detection)
and R3 (severity inflation). Verdicts that cannot be clearly classified as pass or fail
against these rubrics MUST be escalated.

The review-gate-validator is defined in the P43 design doc §"Review Gate Validator." It
inherits from `reviewer` (unlike spec-validator and plan-validator, which inherit from
`base`). Its identity is: "Senior review quality auditor. Verify that a completed review
is thorough, evidence-backed, and suitable for auto-approval. Do not re-review the code —
audit the review process itself."

---

## R2: No Rubber-Stamp Reviews

**Check definition (from P43 design doc §Review Gate Validator):**
No rubber-stamp reviews — a review with zero findings and no evidence is not a valid
review.

**Classification:** Blocking.

### Pass Definition

A review passes R2 when it demonstrates that the reviewer engaged substantively with the
code or document under review. This means the review must satisfy ALL of:

1. **Findings present OR explicit rationale for zero findings.** If the review reports
   zero findings, it MUST contain an explicit explanation of WHY no issues were found.
   This explanation must reference specific aspects of the work that were checked (e.g.,
   "I reviewed all 7 gate implementations for correctness, verified each against its
   spec requirement, and confirmed all test cases pass"). A review that says only
   "LGTM" or "looks good" with zero findings is a rubber stamp.

2. **Evidence of engagement.** The review contains at least one of:
   - Specific code or document sections cited (by line number, section path, or file
     path).
   - Test output or reproduction steps included.
   - Spec traceability table (mapping requirements to verification results).
   - A review history showing multiple rounds with findings resolved across rounds.
   - Concrete observations about the implementation (not just "passes tests").

3. **Reviewer identity and verdict are present.** The review names the reviewer (role
   or identity) and states a clear verdict (pass/fail/needs-rework). An anonymous or
   unsigned review is not a valid review.

A review with nonzero findings automatically satisfies criteria 1 and 2 — the findings
themselves are evidence of engagement. A review with zero findings must satisfy both 1
and 2 explicitly.

#### Positive Examples (Pass)

1. **B32-F1 review report** (guide concept enrichment):
   > **Review History:** Three rounds of review were completed.
   > **Round 1** — 2 major findings: stop words not stripped, slice-alias bug.
   > **Round 2** — 1 residual finding: silent test case change.
   > **Round 3** — finding resolved.
   > **Findings:** 2 blocking (resolved), 1 non-blocking (resolved).
   > **Test Evidence:** `go test ./internal/mcp/...` output with 9 passing tests.
   > **Spec Traceability:** Table mapping REQ-101 through REQ-106 to AC results.

   **Why this passes:** Nonzero findings (3 total across rounds), detailed review
   history, concrete test evidence with command output, spec traceability table.
   Clearly the reviewer engaged deeply.

2. **Hypothetical zero-finding review with explicit rationale:**
   > **Findings:** None.
   > **Rationale for zero findings:** This is a one-line documentation change
   > correcting a typo in `README.md` line 42. I verified the spelling against the
   > Merriam-Webster dictionary, confirmed no other occurrences of the misspelling
   > exist in the repository via `grep`, and verified the markdown renders correctly.
   > The change has no functional impact.
   > **Verdict:** Pass.

   **Why this passes:** Zero findings but explicit rationale references the specific
   change, the verification steps taken, and justifies why no further review is needed.
   This is NOT a rubber stamp — it's a proportionate review for a trivial change.

3. **B31-F1 review report** (hypothetical structure matching real patterns):
   > **Reviewer:** orchestrator
   > **Verdict:** pass
   > **Findings:**
   > **F-001 (minor):** Task 3 `DocService` interface does not document that nil
   >   receivers must be handled — added nil-safety note.
   > **F-002 (minor):** Task 7 test plan does not cover the nil-DocSvc case — test
   >   case added.
   > **Test Evidence:** All 14 new tests pass, pre-existing tests pass.

   **Why this passes:** Two findings (non-zero), both specific and actionable, with
   test evidence. Reviewer identity and verdict present.

#### Negative Examples (Fail)

1. **"LGTM :shipit:"**

   **Why this fails:** Zero findings, zero evidence, no rationale for why no findings
   exist. The reviewer did not engage — this is a textbook rubber stamp.

2. **"Reviewed the code. Everything looks fine. Approved."**

   **Why this fails:** Zero findings, no evidence of what was reviewed, no specific
   observations. "Everything looks fine" is not a rationale — it's a restatement of
   the verdict. No spec traceability, no test evidence, no code references.

3. **"All tests pass. No issues found. ✅"**

   **Why this fails:** Zero findings. While "all tests pass" is technically evidence,
   it's the bare minimum — a CI pipeline could say the same thing. A review must add
   human judgment beyond what automation provides. If the review only reports test
   results without any human analysis, it's a rubber stamp.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **Single trivial finding.** The review has exactly one finding, and it's cosmetic
  (e.g., "fix a typo in a comment" or "add a missing newline"). The finding IS
  evidence of engagement, but barely. Escalate with a note: was the review proportionate
  to the change complexity? If the change was trivial (one-line fix), one cosmetic
  finding may be appropriate. If the change was substantial, escalate for human
  judgment.

- **Review is test-evidence-only.** The review cites test output but makes no
  observations about code quality, design adherence, or spec traceability. Test results
  are machine-checkable — a human review should add something beyond them. Escalate:
  does the review contain any human analysis, or is it just test result transcription?

- **Multi-reviewer scenario where one reviewer rubber-stamps.** If the review process
  involves multiple specialist reviewers and one of them produces zero findings with no
  rationale, but others produce findings — does the overall review pass R2? Escalate:
  the rubber-stamp reviewer did not fulfill their role, but the aggregate review may
  still have adequate coverage via other reviewers.

---

## R3: No Severity Inflation

**Check definition (from P43 design doc §Review Gate Validator):**
No severity inflation — more than 40% of findings classified as blocking suggests the
reviewer is inflating severity rather than distinguishing blocking from non-blocking
issues.

**Classification:** Non-blocking.

### Pass Definition

A review passes R3 when the ratio of blocking findings to total findings is ≤ 40%. That
is:

```
blocking_count / total_findings ≤ 0.40
```

Where:
- `blocking_count` = number of findings classified as "blocking," "major," or equivalent
  (the exact label may vary by reviewer role — consult the review template).
- `total_findings` = total number of findings across all severity levels, including
  "non-blocking," "minor," "nit," and "suggestion."

If `total_findings = 0`, the review has no findings at all, and R3 is vacuously passed
(R2 will catch the rubber stamp, but R3 is not triggered by zero findings).

A review with 1–2 total findings and 1 blocking finding has a 50–100% blocking ratio.
This is technically above 40%, but with such a small sample the ratio is fragile. See
Borderline → Escalate below.

#### Positive Examples (Pass)

1. **B32-F1 review report:**
   > **Findings:**
   > - F-001 (major/blocking): Stop words not stripped.
   > - F-002 (major/blocking): Slice-alias bug.
   > - F-003 (minor/non-blocking): Silent test case change.

   **Blocking ratio:** 2 / 3 = 67%. **Wait — this fails?** No: the findings span 3
   review rounds. Across the entire review process, 2 of 3 findings were blocking
   (67%). But the check evaluates the FINAL review state, not the aggregate across
   rounds. If the resolved findings are still reported in the final review, they count.
   In this case, all 3 findings are listed in the final report, and 2 are blocking
   → 67% > 40%.

   **Re-evaluation:** This example actually FAILS R3 by the strict rubric — 67%
   blocking ratio exceeds 40%. However, the review-gate-validator should consider
   context: both blocking findings were in Round 1 and were resolved by Round 3. The
   blocking ratio in the final round was 0% (0 blocking, 1 non-blocking). The
   validator should evaluate the review's FINAL disposition, not its work-in-progress
   history. If all blocking findings are resolved, they should be weighted differently
   than unresolved blocking findings. This nuance should be captured in a validator
   note.

2. **Hypothetical review with balanced severity:**
   > **Findings:**
   > - F-001 (blocking): Missing error handling in payment path.
   > - F-002 (non-blocking): Variable naming could be clearer.
   > - F-003 (non-blocking): Missing comment on exported function.
   > - F-004 (non-blocking): Test could use table-driven pattern.
   > - F-005 (minor): Extra blank line in imports.

   **Blocking ratio:** 1 / 5 = 20%. **Passes.** The reviewer distinguished the one
   genuinely blocking issue (error handling) from the four non-blocking ones.

3. **Hypothetical review with all non-blocking:**
   > **Findings:**
   > - F-001 (minor): Comment typo in `gate.go:42`.
   > - F-002 (minor): Missing newline at EOF in `checker.go`.

   **Blocking ratio:** 0 / 2 = 0%. **Passes.** The reviewer correctly identified that
   neither issue blocks merge.

#### Negative Examples (Fail)

1. **"All findings are blocking because quality matters":**
   > **Findings:**
   > - F-001 (blocking): Variable name `x` should be `entityCount`.
   > - F-002 (blocking): Missing comment on `merge.Execute`.
   > - F-003 (blocking): Test could use a helper function.

   **Blocking ratio:** 3 / 3 = 100%. **Fails.** Naming, comments, and test helpers
   are not blocking issues — they do not affect correctness or safety. The reviewer
   is inflating severity, treating every observation as a merge-blocker.

2. **"3 of 4 findings are blocking":**
   > **Findings:**
   > - F-001 (blocking): Potential nil dereference in edge case.
   > - F-002 (blocking): Missing input validation.
   > - F-003 (blocking): Race condition in concurrent access.
   > - F-004 (non-blocking): Log message grammar.

   **Blocking ratio:** 3 / 4 = 75%. **Fails** unless the reviewer provides specific
   justification for why each blocking finding is genuinely blocking. Three correctness/
   safety findings could all be genuine blockers, but at 75% the validator should
   scrutinize: is the "race condition" a real data race, or is it a theoretical concern
   about a single-goroutine path? Is "missing input validation" a security boundary or
   an internal function? The ratio alone triggers a finding — the human must verify
   the severity assignments.

3. **"Everything is critical":**
   > **Findings:**
   > - F-001 (blocking): README formatting inconsistent.
   > - F-002 (blocking): Changelog entry missing.

   **Blocking ratio:** 2 / 2 = 100%. **Fails.** Documentation formatting issues are
   not blocking — they do not prevent merge. The reviewer is inflating severity.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **Small sample size (1–3 total findings).** With 1–2 findings, a single blocking
  finding pushes the ratio to 50–100%. This is not necessarily inflation — it could be
  that the only issues found were genuinely blocking. Escalate with the total count and
  a note: "Sample too small for ratio to be meaningful. Human should verify the
  blocking findings are genuinely blocking."

- **All blocking findings are resolved.** A review with 3 blocking findings (100% ratio)
  that were all resolved by the final round is very different from a review with 3
  unresolved blocking findings. The resolved findings still count toward the ratio in
  the final report, but the VALIDATOR SHOULD ADD A NOTE: "All blocking findings were
  resolved by the final review round. The review process was effective despite the
  high blocking ratio in early rounds." This is not a pass — it's an escalated note
  for the human to consider.

- **Domain-appropriate high blocking ratio.** A security review may legitimately have
  a high blocking ratio — security findings are often blocking by nature. If the
  reviewer is `reviewer-security`, escalate rather than failing: "High blocking ratio
  may be appropriate for a security review. Human should verify severity assignments."
  Similarly for `reviewer-conformance` reviews where spec non-conformance is
  inherently blocking.

- **Unclear severity labels.** The review uses non-standard severity labels (e.g.,
  "important," "should-fix," "nice-to-have") that don't map cleanly to
  blocking/non-blocking. Escalate with the label mapping you inferred and ask the
  human to confirm.
