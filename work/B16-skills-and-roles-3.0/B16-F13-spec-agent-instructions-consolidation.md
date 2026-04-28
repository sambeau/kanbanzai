# Specification: Agent Instructions Consolidation

| Field | Value |
|-------|-------|
| **Feature** | FEAT-01KN6-ZA4NYJ25 (agent-instructions-consolidation) |
| **Design** | `work/design/agent-instructions-consolidation.md` |
| **Review Report** | `work/reports/agent-instructions-review.md` |
| **Status** | Draft |

## Overview

This specification defines the requirements for consolidating all agent-facing instruction files in the Kanbanzai project. The work addresses 19 recommendations (R1–R19) from the agent instructions review report, organised into five areas: (1) migrating unique content from three overlapping `.agents/skills/` files into their `.kbz/skills/` counterparts and retiring the old files, (2) upgrading five retained system skills to match `CONVENTIONS.md` structural quality, (3) fixing `.kbz/` store-commit discipline, (4) cleaning up `AGENTS.md` and fixing discovery paths, and (5) verifying context assembly ordering and effort budget inclusion.

The feature is entirely content-focused — no new Go code, MCP tools, or infrastructure is created. The deliverables are edited skill files, edited instruction files, one new bootstrap file (`CLAUDE.md`), and at most minor fixes to `internal/context/assemble.go` if verification reveals ordering issues.

## Scope

### In Scope

- Migration of unique content from `kanbanzai-code-review` into `review-code` and `orchestrate-review`
- Migration of unique content from `kanbanzai-plan-review` into `review-plan`
- Migration of unique content from `kanbanzai-design` (and its `references/`) into `write-design`
- Deletion of the three retired `.agents/skills/` directories after migration
- Structural upgrade of five retained system skills: `kanbanzai-getting-started`, `kanbanzai-workflow`, `kanbanzai-agents`, `kanbanzai-documents`, `kanbanzai-planning`
- Store-commit discipline additions to `kanbanzai-agents`, `kanbanzai-getting-started`, `AGENTS.md`, and `.kbz/roles/base.yaml`
- Incremental cleanup of `AGENTS.md` (deduplicate, remove stale content, add pointers)
- Updates to `refs/document-map.md` to fix stale skill pointers
- Discovery fix in `kanbanzai-getting-started` for stage bindings and `.kbz/skills/`
- Creation of `CLAUDE.md` bootstrap file
- Verification (and fix if needed) of context assembly ordering in `internal/context/assemble.go`
- Verification (and fix if needed) of effort budget inclusion in `handoff` context packets

### Explicitly Excluded

- Full evaluation harness with automated test scenarios (R17 — needs its own design)
- Cross-platform bootstrap files for Cursor, Windsurf, Cline, Aider, etc. (only `CLAUDE.md` is in scope)
- Changes to MCP tool descriptions (review report found them adequate)
- Changes to role YAML files beyond adding the "Store Neglect" anti-pattern to `base.yaml`
- Changes to `.kbz/skills/CONVENTIONS.md`
- Changes to `.kbz/stage-bindings.yaml`
- Changes to `.github/copilot-instructions.md` (already adequate per review report)
- New Go packages, new MCP tools, or new CLI commands

---

## Functional Requirements

### Area 1: Migrate and Retire Overlapping Skills

#### FR-001: Migrate Code Review Edge Case Playbooks

The `review-code` skill MUST include nuanced handling for four edge cases currently documented only in `kanbanzai-code-review`: (a) missing specification, (b) partial implementation, (c) ambiguous conformance, and (d) missing context. Each edge case MUST provide multi-step procedural guidance — not just a terse STOP instruction.

**Acceptance criteria:**

- [ ] `review-code` procedure contains a dedicated subsection or step for each of the four edge cases: missing spec, partial implementation, ambiguous conformance, missing context
- [ ] Each edge case provides at least two concrete procedural steps (not a single "stop and ask" instruction)
- [ ] The partial implementation case instructs the reviewer to set spec_conformance to `concern` and continue other dimensions
- [ ] The ambiguous conformance case provides classification guidance for when an implementation intentionally differs from the spec
- [ ] `review-code` remains under 500 lines after the additions (use `references/` if needed)

#### FR-002: Migrate Remediation Phase to orchestrate-review

The `orchestrate-review` skill MUST include a remediation phase covering: task creation for blocking findings, conflict-check before parallel remediation, re-review scoping (full vs. targeted), and an escalation cycle with iteration cap.

**Acceptance criteria:**

- [ ] `orchestrate-review` procedure contains remediation steps (task creation, conflict-check, re-review scoping, escalation)
- [ ] The procedure specifies an iteration cap for the remediation-re-review cycle
- [ ] The procedure references the `conflict` tool for checking file overlap before parallel remediation
- [ ] `orchestrate-review` remains under 500 lines after the additions (use `references/` if needed)

#### FR-003: Migrate Review Document Creation to orchestrate-review

The `orchestrate-review` skill MUST include a step for writing a persistent review artifact. The step MUST specify the naming convention (`review-{id}-{slug}.md`) and require registration via `doc(action: "register")`.

**Acceptance criteria:**

- [ ] `orchestrate-review` procedure contains a step for writing the review document to `work/reviews/`
- [ ] The naming convention `review-{id}-{slug}.md` is specified
- [ ] The step requires calling `doc(action: "register")` after writing

#### FR-004: Migrate Human Checkpoint Integration to orchestrate-review

The `orchestrate-review` skill MUST document the three trigger scenarios for creating a human checkpoint during review: (a) ambiguous findings where reviewer confidence is low, (b) high-stakes features where the blast radius is large, and (c) disagreement between review dimensions.

**Acceptance criteria:**

- [ ] `orchestrate-review` contains guidance for when to create a human checkpoint
- [ ] All three trigger scenarios are documented: ambiguous findings, high-stakes features, dimension disagreement
- [ ] Each scenario includes what information to present in the checkpoint context

#### FR-005: Migrate Context Budget Strategy

The `orchestrate-review` skill MUST include guidance on context budget allocation: orchestrator target range (~6–14 KB) and sub-agent target range (~12–30 KB). This MAY be inline or in a reference file.

**Acceptance criteria:**

- [ ] Context budget guidance is present in `orchestrate-review` (inline or reference file)
- [ ] Both the orchestrator and sub-agent target ranges are specified

#### FR-006: Verify Per-Dimension Evaluation Questions

Before deleting `kanbanzai-code-review`, the implementer MUST verify that the per-dimension evaluation questions (5 dimensions × 5–7 questions) are covered by the `reviewer-*.yaml` role files. Any dimension lacking evaluation questions MUST have them ported from the old skill into the corresponding role file.

**Acceptance criteria:**

- [ ] Each of the 5 review dimensions (spec conformance, implementation quality, test adequacy, documentation currency, workflow integrity) has evaluation questions in the corresponding `reviewer-*.yaml` role file
- [ ] If any questions were ported, the role file remains well-formed YAML with correct indentation

#### FR-007: Retire kanbanzai-code-review

After FR-001 through FR-006 are satisfied, the `.agents/skills/kanbanzai-code-review/` directory MUST be deleted.

**Acceptance criteria:**

- [ ] `.agents/skills/kanbanzai-code-review/` no longer exists on disk
- [ ] No remaining file in the repository references `kanbanzai-code-review` as a current skill (stale references in review reports or research documents are acceptable — only active instruction files and routing files must be updated)

#### FR-008: Migrate Criterion-by-Criterion Spec Conformance to review-plan

The `review-plan` skill MUST include a step that reads each feature's acceptance criteria from its specification and verifies them against the implementation code — not just checking document approval status.

**Acceptance criteria:**

- [ ] `review-plan` procedure contains a step for reading acceptance criteria from each feature's spec
- [ ] The step instructs the reviewer to verify each criterion against the actual implementation, not just check approval status
- [ ] The output format includes a per-feature, per-criterion conformance table

#### FR-009: Migrate Cross-Cutting Checks to review-plan

The `review-plan` skill MUST include a step for concrete cross-cutting verification: running `go test -race ./...`, calling `health()`, and checking `git status` for uncommitted changes.

**Acceptance criteria:**

- [ ] `review-plan` procedure contains a step with the three cross-cutting checks
- [ ] Each check specifies the exact command or tool call

#### FR-010: Migrate Retrospective and Document Registration to review-plan

The `review-plan` skill MUST include steps for: (a) contributing a retrospective signal via `finish()` at the end of the review, and (b) writing review findings to `work/reviews/` and registering with `doc(action: "register")`.

**Acceptance criteria:**

- [ ] `review-plan` procedure contains a retrospective contribution step
- [ ] `review-plan` procedure contains a document registration step specifying the output path and `doc(action: "register")` call

#### FR-011: Migrate Inputs/Prerequisites to review-plan

The `review-plan` skill MUST document its input prerequisites: what entities, documents, and project state must exist before the review can proceed.

**Acceptance criteria:**

- [ ] `review-plan` contains a prerequisites or inputs section listing what must be true before the review starts

#### FR-012: Retire kanbanzai-plan-review

After FR-008 through FR-011 are satisfied, the `.agents/skills/kanbanzai-plan-review/` directory MUST be deleted.

**Acceptance criteria:**

- [ ] `.agents/skills/kanbanzai-plan-review/` no longer exists on disk
- [ ] No remaining active instruction or routing file references `kanbanzai-plan-review` as a current skill

#### FR-013: Migrate Design Stance and Philosophy to write-design

The `write-design` skill MUST include: (a) a "Design with Ambition" stance — always present the ambitious version first, (b) the human/agent role contract — "Human = Design Manager, Agent = Senior Designer", and (c) iterative process framing — design is messy, that's normal.

**Acceptance criteria:**

- [ ] `write-design` contains a stance or preamble section with the "Design with Ambition" principle
- [ ] The human/agent role contract is documented (human decides, agent proposes)
- [ ] Iterative process framing is present — explicit acknowledgment that design is non-linear

#### FR-014: Migrate Risk Escalation to write-design

The `write-design` skill MUST include a risk surfacing protocol with three tiers: (a) minor risk — mention once in the design document, (b) significant risk — raise clearly with the human and request guidance, (c) security or data-integrity risk — stop immediately and create a checkpoint.

**Acceptance criteria:**

- [ ] `write-design` procedure or a dedicated section describes the 3-tier risk escalation model
- [ ] Each tier specifies the agent's expected action

#### FR-015: Port Design Quality Lens to write-design References

The six-quality evaluation lens from `kanbanzai-design/references/design-quality.md` MUST be ported to `.kbz/skills/write-design/references/design-quality.md`. The `write-design` SKILL.md MUST link to this reference file.

**Acceptance criteria:**

- [ ] `.kbz/skills/write-design/references/design-quality.md` exists and contains the six-quality lens (Simplicity, Minimalism, Completeness, Composability, Honesty, Durability — or equivalent from the source)
- [ ] `write-design` SKILL.md contains a link to this reference file
- [ ] The reference file is linked directly from SKILL.md (one level deep, per CONVENTIONS.md)

#### FR-016: Migrate Design Operational Guidance to write-design

The `write-design` skill MUST include operational guidance currently unique to `kanbanzai-design`: (a) draft lifecycle management, (b) design splitting guidance with signs that a design needs splitting and the supersession protocol, (c) gotchas — registration, content hash drift, `doc refresh`, editing approved docs, and (d) next steps after design — handoff to specification.

**Acceptance criteria:**

- [ ] `write-design` contains or references guidance on draft lifecycle, design splitting, gotchas, and next-steps-after-design
- [ ] `write-design` remains under 500 lines (use `references/` for overflow)

#### FR-017: Retire kanbanzai-design

After FR-013 through FR-016 are satisfied, the `.agents/skills/kanbanzai-design/` directory (including its `references/` subdirectory) MUST be deleted.

**Acceptance criteria:**

- [ ] `.agents/skills/kanbanzai-design/` no longer exists on disk (including `references/`)
- [ ] No remaining active instruction or routing file references `kanbanzai-design` as a current skill

---

### Area 2: Upgrade Retained System Skills

#### FR-018: Add Vocabulary Sections to System Skills

Each of the five retained system skills (`kanbanzai-getting-started`, `kanbanzai-workflow`, `kanbanzai-agents`, `kanbanzai-documents`, `kanbanzai-planning`) MUST include a vocabulary section with 5–15 domain-specific terms. Terms MUST be specific to each skill's domain (not duplicated across all five). Terms MUST pass the 15-year practitioner test: a senior expert would use this exact term when speaking with a peer.

**Acceptance criteria:**

- [ ] Each of the 5 system skills contains a `## Vocabulary` section
- [ ] Each vocabulary section contains between 5 and 15 terms
- [ ] No term appears in more than two skills' vocabulary sections (minimal duplication)
- [ ] Terms are Kanbanzai-specific or workflow-domain-specific, not general software terms

#### FR-019: Convert Anti-Patterns to Structured Format in System Skills

Each of the five retained system skills MUST have its existing prose warnings converted to named anti-patterns with Detect/BECAUSE/Resolve format. Each skill MUST have at least 3 structured anti-patterns. The BECAUSE clause MUST explain the consequence chain, not restate the detection signal.

**Acceptance criteria:**

- [ ] Each of the 5 system skills has at least 3 anti-patterns in Detect/BECAUSE/Resolve format
- [ ] Every BECAUSE clause explains *why* (consequence chain), not *what* (restatement of Detect)
- [ ] Each anti-pattern has a memorable name as a heading

#### FR-020: Add Evaluation Criteria to System Skills

Each of the five retained system skills MUST include 3–5 gradable evaluation criteria. Since system skills produce behaviour (not documents), criteria MUST evaluate observable outcomes. At least one criterion per skill MUST be marked `required`.

**Acceptance criteria:**

- [ ] Each of the 5 system skills contains an evaluation criteria section
- [ ] Each skill has between 3 and 5 criteria
- [ ] At least one criterion per skill is marked `required`
- [ ] Criteria evaluate observable outcomes, not subjective quality

#### FR-021: Add Retrieval Anchors to System Skills

Each of the five retained system skills MUST include a "Questions This Skill Answers" section as the final section of the skill body. Each skill MUST have 5–8 natural-language questions specific to its domain.

**Acceptance criteria:**

- [ ] Each of the 5 system skills has a "Questions This Skill Answers" section as its last body section
- [ ] Each section contains 5–8 natural-language questions
- [ ] Questions are specific to the skill's domain, not generic workflow questions

#### FR-022: System Skill Line Budget

After all upgrades, each system skill MUST remain under 350 lines. If additions cause a skill to exceed 350 lines, existing prose MUST be tightened rather than dropping new structural content.

**Acceptance criteria:**

- [ ] Each of the 5 upgraded system skills is at or below 350 lines
- [ ] No structural content (vocabulary, anti-patterns, evaluation criteria, retrieval anchors) was dropped to meet the line budget

#### FR-023: System Skill Frontmatter Preserved

System skills MUST retain Anthropic's native SKILL.md frontmatter format (simple `name` and `description` fields). Kanbanzai-specific frontmatter fields (`description.expert`, `description.natural`, `triggers`, `roles`, `stage`, `constraint_level`) MUST NOT be added to system skills because they are discovered by Claude Code's native skill scanner which expects the simpler format.

**Acceptance criteria:**

- [ ] No system skill has `description.expert`, `description.natural`, `triggers`, `roles`, `stage`, or `constraint_level` in its frontmatter
- [ ] Each system skill retains its existing `name` and `description` frontmatter fields

---

### Area 3: Fix Store Discipline

#### FR-024: Store Discipline in kanbanzai-agents

The `kanbanzai-agents` skill MUST include explicit store-commit discipline in its commit discipline section. The wording MUST convey that every `.kbz/state/` change is a code change, that store changes must be committed immediately or included in the next logical commit, and that `.kbz/` files must not be left uncommitted at the end of a task.

**Acceptance criteria:**

- [ ] `kanbanzai-agents` commit discipline section contains explicit `.kbz/state/` commit guidance
- [ ] The guidance states that `.kbz/state/` changes are code changes
- [ ] The guidance prohibits leaving `.kbz/` files uncommitted at the end of a task

#### FR-025: Store Discipline in kanbanzai-getting-started

The `kanbanzai-getting-started` pre-task checklist MUST include a checklist item for handling uncommitted `.kbz/` files from a previous session. The item MUST instruct the agent to commit them — not stash, discard, or ignore them.

**Acceptance criteria:**

- [ ] The pre-task checklist contains an item about uncommitted `.kbz/` files
- [ ] The item instructs the agent to commit them
- [ ] The item explicitly prohibits stashing, discarding, or ignoring them

#### FR-026: Store Discipline in AGENTS.md

The `AGENTS.md` Git Discipline section MUST include a statement that `.kbz/state/` files are versioned project state. It MUST prohibit stashing, discarding, or `.gitignore`-ing these files.

**Acceptance criteria:**

- [ ] `AGENTS.md` Git Discipline section contains `.kbz/state/` store discipline guidance
- [ ] The guidance labels these files as "versioned project state, not ephemeral cache"
- [ ] Stashing, discarding, and `.gitignore`-ing `.kbz/` files are explicitly prohibited

#### FR-027: Store Neglect Anti-Pattern in base.yaml

The `.kbz/roles/base.yaml` file MUST include a new "Store Neglect" anti-pattern with Detect, BECAUSE, and Resolve fields. The Detect field MUST reference uncommitted or discarded/stashed `.kbz/state/` files. The BECAUSE clause MUST reference parallel-agent coordination failure modes (MAST FM-2.2, FM-2.5). The Resolve field MUST instruct the agent to commit `.kbz/` changes alongside code changes and commit orphaned `.kbz/` files at task start.

**Acceptance criteria:**

- [ ] `base.yaml` contains a "Store Neglect" anti-pattern entry
- [ ] The entry has `name`, `detect`, `because`, and `resolve` fields
- [ ] The `because` field references store drift, race conditions, or silent data loss for parallel agents
- [ ] The `resolve` field covers both ongoing commit discipline and task-start orphan handling

---

### Area 4: AGENTS.md Cleanup and Discovery

#### FR-028: Deduplicate Pre-Task Checklist

The `AGENTS.md` pre-task checklist MUST be deduplicated. Project-specific items (git status, correct branch, check decision logs) MUST be retained. System-level items already covered by `kanbanzai-getting-started` ("read AGENTS.md", "read design docs") MUST be removed.

**Acceptance criteria:**

- [ ] `AGENTS.md` pre-task checklist does not contain "read AGENTS.md" or equivalent (already in `kanbanzai-getting-started`)
- [ ] `AGENTS.md` pre-task checklist does not contain "read design docs" or equivalent (already in `kanbanzai-getting-started`)
- [ ] `AGENTS.md` pre-task checklist retains git status, correct branch, and decision log checks

#### FR-029: Replace Inline Document Reading Order

The `AGENTS.md` Document Reading Order section MUST be replaced with a pointer to `refs/document-map.md`. Inline document lists that duplicate the document map MUST be removed.

**Acceptance criteria:**

- [ ] `AGENTS.md` no longer contains an inline list of essential design documents
- [ ] `AGENTS.md` contains a pointer to `refs/document-map.md` for document routing

#### FR-030: Remove Phase Labels

All phase labels (e.g., "(Phase 2a)", "(Phase 3)") MUST be removed from `AGENTS.md` Repository Structure annotations. The functional descriptions already provide sufficient context.

**Acceptance criteria:**

- [ ] No phase labels remain in `AGENTS.md` Repository Structure section
- [ ] Functional descriptions for each directory are preserved

#### FR-031: Add Mini-Vocabulary to AGENTS.md

A mini-vocabulary of 5 terms MUST be added near the top of `AGENTS.md` (after the overview, before the checklist). The terms MUST include: *stage binding*, *role*, *skill*, *lifecycle gate*, and *context packet*. Each term gets one line of definition.

**Acceptance criteria:**

- [ ] `AGENTS.md` contains a vocabulary section with exactly 5 terms
- [ ] The section appears before the pre-task checklist
- [ ] All five terms are present: stage binding, role, skill, lifecycle gate, context packet

#### FR-032: Add Pointer to Task-Execution Skills and Stage Bindings

`AGENTS.md` MUST include a pointer to `.kbz/skills/` (the task-execution skill system) and `.kbz/stage-bindings.yaml`. The pointer MUST be positioned where agents will encounter it during orientation.

**Acceptance criteria:**

- [ ] `AGENTS.md` mentions `.kbz/skills/` and `.kbz/stage-bindings.yaml`
- [ ] The mention explains that stage bindings map each workflow stage to a role and skill
- [ ] The pointer is in the main body of the file, not buried in a footnote

#### FR-033: Restructure Decision-Making Rules

The `AGENTS.md` Decision-Making Rules section MUST be restructured to remove phase-numbered log references. The section MUST guide agents to use the knowledge tool or consult `refs/document-map.md` rather than referencing specific files by phase number.

**Acceptance criteria:**

- [ ] No phase numbers remain in the Decision-Making Rules section
- [ ] The section references `refs/document-map.md` or the `knowledge` tool for finding decisions
- [ ] The core guidance ("check for prior decisions before inventing") is preserved

#### FR-034: Update refs/document-map.md

After Area 1 migrations are complete, `refs/document-map.md` MUST be updated to point to `.kbz/skills/review-code/` and `.kbz/skills/review-plan/` instead of the old `.agents/skills/kanbanzai-code-review` and `.agents/skills/kanbanzai-plan-review` paths. Any other stale skill pointers MUST also be corrected.

**Acceptance criteria:**

- [ ] `refs/document-map.md` does not reference `kanbanzai-code-review` as a current skill
- [ ] `refs/document-map.md` does not reference `kanbanzai-plan-review` as a current skill
- [ ] `refs/document-map.md` does not reference `kanbanzai-design` as a current skill
- [ ] All skill pointers in the file point to files that exist on disk

#### FR-035: Fix kanbanzai-getting-started Discovery

The `kanbanzai-getting-started` skill MUST mention the role/skill/stage-binding system so that agents bootstrapping through this skill discover the full task-execution layer. The skill MUST reference `.kbz/stage-bindings.yaml` as the mechanism for finding which role and skill to use for the current workflow stage.

**Acceptance criteria:**

- [ ] `kanbanzai-getting-started` mentions `.kbz/stage-bindings.yaml`
- [ ] `kanbanzai-getting-started` mentions `.kbz/skills/` or "task-execution skills"
- [ ] The mention explains the relationship: stage bindings map stages to roles and skills

#### FR-036: Create CLAUDE.md Bootstrap File

A `CLAUDE.md` file MUST be created at the repository root. It MUST serve as the bootstrap entry point for Claude Code users. It MUST point to `AGENTS.md` for project-specific conventions, point to `.kbz/stage-bindings.yaml` for workflow stage mapping, list the role and skill tables in condensed form, and list critical rules. It MUST stay under 150 lines because it is loaded every turn in Claude Code.

**Acceptance criteria:**

- [ ] `CLAUDE.md` exists at the repository root
- [ ] `CLAUDE.md` points to `AGENTS.md`
- [ ] `CLAUDE.md` points to `.kbz/stage-bindings.yaml`
- [ ] `CLAUDE.md` includes a condensed role table and skill table
- [ ] `CLAUDE.md` includes critical rules (use MCP tools not raw file reads, check git status, follow commit format)
- [ ] `CLAUDE.md` is at or below 150 lines

---

### Area 5: Verify Context Assembly

#### FR-037: Verify Attention-Curve Ordering in Context Assembly

The `internal/context/assemble.go` context assembly pipeline MUST be verified to produce context packets following the attention curve: identity and hard constraints first (highest attention), supporting material (spec sections, knowledge entries) in the middle, and instructions and retrieval anchors last (high attention from recency). If the ordering does not match, it MUST be fixed.

**Acceptance criteria:**

- [ ] `assemble.go` has been read and its ordering documented
- [ ] Identity/role content appears in the first positions of the assembled context
- [ ] Supporting material (spec sections, knowledge entries) appears in middle positions
- [ ] Skill instructions and retrieval anchors appear in later positions
- [ ] If the ordering was incorrect, the fix is covered by a passing test

#### FR-038: Verify Effort Budget Inclusion in Handoff

The `handoff` tool's context assembly path MUST be verified to include effort budget metadata from `.kbz/stage-bindings.yaml` in assembled context packets. If effort budgets are not included, they MUST be added.

**Acceptance criteria:**

- [ ] The code path from `handoff` through context assembly has been traced
- [ ] Effort budget metadata (e.g., "5–15 tool calls") from stage bindings is present in the assembled context packet
- [ ] If effort budgets were missing, the fix is covered by a passing test

---

## Non-Functional Requirements

### NFR-001: Line Budgets

All modified `.kbz/skills/` files MUST remain under 500 lines per `CONVENTIONS.md`. All modified `.agents/skills/` system skills MUST remain under 350 lines per the design constraint. Overflow content MUST be moved to `references/` subdirectories linked one level deep from SKILL.md.

### NFR-002: No Content Loss During Migration

Every unique content item identified in the review report's Appendix B migration checklists MUST be either (a) ported to the target skill, or (b) explicitly dropped with a documented justification in the commit message. No item may be silently omitted.

### NFR-003: Terminology Consistency

Migrated content MUST use the vocabulary terms defined in the target skill's vocabulary section. If the source skill uses a synonym (e.g., "issue" instead of "finding"), the migrated content MUST be updated to use the target skill's canonical term.

### NFR-004: Attention-Curve Compliance

All content additions to `.kbz/skills/` files MUST be placed in the correct section per the attention-curve ordering in `CONVENTIONS.md`: Vocabulary → Anti-Patterns → Checklist → Procedure → Output Format → Examples → Evaluation Criteria → Questions. Content MUST NOT be appended to the end of the file outside the section structure.

### NFR-005: BECAUSE Clause Quality

Every anti-pattern added or restructured in this feature MUST have a BECAUSE clause that explains the consequence chain. BECAUSE clauses that merely restate the Detect signal MUST be revised. The BECAUSE clause should answer "what goes wrong downstream if this anti-pattern occurs?"

### NFR-006: No Disruption to Existing Tests

All existing Go tests MUST continue to pass after any changes to `internal/context/assemble.go`. Run `go test -race ./...` to verify. No test may be deleted or weakened to achieve a pass.

---

## Acceptance Criteria

High-level acceptance criteria for the feature as a whole:

- [ ] The three retired `.agents/skills/` directories (`kanbanzai-code-review`, `kanbanzai-plan-review`, `kanbanzai-design`) no longer exist
- [ ] All unique content from the retired skills is present in the corresponding `.kbz/skills/` targets (per Appendix B checklists in the review report)
- [ ] All five retained system skills have vocabulary, structured anti-patterns, evaluation criteria, and retrieval anchor sections
- [ ] Store-commit discipline for `.kbz/state/` is documented in `kanbanzai-agents`, `kanbanzai-getting-started`, `AGENTS.md`, and `base.yaml`
- [ ] `AGENTS.md` has no duplicated checklist items, no phase labels, no inline document list, and includes pointers to `.kbz/skills/` and stage bindings
- [ ] `refs/document-map.md` contains no stale skill pointers
- [ ] `kanbanzai-getting-started` mentions stage bindings and `.kbz/skills/`
- [ ] `CLAUDE.md` exists at the repository root and is under 150 lines
- [ ] `internal/context/assemble.go` produces attention-curve-ordered context packets (verified)
- [ ] `handoff` includes effort budgets from stage bindings in context packets (verified)
- [ ] All Go tests pass: `go test -race ./...`
- [ ] No `.kbz/skills/` file exceeds 500 lines
- [ ] No `.agents/skills/` system skill exceeds 350 lines

---

## Verification Plan

### Step 1: Migration Completeness Audit

For each of the three migrations, compare the review report's Appendix B checklist (B.1, B.2, B.3) against the modified target skills. Every checklist item MUST be marked as either ported or explicitly dropped with justification.

### Step 2: Structural Quality Check

For each of the five upgraded system skills, verify:
- Vocabulary section present with 5–15 terms
- At least 3 anti-patterns in Detect/BECAUSE/Resolve format
- Evaluation criteria section with 3–5 gradable criteria
- "Questions This Skill Answers" as the final section
- Line count at or below 350

### Step 3: Store Discipline Grep

Run `grep -r "kbz/state" AGENTS.md .agents/skills/kanbanzai-agents/ .agents/skills/kanbanzai-getting-started/ .kbz/roles/base.yaml` and verify that each file contains the required store discipline content per FR-024 through FR-027.

### Step 4: Stale Reference Scan

Run `grep -r "kanbanzai-code-review\|kanbanzai-plan-review\|kanbanzai-design" AGENTS.md .agents/skills/ .kbz/ refs/ .github/` and verify no active instruction or routing file references the retired skills. Research reports and review documents are excluded from this check.

### Step 5: CLAUDE.md Validation

Verify that `CLAUDE.md` exists, is under 150 lines, and contains the five required elements (AGENTS.md pointer, stage bindings pointer, condensed role table, condensed skill table, critical rules).

### Step 6: Context Assembly Verification

Read `internal/context/assemble.go` and trace the ordering of assembled context packet elements. Document the ordering. If it does not follow the attention curve, verify the fix passes existing tests plus any new test added for the ordering guarantee.

### Step 7: Test Suite

Run `go test -race ./...` and confirm all tests pass with no regressions.