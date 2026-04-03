# Role-Scoped Tool Hints

- Status: design proposal
- Purpose: allow users to inject role-scoped tool availability hints into sub-agent prompts via config, making the system portable across different MCP setups
- Date: 2026-04-02
- Related:
  - `work/design/machine-context-design.md` (context assembly and role profiles)
  - `work/design/agent-onboarding.md` (skill discovery gap)
  - `.kbz/roles/` (role definitions)
  - `refs/sub-agents.md` (sub-agent delegation and context propagation)
  - `refs/knowledge-graph.md` (codebase-memory-mcp usage)

---

## 1. Problem Statement

Kanbanzai's `handoff` tool assembles a prompt for each sub-agent containing
role instructions, skill procedures, spec sections, and knowledge entries. It
does not include information about which MCP tools are available on the current
machine. Sub-agents therefore have no awareness of optional tools — most notably
`codebase-memory-mcp` — and default to `grep` and `read_file` for all code
exploration, even when faster, higher-quality alternatives are available.

The current workaround is documented in `refs/sub-agents.md`: orchestrators must
manually include a codebase-memory-mcp context block in every `spawn_agent` call.
This is a fragile convention — easy to forget, not enforced, and breaks down
across delegation chains.

The deeper issue is that tool availability is machine-specific. Not every user
of Kanbanzai has `codebase-memory-mcp` installed. Hardcoding tool references
into skill files or prompt templates would break portability — agents would
receive instructions to use tools that don't exist in their environment.

### 1.1 Observed impact

When `codebase-memory-mcp` is available but sub-agents are not told about it:

- Implementer agents use `grep` and `read_file` for structural questions that
  `search_graph` would answer in a single call
- Reviewer agents scan files line-by-line rather than using `trace_call_path`
  to assess blast radius
- Context windows fill with raw file content instead of targeted symbol output
- Tasks take longer and produce noisier reasoning

### 1.2 Why skill files are not the solution

Skill files could mention graph tools, but:

1. Not all users have `codebase-memory-mcp` — a hardcoded reference breaks
   their workflow or causes agent confusion
2. Sub-agents receiving a skill path via a hint are unlikely to read it fully
   mid-task — they get an outline from `read_file` on a large file and move on
3. The skill files are project assets tracked in Git — they should not encode
   machine-specific tool availability

---

## 2. Design Goals

1. **Portable** — the core system works identically for users with or without
   optional MCP servers; no hardcoded tool references in committed files
2. **Local** — tool availability is configured per-machine, not per-project
3. **Simple** — a flat key-value config; no parsing logic, no special cases in
   the server beyond reading and injecting
4. **Role-aware** — hints are scoped to the roles that benefit from them;
   an implementer hint does not appear in a reviewer's prompt
5. **Flexible** — hint values are opaque strings; users decide whether to write
   a one-liner tool preference or an instruction to read a skill file
6. **Overridable** — a committed project-level config provides team defaults;
   a local (non-committed) config overrides them per-machine

---

## 3. Design

### 3.1 Config format

A `tool_hints` map is added to both `config.yaml` (committed, team-wide) and
`local.yaml` (not committed, per-machine). Keys are role IDs matching those in
`.kbz/roles/`. Values are plain strings — the exact text to inject into the
sub-agent prompt.

```yaml
# .kbz/local.yaml (per-machine, not committed)
tool_hints:
  implementer: "Use search_graph(name_pattern=...) and get_code_snippet() for structural code questions; fall back to grep only for string literals and error messages."
  reviewer-quality: "Before starting your review, read and follow .github/skills/codebase-memory-tracing/SKILL.md"
```

```yaml
# .kbz/config.yaml (committed, team-wide defaults)
tool_hints:
  implementer: "Use grep and read_file for code exploration."
```

### 3.2 Merge strategy

When both files define a hint for the same role, `local.yaml` wins. The merge
is per-key: a key present only in `config.yaml` is used as-is; a key present
in both uses the `local.yaml` value; a key present only in `local.yaml` is
used as-is.

This means a team can ship sensible defaults in `config.yaml` while individual
machines override them to match their local MCP setup.

### 3.3 Role inheritance

If a hint is defined for a parent role (e.g. `implementer`) and the active role
inherits from it (e.g. `implementer-go`), the parent's hint applies unless the
child has its own entry. This follows the same inheritance resolution already
used for role profiles.

Resolution order for a given role:
1. Exact match in merged hints → use it
2. Parent role match (walking the inheritance chain) → use it
3. No match → omit the hint section from the prompt

### 3.4 Injection point

The hint is injected by the `handoff` tool into the generated sub-agent prompt
as a clearly labelled section near the top, before spec context and task
instructions. Placement before the task body ensures the agent sees it before
any other content.

Suggested section label and format:

```
## Available Tools

Use search_graph(name_pattern=...) and get_code_snippet() for structural code
questions; fall back to grep only for string literals and error messages.
```

The section is omitted entirely if no hint resolves for the active role. Prompts
for users without `tool_hints` configured are unchanged.

The same hint is also injected by the `next` tool into the context packet
returned to the calling agent. This ensures orchestrators working directly
(not via sub-agents) also see the hint.

### 3.5 Hint authoring guidance

Two patterns work well in practice:

**Inline instruction** — best for busy roles (implementers) where the agent
acts immediately without reading further:

```
"Use search_graph(name_pattern=...) and get_code_snippet() for structural
code questions; fall back to grep only for string literals."
```

**Read-skill instruction** — best for roles where reading is the first action
(reviewers), making full compliance likely:

```
"Before starting your review, read and follow
.github/skills/codebase-memory-tracing/SKILL.md"
```

The server treats both identically — the value is passed through as-is. The
distinction is purely a matter of authoring preference.

---

## 4. Scope

### In scope

- `tool_hints` key in `LocalConfig` struct (`internal/config/user.go`)
- `tool_hints` key in project `Config` struct (`internal/config/config.go`)
- Merge function: local overrides project, per-key
- Role inheritance resolution for hint lookup
- Injection into `handoff` tool output
- Injection into `next` tool context packet
- Documentation: `refs/sub-agents.md` updated to reference this mechanism

### Out of scope

- Global per-machine config (e.g. `~/.config/kanbanzai/`) — not in current
  architecture; a separate design decision
- Validation of hint content — values are opaque strings; no schema enforcement
- Multiple hints per role — one hint per role keeps the config readable and
  the injection simple
- Hint templating or variable substitution

---

## 5. Alternatives Considered

### 5.1 Hardcode graph tool instructions in skill files

Rejected. Breaks portability — users without `codebase-memory-mcp` receive
instructions for tools that don't exist. Skill files are committed project
assets and should not encode machine-specific configuration.

### 5.2 Require orchestrators to manually include tool context in spawn_agent

The current approach, documented in `refs/sub-agents.md`. Rejected as the
primary solution because it is fragile — easy to forget, not enforced by the
system, and breaks down across delegation chains. The tool hints mechanism
replaces this manual convention with an automatic one.

### 5.3 Single global config in home directory

A `~/.config/kanbanzai/local.yaml` file would allow machine-wide hints across
all projects without duplication. Rejected for now — no global config location
exists in the current architecture, and the most common case (one active project
per machine) is well served by per-project `local.yaml`. Adding a global config
layer is a separate design decision that can be made independently.

### 5.4 Path-to-skill as a special config value

If the server detected that a hint value was a file path, it could read and
inline the skill content automatically. Rejected — adds special-case logic to
the server for a case that is already handled well by an explicit "read this
file" instruction in the hint string. The plain-string design handles both
patterns without branching.

---

## 6. Open Questions

1. Should `next` inject the hint into the context packet, or only `handoff`?
   The case for `next`: orchestrators working directly also benefit. The case
   against: the context packet has a budget and the hint may be redundant for
   experienced orchestrators who already know their tools.

2. Should the merged hints be surfaced by any existing tool (e.g. `status` or
   `health`) so users can verify what will be injected?

3. Is one hint per role sufficient, or will users want to compose hints from
   multiple sources (e.g. a base tool hint plus a project-specific note)?
   Keeping it to one per role is simpler and covers the known use cases.