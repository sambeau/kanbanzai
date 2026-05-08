| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:50:54Z |
| Status | approved |
| Author | architect |
| Feature | FEAT-01KR3MES5ZJ11 — Instruction corpus cleanup |
| Batch | B61 — Tidy contradictions stray files and corpus size |
| Spec | `work/B61-tidy-contradictions-stray-files-corpus-size/B61-F1-spec-instruction-corpus-cleanup.md` |

## Scope

This plan implements all requirements defined in
`work/B61-tidy-contradictions-stray-files-corpus-size/B61-F1-spec-instruction-corpus-cleanup.md`
(REQ-001 through REQ-012, REQ-NF-001 through REQ-NF-005). It covers six tasks
executed in a predominantly parallel configuration, with a single serial
verification task at the end.

It does **not** cover: B60 registry generation, B62 runtime discovery wrappers,
changes to canonical role/skill authoring format, or any rewriting of role
identities or skill procedures.

## Task Breakdown

### Task 1: Replace stray SKILL.md files with directory indexes

- **Description:** Remove `.kbz/skills/SKILL.md` and `.agents/skills/SKILL.md` (the two
  top-level stray skill files that create false discovery surfaces). Create
  `.kbz/skills/README.md` and `.agents/skills/README.md` in their place. Each README must
  explain where real `*/SKILL.md` entries live and must not itself be structured like an
  executable skill (no `name:`, `triggers:`, or `## Procedure` headings).
- **Deliverable:** `.kbz/skills/SKILL.md` and `.agents/skills/SKILL.md` absent from the
  repository; `.kbz/skills/README.md` and `.agents/skills/README.md` present and pointing to
  real skill subdirectories.
- **Depends on:** None
- **Effort:** 1 SP
- **Spec requirements:** REQ-001, REQ-002, REQ-NF-003

### Task 2: Align top-level instruction files on the no-stash rule

- **Description:** Locate and fix every top-level agent instruction file that currently
  advises using `git stash` as a valid workflow action. The confirmed location is
  `.github/copilot-instructions.md` line 132 ("Commit or stash previous work first"). Replace
  `stash` guidance with the canonical no-stash rule already stated in `AGENTS.md`:
  previous work is committed, left for the human, or isolated with worktrees. Verify no
  other top-level instruction files (`AGENTS.md`, `.claude/`, etc.) retain a weaker
  `commit or stash` alternative.
- **Deliverable:** No top-level instruction file instructs agents to use `git stash` for
  Kanbanzai workflow state. Specifically `.github/copilot-instructions.md` line 132 updated.
- **Depends on:** None
- **Effort:** 1 SP
- **Spec requirements:** REQ-003

### Task 3: Reduce over-budget skills and relocate long examples

- **Description:** Two skills exceed 500 lines and one skill has an Examples section over 60
  lines. Address all three together since the fixes overlap:

  1. `.kbz/skills/orchestrate-development/SKILL.md` (520 lines): Move the `## Examples`
     section (~95 lines) to
     `.kbz/skills/orchestrate-development/references/examples-orchestrate-development.md`
     and replace it with a one-line link. This reduces the skill below 500 lines and
     satisfies the examples-relocation requirement.

  2. `.agents/skills/kanbanzai-agents/SKILL.md` (501 lines): Identify the minimum 2+ line
     reduction needed. Candidate: move any inline examples or lengthy illustrative blocks
     to a reference file; alternatively trim redundant prose. Apply the project dual-write
     rule — if a corresponding embedded copy exists at
     `internal/kbzinit/skills/agents/SKILL.md`, apply the same edit there.

  3. Survey all remaining skill files for any `## Examples` section exceeding 60 lines
     (beyond the two already identified) and apply the same move-and-link pattern.

  Required sections that must be preserved in both skills after reduction:
  `orchestrate-development`: procedure, checklist, output-format.
  `kanbanzai-agents`: dispatch, commit, finish, knowledge-contribution guidance.
- **Deliverable:** `orchestrate-development/SKILL.md` < 500 lines with link to reference
  file. `kanbanzai-agents/SKILL.md` < 500 lines (dual-write applied if applicable).
  Reference files present and linked. No remaining skill has an Examples section > 60 lines.
- **Depends on:** None
- **Effort:** 3 SP
- **Spec requirements:** REQ-004, REQ-005, REQ-006, REQ-NF-002, REQ-NF-003, REQ-NF-004

### Task 4: Audit and fix base role tool access

- **Description:** Remove `grep` and `search_graph` from the `tools:` list in
  `.kbz/roles/base.yaml`. Then audit every role file that inherits from `base` to verify
  it still has the tools its skill procedure requires. Add `grep` and/or `search_graph`
  explicitly to any role file that needs them but currently relies on base inheritance.

  Current state (from audit): `grep` and `search_graph` are present in base and also
  explicitly listed in most role files (architect, implementer, implementer-go, researcher,
  reviewer, reviewer-conformance, reviewer-quality, reviewer-testing, documenter,
  doc-checker, doc-editor, doc-pipeline-orchestrator, spec-author, plan-validator,
  review-gate-validator, verifier). Roles that do NOT currently list them explicitly must
  be checked against their skill procedure to determine whether explicit addition is needed.

  The roles that do not currently list `grep`/`search_graph` explicitly include:
  `orchestrator`, `spec-validator`, `doc-copyeditor`, `doc-stylist`. Review each against
  their skill's tool needs before deciding whether to add or omit.
- **Deliverable:** `.kbz/roles/base.yaml` no longer lists `grep` or `search_graph`. Every
  role that needs those tools for its skill procedure lists them explicitly. No stage-bound
  role lost a required tool.
- **Depends on:** None
- **Effort:** 2 SP
- **Spec requirements:** REQ-008, REQ-009, REQ-NF-005

### Task 5: De-duplicate anti-pattern prose between skills and roles

- **Description:** Identify anti-pattern text in skill SKILL.md files that duplicates
  (verbatim or near-verbatim) the `anti_patterns:` entries in the corresponding role YAML.
  For each confirmed duplicate, replace the skill copy with a brief cross-reference such as:
  "Anti-patterns for this skill are documented in the `<role-id>` role
  (`.kbz/roles/<role-id>.yaml`)."

  Primary candidate: `.kbz/skills/orchestrate-development/SKILL.md` has three
  `## Anti-Pattern` headers, which may overlap with `.kbz/roles/orchestrator.yaml`'s
  `anti_patterns:` section. Other candidates to inspect: any skill whose anti-pattern
  section has prose matching role entries.

  REQ-NF-001 constraint: changes must be mechanical replacements only — do not rewrite
  the canonical role entries or add new anti-pattern content.
- **Deliverable:** No skill SKILL.md contains verbatim or near-verbatim anti-pattern prose
  already present in its corresponding role YAML. Replaced prose links to the canonical
  role file.
- **Depends on:** None
- **Effort:** 2 SP
- **Spec requirements:** REQ-007, REQ-NF-001

### Task 6: Measure corpus size and verify invariants

- **Description:** Produce the before-and-after corpus-size report required by REQ-010 and
  REQ-011. The "before" baseline can be derived from the P59 audit report
  (`work/P59-roles-skills-remediation/P59-report-roles-skills-audit.md`) or re-measured
  before applying any changes. After all Tasks 1–5 are merged, run the documented
  line-count command across all skill files and produce the after count.

  The documented repeatable command is:
  `find .kbz/skills -name 'SKILL.md' -print0 | xargs -0 wc -l` and the equivalent for
  `.agents/skills`. Record both subtotals and the combined total.

  Then verify REQ-012: check that every hard workflow invariant identified in the P59 B2
  invariant catalog remains discoverable in the cleaned-up corpus. Cross-reference the
  invariant list against the post-cleanup skill files.

  Output location: `work/B61-tidy-contradictions-stray-files-corpus-size/B61-F1-corpus-report.md`.
- **Deliverable:** Corpus-size report at the output location showing before/after line
  counts. Total ≤ 6,500 lines or a documented exception. Invariant checklist confirming
  all hard workflow rules remain discoverable.
- **Depends on:** Task 1, Task 2, Task 3, Task 4, Task 5
- **Effort:** 1 SP
- **Spec requirements:** REQ-010, REQ-011, REQ-012, REQ-NF-004

## Dependency Graph

```
Task 1 (stray files)           ──┐
Task 2 (no-stash)              ──┤
Task 3 (skill reduction)       ──┼──► Task 6 (corpus report + invariants)
Task 4 (base role tools)       ──┤
Task 5 (anti-pattern dedup)    ──┘
```

Parallel groups: [Task 1, Task 2, Task 3, Task 4, Task 5]
Serial: Task 6 after all parallel tasks complete
Critical path: any of Tasks 1–5 → Task 6 (minimum 2-task chain)

No false dependencies identified. Tasks 1–5 touch distinct files and have no
ordering constraints between them. They can be implemented sequentially or in any
parallel combination.

## Risk Assessment

### Risk: Dual-write miss for kanbanzai-agents

- **Probability:** medium
- **Impact:** medium — embedded copy at `internal/kbzinit/skills/agents/SKILL.md`
  diverges from the live copy, causing deployed consumer installs to receive the
  pre-cleanup text.
- **Mitigation:** Task 3 explicitly calls out the dual-write rule and requires the
  implementer to inspect and update `internal/kbzinit/skills/agents/SKILL.md` in the same
  commit.
- **Affected tasks:** Task 3

### Risk: Base role tool removal breaks a stage-bound role

- **Probability:** low — nearly all roles already list `grep`/`search_graph` explicitly;
  the audit found only four roles that do not (orchestrator, spec-validator, doc-copyeditor,
  doc-stylist). Of these, none appear to require code-search tools.
- **Impact:** high — a role losing a required tool silently degrades agent capability
  without an error.
- **Mitigation:** Task 4 requires explicit verification of each role against its skill's
  tool needs before removing the base entry. REQ-NF-005 and AC-011 enforce this.
- **Affected tasks:** Task 4

### Risk: Examples move breaks link or leaves orphaned reference

- **Probability:** low
- **Impact:** low — only one reference file is needed for `orchestrate-development` and
  optionally one for `kanbanzai-agents`.
- **Mitigation:** Task 3 requires relative links from the skill to the reference file.
  REQ-NF-003 enforces discoverability.
- **Affected tasks:** Task 3

### Risk: Corpus total exceeds 6,500 lines after cleanup

- **Probability:** low — the primary over-budget items are the two identified skills and
  the stray SKILL.md files (457 lines each). Removing stray files alone frees ~914 lines.
- **Impact:** medium — REQ-011 requires documented exception if threshold is missed.
- **Mitigation:** Task 6 measures the after total and documents an exception if needed.
  Early estimation: removing two stray files (~914 lines) plus moving examples (~95 lines)
  plus minor skill trimming should comfortably reduce the total.
- **Affected tasks:** Task 6

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----|
| AC-001 (REQ-001) | Inspection: confirm `.kbz/skills/SKILL.md` absent and README present | Task 1 |
| AC-002 (REQ-002) | Inspection: confirm `.agents/skills/SKILL.md` absent and README present | Task 1 |
| AC-003 (REQ-003) | Search: `grep -r "stash"` in top-level instruction files | Task 2 |
| AC-004 (REQ-004) | Script: `wc -l .kbz/skills/orchestrate-development/SKILL.md` < 500 | Task 3 |
| AC-005 (REQ-005) | Script: `wc -l .agents/skills/kanbanzai-agents/SKILL.md` < 500 | Task 3 |
| AC-006 (REQ-006) | Inspection: examples in reference file, link present in source skill | Task 3 |
| AC-007 (REQ-007) | Search/inspection: duplicate anti-pattern text replaced by references | Task 5 |
| AC-008 (REQ-008, REQ-009) | Inspection: base role lacks `grep`/`search_graph`; needing roles list explicitly | Task 4 |
| AC-009 (REQ-010, REQ-011) | Script: combined corpus count ≤ 6,500 or exception documented | Task 6 |
| AC-010 (REQ-012) | Inspection: B59 invariant names found in post-cleanup corpus | Task 6 |
| AC-011 (REQ-NF-005) | Review: each stage-bound role+skill pair has required tools after base cleanup | Task 4 |
