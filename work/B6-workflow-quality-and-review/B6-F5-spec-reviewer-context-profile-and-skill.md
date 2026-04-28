# Reviewer Context Profile and SKILL Specification

| Document | Reviewer Context Profile and SKILL Specification |
|----------|--------------------------------------------------|
| Status   | Draft                                            |
| Created  | 2026-03-28T00:26:51Z                             |
| Updated  | 2026-03-28T00:26:51Z                             |
| Feature  | `FEAT-01KMRX1HG8BAX`                             |
| Related  | `work/design/code-review-workflow.md` §7, §8, §9, §14.2–14.4 |

---

## 1. Purpose

This specification defines the deliverables for Feature E of the P6 Workflow Quality plan: the reviewer context profile and the code review SKILL document. Together these two artefacts operationalise the review dimensions and profiles defined in `work/design/quality-gates-and-review-policy.md`, making them available to review sub-agents through the standard context assembly mechanism rather than through repeated manual prompting.

The reviewer context profile (`.kbz/context/roles/reviewer.yaml`) provides review conventions and dimension guidance to any agent assembled with `role="reviewer"`. The code review SKILL (`.skills/code-review.md`) is a procedural document that instructs a review agent how to conduct a single review unit end-to-end: what input to expect, how to evaluate each dimension, what output structure to produce, and how to handle edge cases.

A third deliverable updates the quality gates policy document to cross-reference this design and note that the policy is now operationalised through these two artefacts.

---

## 2. Goals

1. Review sub-agents receive structured review conventions and dimension definitions automatically through `context_assemble(role="reviewer")`, without manual prompting.
2. Any agent that follows the code review SKILL produces consistently structured output: per-dimension outcomes and an overall verdict with blocking and non-blocking findings separated.
3. Review expectations are no longer duplicated in chat prompts, feature instructions, or AGENTS.md — they live in one canonical place (the SKILL and the quality gates policy) that both humans and agents can reference.
4. The quality gates policy document explicitly acknowledges that it is now operationalised through the reviewer profile and the review SKILL, with cross-references to both.

---

## 3. Scope

### 3.1 In scope

- Creating `.kbz/context/roles/reviewer.yaml` — a context profile that inherits from `base` and encodes review approach conventions, output format conventions, and dimension definitions.
- Creating `.skills/code-review.md` — a SKILL document covering: input requirements, per-dimension evaluation guidance, structured output format, finding classification (blocking vs non-blocking), and edge case handling.
- Updating `work/design/quality-gates-and-review-policy.md` — adding a cross-reference section that names the reviewer profile and review SKILL as the operational form of the policy, and linking both artefacts.

### 3.2 Deferred

- Orchestration procedure for multi-unit feature reviews — that is Feature F.
- Automated triggering of review when all tasks complete.
- A dedicated `review` MCP tool — explicitly deferred per §10.2 of the design until the pattern stabilises.
- First-class review profile objects in workflow state.
- Review metrics and structured review records.

### 3.3 Explicitly excluded

- Changes to the feature lifecycle state machine (`reviewing`, `needs-rework`) — that is Feature D.
- Changes to merge gate logic — that is separate from this feature.
- Task-level review automation — re-exposing `ReviewService` as a 2.0 tool is a separate concern.
- Cross-feature or full-codebase review strategies.

---

## 4. Acceptance Criteria

### 4.1 Reviewer context profile

**4.1.1** The file `.kbz/context/roles/reviewer.yaml` exists and is valid YAML.

**4.1.2** The profile declares `id: reviewer` and `inherits: base`.

**4.1.3** The profile includes a `description` field that identifies it as a context profile for code review agents.

**4.1.4** The profile contains a `conventions` block with at least three named sub-keys:
  - `review_approach` — conventions about structured (not conversational) review, finding format, blocking citation, non-blocking classification, and uncertainty handling.
  - `output_format` — conventions specifying that the review SKILL's output format must be used, that per-dimension outcomes are reported, that an overall verdict is reported, and that blocking findings are listed separately.
  - `dimensions` — the five review dimensions from the quality gates policy: specification conformance, implementation quality, test adequacy, documentation currency, and workflow integrity, each with a one-line description.

**4.1.5** The profile does not define any conventions that contradict the review dimensions or profiles in `work/design/quality-gates-and-review-policy.md`.

**4.1.6** The profile loads without error through the existing profile inheritance mechanism (`ResolveProfile`): the resolved profile includes all fields from `base` and all fields from `reviewer`.

**4.1.7** `context_assemble(role="reviewer")` returns a context packet that contains the reviewer conventions (review_approach, output_format, dimensions). The packet must not be empty and must reflect the reviewer profile fields.

### 4.2 Code review SKILL document

**4.2.1** The file `.skills/code-review.md` exists and follows the SKILL document structure defined in `.skills/README.md` (Purpose, When to Use, Procedure, Verification, Related sections).

**4.2.2** The SKILL declares its intended audience: review sub-agents (follow the procedure), orchestrators (reference for decomposition guidance), and humans (understand what agents will check).

**4.2.3** The SKILL specifies the required inputs for a review unit:
  - a context packet assembled with `role="reviewer"`
  - the list of files changed (or a scoped subset thereof)
  - the relevant specification sections (or a pointer to the document and section paths)
  - the review profile to apply (defaulting to the Feature Implementation Review Profile when not specified)

**4.2.4** The SKILL provides evaluation guidance for each of the five review dimensions defined in the quality gates policy (§4.1–4.5):
  - Specification conformance
  - Implementation quality
  - Test adequacy
  - Documentation currency
  - Workflow integrity

  Each dimension section must identify the key questions to answer (drawn from the quality gates policy) and the conditions under which `not_applicable` is the correct outcome.

**4.2.5** The SKILL defines the structured output format for a review unit, which must include:
  - review unit identifier (entity ID or file scope)
  - review profile used
  - per-dimension outcome (one of: `pass`, `pass_with_notes`, `concern`, `fail`, `not_applicable`)
  - blocking findings list (empty if none), each with: dimension, severity, location, description, and the specific requirement or convention violated
  - non-blocking notes list (empty if none), each with: dimension, description
  - overall verdict (one of: `approved`, `approved_with_followups`, `changes_required`, `blocked`)

**4.2.6** The SKILL defines the classification rule for blocking vs non-blocking findings, consistent with the quality gates policy (§12): blocking findings prevent completion or merge; non-blocking findings are suggestions that do not block.

**4.2.7** The SKILL provides explicit edge case guidance for at least the following scenarios:
  - Missing specification: how to handle review when no approved spec document exists for the feature.
  - Partial implementation: how to evaluate dimensions when the feature is known to be incomplete.
  - Ambiguous conformance: when it is unclear whether an implementation matches the spec, classify as `concern` rather than `fail` and describe the ambiguity.
  - Missing context: when file content or spec sections cannot be retrieved, note this in the review output rather than omitting the dimension.

**4.2.8** The SKILL explicitly states that it does not cover orchestration (decomposition into multiple review units, lifecycle transitions, task creation) — that is the orchestrator's responsibility.

**4.2.9** The SKILL includes a verification checklist that an agent can use to confirm that a review is complete before submitting output.

**4.2.10** The SKILL cross-references the quality gates policy (`work/design/quality-gates-and-review-policy.md`) as the canonical source for dimension definitions, profile definitions, and blocking/non-blocking classification policy.

### 4.3 Quality gates policy update

**4.3.1** `work/design/quality-gates-and-review-policy.md` contains a new section (or updated existing section) that explicitly states: the review dimensions and profiles defined in this policy are operationalised through the reviewer context profile (`.kbz/context/roles/reviewer.yaml`) and the code review SKILL (`.skills/code-review.md`).

**4.3.2** The policy update includes a reference to `work/design/code-review-workflow.md` as the design that describes the full review workflow.

**4.3.3** The policy update does not change any normative content of the policy — no review dimension, profile, or classification rule is modified. The update is purely additive (cross-references and operationalisation note).

### 4.4 README update

**4.4.1** `.skills/README.md` is updated to include an entry for `code-review` in the Existing SKILLs table, with a brief description of its purpose and when to use it.

---

## 5. Verification

The following checks constitute verification for this feature:

| # | Check | Method |
|---|-------|--------|
| V1 | `.kbz/context/roles/reviewer.yaml` exists and parses as valid YAML | Manual inspection or `profile_get(id="reviewer")` |
| V2 | Profile inherits `base`; resolved packet includes base conventions | `profile_get(id="reviewer", resolved=true)` |
| V3 | `context_assemble(role="reviewer")` returns non-empty packet with reviewer conventions | Call the tool; inspect output |
| V4 | Reviewer conventions include all five dimensions | Inspect profile YAML; compare against quality gates policy §4.1–4.5 |
| V5 | `.skills/code-review.md` exists and follows SKILL format | Manual inspection against `.skills/README.md` format template |
| V6 | SKILL covers all five dimensions with evaluation guidance | Check each dimension has a section in the SKILL |
| V7 | SKILL defines structured output format with all required fields | Inspect output format section against AC 4.2.5 |
| V8 | SKILL defines blocking vs non-blocking classification | Inspect classification section against quality gates policy §12 |
| V9 | SKILL covers all four edge cases | Inspect edge case section against AC 4.2.7 |
| V10 | Quality gates policy references reviewer profile and review SKILL | Inspect updated section in `quality-gates-and-review-policy.md` |
| V11 | No normative content changed in quality gates policy | Diff review: additions only, no changes to existing §4–§12 content |
| V12 | `.skills/README.md` lists `code-review` SKILL | Manual inspection |
```

Now let me create that file and register it: