# Role-Scoped Tool Hints

- Status: final draft
- Purpose: allow users to inject role-scoped tool availability hints into
  sub-agent prompts via config, making the system portable across different
  MCP setups
- Date: 2025-07-14
- Phase: 1 of the codebase-memory integration roadmap
- Related:
  - `work/design/codebase-memory-integration.md` — Phase 2–3 of the
    integration roadmap; provides worktree-aware graph project context that
    complements the general hints defined here
  - `work/design/machine-context-design.md` — context assembly and role
    profiles
  - `work/design/agent-onboarding.md` — skill discovery gap
  - `.kbz/roles/` — role definitions (keys in `tool_hints` match these IDs)
  - `refs/sub-agents.md` — sub-agent delegation and context propagation

---

## 1. Problem Statement

Kanbanzai's `handoff` tool assembles a prompt for each sub-agent containing
role instructions, skill procedures, spec sections, and knowledge entries. It
does not include information about which MCP tools are available on the current
machine. Sub-agents therefore have no awareness of optional tools and default
to `grep` and `read_file` for all code exploration, even when faster,
higher-quality alternatives are available.

The current workaround is documented in `refs/sub-agents.md`: orchestrators
must manually include tool context in every `spawn_agent` call. This is a
fragile convention — easy to forget, not enforced, and breaks down across
delegation chains.

The deeper issue is that tool availability is machine-specific. Not every user
of Kanbanzai has the same MCP servers installed. Hardcoding tool references
into skill files or prompt templates would break portability — agents would
receive instructions to use tools that don't exist in their environment.

### 1.1 Role in the integration roadmap

This design is **Phase 1** of a three-phase integration plan:

| Phase | Design | What it solves |
|-------|--------|----------------|
| **1 (this)** | Role-tool-hints | Agents don't know what tools exist |
| 2 | `codebase-memory-integration.md` §Phase 2 | Agents don't know which graph project to query |
| 3 (deferred) | `codebase-memory-integration.md` §Phase 3 | Indexing requires manual agent action |

Phase 1 is independently valuable and has no external dependencies. It
improves agent behaviour with whatever MCP tools are available — not just
`codebase_memory_mcp` but any server on the machine (security scanners,
database explorers, custom tooling).

Phase 2 builds on Phase 1 by adding a structured `## Code Graph` section to
handoff output (generated from worktree state, not user config). The two
sections are distinct and complementary:

- `## Available Tools` (this design) — general, user-configured, role-scoped
- `## Code Graph` (Phase 2) — specific, auto-generated from worktree record

When both are present, `## Code Graph` appears after `## Available Tools`
because it is more specific and may reference tools already introduced by the
hint.

### 1.2 Observed impact

When optional tools are available but sub-agents are not told about them:

- Implementer agents use `grep` and `read_file` for structural questions that
  graph tools would answer in a single call
- Reviewer agents scan files line-by-line rather than using specialised tools
  to assess blast radius
- Context windows fill with raw file content instead of targeted output
- Tasks take longer and produce noisier reasoning

### 1.3 Why skill files are not the solution

Skill files could mention optional tools, but:

1. Not all users have the same tools — a hardcoded reference breaks the
   workflow for users who don't have it, or causes agent confusion
2. Sub-agents receiving a skill path via a hint are unlikely to read it fully
   mid-task — they get an outline from `read_file` on a large file and move on
3. The skill files are project assets tracked in Git — they should not encode
   machine-specific tool availability

---

## 2. Design Goals

1. **Portable** — the core system works identically for users with or without
   optional MCP servers; no hardcoded tool references in committed files
2. **Local-first** — tool availability is configured per-machine in
   `local.yaml`; team defaults can be committed in `config.yaml`
3. **Simple** — a flat key-value config; no parsing logic, no special cases in
   the server beyond reading and injecting
4. **Role-aware** — hints are scoped to the roles that benefit from them;
   an implementer hint does not appear in a reviewer's prompt
5. **Flexible** — hint values are opaque strings; users decide whether to
   write a one-liner tool preference or an instruction to read a skill file
6. **Overridable** — `local.yaml` (per-machine) overrides `config.yaml`
   (per-project) on a per-key basis

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
  implementer-go: |
    Use search_graph(name_pattern=...) and get_code_snippet() for structural
    code questions (callers, callees, symbol definitions). Fall back to grep
    only for string literals and error messages. Read
    .github/skills/codebase-memory-exploring/SKILL.md before your first
    graph query.
  reviewer: |
    Before starting your review, read and follow
    .github/skills/codebase-memory-tracing/SKILL.md. Use trace_call_path on
    any function whose signature changed to verify all callers are updated.
```

```yaml
# .kbz/config.yaml (committed, team-wide defaults)
tool_hints:
  implementer: "Use grep and read_file for code exploration."
```

### 3.2 Implementation

#### Config structs

In `internal/config/user.go`, add `ToolHints` to `LocalConfig`:

```go
type LocalConfig struct {
    User struct {
        Name string `yaml:"name"`
    } `yaml:"user"`
    GitHub   GitHubConfig      `yaml:"github,omitempty"`
    ToolHints map[string]string `yaml:"tool_hints,omitempty"`
}
```

In `internal/config/config.go`, add `ToolHints` to `Config`:

```go
type Config struct {
    // ... existing fields ...
    ToolHints map[string]string `yaml:"tool_hints,omitempty"`
}
```

No `Default*Config()` function is needed — an empty or nil map means no hints,
which preserves existing behaviour for all users.

No schema version bump is required — the field is optional and additive.
Existing `config.yaml` files without `tool_hints` parse identically.

#### Merge function

A new function resolves the effective hints by merging project and local
configs:

```go
// MergeToolHints returns the effective tool hints map. Local hints override
// project hints on a per-key basis. Either or both inputs may be nil.
func MergeToolHints(project, local map[string]string) map[string]string {
    if len(project) == 0 && len(local) == 0 {
        return nil
    }
    merged := make(map[string]string, len(project)+len(local))
    for k, v := range project {
        merged[k] = v
    }
    for k, v := range local {
        merged[k] = v // local wins
    }
    return merged
}
```

This is called once at server startup (or config reload) and the merged map is
stored on the MCP server struct for use by `handoff` and `next`.

#### Role inheritance resolution

If a hint is defined for a parent role (e.g. `implementer`) and the active
role inherits from it (e.g. `implementer-go`), the parent's hint applies
unless the child has its own entry. This follows the same inheritance
resolution already used for role profiles in
`internal/context/role_resolve.go`.

Resolution order for a given role:

1. Exact match in merged hints → use it
2. Parent role match (walking the `inherits` chain) → use it
3. No match → omit the hint section from the prompt

```go
// ResolveToolHint returns the effective hint for the given role ID, walking
// the inheritance chain if no exact match exists. Returns "" if no hint
// resolves.
func ResolveToolHint(hints map[string]string, roleID string, resolver RoleResolver) string {
    if hint, ok := hints[roleID]; ok {
        return hint
    }
    resolved, err := resolver.Resolve(roleID)
    if err != nil {
        return ""
    }
    // Walk inheritance chain (ResolvedRole includes the parent chain)
    for _, ancestor := range resolved.InheritanceChain {
        if hint, ok := hints[ancestor]; ok {
            return hint
        }
    }
    return ""
}
```

The `RoleResolver` interface already exists in `internal/context/pipeline.go`
and is available in both the 3.0 pipeline path and the legacy 2.0 assembly
path.

### 3.3 Injection point

#### Handoff (3.0 pipeline path)

The hint is injected as a new section in `stepAssembleSections` in
`internal/context/pipeline.go`. It is placed early in the section ordering —
after Identity and Role but before Procedure and Knowledge — so the agent
sees tool guidance before task-specific content.

Section label:

```markdown
## Available Tools

Use search_graph(name_pattern=...) and get_code_snippet() for structural
code questions; fall back to grep only for string literals and error messages.
```

The section is omitted entirely if no hint resolves for the active role.
Prompts for users without `tool_hints` configured are unchanged.

#### Handoff (legacy 2.0 path)

In `internal/mcp/handoff_tool.go`, the hint is rendered in
`renderHandoffPrompt` before the "Additional Instructions" section. Same
`## Available Tools` heading, same omission rule.

#### Next

In `internal/mcp/next_tool.go`, the resolved hint is included in the
structured context output as a `tool_hint` string field alongside
`role_profile`, `constraints`, etc.:

```json
{
  "context": {
    "role_profile": "implementer-go",
    "tool_hint": "Use search_graph(name_pattern=...) ...",
    "...": "..."
  }
}
```

**Decision (resolving Open Question 1 from the previous draft):** Yes, `next`
includes the hint. Orchestrators working directly (not via sub-agents) benefit
from the same tool guidance. The hint is a single string — its token cost is
negligible against the context budget.

#### Coexistence with Phase 2

When Phase 2 (worktree-aware graph context) is implemented, `handoff` will
emit both sections:

1. `## Available Tools` — from tool hints (this design)
2. `## Code Graph` — from worktree `GraphProject` field (Phase 2)

These are non-overlapping. `## Available Tools` provides general tool guidance
that may cover tools beyond `codebase_memory_mcp`. `## Code Graph` provides
the specific project name and per-tool call examples. An agent receiving both
gets general guidance *and* specific graph context.

### 3.4 Hint authoring guidance

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

**Recommended starter hints for projects using `codebase_memory_mcp`:**

```yaml
# .kbz/local.yaml
tool_hints:
  implementer-go: |
    Use search_graph and trace_call_path for structural code questions
    (callers, callees, symbol definitions, impact analysis). Use
    get_code_snippet to read specific functions by qualified name. Fall
    back to grep only for string literals and error messages.
  reviewer: |
    Before starting your review, read .github/skills/codebase-memory-tracing/SKILL.md.
    Use trace_call_path on any function whose signature changed to verify
    all callers are updated. Use detect_changes for blast radius assessment.
  researcher: |
    Use get_architecture for codebase orientation. Use search_graph for
    finding definitions and query_graph for structural patterns. Prefer
    graph tools over reading files for understanding code structure.
```

---

## 4. Scope

### In scope

- `tool_hints` field on `LocalConfig` (`internal/config/user.go`)
- `tool_hints` field on `Config` (`internal/config/config.go`)
- `MergeToolHints` function for per-key local-overrides-project merge
- `ResolveToolHint` function with role inheritance resolution
- Injection into `handoff` output as `## Available Tools` section (both 3.0
  pipeline and legacy 2.0 paths)
- Injection into `next` context output as `tool_hint` field
- Surfacing via `health` tool: merged hints are included in the health report
  under a `tool_hints` section so users can verify what will be injected
- Documentation: `refs/sub-agents.md` updated to reference this mechanism and
  deprecate the manual tool-context convention

### Out of scope

- `codebase-memory-mcp` graph project injection — covered by Phase 2 of
  `work/design/codebase-memory-integration.md`
- Global per-machine config (e.g. `~/.config/kanbanzai/`) — not in current
  architecture; a separate design decision
- Validation of hint content — values are opaque strings; no schema
  enforcement
- Multiple hints per role — one hint per role keeps the config readable and
  the injection simple; users who need composition can write a multi-line
  YAML string
- Hint templating or variable substitution

---

## 5. Alternatives Considered

### 5.1 Hardcode graph tool instructions in skill files

Rejected. Breaks portability — users without optional MCP servers receive
instructions for tools that don't exist. Skill files are committed project
assets and should not encode machine-specific configuration.

### 5.2 Require orchestrators to manually include tool context in spawn_agent

The current approach, documented in `refs/sub-agents.md`. Rejected as the
primary solution because it is fragile — easy to forget, not enforced by the
system, and breaks down across delegation chains. Tool hints replace this
manual convention with an automatic one.

### 5.3 Single global config in home directory

A `~/.config/kanbanzai/local.yaml` file would allow machine-wide hints across
all projects without duplication. Rejected for now — no global config location
exists in the current architecture, and the most common case (one active
project per machine) is well served by per-project `local.yaml`. Adding a
global config layer is a separate design decision that can be made
independently and would subsume tool hints automatically.

### 5.4 Path-to-skill as a special config value

If the server detected that a hint value was a file path, it could read and
inline the skill content automatically. Rejected — adds special-case logic to
the server for a case that is already handled well by an explicit "read this
file" instruction in the hint string. The plain-string design handles both
patterns without branching.

### 5.5 Auto-detection of available MCP tools

The server could query the MCP host for the tool list and generate hints
automatically. Rejected — MCP servers do not have a standard mechanism to
discover sibling servers. The tool list the agent sees is assembled by the
host (e.g. Claude Code, VS Code + Copilot) and is not visible to individual
servers. User-authored hints are the correct layer for this information.

---

## 6. Resolved Questions

These were open in the previous draft and are now resolved:

**Q1. Should `next` inject the hint?**
Yes. Orchestrators working directly benefit from the same guidance. The token
cost of a single hint string is negligible. Both `handoff` and `next` emit
the resolved hint.

**Q2. Should merged hints be surfaced for verification?**
Yes, via `health`. The health report already covers entities, knowledge,
worktrees, branches, and context profiles. Adding a `tool_hints` section
(showing the merged result per role) is consistent and helps users verify
their config before dispatching agents.

**Q3. Is one hint per role sufficient?**
Yes. Users who need to express multiple concerns can use multi-line YAML
strings. Keeping it to one value per key avoids composition complexity and
keeps the injection logic trivial.

---

## 7. Summary of Proposed Changes

| Component | File(s) | Change | Lines (est.) |
|-----------|---------|--------|-------------|
| Config struct | `internal/config/config.go` | Add `ToolHints map[string]string` field | ~3 |
| Local config struct | `internal/config/user.go` | Add `ToolHints map[string]string` field | ~3 |
| Merge function | `internal/config/merge.go` (new) or inline | `MergeToolHints(project, local)` | ~15 |
| Resolve function | `internal/context/tool_hints.go` (new) | `ResolveToolHint(hints, roleID, resolver)` | ~20 |
| Pipeline injection | `internal/context/pipeline.go` | New section in `stepAssembleSections` | ~15 |
| Legacy injection | `internal/mcp/handoff_tool.go` | New section in `renderHandoffPrompt` | ~10 |
| Next injection | `internal/mcp/next_tool.go` | Add `tool_hint` to `nextContextToMap` | ~5 |
| Server wiring | `internal/mcp/server.go` | Merge hints at startup, pass to pipeline/assembly | ~10 |
| Health reporting | `internal/mcp/health_tool.go` | Emit merged hints in report | ~10 |
| Documentation | `refs/sub-agents.md` | Reference tool hints, deprecate manual convention | ~10 |

**Total: ~100 lines of Go + documentation updates.**