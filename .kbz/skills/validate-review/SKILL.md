---
name: validate-review
description:
  expert: "Audit a completed review for thoroughness, evidence quality, and auto-approval suitability. Executes 8 validation checks (R1–R8) against a review report, producing a summary verdict and a full per-check analysis. Does not re-review the code — validates the review process itself."
  natural: "Check whether a review is thorough enough for auto-approval, or validate that a review meets the quality bar"
triggers:
  - validate a review
  - audit a review for quality
  - check if a review can be auto-approved
  - run the review gate validator
  - verify the review process was thorough
roles: [review-gate-validator]
stage: reviewing
constraint_level: low
---

## Vocabulary

- **review-gate-validator** — the role that audits a completed review for process quality; inherits from reviewer, not base
- **blocking check** — a validation check whose failure prevents stage advancement (R1, R2, R4, R5, R7)
- **non-blocking check** — a validation check whose failure produces findings but does not prevent stage advancement (R3, R6, R8)
- **rubber-stamp review** — a review with zero findings and no evidence of reviewer engagement; the #1 quality failure in multi-agent review systems
- **severity inflation** — classifying more than 40% of findings as blocking, indicating the reviewer is not distinguishing blocking from non-blocking issues
- **per-dimension outcome** — the verdict (pass/pass_with_notes/concern/fail) assigned to each review dimension independently
- **aggregate verdict** — the overall review verdict derived from per-dimension outcomes
- **deduplication pass** — a post-review step that merges or deconflicts overlapping findings from multiple specialist reviewers
- **spec anchor** — a citation of a specific spec requirement (by REQ-ID or AC number) that a finding references as its basis
- **adaptive reviewer selection** — choosing which specialist reviewers to engage based on the change scope rather than always using the full panel
- **review unit** — the code, document, or component being reviewed; a single review covers one review unit
- **finding classification** — the structured metadata attached to each finding: severity (blocking/non-blocking), dimension, evidence, and remediation
- **fresh session** — a clean context containing only the review document, rubric references, and parent spec; no conversation history from the review author
- **escalate** — flag a borderline case for human judgment rather than forcing a pass/fail decision
- **verdict** — the validator's overall judgment: pass, pass_with_notes, or fail
- **evidence score** — a qualitative assessment of how well the review supports its claims with citations, test output, or spec traceability
- **dimension coverage** — whether every acceptance criterion is assigned to at least one specialist reviewer

## Anti-Patterns

### Re-Reviewing the Code
- **Detect:** The validator opens source files, runs tests, or evaluates implementation correctness instead of auditing the review document
- **BECAUSE:** Re-reviewing duplicates the specialist reviewers' work and defeats the purpose of the review-gate-validator as a process auditor — the validator does not have the specialist context to re-review correctly, and doing so masks process failures (a thorough validator could compensate for a rubber-stamp review, creating false confidence)
- **Resolve:** Read only the review report, the parent specification, and the rubric references. If the review claims a file was checked, verify the claim is present in the review text — do not verify the file yourself

### Rubber-Stamp Acceptance
- **Detect:** The validator approves a review that has zero findings and no explicit rationale for why zero findings is appropriate
- **BECAUSE:** MAST FM-3.1 identifies LLM sycophancy making approval the path of least resistance — a validator that rubber-stamps reviews becomes the single point of quality failure because all downstream stages trust the validator's verdict
- **Resolve:** For R2, require the review to satisfy all three criteria: findings present OR explicit rationale for zero findings, evidence of engagement, and reviewer identity with verdict. If any criterion fails, R2 fails

### Content Judgment
- **Detect:** The validator evaluates whether the review's findings are *correct* rather than whether the review process is *thorough* — e.g., "this finding about error handling is wrong because the code uses a different pattern"
- **BECAUSE:** The validator is a process auditor, not a code reviewer. Judging finding correctness requires specialist domain knowledge the validator does not have and creates rework loops where the validator and reviewer disagree on substance rather than process
- **Resolve:** Evaluate only whether the review process was thorough: Were findings structured? Was evidence cited? Were spec requirements referenced? Do not evaluate whether findings are technically correct

### Hallucinated Completeness
- **Detect:** The validator reports a check as passed without having enumerated the specific evidence from the review document — e.g., claiming R1 passes because "the review looks thorough" without listing each reviewer's structured output
- **BECAUSE:** LLM tendency toward affirmative completion means checks pass by default when evidence is not explicitly searched for — this is the validator-equivalent of rubber-stamp review and defeats the gate entirely
- **Resolve:** For each check, enumerate the specific evidence found in the review document before declaring pass/fail. If R7 requires covering every acceptance criterion, list each AC and the reviewer assigned to it. Report counts: "3 of 5 ACs covered"

### Severity Second-Guessing
- **Detect:** The validator reclassifies individual findings' severity (e.g., "this should be non-blocking, not blocking") during R3
- **BECAUSE:** R3 checks the *aggregate ratio* of blocking to total findings — it does not reclassify individual findings. Individual severity reassessment requires specialist domain knowledge the validator does not have, and the 40% threshold was chosen to allow legitimate blocking-heavy reviews (like security reviews) to pass while flagging systemic inflation
- **Resolve:** Apply the 40% threshold to the ratio as reported. Do not reclassify individual findings. For borderline cases (small sample, domain-appropriate high ratio), escalate rather than fail

### Missing Escalation
- **Detect:** The validator forces a pass/fail decision on a borderline case that the rubrics say should be escalated — e.g., a 1-finding review where the single finding is cosmetic and the rubrics say to escalate
- **BECAUSE:** Borderline cases are borderline because the rubrics cannot resolve them deterministically — forcing a binary outcome on an ambiguous case produces unreliable verdicts that erode trust in the validator
- **Resolve:** When the rubrics say "escalate," produce an escalated finding with the specific reason and defer to human judgment. Escalated findings do not block stage advancement but are visible in the validator report

### Unanchored R4 Pass
- **Detect:** The validator passes R4 because blocking findings "look like they cite specs" without cross-referencing the cited REQ-IDs against the parent specification
- **BECAUSE:** Fabricated spec references are a known failure mode — a reviewer can claim a finding cites REQ-042 when that requirement does not exist in the spec. Accepting unverified references creates false traceability and masks spec gaps
- **Resolve:** For R4, cross-reference every cited REQ-ID, AC number, or spec section reference in blocking findings against the parent specification document. Flag any unmatched reference as an R4 failure

## Checklist

Copy this checklist into the validator session and check off each item as completed:

```
Review Gate Validator Checklist
- [ ] Read the review report (full document)
- [ ] Read the parent specification (for R4 and R7 cross-referencing)
- [ ] Read the R2 and R3 rubrics from validator-rubrics/review-gate-validator-rubrics.md
- [ ] R1: Verify every reviewer produced structured output with evidence — enumerate reviewers
- [ ] R2: Check for rubber-stamp reviews — apply R2 rubric
- [ ] R3: Check for severity inflation — apply R3 rubric
- [ ] R4: Verify every blocking finding cites a spec requirement — cross-reference with spec
- [ ] R5: Verify aggregate verdict is consistent with per-dimension outcomes
- [ ] R6: Check whether a deduplication pass was run
- [ ] R7: Verify every acceptance criterion is covered by at least one reviewer
- [ ] R8: Verify reviewer selection was adaptive to change scope
- [ ] Produce summary verdict (pass/pass_with_notes/fail)
- [ ] Write full report with per-check analysis
```

## Procedure

1. **Read the review report.** Load the complete review document — this is the input being validated. Identify: the review unit, the reviewer roles, the per-dimension outcomes, all findings with their classifications, and the aggregate verdict. IF the review document cannot be found or is incomplete, STOP and ask for the correct review document path.

2. **Read the parent specification.** Load the specification document that the review was conducted against. Extract: all REQ-IDs, all acceptance criteria (AC) with their identifiers, and the scope section. This document is needed for R4 (cross-referencing spec citations) and R7 (AC coverage). IF the parent specification cannot be found, STOP and ask for the spec document path.

3. **Read the R2 and R3 rubrics.** Load `work/P43-fast-track-architecture/validator-rubrics/review-gate-validator-rubrics.md`. R2 and R3 are the only checks that rely on LLM classification rather than structural pattern matching — the rubrics define pass/fail definitions, positive/negative examples, and escalation patterns. Do not evaluate R2 or R3 from memory.

4. **Execute R1–R8 in order.** For each check, enumerate the specific evidence from the review document before declaring pass/fail. Use the rubric definitions for R2 and R3. For R4, cross-reference every cited REQ-ID against the parent specification. For R7, enumerate every AC against the reviewers who cover it. Report exact counts, not estimates.

5. **Classify each check result.** For each of R1–R8, assign: **pass** (check satisfied with specific evidence), **fail** (check failed with specific reasons), or **escalate** (borderline case per rubric guidance, deferred to human). Do not skip checks — even non-blocking checks must be evaluated.

6. **Determine the aggregate verdict.** Apply these rules:
   - **fail:** any blocking check (R1, R2, R4, R5, R7) failed
   - **pass_with_notes:** no blocking checks failed, but at least one non-blocking check (R3, R6, R8) failed OR at least one check was escalated
   - **pass:** all eight checks passed with no escalations
   IF the verdict is fail, include which blocking check(s) failed and what must be fixed.

7. **Produce two outputs.** Write a **summary** (≤300 words) with verdict, count of blocking/non-blocking failures, evidence score (qualitative: strong/adequate/weak), and reference to the full report path. Write a **full report** with per-check analysis, evidence citations, uncertain findings, and R2/R3 rubric application notes. The summary goes to the orchestrator; the full report is written to disk and registered.

## Output Format

### Summary (to orchestrator)

```
Verdict: pass | pass_with_notes | fail
Blocking failures: N (R1,R2,R4,R5,R7) — list which failed
Non-blocking failures: N (R3,R6,R8) — list which failed
Escalations: N — list which checks
Evidence score: strong | adequate | weak
Full report: work/{feature}/reports/validate-review-{timestamp}.md
```

### Full Report (to document store)

```
# Review Gate Validation Report

**Feature:** {feature-id}
**Review document:** {review-doc-path}
**Parent specification:** {spec-doc-path}
**Validator:** review-gate-validator
**Timestamp:** {ISO 8601}

## Verdict

{pass | pass_with_notes | fail}
{One-sentence summary of why}

## Per-Check Analysis

### R1: Structured Output from Every Reviewer — {pass | fail | escalate}
**Evidence:**
- Reviewer {role}: {has structured output? findings count?}
- Reviewer {role}: {has structured output? findings count?}
**Result:** {pass/fail/escalate with reasoning}

### R2: No Rubber-Stamp Reviews — {pass | fail | escalate}
**Rubric application:**
- Criterion 1 (findings or explicit rationale): {met/not met — cite review text}
- Criterion 2 (evidence of engagement): {met/not met — cite review text}
- Criterion 3 (reviewer identity and verdict): {met/not met — cite review text}
**Result:** {pass/fail/escalate with reasoning}

### R3: No Severity Inflation — {pass | fail | escalate}
**Rubric application:**
- Blocking findings: {count}
- Total findings: {count}
- Blocking ratio: {percentage}%
- Threshold: 40%
**Result:** {pass/fail/escalate with reasoning}
{Escalation note if applicable: small sample, domain-appropriate, resolved findings, etc.}

### R4: Blocking Findings Cite Spec Requirements — {pass | fail | escalate}
**Evidence:**
- Blocking finding F-00N: cites {REQ-ID/AC/section} — {verified in spec | NOT FOUND in spec}
{Repeat for each blocking finding}
**Result:** {pass/fail/escalate with reasoning}

### R5: Aggregate Verdict Consistency — {pass | fail | escalate}
**Evidence:**
- Per-dimension outcomes: {list each dimension and its verdict}
- Aggregate verdict: {overall verdict from review}
- Consistency check: {do the per-dimension outcomes support the aggregate?}
{Note any dimension with concern or fail that was overridden in the aggregate}
**Result:** {pass/fail/escalate with reasoning}

### R6: Deduplication Pass — {pass | fail | escalate}
**Evidence:**
- Number of reviewers: {N}
- Overlapping findings detected: {yes/no — cite examples}
- Deduplication pass documented: {yes/no — cite review text}
**Result:** {pass/fail/escalate with reasoning}

### R7: Acceptance Criterion Coverage — {pass | fail | escalate}
**Evidence:**
- AC-1: covered by {reviewer role(s)} — {evidence from review}
- AC-2: covered by {reviewer role(s)} — {evidence from review}
{Repeat for each AC}
- Uncovered ACs: {list any AC with no reviewer assigned}
**Result:** {pass/fail/escalate with reasoning}

### R8: Adaptive Reviewer Selection — {pass | fail | escalate}
**Evidence:**
- Change scope: {summary of what changed — files, components, concerns}
- Reviewers selected: {list roles}
- Appropriateness: {does the panel match the scope? e.g., security reviewer selected when auth code changed? documentation-only change not sent to full panel?}
**Result:** {pass/fail/escalate with reasoning}

## Finding Summary

| Check | Classification | Result | Notes |
|-------|---------------|--------|-------|
| R1    | Blocking       |        |       |
| R2    | Blocking       |        |       |
| R3    | Non-blocking   |        |       |
| R4    | Blocking       |        |       |
| R5    | Blocking       |        |       |
| R6    | Non-blocking   |        |       |
| R7    | Blocking       |        |       |
| R8    | Non-blocking   |        |       |
```

## Examples

### BAD: Validator re-reviews the code

```
R1: PASS — I reviewed feature.go and the error handling looks correct.
The implementation matches what I would expect.
```

**WHY BAD:** The validator opened the source file and evaluated the code — this is re-reviewing. R1 checks whether every reviewer produced structured output with evidence, not whether the code is correct. The validator has no business reading feature.go.

### BAD: Hallucinated R7 pass

```
R7: PASS — All acceptance criteria appear to be covered by the review panel.
```

**WHY BAD:** No enumeration of acceptance criteria. No mapping of ACs to reviewers. The word "appear" signals that the validator did not actually check — this is hallucinated completeness. A machine or human cannot verify this claim.

### GOOD: Evidence-backed R7 with enumeration

```
R7: Acceptance Criterion Coverage — PASS
**Evidence:**
- AC-1 (entity creation): covered by reviewer-conformance (review §2, verified feature.go L34-52)
- AC-2 (input validation): covered by reviewer-conformance (review §2, verified feature.go L55-71)
- AC-3 (error response format): covered by reviewer-quality (review §3, error wrapping analysis)
- AC-4 (test coverage): covered by reviewer-testing (review §4, 14 test cases verified)
- AC-5 (security boundary): covered by reviewer-security (review §5, input sanitisation check)
**Result:** PASS — 5 of 5 acceptance criteria covered by at least one reviewer.
```

**WHY GOOD:** Every AC is enumerated with the covering reviewer and the evidence location in the review document. Counts are explicit. A human can verify each claim by looking at the cited review sections. A machine can parse the coverage ratio.

### GOOD: R2 fail with rubric application

```
R2: No Rubber-Stamp Reviews — FAIL
**Rubric application:**
- Criterion 1 (findings or explicit rationale): NOT MET — review has zero findings.
  The review text is: "LGTM :shipit:" — no explicit rationale for why no findings
  exist. No reference to what was checked.
- Criterion 2 (evidence of engagement): NOT MET — no code sections cited, no test
  output, no spec traceability, no review history.
- Criterion 3 (reviewer identity and verdict): PARTIALLY MET — verdict "approved"
  is implied by ":shipit:" but reviewer identity is missing (unsigned review).
**Result:** FAIL — R2 is blocking. The reviewer-quality review is a textbook
rubber stamp per the R2 rubric negative example #1.
```

**WHY GOOD:** Each of the three R2 criteria is evaluated against specific evidence from the review document. The rubric is explicitly cited (negative example #1). The reasoning is traceable — a human can verify each criterion assessment independently. This is the pattern for all rubric-based checks.

## Evaluation Criteria

1. Does the output contain both a summary (≤300 words) and a full report with per-check analysis? **Weight: required.**
2. Does every check result (pass/fail/escalate) cite specific evidence from the review document? **Weight: required.**
3. Does R2 apply all three rubric criteria (findings/rationale, evidence of engagement, reviewer identity) with explicit evidence for each? **Weight: required.**
4. Does R3 compute the exact blocking ratio with numerator and denominator, and apply the 40% threshold correctly? **Weight: required.**
5. Does R4 cross-reference every cited REQ-ID against the parent specification, flagging unmatched references? **Weight: required.**
6. Are blocking and non-blocking checks correctly classified per REQ-RVW-003 (R1,R2,R4,R5,R7 blocking; R3,R6,R8 non-blocking)? **Weight: required.**
7. Does R7 enumerate every acceptance criterion against the reviewer(s) covering it, with explicit counts? **Weight: high.**
8. Are borderline cases escalated per the rubric escalation patterns rather than forced to pass/fail? **Weight: high.**

## Questions This Skill Answers

- How do I validate that a code review is thorough enough for auto-approval?
- What are the 8 review gate validation checks and which are blocking?
- How do I detect a rubber-stamp review?
- What is the severity inflation threshold and how is it calculated?
- How do I verify that blocking findings cite spec requirements?
- When should I escalate a borderline validation case instead of passing or failing?
- What is the difference between the summary and full report outputs?
- How do I verify that every acceptance criterion is covered by at least one reviewer?
- What does the review-gate-validator role do versus the specialist reviewers?
- How do I avoid re-reviewing the code when validating a review?
