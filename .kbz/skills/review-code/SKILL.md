---
name: review-code
description:
  expert: "Multi-dimension code review producing classified findings
    with evidence-backed verdicts against acceptance criteria"
  natural: "Review code changes against a spec and produce a structured
    report of what's right and what needs fixing"
triggers:
  - review code changes
  - evaluate implementation against spec
  - check code quality
  - produce review findings for feature
roles: [reviewer, reviewer-conformance, reviewer-quality,
        reviewer-security, reviewer-testing]
stage: reviewing
constraint_level: medium
---

## Vocabulary

- **finding** — a single observation about the code under review, classified as
  blocking or non-blocking. Never called an "issue," "problem," or "comment" within
  this skill.
- **finding classification** — the severity bucket for a finding: **blocking**
  (violates a spec requirement, must be fixed before approval) or **non-blocking**
  (improvement suggestion, style note, or observation that does not prevent approval).
- **evidence-backed verdict** — a verdict supported by specific code locations and
  spec citations, not by qualitative impressions.
- **acceptance criteria traceability** — the practice of linking each blocking finding
  (and each "pass" claim) to a numbered acceptance criterion in the specification.
- **per-dimension outcome** — the result for a single evaluation dimension:
  `pass` (fully meets requirements), `pass_with_notes` (meets requirements with
  non-blocking observations), `concern` (potential gap that needs human judgement),
  `fail` (violates a spec requirement).
- **review unit** — a cohesive group of files evaluated together in a single review
  pass. Defined by the orchestrator, not by this skill.
- **dimension** — an independent evaluation axis (e.g., spec_conformance,
  implementation_quality, test_adequacy, security). Each dimension is assessed in
  isolation. Which dimensions apply depends on the reviewer role.
- **structured review output** — the machine-parseable format this skill produces.
  Every field has exactly one interpretation.
- **remediation recommendation** — a concrete suggestion attached to a non-blocking
  finding. Blocking findings cite the violated requirement; the fix is implied.
- **spec conformance gap** — a mismatch between what the spec requires and what the
  code implements. Always blocking.
- **review profile** — the set of dimensions required for this review, determined by
  the reviewer role assigned to this agent.
- **aggregate verdict** — the overall review outcome, derived from per-dimension
  outcomes. Set by the orchestrator after collating all reviewers, not by this skill.
- **clearance** — an approved verdict backed by per-dimension evidence. Distinguished
  from a rubber-stamp by the presence of substantive evidence.
- **spec anchor** — a reference to a specific numbered acceptance criterion or spec
  section that grounds a finding or a pass claim.
- **validation loop** — the iterative check at the end of the procedure where findings
  are verified against classification criteria before output is finalised.

## Anti-Patterns

### Rubber-Stamp Review (MAST FM-3.1)

- **Detect:** Verdict is "approved" with zero findings AND no per-dimension evidence
  citations.
- **BECAUSE:** LLM sycophancy makes approval the path of least resistance. FM-3.1 is
  the #1 quality failure mode in multi-agent systems (MAST, 2024). A review that
  approves without evidence is indistinguishable from no review at all.
- **Resolve:** Require at least one finding OR substantive per-dimension evidence for
  every clearance. If the code genuinely has no findings, demonstrate that by citing
  specific code locations and spec criteria checked.

### Severity Inflation

- **Detect:** More than 40% of findings classified as blocking.
- **BECAUSE:** Over-classifying non-blocking findings as blocking dilutes the signal
  from genuine spec violations and creates unnecessary remediation cycles. A blocking
  finding requires a specific violated acceptance criterion — if you cannot cite one,
  the finding is non-blocking.
- **Resolve:** Re-check each blocking finding against the classification criteria.
  Blocking requires a specific violated requirement. Reclassify any finding that lacks
  a spec anchor to non-blocking.

### Dimension Bleed

- **Detect:** A finding in one dimension influences the verdict of another dimension
  (e.g., poor test coverage lowering the implementation_quality score).
- **BECAUSE:** Dimensions are independent evaluation axes. A poor test_adequacy result
  does not make the implementation incorrect — it means the tests are insufficient.
  Bleeding dimensions produces unreliable verdicts because a single weakness gets
  counted multiple times.
- **Resolve:** Evaluate each dimension in complete isolation. Cross-reference findings
  only in the aggregate verdict (which the orchestrator produces, not this skill).

### Prose Commentary

- **Detect:** Output contains qualitative prose ("well-structured," "clean code,"
  "looks good") instead of structured findings with locations and evidence.
- **BECAUSE:** Prose is ambiguous and cannot be machine-parsed for remediation routing.
  "Well-structured" means different things to different readers. Structured output has
  exactly one interpretation — dimension, location, evidence, classification.
- **Resolve:** Replace every qualitative statement with a finding entry containing
  dimension, location, and evidence. If a positive observation is worth recording,
  record it as per-dimension evidence with a spec anchor, not as prose.

### Missing Spec Anchor

- **Detect:** A blocking finding does not cite a specific spec requirement or
  acceptance criterion.
- **BECAUSE:** Without a spec anchor, the finding is an opinion, not a conformance gap.
  Opinions cannot be objectively verified or remediated — the implementer has no way to
  know what "correct" looks like. Spec-anchored findings have a clear resolution path.
- **Resolve:** Link every blocking finding to a numbered acceptance criterion or spec
  section. If no spec requirement covers the observation, reclassify as non-blocking
  with a remediation recommendation.

## Checklist

Copy this checklist and track your progress:

- [ ] Read spec section(s) fully — understand what was required
- [ ] Read all files in the file list — understand what was implemented
- [ ] Confirm review profile and identify required dimensions
- [ ] Evaluate each required dimension independently
- [ ] Classify all findings as blocking or non-blocking
- [ ] Verify every blocking finding has a spec anchor
- [ ] Check for dimension bleed — no cross-contamination between verdicts
- [ ] Produce structured output in the required format

## Procedure

### Step 1: Orient

1. Read the spec section(s) fully. Identify numbered acceptance criteria.
2. Read all files in the file list. Understand what was implemented.
3. Note the review profile — this determines which dimensions you evaluate.
4. IF any input is missing (no spec, no file list, no review profile) →
   **STOP.** Report a Missing Context finding. Do not proceed without inputs.
5. IF the spec is ambiguous or incomplete for any dimension you must evaluate →
   **STOP.** Report the ambiguity as a finding. Do not infer intent. The cost of
   asking is low; the cost of guessing wrong is high.

### Step 2: Evaluate each dimension independently

For each dimension in your review profile:

1. Work through the dimension's evaluation questions (provided by your role).
2. For each acceptance criterion relevant to this dimension, check whether the
   implementation satisfies it. Record the specific code location as evidence.
3. Record a per-dimension outcome: pass, pass_with_notes, concern, or fail.
4. Record any findings with their classification (blocking or non-blocking).
5. IF you notice a finding that belongs to a different dimension — record it
   under that dimension, not the current one. Do not let it influence the
   current dimension's verdict.

### Step 3: Validate and iterate

Before producing final output, validate your findings:

1. Does every blocking finding cite a specific spec requirement?
   IF NOT → either add the spec anchor or reclassify as non-blocking.
2. Is any dimension's verdict influenced by another dimension's result?
   IF YES → re-evaluate the affected dimension in isolation.
3. Are more than 40% of findings classified as blocking?
   IF YES → re-check each blocking classification against the criteria.
4. Does every "pass" verdict have at least one evidence citation?
   IF NOT → add evidence or reconsider the verdict.

IF any validation check fails → fix the problem → re-validate from step 1.
Repeat until all checks pass. Only then produce the structured output.

## Edge Cases

### Missing Spec

The review unit has no linked spec or the spec reference resolves to nothing.

1. Set the `spec_conformance` per-dimension outcome to `not_applicable`.
2. Record a non-blocking finding under `spec_conformance` stating that
   conformance cannot be assessed because no spec is linked.
3. Continue evaluating all other dimensions normally — implementation quality,
   test adequacy, documentation currency, and workflow integrity do not require
   a spec to assess.
4. In the overall verdict rationale, note that spec conformance was skipped due
   to missing spec. The verdict is derived from the remaining dimensions only.

### Partial Implementation

The implementation contains stubs, TODOs, placeholder returns, or missing code
paths that indicate incomplete work.

1. Record a blocking finding under `implementation_quality` describing exactly
   what is missing — cite the stub location and what the code path should do.
2. Set `spec_conformance` to `concern` for any acceptance criteria that depend
   on the incomplete code, and evaluate remaining criteria normally.
3. If the incomplete code lacks corresponding tests, record a blocking finding
   under `test_adequacy` for each untested stub or placeholder path.
4. Set the overall verdict to `needs_remediation`.
5. Do not infer what the finished implementation would look like. Evaluate only
   what exists. Speculation about intent is not evidence.

### Ambiguous Conformance

The implementation differs from the spec, but the difference may be intentional
(e.g., an improved algorithm, a reordered sequence that preserves semantics).

1. Record the deviation as a finding under `spec_conformance` with per-dimension
   outcome `concern` — not `fail`. Reserve `fail` for clear contradictions.
2. Describe precisely what the spec says (cite the spec anchor) versus what the
   implementation does (cite the code location). Do not editorialize.
3. Classify the finding:
   - **Non-blocking** if the implementation appears intentionally better or
     equivalent and does not omit or contradict any stated requirement.
   - **Blocking** if the deviation omits a requirement or contradicts a spec
     constraint, even if the alternative seems reasonable.
4. If the deviation improves on the spec without omitting requirements, attach a
   note recommending that the spec be updated to reflect the implementation.
5. Name the ambiguity explicitly (e.g., "AC-3 requires X; implementation does Y
   instead") so the orchestrator or human reviewer can resolve it with full
   context.

### Missing Context

Required inputs are absent — files are inaccessible, the context packet is
incomplete, or the review profile references artifacts that cannot be found.

1. Record specifically what is missing: which files, which spec sections, or
   which context packet fields are absent.
2. Set the overall verdict to `blocked`.
3. For each affected dimension, record a blocking finding listing the missing
   input that prevents evaluation.
4. Set each affected dimension's per-dimension outcome to `not_applicable` with
   a note identifying the missing dependency.
5. Do not produce a partial review presented as complete. A blocked verdict with
   an explicit list of missing inputs is more useful than a review that silently
   omits dimensions.

## Output Format

```
Review Unit: <unit-id>
Files: <file list>
Spec: <spec reference and section(s)>
Reviewer Role: <role name>

Overall: approved | approved_with_followups | needs_remediation | rejected

Dimensions:
  <dimension_name>: <pass | pass_with_notes | concern | fail>
    Evidence:
      - <spec anchor>: <what was checked, code location>
      - <spec anchor>: <what was checked, code location>
    Findings:
      - [blocking] <description> (spec: <anchor>, location: <file:lines>)
      - [non-blocking] <description> (location: <file:lines>)
        Recommendation: <remediation suggestion>

  <dimension_name>: <pass | pass_with_notes | concern | fail>
    Evidence:
      - ...
    Findings:
      - ...

Finding Summary:
  Blocking: <count>
  Non-blocking: <count>
  Total: <count>
```

**Overall verdict rules:**
- `approved` — all dimensions pass, zero findings.
- `approved_with_followups` — all dimensions pass or pass_with_notes, zero blocking
  findings, one or more non-blocking findings.
- `needs_remediation` — one or more blocking findings exist.
- `rejected` — a dimension has outcome `fail` with multiple blocking findings
  indicating fundamental spec misalignment.

## Examples

### BAD: Rubber-stamp with prose

```
Review Unit: service-layer
Overall: approved
Notes: Code is well-structured and follows Go conventions. Good use
of error handling. Tests look comprehensive.
```

**WHY BAD:** No findings. No evidence citations. No per-dimension verdicts. Qualitative
prose ("well-structured," "comprehensive") with no structured data. A human or machine
cannot determine what was actually checked. This is FM-3.1 — indistinguishable from not
reviewing at all.

### GOOD: Evidence-backed structured review

```
Review Unit: service-layer
Files: internal/service/feature.go, internal/service/feature_test.go
Spec: work/spec/feature-lifecycle.md §3 (AC-1 through AC-5)
Reviewer Role: reviewer-conformance

Overall: approved_with_followups

Dimensions:
  spec_conformance: pass
    Evidence:
      - AC-1: entity creation verified (feature.go L34-52, NewFeature constructor)
      - AC-2: input validation verified (feature.go L55-71, Validate method)
      - AC-3: error response format verified (feature.go L73-89, error wrapping)
  implementation_quality: pass_with_notes
    Evidence:
      - Error handling present on all exported functions
      - Interface accepted at consumer (feature.go L8), struct returned
    Findings:
      - [non-blocking] Error wrapping in CreateFeature (feature.go L48) uses
        fmt.Errorf without %w — loses error chain for callers using errors.Is
        Recommendation: Use fmt.Errorf("create feature: %w", err) to preserve
        the error chain
  test_adequacy: pass
    Evidence:
      - 14 test cases covering happy path, validation failures, and duplicate
        detection (feature_test.go L12-189)
      - Table-driven pattern used throughout

Finding Summary:
  Blocking: 0
  Non-blocking: 1
  Total: 1
```

**WHY GOOD:** Per-dimension verdicts with specific evidence. The single finding has a
location, explanation, and remediation recommendation. Spec requirements cited by number.
A machine can parse this; a human can verify each claim.

### GOOD: Evidence-backed clearance with zero findings

```
Review Unit: storage-layer
Files: internal/store/yaml.go, internal/store/yaml_test.go
Spec: work/spec/entity-storage.md §2 (AC-4 through AC-6)
Reviewer Role: reviewer-conformance

Overall: approved

Dimensions:
  spec_conformance: pass
    Evidence:
      - AC-4: YAML serialisation verified (yaml.go L12-34, Marshal method)
      - AC-5: canonical field order verified (yaml.go L36-58, fieldOrder slice)
      - AC-6: round-trip determinism verified (yaml_test.go L102,
        TestStore_RoundTrip confirms identical output)
  implementation_quality: pass
    Evidence:
      - Error wrapping with %w throughout (yaml.go L22, L41, L67)
      - No exported functions without doc comments
      - Interface accepted at consumer (yaml.go L8), struct returned
  test_adequacy: pass
    Evidence:
      - 22 test cases including round-trip serialisation (yaml_test.go L15-198)
      - Error path coverage via TestStore_CreateConflict (yaml_test.go L145)
      - Table-driven pattern for serialisation variants

Finding Summary:
  Blocking: 0
  Non-blocking: 0
  Total: 0
```

**WHY GOOD:** Zero findings but substantive evidence for every dimension. The reviewer
demonstrably examined the code — each pass verdict is backed by specific locations and
spec anchors. This is a legitimate clearance, not a rubber stamp. Best example placed
last because it demonstrates the hardest case: saying "approved" credibly.

## Evaluation Criteria

These criteria are for evaluating the review output, not for self-evaluation during the
review. They are phrased as gradable questions to support automated LLM-as-judge
evaluation.

1. Does every dimension have an explicit outcome (pass / pass_with_notes / concern /
   fail)? **Weight: required.**
2. Does every blocking finding cite a specific spec requirement?
   **Weight: required.**
3. Does the output distinguish blocking from non-blocking findings?
   **Weight: required.**
4. Can a machine extract all findings from the output without ambiguity?
   **Weight: high.**
5. Are dimensions evaluated independently — no bleed between verdicts?
   **Weight: high.**
6. Is every "approved" or "pass" verdict backed by at least one evidence citation?
   **Weight: high.**
7. Does the output avoid qualitative prose in place of structured data?
   **Weight: medium.**

## Questions This Skill Answers

- How do I review code changes against a specification?
- What dimensions should I evaluate during code review?
- How do I classify a finding as blocking vs non-blocking?
- What format should my review output use?
- When should I stop and report missing context during review?
- What does a well-evidenced "approved" verdict look like?
- How do I handle ambiguous or incomplete spec sections during review?
- What is the difference between a concern and a fail?
- How do I avoid rubber-stamping a review?