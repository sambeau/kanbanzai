| Field  | Value                                                          |
|--------|----------------------------------------------------------------|
| Date   | 2026-05-08                                                     |
| Status | Draft                                                          |
| Author | sambeau (architect role, write-design skill)                   |
| Plan   | P59 — Roles & Skills Discoverability and Enforcement Remediation |
| Source | `work/P59-roles-skills-remediation/P59-report-roles-skills-audit.md` |

## Overview

P59 is the rules-and-prompts companion to P44 (architectural enforcement). The
audit report concludes that Kanbanzai's role/skill *content* is largely sound,
but the **distribution mechanism** is failing: agents do not reliably load the
corpus, when they do load it the volume defeats the U-curve the rules were
written to exploit, and the highest-leverage rules live in advisory prose
rather than in tool semantics where they can be enforced.

This design proposes a three-track remediation, sequenced so that the highest
leverage interventions land first:

1. **Inject** — push role identity and a small set of operational constraints
   into every relevant MCP response so the rules cannot be missed.
2. **Enforce** — move five high-violation rules from skill markdown into MCP
   tool semantics (refusals, validations, derived dispatch).
3. **Tidy and unify** — collapse the three drifting registries (CLAUDE.md,
   copilot-instructions.md, README) into one generated source of truth, fix
   the live contradictions, cut the corpus by ~50%, and add the discovery
   surfaces the audit identified as missing (`.claude/skills/`, `OPENAI.md`,
   optionally `.cursor/rules/`).

The work is decomposed into five batches under P59. Each batch maps to a
distinct concern and a distinct write scope so they can be sequenced with
minimal merge contention.

## Goals and Non-Goals

### Goals

- **G1.** Make the operational constraint set visible to every agent on every
  task claim, regardless of host or model. The agent must not be able to
  proceed without seeing role identity and the top constraints.
- **G2.** Convert the five most-violated advisory rules into MCP tool
  invariants, so non-compliance becomes a refusal rather than a bad outcome.
- **G3.** Collapse the role/skill registry to a single source of truth. The
  `CLAUDE.md`, `.github/copilot-instructions.md`, and README copies are
  derived (generated or one-line redirects) and cannot drift.
- **G4.** Resolve every live contradiction across the top-level instruction
  files, beginning with the `stash` contradiction between `CLAUDE.md` and the
  `kanbanzai-getting-started` skill.
- **G5.** Add the discovery surfaces the major runtimes actually look for:
  Anthropic Skills (`.claude/skills/`) for Claude, an `OPENAI.md` redirect
  for runtimes that probe it, and rule text in MCP tool descriptions for
  DeepSeek (which over-trusts tool descriptions vs. markdown).
- **G6.** Cut the role/skill corpus to ≈6,000 lines (≈50% reduction) without
  losing rules — savings come from de-duplication, moving examples to
  `references/`, and bringing the two over-budget skills under 500 lines.
- **G7.** Add a CI guard that detects future divergence between the registry
  source of truth and its generated copies.

### Non-Goals

- **NG1.** This plan does not re-author the role or skill *content*. The
  audit is explicit that content is largely fine; the work is structural.
- **NG2.** This plan does not duplicate P44's architectural-enforcement
  workstream. Where the two overlap (e.g. handoff-only dispatch enforcement),
  the work belongs in P44; P59 supplies the rule list and the prompt-layer
  changes.
- **NG3.** This plan does not change the stage-bindings model itself. It
  treats `stage-bindings.yaml` as the canonical source and propagates from it.
- **NG4.** This plan does not target hosts other than Claude Code/CLI,
  Codex/GPT-via-MCP, and DeepSeek. A `.cursor/rules/` shim is in scope only
  as a "Could", not a "Must".
- **NG5.** This plan does not migrate skills to a new authoring format. The
  current YAML+SKILL.md format is preserved.
- **NG6.** Telemetry of skill obedience (audit suggestion #17) is out of
  scope for the initial remediation; revisit after the structural fixes land.

## Problem and Motivation

The P59 report establishes the failure pattern in detail; the design only
needs to restate the operative claims:

- The corpus today comprises ≈11,400 lines of skill text across 36 files,
  22 role YAMLs, three top-level instruction files, five reference files, and
  a stage-bindings registry. A complete load is well past 100K tokens.
- Reaching any operational rule from a Claude Code session requires four
  indirect file reads (`CLAUDE.md` → `AGENTS.md` → `stage-bindings.yaml` →
  role + skill). Each hop is an opportunity for the agent to skip, partially
  read, or be diverted by an intervening tool call.
- None of the three target runtimes auto-loads the skill files: Claude Code
  loads only `CLAUDE.md`; Codex/GPT loads `AGENTS.md`; DeepSeek loads
  whatever the host injects, with no skill-file discovery convention of its
  own.
- The volume of the corpus defeats the U-curve attention model the corpus
  was written to exploit. Agents are forced to load selectively, but the
  selection mechanism is exactly the indirect chain they tend to skip.
- The most-violated rules (handoff-only dispatch, entity existence,
  commit-before-task, no `.kbz/state/` shell reads, gate prerequisites) are
  carried as prose advice rather than as tool invariants. Prose advice is
  not control.
- A direct contradiction exists today: `CLAUDE.md` says "commit or stash
  previous work first"; `kanbanzai-getting-started` says never stash. LLMs
  resolve this by recency, not by hierarchy.
- Two stray `SKILL.md` files at `.kbz/skills/SKILL.md` and
  `.agents/skills/SKILL.md` create discoverability ambiguity (the genuine
  `prompt-engineering` skill lives one directory deeper).

If nothing changes: agents will continue to operate on whichever subset of
rules happens to land in their context, compliance will continue to vary by
runtime and by session, and the project will keep absorbing the cost of
incidents whose root cause is "rule was never loaded" rather than "rule was
disregarded." The P50/P55/P56/P57/P58 evidence trail bears this out.

## Design

The remediation is organised into **five batches** under P59. Each batch
owns a distinct surface area and can be reviewed and merged independently.
The batches are ordered by leverage; B1 and B2 deliver most of the
compliance improvement and should land first.

```
P59 — Roles & Skills Remediation
  ├── B1  Inject (constraint card + stage-binding hydration)
  ├── B2  Enforce (move 5 rules into MCP tool semantics)        ⟷ P44 boundary
  ├── B3  Unify (single source of truth for the registry; CI guard)
  ├── B4  Tidy (resolve contradictions, delete stray files, cut corpus)
  └── B5  Discover (.claude/skills/, OPENAI.md, tool-description rules)
```

### Components and responsibilities

#### B1 — Inject: constraint card and stage-binding hydration

A new component, the **Constraint Card Renderer**, is added to the MCP
server. Its responsibilities:

- Compose a fixed-shape ~25-line block containing role identity, the top 5
  hard constraints, tool routing rules, and a "never compose prompts
  manually" reminder.
- The block is **generated programmatically from the role YAML and a
  curated `constraints.yaml` registry**, never hand-written, so it cannot
  drift from the authoritative source.
- The block is prepended (not appended) to the response body of:
  - `next` (when a task is claimed),
  - `handoff` (when a task is dispatched to a sub-agent),
  - any other response that delivers task or feature context.
- Stage-binding payloads (role name, skill names, effort budget, sub-agent
  profile, prerequisites) are inlined into the same response so the agent
  no longer has to remember to read `.kbz/stage-bindings.yaml`.

Boundaries:

- The renderer does **not** read or rewrite skill markdown. It composes
  short structured text from typed sources.
- The renderer is unit-tested at the wire level: golden tests assert the
  exact rendered card for each role × stage pair.
- The renderer is the only place that knows the top-N constraint list; both
  the human-facing CLI and the MCP responses go through it.

Failure modes:

- A role YAML missing required fields (identity, vocabulary, top
  constraints) is a hard error at server start. The card cannot be silently
  empty.
- An unknown stage falls back to a generic card that names the failure
  loudly ("UNKNOWN STAGE; load stage-bindings.yaml manually") so the
  problem surfaces in the agent transcript rather than silently passing.

#### B2 — Enforce: rules as MCP tool invariants

The five high-violation rules are converted into tool-level refusals. This
batch is the prompt-layer counterpart of P44; ownership of the actual
enforcement code may live in P44 with P59 supplying the rule list and the
removed prose. Each rule maps to one tool change:

| # | Rule | Enforcement point | Failure mode |
|---|------|-------------------|--------------|
| 1 | Always use `handoff`, never compose prompts manually | Remove `spawn_agent` from orchestrator's `tools` list; route via a `dispatch_task` wrapper that internally calls `handoff` | Tool unavailable; refusal explains the alternative |
| 2 | Verify entity exists before working on it | `next` and `handoff` refuse when `entity_id` is unregistered | Refusal includes "create the entity first or pick a registered ID" |
| 3 | Commit `.kbz/state/` before starting a task | `next` refuses to claim if `git status` shows orphaned `.kbz/state/` files | Refusal lists the files; suggests the commit message format |
| 4 | Don't read `.kbz/state/` with shell tools | Not enforceable against the host filesystem; instead, every MCP response carries a one-line warning | Warning is in the constraint card from B1 |
| 5 | Stage gates require artefacts | Already enforced for features; extend to bugs (P56-tracked) | Existing refusal pattern reused |

Boundaries:

- Each refusal carries a stable error code so callers (and CI tests) can
  assert the refusal happened.
- An `override` + `override_reason` escape hatch exists for the gate
  enforcements (per existing convention) but **not** for rule 1 (handoff
  bypass) — that one is a hard architectural invariant.
- The corresponding prose in `orchestrator.yaml` and
  `orchestrate-development/SKILL.md` is reduced to a one-line cross-reference
  to the tool error code, eliminating the "duplicated verbatim" problem the
  audit calls out.

#### B3 — Unify: single source of truth, derived registries, CI guard

Today the role/skill registry appears (and drifts independently) in three
places: `CLAUDE.md`, `.github/copilot-instructions.md`, and the project
`README.md`. The design collapses this to one source plus generated copies.

Components:

- **Registry source.** `stage-bindings.yaml` plus `.kbz/roles/*.yaml` are
  the canonical registry. No other file declares roles or skills.
- **Generator.** A new `kanbanzai docs sync` subcommand (or `internal/`
  generator invoked at install time) emits the role/skill table sections
  of `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md` from
  the canonical source. Generated regions are bracketed with marker
  comments so hand-edits to surrounding prose are preserved.
- **CI guard.** A `make registry-check` target runs the generator in
  `--check` mode (no writes) and fails CI if the generated content
  diverges from what is on disk. This is added to the existing CI matrix.

Boundaries:

- The generator does not touch `AGENTS.md` in this batch — `AGENTS.md` is
  the canonical narrative file for Codex/GPT and is kept hand-authored. The
  registry section inside `AGENTS.md` is in scope for the same generator if
  it proves stable in the other three files first.
- `CLAUDE.md` is reduced to a thin redirect plus a generated registry
  table. The half-and-half state the audit identifies as "actively
  misleading Claude" is eliminated.

#### B4 — Tidy: contradictions, stray files, corpus cut

This batch is mechanical cleanup that the audit lists as "Must":

- Delete `.kbz/skills/SKILL.md` and `.agents/skills/SKILL.md` (the two
  stray top-level files). Replace each with a `README.md` index.
- Resolve the `stash` contradiction. The decision (see Decisions §D1) is
  to forbid stashing project-wide; `CLAUDE.md` is updated and the
  `kanbanzai-getting-started` text is left as-is.
- Trim the two over-budget skills (`orchestrate-development` 520 lines;
  `kanbanzai-agents` 501 lines) under the 500-line limit by relocating
  long examples to `references/`.
- De-duplicate the anti-patterns that appear verbatim in both a role YAML
  and a skill SKILL.md. Canonical copy stays in the role; the skill carries
  a one-line cross-reference. (This shrinks the corpus and removes a class
  of drift bugs.)
- Move every `Examples` section >60 lines from a SKILL.md to
  `references/examples-<skill>.md`. The convention already supports this.
- Remove `grep` and `search_graph` from `base.yaml`'s `tools` list so they
  no longer propagate to every inheriting role by default. Roles that
  legitimately need them re-add them explicitly.

Target outcome: corpus reduced from ≈11,400 lines to ≈6,000 lines; no
contradictions; no duplicated anti-patterns; no over-budget skills.

#### B5 — Discover: surfaces the runtimes actually look for

- **`.claude/skills/`.** Add Anthropic-format skill wrappers (single-line
  `description`, correct frontmatter) for the highest-leverage subset of
  `.kbz/skills/` and `.agents/skills/` (`orchestrate-development`,
  `implement-task`, `kanbanzai-getting-started`, `kanbanzai-workflow`,
  `write-spec`, `write-design`, `review-code`). Each wrapper is a thin
  pointer to the canonical skill plus the frontmatter Claude needs for
  auto-surfacing. Decision §D3 captures the symlink-vs-copy choice.
- **`OPENAI.md`.** A two-line redirect to `AGENTS.md` for any GPT-class
  runtime that probes for it. Trivial cost; closes a discoverability hole.
- **Tool-description rule injection.** The `next`, `handoff`,
  `spawn_agent`, `dispatch_task`, `entity`, `worktree`, and `pr` tool
  descriptions are extended to embed the relevant rule (e.g. `next`'s
  description gains "the orchestrator must dispatch via `handoff`, not
  `spawn_agent`"). This disproportionately helps DeepSeek, which the audit
  shows over-trusts tool descriptions relative to skill markdown.
- **DeepSeek host check (operational, not code).** Document the host's
  system-prompt loading behaviour in `refs/sub-agents.md`. The audit notes
  this may resolve most DeepSeek symptoms with no code change at all.
- **`.cursor/rules/` shim.** One `.mdc` file pointing at `AGENTS.md`.
  Listed as Could; included only if B5 has slack capacity.

### Data flows

```
       role YAML  ─┐
                   ├─►  Constraint Card Renderer  ─►  next/handoff response
constraints.yaml ─┘                                   (prepended block)

stage-bindings.yaml ──► Stage Hydrator ──► next/handoff response
                                            (inlined binding payload)

stage-bindings.yaml ─┐
                     ├─► Registry Generator ─►  CLAUDE.md (table region)
.kbz/roles/*.yaml ──┘                            copilot-instructions.md (table)
                                                 README.md (table region)
                                          └─►  CI: `make registry-check`
```

### Why this addresses the audit findings

- **§3 indirection** — B1 inlines the stage binding; the agent no longer
  needs the four-hop chain to reach an operational rule.
- **§3 volume defeats U-curve** — B1's fixed ~25-line card sits in the
  recency peak; B4 cuts the corpus by ≈50% so selective loads fit the
  attention budget.
- **§4 multi-file drift** — B3 derives the registry from one source; B4
  resolves the live contradiction.
- **§5 DeepSeek** — B5 puts rules into tool descriptions (which DeepSeek
  trusts) and into the constraint card (which lands in the recency peak,
  where DeepSeek attends most reliably).
- **§6A high-violation rules** — B2 moves them into tool semantics.
- **§6C Anthropic Skills** — B5 adds `.claude/skills/`.
- **§6D corpus cut** — B4.
- **§6E multi-file drift** — B3.
- **§6F OPENAI.md** — B5.
- **§6G stage-bindings in responses** — B1.

## Alternatives Considered

### A1. Do nothing (status quo)
- **Trade-offs:** zero engineering cost; preserves the current
  authoring-friendly file layout.
- **Why rejected:** the audit and the P50/P55/P56/P57/P58 evidence trail
  document compounding incident cost from non-compliance. The status quo's
  cost is paid in production incidents, not engineering hours.

### A2. Single mega-prompt (collapse all roles/skills into one file)
- **Trade-offs:** removes the indirection problem in one stroke. Easy to
  load.
- **Why rejected:** would push the prompt well past the 15–40% context-
  utilisation sweet spot the project's own research cites. Loses the
  selective-load benefit. Re-creates the U-curve degradation the corpus
  structure was designed to avoid.

### A3. Move everything into MCP tool semantics
- **Trade-offs:** strongest enforcement; rules become invariants rather
  than advice.
- **Why rejected:** much of the corpus is *guidance* (vocabulary,
  anti-patterns, examples), not *invariants*. Encoding guidance as tool
  refusals would over-constrain the agent and strip the project of the
  authoring discipline it currently has. The chosen design moves only the
  five high-violation rules into tool semantics and leaves guidance in
  prose.

### A4. Replace the YAML/SKILL.md format with Anthropic Skills natively
- **Trade-offs:** unifies on one format; gets Claude auto-discovery for
  free.
- **Why rejected:** Anthropic Skills format is Claude-specific; adopting
  it as the canonical format would worsen the position with GPT and
  DeepSeek. The chosen design uses `.claude/skills/` as a *thin discovery
  surface* over the existing canonical files (B5), keeping the YAML/SKILL.md
  format as the source of truth.

### A5. Push rules into the system prompt only (no tool-description changes)
- **Trade-offs:** simplest path; one place to edit.
- **Why rejected:** the audit shows DeepSeek under-attends to long system
  prompts and over-trusts tool descriptions. System-prompt-only injection
  helps Claude and GPT but barely moves DeepSeek. The chosen design uses
  both surfaces.

### A6. Generate `AGENTS.md` from the canonical source (along with the others)
- **Trade-offs:** complete unification; eliminates AGENTS.md drift.
- **Why deferred (not rejected):** `AGENTS.md` is more narrative than the
  other top-level files and a generator that round-trips it well is more
  work than the cleanup it would save in this round. B3 generates the
  three less-narrative files first; bringing `AGENTS.md` into the
  generator is in scope for a follow-on plan once the pattern is proven.

## Decisions

### D1. The `stash` contradiction is resolved by forbidding stashing.
- **Decision.** Project-wide rule: never `git stash`. Use worktrees or
  commits.
- **Context.** `CLAUDE.md` allows "commit or stash previous work first";
  `kanbanzai-getting-started` forbids stashing.
- **Rationale.** Worktrees exist precisely to avoid the lost-work risk of
  `git stash`. The skill's stricter rule is the one rooted in workflow
  discipline; the `CLAUDE.md` text is a relic from before worktree
  adoption.
- **Consequences.** `CLAUDE.md` (and the generated copies in B3) align on
  the no-stash rule. Users who relied on the looser wording need one
  transitional release-notes line.

### D2. The constraint card is generated, not hand-written.
- **Decision.** A renderer composes the card programmatically from typed
  YAML inputs.
- **Context.** Hand-written cards drift from role YAML the moment a role
  changes.
- **Rationale.** A generator over typed YAML inputs cannot diverge by
  accident. Golden tests pin the rendered output.
- **Consequences.** Adds a small renderer module and a `constraints.yaml`
  registry. Agents see a stable, version-controlled card on every task
  claim.

### D3. `.claude/skills/` entries are **copies**, not symlinks.
- **Decision.** Generate `.claude/skills/<skill>/SKILL.md` files; do not
  symlink.
- **Context.** Symlinks would be DRY but are not uniformly tolerated by
  shell environments, Windows hosts, or CI runners that materialise
  archives.
- **Rationale.** Copies are portable; their content is generated from the
  canonical skill so the DRY property is preserved at build time. The CI
  guard from B3 covers divergence.
- **Consequences.** Each `.claude/skills/<skill>/SKILL.md` is generated;
  the canonical file remains under `.kbz/skills/`. The wrapper carries
  Anthropic-specific frontmatter (`name`, single-line `description`) plus
  a one-line "see canonical:" pointer.

### D4. Tool-description rule injection is mandatory; system-prompt-only is not enough.
- **Decision.** B5 edits the seven listed tool descriptions to embed the
  relevant rule text.
- **Context.** Audit §5 quantifies DeepSeek's tool-description trust vs.
  markdown trust.
- **Rationale.** A multi-surface approach hits all three runtimes' trust
  profiles. The cost of editing tool descriptions is small; the upside on
  DeepSeek is large.
- **Consequences.** The seven listed tool descriptions grow by 1–3 lines
  each. The descriptions are reviewed for tokens-per-rule efficiency.

### D5. `spawn_agent` is removed from the orchestrator's tools list; a `dispatch_task` wrapper enforces handoff-only dispatch.
- **Decision.** Orchestrator role no longer lists `spawn_agent`;
  `dispatch_task` becomes the canonical dispatch tool and routes through
  `handoff` internally.
- **Context.** The most-violated rule in the corpus is "always use
  `handoff`, never `spawn_agent` directly." Prose has not stopped this.
- **Rationale.** Tool removal is the only intervention that cannot be
  bypassed by an agent ignoring prose. `dispatch_task` retains the
  ergonomics of `spawn_agent` while enforcing the constraint.
- **Consequences.** Coordinated with P44. `dispatch_task` becomes the
  documented dispatch tool; `spawn_agent` remains available to non-
  orchestrator roles that genuinely need it (e.g. researcher fan-out).

### D6. The corpus cut targets ≈50% reduction without losing rules.
- **Decision.** Target ≈6,000 lines total skill text. Achieve via
  de-duplication and example relocation, not by trimming rules.
- **Context.** The audit's claim that ~6,000 lines is achievable.
- **Rationale.** Most of the savings come from de-duplication and example
  relocation, both of which preserve every rule. Trimming to budget is a
  byproduct of those moves rather than a separate "rewriting" exercise.
- **Consequences.** Skill author discipline tightens. The
  `references/examples-*.md` directory becomes the canonical home for
  long examples.

### D7. The generator covers `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md` in this round; `AGENTS.md` is deferred.
- **Decision.** B3 ships the generator for three files, not four.
- **Context.** `AGENTS.md` is more narrative and round-tripping it
  cleanly is more engineering than this round needs.
- **Rationale.** Land the pattern on the three lowest-narrative files
  first. Re-evaluate including `AGENTS.md` once the marker-region
  approach is proven.
- **Consequences.** `AGENTS.md` remains the highest-drift file in the
  short term, mitigated by the human-author discipline already documented
  in `kanbanzai-workflow`.

### D8. P59 owns the *prompt-layer* changes for handoff-only dispatch and gate enforcement; P44 owns the *tool-implementation* changes.
- **Decision.** B2 specifies and documents; the implementing features
  live in (or are co-scheduled with) P44.
- **Context.** Audit §6A flags the overlap with P44.
- **Rationale.** Splitting on layer keeps each plan's review surface
  small. P44 implements; P59 supplies the rule list, removes the
  redundant prose, and updates tool descriptions.
- **Consequences.** B2 carries an explicit dependency on the relevant P44
  features and ships when those features ship. The dev-plan (next stage)
  must enumerate the P44 entities.

## Dependencies

### Internal

- **P44 — Architectural enforcement.** B2 depends on P44 features for the
  handoff-only dispatch enforcement, the entity-existence refusal in
  `next`/`handoff`, and the commit-before-task gate. P59's dev-plan must
  list the specific P44 features it consumes.
- **P56 — Bug stage-gate enforcement (in flight).** B2 rule #5 (gate
  prerequisites for bugs) lands when P56 lands; P59 carries only the
  prose cleanup.
- **`kanbanzai serve` MCP server.** B1's renderer plugs into the existing
  response-composition path used by `next` and `handoff`.
- **`stage-bindings.yaml` and `.kbz/roles/*.yaml`.** B3's generator treats
  these as canonical; their schema is already stable.

### External

- **Anthropic Skills format spec.** B5's `.claude/skills/` wrappers must
  match the published frontmatter spec at the time of implementation.
  The spec has been stable since October 2024 but should be re-verified
  during dev-planning.
- **Codex/AGENTS.md convention.** `OPENAI.md` is a redirect to
  `AGENTS.md`; both are unchanged conventions of the host runtimes.
- **DeepSeek host configuration.** B5's documentation step depends on
  knowing which host (OpenCode, Continue, custom MCP client) is loading
  DeepSeek and whether it injects `AGENTS.md`. This is operational, not
  code.

### Sequencing

```
B1 ──► B2*           (B2 depends on P44 features; B1 unblocks early wins)
   ╲
    ╲─► B3 ──► B5    (B5's generated .claude/skills/ uses B3's generator)
         │
         └─► B4      (B4 cleanup is safer once the generator exists)
```

`*` B2 may land in parallel with B3/B4 if the P44 dependencies are met.
B1 is independent and should ship first.

### Risks (high-level; full risk analysis belongs in the spec)

- **Generator round-trip fidelity.** If hand-edits to surrounding prose
  in `CLAUDE.md` are clobbered, contributors will lose trust in the
  generator. Mitigated by marker-comment regions and a `--check` mode that
  is mandatory in CI.
- **Constraint card token cost.** A 25-line block on every response is
  not free. Mitigated by keeping the card under a published byte budget
  enforced by tests, and by making the card omittable for `status`/`get`
  responses where it adds no value.
- **DeepSeek host opacity.** The audit's recommendation to verify the
  host's system-prompt loading is operational — if no host configuration
  is documented, B5's gains for DeepSeek are reduced. Treat the host
  documentation step as a hard prerequisite for declaring the DeepSeek
  improvements done.

## Open Questions

1. Does the constraint card belong on every MCP response, or only on
   those that deliver task/feature context? The design takes the second
   position; revisit during specification.
2. What is the exact contents of `constraints.yaml`? B1 needs the top-N
   list pinned. Proposed list is the five from B2 plus "do not read
   `.kbz/state/` with shell tools" and "verify before assuming"; final
   list belongs in the spec.
3. Should the `.claude/skills/` wrappers cover all skills or only the
   high-leverage subset listed in §B5? Current design says subset; full
   coverage can follow if the subset proves valuable.
4. Generator language and runtime: a Go subcommand of `kanbanzai`, a Make
   target shelling out to a script, or a build-time hook? Defer to
   dev-planning.
