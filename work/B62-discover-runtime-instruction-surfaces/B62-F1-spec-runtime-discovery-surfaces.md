| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:06:29Z |
| Status | Draft |
| Author | spec-author |
| Plan | P59 — Roles & Skills Discoverability and Enforcement Remediation |
| Batch | B62 — Discover runtime instruction surfaces |
| Feature | FEAT-01KR3MEYRQ9RG — Runtime discovery surfaces |
| Design | `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` |

## Overview

This specification implements the B5 Discover portion of the design described in `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` (`P59-roles-skills-remediation/design-p59-design-roles-skills-remediation`). It covers the instruction surfaces that target runtimes actually discover: Anthropic Skills wrappers for Claude, an `OPENAI.md` redirect for GPT-class hosts, tool-description rule text for DeepSeek-sensitive clients, and operational documentation for DeepSeek host loading.

The problem is that Kanbanzai's canonical role and skill files are not automatically loaded by every target runtime. Claude needs `.claude/skills/` discovery metadata, GPT-class hosts commonly look for `AGENTS.md` or related redirects, and DeepSeek over-weights tool descriptions relative to long markdown instructions. This feature adds discovery surfaces without changing the canonical authoring format.

## Scope

In scope:

- Generated `.claude/skills/<skill>/SKILL.md` wrappers for the high-leverage skill subset named in the design.
- A root-level `OPENAI.md` redirect to `AGENTS.md`.
- Tool-description rule text for `next`, `handoff`, `spawn_agent`, `dispatch_task`, `entity`, `worktree`, and `pr`.
- Documentation of DeepSeek host system-prompt loading behaviour in `refs/sub-agents.md` or the canonical sub-agent reference.
- Optional `.cursor/rules/` shim if capacity remains after required surfaces are complete.
- Sync/check behaviour for generated discovery wrappers.

Out of scope:

- Replacing canonical `.kbz/skills/` or `.agents/skills/` files with Anthropic Skills as the source of truth.
- Enforcing the MCP invariants themselves; B59 defines invariant semantics.
- Rewriting the full top-level registry tables; B60 covers generated registry surfaces.
- Supporting runtimes beyond Claude, GPT-class hosts, DeepSeek-hosted clients, and optional Cursor.

Related work checked:

- `work/P59-roles-skills-remediation/P59-report-roles-skills-audit.md` identifies missing `.claude/skills/`, missing `OPENAI.md`, and DeepSeek's tool-description sensitivity.
- `work/P44-model-routing-agent-launcher/P44-F1-design-prompt-assembly-gate.md` informs dispatch-related tool-description wording, but P44 owns dispatch implementation.
- `.agents/skills/kanbanzai-getting-started/SKILL.md` and `.agents/skills/kanbanzai-workflow/SKILL.md` are high-leverage runtime discovery candidates.

## Functional Requirements

- **REQ-001:** The repository must contain generated Anthropic-format wrappers under `.claude/skills/` for these required skills: `orchestrate-development`, `implement-task`, `kanbanzai-getting-started`, `kanbanzai-workflow`, `write-spec`, `write-design`, and `review-code`.
- **REQ-002:** Each `.claude/skills/` wrapper must contain frontmatter with a skill name and a single-line description suitable for Claude skill discovery.
- **REQ-003:** Each `.claude/skills/` wrapper must state the canonical skill path it mirrors and must direct readers to the canonical file for full procedure text.
- **REQ-004:** `.claude/skills/` wrappers must be generated copies, not symlinks.
- **REQ-005:** A check mode must detect when a generated `.claude/skills/` wrapper diverges from its canonical source metadata or required wrapper format.
- **REQ-006:** The repository root must contain `OPENAI.md` as a short redirect to `AGENTS.md` and must not duplicate the full AGENTS content.
- **REQ-007:** Tool descriptions for `next`, `handoff`, `spawn_agent`, `dispatch_task`, `entity`, `worktree`, and `pr` must include concise rule text relevant to the tool's workflow hazard.
- **REQ-008:** Tool-description rule text must align with B59 invariant codes where a rule has an invariant code.
- **REQ-009:** DeepSeek host loading behaviour must be documented in `refs/sub-agents.md` or the canonical sub-agent reference. The documentation must state which host is used, whether it injects `AGENTS.md`, whether it injects tool descriptions, and what manual configuration remains required.
- **REQ-010:** If DeepSeek host behaviour cannot be verified, the documentation must explicitly state that it is unknown and the DeepSeek portion of the feature must not be marked complete.
- **REQ-011:** If the optional Cursor shim is implemented, `.cursor/rules/` must contain a single redirect-style rule pointing at `AGENTS.md` and must not duplicate canonical rule prose.
- **REQ-012:** Generated discovery wrappers must be included in the same drift-check mechanism as other generated instruction surfaces or in a dedicated check target that CI can run.

## Non-Functional Requirements

- **REQ-NF-001:** Each `.claude/skills/` wrapper must be no more than 80 lines.
- **REQ-NF-002:** `OPENAI.md` must be no more than 20 lines.
- **REQ-NF-003:** Tool-description additions must be concise: each modified description may grow by no more than 3 lines for P59 rule text.
- **REQ-NF-004:** Discovery wrappers must preserve canonical-source ownership: changing a canonical skill's discovery description must update generated wrappers through the generator or sync mechanism, not hand edits.
- **REQ-NF-005:** Required discovery surfaces must work on filesystems that do not preserve symlinks.

## Constraints

- `.claude/skills/` wrappers are discovery surfaces only. Canonical skill content remains under `.kbz/skills/` or `.agents/skills/`.
- Generated wrappers must be portable copies, not symlinks.
- Tool-description rule text must be short and specific to the tool. Long explanations belong in canonical skills, not tool descriptions.
- `OPENAI.md` is a redirect, not a second instruction corpus.
- DeepSeek host documentation is a completion prerequisite for DeepSeek claims. Unknown host behaviour must remain visible.
- Optional Cursor support must not delay required Claude, GPT, and DeepSeek discovery surfaces.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given `.claude/skills/`, when the directory is inspected, then wrappers exist for all seven required skills.
- **AC-002 (REQ-002, REQ-003):** Given any required wrapper, when its content is inspected, then it has valid discovery frontmatter, a single-line description, and a canonical source path.
- **AC-003 (REQ-004, REQ-NF-005):** Given the generated wrappers, when filesystem metadata is inspected, then wrappers are regular files and not symlinks.
- **AC-004 (REQ-005, REQ-012):** Given a wrapper with stale generated content, when the wrapper check runs, then it exits non-zero and reports the wrapper path.
- **AC-005 (REQ-006):** Given `OPENAI.md`, when it is inspected, then it redirects to `AGENTS.md` in no more than 20 lines and does not duplicate the full instruction corpus.
- **AC-006 (REQ-007, REQ-008):** Given the seven named tool descriptions, when they are inspected or rendered, then each includes concise P59 rule text aligned with relevant B59 invariant codes.
- **AC-007 (REQ-009):** Given the DeepSeek host documentation, when it is read, then it names the host, describes AGENTS.md loading, describes tool-description loading, and states remaining manual configuration.
- **AC-008 (REQ-010):** Given unverifiable DeepSeek host behaviour, when the feature is reviewed, then the DeepSeek item is marked incomplete rather than assumed complete.
- **AC-009 (REQ-011):** Given optional Cursor support is implemented, when `.cursor/rules/` is inspected, then it contains only redirect-style guidance to `AGENTS.md`.
- **AC-010 (REQ-NF-001):** Given each `.claude/skills/` wrapper, when line counts are measured, then no wrapper exceeds 80 lines.
- **AC-011 (REQ-NF-003):** Given tool-description diffs, when added lines are counted, then each description grows by no more than 3 P59 rule-text lines.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | List `.claude/skills/` and confirm wrappers for all seven required skills. |
| AC-002 | Inspection/test | Parse wrapper frontmatter and confirm single-line description plus canonical source path. |
| AC-003 | Filesystem inspection | Verify wrappers are regular files rather than symlinks. |
| AC-004 | Integration test | Make a fixture wrapper stale and assert check mode fails with the wrapper path. |
| AC-005 | Inspection | Read `OPENAI.md` and confirm redirect behaviour, size limit, and no duplicated corpus. |
| AC-006 | Test/inspection | Render or inspect tool descriptions and confirm concise B59-aligned rule text. |
| AC-007 | Inspection | Read DeepSeek host documentation and verify the host, AGENTS.md loading, tool-description loading, and manual configuration fields. |
| AC-008 | Review | Confirm completion status treats unknown DeepSeek host behaviour as incomplete. |
| AC-009 | Inspection | If implemented, inspect `.cursor/rules/` for redirect-only guidance. |
| AC-010 | Script | Count wrapper lines and assert the 80-line maximum. |
| AC-011 | Diff review | Count added P59 rule-text lines per tool-description diff and assert the 3-line maximum. |
