| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | approved |
| Feature| FEAT-01KR125SBM4FN              |
| Design | P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene |

# Specification: Documentation and Fast-Track Integration

## Overview

This feature updates documentation and skill files to reflect the orchestrator context hygiene changes, and integrates the new constraints into the fast-track profile. It is the final integration feature — it ensures the other four features' changes are coherently reflected in the project's guidance files.

### Problem

Three integration gaps remain after the other features are implemented:

1. The `.github/skills/codebase-memory-*/SKILL.md` files instruct all agents to use graph tools (`search_graph`, `query_graph`, `trace_call_path`). If the orchestrator role forbids these tools, the skill files need to note that graph tools are for implementer and reviewer roles, not orchestrator.
2. The fast-track profile should load a lightweight orchestrator role that excludes `grep` and `search_graph` (per Component 6 of the design).
3. The `verifying` stage binding may need to reference the new `verifier` role and `verify-closeout` skill.

### Design References

This specification implements Component 6 and the remaining integration concerns from the approved design:

- **Component 6:** Fast-Track Integration — fast-track sessions load lightweight orchestrator role
- **Open Question 3:** codebase-memory-mcp skill interaction — skill files contextualised per-role
- **Component 7:** Pipeline integration — verifying stage dispatches verifier

### Related Specifications

- **FEAT-01KR12539CXH6 (Orchestrator role hardening)** — Removes grep/search_graph from orchestrator; documentation must reflect this
- **FEAT-01KR125SBMBCT (Constraint pinning)** — Adds role reminder; fast-track must include it
- **FEAT-01KR125SBMPQT (Close-out verifier)** — Creates verifier role/skill; stage bindings may need updating

## Scope

### In Scope

- Updating `.github/skills/codebase-memory-exploring/SKILL.md` to note graph tools are for implementer and reviewer roles
- Updating `.github/skills/codebase-memory-tracing/SKILL.md` with the same note
- Updating `.github/skills/codebase-memory-quality/SKILL.md` with the same note
- Updating `.github/skills/codebase-memory-reference/SKILL.md` with the same note
- Ensuring the fast-track profile section of `orchestrate-development/SKILL.md` references the lightweight orchestrator role (without grep/search_graph)
- Updating `.kbz/stage-bindings.yaml` if needed to reference the verifier role at the `verifying` stage
- Verifying cross-references between all five features are consistent

### Out of Scope

- Creating new codebase-memory-mcp skill files
- Modifying the `orchestrate-development` full procedure (only fast-track section)
- Any Go code changes
- Modifying skill files not listed in In Scope

## Functional Requirements

### REQ-001: Codebase-Memory Skills — Exploring

`.github/skills/codebase-memory-exploring/SKILL.md` SHALL include a note that graph tools (`search_graph`, `query_graph`, `get_code_snippet`, `trace_call_path`) are available to implementer and reviewer roles. The note SHALL state that the orchestrator role does not use graph tools — it delegates code understanding to sub-agents.

### REQ-002: Codebase-Memory Skills — Tracing

`.github/skills/codebase-memory-tracing/SKILL.md` SHALL include the same note as REQ-001, adapted to the tracing skill's context.

### REQ-003: Codebase-Memory Skills — Quality

`.github/skills/codebase-memory-quality/SKILL.md` SHALL include the same note as REQ-001, adapted to the quality skill's context.

### REQ-004: Codebase-Memory Skills — Reference

`.github/skills/codebase-memory-reference/SKILL.md` SHALL include the same note as REQ-001, adapted to the reference skill's context.

### REQ-005: Role-Scoped Tool Guidance

The notes added in REQ-001 through REQ-004 SHALL be consistent in wording across all four files. The exact text may vary to fit each skill's context, but the core message — graph tools are for implementer/reviewer, not orchestrator — SHALL be preserved.

### REQ-006: Fast-Track Orchestrator Role Reference

The fast-track profile section of `orchestrate-development/SKILL.md` SHALL reference that fast-track sessions use the orchestrator role with `grep` and `search_graph` removed (per F1's tool restriction).

### REQ-007: Stage Bindings — Verifying Stage

`.kbz/stage-bindings.yaml` SHALL be reviewed for the `verifying` stage. If the verifying stage binding does not reference the `verifier` role and `verify-closeout` skill, it SHALL be updated to include them.

### REQ-008: Cross-Reference Consistency

The specification documents for all five features SHALL be reviewed for consistent cross-references. Any feature's "Related Specifications" section that references another feature by the wrong ID or describes a dependency incorrectly SHALL be corrected.

### REQ-009: No Existing Content Regression

Existing content in modified files SHALL NOT be changed beyond the specified additions. No existing instructions, examples, or formatting SHALL be altered.

## Non-Functional Requirements

### NFR-001: Minimal Diff

Each file change SHALL produce the smallest possible diff — only the added note or reference, no unrelated whitespace or formatting changes.

### NFR-002: Consistent Tone

Added notes SHALL match the existing tone and style of each skill file. No change in authorial voice.

## Acceptance Criteria

- [ ] **AC-001:** `.github/skills/codebase-memory-exploring/SKILL.md` contains a role-scoped tool note
- [ ] **AC-002:** `.github/skills/codebase-memory-tracing/SKILL.md` contains a role-scoped tool note
- [ ] **AC-003:** `.github/skills/codebase-memory-quality/SKILL.md` contains a role-scoped tool note
- [ ] **AC-004:** `.github/skills/codebase-memory-reference/SKILL.md` contains a role-scoped tool note
- [ ] **AC-005:** All four notes convey the same core message: graph tools for implementer/reviewer, not orchestrator
- [ ] **AC-006:** Fast-track profile section of `orchestrate-development/SKILL.md` references the lightweight orchestrator role
- [ ] **AC-007:** `.kbz/stage-bindings.yaml` verifying stage references verifier role and verify-closeout skill (if applicable)
- [ ] **AC-008:** Cross-references between all five feature specs are consistent
- [ ] **AC-009:** No existing content in any modified file is altered beyond the specified additions
- [ ] **AC-010:** All modified Markdown files are valid with no broken links or syntax errors

## Verification Plan

| Requirement | Verification Method | Acceptance Criterion |
|-------------|-------------------|---------------------|
| REQ-001–004 | Manual inspection of each codebase-memory skill file | AC-001–004 |
| REQ-005 | Compare note text across four files for consistency | AC-005 |
| REQ-006 | Manual inspection of fast-track profile section | AC-006 |
| REQ-007 | Manual inspection of stage-bindings.yaml verifying stage | AC-007 |
| REQ-008 | Review all five spec "Related Specifications" sections | AC-008 |
| REQ-009 | Diff each file against original, confirm only additions | AC-009 |
| NFR-001 | Verify diff size proportional to change scope | AC-009 |
| NFR-002 | Manual review of tone consistency | AC-001–004 |
