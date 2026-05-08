| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:06:29Z |
| Status | approved |
| Author | spec-author |
| Plan | P59 — Roles & Skills Discoverability and Enforcement Remediation |
| Batch | B60 — Unify role and skill registries |
| Feature | FEAT-01KR3MEJGGMT5 — Generated role and skill registry surfaces |
| Design | `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` |

## Overview

This specification implements the B3 Unify portion of the design described in `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` (`P59-roles-skills-remediation/design-p59-design-roles-skills-remediation`). It covers generated role and skill registry sections for the top-level instruction surfaces that currently drift independently.

The problem is that `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md` each describe roles, skills, and stage bindings in separate hand-maintained text. When those copies drift, agents receive conflicting or incomplete guidance. This specification makes `.kbz/stage-bindings.yaml` plus `.kbz/roles/*.yaml` the single source of truth for the registry and requires derived surfaces to be checkable in CI.

## Scope

In scope:

- A registry extraction model derived from stage bindings and role metadata.
- Generated registry regions in `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md`.
- Marker comments that preserve surrounding hand-authored prose.
- A sync mode that writes generated regions.
- A check mode that fails when generated regions are stale.
- A CI target that runs the check mode.

Out of scope:

- Generating the full narrative body of `AGENTS.md`.
- Changing the stage-binding schema for this feature.
- Re-authoring roles or skills for content quality; B61 covers cleanup and de-duplication.
- Creating `.claude/skills/` wrappers; B62 covers runtime discovery wrappers.

Related work checked:

- `work/P59-roles-skills-remediation/P59-report-roles-skills-audit.md` identifies multi-file registry drift as a primary failure mode.
- `.kbz/stage-bindings.yaml` is the canonical map of stage to role and skill.
- `.kbz/roles/*.yaml` provides role identity and inheritance metadata used by generated registry surfaces.

## Functional Requirements

- **REQ-001:** The registry generator must derive stage names, stage descriptions, bound roles, bound skills, human-gate flags, document types, and prerequisites from `.kbz/stage-bindings.yaml`.
- **REQ-002:** The registry generator must derive role identity and inheritance metadata from `.kbz/roles/*.yaml`.
- **REQ-003:** The generator must update only explicitly marked generated regions in `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md`.
- **REQ-004:** Each generated region must include marker comments that identify the region as generated, name the source inputs, and instruct maintainers not to hand-edit inside the region.
- **REQ-005:** Sync mode must write current generated content into all in-scope generated regions while preserving all content outside the marker comments byte-for-byte.
- **REQ-006:** Check mode must compare current generated content with on-disk generated regions without writing files and must return a non-zero result when any generated region is stale.
- **REQ-007:** The repository must expose a `make registry-check` target or equivalent CI-callable target that runs check mode.
- **REQ-008:** CI must run the registry check so a pull request that changes canonical registry inputs without updating generated regions fails before merge.
- **REQ-009:** The generated content must include enough information for a reader to locate the canonical source files for any role, skill, or stage binding listed.
- **REQ-010:** The generator must fail with a clear error if a generated region marker is missing, duplicated, or nested incorrectly in an in-scope file.
- **REQ-011:** The generator must leave `AGENTS.md` narrative content hand-authored in this round and must not rewrite it.

## Non-Functional Requirements

- **REQ-NF-001:** Running sync mode twice on an unchanged repository must produce no diff on the second run.
- **REQ-NF-002:** Check mode must complete in under 2 seconds on the repository's current role and skill corpus in local execution.
- **REQ-NF-003:** Generated registry tables must be deterministic: input files in the same state must produce identical output ordering and formatting across runs.
- **REQ-NF-004:** Generated content must not duplicate long skill procedures, examples, or anti-pattern bodies; it may list names, paths, short descriptions, and stage bindings.
- **REQ-NF-005:** Generator errors must name the file and marker or source input that caused the failure.

## Constraints

- `.kbz/stage-bindings.yaml` and `.kbz/roles/*.yaml` are the canonical registry sources for this feature.
- The generator must preserve hand-authored prose outside generated regions.
- `AGENTS.md` is intentionally deferred. This feature must not make it generated by accident.
- The registry generator must not infer stage bindings by scanning prose in `CLAUDE.md`, `.github/copilot-instructions.md`, `README.md`, or SKILL.md files.
- Generated regions must be suitable for code review: diffs should show changed rows or text, not wholesale rewrites caused by nondeterministic ordering.
- B61 cleanup may reduce corpus size, but this feature must remain correct whether that cleanup lands before or after the generator.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given current stage bindings and role YAML files, when the generator extracts registry data, then every configured stage and bound role appears in the extracted model with source paths.
- **AC-002 (REQ-003, REQ-004):** Given each in-scope output file, when generated regions are inspected, then marker comments identify the generated region and canonical source inputs.
- **AC-003 (REQ-005):** Given hand-authored text before and after a generated region, when sync mode runs, then that surrounding text is preserved byte-for-byte.
- **AC-004 (REQ-006):** Given a stale generated region, when check mode runs, then it exits non-zero and reports the stale file.
- **AC-005 (REQ-007, REQ-008):** Given the repository CI configuration, when CI runs, then the registry-check target executes and fails on stale generated regions.
- **AC-006 (REQ-009):** Given a role, skill, or stage listed in a generated table, when a reader follows its path or source reference, then the canonical file exists in the repository.
- **AC-007 (REQ-010):** Given a missing, duplicated, or nested marker in an in-scope file, when sync or check mode runs, then it fails with the affected file and marker name.
- **AC-008 (REQ-011):** Given `AGENTS.md`, when sync mode runs, then `AGENTS.md` is not modified.
- **AC-009 (REQ-NF-001):** Given sync mode is run twice without input changes, when `git diff` is checked after the second run, then no generated-region diff exists.
- **AC-010 (REQ-NF-002):** Given local repository inputs, when check mode runs under timing instrumentation, then it completes under 2 seconds.
- **AC-011 (REQ-NF-004):** Given generated registry content, when it is inspected, then it does not include full skill procedures, long examples, or anti-pattern bodies.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Unit test | Load fixture and repository stage bindings plus roles and assert the extracted model contains all expected stages, roles, and paths. |
| AC-002 | Inspection | Inspect `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md` for generated-region markers and source-input notes. |
| AC-003 | Unit test | Run sync mode against a fixture file with surrounding prose and assert bytes outside markers are unchanged. |
| AC-004 | Integration test | Introduce a stale generated fixture and assert check mode exits non-zero with the file path. |
| AC-005 | CI/config test | Run or inspect the CI target and assert `make registry-check` or equivalent is invoked. |
| AC-006 | Inspection | Validate that each generated path points to an existing canonical source file. |
| AC-007 | Unit test | Exercise missing, duplicated, and nested marker fixtures and assert clear errors. |
| AC-008 | Regression test | Run sync mode and assert `AGENTS.md` is unchanged. |
| AC-009 | Integration test | Run sync mode twice and assert the second run produces no diff. |
| AC-010 | Test | Time check mode on repository inputs and assert it completes under 2 seconds. |
| AC-011 | Inspection | Review generated content for absence of full procedures, long examples, and anti-pattern bodies. |
