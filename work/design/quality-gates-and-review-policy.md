# Quality Gates and Review Policy

- Status: policy draft
- Purpose: define the standard quality gates and review expectations for workflow-managed work
- Date: 2026-03-18
- Related:
  - `workflow-design-basis.md`
  - `phase-1-specification.md`
  - `agent-interaction-protocol.md`

---

## 1. Purpose

This document defines the review and quality gate policy for work managed through the workflow system.

Its purpose is to ensure that repeated expectations such as:

- “does this match the specification?”
- “is the implementation high quality?”
- “are the tests suitable?”
- “is the documentation up to date?”

are not left as ad hoc prompts in chat.

Instead, these expectations should be:

- explicitly defined
- consistently applied
- reusable across features and bugs
- understandable by humans
- operationalizable by AI agents
- eventually queryable and enforceable by the workflow system

This policy exists to reduce repeated prompting, improve consistency, and make “done” mean something reliable.

---

## 2. Core Principle

No meaningful unit of work should be treated as complete merely because code exists.

Completion requires quality review against the correct standard.

In this policy, quality review means checking work across multiple dimensions, not just asking whether the code “looks fine”.

At minimum, the workflow must be able to answer:

- does the implementation match the approved specification?
- is the implementation technically sound and idiomatic?
- are the tests appropriate and sufficient?
- is the documentation accurate and up to date?

---

## 3. Scope

This policy applies to:

- feature implementation review
- bugfix review
- readiness-for-merge review
- completion review
- documentation update review
- test adequacy review

This policy is intended to guide:

- human reviewers
- AI agents performing reviews
- workflow quality gates
- future review automation

This document defines the policy, not the exact implementation mechanism.

---

## 4. Review Dimensions

All substantive implementation review should be broken into explicit review dimensions.

## 4.1 Specification Conformance

The review must check whether the implemented work matches the approved intended behavior.

Questions include:

- was everything required by the relevant specification implemented?
- was anything materially omitted?
- was anything added that conflicts with the specification?
- was any behavior changed without a corresponding decision or spec change?
- if implementation deviated from the specification, is that deviation explicitly recorded and justified?

This is the most important review dimension for feature work.

## 4.2 Implementation Quality

The review must assess the quality of the implementation itself.

Questions include:

- is the solution appropriate to the task?
- is the code idiomatic for the language, framework, and codebase?
- is the structure understandable and maintainable?
- are abstractions appropriate in size and complexity?
- does the implementation introduce unnecessary complexity?
- are naming, organization, and interfaces clear?
- are comments present where needed and absent where unnecessary?
- is the code aligned with existing project conventions?

“High standard” does not mean maximal cleverness. It means clear, correct, idiomatic, maintainable work.

## 4.3 Test Adequacy

The review must assess both the presence and quality of tests.

Questions include:

- does the change have appropriate automated test coverage where warranted?
- do the tests meaningfully verify behavior rather than just existence?
- do the tests match the relevant acceptance criteria?
- are tests focused, readable, and maintainable?
- are important edge cases or regressions covered?
- are tests placed at the correct level (unit, integration, end-to-end, manual)?
- do the tests appear brittle, redundant, or misleading?

The goal is not “maximum test count”. The goal is appropriate verification.

## 4.4 Documentation Currency

The review must check whether relevant documentation has been updated.

Questions include:

- is user-facing documentation up to date?
- is developer-facing documentation up to date where affected?
- do changed workflows or interfaces have updated explanation?
- are references to behavior, commands, fields, or states still correct?
- does the documentation meet project standards?

Documentation should not be updated mechanically if it was not affected. But if the implementation changes behavior, documentation must not be left stale.

## 4.5 Workflow Integrity

The review should check whether workflow records still align with the work.

Questions include:

- is the correct feature, bug, or task linked?
- is the current state accurate?
- are relevant decisions linked?
- is supersession handled correctly where applicable?
- is the work being marked complete at the correct workflow stage?
- is any follow-up work missing from the workflow record?

This dimension helps prevent workflow drift even when code is correct.

---

## 5. Review Profiles

A review profile is a named bundle of required review dimensions for a class of work.

Profiles allow review expectations to be reused without restating them in chat.

## 5.1 Feature Implementation Review Profile

This is the default review profile for feature work.

Required dimensions:

- specification conformance
- implementation quality
- test adequacy
- documentation currency
- workflow integrity

This profile should be treated as the default answer to:

> “Please check the implementation of feature X.”

If no special profile is specified, feature review should use this one.

## 5.2 Bugfix Review Profile

This is the default review profile for bugfix work.

Required dimensions:

- bug conformance
- implementation quality
- regression prevention
- documentation currency where affected
- workflow integrity

Where “bug conformance” means:

- does the fix address the reported observed behavior?
- does it preserve the expected behavior?
- is the bug classification still correct?
- does the fix follow the right path for:
  - implementation defect
  - specification defect
  - design problem

Where “regression prevention” means:

- does the fix include or imply appropriate regression coverage?
- is there enough evidence the bug will not immediately recur?
- has the reproduction path been used for verification where possible?

## 5.3 Merge Readiness Review Profile

This profile is for deciding whether a branch or feature is ready to merge.

Required dimensions:

- all required review dimensions for the work type pass
- required validations pass
- workflow state is current
- no unresolved blocking review findings remain
- required documentation updates are complete
- no known stale-spec or stale-branch issue blocks integration

This profile is stricter than ordinary implementation review because it is an integration gate.

## 5.4 Lightweight Review Profile

This is for small, low-risk changes where a full feature review is unnecessary.

Required dimensions:

- implementation quality
- test adequacy where applicable
- documentation currency where applicable
- workflow integrity

Specification conformance may be omitted only when there is no separate approved specification for the change.

Use this sparingly.

---

## 6. Feature Review Policy

When reviewing a feature implementation, the reviewer must apply the Feature Implementation Review Profile unless a more specific profile is defined.

The review must answer, at minimum:

1. did the implementation satisfy the specification?
2. is the implementation technically sound and idiomatic?
3. are the tests appropriate and sufficient?
4. is the documentation up to date and standards-compliant?
5. is the workflow state consistent with what was done?

A feature should not be treated as “done” merely because it compiles or because the main happy path appears to work.

---

## 7. Bugfix Review Policy

When reviewing a bugfix, the reviewer must apply the Bugfix Review Profile unless a more specific profile is defined.

The review must answer, at minimum:

1. does the fix address the reported problem?
2. was the bug classified correctly?
3. was the right workflow path taken?
4. is there adequate regression protection?
5. is documentation updated if behavior changed?
6. is the workflow state accurate and complete?

A bugfix is not complete simply because the symptom seems gone in one run.

---

## 8. Documentation Review Policy

Documentation review is part of implementation review, but it also deserves explicit policy.

Documentation must be reviewed for:

- correctness
- currency
- appropriate scope
- conformance to project documentation standards

Documentation review does not require rewriting unrelated documents.
It requires ensuring that all documentation affected by the change is now accurate.

Documentation should be considered stale if it:

- describes behavior that no longer exists
- omits new user-visible behavior
- references outdated structures, commands, or terms
- contradicts canonical workflow or implementation behavior

---

## 9. Test Review Policy

Tests must be reviewed for suitability, not merely presence.

Test review must consider:

- whether the change needed tests
- whether the chosen test level was appropriate
- whether the tests actually check behavior
- whether acceptance criteria are covered
- whether regressions are guarded against where appropriate
- whether tests are maintainable and non-misleading

The correct outcome of test review may be:

- sufficient
- insufficient
- unnecessary for this particular change
- partially sufficient with follow-up needed

The policy should allow reviewers to say “no additional test required” when that conclusion is justified, rather than forcing meaningless tests.

---

## 10. Code Quality Policy

Code quality review must consider:

- correctness
- clarity
- idiomatic style
- maintainability
- complexity
- alignment with project conventions

The policy does **not** require:

- over-engineering
- abstraction for hypothetical future use
- comments where code is already self-evident
- refactoring unrelated areas unless needed for correctness

The quality standard should be demanding but not ornamental.

---

## 11. Review Output Format

Review findings should be reported by dimension, not as one undifferentiated judgment.

At minimum, review output should include:

- object reviewed
- review profile used
- result for each required dimension
- important concerns
- blocking issues
- non-blocking follow-ups
- overall outcome

Suggested dimension outcomes:

- `pass`
- `pass_with_notes`
- `concern`
- `fail`
- `not_applicable`

Suggested overall outcomes:

- `approved`
- `approved_with_followups`
- `changes_required`
- `blocked`

This structure makes review much more reusable and machine-friendly.

---

## 12. Blocking vs Non-Blocking Findings

The review policy should distinguish between:

### Blocking findings
Issues that prevent the work from being considered complete or ready to merge.

Examples:
- implementation does not satisfy the specification
- invalid or missing critical tests
- serious bug remains unaddressed
- documentation is materially incorrect
- workflow record is in a misleading or broken state

### Non-blocking findings
Issues worth noting but not severe enough to block completion.

Examples:
- minor naming improvements
- useful follow-up cleanups
- non-critical documentation polish
- optional extra test coverage
- minor comment improvements

This distinction is essential. Without it, review becomes noisy and indecisive.

---

## 13. Human and Agent Responsibilities in Review

## 13.1 Human role
Humans remain the final authority on:
- whether the result matches intended product direction
- whether the review outcome is acceptable
- whether to approve, request changes, or defer

## 13.2 Agent role
Agents may:
- perform structured review
- check conformance against specifications
- inspect tests
- inspect docs
- summarize findings
- propose approval or changes required

Agents must not pretend a human approval has happened when it has not.

---

## 14. Relationship to the Workflow System

This policy should ultimately be represented in the workflow system as structured review expectations rather than only prose.

In later phases, the workflow system should be able to represent concepts such as:

- review profiles
- required review dimensions by work type
- quality gates before merge or completion
- review outcomes as structured records
- project-specific review policy variations

Phase 1 does not need to fully implement this as first-class state, but the workflow and protocol should be designed to allow it.

---

## 15. Relationship to the Agent Interaction Protocol

The `agent-interaction-protocol.md` defines **how agents should behave**.

This document defines **what review and quality checks they must apply** when performing implementation review.

The relationship is:

- the interaction protocol governs normalization and safe commit behavior
- this review policy governs evaluation and quality-gate behavior

Both are needed.

---

## 16. Relationship to Skills and Instructions

This policy should not live only as a repeated user prompt.

It should eventually be surfaced through:

- static instruction files
- workflow-specific skills
- machine-readable workflow policy
- review-oriented tooling and MCP support

A likely future pattern is:

- runtime instructions teach the agent that implementation review must follow this policy
- skills package the review flow for common runtimes
- workflow policy exposes the required review dimensions and profiles
- MCP tools support structured review reporting

That is how the repeated sentence becomes part of the system rather than part of your memory.

---

## 17. Bootstrap Guidance

Until the workflow system can represent review policy structurally, this document should serve as the canonical policy reference for implementation review.

During the bootstrap phase:

- humans may still explicitly ask for review
- agents should already apply this policy when asked
- review results should be written or summarized in a way that can later be formalized
- repeated review prompts should increasingly be replaced with references to this policy

The goal is to stop relying on repeated manual prompting as early as possible.

---

## 18. Standard Default Review Instruction

Until the workflow system can invoke review profiles formally, the default interpretation of a request like:

> “Please check the implementation of feature X.”

should be:

> Review feature X using the default feature implementation review profile. Check specification conformance, implementation quality, test adequacy, documentation currency, and workflow integrity. Report findings by dimension, identify blockers separately from non-blocking notes, and state whether the feature is approved, approved with follow-ups, or requires changes.

This should be treated as the canonical replacement for repeatedly typing the longer manual prompt.

---

## 19. Future Evolution

Likely future additions include:

- first-class review profile objects
- project-level override of review profiles
- structured review records
- explicit merge gate definitions
- bugfix-specific and release-specific gate policies
- policy query support through the workflow interface
- structured “review complete” workflow transitions

These are deferred, but the architecture should leave room for them.

---

## 20. Summary

This document defines the standard quality gates and review expectations for workflow-managed work.

Its key commitments are:

- implementation review must be explicit and multi-dimensional
- specification conformance, code quality, tests, documentation, and workflow integrity are separate review concerns
- default review expectations should be reusable and not retyped by hand
- review findings should be structured
- blockers must be distinguished from non-blocking notes
- this policy should ultimately be represented by the workflow system, not only remembered by humans

This document exists so that the repeated “please check the implementation…” instruction can become a stable part of the process rather than a repeated manual prompt.