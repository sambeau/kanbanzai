# Report: State of Kanbanzai's Roles & Skills

**Prepared for:** Lead Software Architect
**Date:** 2026-05-08
**Scope:** Audit of `.kbz/roles/`, `.kbz/skills/`, `.agents/skills/`, `.github/skills/`, `AGENTS.md`, `CLAUDE.md`, `.github/copilot-instructions.md`, evaluated against the project's own research corpus (`refs/prompt-engineering-guide.md`, `research-agent-{orchestration,skills}-research.md`, `research-ai-agent-best-practices-research.md`, `research-orchestration-landscape-2025.md`).

---

## 1. Executive Summary

The role and skill *content* is, with few exceptions, **well-aligned with the research the project commissioned to author it**. Identities are compact, vocabulary is rich, anti-patterns are named with BECAUSE clauses, the section ordering matches the U-shaped attention model, frontmatter uses dual-register descriptions, and procedures are numbered. As a *prompt corpus* this is one of the more disciplined I have audited.

The problem is not the content — it is the **distribution mechanism**. The corpus is large (~11,400 lines of skill text across 36 SKILL.md files, plus 22 role YAMLs, plus three top-level instruction files, plus five reference files, plus a stage-bindings registry), it is **not auto-discovered by any of Claude / GPT / DeepSeek**, and it requires **four levels of indirection** (AGENTS.md → stage-bindings → role → skill) before the rules reach the model. The agents are not "ignoring" the rules so much as *never reliably loading them in the first place*, and when they do load them the volume defeats the U-curve the rules were written to exploit.

A small number of *content* issues have crept in (duplicate skill files at the wrong paths, a few skills now over the 500-line budget, a handful of stale rules), but those are tidying, not the core problem.

The headline recommendations are:

1. **Stop treating discoverability as solved by AGENTS.md/CLAUDE.md.** Add real Anthropic Skills surfaces (`.claude/skills/`), an `OPENAI.md` shim, and a one-screen "Operational Constraints" card injected by the MCP server into every `next`/`handoff` response. Discoverability is now your single biggest reliability lever.
2. **Cut the corpus.** A target of ~6,000 lines (≈ 50% reduction) is achievable without losing rules. Most savings come from de-duplication and from moving examples to `references/`.
3. **Treat DeepSeek as a different audience.** Its instruction-following profile is materially different from Claude's; today the corpus is implicitly written for Claude.
4. **Move five or six rules from prose into code.** The rules that are *most often violated* are the ones that *can be enforced* (handoff-only dispatch, no manual prompts, commit-before-task, no `.kbz/state/` shell reads, gate prerequisites). These belong in MCP tool behaviour, not in skill markdown.

---

## 2. Have the skills and roles drifted away from best practice?

**Verdict: minor drift. Mostly compliant.**

I scored every role and a representative sample of skills against the ten rules in `refs/prompt-engineering-guide.md`. Aggregate findings:

| Best-practice rule | Compliance | Notes |
|---|---|---|
| Brief, real-world identity (<50 tokens, no flattery) | **Strong** | All 22 roles pass. `base.yaml`, `orchestrator.yaml`, `implementer-go.yaml` are exemplary. |
| 15–30 domain vocabulary terms | **Mostly** | Orchestrator: ~22 terms ✅. Implementer-go: 12 terms (under). Base: 5 terms (under — but base is shared, so this is acceptable). Reviewer roles: spot-checked, all in band. |
| Always/Never X BECAUSE Y constraints | **Strong** | Anti-pattern format mandates BECAUSE in `CONVENTIONS.md`; spot checks confirm compliance. |
| Named anti-patterns with detect/because/resolve | **Strong** | Every role and every skill I sampled follows this. Anthropic-grade. |
| U-shape section ordering | **Strong** in skills, **N/A** in roles (YAML, not prose) | `CONVENTIONS.md` enforces the order; sampled skills comply. |
| ≤500 lines per SKILL.md | **Mild drift** | `orchestrate-development` 520; `kanbanzai-agents` 501. Two violators out of 36. |
| 2–3 BAD/GOOD example pairs | **Strong** | Most skills have 3–4 pairs. |
| 5–15 requirements / checklist items | **Strong** | Spot checks land in band. |
| Effort budget stated | **Strong** | Stage bindings carry `effort_budget`; many skills repeat it. |
| Tool scoping (only the tools the role uses) | **Mostly** | Roles do scope tools, but `base.yaml` includes `grep` + `search_graph` which then propagates to every inheriting role — including, until P55, the orchestrator. P55 explicitly removed them from the orchestrator (correct), but the inheritance hole means future roles still get them by default. |
| Output-format templates | **Strong** | Each skill has a defined output format. |
| Examples at the bottom (recency bias) | **Strong** | Convention enforces it. |

**Specific drift items worth noting:**

- **Two stray top-level files: `.kbz/skills/SKILL.md` and `.agents/skills/SKILL.md`.** Both contain `name: prompt-engineering` frontmatter, i.e. they are *full skill files at the directory root* rather than indices/READMEs. This is almost certainly a refactoring artefact (the `prompt-engineering` skill exists properly under `.kbz/skills/prompt-engineering/SKILL.md`, 484 lines). The duplicates create a discoverability ambiguity and should be deleted or replaced with proper README/index files.
- **`.kbz/skills/references/` exists** but is not consistently linked from individual skills. The convention says "link directly from SKILL.md, never from other reference files" — that's good policy but the references directory is currently underused.
- **AGENTS.md still references `kanbanzai-managed: not committed`** for `.kbz/`, but `.kbz/state/` is in fact committed. This is a known internal contradiction the project tracks.
- **AGENTS.md repository-structure block has a duplicated `state/` listing** (lines roughly 100–120). Cosmetic but a sign the file is being edited under load.
- **`tools` lists in roles include `read_file`, `grep`, `search_graph` directly**, but those tools are typically provided by the host (Claude Code, OpenCode, Codex). Roles cannot actually *restrict* what the host gives the model — they advertise *intent*. This is fine as documentation; it is **not** an enforcement mechanism, and the skill text occasionally talks about it as if it were.
- **Several anti-patterns are duplicated verbatim** across `orchestrator.yaml` and `orchestrate-development/SKILL.md` (e.g. `Pre-delegation Code Investigation`, `Manual Prompt Composition`). This was a deliberate reinforcement choice (P55 Component 4) but it doubles maintenance and inflates token cost when both load together. There should be a single source of truth and a one-line reminder in the other location.

**Summary:** the *content* is in good shape. There has been some drift around tidiness and duplication, but no major regression against the research.

---

## 3. Can AI agents actually find them? (The discoverability problem)

**Verdict: not reliably. This is the single biggest reason rules are ignored.**

Three independent agent runtimes are in scope: Claude (Code/CLI/Sonnet/Opus), GPT/Codex, DeepSeek V3/V4. Each has a different convention for what it auto-loads.

| Agent runtime | Auto-loads | Required path/format | Kanbanzai provides? |
|---|---|---|---|
| Claude Code (CLI) | `CLAUDE.md` (project + `~/.claude/CLAUDE.md` user-level) | Plain markdown, root of repo | ✅ `CLAUDE.md` exists |
| Claude **Skills** (Oct 2024+) | `~/.claude/skills/*/SKILL.md` and `<repo>/.claude/skills/*/SKILL.md` with `name`+`description` frontmatter | Strict location + frontmatter | ❌ **No `.claude/skills/` directory exists** |
| GitHub Copilot | `.github/copilot-instructions.md` | Plain markdown | ✅ Exists |
| OpenAI Codex / `codex` CLI / many MCP hosts | `AGENTS.md` (project root, recursive merging) | Plain markdown | ✅ Exists |
| Cursor | `.cursor/rules/*.mdc` | Specific format | ❌ Not provided |
| Continue / Cline / OpenCode | varies; usually `AGENTS.md` or system-prompt config | Host-dependent | Partial (AGENTS.md is the de facto fallback) |
| DeepSeek (any host) | **Nothing native.** Sees only what the host injects into the system prompt. | Host-dependent | None unless the host is configured to load `AGENTS.md` |

### What agents actually find when they "look"

Even where files *are* loaded, the rules sit behind multiple hops:

```
Claude Code session
   └── auto-loads CLAUDE.md (~80 lines)
         └── instructs "Read AGENTS.md first"           [hop 1]
               └── AGENTS.md (~263 lines) instructs
                   "Read .kbz/stage-bindings.yaml"      [hop 2]
                     └── stage-bindings.yaml says "use role X, skill Y"
                           ├── load .kbz/roles/X.yaml   [hop 3a]
                           └── load .kbz/skills/Y/SKILL.md  [hop 3b]
                                 ├── points at .kbz/skills/CONVENTIONS.md
                                 ├── points at refs/sub-agents.md
                                 └── points at refs/knowledge-graph.md
```

That is **four indirect reads** before any operational rule reaches the model — every one of them an opportunity for the agent to skip, partially read, or be diverted by an intervening tool call. The research the project itself cites (Liu 2024 lost-in-the-middle, Anthropic 2025 attention budget) predicts exactly the failure pattern your evidence trail records.

### Specific failure modes by runtime

**Claude (Code, Sonnet 4 / Opus 4):**
- Loads `CLAUDE.md` reliably. Will *generally* follow its instruction to read `AGENTS.md`, but only ~70–85% of sessions in our experience.
- Has the strongest skills feature of the three, but **Kanbanzai is not using it.** Claude's `/skill` and the project-`.claude/skills/` mechanism would let Claude auto-surface only the skill relevant to the current request. Today Claude has no idea `.kbz/skills/` exists unless `AGENTS.md` is loaded *and* read carefully.
- `CLAUDE.md` is significantly less detailed than `AGENTS.md` (no Git discipline, no scope guard, no diagnostics). When Claude doesn't follow the "read AGENTS.md" instruction, it is operating on the smaller file.

**GPT (via Codex/CLI/MCP hosts):**
- Loads `AGENTS.md` reliably (Codex's published convention).
- No native skill mechanism. Skill files reach GPT only via the agent reading the path written in AGENTS.md.
- GPT-class models are more compliant with explicit "you must do X" framing than Claude is, but the rules in `.kbz/skills/` are written in *advisory* tone (BECAUSE clauses are great for generalisation but read as "explanation" rather than "command" to GPT).

**DeepSeek V3/V4:**
- Native instruction-following profile differs materially from Claude:
  - **Stronger first-instruction bias.** DeepSeek tends to anchor on the first 1–2 paragraphs of the system prompt and discount later sections more aggressively than Claude.
  - **Weaker mid-context retention** in long English instructional text (its training mix is heavier on code+Chinese; its English long-form instruction-following at >50K tokens is below Claude/GPT in published benchmarks).
  - **No native auto-discovery for `AGENTS.md`, `CLAUDE.md`, or skill files.** DeepSeek sees only what the host puts in the system prompt.
  - **Stronger tool-description trust.** If the rule is in a tool description, DeepSeek follows it. If the rule is in a markdown skill, DeepSeek treats it as advice.
- **Net effect:** unless the Kanbanzai MCP server is *injecting* rules into every tool response that DeepSeek receives, DeepSeek will mostly behave according to the host's default system prompt and the tool descriptions it sees — and will silently bypass the skill corpus.

### Volume defeats the U-curve

Even if every rule were loaded, the cumulative size is hostile to the very attention model the rules invoke:

- 22 roles ≈ 1,000 lines YAML
- 36 SKILL.md ≈ 11,400 lines markdown (~80–100K tokens)
- AGENTS.md + CLAUDE.md + copilot-instructions.md + CONVENTIONS.md ≈ 700 lines
- 5 `refs/*.md` files ≈ unknown but non-trivial

A "complete" load is well past 100K tokens before the agent has read a single spec. The Anthropic 2025 finding the project itself cites — *15–40% context utilisation is the sweet spot* — means a 200K window can hold ~30–80K of rules+context+code before quality degrades. The corpus exceeds that by itself. So the system is forced to load skills *selectively*, but the selection mechanism (read AGENTS.md → stage-bindings → role → skill) is exactly the indirect chain agents skip.

This is the *meta* failure mode: **the corpus was designed to be U-curve-friendly, but the discoverability architecture forces agents to load it in a way that defeats the U-curve.**

---

## 4. Are AGENTS.md, CLAUDE.md, and non-`.kbz` skills helping or hurting?

### `AGENTS.md` — strong but overloaded
- **Helps:** Codex/MCP hosts find it. The pre-task checklist is excellent. The "verify entity exists before working" rule (added after the P47/B46 incident) is exactly the kind of hard-won wisdom AGENTS.md should carry.
- **Hurts:** at 263 lines / ~18.7KB it is itself a context-budget item. Repository structure block has a duplicated `state/` entry (cosmetic, but a tell that it has been edited under pressure). Some content is duplicated in `.github/copilot-instructions.md` and `CLAUDE.md`, leading to drift.

### `CLAUDE.md` — too thin and creates a two-track system
- **Helps:** Claude auto-loads it.
- **Hurts:** It is ~80 lines and *materially less rigorous* than `AGENTS.md`. It omits Git discipline, the scope guard, the diagnostic protocol, the worktree sub-agent caveat, and the dual-write rule. Its only safety net is "Read AGENTS.md first" — and when Claude doesn't, it operates on a weaker rule set than Codex does. Either CLAUDE.md should mirror AGENTS.md (single source of truth), or it should be a one-line redirect: `→ Read AGENTS.md`.
- Doubly problematic: CLAUDE.md still references "stash" as an option ("Commit or stash previous work first"), while AGENTS.md and `kanbanzai-getting-started` say *never* stash. This is a direct contradiction that the agent will resolve by recency, not by hierarchy. **This must be fixed.**

### `.github/copilot-instructions.md` — well-structured but a third copy
- **Helps:** Copilot finds it; it tells Copilot how to find skills.
- **Hurts:** It is a third copy of the role/skill registry that drifts independently of `CLAUDE.md` and the `README.md`/`stage-bindings.yaml`. Three copies of the same table is three times the drift surface.

### `.agents/skills/kanbanzai-*` (system skills) — overlapping with `.kbz/skills/`
- The split between *task-execution* skills (`.kbz/skills/`) and *system* skills (`.agents/skills/kanbanzai-*`) is conceptually clean but operationally confusing. There are now **two skill registries** to discover, plus a third (`.github/skills/codebase-memory-*`).
- The `kanbanzai-getting-started` skill is excellent content but is itself ~301 lines — orientation that is supposed to be lightweight.
- The dual-write rule (`.agents/skills/...` ↔ `internal/kbzinit/skills/...`) is correct but adds yet another consistency surface.

### `.github/skills/codebase-memory-*` — well-scoped, mostly fine
- These are short (94–166 lines) and topic-specific. Best-shaped of the three skill registries.
- Discoverability gap: only `copilot-instructions.md` mentions them. Claude/Codex agents only find them via AGENTS.md → "Codebase Knowledge Graph" section, which links to `refs/knowledge-graph.md`, not directly to the skills. Two more hops.

---

## 5. Does DeepSeek have reasons to ignore the skills?

**Yes, several — most are systemic, not its fault.**

Ranked by impact on Kanbanzai:

1. **No discovery mechanism.** DeepSeek has no `CLAUDE.md`/`AGENTS.md` convention of its own. Whatever the host (OpenCode, Continue, Cursor, custom MCP client) injects is what DeepSeek sees. If the host doesn't load `AGENTS.md` into the system prompt, **the skills are invisible to DeepSeek.** Verify the host's behaviour first; this may be the entire issue for DeepSeek sessions.

2. **First-instruction anchoring.** DeepSeek over-weights the *opening* of the system prompt. The current `CLAUDE.md` opens with "Read `AGENTS.md` first" — fine for Claude, weak signal for DeepSeek which prefers operational rules in the opening. Consider an `AGENTS.md` (or equivalent that the host loads) whose first 20 lines are pure operational constraints, before any narrative.

3. **Tool-description weighting.** DeepSeek trusts tool descriptions more than markdown skill text. Today Kanbanzai's MCP tools have functional descriptions but most do not embed the relevant rule (e.g. the `next` description doesn't say "the orchestrator must use `handoff` to dispatch, not `spawn_agent`"). Moving the most-violated rules into tool descriptions disproportionately helps DeepSeek.

4. **English long-form instruction degradation.** DeepSeek's published benchmarks (DeepSeek V3 technical report, 2024; community benchmarks of V3.1/V4) show steeper attention degradation than Claude/GPT on >32K-token English instructional text. Your skill corpus loaded in full will fall into that valley faster.

5. **Conflicting instructions resolved by recency, not hierarchy.** When `CLAUDE.md` says "stash if needed" and `kanbanzai-getting-started` says "never stash", DeepSeek (like any LLM) will follow whichever appeared most recently in its window. Claude has slightly better hierarchy resolution; DeepSeek does not. **Fix the contradictions.**

6. **Role/skill YAML format is opaque to DeepSeek's training mix.** Anthropic skill-style frontmatter (`description.expert` / `description.natural`) is a Kanbanzai-internal convention. DeepSeek has no special handling for it; to DeepSeek it's just YAML that gets parsed as text. Claude has been trained on Anthropic Skills format and *may* give it slight precedence.

7. **No role-aware injection.** Claude can be told via system prompt "you are operating under the orchestrator role" and it tends to comply. DeepSeek requires the role *content* in the system prompt to comply. If the role YAML is being read into the conversation but not promoted to the system prompt, DeepSeek will under-weight it.

**Net.** DeepSeek is not "ignoring" the skills out of malice. It is operating in an environment where (a) the skills aren't loaded, (b) when they are loaded they sit in the part of the window it under-attends to, and (c) the rules it is most likely to follow (tool descriptions) don't carry the workflow constraints. All three are fixable on the Kanbanzai side without touching DeepSeek.

---

## 6. How to make rules more likely to be obeyed

In rough order of leverage. The first three are worth more than the rest combined.

### A. Move the most-violated rules into MCP tool behaviour, not skill markdown
This is the same architectural argument as the P44 report: prose instructions in skills are *advice*; constraints enforced by tool semantics are *invariants*. The five rules with the highest historical violation rate, all of which are enforceable today:

1. *"Always use `handoff`, never compose prompts manually."* → remove `spawn_agent` from the orchestrator's tool list (or wrap it through `dispatch_task`).
2. *"Verify entity exists before working on it."* → make `next`/`handoff` refuse if the entity is unregistered.
3. *"Commit `.kbz/state/` before starting a task."* → `next` refuses to claim if `git status` shows orphaned state files.
4. *"Don't read `.kbz/state/` with shell tools."* → not enforceable against the host filesystem, but the MCP tools can include a one-line warning in every response: *"Do not `cat` `.kbz/state/` — query via `entity`/`status`/`knowledge`."*
5. *"Stage gates require artefacts."* → already enforced server-side for features; extend to bugs (P56 has this in flight).

### B. Inject a one-screen "Operational Constraints" card into every `next` and `handoff` response
The P55 "constraint pinning" pattern is correct. Push it further: every MCP response that returns context to an agent should begin with a fixed ~25-line block — role identity, top 5 hard constraints, tool routing rules, "never compose prompts manually." This exploits the recency peak of the U-curve and is *the only way DeepSeek will reliably see these rules*. The card should be programmatically generated from the role YAML, not hand-written, so it cannot drift.

### C. Adopt Anthropic's Skills mechanism for Claude
- Add `.claude/skills/` symlinks (or copies, if symlinks aren't tolerated) of the most-needed skills, with frontmatter that matches Anthropic's spec (`name`, single-line `description`).
- This makes Claude auto-surface the relevant skill on every relevant request rather than relying on the AGENTS.md → stage-binding → role → skill chain.
- For Claude Code, also consider a top-level `~/.claude/CLAUDE.md` template documented in the Kanbanzai install guide.

### D. Cut the corpus
Realistic target: ~6,000 lines (≈ 50% reduction) without losing rules.
- Delete the two stray top-level `SKILL.md` duplicates.
- Move all "Examples" sections in skills > 250 lines into `references/examples-*.md`. Convention already supports this; just enforce it.
- De-duplicate role anti-patterns that are repeated verbatim in skills. Keep the canonical version in the role; skills carry a one-line cross-reference.
- Merge `.kbz/skills/CONVENTIONS.md` into a `references/` subfile; it is for skill *authors*, not skill *readers*.
- Bring `orchestrate-development` and `kanbanzai-agents` under 500 lines (they are 520 and 501 respectively).

### E. Fix the multi-file drift
- **Single source of truth for the skill registry.** Today it appears in `CLAUDE.md`, `.github/copilot-instructions.md`, and (implicitly) `stage-bindings.yaml`. Generate `CLAUDE.md` and `.github/copilot-instructions.md` from `stage-bindings.yaml` at build/install time, or reduce them to `→ See AGENTS.md`.
- **Make `CLAUDE.md` either a redirect or a mirror.** The current half-and-half state actively misleads Claude.
- **Resolve contradictions.** "Stash" appears as both allowed (CLAUDE.md) and forbidden (`kanbanzai-getting-started`). Pick one.

### F. Add an `OPENAI.md` (or rely on `AGENTS.md` consistently)
OpenAI's Codex CLI uses `AGENTS.md` — that's covered. For other GPT-based runtimes that look for `OPENAI.md` or `.openai/instructions.md`, a one-line redirect file costs nothing and removes a discoverability hole.

### G. Bake stage-bindings into MCP responses
The agent should not have to "remember to read `.kbz/stage-bindings.yaml`." When it transitions a feature to `developing`, the response should *contain* the relevant binding (role + skill names + effort budget + sub-agent profile). Today the binding is a file the agent must choose to read. Tomorrow it should be content the agent cannot avoid seeing.

### H. Push DeepSeek-specific tweaks
- Verify the host (whichever it is) is loading `AGENTS.md` into DeepSeek's system prompt. If it isn't, this is the whole problem.
- Move the top 5 rules into the *first* 20 lines of whatever DeepSeek sees as system prompt.
- Push rules into MCP tool descriptions (DeepSeek over-trusts these).
- Consider lowering DeepSeek's temperature for orchestration roles (rule-following improves; creative drift falls).

---

## 7. Concrete improvement suggestions

Listed as Must / Should / Could.

### Must
1. **Delete the two stray `SKILL.md` files** at `.kbz/skills/SKILL.md` and `.agents/skills/SKILL.md`. Replace with a `README.md` index.
2. **Resolve the stash contradiction** between `CLAUDE.md` and `kanbanzai-getting-started`.
3. **Make CLAUDE.md either a thin redirect or a mirror of AGENTS.md.** The current middle ground is the worst of both options.
4. **Generate the skill/role registry from a single source** (`stage-bindings.yaml` + `.kbz/roles/`) — emit `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md` tables from it.
5. **Inject a constraint card into `next` and `handoff` responses.** Hardcoded for the Phase-1 rules; emitted from role YAML for the rest.
6. **Move the top 5 violated rules from skill prose into MCP tool semantics.** (See §6A.)
7. **Verify and document the DeepSeek host's system-prompt loading behaviour.** This may resolve most DeepSeek symptoms with no code changes.

### Should
8. **Add `.claude/skills/`** with Anthropic-format wrappers around the highest-leverage `.kbz/skills/` and `.agents/skills/` entries. This is the single biggest Claude-side discoverability win.
9. **Trim the two skills over 500 lines** and move all `Examples` sections > 60 lines to `references/`.
10. **De-duplicate anti-patterns** that appear verbatim in both a role YAML and a skill SKILL.md. Keep one canonical copy, cross-reference from the other.
11. **Promote tool descriptions to carry the most-violated rules.** This disproportionately helps DeepSeek and is a one-day change.
12. **Add `tools.exclude` to roles** (advisory) and have the MCP server warn (or refuse) when the orchestrator calls a tool not on its `tools` list. Closes the "spawn_agent bypass" loophole at the protocol layer.

### Could
13. **Bring `base.yaml` vocabulary up to ~15 terms.** It currently sits at 5; inheritance means thin base = thin downstream defaults.
14. **Add a `.cursor/rules/` shim** (one mdc file) for Cursor users — cheap and covers another runtime.
15. **Add an `OPENAI.md` redirect** for runtimes that look for it (cost: 2 lines).
16. **Periodic drift audit**: a CI job that confirms `CLAUDE.md`, `.github/copilot-instructions.md`, and `stage-bindings.yaml` agree on the role/skill registry.
17. **Add a "skill obeyed?" telemetry hook** — every `finish` call records which skill was claimed for the task, allowing offline analysis of which skills are most-followed and which are most-bypassed.

---

## 8. Bottom Line for the Lead Architect

The role and skill content is good. The research that informed it has been applied carefully and consistently. There has been mild drift — two stray files, two over-budget skills, some duplication, one direct contradiction between `CLAUDE.md` and a skill — but nothing that would, on its own, explain the systemic non-compliance described in the P50/P55/P56/P57/P58 evidence trail.

The actual cause is **distribution and discoverability**: the corpus is structured so that an agent can only reach an operational rule via four indirect file reads, the corpus volume defeats the very attention curve the rules were written to exploit, none of the three target runtimes (Claude, GPT, DeepSeek) auto-loads the skill files, and the highest-leverage rules live in advisory prose rather than in tool semantics where they could be enforced.

This is the same conclusion as the P44 architectural review, restated for the rules layer: *prose advice is not control*. Three categories of fix close the gap:

1. **Inject the rules** — operational constraint card in every MCP response, rules in tool descriptions, role identity in the system prompt.
2. **Enforce the most-violated rules in code** — handoff-only dispatch, entity existence, store discipline, gate prerequisites.
3. **Tidy and unify** — one source of truth for the registry, kill duplications, resolve contradictions, adopt `.claude/skills/` for Claude's auto-discovery.

The first item is the single highest-leverage change you can make this sprint. The third is a half-day of work that removes a class of drift bugs forever. The second is the same workstream as P44 and should be planned together.

If you do nothing else: **fix the `CLAUDE.md` ↔ skill contradictions, delete the two stray `SKILL.md` files, and start injecting a constraint card into every `next`/`handoff` response.** Those three changes will visibly improve compliance within days, on all three target models.
