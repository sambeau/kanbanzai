---
id: DOC-01KM8JV0HTDBN
type: design
title: Git Commit Policy
status: submitted
feature: FEAT-01KM8JT7542GZ
created_by: human
created: 2026-03-21T16:14:48Z
updated: 2026-03-21T16:14:48Z
---
# Git Commit Policy

- Status: policy draft
- Purpose: define how Git commits should be created, structured, described, and managed in the workflow system
- Date: 2026-03-18
- Related:
  - `work/design/workflow-design-basis.md`
  - `work/design/agent-interaction-protocol.md`
  - `work/design/quality-gates-and-review-policy.md`
  - `work/spec/phase-1-specification.md`

---

## 1. Purpose

This document defines the Git commit policy for work managed through the workflow system.

Its purpose is to ensure that commits are:

- coherent
- traceable
- reviewable
- bisectable
- safe for concurrent human and AI work
- aligned with workflow objects such as features, tasks, bugs, and decisions

This policy exists because source control is not just a storage mechanism. In an AI-assisted workflow, it is also:

- an audit trail
- a review surface
- a recovery mechanism
- a coordination layer
- an important source of context for future work

A bad commit history creates confusion, hides mistakes, and makes verification harder. A good commit history strengthens the entire process.

---

## 2. Core Principles

### 2.1 One coherent change per commit

A commit should represent one coherent unit of change.

That means:

- no unrelated changes bundled together
- no drive-by cleanup mixed into feature work
- no speculative refactors hidden inside bug fixes
- no formatting churn mixed with behavioral changes unless required

A commit should answer a clear question:

> What changed, and why did these changes belong together?

### 2.2 Commits must be traceable to workflow objects

Every substantive commit should be traceable to at least one workflow object, such as:

- Epic
- Feature
- Task
- Bug
- Decision

The relationship does not always have to be one-to-one, but it must be clear.

### 2.3 Commit history must support review and diagnosis

The commit history should help a reviewer or future agent understand:

- what changed
- why it changed
- which workflow object it belongs to
- whether the change is likely safe
- where to look if something broke

This means commits should remain understandable in isolation.

### 2.4 Commits should help, not hinder, Git bisect

Where practical, commits should be:

- internally consistent
- buildable or at least structurally coherent
- meaningful as units of history

The ideal is not perfection at every tiny step, but commits should not leave the branch in a confusing or arbitrarily broken state without good reason.

### 2.5 Commit discipline matters more with AI agents

AI agents can create large volumes of changes quickly. This makes commit discipline more important, not less.

The commit policy should make it harder for agents to:

- hide unrelated changes
- create vague history
- dump large unreviewable bundles
- checkpoint recklessly
- muddy the audit trail

---

## 3. Scope

This policy applies to:

- direct human commits
- AI-agent-created commits
- commits on feature branches
- commits on bugfix branches
- commits containing workflow state updates
- commits containing documentation changes
- commits intended for later merge into shared branches

This policy does not define:

- the full branching strategy
- pull request review workflow
- merge policy in full

Those are related but separate concerns.

---

## 4. What Should Be in a Commit

## 4.1 A commit should contain one coherent unit of work

Examples of coherent units of work:

- implement one task
- fix one bug
- record one decision and related necessary updates
- add one schema change and its immediate required updates
- update documentation for one completed behavior change

Examples of non-coherent commits:

- bug fix + unrelated cleanup
- feature work + formatting unrelated files
- task implementation + opportunistic refactor in another subsystem
- documentation rewrite unrelated to the code change being committed

## 4.2 Allowed commit scopes

A commit may span multiple files if those files belong to the same logical change.

Examples:

- API handler + model + tests for one task
- bug fix + regression test + relevant documentation update
- workflow state update + linked spec update for the same feature

The policy is not “one file per commit”.
The policy is “one coherent change per commit”.

## 4.3 Workflow state and implementation changes

Where workflow state changes are directly tied to code changes, they may be committed together if doing so improves traceability.

Examples where combined commits are acceptable:

- implementing `TASK-123` and updating its status/completion metadata
- fixing `BUG-027` and recording the linked fix reference
- adding a decision and updating the affected feature state in the same coherent change

Examples where separate commits may be preferable:

- a broad planning update before implementation begins
- specification revision before any implementation work starts
- cleanup of stale workflow state unrelated to the code currently being committed

The rule is:

> combine workflow and implementation changes when they form one coherent change;
> separate them when combining would blur meaning.

## 4.4 Generated files

Generated projections, reports, or derived artifacts should generally not be committed unless one of the following is true:

- the project explicitly requires them in version control
- they are treated as stable public artifacts
- they are necessary for review or downstream tooling
- the policy for that artifact says they must be committed

If generated artifacts are not canonical and can be regenerated, they should usually be excluded from ordinary commits.

---

## 5. What Must Not Be in a Commit

A commit must not include:

- unrelated cleanups
- speculative changes
- hidden refactors unrelated to the stated purpose
- leftover debug changes
- unfinished unrelated work
- generated noise with no review value
- broad formatting churn unrelated to the target change
- accidental file moves or renames outside scope
- partially normalized workflow state that misrepresents reality

Agents and humans alike must avoid the temptation to “sneak in” extra work.

---

## 6. Commit Message Policy

## 6.1 Message goals

A commit message should make it easy to understand:

- what changed
- why it changed
- which workflow object it belongs to

## 6.2 Required structure

Commit messages should include:

1. a change type
2. a workflow reference where applicable
3. a concise summary

Recommended format:

`<type>(<object-id>): <summary>`

Examples:

- `feat(FEAT-152): add profile editing API and validation`
- `fix(BUG-027): prevent avatar uploads hanging on large files`
- `docs(FEAT-152): update profile editing user documentation`
- `workflow(TASK-152.3): mark upload task complete after verification`
- `decision(DEC-041): record no-client-side-cropping choice`

## 6.3 Preferred commit types

Recommended standard commit types:

- `feat` — new feature behavior
- `fix` — bug fix
- `docs` — documentation change
- `test` — test-only change
- `refactor` — behavior-preserving structural improvement
- `workflow` — workflow-state-only change
- `decision` — decision-record change
- `chore` — small maintenance change with no better category

If the project later standardizes a stricter type vocabulary, this policy may be updated.

## 6.4 Summary style

The summary should be:

- concise
- specific
- readable
- action-oriented where possible

Good:

- `fix(BUG-027): show upload error for files over size limit`

Bad:

- `fix stuff`
- `updates`
- `work on feature`
- `misc changes`

## 6.5 Multi-line commit messages

When useful, a commit message may include a body that explains:

- rationale
- important constraints
- non-obvious implementation notes
- key follow-up implications

This is especially useful for:
- decisions
- bug fixes with subtle root cause
- commits that intentionally preserve an important tradeoff

---

## 7. Preconditions Before Commit

Before creating a commit, the committer should ensure that:

- the commit contents match one coherent purpose
- relevant validation has been run where appropriate
- workflow state is not left misleading
- the commit message is accurate
- no unrelated files are included
- required linked docs or tests for the scope are included where needed

For AI agents, this means the agent should not commit immediately after code generation without first checking scope and relevance.

---

## 8. Commits and Review Dimensions

A commit does not itself prove that work is complete.

Even a good commit may still fail review.

The quality gates and review policy still apply.

However, good commit discipline should make later review easier by making it clear:

- which commit belongs to which task or bug
- where relevant tests were added
- where docs changed
- where workflow state changed

---

## 9. Commit Policy for Features

Feature work should generally commit along task or coherent-subtask boundaries.

Recommended pattern:

- one meaningful implementation step per commit
- include tests and docs when naturally part of the same change
- avoid very large “complete entire feature” commits unless the feature is genuinely tiny

Feature commits should usually be traceable to:

- `Feature`
- `Task`
- and optionally `Decision`

Examples:

- `feat(FEAT-152.1): add profile persistence model`
- `feat(FEAT-152.2): implement profile update endpoint`
- `test(FEAT-152.2): add endpoint validation coverage`
- `docs(FEAT-152): document profile editing behavior`

---

## 10. Commit Policy for Bugs

Bugfix commits should generally include, where applicable:

- the fix
- regression protection
- any necessary documentation updates
- linked workflow state update if it is part of the same coherent change

A bugfix commit should make it easy to answer:

- what bug was fixed?
- what changed to fix it?
- how is recurrence guarded against?

Examples:

- `fix(BUG-027): reject oversized avatar uploads with clear error`
- `test(BUG-027): add regression coverage for upload size failures`

---

## 11. Commit Policy for Workflow State

Workflow-state-only commits are acceptable when they represent a coherent workflow action.

Examples:

- creating a new feature record
- recording a decision
- correcting broken workflow links
- updating planning state before implementation begins

These commits should use a `workflow` or `decision` style message and remain tightly scoped.

Examples:

- `workflow(FEAT-152): create feature record and scaffold spec`
- `decision(DEC-041): record avatar cropping decision`

Workflow commits must not be used as a dumping ground for miscellaneous process edits.

---

## 12. Commit Policy for Documentation

Documentation-only commits are acceptable and often desirable when:

- documentation changes are substantial enough to review separately
- documentation changes are not naturally tied to a single code change
- documentation corrections are needed independently of implementation changes

If documentation is tightly coupled to a specific implementation change, combining them in one coherent commit is acceptable.

Examples:

- `docs(FEAT-152): add profile editing manual page`
- `docs(BUG-027): clarify avatar upload size limits`

---

## 13. Commit Policy for Refactors

Refactors must be especially disciplined.

A `refactor` commit should:

- preserve behavior
- be reviewable as structural change
- not be mixed with feature behavior changes unless unavoidable

If a refactor is necessary to enable a feature or fix, consider:

- separating the refactor into its own commit first, if that improves clarity
- or documenting clearly in the commit message why the structural change belonged with the main change

Refactor commits must not become cover for unrelated cleanup.

---

## 14. Prohibited Commit Behaviors for Agents

Unless explicitly instructed otherwise, AI agents must not:

- create vague “checkpoint” commits
- create commits with generic summaries
- commit unrelated touched files just because they were modified
- amend or rewrite published history
- force-push shared branches
- bypass validation as a shortcut
- commit speculative changes “for later”
- treat generated output as worth committing by default
- mark work complete in workflow state if review/verification is not actually done

This is especially important in autonomous or semi-autonomous flows.

---

## 15. Checkpoint Commits

Checkpoint commits are discouraged by default.

They are only acceptable when:

- explicitly requested
- operating in a private experimental branch
- used as a temporary safety mechanism during risky local work
- clearly labeled as temporary or intermediate

Even then, they should still be coherent and clearly described.

Examples:

- `chore(FEAT-152): checkpoint before schema migration refactor`

Checkpoint commits must not be merged into shared history casually.

---

## 16. Relationship to Branches and Pull Requests

This policy applies within branches before pull requests as well as at merge time.

A pull request should ideally present a branch history that is:

- understandable
- scoped
- traceable
- not noisy with irrelevant commits

If squash merge is the team norm, commit quality still matters because:

- it helps review before squashing
- it helps debugging before merge
- it helps the agent reason about its own work during execution
- not all branches will necessarily be squashed immediately

---

## 17. Error Correction

If a wrong commit is made, the response should depend on context.

### 17.1 Before publishing
If the branch is private and unpublished, local cleanup may be acceptable.

### 17.2 After publishing or shared visibility
Prefer non-destructive correction:

- follow-up corrective commit
- workflow correction
- explicit supersession where appropriate

Avoid destructive history rewriting on shared branches unless explicitly approved.

The workflow should preserve an understandable audit trail.

---

## 18. Relationship to the Workflow System

This policy should eventually be represented in the workflow system more formally.

Future capabilities may include:

- commit policy query support
- commit linting against workflow objects
- required workflow references in commit messages
- validation that changed files align with intended task scope
- merge gates based on commit-policy compliance

Phase 1 does not need to implement all of this, but the implementation should not block it.

---

## 19. Relationship to Agent Protocol and Review Policy

This policy works together with:

- `work/design/agent-interaction-protocol.md`
  - defines how agents behave around normalization and commit safety
- `work/design/quality-gates-and-review-policy.md`
  - defines how work is evaluated before being considered complete

This commit policy defines:

- how changes should be recorded in Git

The interaction protocol defines:

- how agents should decide and behave before committing

The review policy defines:

- how we judge whether the work is acceptable after it has been committed

All three are needed.

---

## 20. Bootstrap Guidance

During the bootstrap phase of the workflow tool itself:

- humans and agents should begin following this policy immediately
- simplicity should be preferred over elaborate automation
- manual discipline is acceptable until enforcement tooling exists
- the workflow tool’s own development should be committed in a way that is traceable to its own epics, features, tasks, bugs, and decisions whenever practical

The goal is to start living by the policy before the system can fully enforce it.

---

## 21. Standard Default Guidance for Agents

Until the workflow system can enforce this more formally, agents should treat the following as default commit behavior:

> Create commits that represent one coherent unit of work, include the relevant workflow reference in the commit message, avoid unrelated changes, and ensure the commit history remains understandable, reviewable, and useful for diagnosis.

This should be treated as the default working rule unless a more specific policy overrides it.

---

## 22. Acceptance Criteria

This policy is acceptable only if following it would produce a commit history that:

1. is traceable to workflow objects
2. avoids bundling unrelated changes
3. produces useful commit messages
4. supports review and diagnosis
5. works for both humans and AI agents
6. does not require repeated manual prompting to maintain discipline
7. supports gradual future automation and enforcement

---

## 23. Summary

This document defines how Git commits should be created in the workflow system.

Its key commitments are:

- one coherent change per commit
- traceability to workflow objects
- meaningful, structured commit messages
- no unrelated bundled changes
- workflow state updates committed coherently
- no reckless checkpointing or vague AI history
- commit discipline as a first-class part of the process

This policy exists so that source control becomes a reliable part of the workflow system rather than an unmanaged side channel.