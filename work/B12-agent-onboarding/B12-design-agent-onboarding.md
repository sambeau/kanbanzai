# Agent Onboarding and Skill Discovery

- Status: design proposal
- Purpose: close the skill discovery gap revealed by first real-world usage — agents have kanbanzai MCP tools available but no awareness of the workflow protocol
- Date: 2026-03-30
- Plan: P12-agent-onboarding
- Related:
  - `work/design/fresh-install-experience.md` (P11 — MCP config, skills, roles, document layout)
  - `work/design/init-command.md` (original init command design)
  - `work/design/kanbanzai-1.0.md` (skills-based onboarding, §4)
  - `work/spec/init-command.md` (init command specification)
  - `.agents/skills/kanbanzai-getting-started/SKILL.md` (current session-start skill)
  - `.agents/skills/kanbanzai-workflow/SKILL.md` (stage gates and lifecycle)

---

## 1. Problem Statement

P11 delivered a comprehensive fresh-install experience: MCP server configuration,
embedded skills, default context roles, and a standard document layout. First
real-world testing revealed that an agent (Claude via GitHub Copilot in Zed)
had kanbanzai MCP tools available and functioning but:

1. **Did not use them.** The agent created documents by writing files directly,
   bypassing all lifecycle enforcement.
2. **Did not follow the workflow.** The agent skipped the Features and
   Specification stages entirely, jumping from design straight to a dev plan.
3. **Did not read the skills.** The six installed skill files were never opened.

The root cause is a **skill discovery gap**: the skills are installed to
`.agents/skills/kanbanzai-*/SKILL.md`, which follows the Anthropic Agents
specification. GitHub Copilot does not read this location — it reads
`.github/copilot-instructions.md`. Without reading the skills, the agent had
no awareness of the kanbanzai protocol, stage gates, or the requirement to use
MCP tools for workflow operations.

### 1.1 The discovery chain

For kanbanzai to work, three things must happen in sequence:

1. **MCP tools available** — the editor discovers `.mcp.json`, starts the
   kanbanzai server, and the agent can call tools like `status` and `next`.
   P11 solved this. ✅
2. **Agent knows the protocol** — the agent reads instructions that tell it
   to use the MCP tools, follow stage gates, and not write files directly.
   This is broken. ❌
3. **Workflow enforced** — the agent uses MCP tools per the protocol, and the
   tools enforce lifecycle transitions, document approval gates, and
   referential integrity. Cannot work without step 2. ❌

### 1.2 The editor fragmentation problem

There is no universal standard for where agents read project-level instructions:

| Editor / Platform | Primary instruction source |
|---|---|
| Claude Code | `AGENTS.md`, `.claude/` |
| GitHub Copilot (VS Code, Zed) | `.github/copilot-instructions.md` |
| Cursor | `.cursorrules`, `.cursor/rules/` |
| Windsurf | `.windsurfrules` |
| Gemini CLI | `AGENTS.md`, `.agents/` |
| Generic / convention | `AGENTS.md` |

The P11 design deliberately chose not to generate editor-specific files:

> "Kanbanzai does not generate editor-specific skill locations, does not create
> symlinks, and does not attempt to target every known path."

That decision was about **skill files** (detailed procedures, 100–200 lines each).
It remains correct — duplicating six skills into four locations would be
unmaintainable. But the decision does not address **entry-point files** — short
pointer documents (15–30 lines) that orient the agent and direct it to the skills.

### 1.3 Secondary gaps identified

The first-usage session also revealed:

- **No "don't write files directly" rule.** The skills say not to substitute
  grep/find for kanbanzai tool calls (reading), but have no equivalent rule
  about writing. The agent created documents with `edit_file` instead of using
  `doc` and `entity` tools.
- **No specification skill.** There is a `kanbanzai-design` skill but no
  `kanbanzai-specification` skill. The specification stage is defined in the
  workflow skill's stage gates table but has no procedural guidance, making it
  easier to skip.
- **No MCP-level orientation.** When an agent calls `status` or `next` without
  having read the skills, the response contains project state but no hint that
  skills exist or should be read. The tools assume the agent already knows the
  protocol.

---

## 2. Design

### 2.1 Generate AGENTS.md on init

`kbz init` will generate an `AGENTS.md` file at the project root. This is the
single highest-impact change because `AGENTS.md` is the closest thing to a
universal convention — it is read natively or by convention across most agent
platforms.

**Content:** A short, focused document (~30 lines) that:

1. Identifies this as a kanbanzai-managed project
2. Gives the three essential first steps: call `status`, call `next`, read the
   getting-started skill
3. States the "use MCP tools, don't write files directly" rule
4. Lists the stage gate progression (one line)
5. Points to the six skills by name with a brief "when to read" note

**AGENTS.md is a pointer, not a policy document.** It does not duplicate skill
content. It tells agents *where to look* and *what the essential rules are*.
Projects that need custom agent instructions can extend it — the generated
version is a starting scaffold.

**Version-aware conflict logic:** Same pattern as `.mcp.json`:

- A `<!-- kanbanzai-managed: v1 -->` marker at the top of the file
- On re-init: if the marker is present and the version is older, overwrite;
  if no marker, leave the file alone (user-maintained); if version is current
  or newer, skip
- New `--skip-agents-md` flag to opt out

**Placement:** Project root (`AGENTS.md`). This is where Claude Code, Gemini
CLI, and most conventions expect it.

### 2.2 Generate .github/copilot-instructions.md on init

For GitHub Copilot users (VS Code, Zed), `kbz init` will also generate
`.github/copilot-instructions.md`. This is a thin pointer (~15 lines) that
tells Copilot to read `AGENTS.md` and lists the three essential first steps.

**Why a separate file instead of just AGENTS.md:** GitHub Copilot does not
read `AGENTS.md` by default. It specifically reads
`.github/copilot-instructions.md`. Without this file, Copilot users get
no orientation at all — which is exactly the failure mode we observed.

**Same version-aware logic** as AGENTS.md: managed marker, version check,
skip if user-maintained.

**The .github/ directory:** Created if absent. If it already exists, only
the instructions file is written/updated.

### 2.3 Update skill content

Two targeted skill updates, not a rewrite:

#### 2.3.1 Add "don't write files directly" to getting-started and workflow skills

In `kanbanzai-getting-started/SKILL.md`, add a rule to the Session Start
section:

> **Use kanbanzai MCP tools for all workflow operations.** Do not create
> documents by writing files directly with `edit_file` or similar — this
> bypasses lifecycle enforcement, document registration, and health checks.
> Use `doc` to register and manage documents, `entity` to create and
> transition entities.

In `kanbanzai-workflow/SKILL.md`, add the same principle to the Emergency
Brake section:

> **Direct file writes bypassing MCP tools.** You are about to create a
> document in `work/` or an entity in `.kbz/state/` without using the
> corresponding kanbanzai tool (`doc`, `entity`).

#### 2.3.2 Add a kanbanzai-specification skill

A new skill at `.agents/skills/kanbanzai-specification/SKILL.md` covering:

- Purpose: guide the specification process from approved design to approved spec
- When to use: after design approval, before dev plan and task creation
- Roles: human is the Product Owner (owns acceptance criteria), agent is the
  Specification Writer (proposes structure, verifiable criteria, edge cases)
- What a good specification contains: scope, acceptance criteria (testable),
  constraints, out-of-scope, verification approach
- The specification quality bar: every acceptance criterion must be
  independently verifiable; "it works correctly" is not an acceptance criterion
- The approved specification invariant: no unresolved questions, no ambiguous
  criteria, single direction (not alternatives)
- Relationship to design: the spec operationalises the design — it says what
  to verify, not what to build. The design document says what to build.

This makes the specification stage a concrete, procedural thing rather than
just a row in the stage gates table.

### 2.4 MCP tool orientation breadcrumbs

When `status` (project overview) or `next` (empty queue) responds, and the
response suggests the agent may be unoriented, include a breadcrumb in the
response:

```json
{
  "orientation": {
    "message": "This is a kanbanzai-managed project. Read .agents/skills/kanbanzai-getting-started/SKILL.md for workflow guidance.",
    "skills_path": ".agents/skills/"
  }
}
```

**When to include it:** Always in the project-level `status` response (it's
small and harmless). In `next`, only when the queue is empty (the agent is
likely orienting, not mid-flow).

This catches agents that have MCP tools but missed the skills entirely — they
get a breadcrumb on their first tool call.

---

## 3. Design Decisions

### 3.1 AGENTS.md is a scaffold, not a managed file

**Decision:** AGENTS.md uses a version marker for managed updates but is
designed to be extended by project owners. If the marker is removed or the
file is substantially modified, init will not overwrite it.

**Rationale:** Unlike skills (which are kanbanzai-managed and version-updated),
AGENTS.md is the project's own document. Kanbanzai provides the starting
content; the project owner may add project-specific conventions, reading
orders, or constraints. Overwriting those on re-init would be destructive.

**Alternative considered:** Making AGENTS.md fully managed (like skills).
Rejected because every project's agent instructions are different — a Go
project needs different conventions than a Python project. The kanbanzai
portion is just the workflow section.

### 3.2 Copilot instructions file is a pointer, not a duplicate

**Decision:** `.github/copilot-instructions.md` contains ~15 lines that point
to `AGENTS.md`. It does not duplicate the AGENTS.md content.

**Rationale:** Maintaining two copies of the same content is a known failure
mode. The pointer approach means there is one source of truth (AGENTS.md) and
one redirect (copilot-instructions). If a project already has a
copilot-instructions file, the managed-marker logic will leave it alone.

**Alternative considered:** Generating a full copilot-instructions file with
all the kanbanzai rules. Rejected — divergence between AGENTS.md and
copilot-instructions would be inevitable and confusing.

### 3.3 No Cursor/Windsurf/other editor-specific files

**Decision:** `kbz init` generates only `AGENTS.md` and
`.github/copilot-instructions.md`. No `.cursorrules`, `.windsurfrules`, or
other editor-specific files.

**Rationale:** Cursor reads `.mcp.json` natively and has recently added
support for `AGENTS.md`. Windsurf is similar. The Copilot instructions file
is the exception because Copilot has no fallback to `AGENTS.md` and is a
major platform. If other editors prove equally unable to discover AGENTS.md,
we can add pointer files for them later — but we do not speculate.

### 3.4 Orientation breadcrumbs are data, not instructions

**Decision:** The MCP tool orientation field contains a short message and a
path, not a full set of instructions. It says "read the skill file," not
"here are the rules."

**Rationale:** MCP tool responses are structured data consumed by agents. Long
instructional text in tool responses is unreliable — agents may ignore it,
truncate it, or treat it as data rather than instructions. A short breadcrumb
with a file path is actionable: the agent can read the file. Trying to embed
the full protocol in a JSON response would be fragile and hard to maintain.

### 3.5 Specification skill follows the design skill's structure

**Decision:** The new `kanbanzai-specification` skill mirrors the structure
and tone of `kanbanzai-design`. Same YAML frontmatter convention, same
section layout (Purpose, When to Use, Roles, Process, Quality Bar, Gotchas,
Related).

**Rationale:** Consistency across skills makes them predictable. An agent
that has read the design skill knows what to expect from the specification
skill. A different format would create unnecessary cognitive load.

---

## 4. Scope

### 4.1 In scope

1. Generate `AGENTS.md` on `kbz init` (new and existing projects without one)
2. Generate `.github/copilot-instructions.md` on `kbz init`
3. Version-aware conflict logic for both files (same pattern as `.mcp.json`)
4. `--skip-agents-md` CLI flag
5. Update `kanbanzai-getting-started` skill: add "use MCP tools, don't write
   directly" rule
6. Update `kanbanzai-workflow` skill: add direct-write to the emergency brake
   list
7. New `kanbanzai-specification` skill
8. Orientation breadcrumb in `status` (project overview) and `next` (empty
   queue) responses
9. Update getting-started docs to mention AGENTS.md and copilot-instructions

### 4.2 Out of scope

- Generating `.cursorrules`, `.windsurfrules`, or other editor-specific files
- Changing the `.agents/skills/` location or adding editor-specific skill paths
- MCP server reliability improvements (editor-level concern)
- Multi-project MCP server support
- Generating a full AGENTS.md policy document (that's the project owner's job)

---

## 5. Risks

### 5.1 AGENTS.md conflicts with existing files

**Risk:** A project already has an AGENTS.md that has nothing to do with
kanbanzai. Overwriting it would be destructive.

**Mitigation:** The managed-marker check. If the file exists and has no
kanbanzai marker, it is left untouched and a warning is printed suggesting
the user add the kanbanzai section manually.

### 5.2 Copilot instructions may evolve

**Risk:** GitHub may change where Copilot reads instructions, making the
generated file irrelevant.

**Mitigation:** The file is a thin pointer. If the location changes, updating
init is a one-line path change. The content (pointing to AGENTS.md) remains
valid regardless.

### 5.3 Orientation breadcrumbs may be ignored

**Risk:** Agents may treat the orientation field as data rather than acting
on it.

**Mitigation:** Breadcrumbs are a belt-and-suspenders measure, not the
primary fix. The primary fix is AGENTS.md and copilot-instructions — files
that agents read before any tool call. If breadcrumbs prove useless, they
cost almost nothing and can be removed.

### 5.4 Specification skill may be too prescriptive

**Risk:** Different projects have different specification needs. A rigid
skill may not fit all contexts.

**Mitigation:** The skill describes principles and quality criteria, not a
rigid template. It mirrors the design skill's approach: here is the process,
here is the quality bar, here are the gotchas. Projects can override with
their own conventions in AGENTS.md.

---

## 6. Feature Decomposition (Proposed)

| # | Feature | Dependencies |
|---|---|---|
| A | AGENTS.md generation | None |
| B | Copilot instructions generation | A (shares the managed-marker pattern) |
| C | Skill content updates | None |
| D | Specification skill | None |
| E | MCP orientation breadcrumbs | None |

A and B are init-command changes. C and D are skill-content changes. E is an
MCP server change. All are independently testable. A is the highest priority
because it has the broadest impact.