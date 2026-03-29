# Fresh Install Experience Design

- Status: design proposal
- Purpose: redesign the out-of-box experience for new kanbanzai projects — MCP server connection, skills consolidation, default context roles, and standard document layout
- Date: 2026-03-29T09:46:34Z
- Related:
  - `work/design/init-command.md` (original init command design)
  - `work/design/kanbanzai-1.0.md` (skills-based onboarding, §4)
  - `work/design/machine-context-design.md` (context profiles)
  - `work/design/code-review-workflow.md` (code review skill, §9)
  - `work/spec/phase-2b-specification.md` §11 (context profile storage and suggested initial profiles)
  - `work/spec/hardening.md` §6 (init edge cases)

---

## 1. Scope

This document covers what `kbz init` installs on a **newly created project**. It does not prescribe changes to the kanbanzai project's own file layout, existing skill files, or directory structure. The kanbanzai project is the reference implementation and happens to use the same conventions — but the deliverable here is the installer behaviour and the files it writes.

---

## 2. Problem Statement

Running `kbz init` on a new project leaves the developer in a broken state on four dimensions.

### 2.1 The MCP server is not connected

`kbz init` creates `.kbz/config.yaml`, the `work/` directories, and the `.agents/skills/` files, but produces no MCP server configuration. The editor does not know the server exists. When an agent reads the installed skills and follows the getting-started skill, it immediately calls `next` — a tool that does not exist because the server is not running. The failure mode depends on the editor (silent failure, hallucinated response, or an error) but none are helpful.

The `kanbanzai-1.0` design doc specified that `init` should generate an editor-specific config snippet. This was never implemented.

### 2.2 No context roles are installed

The `init` command creates `.kbz/config.yaml` and the `.kbz/` directory structure, but does not create `.kbz/context/roles/`. Agents calling `context_assemble` find nothing. The getting-started and agents skills reference role-based context assembly without acknowledging that roles must first exist.

The Phase 2b specification §11.8 describes a standard set of suggested profiles but leaves creation entirely to the user with no init-time scaffolding.

### 2.3 The `.skills/` directory is a design artifact with no architectural basis

The kanbanzai project has a `.skills/` directory containing `code-review.md`, `plan-review.md`, `document-creation.md`, and `README.md`. These files:

- Are **not** installed by `kbz init` (so new projects do not get them)
- Are **not** read by `context_assemble` (the original design said they would be; this was never implemented)
- Are **not** protected by the `kanbanzai-managed` versioning marker
- Overlap in content and purpose with the embedded skills in `.agents/skills/kanbanzai-*/`

The `reviewer` context profile in this project references "the review SKILL" — a reference that resolves only because `.skills/code-review.md` exists in this project. In any other kanbanzai-managed project, that reference is dangling.

The doc-currency health checker scans `.skills/*.md` for stale tool references — a coupling that only exists because `.skills/` was never cleaned up.

This problem is best fixed at the installer level: new projects should receive the code-review and plan-review procedures as installed skills, making them available everywhere kanbanzai is used.

### 2.4 The default document layout is inconsistent

`DefaultDocumentRoots()` creates five directories on init: `work/design`, `work/spec`, `work/dev`, `work/research`, `work/reports`. This layout has several problems:

- `work/dev` and `work/plan` are conflated. The embedded `kanbanzai-documents` skill acknowledges the ambiguity by listing both as valid locations for `dev-plan` documents. But `InferDocType("plan")` currently returns `"design"` — so batch-importing from `work/plan/` silently registers documents with the wrong type, causing stage gate failures. Meanwhile there is no dedicated home for human-facing project planning documents (roadmaps, scope docs, decision logs) that are distinct from agent-facing feature dev-plans.
- `work/reviews/` is used by the review skills for review artefacts, but is not in `DefaultDocumentRoots()`. A fresh install never creates it, and it has no document type mapping.
- There is no place for retrospective synthesis documents (`retro` tool output).
- There is no guidance file for humans or agents about what goes where.
- Naming is inconsistent: `reports` is plural while all other names are singular or short.

The root of the `work/plan` vs `work/dev` confusion is architectural: the stage gate in `advance.go` checks document **type** (`dev-plan`), not path. So the path has never mattered to the gate — but `InferDocType()` maps folder name to type, and `"plan"` was never given a case, defaulting silently to `"design"`. This, combined with the skill listing both paths as valid, created the persistent ambiguity.

---

## 3. Goals

1. An agent opening a freshly initialised project can call `next` and get a meaningful response without any manual configuration beyond installing the binary.
2. A newly initialised project has working context profiles that any agent can assemble immediately.
3. The code review and plan review procedures are available to any kanbanzai-managed project as installed skills, not bespoke files in a specific project.
4. The document layout created by `init` is unambiguous, self-consistent, self-documenting for humans, and matches what the installed skills describe.
5. The conceptual distinction between **skills** (procedural) and **roles** (contextual conventions) is clear and consistently applied.

---

## 4. Non-Goals

- **Editor auto-detection.** We do not attempt to detect which editor is in use.
- **Migrating the kanbanzai project's own layout.** The kanbanzai project is not changed by this work. Its existing `.skills/` directory, `work/` structure, and role files are left as-is.
- **Retroactively updating existing user projects.** Changes to `DefaultDocumentRoots()` apply to new installs only. Existing `config.yaml` files are not modified.
- **A `developer.yaml` role template.** The developer role is inherently language- and project-specific. Shipping a Go-flavoured developer role to a Python project would be worse than shipping nothing.

---

## 5. Design

### 5.1 MCP server connection

#### 5.1.1 `.mcp.json`

`kbz init` writes a `.mcp.json` file at the project root. This format is supported by Claude Code, Cursor, and VS Code (with Copilot or Claude extensions) as the standard project-local MCP server declaration. It is safe to commit — the command is PATH-relative and contains no machine-specific content.

```json
{
  "mcpServers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"]
    }
  }
}
```

`.mcp.json` is committed alongside the rest of the project. Any contributor who clones the repository gets the server configured automatically — the same way they get the skill files. If the binary is not on PATH, the server silently fails to start, which is the correct precondition failure.

The file carries a `# kanbanzai-managed` comment in a top-level `_managed` key (or equivalent comment-compatible field) so that conflict detection logic can identify it. If `.mcp.json` already exists without the managed marker, `init` skips it and warns. If it carries the marker and is at an older version, it is updated.

#### Design decision FI-D-001: `.mcp.json` is committed, not gitignored

`.mcp.json` contains no machine-specific content when using a PATH-relative command. Committing it means every collaborator and every CI environment gets the server configured without manual steps. This is the same philosophy as committing `.agents/skills/` — the tool configuration is part of the repository.

If a user wants to use a different binary path, they can edit `.mcp.json` locally and add it to their personal `.git/info/exclude`. The committed version serves as the correct default.

#### 5.1.2 Zed support

If a `.zed/` directory already exists in the repository at init time, `kbz init` also writes `.zed/settings.json` (or merges into it if it already exists) with the Zed context server declaration:

```json
{
  "context_servers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"]
    }
  }
}
```

Note: `command` is a flat string (the binary name or path), with `args` as a sibling key. The nested `{"path": ..., "args": [...]}` form is not supported by Zed and will be silently ignored.

If `.zed/settings.json` already exists without the managed marker, `init` skips it and prints a note directing the user to `docs/getting-started.md` for the snippet to add manually.

**New-project behaviour (updated):** For new projects, `kbz init` always writes `.zed/settings.json`, creating the `.zed/` directory if it does not exist. Zed creates `.zed/` lazily on first open, which happens *after* `kbz init` runs — so waiting for the directory to appear as a signal is not viable for new projects. For existing projects, the absence of `.zed/` remains a reliable signal that the project does not use Zed, and the file is not written.

#### 5.1.3 Getting-started skill update

**The discovery model.** `.mcp.json` is the primary discovery mechanism. When an editor finds `.mcp.json`, it starts the kanbanzai server and registers its tools (`next`, `entity`, `doc`, `status`, and others) in the agent's tool list. The agent discovers kanbanzai through the tool catalogue — not through the skill files. Skills are the *procedure layer*: they tell the agent how to use those tools effectively once it knows they exist.

This means the `kanbanzai-getting-started` skill has a secondary but important role: it provides orientation and procedure, and it serves as a fallback discovery signal when an agent scans the skills directory without yet having seen the tool list.

**The description must be self-identifying.** The current description reads: *"Use at the start of any agent session in a Kanbanzai-managed project"*. This is circular — an agent must already know it is in a Kanbanzai project to apply this condition. It should instead state that fact directly. The updated description:

```
description: >
  This repository is managed with Kanbanzai. Read this skill at the start of
  every session, before writing any code or running any searches. Kanbanzai
  provides MCP tools (next, entity, doc, status, and others) that replace
  manual grep and file searching for project state and work queue. If you can
  see a .kbz/ directory or kanbanzai tools in your tool list, this skill
  applies.
```

**The skill body gains a preflight section** as its first item, before any tool calls:

> **Preflight check:** Kanbanzai works through MCP tools. Before calling `next` or any other tool, confirm the kanbanzai MCP server is connected (your editor should show it as active in its server list). If the kanbanzai tools are not available, the server is not running — the project's `.mcp.json` configures most editors automatically; consult `docs/getting-started.md` for manual setup instructions.

#### Design decision FI-D-007: `.mcp.json` is the primary discovery mechanism; skills are the procedure layer

An agent that finds the kanbanzai tools available in its tool list knows it is in a Kanbanzai-managed project without reading any skill file. An agent that finds the skill files but no tools is in a degraded state: the server is not running. The getting-started skill must handle both cases — confirming context for the first and diagnosing the problem for the second.

The skill description is therefore written as a statement of fact ("this repository is managed with Kanbanzai"), not a conditional that requires the agent to already know the context. This removes the circular dependency and ensures the skill functions as both a discovery signal and a procedure guide.

---

### 5.2 Skills consolidation

#### 5.2.1 Two new embedded skills

Two skills are added to the set installed by `kbz init`:

**`kanbanzai-review`** (`.agents/skills/kanbanzai-review/SKILL.md`)

The code review skill covers: review orientation, per-dimension evaluation guidance, structured output format, finding classification (blocking vs non-blocking), edge case handling, and the full orchestration procedure for feature-level reviews. It is the single canonical source for agents conducting code reviews in any kanbanzai-managed project.

**`kanbanzai-plan-review`** (`.agents/skills/kanbanzai-plan-review/SKILL.md`)

The plan review skill covers: plan scope verification, feature completion checks, spec conformance, documentation currency, cross-cutting checks, retrospective contribution, and the review report format.

After this change, eight embedded skills are installed by `kbz init`: `agents`, `design`, `documents`, `getting-started`, `planning`, `workflow`, `review`, `plan-review`.

#### 5.2.2 `kanbanzai-workflow` skill update

The `kanbanzai-workflow` skill is updated to describe the `reviewing` and `needs-rework` feature lifecycle states, which it currently omits entirely. It directs agents to `kanbanzai-review` for the review procedure.

#### 5.2.3 `kanbanzai-documents` skill update

The `kanbanzai-documents` skill is updated to reflect the new document layout (§5.4) and to absorb any step-by-step or troubleshooting content worth preserving. The type table is updated to include the new `plan` and `retrospective` types.

#### Design decision FI-D-002: skills are procedural; roles are contextual conventions

This distinction is made explicit here to prevent future drift:

**Skills** answer "how do I do X?" They contain ordered steps, tool call sequences, output format templates, and edge case handling. A skill is task-type-specific but project-agnostic — any kanbanzai-managed project can follow `kanbanzai-review` regardless of language or domain.

**Roles** answer "what should I know while doing X?" They contain conventions, constraints, package scope, and architectural overview specific to a project. A role is inherently project-specific. The `developer.yaml` for a Go project will look nothing like one for a Rust project — this is correct and intentional.

Content that belongs in skills: procedure, tool call sequences, output templates, conditional logic.  
Content that belongs in roles: coding standards, test conventions, package boundaries, architecture overview.  
Content that must not be duplicated between the two: a skill should not describe Go conventions; a role should not describe how to conduct a review.

---

### 5.3 Default context roles

`kbz init` creates two role files in `.kbz/context/roles/`:

#### `base.yaml` — installed as a scaffold

The base role is created with placeholder content and inline comments that direct the project owner to fill in their conventions. It is intentionally empty of kanbanzai-system content — that lives in the skills. The scaffold communicates the schema and expected content without imposing any specific project's conventions on a new install.

```yaml
id: base
description: "Project-wide conventions for all agents"
# Add your project's global conventions here.
# All other roles inherit from base unless they declare their own inherits field.
conventions: []
# architecture:
#   summary: "One paragraph describing the overall project structure"
#   key_interfaces:
#     - "The most important files/packages and what they do"
```

#### `reviewer.yaml` — installed fully populated

The reviewer role is universal. Every project that uses the feature review gate needs the same review dimensions, output format conventions, and approach guidance. Unlike the developer role, the reviewer role has no project-specific variation in its core content. The full reviewer role content is embedded in the binary and written on init.

The reviewer profile references `kanbanzai-review` as the SKILL to follow for review procedure — this reference is now valid in every kanbanzai project because `kanbanzai-review` is an installed skill.

#### `developer.yaml` — not installed

The developer role is intentionally omitted. It is language- and framework-specific by design. The `base.yaml` scaffold and the `kanbanzai-getting-started` skill both note that a `developer.yaml` can be created when the project has developer-specific conventions to encode.

#### Design decision FI-D-003: `reviewer.yaml` is version-managed

Like the installed skill files, `reviewer.yaml` carries a `kanbanzai-managed` marker and a version string. `kbz init --update-managed` (see §5.5) updates it alongside skill files. On existing installs, if `.kbz/context/roles/reviewer.yaml` already exists without the managed marker, `init` skips it and warns. If it carries the marker and is at an older version, it is updated.

---

### 5.4 Standard document layout

#### 5.4.1 Proposed directories

The following directories are created by `kbz init` on a new project, along with `work/README.md`:

| Directory | Document type | Audience | Purpose |
|---|---|---|---|
| `work/design/` | `design` | Human + Agent | Architecture, vision, approach decisions, policies |
| `work/spec/` | `specification` | Human + Agent | Acceptance criteria, binding contracts |
| `work/plan/` | `plan` | Human | Project planning: roadmaps, scope docs, decision logs, phase plans |
| `work/dev/` | `dev-plan` | Agent | Feature dev plans, implementation plans, task breakdowns |
| `work/research/` | `research` | Human + Agent | Analysis, exploration, background |
| `work/report/` | `report` | Human + Agent | Structured reports: audit reports, post-mortems, general reports |
| `work/review/` | `report` | Agent | Feature and plan review artefacts from the reviewing lifecycle gate |
| `work/retro/` | `retrospective` | Human + Agent | Retrospective synthesis documents (output of the `retro` tool) |

#### Design decision FI-D-004: `work/plan/` and `work/dev/` are separate directories with distinct document types

Human-facing project planning documents (roadmaps, scope documents, decision logs, phase plans) and agent-facing feature dev-plans serve different audiences, have different lifecycles, and should not share a directory or a document type.

`work/plan/` maps to a new `plan` document type. `work/dev/` retains the existing `dev-plan` type.

The stage gate for the `dev-planning` feature lifecycle state checks for an approved document of type `dev-plan` on the feature entity — it is entirely unaffected by this separation. A `plan`-type document in `work/plan/` will never satisfy a `dev-plan` gate, which is correct: human project plans are not the same artefact as a feature's implementation plan.

This also resolves the persistent `InferDocType("plan")` bug: the function currently returns `"design"` for the basename `"plan"` (no case exists). Adding a `"plan"` → `"plan"` case fixes silent type misclassification when batch-importing from `work/plan/`.

#### Design decision FI-D-005: `work/report/` is singular; `work/review/` is a separate directory

All directory names use singular or short forms. `reports` (plural) is renamed to `report`. `InferDocType()` gains a `"report"` case; the existing `"reports"` case is retained for backwards compatibility with projects that already use `work/reports/`.

Feature and plan review artefacts go in `work/review/` rather than the general `work/report/`. Review reports are produced by a specific lifecycle gate, follow a specific naming convention (`review-{id}-{slug}.md`), and are generated in quantity on active projects. Keeping them in a dedicated directory makes them easier to locate without a viewer tool.

Both directories map to document type `report` — the distinction is the directory, not the type.

#### Design decision FI-D-006: `work/retro/` uses a new `retrospective` document type

Retrospective synthesis documents are a meaningfully different kind of artefact from review reports. Reviews are point-in-time quality gate outputs tied to a specific feature or plan. Retrospectives are aggregate pattern analyses over a period of time, intended to improve future process. Distinguishing them at the type level (not just the directory) enables future tooling to filter and query them independently.

A new `retrospective` document type is introduced. `InferDocType()` gains a `"retro"` case → `"retrospective"`. `model.AllDocumentTypes()` is updated to include it. All validation that checks document types is updated accordingly.

#### 5.4.2 `work/README.md`

`kbz init` creates `work/README.md` as a static, human-readable directory map. It is not generated dynamically and is not updated by subsequent `init` runs. It is committed as part of the project.

Content template:

```markdown
# work/

Workflow documents for this project. Register all documents with kanbanzai after creation.

| Directory | Type | Contents |
|---|---|---|
| `design/` | design | Architecture decisions, technical vision, policies |
| `spec/` | specification | Acceptance criteria and binding contracts |
| `plan/` | plan | Project planning: roadmaps, scope, decision logs |
| `dev/` | dev-plan | Feature implementation plans and task breakdowns |
| `research/` | research | Analysis, exploration, background reading |
| `report/` | report | Audit reports, post-mortems, general reports |
| `review/` | report | Feature and plan review reports |
| `retro/` | retrospective | Retrospective synthesis documents |

AI agents: see the `kanbanzai-documents` skill for registration instructions.
```

#### 5.4.3 Backwards compatibility

`DefaultDocumentRoots()` changes for new installs only. Existing projects' `config.yaml` files are not modified. `InferDocType()` retains all existing cases (`"dev"`, `"reports"`, etc.) so batch imports of projects using old layouts continue to infer types correctly. The new cases (`"plan"`, `"report"`, `"retro"`) are purely additive.

---

### 5.5 Init command surface changes

**New files written on `kbz init` for a new project:**
- `.mcp.json`
- `.zed/settings.json` (only if `.zed/` already exists)
- `.kbz/context/roles/base.yaml`
- `.kbz/context/roles/reviewer.yaml`
- `work/README.md`
- Two additional skill directories: `kanbanzai-review` and `kanbanzai-plan-review`

**`--update-skills` flag scope expands:**
The flag is extended to also update managed role files (`reviewer.yaml`, and any future managed roles). Both skill files and managed role files carry the `kanbanzai-managed` marker and follow identical version-aware update logic. The flag may be renamed `--update-managed` in the specification to reflect its broadened scope — this is a decision for the spec to formalise.

**New `--skip-roles` flag:**
Skips creation of `.kbz/context/roles/` files. Complements the existing `--skip-skills`. Useful for projects that manage their roles separately or already have them.

**New `--skip-mcp` flag:**
Skips creation of `.mcp.json` (and `.zed/settings.json`). Useful when the user wants to configure their editor manually or needs a custom configuration.

---

## 6. What this does not change

- `context_assemble` does not change. It reads from `.kbz/context/roles/` (unchanged) and from knowledge entries. It does not read from skill files — this was a design intention that was never implemented and is not revived here.
- The document intelligence layer (`doc`, `doc_intel`) does not change. It operates on whatever document roots are registered in `config.yaml`.
- The `handoff` and `next` tools' skill delivery mechanism does not change.
- The six existing embedded skills continue to be installed (with updates to `kanbanzai-workflow` and `kanbanzai-documents` as described in §5.2).
- The kanbanzai project's own `.skills/` directory, `work/` layout, and role files are untouched by this work.

---

## 7. Summary of design decisions

| ID | Decision |
|---|---|
| FI-D-001 | `.mcp.json` is committed to the repository (PATH-relative command, no machine-specific content) |
| FI-D-002 | Skills are procedural (how to do X); roles are contextual conventions (what to know while doing X). No overlap. |
| FI-D-003 | `reviewer.yaml` carries the `kanbanzai-managed` marker and is updated by `--update-managed` |
| FI-D-004 | `work/plan/` (new `plan` type, human-facing) and `work/dev/` (existing `dev-plan` type, agent-facing) are separate directories |
| FI-D-005 | `work/report/` (singular) and `work/review/` (separate directory for review gate artefacts) both use document type `report` |
| FI-D-006 | `work/retro/` uses a new `retrospective` document type, distinct from `report` |
| FI-D-007 | `.mcp.json` is the primary discovery mechanism (tools); skills are the procedure layer. The getting-started skill description is self-identifying, not conditional. |