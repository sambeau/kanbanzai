| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | approved |
| Feature| FEAT-01KR12539CXH6              |
| Design | P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene |

# Specification: Orchestrator Role Hardening

## Overview

This feature hardens the Kanbanzai orchestrator role against context pollution by adding an explicit anti-pattern, removing code-investigation tools, and introducing a hard constraint. These changes prevent the orchestrator from reading implementation source code before delegating to sub-agents — the primary vector for context rot identified in P41 research and the P50 incident.

### Problem

The orchestrator retains investigation tools (`read_file`, `grep`, `search_graph`) and uses them to pre-understand code before delegating. Each investigation loads implementation details that compete with orchestration constraints and role identity. Over 8–12 tasks, accumulated code fragments cause goal drift: forgotten close-out steps, skipped reviews, and degraded decisions.

### Design References

This specification implements Decisions 1–3 and Components 1–3 from the approved design `P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene`:

- **Decision 1:** Add "Pre-delegation Code Investigation" anti-pattern
- **Decision 2:** Remove `grep` and `search_graph` from orchestrator tools
- **Decision 3:** Add hard constraint (ℋ) to `orchestrate-development` Phase 1

The feature does NOT cover constraint pinning (Component 4, FEAT-01KR125SBMBCT), fast-track review dispatch (Component 5, FEAT-01KR125SBM4JQ), close-out verification (Component 7, FEAT-01KR125SBMPQT), or documentation updates (Component 6, FEAT-01KR125SBM4FN).

### Related Specifications

- **P52-fast-track-orchestration** — The fast-track behavioural profile that this feature's hard constraint extends.
- **orchestrate-development** skill — Modified by this feature (Phase 1 hard constraint).
- **orchestrator** role — Modified by this feature (anti-pattern, tool list).

No conflicting requirements were found in related specifications.

## Scope

### In Scope

- Adding the "Pre-delegation Code Investigation" anti-pattern to `orchestrator.yaml`
- Adding the same anti-pattern to the anti-patterns section of `orchestrate-development/SKILL.md`
- Removing `grep` from the orchestrator role's tool list in `orchestrator.yaml`
- Removing `search_graph` from the orchestrator role's tool list in `orchestrator.yaml`
- Adding a hard constraint (ℋ) to Phase 1 of `orchestrate-development/SKILL.md`
- Verifying that `read_file` remains in the orchestrator tool list

### Out of Scope

- Constraint pinning in `next`/`handoff` responses (FEAT-01KR125SBMBCT)
- Fast-track review dispatch (FEAT-01KR125SBM4JQ)
- Close-out verifier role and skill (FEAT-01KR125SBMPQT)
- Documentation and skill file updates (FEAT-01KR125SBM4FN)
- Restricting `read_file` to document-only paths (deferred per design Open Question 1)
- Any Go code changes — this feature modifies YAML roles and Markdown skills only
- Runtime enforcement of the anti-pattern — this is a procedural constraint, not a tool-level block

## Functional Requirements

### REQ-001: Anti-Pattern Definition

The orchestrator role file (`orchestrator.yaml`) SHALL include a new anti-pattern named "Pre-delegation Code Investigation" with the following structure:

- **name:** `"Pre-delegation Code Investigation"`
- **detect:** Describes the behaviour: orchestrator reads source files, traces call paths, or searches the code graph to understand implementation details before delegating
- **because:** Explains that implementation understanding belongs to the sub-agent, and every code fragment loaded competes with orchestration constraints, causing context rot and forgotten close-out steps
- **resolve:** States: delegate immediately via `handoff`, trust the pipeline, and if the dev-plan is unclear fix the dev-plan rather than reading code

### REQ-002: Anti-Pattern Placement

The new anti-pattern SHALL be inserted into the `anti_patterns` list in `orchestrator.yaml` in alphabetical order by name, consistent with existing anti-patterns.

### REQ-003: Anti-Pattern in Skill

The same anti-pattern SHALL be added to the anti-patterns section of `orchestrate-development/SKILL.md`, following the existing format and structure of that section. The anti-pattern SHALL be placed after the "Manual Prompt Composition" anti-pattern (the last existing entry) to maintain logical grouping.

### REQ-004: Tool Removal — grep

The `grep` entry SHALL be removed from the `tools` list in `orchestrator.yaml`.

### REQ-005: Tool Removal — search_graph

The `search_graph` entry SHALL be removed from the `tools` list in `orchestrator.yaml`.

### REQ-006: Tool Retention — read_file

The `read_file` entry SHALL remain in the `tools` list in `orchestrator.yaml`. It MUST NOT be removed.

### REQ-007: Tool List Ordering

The remaining tools in `orchestrator.yaml` SHALL maintain their existing order after `grep` and `search_graph` are removed. No reordering beyond the removal of these two entries.

### REQ-008: Hard Constraint Definition

Phase 1 of `orchestrate-development/SKILL.md` SHALL include a new hard constraint with the label "Constraint ℋ — No Code Investigation" that:

- States the orchestrator MUST NOT read source files, trace call paths, or search the code graph to understand implementation areas before dispatching
- States the sub-agent receives sufficient context via `handoff` (dev-plan, spec sections, knowledge entries, file paths)
- States every line of source code read competes with orchestration constraints and accelerates context rot
- States if the dev-plan is unclear about what to build, the orchestrator SHALL flag it rather than reading code to compensate
- Is marked as a **hard constraint (ℋ)** — non-negotiable, violation blocks stage advance

### REQ-009: Hard Constraint Placement

The hard constraint SHALL be placed at the start of Phase 1 (Read the Dev-Plan) in `orchestrate-development/SKILL.md`, before the existing step-by-step instructions. It SHALL be visually distinct from procedural steps — using a blockquote or bold formatting consistent with existing constraint formatting in the skill file.

### REQ-010: YAML Validity

After all modifications, `orchestrator.yaml` SHALL remain valid YAML. The file SHALL parse without errors.

### REQ-011: Markdown Validity

After all modifications, `orchestrate-development/SKILL.md` SHALL remain valid Markdown with no broken syntax, links, or formatting.

## Non-Functional Requirements

### NFR-001: Format Consistency

The new anti-pattern in both files SHALL follow the exact YAML structure and indentation conventions of existing anti-patterns in the respective files. No deviation in field naming, whitespace, or commenting style.

### NFR-002: No Semantic Change to Existing Content

Modifications SHALL be additive (adding anti-pattern, adding constraint) or subtractive (removing two tool entries). No existing anti-pattern, vocabulary entry, checklist item, or procedure step SHALL be modified in wording, ordering, or semantics beyond the specified additions and removals.

### NFR-003: Diff Minimisation

The diff for each file SHALL be minimal — only the lines added or removed per the requirements. No unrelated whitespace changes, reformatting, or reordering.

### NFR-004: Immutable Tool Behaviour

Removing `grep` and `search_graph` from the orchestrator role SHALL NOT affect the tool availability for any other role (implementer, reviewer, verifier, architect, spec-author, etc.). Each role's tool list is independent.

## Acceptance Criteria

- [ ] **AC-001:** `orchestrator.yaml` contains an anti-pattern named "Pre-delegation Code Investigation" with detect, because, and resolve fields whose content matches the design specification (Component 1)
- [ ] **AC-002:** The new anti-pattern is placed in alphabetical order within the `anti_patterns` list of `orchestrator.yaml`
- [ ] **AC-003:** `orchestrate-development/SKILL.md` contains the same "Pre-delegation Code Investigation" anti-pattern in its anti-patterns section
- [ ] **AC-004:** `grep` is not present in the `tools` list of `orchestrator.yaml`
- [ ] **AC-005:** `search_graph` is not present in the `tools` list of `orchestrator.yaml`
- [ ] **AC-006:** `read_file` is present in the `tools` list of `orchestrator.yaml`
- [ ] **AC-007:** Phase 1 of `orchestrate-development/SKILL.md` contains a hard constraint labeled "Constraint ℋ — No Code Investigation" with content matching the design specification (Component 3)
- [ ] **AC-008:** The hard constraint is placed at the start of Phase 1, before the existing numbered steps
- [ ] **AC-009:** `orchestrator.yaml` parses as valid YAML with no syntax errors
- [ ] **AC-010:** `orchestrate-development/SKILL.md` is valid Markdown with no broken links or syntax
- [ ] **AC-011:** No existing anti-patterns, vocabulary entries, checklist items, or procedure steps in either file have been modified beyond the specified additions and removals
- [ ] **AC-012:** Tools `grep` and `search_graph` remain present in the implementer, reviewer, and architect role files

## Verification Plan

| Requirement | Verification Method | Acceptance Criterion |
|-------------|-------------------|---------------------|
| REQ-001 | Manual inspection of `orchestrator.yaml` anti-pattern section | AC-001 |
| REQ-002 | Verify alphabetical ordering by name in `anti_patterns` list | AC-002 |
| REQ-003 | Manual inspection of `orchestrate-development/SKILL.md` anti-patterns section | AC-003 |
| REQ-004 | `grep` for "grep" in `orchestrator.yaml` tools list returns no match | AC-004 |
| REQ-005 | `grep` for "search_graph" in `orchestrator.yaml` tools list returns no match | AC-005 |
| REQ-006 | `grep` for "read_file" in `orchestrator.yaml` tools list returns a match | AC-006 |
| REQ-007 | Manual diff inspection confirms only the two tool entries removed, no reordering | AC-004, AC-005 |
| REQ-008 | Manual inspection of Phase 1 in `orchestrate-development/SKILL.md` | AC-007 |
| REQ-009 | Verify constraint appears before first numbered step in Phase 1 | AC-008 |
| REQ-010 | `yq eval` or equivalent YAML parser on `orchestrator.yaml` exits zero | AC-009 |
| REQ-011 | Markdown linter or render check on `orchestrate-development/SKILL.md` | AC-010 |
| NFR-002 | Diff against original files confirms no unexpected changes | AC-011 |
| NFR-004 | `grep` for "grep" and "search_graph" in implementer.yaml, reviewer.yaml, architect.yaml confirms presence | AC-012 |
