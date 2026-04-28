# Sub-Agent Context Propagation: A Real-World Validation

- Status: research note
- Date: 2025-07-26
- Related:
  - `work/design/machine-context-design.md` §2.3 (sharing problem), §2.4 (scoping problem), §11.1 (AGENTS.md limitations)
  - `work/design/agent-interaction-protocol.md`
  - `AGENTS.md` §Delegating to Sub-Agents

---

## 1. The Observation

During the Phase 1 review remediation, multiple sub-agents were spawned to work in parallel on different work packages (Crockford normalization, prefix resolution, display formatting, test migration, decision log updates). All of these agents had access to `codebase-memory-mcp` — the knowledge graph was indexed and ready — but none of them used it. Instead, they defaulted to `grep`, `find_path`, and `read_file` for all code exploration, including structural questions (finding callers, understanding interfaces, tracing dependencies) where the graph would have been faster and more precise.

The cause was straightforward: spawned agents don't see `AGENTS.md`. They receive only the instructions written into the `spawn_agent` message. The top-level agent had full knowledge of the graph (project name, tool preferences, fallback policy), but failed to propagate it because nothing required or reminded it to do so.

## 2. The Manual Fix

We added two sections to `AGENTS.md`:

1. **Expanded `codebase-memory-mcp` section** — includes the concrete project name, a tool selection table, and explicit when-to-use-what guidance.
2. **New "Delegating to Sub-Agents" section** — requires every `spawn_agent` call to include graph context, and includes a recursive propagation rule: sub-agents that spawn further sub-agents must pass the same context forward.

This works, but it has obvious limitations:

- It depends on the top-level agent reading and following the instructions every time.
- The boilerplate must be manually pasted into each delegation message.
- If the project name, tool preferences, or conventions change, the instructions must be updated in prose — there's no structured source of truth.
- There's no verification that propagation actually happened.
- The pattern doesn't compose across projects with different tool configurations.

## 3. Why This Matters for Kanbanzai

This is a textbook instance of the problems described in the machine context design:

- **The sharing problem (§2.3):** Multiple agents independently failed to discover the same project tool — the knowledge graph — because there was no mechanism to share that knowledge between agent sessions. Each agent started fresh with no awareness of available infrastructure.

- **The scoping problem (§2.4):** Even when the right context exists (AGENTS.md has the information), it's delivered as a flat, unstructured document that the delegation mechanism doesn't know how to extract relevant pieces from. The entire file is either present (top-level agent) or absent (sub-agents).

- **AGENTS.md limitations (§11.1):** The instruction file is "flat and unstructured, not role-scoped, not machine-writable, not composable with design context." The sub-agent propagation rule we added is a workaround for the lack of structured context delivery — it's a human-authored prompt engineering patch for a system architecture problem.

## 4. What Kanbanzai Could Do Instead

With Kanbanzai's context assembly model (machine context design §8), the solution would be structural rather than prompt-based:

**Tier 1 (project conventions)** would include tool availability as structured data — not prose instructions, but a machine-readable declaration that this project has a codebase knowledge graph, what it's called, and how to use it. Every agent session would receive this automatically as part of context assembly, regardless of delegation depth.

**Context profiles (§6)** would scope the delivery. A "code reviewer" profile might emphasise graph exploration tools. An "editor" profile doing mechanical text replacements might not need them. The system would decide what to include based on the agent's role and task, not based on whether someone remembered to paste a block of text.

**The MCP server itself** would be the delivery mechanism. When an agent connects and requests context for its task, the server assembles the relevant conventions, architecture knowledge, and tool availability into a targeted context package. No human-authored propagation chain required.

The difference is between:

- **Current state:** "Please remember to tell your sub-agents about the knowledge graph, and tell them to tell their sub-agents."
- **Kanbanzai state:** every agent that connects to the MCP server automatically receives the right tool context for its role and task.

## 5. Design Implications

This experience validates several design choices and suggests one refinement:

**Validated:**
- Context must be delivered structurally, not through prose instruction chains. Manual propagation is fragile.
- Tool availability is project-level context (Tier 1) — it applies to all agents working on the project, regardless of role.
- The four-layer instruction stack (§8.4) correctly identifies that project conventions should be injected at the base layer, not rediscovered per session.

**Suggested refinement:**
- The machine context design should explicitly call out "available development tools and their preferences" as a Tier 1 convention category alongside coding style, naming conventions, and architectural patterns. The current Tier 1 description (§5.1) focuses on code conventions and doesn't explicitly mention tool configuration, but tool availability is equally important project knowledge that every agent needs.

## 6. Summary

We experienced — and manually patched — a context propagation failure that is exactly the class of problem Kanbanzai's machine context system is designed to eliminate. The manual fix (recursive prose instructions in AGENTS.md) works but is fragile, unverifiable, and doesn't scale. This is a concrete, first-hand validation that structured context delivery to delegated agents is a real need, not a theoretical one.