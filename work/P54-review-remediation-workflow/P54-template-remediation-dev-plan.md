# Template: Remediation Dev-Plan

| Field    | Value                                                     |
|----------|-----------------------------------------------------------|
| Plan     | P54-review-remediation-workflow                           |
| Type     | template                                                  |
| Status   | Draft                                                     |
| Author   | [orchestrator name]                                       |
| Date     | [date]                                                    |

---

## Overview

This template defines the required structure for a remediation dev-plan — the artifact that bridges a failed review and executable remediation work. Populate each section from the review report findings.

**Source review report:** `[REVIEW-DOC-ID]` — [link to review report]

**Remediation owner:** `[FEAT-xxx]` | `[BATCH-xxx]` | `[PLAN-xxx]`

**Ownership model:** Single-feature | Batch-level | Cross-cutting (see [Ownership Model Guide](P54-guide-ownership-model.md))

---

## 1. Scope

<!-- INLINE GUIDANCE: Cite the original review report ID and verdict. List every affected entity. Reference the spec or design documents whose requirements were violated. Include only what is needed to resolve the blocking findings. -->

**Original review:** `[REVIEW-DOC-ID]` — verdict: `fail`

**Affected entities:**

| Entity ID | Type | Description |
|-----------|------|-------------|
| `[FEAT-xxx]` | feature | [what this feature does, why it's affected] |
| ... | | |

**Spec documents referenced:**

| Document ID | Section | Requirement Violated |
|-------------|---------|---------------------|
| `[SPEC-DOC-ID]` | §[N] | [FR-xxx]: [one-line description] |
| ... | | |

**Out of scope for this remediation:**

- [things explicitly not being fixed — cite deferral decisions with rationale]

---

## 2. Task Breakdown

<!-- INLINE GUIDANCE: One task or task group per blocking finding. Each task must name the finding(s) it addresses. Group findings with shared root causes. Task descriptions should be concrete enough for an implementer to act on without re-reading the full review report. -->

| Task | Finding(s) Addressed | Affected Entity | Description |
|------|---------------------|-----------------|-------------|
| T1 | BF-1, BF-3 | `[FEAT-xxx]` | [concrete implementation description] |
| T2 | BF-2 | `[FEAT-xxx]` | [concrete implementation description] |
| ... | | | |

**Deferred findings (not to be fixed in this remediation):**

| Finding ID | Rationale | Owner |
|-----------|-----------|-------|
| BF-N | [why this finding is deferred, not ignored] | [who owns the deferral decision] |

---

## 3. Dependency Graph

<!-- INLINE GUIDANCE: Order tasks so that prerequisites complete before dependents. Fixes that must land before tests or re-review should be sequenced first. Tasks with no dependencies can be parallelized. -->

```
[Task with no dependencies]
├── [Task that depends on above]
│   └── [Task that depends on above]
└── [Task that depends on above, independent from sibling]

Parallel groups: [[T1, T2], [T3, T4]]
Critical path: T1 → T3 → T5
```

**Task dependency table:**

| Task | Depends On | Can Parallelize With |
|------|-----------|---------------------|
| T1 | — | T2 |
| T2 | — | T1 |
| T3 | T1 | T4 |
| T4 | T2 | T3 |
| ... | | |

---

## 4. Risk Assessment

<!-- INLINE GUIDANCE: Identify risks specific to this remediation. Common categories: dirty working trees (uncommitted changes), entity lifecycle state drift (feature in unexpected status), scope ambiguity (finding wording is unclear), dependency unavailability (P53 not shipped yet), and cross-feature conflict (two remediation tasks touching the same files). -->

| Risk | Probability | Impact | Mitigation | Affected Tasks |
|------|------------|--------|------------|---------------|
| [risk description] | High/Med/Low | High/Med/Low | [what to do about it] | [task IDs] |
| ... | | | | |

**Dirty state inventory** (from Phase 2 scope inspection):

| Entity | Worktree Status | Uncommitted Changes |
|--------|----------------|-------------------|
| `[FEAT-xxx]` | active/stale | yes/no — [details] |
| ... | | |

---

## 5. Verification Approach

<!-- INLINE GUIDANCE: For each finding, specify how verification will prove it is resolved. Verification evidence can be: test results, manual inspection notes, documentation diff, or re-review checklist items. Every finding must have at least one verification method. -->

| Finding ID | Verification Method | Evidence Required | Verifying Task |
|-----------|-------------------|-------------------|---------------|
| BF-1 | [test / inspection / doc review] | [what constitutes proof] | T1 |
| BF-2 | [test / inspection / doc review] | [what constitutes proof] | T2 |
| ... | | | |

**Re-review checklist** (to be completed before the re-review report):

- [ ] All remediation tasks are terminal (`done` or `not-planned`)
- [ ] Verification evidence collected for every finding
- [ ] Original review content hash unchanged
- [ ] Affected entities in correct lifecycle state
- [ ] Audit trail traversable in both directions

---

## 6. Traceability Matrix

<!-- INLINE GUIDANCE: This is the accountability section. Every blocking finding from the original review MUST appear here — either mapped to a task or explicitly deferred with rationale. No silent omissions. This matrix is the single source of truth for "did we address everything?" -->

| Finding ID | Finding Summary | Remediation Task(s) | Verification Method | Status |
|-----------|----------------|--------------------|--------------------|--------|
| BF-1 | [one-line from review] | T1 | [method from §5] | [pending/in-progress/resolved/deferred] |
| BF-2 | [one-line from review] | T2 | [method from §5] | [pending/in-progress/resolved/deferred] |
| BF-3 | [one-line from review] | T1, T3 | [method from §5] | [pending/in-progress/resolved/deferred] |
| ... | | | | |

**Coverage check:**
- Total blocking findings in original review: **[N]**
- Findings mapped to tasks: **[N]**
- Findings deferred: **[N]**
- Unaccounted findings: **0** (must be zero — [NFR-01])

---

## See Also

- [Review Remediation Procedure](P54-report-review-remediation-procedure.md)
- [P54 Specification: Review Remediation Workflow](P54-spec-review-remediation-workflow.md)
- [P54 Design: Review Remediation Workflow](P54-design-review-remediation-workflow.md)
- [Ownership Model Guide](P54-guide-ownership-model.md)
- [Re-Review Report Template](P54-template-re-review-report.md)
