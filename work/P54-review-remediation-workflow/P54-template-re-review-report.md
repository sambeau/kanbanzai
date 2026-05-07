# Template: Re-Review Report

| Field    | Value                                                     |
|----------|-----------------------------------------------------------|
| Plan     | P54-review-remediation-workflow                           |
| Type     | report                                                    |
| Subtype  | review_remediation                                        |
| Status   | Draft                                                     |
| Author   | [orchestrator name]                                       |
| Date     | [date]                                                    |

---

## Overview

This re-review report records the resolution of blocking findings from a prior review. It reuses the `report` document type with `review_remediation` subtype. The original review report remains immutable — this document records resolution, not replacement.

**Original review report:** `[ORIGINAL-REVIEW-DOC-ID]`

**Original review verdict:** `fail`

**Remediation dev-plan:** `[REMEDIATION-DEV-PLAN-DOC-ID]`

**Re-review verdict:** `[resolved / partially-resolved / not-resolved]`

---

## Original Review Citation

| Field | Value |
|-------|-------|
| Document ID | `[ORIGINAL-REVIEW-DOC-ID]` |
| Title | [title of original review] |
| Date | [date of original review] |
| Verdict | `fail` |
| Content hash (at time of remediation) | `[sha256]` |
| Status | `approved` — unchanged |

> **Immutability check:** The original review report content hash has not changed since remediation began (FR-06). Resolution is recorded here, not by editing the original.

---

## Finding Resolution Status

<!-- INLINE GUIDANCE: List every blocking finding from the original review. Each finding must have a resolution status: resolved (all closure criteria met per FR-05), deferred (explicit decision not to fix, with rationale and owner), or unresolved (still in progress — re-review is premature). -->

| Finding ID | Finding Summary | Resolution Status | Remediation Task(s) | Verification Evidence |
|-----------|----------------|-------------------|--------------------|--------------------|
| BF-1 | [one-line from original review] | resolved | T1, T3 | [link to evidence] |
| BF-2 | [one-line from original review] | resolved | T2 | [link to evidence] |
| BF-3 | [one-line from original review] | deferred | — | [rationale + owner] |
| ... | | | | |

**Resolution summary:**

| Status | Count |
|--------|-------|
| Resolved | [N] |
| Deferred | [N] |
| Unresolved | [N] |
| **Total** | **[N]** (must equal total blocking findings in original review) |

---

## Verification Evidence

<!-- INLINE GUIDANCE: For each resolved finding, provide concrete evidence that the issue is fixed. Evidence types: test output (pasted or linked), manual inspection notes, documentation diffs, commit SHAs. -->

### BF-1: [finding summary]

**Resolution:** [what was done to fix it]

**Evidence:**
```
[test output, inspection notes, commit SHA, or document reference]
```

**Verification date:** [date]

**Verified by:** [orchestrator name]

---

### BF-2: [finding summary]

**Resolution:** [what was done to fix it]

**Evidence:**
```
[test output, inspection notes, commit SHA, or document reference]
```

**Verification date:** [date]

**Verified by:** [orchestrator name]

---

<!-- Repeat for each resolved finding -->

---

## Deferred Findings

<!-- INLINE GUIDANCE: Every deferred finding must have an explicit rationale and an owner. A finding is deferred when the decision is made not to fix it — not when it was forgotten. -->

### DF-1: BF-3 — [finding summary]

**Rationale for deferral:** [why this finding is not being fixed now]

**Deferral owner:** [who approved the deferral]

**Revisit criteria:** [under what conditions this should be reconsidered]

---

## Aggregate Resolution Verdict

<!-- INLINE GUIDANCE: The re-review verdict follows the same logic as any review verdict. -->

| Criterion | Status |
|-----------|--------|
| All blocking findings accounted for (resolved or deferred) | ✅ / ❌ |
| Every resolved finding has verification evidence (FR-05) | ✅ / ❌ |
| Every deferred finding has rationale and owner (FR-03) | ✅ / ❌ |
| Original review report unchanged (FR-06) | ✅ / ❌ |
| Audit trail traversable in both directions (NFR-02) | ✅ / ❌ |

**Final verdict:** `[resolved]` | `[partially-resolved]` | `[not-resolved]`

**If `partially-resolved` or `not-resolved`:** [explain what remains and next steps]

---

## Audit Trail

<!-- INLINE GUIDANCE: Verify the full chain is traversable in both directions. This confirms NFR-02 compliance. -->

**Forward trace** (review → remediation → resolution):
```
Original Review ([REVIEW-DOC-ID])
  → BF-1: [summary]
    → Remediation Dev-Plan ([DEV-PLAN-DOC-ID]) §Traceability Matrix
      → Task T1 ([TASK-xxx]): [description]
        → Verification: [evidence reference]
          → Re-Review Report (this document): resolved
```

**Backward trace** (resolution → remediation → review):
```
Re-Review Report (this document): BF-1 resolved
  → Verification: [evidence reference]
    → Task T1 ([TASK-xxx])
      → Remediation Dev-Plan ([DEV-PLAN-DOC-ID]) §Traceability Matrix
        → Original Review ([REVIEW-DOC-ID]) §Finding BF-1
```

---

## See Also

- [Review Remediation Procedure](P54-report-review-remediation-procedure.md)
- [P54 Specification: Review Remediation Workflow](P54-spec-review-remediation-workflow.md)
- [Remediation Dev-Plan Template](P54-template-remediation-dev-plan.md)
- [Ownership Model Guide](P54-guide-ownership-model.md)
