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

## 1. Problem Statement

Running `kbz init` on a new project leaves the developer in a broken state on four dimensions.

### 1.1 The MCP server is not connected

`kbz init` creates `.kbz/config.yaml`, the `work/` directories, and the `.agents/skills/` files, but produces no MCP server configuration. The editor does not know the server exists. When an agent reads the installed skills and follows the getting-started skill, it immediately calls `next` — a tool that does not exist because the server is not running. The failure mode depends on the editor (silent failure, hallucinated response, or an error) but none are helpful.

The `kanbanzai-1.0` design doc specified that `init` should generate an editor-specific config snippet. This was never implemented.

### 1.2 No context roles are installed

The `init` command creates `.kbz/config.yaml` and the `.kbz/` directory structure, but does not create `.kbz/context/roles/`. Agents calling `context_assemble` find nothing. The getting-started and agents skills reference role-based context assembly without acknowledging that roles must first exist.

The Phase 2b specification §11.8 describes a standard set of suggested profiles but leaves creation entirely to the user with no init-time scaffolding.

### 1.3 The `.skills/` directory is a design artifact with no architectural basis

The kanbanzai project has a `.skills/` directory containing `code-review.md`, `plan-review.md`, `document-creation.md`, and `README.md`. These files:

- Are **not** installed by `kbz init` (so new projects do not get them)
- Are **not** read by `context_assemble` (the original design said they would be; this was never implemented)
- Are **not** protected by the `kanbanzai-managed` versioning marker
- Overlap in content and purpose with the embedded skills in `.agents/skills/kanbanzai-*/`

The `kanbanzai-review` context profile installed in this project references "the review SKILL" — a reference that resolves only because `.skills/code-review.md` exists in this project. In any other kanbanzai-managed project, the reference is dangling.

The doc-currency health checker scans `.skills/*.md` for stale tool references — a coupling that only exists because `.skills/` was never cleaned up.

### 1.4 The default document layout is inconsistent

`DefaultDocumentRoots()` creates five directories on init: `work/design`, `work/spec`, `work/dev`, `work/research`, `work/reports`. This layout has three problems:

- `work/dev` is the wrong name. The kanbanzai project itself uses `work/plan/` for implementation plans and decision logs. The embedded `kanbanzai-documents` skill acknowledges this inconsistency by saying `work/plan/ or work/dev/`. The install creates a directory that the project's own skill says is optional.
- `work/reviews/` is used by both `.skills/code-review.md` and `.skills/plan-review.md` for review artefacts, but is not in `DefaultDocumentRoots()`. A fresh install never creates it, and it has no document type mapping.
- There is no place for retrospective synthesis documents (`retro` tool output), no guidance file for humans or agents about what goes where, and no singular naming convention (`reports` is plural; all other names are singular or short).

---

## 2. Goals

1. An agent opening a freshly initialised project can call `next` and get a meaningful response without any manual configuration.
2. A newly initialised project has working context profiles that any agent can assemble.
3. The code review and plan review procedures are available to any kanbanzai-managed project, not just this one.
4. The document layout created by `init` matches what the skills describe, matches how the kanbanzai project itself is organised, and is self-documenting for humans.
5. The conceptual distinction between **skills** (procedural) and **roles** (contextual conventions) is clear and consistently applied.

---

## 3. Non-Goals

- Editor auto-detection. We do not attempt to detect which editor is in use and generate editor-specific config. We generate a portable, widely-supported format.
- Zed-specific configuration. Zed uses a different context-server config format. We will document it in `docs/getting-started.md` but not generate it during init.
- A `developer.yaml` role template. The developer role is inherently language- and project-specific. Shipping a Go-flavoured developer role to a Python project would be worse than shipping nothing. Users create it themselves.
- Retroactively migrating existing projects. The changes to `DefaultDocumentRoots()` apply to new installs only. Existing `.kbz/config.yaml` files are not modified.

---

## 4. Design

### 4.1 MCP server connection: `.mcp.json`

`kbz init` writes a `.mcp.json` file at the project root. This format is supported by Claude Code, Cursor, and VS Code (with Copilot or Claude extensions) as the standard project-local MCP server declaration. It is safe to commit — no machine-specific paths are required.

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

The command is PATH-relative (`kanbanzai`, not `/usr/local/bin/kanbanzai`). This means the server starts only if the binary is on the user's PATH, which is the correct precondition.

`.mcp.json` is committed alongside the rest of the project. Any contributor who clones the repository gets the server configured automatically — the same way they get the skills files.

**On existing installs:** if `.mcp.json` already exists and does not carry the `kanbanzai-managed` marker, `init` skips it (same behaviour as the existing skill file conflict logic). If it does carry the marker, it is updated to the current format.

**Updating the getting-started skill:** the `kanbanzai-getting-started` skill gains a preflight note: before calling any tools, verify the kanbanzai MCP server is available. If tools are not accessible, the server is not running — consult `docs/getting-started.md` for setup instructions.

**`.gitignore` entry:** `kbz init` already appends to `.gitignore` (or creates it) for `.kbz/cache/` and `.kbz/local.yaml`. No change needed — `.mcp.json` is committed, not ignored.

#### Design decision FI-D-001: `.mcp.json` is committed, not gitignored

`.mcp.json` contains no machine-specific content when using a PATH-relative command. Committing it means every collaborator and every CI environment gets the server configured without manual steps. This is the same philosophy as committing `.agents/skills/` — the tool configuration is part of the repository.

If a user wants to use a different binary path, they can edit `.mcp.json` locally and add it to `.gitignore` themselves. The committed version serves as the correct default.

---

### 4.2 Skills consolidation: eliminate `.skills/`

The `.skills/` directory is eliminated. Its content is redistributed:

| Current file | Disposition |
|---|---|
| `.skills/code-review.md` | Becomes embedded skill `kanbanzai-review` in `.agents/skills/kanbanzai-review/SKILL.md` |
| `.skills/plan-review.md` | Becomes embedded skill `kanbanzai-plan-review` in `.agents/skills/kanbanzai-plan-review/SKILL.md` |
| `.skills/document-creation.md` | Content merged into `kanbanzai-documents`; file deleted |
| `.skills/README.md` | Deleted |

After this change, eight embedded skills are installed by `kbz init`: the existing six (`agents`, `design`, `documents`, `getting-started`, `planning`, `workflow`) plus the two new ones (`review`, `plan-review`).

The `kanbanzai-review` skill covers: review orientation, per-dimension evaluation guidance, structured output format, finding classification (blocking vs non-blocking), edge case handling, and the full orchestration procedure. It is the single canonical source for agents conducting feature-level code reviews.

The `kanbanzai-plan-review` skill covers: plan scope verification, feature completion checks, spec conformance, documentation currency, cross-cutting checks (tests, health), retrospective contribution, and the review report format.

The `kanbanzai-workflow` skill is updated to describe the `reviewing` and `needs-rework` feature lifecycle states, which it currently omits entirely. It directs agents to `kanbanzai-review` for the review procedure.

The `kanbanzai-documents` skill absorbs the step-by-step registration procedure and troubleshooting content from `document-creation.md`.

#### Design decision FI-D-002: skills are procedural; roles are contextual conventions

This distinction is made explicit here to prevent future drift:

- **Skills** answer "how do I do X?" They contain ordered steps, tool call sequences, output format templates, and edge case handling. A skill is task-type-specific but project-agnostic — any kanbanzai-managed project can follow the `kanbanzai-review` skill regardless of language or domain.

- **Roles** answer "what should I know while doing X?" They contain conventions, constraints, package scope, and architectural overview for a specific project. A role is inherently project-specific. The `developer.yaml` for a Go project will look nothing like one for a Rust project — this is correct.

Content that belongs in skills: procedure, tool call sequences, output templates, conditional logic.
Content that belongs in roles: coding standards, test conventions, package boundaries, architecture overview, commit format.

Content that should not be duplicated between the two: there is no reason for a skill to describe Go conventions, and no reason for a role to describe how to conduct a review.

#### Design decision FI-D-003: doc-currency health checker scans `.agents/skills/` instead of `.skills/`

The `doc_currency_health` Tier 1 check currently scans `.skills/*.md` for stale tool references. After this change, it scans `.agents/skills/kanbanzai-*/SKILL.md` instead. The `.skills/` scan is removed.

---

### 4.3 Default context roles

`kbz init` creates two role files in `.kbz/context/roles/`:

**`base.yaml` — installed as a scaffold**

The base role is created with placeholder content and comments that direct the project owner to fill in their conventions. It is deliberately empty of kanbanzai-system content — that content lives in the skills. The scaffold communicates the schema and expected content without imposing kanbanzai's Go conventions on every project.

```yaml
id: base
description: "Project-wide conventions for all agents"
# Add your project's global conventions below.
# All other roles inherit from base unless they declare their own.
conventions: []
# architecture:
#   summary: "One paragraph describing the overall project structure"
#   key_interfaces:
#     - "The most important files/packages and what they do"
```

**`reviewer.yaml` — installed fully populated**

The reviewer role is universal. Every project that uses the feature review gate needs the same review dimensions, output format conventions, and approach guidance. Unlike the developer role, there is no project-specific variation in how to conduct a structured review. The full content of the current `reviewer.yaml` is embedded in the binary and written on init.

**`developer.yaml` — not installed**

The developer role is intentionally omitted. It is language- and framework-specific. Installing a scaffold that a developer must fill in before it is useful adds friction without value. The `base.yaml` scaffold and the `kanbanzai-getting-started` skill both note that a `developer.yaml` can be created when conventions specific to a developer role are needed.

#### Design decision FI-D-004: `reviewer.yaml` is version-managed

Like the installed skill files, `reviewer.yaml` carries a `# kanbanzai-managed` marker and a version. `kbz init --update-skills` updates it (the flag is renamed or extended — see §4.5).

**On existing installs:** if `.kbz/context/roles/reviewer.yaml` already exists without the managed marker, `init` skips it and warns. If it carries the marker and is at an older version, it is updated.

---

### 4.4 Standard document layout

The document layout created by `kbz init` on a new project changes as follows.

#### 4.4.1 Proposed standard directories

| Directory | Document type | Purpose |
|---|---|---|
| `work/design/` | `design` | Architecture, vision, approach decisions, policies |
| `work/spec/` | `specification` | Acceptance criteria, binding contracts |
| `work/plan/` | `dev-plan` | Implementation plans, dev plans, decision logs, progress tracking |
| `work/research/` | `research` | Analysis, exploration, background |
| `work/report/` | `report` | Structured reports: review reports, audit reports, post-mortems |
| `work/retro/` | `retrospective` | Retrospective synthesis documents (output of the `retro` tool) |
| `work/README.md` | — | Human-readable map of the work/ directory (see §4.4.3) |

#### Design decision FI-D-005: `work/dev/` is replaced by `work/plan/`

`work/dev/` was the original name in `DefaultDocumentRoots()`. The kanbanzai project itself never used it — its own documents have always lived in `work/plan/`. The embedded `kanbanzai-documents` skill acknowledged the inconsistency by listing both. The `dev/` name is dropped. The document type (`dev-plan`) is unchanged; only the directory name changes.

`InferDocType()` gains a `"plan"` case → `"dev-plan"`. The existing `"dev"` case is retained for backwards compatibility with projects that already use `work/dev/`.

#### Design decision FI-D-006: `work/reports/` is renamed `work/report/` (singular)

All other directory names are singular or short (`design`, `spec`, `plan`, `research`). `reports` is the only plural. The rename makes naming consistent. `InferDocType()` gains a `"report"` case → `"report"`. The existing `"reports"` case is retained for backwards compatibility.

#### Design decision FI-D-007: `work/review/` is not in the standard layout; review artefacts go in `work/report/`

Feature review reports and plan review reports are a type of report artefact — they have document type `report` and are registered the same way as other reports. A separate `work/reviews/` (or `work/review/`) directory adds specificity without architectural benefit. The review skills (`kanbanzai-review`, `kanbanzai-plan-review`) are updated to write review reports to `work/report/` following the existing naming convention.

#### Design decision FI-D-008: `work/retro/` is introduced as a distinct directory for retrospective documents

Retrospective synthesis documents (produced by `retro(action: "report")`) are a different kind of artefact from review reports. Reviews are point-in-time quality gate outputs tied to a specific feature or plan. Retrospectives are aggregate pattern analyses over a period of time. Separating them makes the intent clear and keeps `work/report/` focused on structured artefacts with blocking/non-blocking findings.

A new document type `retrospective` is introduced for this directory. `InferDocType()` gains a `"retro"` case → `"retrospective"`. The `doc` tool's type validation is updated to accept `retrospective`.

#### 4.4.2 Backwards compatibility

`DefaultDocumentRoots()` changes for new installs only. The init command does not touch existing projects' `config.yaml`. Existing projects using `work/dev/`, `work/reports/`, or `work/reviews/` continue to work — those paths remain registered in their config. The `InferDocType()` function retains all old cases so batch imports of old-layout projects continue to infer types correctly.

#### 4.4.3 `work/README.md`

`kbz init` creates `work/README.md` as a human-readable directory map. It is a static file — not generated dynamically, not a living document. Content:

```markdown
# work/

This directory contains all workflow documents for this project.

| Directory | Type | Contents |
|---|---|---|
| `design/` | design | Architecture decisions, technical vision, policies |
| `spec/` | specification | Acceptance criteria and binding contracts |
| `plan/` | dev-plan | Implementation plans, dev plans, decision logs |
| `research/` | research | Analysis, exploration, background reading |
| `report/` | report | Review reports, audit reports, post-mortems |
| `retro/` | retrospective | Retrospective synthesis documents |

All documents must be registered with the kanbanzai system after creation.
AI agents: see the `kanbanzai-documents` skill for registration instructions.
```

This file is committed as part of the project. It is not managed or updated by `kbz init` on subsequent runs.

---

### 4.5 Init command surface changes

The init command gains the following behaviour changes:

**New files written on `init` for a new project:**
- `.mcp.json` (MCP server configuration)
- `.kbz/context/roles/base.yaml` (scaffold)
- `.kbz/context/roles/reviewer.yaml` (fully populated, managed)
- `work/README.md`

**`--update-skills` flag scope expands:**
The flag is renamed to `--update-managed` (or `--update-skills` is kept and its scope extended) to also update managed role files (currently just `reviewer.yaml`) alongside skill files. Both categories carry the `kanbanzai-managed` marker and follow the same version-aware update logic.

**New `--skip-roles` flag:**
Skips creation of `.kbz/context/roles/` files. Complements the existing `--skip-skills`. Useful for projects that manage their roles separately or already have them.

**New `--skip-mcp` flag:**
Skips creation of `.mcp.json`. Useful when the user wants to configure their editor manually or is using an editor (e.g. Zed) that requires a different format.

---

## 5. What this does not change

- The `context_assemble` tool does not change. It reads from `.kbz/context/roles/` (unchanged) and from knowledge entries. It does not read from skills files — this was a design intention that was never implemented and is not revived here.
- The document intelligence layer (`doc`, `doc_intel`) does not change. It continues to operate on whatever document roots are registered in `config.yaml`, regardless of their names.
- The `handoff` and `next` tools' skill delivery mechanism does not change.
- The six existing embedded skills continue to be installed unchanged (except for updates to `kanbanzai-workflow` and `kanbanzai-documents` as described in §4.2).

---

## 6. Migration for this repository

The kanbanzai project itself uses kanbanzai. The following one-time migration is required to align this repository with the new design:

1. Run `kbz init --update-managed` to install the new embedded skills and the managed role files (once implemented).
2. Delete `.skills/code-review.md`, `.skills/plan-review.md`, `.skills/document-creation.md`, `.skills/README.md` and the `.skills/` directory.
3. Rename `work/reports/` → `work/report/` and update all document records whose path contains `work/reports/`.
4. Create `work/retro/` and move any retrospective synthesis documents there.
5. Update `AGENTS.md` to remove references to `.skills/` and replace them with references to `.agents/skills/kanbanzai-review/` and `.agents/skills/kanbanzai-plan-review/`.
6. Update `work/plan/phase-*/` references across documents that point to `work/reports/` or `work/reviews/`.

---

## 7. Open questions

The following items require explicit decision before specification can begin. They are recorded here rather than resolved unilaterally.

**OQ-1: Zed support in `kbz init`**
Should `init` also create `.zed/settings.json` if a `.zed/` directory already exists in the repository? This would provide out-of-box support for Zed users without requiring them to consult the documentation. Risk: writing to `.zed/settings.json` would overwrite any existing Zed configuration unless we implement careful merge logic, which adds significant complexity.

**OQ-2: `work/review/` vs `work/report/`**
This design proposes that feature and plan review artefacts go in `work/report/` (§4.4.3, FI-D-007). An alternative is to keep a separate `work/review/` directory. The argument for separation: review reports are produced by a specific lifecycle gate and have a specific naming convention; mixing them with general reports may cause confusion. The argument against: the document type is the same (`report`), and the naming convention already distinguishes them. Which do you prefer?

**OQ-3: `work/dev/` retention**
This design removes `work/dev/` from `DefaultDocumentRoots()` in favour of `work/plan/`. Should `work/dev/` be deprecated with a warning when encountered by the health checker, or simply retained silently for backwards compatibility?

**OQ-4: `retrospective` as a new document type**
Introducing `retrospective` as a document type (for `work/retro/`) requires changes to document type validation across the storage and service layers. An alternative is to use type `report` for both `work/report/` and `work/retro/`, distinguished only by directory. Which is preferable?