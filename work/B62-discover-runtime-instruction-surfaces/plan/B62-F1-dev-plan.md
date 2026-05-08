# B62-F1 Dev-Plan: Runtime Discovery Surfaces

| Field  | Value                                                          |
|--------|----------------------------------------------------------------|
| Date   | 2026-05-08T11:48:08Z                                           |
| Status | Draft                                                          |
| Author | architect                                                      |
| Feature | FEAT-01KR3MEYRQ9RG — Runtime Discovery Surfaces               |
| Batch  | B62 — Discover runtime instruction surfaces                    |
| Plan   | P59 — Roles & Skills Discoverability and Enforcement Remediation |

---

## Scope

This plan implements the requirements defined in
`work/B62-discover-runtime-instruction-surfaces/B62-F1-spec-runtime-discovery-surfaces.md`
(FEAT-01KR3MEYRQ9RG). It covers all five delivery surfaces:

1. Generated Anthropic-format `.claude/skills/` wrappers for the seven required
   canonical skills, plus a generator script and wrapper format specification.
2. A drift-check CI target that detects stale generated wrappers.
3. A root-level `OPENAI.md` redirect to `AGENTS.md`.
4. Concise P59 rule text added to the five live Kanbanzai tool descriptions
   (`next`, `handoff`, `entity`, `worktree`, `pr`).
5. DeepSeek host loading behaviour documented in `refs/sub-agents.md`.

The optional Cursor shim (REQ-011) is tracked as Task T6 and is sequenced last;
it is in scope only if the required surfaces are complete and capacity remains.

**Out of scope for this plan:**

- Replacing canonical `.kbz/skills/` or `.agents/skills/` files with the
  generated wrappers.
- Enforcing MCP invariants (B59 owns invariant semantics).
- Rewriting top-level registry tables (B60).
- Adding tool descriptions for `spawn_agent` and `dispatch_task`, neither of
  which is a live Kanbanzai MCP tool in the current server; `dispatch_task` is
  a P44 deliverable.

---

## Task Breakdown

### T1: Create `.claude/skills/` wrapper files and generator script

- **Description:** Author a generator script (shell or Go-based) that produces
  Anthropic-format `SKILL.md` wrappers under `.claude/skills/<skill>/SKILL.md`
  for the seven required skills. Run the generator to produce the initial set of
  files. Each wrapper must have: (a) YAML frontmatter with `name` and a
  single-line `description`; (b) a canonical-path pointer line; (c) a brief
  summary of when Claude should use this skill; (d) no more than 80 lines
  total. The files must be regular files, not symlinks.
  
  **Required skill wrappers:**
  - `.claude/skills/orchestrate-development/SKILL.md`
    → canonical: `.kbz/skills/orchestrate-development/SKILL.md`
  - `.claude/skills/implement-task/SKILL.md`
    → canonical: `.kbz/skills/implement-task/SKILL.md`
  - `.claude/skills/kanbanzai-getting-started/SKILL.md`
    → canonical: `.agents/skills/kanbanzai-getting-started/SKILL.md`
  - `.claude/skills/kanbanzai-workflow/SKILL.md`
    → canonical: `.agents/skills/kanbanzai-workflow/SKILL.md`
  - `.claude/skills/write-spec/SKILL.md`
    → canonical: `.kbz/skills/write-spec/SKILL.md`
  - `.claude/skills/write-design/SKILL.md`
    → canonical: `.kbz/skills/write-design/SKILL.md`
  - `.claude/skills/review-code/SKILL.md`
    → canonical: `.kbz/skills/review-code/SKILL.md`

- **Deliverable:**
  - `.claude/skills/` directory with seven `SKILL.md` wrapper files.
  - A generator script (e.g. `scripts/gen-claude-skills.sh` or a `make`
    target) that can regenerate all wrappers from their canonical sources.

- **Depends on:** None (independent).

- **Effort:** 3 points (medium).

- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-NF-001,
  REQ-NF-004, REQ-NF-005.

- **Acceptance criteria:**
  - Seven `SKILL.md` files exist under `.claude/skills/`.
  - Each wrapper has valid YAML frontmatter (`name`, single-line `description`).
  - Each wrapper contains the canonical source path.
  - All wrappers are regular files (not symlinks): `test ! -L`.
  - No wrapper exceeds 80 lines: `wc -l`.
  - Generator script exists and is executable.

---

### T2: Implement wrapper drift check CI target

- **Description:** Implement a check mode that detects when a generated
  `.claude/skills/` wrapper has drifted from the expected wrapper format or
  from the canonical skill's discovery metadata (name, description). The check
  must: (a) exit non-zero and print the offending wrapper path when stale
  content is detected; (b) be runnable by CI as a `make` target or shell
  invocation. The check does not need to re-generate — it only needs to
  validate that the current files on disk match what the generator would
  produce. A fixture-based integration test (stale wrapper → non-zero exit)
  is required.

- **Deliverable:**
  - Check script or Go test (e.g. `make claude-skills-check` or
    `go test ./... -run TestClaudeSkillsDrift`).
  - Makefile target or CI hook that runs the check.
  - Integration test: fixture with stale wrapper → check exits non-zero and
    reports path.

- **Depends on:** T1 (needs wrapper format and generator to define "expected"
  state).

- **Effort:** 2 points (small).

- **Spec requirements:** REQ-005, REQ-012.

- **Acceptance criteria:**
  - Given a wrapper with stale content, check exits non-zero and prints the
    wrapper path.
  - Given all wrappers current, check exits zero.
  - Check is listed in CI or a Makefile target.

---

### T3: Create `OPENAI.md` redirect

- **Description:** Create `OPENAI.md` at the repository root. The file must be
  a short redirect to `AGENTS.md` — it must not duplicate the full instruction
  corpus from `AGENTS.md`. The file must be no more than 20 lines. Include a
  one-sentence explanation of why this file exists (GPT-class hosts probe for
  `OPENAI.md` by convention).

- **Deliverable:**
  - `/OPENAI.md` (repository root).

- **Depends on:** None (independent).

- **Effort:** 1 point (small).

- **Spec requirements:** REQ-006, REQ-NF-002.

- **Acceptance criteria:**
  - File exists at the repository root.
  - Line count ≤ 20: `wc -l OPENAI.md`.
  - File contains a reference to `AGENTS.md`.
  - File does not duplicate the full AGENTS.md content (visual inspection).

---

### T4: Add P59 rule text to Kanbanzai tool descriptions

- **Description:** Edit the five live Kanbanzai tool description strings in Go
  source to embed concise P59 rule text. Each addition must be ≤ 3 lines.
  Rule text should reference B59 invariant codes where codes are defined; if
  codes are not yet published by P59-B2, use plain rule text and add a
  `// TODO(P59-B2): align with invariant code` comment.

  **Targeted tools and their workflow hazard:**
  - `next` (`internal/mcp/next_tool.go`): "Claim before work begins — do not
    start implementing without a claimed task. Orchestrators must dispatch via
    `handoff`, not direct `spawn_agent`."
  - `handoff` (`internal/mcp/handoff_tool.go`): "Use INSTEAD OF calling
    `spawn_agent` directly — this is the only safe dispatch path; direct
    `spawn_agent` bypasses context assembly and stage gate enforcement."
  - `entity` (`internal/mcp/entity_tool.go`): "Verify entity existence before
    working on it. Do not create worktrees, write files, or open PRs against
    unregistered entity names."
  - `worktree` (`internal/mcp/worktree_tool.go`): "One worktree per entity.
    Do not commit directly to main — all implementation work must be isolated
    in an entity worktree."
  - `pr` (`internal/mcp/pr_tool.go`): "Use INSTEAD OF raw
    `create_pull_request` — `pr` is entity-aware and enforces the branch/title
    conventions required by stage gate merge checks."

  Note: `spawn_agent` and `dispatch_task` are not live Kanbanzai MCP tools;
  no changes to Go source are required for those two.

- **Deliverable:**
  - Updated description strings in `next_tool.go`, `handoff_tool.go`,
    `entity_tool.go`, `worktree_tool.go`, `pr_tool.go`.
  - Each description grows by no more than 3 lines of P59 rule text.

- **Depends on:** None (independent; depends on P59-B2 for invariant codes
  but rule text can land without codes if codes are not yet published).

- **Effort:** 2 points (small).

- **Spec requirements:** REQ-007, REQ-008, REQ-NF-003.

- **Acceptance criteria:**
  - Inspection of the five tool description strings confirms P59 rule text is
    present and relevant to the tool's workflow hazard.
  - Diff review: each tool description grows by ≤ 3 lines of rule text.
  - Rule text uses B59 invariant codes where defined, or a TODO comment where
    not yet defined.
  - Existing unit tests in `internal/mcp/doc_currency_health_test.go` and
    tool-specific test files continue to pass.

---

### T5: Document DeepSeek host loading behaviour

- **Description:** Research and document how DeepSeek-hosted clients load
  instruction context. Update `refs/sub-agents.md` with a DeepSeek-specific
  section that states: (a) which host/platform is under discussion; (b) whether
  the host injects `AGENTS.md` automatically; (c) whether the host injects MCP
  tool descriptions; (d) what manual configuration remains required to load
  Kanbanzai instruction context on DeepSeek. Per REQ-010: if host behaviour
  cannot be verified, the documentation must explicitly state it is unknown and
  the DeepSeek portion must not be marked complete in review.

- **Deliverable:**
  - Updated `refs/sub-agents.md` with a DeepSeek section.

- **Depends on:** None (independent).

- **Effort:** 2 points (small — documentation work; research cost is
  bounded by REQ-010 which permits "unknown" as a valid answer).

- **Spec requirements:** REQ-009, REQ-010.

- **Acceptance criteria:**
  - `refs/sub-agents.md` contains a DeepSeek section naming the host.
  - Section addresses: AGENTS.md injection, tool-description injection,
    manual configuration requirements.
  - If behaviour is unknown, the section explicitly states that and the
    feature completion status reflects the incompleteness (per REQ-010).

---

### T6: Create Cursor rules shim (optional)

- **Description:** *Optional surface — implement only if T1–T5 are complete
  and capacity remains.* Create `.cursor/rules/kanbanzai.mdc` as a
  redirect-style rule pointing at `AGENTS.md`. The file must not duplicate
  canonical rule prose. A single rule that says "See AGENTS.md for all
  Kanbanzai workflow guidance" is sufficient.

- **Deliverable:**
  - `.cursor/rules/kanbanzai.mdc` (redirect only, no duplicated corpus).

- **Depends on:** None strictly (can be authored independently).

- **Effort:** 1 point (small).

- **Spec requirements:** REQ-011.

- **Acceptance criteria (if implemented):**
  - File exists at `.cursor/rules/kanbanzai.mdc`.
  - File contains a reference to `AGENTS.md`.
  - File does not duplicate full rule prose from `AGENTS.md`.

---

## Dependency Graph

```
T1: .claude/skills/ wrappers       (no dependencies)
T2: Wrapper drift check            → depends on T1
T3: OPENAI.md redirect             (no dependencies)
T4: Tool description rule text     (no dependencies)
T5: DeepSeek documentation         (no dependencies)
T6: Cursor shim (optional)         (no dependencies, sequenced last)
```

**Parallel groups:**
- Group A (fully independent): T1, T3, T4, T5
- Group B (after T1): T2
- Group C (optional, after required surfaces): T6

**Critical path:** T1 → T2 (two hops, both fast)

**Parallelisation opportunity:** T1, T3, T4, and T5 can be dispatched
concurrently. T2 must be serialised after T1. T6 is held until T1–T5 are
verified complete.

---

## Interface Contracts

### C1: Wrapper file format

Every `.claude/skills/<skill>/SKILL.md` file produced by T1 must conform to
this schema so T2's drift check can validate it deterministically:

```
---
name: <skill-name>           # e.g. "kanbanzai-getting-started"
description: <single line>   # max ~120 chars; used by Claude for skill discovery
---

<!-- kanbanzai-generated: true -->
<!-- canonical: <relative-path-from-repo-root> -->

# <Human-readable skill title>

When to use this skill: <one-sentence trigger condition>

For the full procedure, vocabulary, and examples see the canonical skill:
`<relative-path-from-repo-root>`
```

**Invariants the format must satisfy:**
- Frontmatter is valid YAML with exactly `name` and `description` keys.
- `description` is a single line (no embedded newlines).
- Canonical path is present and points to the actual source file.
- Total line count ≤ 80 (REQ-NF-001).
- File is a regular file, not a symlink (REQ-NF-005).

### C2: Drift-check input/output contract

The T2 check target consumes:
- `.claude/skills/*/SKILL.md` files on disk.
- The canonical metadata from the seven source skill files.

It produces:
- Exit 0 when all wrappers are current.
- Exit non-zero when one or more wrappers are stale; stale wrapper paths
  printed to stdout (one per line).

The check does **not** re-generate files. Regeneration is the generator
script's responsibility (T1 deliverable).

### C3: Tool description text budget

Tool description rule text added in T4 must conform to the following:
- Maximum 3 additional lines of P59 rule text per tool (REQ-NF-003).
- Rule text must be self-contained: readable without referencing external
  documents (tool descriptions are shown to the model inline).
- If a B59 invariant code is defined for the rule, the code appears in a
  trailing parenthetical, e.g.: `"... (P59-INV-03)"`.
- If no invariant code exists yet, a Go comment
  `// TODO(P59-B2): align with invariant code` is placed on the line above
  the rule text string.

### C4: DeepSeek documentation schema

The T5 deliverable (section in `refs/sub-agents.md`) must address these
four fields so downstream reviewers can assess completeness:

| Field | What it answers |
|-------|----------------|
| Host | Which hosting platform / API is under discussion |
| AGENTS.md injection | Does the host automatically inject AGENTS.md into the system prompt? |
| Tool-description injection | Does the host surface MCP tool descriptions to the model? |
| Manual configuration | What manual steps does a Kanbanzai operator need to take on this host? |

If a field is unknown or unverifiable, the value must be the literal string
`"Unknown — see REQ-010"` rather than an assumption.

---

## Risk Assessment

### Risk: Canonical skill frontmatter is absent or malformed

- **Probability:** Medium (`.kbz/skills/` SKILL.md files use custom frontmatter
  but it is not uniformly structured; agent skills under `.agents/skills/` use
  a different format).
- **Impact:** Medium (generator script cannot extract `name`/`description`
  without reading the correct field; wrappers could have wrong metadata).
- **Mitigation:** T1 reads the canonical frontmatter directly. Where the
  canonical file lacks an Anthropic-compatible `description` field, the
  generator derives a one-line description from the skill's `description`
  YAML field or the first `description` line in the frontmatter. Document
  the derivation rule in the generator script.
- **Affected tasks:** T1, T2.

### Risk: DeepSeek host loading behaviour is unverifiable

- **Probability:** High (no official documentation of DeepSeek host
  AGENTS.md injection; community reports are anecdotal).
- **Impact:** Low (REQ-010 explicitly permits "unknown" as a valid answer;
  the feature is not blocked).
- **Mitigation:** T5 states the behaviour as "Unknown" per REQ-010 and marks
  the DeepSeek portion of the feature accordingly at review. The implementer
  should search public DeepSeek API and Coder platform documentation before
  concluding "unknown".
- **Affected tasks:** T5.

### Risk: B59 invariant codes are not yet published when T4 is implemented

- **Probability:** Medium (P59-B2 is a separate workstream; its delivery
  timeline relative to B62 is not guaranteed).
- **Impact:** Low (REQ-008 says "where a rule has an invariant code" — if
  no codes exist yet, rule text lands without codes; a TODO comment tracks
  the alignment work).
- **Mitigation:** Per Interface Contract C3, add
  `// TODO(P59-B2): align with invariant code` inline. The doc_currency
  health checker does not scan Go comments, so no test drift is introduced.
- **Affected tasks:** T4.

### Risk: `.claude/skills/` directory conflicts with future Anthropic Skills format changes

- **Probability:** Low (Anthropic Skills format is stable in production today).
- **Impact:** Medium (all seven wrappers would need regenerating if the
  frontmatter schema changes).
- **Mitigation:** The generator script is the single mutation point. When the
  format changes, update the generator and re-run it. T2's drift check will
  catch any manual drift between generator runs.
- **Affected tasks:** T1, T2.

---

## Verification Approach

Maps every acceptance criterion from the specification to the task that produces
the verification evidence.

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|---------------------|----------------|
| AC-001 (REQ-001): seven wrappers exist in `.claude/skills/` | Directory listing | T1 |
| AC-002 (REQ-002, REQ-003): wrappers have valid frontmatter and canonical path | Frontmatter parse + content inspection | T1 |
| AC-003 (REQ-004, REQ-NF-005): wrappers are regular files, not symlinks | `test ! -L` per file | T1 |
| AC-004 (REQ-005, REQ-012): stale wrapper causes check to exit non-zero with path | Integration test (fixture with stale wrapper) | T2 |
| AC-005 (REQ-006): `OPENAI.md` redirects to `AGENTS.md` in ≤ 20 lines | Inspection + `wc -l` | T3 |
| AC-006 (REQ-007, REQ-008): seven tool descriptions include concise P59 rule text | Code inspection + diff review | T4 |
| AC-007 (REQ-009): DeepSeek section covers host, loading, and manual config | Documentation inspection | T5 |
| AC-008 (REQ-010): unknown behaviour marked incomplete | Review gate check | T5 |
| AC-009 (REQ-011): Cursor shim redirects to `AGENTS.md` only | Inspection | T6 (if implemented) |
| AC-010 (REQ-NF-001): no wrapper exceeds 80 lines | `wc -l` per file | T1 |
| AC-011 (REQ-NF-003): each tool description grows by ≤ 3 lines | Diff line count | T4 |

**Integration verification points:**

- After T1 and T2 are both complete: run `make claude-skills-check` (or
  equivalent) against the committed wrappers. Confirm exit 0.
- After T4: run the existing `doc_currency_health_test.go` suite and any
  tool-level tests for the five modified tools. Confirm no regressions.
- Final review gate: all five required surfaces (T1–T5) must be marked done
  before the feature can advance to reviewing stage. T6 is optional and does
  not block the gate.
