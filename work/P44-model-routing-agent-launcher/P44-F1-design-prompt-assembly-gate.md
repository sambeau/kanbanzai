| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07T14:21:22Z           |
| Status | Draft                          |
| Author | sambeau                        |

## Overview

This design proposes a **prompt assembly gate** — a non-bypassable quality check that P44's `dispatch_task` runs before dispatching any sub-agent. It addresses a systemic gap discovered in May 2026: sub-agent prompts are missing roles, skills, tool hints, vocabulary, anti-patterns, and any mention of `codebase-memory-mcp` or `kanbanzai_edit_file` — despite all these features being implemented in the 3.0 context assembly pipeline.

The root causes are: (1) tool hints are implemented but gated on configuration that doesn't exist, (2) hardcoded defaults were never added as a safety net, and (3) the orchestrator may be bypassing `handoff` and composing prompts manually from `next(id)` JSON output. P44's `dispatch_task` will make the pipeline invisible — this design ensures it's correct before that happens.

## Goals and Non-Goals

**Goals:**
- Guarantee every sub-agent prompt includes role identity, skill procedure, and tool guidance
- Add hardcoded default tool hints so `codebase-memory-mcp` and `kanbanzai_edit_file` appear in every implementer prompt
- Add assembly gate checks to `dispatch_task` that catch degraded prompts before dispatch
- Make `next(id)` return a rendered prompt field to reduce manual composition
- Zero configuration required — existing projects get the fix immediately

**Non-Goals:**
- Not modifying the 3.0 pipeline structure itself
- Not changing the `stage-bindings.yaml` schema
- Not removing the config override mechanism — config still wins over defaults
- Not building a prompter sub-agent step (evaluated as Alternative C, rejected for now)

## Problem and Motivation

Sub-agent prompts in Kanbanzai are missing critical features that were designed, built, and tested — but never reach the agents they're meant to guide. An investigation of the latest pipeline work (May 2026) found that sub-agent prompts are bare: no role identity, no vocabulary, no anti-patterns, no tool lists, no procedure guidance, and — critically — no mention of `codebase-memory-mcp` or the hash-based `kanbanzai_edit_file` tool.

### What's broken

**1. Tool hints are implemented but never populated.** The 3.0 context assembly pipeline (P51, `internal/context/pipeline.go`) has Position 6 ("Available Tools") that renders role-scoped tool hints via `stepResolveToolHint`. This function returns immediately when `MergedToolHints` is empty — which it always is, because no tool hints exist in `.kbz/config.yaml` and no hardcoded defaults exist in code. The `stepResolveToolHint` function and the entire tool hints merge infrastructure (`internal/config/tool_hints.go`) are dead code in every deployed project.

**2. No mention of `codebase-memory-mcp` or `kanbanzai_edit_file`.** These tools — built in P42 (hash-anchored edit tool) and the knowledge graph system — never appear in sub-agent prompts. Sub-agents don't know they can use them. The `implement-task` skill was updated to remove heredoc and recommend `kanbanzai_edit_file`, but without tool hints, sub-agents default to `edit_file` and `grep` — losing entity-scoped writes and structural code search.

**3. The orchestrator may be composing prompts manually despite the rule.** The `orchestrate-development` skill mandates using `handoff(task_id, role: "implementer-go")` for all sub-agent prompts. But the evidence (bare prompts with no pipeline sections) suggests the orchestrator reads `next(id)` JSON output and composes prompts by hand — losing all pipeline-assembled content. The pipeline runs unconditionally when `handoff` is called; bare prompts are a strong signal of bypass.

**4. P44 will make this invisible.** The P51 design §1.8.1 explicitly flagged this: "Pipeline becomes invisible after P44 — severity raised from Medium to High." When P44's `dispatch_task` replaces `handoff` as the dispatch mechanism, the pipeline will run silently inside the agent launcher. If the pipeline is producing degraded output today, nobody notices because the orchestrator sees the prompt. After P44, nobody sees the prompt — and degraded output propagates silently.

### Why this matters

Sub-agents without role guidance make avoidable mistakes. Sub-agents without tool hints don't discover the most effective tools. Sub-agents without vocabulary and anti-patterns produce inconsistent code. The investment in roles, skills, and tool hints is wasted if the content never reaches its target.

### Who is affected

Every feature in the developing stage. Every sub-agent dispatched by an orchestrator. This is systemic — not isolated to one feature or batch.

### Evidence log

This problem has been observed across multiple plans, persisting despite fixes to the pipeline itself:

| Date | Plan | Observation |
|------|------|-------------|
| 2026-05-03 | P51 | Legacy fallback silently degraded prompts. P51 removed the fallback, making the pipeline the only path — but orchestrators could still bypass `handoff` entirely. |
| 2026-05-04 | P55/P56 | Orchestrator dispatched sub-agents for bug lifecycle work. Prompts were bare: no role, no skill, no tools, no knowledge. Source traced to `next(id)` JSON being hand-assembled into prompts instead of using `handoff`. |
| 2026-05-07 | P57 | **Confirmed bypass despite P58 fix.** P58 added hardcoded default tool hints — verified working via direct `handoff` call. But all four P57 implementation prompts are manually composed from `next(id)` output: no Role section, no Vocabulary, no Anti-Patterns, no Available Tools, no Knowledge, no Evaluation Criteria. The orchestrator skill says "Always use `handoff`" — the orchestrator ignores it. |
| 2026-05-07 | P58 | P58 implemented and merged. Direct `handoff(task_id, role: "implementer-go")` now produces all 11 pipeline sections including Available Tools with `codebase_memory_mcp_search_code` and `kanbanzai_edit_file`. The pipeline is fixed — but the orchestrator still doesn't call it. |

**Root cause analysis:** The AI chat-based orchestrator receives `next(id)` which returns structured JSON context (spec sections, knowledge entries, file paths). It then composes a prompt by hand from that JSON. The `handoff` tool produces a complete rendered prompt, but the orchestrator has to (a) know about it, (b) choose to call it, and (c) forward its output to `spawn_agent`. At each step, the orchestrator can take a shortcut. The evidence shows it always does.

**Architectural implication:** The chat-based orchestrator is the wrong architecture for prompt assembly enforcement. A prompt gate that depends on agent discipline is not a gate — it's a suggestion. The gate must live in the dispatch mechanism itself, where the orchestrator cannot bypass it. This means P44's `dispatch_task` — a tool that runs the pipeline internally and sends the assembled prompt directly to the provider — is not an optimisation. It's the only architecture that works.

## Design

The core idea: **P44's `dispatch_task` must enforce prompt assembly as a non-bypassable gate, with hardcoded fallbacks that guarantee minimum prompt quality even in unconfigured projects.** A sub-agent cannot be dispatched unless its prompt passes assembly-gate checks.

### Component overview

```
dispatch_task(category, task_id)
    │
    ├── 1. Assembly gate: run 3.0 pipeline (non-bypassable)
    │       ├── stepResolveStage → stepLookupBinding
    │       ├── stepResolveRole    → stepLoadSkill → stepSurfaceKnowledge
    │       ├── stepResolveToolHint (WITH hardcoded fallbacks)
    │       └── stepAssembleSections
    │
    ├── 2. Gate checks: validate assembled prompt
    │       ├── Required sections present? (Role, Vocabulary, Procedure, Tools)
    │       ├── Minimum token budget met? (> ~500 tokens)
    │       └── Role resolved? Skill loaded?
    │
    ├── 3. Provider dispatch
    │       └── Send assembled prompt to provider API
    │
    └── 4. Response → orchestrator
```

### Change 1: Hardcoded default tool hints

Add a fallback map in `stepResolveToolHint` (or at the `Pipeline` level) that activates when `MergedToolHints` is empty or misses the resolved role. This is a code change, not a config change — it guarantees every `handoff` prompt includes tool guidance.

```go
// internal/context/pipeline.go — stepResolveToolHint amendment
var defaultToolHints = map[string]string{
    "implementer-go": `## Available Tools

**Codebase navigation (preferred over grep):**
- search_graph — find functions, classes, and routes by name or natural language
- codebase_memory_mcp_search_code — graph-augmented code search with structural ranking
- get_code_snippet — read source code for a specific function/class
- query_graph — execute Cypher queries against the knowledge graph
- trace_path — trace call/data-flow paths through the code graph

**File operations (use entity_id for worktree-isolated writes):**
- kanbanzai_edit_file — create or edit files with entity_id scoping
- write_file — write file content with entity_id scoping
- read_file — read file content
- grep — search file content by pattern (use search_graph for structural queries)

**Workflow:**
- finish — mark a task done and contribute knowledge
- entity — manage workflow entities (get, list, update)
- status — check project/entity health and progress`,

    "architect": `## Available Tools
- search_graph — structural code search
- query_graph — complex graph queries
- trace_path — call chain analysis
- codebase_memory_mcp_search_code — code search with ranking
- decompose — task breakdown from specs`,

    "reviewer-conformance": `## Available Tools
- search_graph — find functions and trace calls
- codebase_memory_mcp_search_code — code search
- get_code_snippet — read function source
- read_file — read files directly`,

    "reviewer-quality": `## Available Tools
- search_graph — structural analysis
- codebase_memory_mcp_search_code — code search
- get_code_snippet — read source`,

    "reviewer-security": `## Available Tools
- search_graph — trace data flow
- query_graph — find all callers/callees
- trace_path — trace data flow paths`,

    "reviewer-testing": `## Available Tools
- search_graph — find test coverage gaps
- codebase_memory_mcp_search_code — search for test patterns`,

    "spec-author": `## Available Tools
- doc_intel — search and classify document content
- entity — workflow entity management
- doc — document registration and approval`,
}

func (p *Pipeline) stepResolveToolHint(state *PipelineState) {
    // 1. Try merged config hints.
    if len(p.MergedToolHints) > 0 && state.Role != nil {
        state.ToolHint = ResolveToolHint(p.MergedToolHints, state.Role.ID, p.ToolHintRoleStore)
        if state.ToolHint != "" {
            return
        }
    }
    // 2. Try inheritance chain through config hints.
    // (already handled by ResolveToolHint above)

    // 3. Hardcoded fallback: use defaults if no hint resolved.
    if state.Role != nil {
        if hint, ok := defaultToolHints[state.Role.ID]; ok {
            state.ToolHint = hint
            return
        }
    }
}
```

**Config still wins.** `MergedToolHints` from `config.yaml` / `local.yaml` overrides the hardcoded defaults per-key. Projects with custom tool guidance are unaffected. Projects without any config get the defaults.

### Change 2: Assembly gate checks

Add a `dispatch_task` implementation (P44 Phase 1) that validates pipeline output before dispatch. The gate checks:

| Check | Condition | Action on failure |
|-------|-----------|-------------------|
| Role resolved | `state.Role != nil` | Error — cannot dispatch without role identity |
| Skill loaded | `state.Skill != nil` | Error — cannot dispatch without procedure |
| Tool hint present | `state.ToolHint != ""` | Warn, but proceed — better to dispatch with a warning than block entirely |
| Minimum sections | At least 5 sections present | Warn |
| Minimum tokens | > 500 tokens assembled | Warn (may indicate assembly failure) |

These checks run synchronously before provider dispatch. A failing role/skill check blocks dispatch entirely — the orchestrator receives an actionable error ("role `implementer-go` not resolvable") rather than a silently degraded sub-agent.

### Change 3: Ship default tool hints in `kbz init` config

In addition to the hardcoded fallbacks, `kbzinit` should write a `config.yaml` that includes the default tool hints. This makes them visible, discoverable, and overridable. The hardcoded fallback is a safety net; the config file is the canonical source.

### Change 4: Make `next(id)` return a rendered prompt field

Currently, `next(id)` returns structured JSON context and `handoff(task_id)` returns Markdown. If `next` also included a `handoff_prompt` field with the same pipeline output, the orchestrator would see the rendered prompt alongside the JSON and have less reason to compose manually. This is a low-effort change: `nextClaimMode` already has access to the pipeline; add one field to the response map.

```go
// In nextClaimMode, after pipeline.Run(input):
prompt := kbzctx.RenderPrompt(result)
// Add to response:
response["handoff_prompt"] = prompt
```

### What this design does NOT change

- The 3.0 pipeline itself is not modified structurally — all 11 positions remain
- `stage-bindings.yaml` schema is not modified
- The orchestrator skill (`orchestrate-development`) is not modified — it already mandates `handoff`
- Config merge semantics are unchanged — local always overrides project
- The `handoff` tool continues to work exactly as it does today, just with guaranteed tool hints

### Failure modes and handling

| Failure mode | Impact | Handling |
|-------------|--------|----------|
| Pipeline fails entirely (no binding, no role) | Cannot dispatch | `dispatch_task` returns structured error; orchestrator falls back to direct `handoff` + manual `spawn_agent` |
| Role resolved but no tool hint | Sub-agent lacks tool guidance | Warn + proceed with defaults; tool hint is advisory, not mandatory |
| Skill not found | Sub-agent lacks procedure | Error — block dispatch; skill is mandatory for task execution |
| Knowledge surfacing fails | No knowledge entries in prompt | Proceed — knowledge is best-effort |
| Tool hint config has stale entries | Sub-agent tries non-existent tools | Model discovers tool isn't available and adapts; worse case is an unused instruction |

## Alternatives Considered

### Alternative A: Config-only tool hints (status quo improvement)

Ship default tool hints only via `kbz init` config, no hardcoded fallback.

**Pros:** Clean separation — config is config, code is code. No magic defaults hidden in Go source.

**Cons:** Existing projects without tool hints stay broken until someone manually updates their config. Every new project gets tool hints, but the systemic gap remains for current work. Violates "don't make the user configure for reasonable defaults."

**Verdict:** Rejected as the sole fix. Good as a supplementary approach (Change 3) but insufficient alone.

### Alternative B: Remove tool hints from config entirely, only use hardcoded defaults

**Pros:** Zero configuration. Always correct. No drift between config and code.

**Cons:** Removes the override mechanism. Projects that want custom tool guidance (e.g., a project that primarily uses a different code search tool) can't customize. Violates the principle that config should be able to override built-in behaviour.

**Verdict:** Rejected. The merge strategy (config overrides hardcoded) preserves both goals.

### Alternative C: Prompter sub-agent step

Before dispatching an implementer, dispatch a lightweight "prompter" sub-agent that calls `handoff(task_id)` and returns the rendered prompt. The orchestrator passes this prompt directly to the implementer.

**Pros:** Pipeline execution is invisible to the orchestrator's context window. Works today without code changes. Aligns with the sub-agent delegation pattern.

**Cons:** Extra round-trip per task dispatch (latency). Doesn't prevent manual composition — the orchestrator can still skip the prompter step. Adds orchestration complexity.

**Verdict:** Useful as a stopgap but not a structural fix. P44's `dispatch_task` with assembly gates makes this unnecessary.

### Alternative D: Pre-generate prompts during dev-planning

During the dev-planning stage, generate the full handoff prompt for each task and embed it in the dev-plan document. The orchestrator reads it out during implementation.

**Pros:** Zero runtime assembly cost. Prompts are reviewed and approved alongside the plan. No assembly gate complexity in dispatch.

**Cons:** Prompts become stale if roles, skills, or knowledge evolve between planning and implementation. Changes to tool hints, anti-patterns, or vocabulary require re-generating prompts. Adds dev-planning stage burden.

**Verdict:** Rejected for now. The staleness risk outweighs the latency benefit. Worth revisiting if assembly latency becomes a bottleneck.

## Decisions

### Decision 1: Hardcoded default tool hints as a safety net, config as the canonical source

**Context:** The tool hints pipeline is complete but gated on config that doesn't exist. Without defaults, every project is silently missing tool guidance in sub-agent prompts.

**Rationale:** Hardcoded defaults guarantee the pipeline produces complete prompts in all environments. Config overrides let projects customize without losing the safety net. The alternative (config-only) leaves existing projects broken.

**Consequences:**
- Positive: Every sub-agent prompt includes tool guidance immediately, with zero config changes
- Positive: `codebase-memory-mcp` and `kanbanzai_edit_file` appear in every implementer prompt
- Negative: Default tool hints live in Go source, which is less discoverable than config — mitigated by `kbz init` also writing them
- Negative: Changing defaults requires a code change, not just a config update

### Decision 2: Assembly gate checks are synchronous and blocking for critical failures (role, skill)

**Context:** P44's `dispatch_task` will run the pipeline invisibly. Without gate checks, a silently degraded pipeline produces silently degraded sub-agents.

**Rationale:** The P51 design already identified this risk (severity: High). Gate checks are the mitigation. Critical failures (no role, no skill) must block dispatch because the resulting sub-agent would lack identity and procedure — two essential prompt components.

**Consequences:**
- Positive: Degraded dispatch is caught before it reaches a sub-agent
- Positive: Orchestrator receives actionable errors, not silent degradation
- Negative: Gate failures block dispatch, potentially stalling a feature — the orchestrator must handle the error (fall back to manual `handoff` + `spawn_agent`)
- Negative: Gate logic adds complexity to `dispatch_task`

### Decision 3: Assembly gate checks are advisory (warn) for non-critical gaps (tool hints, token budget)

**Context:** Not all prompt gaps are dispatch-blocking. Missing tool hints degrades quality but doesn't prevent task completion. A sub-agent without tool hints can still use the MCP server's tool list to discover available tools.

**Rationale:** The cost of blocking dispatch for non-critical gaps (orchestrator stall, human intervention) outweighs the benefit. A warning preserves visibility without stopping work.

**Consequences:**
- Positive: Dispatch isn't blocked by configuration gaps
- Positive: Warnings surface issues for the orchestrator to address
- Negative: A sub-agent may receive a prompt without tool guidance — it will still function, just less efficiently

### Decision 4: `next(id)` returns a `handoff_prompt` field alongside structured context

**Context:** The orchestrator currently uses `next(id)` to claim tasks and receives JSON context. It may then compose prompts manually. Adding a rendered prompt field reduces the incentive to bypass `handoff`.

**Rationale:** This is a one-field addition with zero risk — the pipeline already runs during `nextClaimMode`. Exposing the rendered output makes the orchestrator aware that a rich prompt exists and encourages its use.

**Consequences:**
- Positive: Orchestrator sees the rendered prompt without an extra `handoff` call
- Positive: Backward compatible — existing JSON consumers are unaffected
- Negative: Slightly increases `next` response size

## Dependencies

- **P51 (Handoff Pipeline Unification):** Must be complete and verified before P44 Phase 1 begins. This design depends on the pipeline being correct — gate checks are only useful if the pipeline itself produces correct output.
- **P42 (Hash-Anchored Edit Tool):** `kanbanzai_edit_file` must exist and be registered in the MCP server for the tool hint to be accurate.
- **P43 (Fast-Track Architecture):** Fast-track dispatch mode may need its own tool hint profile — lighter than the full implementer profile.
- **P55 (Orchestrator Context Hygiene):** The orchestrator's role reminder and constraint pinning must work alongside assembly gates.

## Open Questions

1. **Should we abandon the chat-based orchestrator entirely?** The P57 evidence is damning: despite a working pipeline, a verified fix (P58), and an explicit skill rule, the orchestrator still composes prompts by hand. P44's `dispatch_task` would enforce the pipeline in code, but the orchestrator still needs to call `dispatch_task` instead of `spawn_agent`. If the orchestrator can also bypass `dispatch_task`, we're back to the same problem one layer up. A more radical architecture — where the MCP server itself spawns sub-agents in response to task state transitions, removing the orchestrator from the dispatch loop entirely — may be necessary.

2. **Should assembly gate failures trigger a checkpoint?** If `dispatch_task` can't resolve a role or skill, the orchestrator can't proceed. A checkpoint asking the human to fix the configuration might be better than an error the orchestrator can't resolve. But checkpoints block all work, not just one feature.

2. **Should the hardcoded defaults be per-role only, or also per-skill?** The design proposes per-role defaults. Skills could theoretically have different tool preferences — e.g., `implement-task` (Go) vs. `implement-task` (TypeScript) might need different tool hints. But the current skill system doesn't have per-skill tool hints, and adding that is a separate feature.

3. **How should `dispatch_task` handle the case where the pipeline fails but the orchestrator has a handoff-generated prompt?** If the orchestrator called `handoff` before `dispatch_task`, it has a valid prompt. Should `dispatch_task` accept an optional `prompt` parameter to skip re-assembly? This would save a pipeline run but introduces a bypass path.
