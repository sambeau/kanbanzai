---
name: validate-spec
description:
  expert: "Structured specification validation producing a gate-checkable report
    with 10 structural checks (S1-S10), each classified as blocking or non-blocking,
    and a summary verdict (pass/pass_with_notes/fail) for the orchestrator"
  natural: "Validate a specification document against the 10 quality checks —
    tell me if it passes, has notes, or fails, and produce a full report"
triggers:
  - validate a specification
  - run spec validation
  - check a spec for quality
  - review a specification against the 10 checks
  - produce a spec validation report
roles: [spec-validator]
stage: specifying
constraint_level: low
---

## Vocabulary

- **blocking check** — a structural gate that must pass for the specification to be approved; failure prevents advancement past the validating stage
- **non-blocking check** — a quality signal that does not block advancement but must be reported; non-blocking failures produce notes in the verdict
- **evidence score** — the fraction of checks (S1-S10) whose evidence was positively identified during validation, regardless of pass/fail outcome
- **structural completeness** — the property that all required sections exist and contain content, checked by S1
- **design traceability** — the property that every requirement can be followed back to a specific design document section or decision, checked by S2 and S9
- **requirement traceability** — the property that every REQ-ID appears in the Verification Plan, checked by S4
- **acceptance criterion testability** — the property that a criterion describes an observable outcome verifiable by a tester of ordinary skill, checked by S5
- **checkbox format** — the convention that acceptance criteria appear as markdown checkboxes (`- [ ]`) for completion tracking during verification, checked by S6
- **implementation instruction** — a requirement that prescribes internal data structures, algorithms, or API signatures rather than observable behaviour, detected by S7
- **scope boundary** — the explicit demarcation of what the specification covers (in-scope) and what it does not (out-of-scope), checked by S8
- **orphaned requirement** — a requirement whose REQ-ID has no traceable parent in the design document or whose parent reference is circular or unverifiable, detected by S9
- **measurable threshold** — a numeric bound (percentage, time, count, rate) that makes a non-functional requirement verifiable, checked by S10
- **rubric** — a concrete classification guide consulted during S5 and S7 to determine pass, fail, or escalate for borderline cases
- **borderline → escalate** — a verdict pattern for S5 or S7 where the criterion cannot be clearly classified as pass or fail against the rubric; the finding is escalated to the human with context rather than being forced into a binary

## Anti-Patterns

### Rubber-Stamp Validation
- **Detect:** Declaring checks passed without enumerating the specific evidence found — e.g., reporting "S4 passes" without listing every REQ-ID against the Verification Plan
- **BECAUSE:** Hallucinated completeness means checks pass by default when evidence is not explicitly searched for — this defeats the validating stage gate entirely and allows unverifiable specifications to advance to implementation
- **Resolve:** For each check, enumerate the evidence found before declaring pass/fail. Report counts: "S4: 12 REQ-IDs found in Requirements section, 12 REQ-IDs found in Verification Plan — no orphans"

### Content Judgment
- **Detect:** Evaluating whether a requirement is *correct* rather than whether it is *well-formed* — commenting on the substance of what the requirement asks for instead of its structural quality
- **BECAUSE:** The spec-validator is a quality gate, not a design review. Judging correctness duplicates the design review and couples the validator to domain knowledge it cannot have
- **Resolve:** Restrict evaluation to the 10 structural checks. If a requirement looks questionable in substance, do not report it as a finding — the design traceability check (S2) and orphan check (S9) will catch structural gaps without evaluating merit

### Assumed Traceability
- **Detect:** Marking a requirement as traceable to the design without citing the specific design section or decision that it derives from
- **BECAUSE:** Untethered traceability claims accumulate into false confidence that every requirement has a parent — broken traceability means requirements that no design ever called for survive review undetected
- **Resolve:** For S2 and S9, cite the specific design section or parent document reference. If no parent is found, report the gap explicitly rather than assuming the relationship exists

### Tool Overconsumption
- **Detect:** Making more than 5 tool calls during a single validation session — reading every document section individually instead of reading the full spec in one call
- **BECAUSE:** The validate-spec skill operates under a 5-tool-call budget (REQ-NF-001). Exceeding this budget means the skill cannot complete its checks within the constraint and must hand off incomplete results
- **Resolve:** Batch reads: one `read_file` for the spec, one `read_file` for the design doc (if available as a path), one `doc_intel(action: "outline")` for structural navigation, one `doc(action: "register")` for the report. Use the fifth call only when essential

### Orphaned Finding
- **Detect:** A check produces a finding (pass or fail) but that finding does not appear in the report document registered to the document store
- **BECAUSE:** The orchestrator relies on the registered report for gate decisions. Findings communicated only in the handoff summary are invisible to downstream tooling and cannot be referenced in approvals or later revalidation
- **Resolve:** Every finding produced during validation must appear in both the summary (for the orchestrator) and the full report (registered to the document store). Cross-check the summary against the report before finishing

## Prerequisites

Before starting validation, confirm:

1. The specification document exists and is accessible via `read_file` or `doc(action: "content")`.
2. The parent design document path or document ID is known. If the spec does not reference a design document, validation proceeds but S2 and S9 will fail.
3. The S5/S7 rubric file is available at `work/P43-fast-track-architecture/validator-rubrics/spec-validator-rubrics.md`. Read this file as part of the validation procedure.

IF any prerequisite is missing → STOP and report the missing context. Do not proceed with partial inputs.

## Checklist

Complete this checklist during validation. Every item must have an explicit answer:

- [ ] S1: Are all five required sections present? (Problem Statement, Requirements, Constraints, Acceptance Criteria, Verification Plan)
- [ ] S2: Does the Overview or Problem Statement reference the parent design document?
- [ ] S3: Does every requirement have a unique REQ-ID?
- [ ] S4: Does every REQ-ID appear in the Verification Plan?
- [ ] S5: Is every acceptance criterion testable per the rubric? (Consult S5 rubric)
- [ ] S6: Are acceptance criteria in checkbox format (`- [ ]`)?
- [ ] S7: Is any requirement a disguised implementation instruction? (Consult S7 rubric)
- [ ] S8: Does the scope section state both in-scope AND out-of-scope?
- [ ] S9: Are there any orphaned requirements not traceable to the parent design?
- [ ] S10: Do non-functional requirements have measurable thresholds?

## Procedure

### Step 1: Read inputs

1. Read the specification document: `read_file(path: "<spec-path>")` or `doc(action: "content", id: "<spec-doc-id>")`. This is one tool call.
2. Read the parent design document. If the spec references it by path, use `read_file`. If by document ID, use `doc(action: "content", id: "<design-doc-id>")`. This is a second tool call.
3. Read the S5/S7 rubrics: `read_file(path: "work/P43-fast-track-architecture/validator-rubrics/spec-validator-rubrics.md")`. This is a third tool call.

**Total tool calls so far: 3.** Two remain for the report registration and any essential follow-up.

IF the specification document cannot be read → STOP. Report "Spec document not accessible."
IF the parent design document is referenced but cannot be read → note this but proceed; S2 and S9 will fail.
IF the rubric file cannot be read → STOP. "S5/S7 rubrics not accessible — cannot evaluate testability or implementation instruction detection."

### Step 2: Execute structural checks (S1, S3, S6, S8, S10)

Execute these checks by pattern matching against the document structure. These checks do not require the rubrics.

**S1 — All required sections present.** Scan the spec for these five section headings: Problem Statement, Requirements, Constraints, Acceptance Criteria, Verification Plan. Each must exist and contain non-empty content (more than a placeholder or "TBD"). Classify: blocking.

**S3 — Every requirement has a unique REQ-ID.** Extract every requirement identifier from the Requirements section using the pattern `REQ-` (prefix). Check for: duplicates (same ID appears twice), missing IDs (a requirement bullet without an ID), malformed IDs (does not match `REQ-\d+` or `REQ-NF-\d+` pattern). Classify: blocking.

**S6 — Acceptance criteria use checkbox format.** Scan the Acceptance Criteria section. Check whether criteria are formatted as markdown checkboxes (`- [ ]` or `- [x]`). Count how many criteria use checkbox format vs. other formats. Classify: non-blocking.

**S8 — Scope states both in-scope and out-of-scope.** Search the spec for explicit "in scope" / "out of scope" or "does NOT cover" language. Both directions must be present. A scope section that only lists what is covered is incomplete. Classify: non-blocking.

**S10 — Non-functional requirements have measurable thresholds.** For each non-functional requirement (typically REQ-NF-*), check for a numeric or time-bounded threshold: percentages, milliseconds, bytes, counts per second, percentiles, or explicit numeric bounds. Requirements using only qualitative adjectives ("fast," "robust," "scalable") without a numeric anchor fail this check. Classify: blocking.

### Step 3: Execute traceability checks (S2, S4, S9)

These checks require cross-referencing sections against each other and against the parent design document.

**S2 — Overview or Problem Statement references the parent design document.** Search the Problem Statement / Overview section for: a file path (e.g., `work/design/...`), a document ID (e.g., `DOC-...`), or an explicit title matching the parent design. The reference must be specific enough to locate the design document. A vague reference ("per the design") without a path or ID fails. Classify: blocking.

**S4 — Every REQ-ID appears in the Verification Plan.** Cross-reference: enumerate every REQ-ID found in the Requirements section (from S3). Then enumerate every REQ-ID referenced in the Verification Plan. Any REQ-ID present in Requirements but absent from the Verification Plan is an orphan. Report the count and list the orphaned IDs. Classify: blocking.

**S9 — No orphaned requirements not traceable to the parent design.** For each requirement, check whether the requirement's subject matter can be traced to a section, decision, or concern in the parent design document. This check uses document outline comparison: `doc_intel(action: "outline", id: "<design-doc-id>")` to get the design structure, then maps each requirement to the design section it derives from. Requirements with no identifiable parent design section are orphaned. Classify: non-blocking.

IF `doc_intel` is used for S9, this is the fourth tool call. One call remains.

### Step 4: Execute rubric-guided checks (S5, S7)

**These checks require the rubric file read in Step 1.** Read the S5 and S7 sections of the rubric and apply them to each acceptance criterion and requirement respectively.

**S5 — Every acceptance criterion is testable.** For each acceptance criterion, apply the S5 rubric from the rubric file. Classify each criterion as pass (observable outcome stated, conditions explicit, no undefined subjective terms), fail (subjective language without measurable anchor, no observable outcome), or borderline → escalate (implicit observability, collective testability, benchmarked but unanchored). A single fail means S5 fails. Borderline findings are reported as notes with context. Classify: blocking.

**S7 — No requirement is a disguised implementation instruction.** For each requirement and acceptance criterion, apply the S7 rubric from the rubric file. Classify each as pass (describes WHAT, not HOW), fail (prescribes internal data structures, algorithms, or API signatures), or borderline → escalate (design-authorised naming, structural necessity, data shape vs. data structure ambiguity). Fail findings are reported as notes (non-blocking). Borderline findings are escalated with context. Classify: non-blocking.

### Step 5: Produce summary (for orchestrator)

Compute the verdict from the check results:

- **pass:** All blocking checks (S1, S2, S3, S4, S5, S10) pass. Non-blocking checks may have notes.
- **pass_with_notes:** All blocking checks pass, but one or more non-blocking checks (S6, S7, S8, S9) produced findings.
- **fail:** One or more blocking checks failed.

Count findings:
- `blocking_passed` / `blocking_total` (6 total: S1-S5, S10)
- `non_blocking_with_findings` / `non_blocking_total` (4 total: S6-S9)
- `escalated` — count of borderline → escalate findings from S5/S7

Compute evidence score: the fraction of the 10 checks whose evidence was positively identified (regardless of pass/fail). A check whose evidence could not be evaluated (e.g., no non-functional requirements existed for S10) counts as N/A and is excluded from the denominator.

Present the summary as:

```
## Spec Validation Summary

| Field | Value |
|-------|-------|
| Spec | <path or doc-id> |
| Validator | spec-validator |
| Verdict | pass / pass_with_notes / fail |
| Evidence Score | X/Y checks evaluated |
| Blocking | A/6 passed |
| Non-Blocking | B/4 with findings |
| Escalated | N borderline findings |
| Full Report | DOC-xxx |
```

### Step 6: Write and register full report

Write the full report to `work/reviews/spec-validation-<feature-slug>.md` using the output format below. Use `write_file` to create the file.

Register the report: `doc(action: "register", path: "work/reviews/spec-validation-<feature-slug>.md", type: "report", title: "Spec Validation: <spec-title>")`.

**This is the fifth tool call.** IF all five calls have been consumed and the report cannot be registered → report the file path in the summary and instruct the orchestrator to register it manually.

## Output Format

The full report follows this structure:

```
# Spec Validation: <spec-document-title>

| Field | Value |
|-------|-------|
| Spec | <path or doc-id> |
| Design | <path or doc-id of parent design> |
| Validator | spec-validator |
| Date | <ISO 8601> |
| Verdict | pass / pass_with_notes / fail |
| Evidence Score | X/Y |

## Structural Checks

### S1 — Required Sections (blocking)
| Section | Present |
|---------|---------|
| Problem Statement | ✅ / ❌ |
| Requirements | ✅ / ❌ |
| Constraints | ✅ / ❌ |
| Acceptance Criteria | ✅ / ❌ |
| Verification Plan | ✅ / ❌ |

**Verdict:** ✅ Pass / ❌ Fail
**Evidence:** <what was found>

### S3 — Unique REQ-IDs (blocking)
| Check | Result |
|-------|--------|
| Total REQ-IDs | N |
| Duplicates | N (list if any) |
| Missing IDs | N (list if any) |
| Malformed IDs | N (list if any) |

**Verdict:** ✅ Pass / ❌ Fail
**Evidence:** <enumeration of IDs found>

### S6 — Checkbox Format (non-blocking)
| Format | Count |
|--------|-------|
| Checkbox (`- [ ]`) | N |
| Other format | N |

**Verdict:** ✅ Pass / ⚠️ Findings
**Evidence:** <description>

### S8 — Scope Boundary (non-blocking)
| Direction | Present |
|-----------|---------|
| In-scope | ✅ / ❌ |
| Out-of-scope | ✅ / ❌ |

**Verdict:** ✅ Pass / ⚠️ Findings
**Evidence:** <quoted scope statements>

### S10 — Measurable Thresholds (blocking)
| REQ-ID | Threshold | Measurable? |
|--------|-----------|-------------|
| REQ-NF-001 | <threshold text> | ✅ / ❌ |

**Verdict:** ✅ Pass / ❌ Fail / N/A (no NF requirements)
**Evidence:** <enumeration of NF requirements and their thresholds>

## Traceability Checks

### S2 — Design Reference (blocking)
**Evidence:** <quoted reference from Problem Statement or absence thereof>
**Verdict:** ✅ Pass / ❌ Fail

### S4 — Verification Plan Coverage (blocking)
| REQ-ID | In Verification Plan? |
|--------|----------------------|
| REQ-001 | ✅ / ❌ |
| ... | ... |

**Orphaned REQ-IDs:** <list or "none">
**Verdict:** ✅ Pass / ❌ Fail
**Evidence:** <count of REQ-IDs in Requirements vs. Verification Plan>

### S9 — Design Traceability (non-blocking)
| REQ-ID | Design Section | Traceable? |
|--------|---------------|------------|
| REQ-001 | <section-path> | ✅ / ❌ |
| ... | ... | ... |

**Orphaned requirements:** <list or "none">
**Verdict:** ✅ Pass / ⚠️ Findings
**Evidence:** <design outline sections used for mapping>

## Rubric-Guided Checks

### S5 — Testable Acceptance Criteria (blocking)
| AC | Observable? | Conditions Explicit? | Subjective Terms? | Verdict |
|----|------------|---------------------|-------------------|---------|
| AC-001 | ✅ | ✅ | None | ✅ Pass |
| ... | ... | ... | ... | ... |

**Borderline → Escalate:** <list with context, or "none">
**Verdict:** ✅ Pass / ❌ Fail
**Evidence:** <application of S5 rubric to each AC>

### S7 — Implementation Instruction Detection (non-blocking)
| REQ/AC | Internal Structures? | Algorithms? | API Design? | Verdict |
|--------|--------------------|-------------|-------------|---------|
| REQ-001 | ❌ | ❌ | ❌ | ✅ Pass |
| ... | ... | ... | ... | ... |

**Borderline → Escalate:** <list with context, or "none">
**Verdict:** ✅ Pass / ⚠️ Findings
**Evidence:** <application of S7 rubric to each requirement>

## Finding Summary

| Check | Classification | Verdict | Detail |
|-------|---------------|---------|--------|
| S1 | blocking | ✅ / ❌ | |
| S2 | blocking | ✅ / ❌ | |
| S3 | blocking | ✅ / ❌ | |
| S4 | blocking | ✅ / ❌ | |
| S5 | blocking | ✅ / ❌ | |
| S6 | non-blocking | ✅ / ⚠️ | |
| S7 | non-blocking | ✅ / ⚠️ | |
| S8 | non-blocking | ✅ / ⚠️ | |
| S9 | non-blocking | ✅ / ⚠️ | |
| S10 | blocking | ✅ / ❌ / N/A | |

## Verdict

<Final assessment. For fail: list which blocking checks failed and what must be fixed. For pass_with_notes: list non-blocking findings. For pass: confirm all gates clear.>
```

## Examples

### BAD: Vague validation without evidence enumeration

> S4 passes. All requirements are covered in the verification plan.

**WHY BAD:** Declares S4 passed without listing REQ-IDs or cross-referencing them against the Verification Plan. This is rubber-stamp validation — the validator cannot know whether coverage is complete without enumerating both sides. A reviewer reading this report cannot verify the claim.

### BAD: Content judgment instead of structural check

> S5 fails. The requirement that "merge must fail when no report exists" is a bad idea — the system should be more lenient and allow overrides in this case.

**WHY BAD:** The validator is evaluating whether the requirement is *correct*, not whether it is *well-formed*. The requirement describes an observable outcome with explicit conditions (feature in reviewing, no reports, merge called → rejected) — it is testable and passes S5. The validator's opinion about override behaviour is a design concern, not a spec quality concern.

### GOOD: Structured validation with enumerated evidence

```
### S4 — Verification Plan Coverage (blocking)

| REQ-ID | In Verification Plan? |
|--------|----------------------|
| REQ-001 | ✅ (Test: automated gate evaluation) |
| REQ-002 | ✅ (Test: automated message content check) |
| REQ-003 | ✅ (Test: automated document query) |
| REQ-004 | ✅ (Test: automated status transition) |

**Orphaned REQ-IDs:** none
**Verdict:** ✅ Pass
**Evidence:** 4 REQ-IDs found in Requirements section (REQ-001 through REQ-004).
All 4 appear in the Verification Plan with specific verification methods.
```

**WHY GOOD:** Every REQ-ID is enumerated. The cross-reference between Requirements and Verification Plan is explicit. A reviewer can independently verify the claim by reading the spec. The evidence supports the verdict without requiring trust in the validator.

### GOOD: S5 borderline escalation with context

```
**Borderline → Escalate:**
- AC-004: "The feature flag state is updated." This describes an internal state change
  without stating how to observe it. A tester would need to know whether this means a
  config file change, a database row, or an API response. **Recommendation:** Specify
  the observable signal — e.g., "the feature flag state is updated AND the change is
  visible in the next `status()` response."
```

**WHY GOOD:** The borderline finding is reported with the specific AC, the reason it cannot be classified, and a concrete recommendation. The validator does not force a binary pass/fail on an ambiguous criterion. The human reviewer has enough context to resolve the escalation without re-reading the spec.

## Evaluation Criteria

1. Does the report enumerate evidence for every check rather than declaring results? Weight: required.
2. Are all 10 checks (S1-S10) present in the report with correct blocking/non-blocking classification? Weight: required.
3. Are S5 and S7 evaluated against the rubric file, with borderline cases escalated rather than forced? Weight: required.
4. Does the summary include verdict, blocking/non-blocking counts, evidence score, and the full report document ID? Weight: high.
5. Is the full report registered to the document store via `doc(action: "register")`? Weight: high.
6. Does the validation stay within the 5-tool-call budget? Weight: high.
7. Are all findings free of content judgment (correctness evaluation)? Weight: medium.
8. Are orphaned requirements (S4, S9) enumerated by ID rather than reported as a count alone? Weight: medium.

## Questions This Skill Answers

- How do I validate a specification document in Kanbanzai?
- What are the 10 structural checks for spec quality?
- Which spec validation checks are blocking vs. non-blocking?
- How do I determine if an acceptance criterion is testable?
- How do I detect implementation instructions disguised as requirements?
- What is the S5 rubric for testability classification?
- What is the S7 rubric for implementation instruction detection?
- How do I produce a spec validation summary for the orchestrator?
- How do I register a spec validation report to the document store?
- What should I do when a criterion is borderline between pass and fail?
