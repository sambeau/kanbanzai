| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:06:29Z |
| Status | approved |
| Author | spec-author |
| Plan | P59 — Roles & Skills Discoverability and Enforcement Remediation |
| Batch | B61 — Tidy contradictions stray files and corpus size |
| Feature | FEAT-01KR3MES5ZJ11 — Instruction corpus cleanup |
| Design | `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` |

## Overview

This specification implements the B4 Tidy portion of the design described in `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` (`P59-roles-skills-remediation/design-p59-design-roles-skills-remediation`). It covers mechanical cleanup of contradictions, stray skill files, duplicated role and skill prose, over-budget skills, long examples, and inherited tool lists.

The problem is that agents receive contradictory and overly large instruction corpora. The live `stash` contradiction causes rule conflicts, stray `SKILL.md` files create false discovery surfaces, and duplicated anti-patterns make drift likely. The cleanup must reduce corpus size without deleting rules or changing the underlying role and skill model.

## Scope

In scope:

- Replacing the two stray top-level `SKILL.md` files with directory indexes.
- Aligning top-level instructions on the project-wide no-stash rule.
- Bringing over-budget skills under the project line-count limit.
- Moving long examples to reference documents and linking to them.
- Replacing duplicate anti-pattern prose with cross-references to canonical role entries.
- Removing broad inherited `grep` and `search_graph` tool access from the base role and restoring them explicitly only where needed.
- Measuring corpus size before and after cleanup.

Out of scope:

- Changing the canonical role and skill authoring format.
- Rewriting role identities, stage bindings, or skill procedures for new behaviour.
- Implementing the B60 registry generator or B62 runtime discovery wrappers.
- Deleting rules to meet the size target.

Related work checked:

- `work/P59-roles-skills-remediation/P59-report-roles-skills-audit.md` identifies the stash contradiction, stray skill files, duplicated anti-patterns, and corpus size problem.
- `.agents/skills/kanbanzai-getting-started/SKILL.md` already states the no-stash rule and is the canonical behaviour for session start.
- `AGENTS.md` store and git discipline sections require committing workflow state and not hiding work in stash.

## Functional Requirements

- **REQ-001:** `.kbz/skills/SKILL.md` must be removed as a skill file and replaced with a `README.md` index that explains where real `.kbz/skills/*/SKILL.md` entries live.
- **REQ-002:** `.agents/skills/SKILL.md` must be removed as a skill file and replaced with a `README.md` index that explains where real `.agents/skills/*/SKILL.md` entries live.
- **REQ-003:** Top-level agent instruction files must consistently state that Kanbanzai agents do not use `git stash`; previous work is committed, left for the human, or isolated with worktrees.
- **REQ-004:** The `orchestrate-development` skill must be reduced below 500 lines without deleting required procedure, checklist, or output-format rules.
- **REQ-005:** The `kanbanzai-agents` skill must be reduced below 500 lines without deleting required dispatch, commit, finish, or knowledge-contribution rules.
- **REQ-006:** Any SKILL.md `Examples` section longer than 60 lines must be moved to a `references/examples-<skill>.md` file or equivalent reference location, and the skill must link to the moved examples.
- **REQ-007:** Anti-pattern prose duplicated verbatim or near-verbatim between a role YAML file and a skill SKILL.md must keep one canonical copy in the role and replace the skill copy with a concise reference.
- **REQ-008:** The base role must not expose `grep` and `search_graph` by default if those tools propagate to roles that do not need them.
- **REQ-009:** Roles that legitimately need `grep` or `search_graph` after REQ-008 must list them explicitly in that role file.
- **REQ-010:** The cleanup must produce a before-and-after corpus-size report that counts skill text lines and identifies the largest remaining skill files.
- **REQ-011:** The total skill corpus must be reduced to no more than 6,500 counted lines unless a documented exception is approved during review.
- **REQ-012:** The cleanup must not remove or weaken any hard workflow rule identified by P59 B2.

## Non-Functional Requirements

- **REQ-NF-001:** Cleanup changes must be reviewable as mechanical moves, deletions, or cross-reference substitutions; broad rewrites are not acceptable for this feature.
- **REQ-NF-002:** Moving examples to reference files must preserve their text content except for heading normalization and link context.
- **REQ-NF-003:** Any generated or moved reference file must be discoverable from the source skill through a relative link.
- **REQ-NF-004:** The corpus-size measurement must be repeatable by a documented command or script.
- **REQ-NF-005:** Removing default tool access from the base role must not leave any stage-bound role without the tools required by its skill procedure.

## Constraints

- Rules are preserved; savings come from de-duplication, moving examples, and deleting false discovery files.
- The no-stash rule is the resolved behaviour. This feature must not preserve a weaker `commit or stash` alternative in generated or hand-authored instruction text.
- Task-execution skills under `.kbz/skills/` are project-local; no dual-write to `internal/kbzinit/skills/` is required for those files. Changes to `.agents/skills/kanbanzai-*` must follow the project's dual-write rule where a corresponding embedded copy exists.
- Directory indexes that replace stray `SKILL.md` files must not themselves look like executable skills to host runtimes.
- B60 may later generate registry sections. This cleanup must not depend on B60 landing first.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given `.kbz/skills/`, when the directory is inspected, then `.kbz/skills/SKILL.md` is absent and `.kbz/skills/README.md` explains the canonical skill locations.
- **AC-002 (REQ-002):** Given `.agents/skills/`, when the directory is inspected, then `.agents/skills/SKILL.md` is absent and `.agents/skills/README.md` explains the canonical skill locations.
- **AC-003 (REQ-003):** Given top-level instruction files, when their git-start guidance is inspected, then none instruct agents to use `git stash` for Kanbanzai workflow state.
- **AC-004 (REQ-004):** Given `.kbz/skills/orchestrate-development/SKILL.md`, when counted, then it has fewer than 500 lines and retains procedure, checklist, and output-format sections.
- **AC-005 (REQ-005):** Given `.agents/skills/kanbanzai-agents/SKILL.md`, when counted, then it has fewer than 500 lines and retains dispatch, commit, finish, and knowledge-contribution guidance.
- **AC-006 (REQ-006):** Given any skill that previously had an `Examples` section longer than 60 lines, when inspected, then the examples are in a reference file and the skill links to it.
- **AC-007 (REQ-007):** Given role and skill files, when duplicate anti-pattern text is searched, then duplicated skill copies have been replaced by references to the canonical role entries.
- **AC-008 (REQ-008, REQ-009):** Given role tool lists, when inheritance is resolved, then `grep` and `search_graph` are not inherited from base and roles that need them list them explicitly.
- **AC-009 (REQ-010, REQ-011):** Given the corpus-size report, when before-and-after counts are compared, then the final counted corpus is at or below 6,500 lines or the exception is documented.
- **AC-010 (REQ-012):** Given the B59 invariant catalog, when cleanup diffs are reviewed, then every hard workflow invariant remains discoverable after cleanup.
- **AC-011 (REQ-NF-005):** Given stage-bound role and skill pairs, when tool needs are reviewed, then no role lost a required tool because of base-role cleanup.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Confirm `.kbz/skills/SKILL.md` is absent and `.kbz/skills/README.md` points to real skill directories. |
| AC-002 | Inspection | Confirm `.agents/skills/SKILL.md` is absent and `.agents/skills/README.md` points to real skill directories. |
| AC-003 | Search | Search top-level instruction files for stash guidance and verify no Kanbanzai rule permits stash use. |
| AC-004 | Script | Count lines in `.kbz/skills/orchestrate-development/SKILL.md` and inspect required sections. |
| AC-005 | Script | Count lines in `.agents/skills/kanbanzai-agents/SKILL.md` and inspect required sections plus embedded dual-write copy if applicable. |
| AC-006 | Inspection | Inspect moved example reference files and links from source skills. |
| AC-007 | Search/inspection | Search for duplicate anti-pattern prose and verify skill files reference canonical role entries. |
| AC-008 | Test/inspection | Resolve role inheritance or inspect role files to verify base no longer carries broad code-search tools and required roles re-add them. |
| AC-009 | Script | Run the documented corpus-size command and verify the final count threshold or exception. |
| AC-010 | Inspection | Compare B59 invariant names against post-cleanup discoverable surfaces. |
| AC-011 | Review | Review each stage-bound role and skill pair for required tool availability after base cleanup. |
