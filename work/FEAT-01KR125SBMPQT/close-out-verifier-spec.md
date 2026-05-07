| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | approved |
| Feature| FEAT-01KR125SBMPQT              |
| Design | P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene |

# Specification: Close-Out Verifier Sub-Agent

## Overview

This feature creates a `verifier` role and `verify-closeout` skill that enforces the Definition of Done at the `verifying` stage. The verifier is a clean-context sub-agent with a strict 10-item checklist. The orchestrator dispatches it, reads the pass/fail report, and acts — it never self-verifies. This applies to both full-procedure and fast-track features.

### Problem

The orchestrator is the worst agent to verify close-out. By end of cycle its context is saturated, and close-out steps (merge verification, branch deletion, knowledge curation) are exactly what it forgets. The Definition of Done lists ten conditions — relying on the orchestrator to verify them from memory is the root cause of forgotten close-out steps.

### Design References

This specification implements Decision 7 and Component 7 from the approved design:

- **Decision 7:** Delegate close-out verification to a clean-context sub-agent
- **Component 7:** Close-Out Verification Sub-Agent

### Related Specifications

- **FEAT-01KR12539CXH6 (Orchestrator role hardening)** — Prevents the orchestrator from investigating code; verification delegation is the positive counterpart.
- **FEAT-01KR125SBMBCT (Constraint pinning)** — Keeps the orchestrator aware of its role; verification delegation removes the need for the orchestrator to remember close-out steps.
- **FEAT-01KR125SBM4JQ (Fast-track review dispatch)** — Ensures review happens; the verifier checks that it did.

## Scope

### In Scope

- Creating a new role file: `.kbz/roles/verifier.yaml`
- Creating a new skill file: `.kbz/skills/verify-closeout/SKILL.md`
- The role defines: identity, vocabulary, anti-patterns, and tool list for the verifier sub-agent
- The skill defines: a 10-item checklist mapped to the Definition of Done, with concrete verification actions per item, and a structured pass/fail output format
- The verifier operates at the `verifying` stage, dispatched by the orchestrator via `spawn_agent`

### Out of Scope

- Go code changes — this feature creates YAML and Markdown files only
- Modifying the `orchestrate-development` or `orchestrate-review` skills to invoke the verifier (that integration is FEAT-01KR125SBM4FN)
- Runtime enforcement of verification — the verifier is a procedural tool dispatched by the orchestrator
- Modifying the stage bindings to add a `verifying` stage binding for the verifier role (the verifying stage already exists; the binding update is FEAT-01KR125SBM4FN)

## Functional Requirements

### REQ-001: Verifier Role File

A new file `.kbz/roles/verifier.yaml` SHALL be created with the following content:

- **id:** `verifier`
- **inherits:** `base`
- **identity:** `"Methodical close-out auditor"`
- **vocabulary:** checklist-driven, evidence-backed, binary verdict (pass/fail per item), no judgment calls, no conversation
- **anti_patterns:** at minimum three entries:
  1. Accepting orchestrator assurances without independent verification
  2. Skipping checklist items (every item must be checked, every check must have evidence)
  3. Assumption without evidence (no pass verdict without a concrete verification action)
- **tools:** `entity`, `status`, `doc`, `knowledge`, `read_file`, `grep`

### REQ-002: Verifier Tool List

The verifier's tool list SHALL include exactly: `entity`, `status`, `doc`, `knowledge`, `read_file`, `grep`. It SHALL NOT include `write_file`, `kanbanzai_edit_file`, `terminal`, `search_graph`, `spawn_agent`, or any implementation tools. The verifier checks, it does not create or modify.

### REQ-003: Verify-Closeout Skill File

A new skill file `.kbz/skills/verify-closeout/SKILL.md` SHALL be created with the standard Kanbanzai skill format (frontmatter with name, description, triggers, roles, stage, constraint_level).

### REQ-004: Skill Metadata

The skill metadata SHALL specify:
- **name:** `verify-closeout`
- **triggers:** at minimum "verify close-out", "check definition of done", "audit feature completion"
- **roles:** `[verifier]`
- **stage:** `verifying`

### REQ-005: Ten-Item Checklist

The skill SHALL contain a numbered 10-item checklist, one item per Definition of Done condition. Each item SHALL include:

1. The DoD condition description
2. A concrete verification action (e.g., "run `git status --porcelain` and confirm no output")
3. The evidence to record (e.g., "command output or exit code")
4. The pass/fail criterion

The ten items SHALL map to:
1. All tasks terminal
2. All changes committed (`git status --porcelain`)
3. Temporary files removed (check for untracked files outside tracked paths)
4. Tests pass (`go test ./...` on main)
5. Code reviewed (review document registered, no blocking findings)
6. Full lifecycle advanced (feature status is `verifying`)
7. Merge ancestry verified (`git merge-base --is-ancestor`)
8. Branch deleted and verified absent (`git branch | grep`)
9. Worktrees removed (`git worktree list`)
10. Knowledge curated and entities closed

### REQ-006: Output Format

The skill SHALL specify a structured output format: a list of 10 items, each with `item`, `status` (`pass` or `fail`), and `evidence`. A summary at the top SHALL state the aggregate verdict: `all-pass` or `failures-found` with the count of failed items.

### REQ-007: No Conversational Interface

The skill SHALL instruct the verifier to produce the structured output directly — no conversation, no questions, no requests for clarification. If a check cannot be performed (e.g., missing tool), the item is marked `fail` with the reason. The verifier does not ask the orchestrator for help.

### REQ-008: Skill Anti-Patterns

The skill SHALL include at minimum two anti-patterns:
1. Trusting orchestrator assurances (the verifier checks independently)
2. Partial checklist execution (all ten items must be checked)

### REQ-009: YAML Validity

`verifier.yaml` SHALL be valid YAML. It SHALL follow the structure and conventions of existing role files (`orchestrator.yaml`, `reviewer.yaml`, `implementer.yaml`).

### REQ-010: Markdown Validity

`verify-closeout/SKILL.md` SHALL be valid Markdown. It SHALL follow the structure and conventions of existing skill files (`review-code/SKILL.md`, `implement-task/SKILL.md`).

## Non-Functional Requirements

### NFR-001: Format Consistency

Both files SHALL follow the exact conventions of existing role and skill files: YAML structure, indentation, markdown heading levels, frontmatter format, vocabulary list format, anti-pattern list format, and checklist format.

### NFR-002: Checklist Completeness

Every Definition of Done item from the design SHALL have a corresponding checklist item in the skill. No DoD item SHALL be omitted. The checklist SHALL NOT include items beyond the ten DoD conditions.

### NFR-003: Self-Contained Skill

The skill SHALL be self-contained — the verifier sub-agent SHALL be able to execute the checklist without reading any other skill or role file. All required instructions, verification actions, and output formats are within the skill itself.

## Acceptance Criteria

- [ ] **AC-001:** `.kbz/roles/verifier.yaml` exists and is valid YAML
- [ ] **AC-002:** Verifier identity is "Methodical close-out auditor"
- [ ] **AC-003:** Verifier tools are exactly: `entity`, `status`, `doc`, `knowledge`, `read_file`, `grep`
- [ ] **AC-004:** Verifier has at minimum three anti-patterns matching REQ-001
- [ ] **AC-005:** `.kbz/skills/verify-closeout/SKILL.md` exists and is valid Markdown
- [ ] **AC-006:** Skill metadata specifies name `verify-closeout`, roles `[verifier]`, stage `verifying`
- [ ] **AC-007:** Skill contains a 10-item checklist mapping to all DoD conditions
- [ ] **AC-008:** Each checklist item has a concrete verification action, evidence field, and pass/fail criterion
- [ ] **AC-009:** Skill specifies structured output format with per-item pass/fail and evidence
- [ ] **AC-010:** Skill specifies no conversational interface — output only
- [ ] **AC-011:** Skill has at minimum two anti-patterns
- [ ] **AC-012:** Verifier role file follows conventions of existing role files
- [ ] **AC-013:** Skill file follows conventions of existing skill files
- [ ] **AC-014:** No DoD item is omitted from the checklist

## Verification Plan

| Requirement | Verification Method | Acceptance Criterion |
|-------------|-------------------|---------------------|
| REQ-001 | Manual inspection of `.kbz/roles/verifier.yaml` | AC-001, AC-002, AC-012 |
| REQ-002 | Verify tool list in verifier.yaml | AC-003 |
| REQ-001 | Count anti-patterns in verifier.yaml | AC-004 |
| REQ-003 | Manual inspection of skill file | AC-005, AC-013 |
| REQ-004 | Verify frontmatter metadata | AC-006 |
| REQ-005 | Count checklist items, verify each has action + evidence + criterion | AC-007, AC-008, AC-014 |
| REQ-006 | Verify output format section | AC-009 |
| REQ-007 | Verify no-conversation instruction | AC-010 |
| REQ-008 | Count anti-patterns in skill | AC-011 |
| REQ-009 | `yq eval` on verifier.yaml | AC-001 |
| REQ-010 | Markdown linter on skill file | AC-005 |
