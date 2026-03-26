# Kanbanzai 1.0 Design

- Status: draft design
- Purpose: define what Kanbanzai 1.0 means — a distributable, installable workflow tool ready for use by projects other than itself
- Date: 2026-05-31
- Related:
  - `work/design/workflow-design-basis.md`
  - `work/design/document-centric-interface.md`
  - `work/design/agent-interaction-protocol.md`
  - `work/plan/phase-4-scope.md`
  - `work/plan/phase-4b-review.md`

---

## 1. Purpose

This document defines what Kanbanzai 1.0 means and what needs to be true before that label can be applied.

Through Phases 1–4, Kanbanzai has been built by and for its own development. It has never been installed by anyone other than its authors. It has never been used on a project other than itself. The binary is built from source. The configuration assumes a specific local environment. The agent instructions are embedded in a project-specific `AGENTS.md` that a new user would have no reason to create or know the contents of.

1.0 means: **someone else can use this**.

That someone is a team running a design-led, agent-assisted software project. They want to coordinate human designers, AI agents, and software developers through a shared, structured workflow. They have a git repository. They have an MCP-capable AI editor. They do not have a Go toolchain or any prior knowledge of Kanbanzai's internals.

1.0 is not the most powerful version of Kanbanzai. It is the first version that a stranger can install, configure, and use productively without assistance from the authors.

---

## 2. What Changes at 1.0

Today, Kanbanzai is a project that happens to produce a tool. At 1.0, it becomes a tool that is also a project.

The internal entity model, lifecycle state machines, MCP operations, document intelligence, and orchestration tools are all substantially complete. The gap is not features. The gap is the **boundary between the tool and the world**:

- How does a new user get the binary?
- How does a new project get configured?
- How does an AI agent in a new project know how to use the tool?
- How does a third-party product consume Kanbanzai's output?
- What interface does Kanbanzai expose to consumers that is stable and versioned?

These are the questions 1.0 must answer.

---

## 3. The Viewer Is a Separate Product

A recurring discussion has been whether Kanbanzai should include a web-based dashboard for non-technical stakeholders — designers, managers, and others who need visibility into project state without needing terminal access or MCP client configuration.

The decision is: **the viewer is a separate product, not part of Kanbanzai**.

The rationale:

**Separation of concerns.** Kanbanzai is a workflow engine: an MCP server, a CLI, and a structured file store. It is a clean Go binary with no frontend dependencies. Adding a web server, a JavaScript build pipeline, static assets, CSS, icons, and HTML templates would compromise this. A team using Kanbanzai only for agent coordination should not carry the weight of a web viewer they do not use.

**The schema is the interface.** The `.kbz` directory structure and YAML file formats are already a well-defined interface. Any tool that can read YAML and understands the schema can be a viewer. The viewer does not need to call Kanbanzai APIs or depend on Kanbanzai's Go packages. It just needs to understand the file format.

**Git is the transport.** The workflow is git-native. Workflow state does not change until it is committed and pushed. A viewer that pulls from git is always in sync with the shared state of the project by definition. No API server, no real-time sync protocol, and no running Kanbanzai process is required for read-only viewing.

**Independent release cadence.** A viewer has different requirements from the workflow engine: frontend frameworks change, design trends evolve, platform targets differ. Decoupling the release cycles means Kanbanzai can be stable and the viewer can iterate independently.

**Commercial potential.** A viewer that is a distinct, distributable product (desktop app, web server, or both) is easier to consider as a commercial offering. Kanbanzai as an open-source workflow engine and a commercial viewer is a well-understood model.

What 1.0 must do to enable the viewer is covered in §6.

---

## 4. Skills-Based Onboarding

### 4.1 The Problem

An AI agent working in a new Kanbanzai-managed project needs to know:

- That Kanbanzai is in use and what it does
- The workflow stage gates and when each stage requires human approval
- How to collaborate with the human during planning and design
- How to use `context_assemble` to get role-specific instructions
- How documents are registered and managed
- How to write commits and interact with the workflow system

Today, all of this lives in a large `AGENTS.md` that is specific to the Kanbanzai project. A new project would have no such file, and writing one from scratch is an unreasonable onboarding requirement.

### 4.2 The Solution: Installed Skills

`kanbanzai init` installs a set of skill files into the project's `.skills/` directory. Skills are markdown files that AI editors and agent runtimes discover and read automatically. They are the standard mechanism for providing procedural and contextual knowledge to AI agents, and are supported across all major AI-enabled editors.

Skills installed by `kanbanzai init`:

| File | Purpose |
|---|---|
| `.skills/kanbanzai-getting-started.md` | Bootstrap: what to do at the start of any agent session |
| `.skills/kanbanzai-workflow.md` | Stage gates, lifecycle, when to stop and ask the human |
| `.skills/kanbanzai-planning.md` | How to run a planning conversation; scope before design |
| `.skills/kanbanzai-design.md` | How to collaborate on a design document; draft, surface options, get approval |
| `.skills/kanbanzai-documents.md` | Document types, registration, approval workflow |
| `.skills/kanbanzai-agents.md` | Agent interaction protocol, commits, knowledge entries, context assembly |

These files are authoritative and are not intended to be edited by users. They are versioned with the Kanbanzai binary and should be updated when the binary is updated.

### 4.3 Skills vs. AGENTS.md

`AGENTS.md` is a project-owned file. Its contents describe the project — its conventions, its structure, its decisions. Kanbanzai must not overwrite or significantly modify an existing `AGENTS.md` when initialising a mature project.

Skills are tool-owned files. Their contents describe how to use Kanbanzai. They live in `.skills/` with a `kanbanzai-` prefix to avoid conflict with other skills the project may already define.

A project's `AGENTS.md` may reference the Kanbanzai skills directory, but this is optional. Editors and agents discover `.skills/` independently.

### 4.4 Skills and Sub-Agents

Top-level agents running inside an editor discover `.skills/` automatically. Sub-agents spawned programmatically do not. To ensure sub-agents receive workflow knowledge, `context_assemble` includes relevant skill content in its context packet, filtered by role and current task stage. This means workflow instructions reach all agents regardless of how they are spawned, without requiring the parent agent to manually propagate context.

### 4.5 Skill Updates

When `kanbanzai init` is run in a project that already has Kanbanzai skills installed:

- If the file carries a `kanbanzai-managed` marker and the version matches → no action needed
- If the file carries a `kanbanzai-managed` marker and the version is older → overwrite with the current version
- If the file exists but has no `kanbanzai-managed` marker → error and stop; do not overwrite

This makes skills safely idempotent for standard use while protecting against accidental overwriting of user-created files.

---

## 5. Distribution and Installation

### 5.1 Pre-Compiled Binaries

Kanbanzai must be installable without a Go toolchain. Distribution is via pre-compiled binaries published to GitHub Releases on every tagged version.

Target platforms, in priority order:

| Priority | Platform | Notes |
|---|---|---|
| 1.0 alpha | macOS (ARM64) | Primary development target |
| 1.0 beta | macOS (AMD64), Linux (AMD64), Linux (ARM64) | Dev machines and servers |
| 1.0 | Windows (AMD64) | Full coverage |

Each release includes a checksum file for verification. Archives are `.tar.gz` for macOS and Linux, `.zip` for Windows.

Release automation uses GoReleaser and GitHub Actions, triggered on version tags.

### 5.2 Installation Experience

A new user should be able to install Kanbanzai with a single command. The exact mechanism (direct download, install script, package manager) is an implementation detail, but the outcome is: one command, then `kanbanzai` is available in `PATH`.

Homebrew tap support is desirable but deferred past 1.0 alpha.

### 5.3 Initialising a New Project

```
kanbanzai init
```

Run once in a project directory (which must already be a git repository). Creates:

- `.kbz/config.yaml` — project configuration with a default prefix registry and schema version
- `.skills/kanbanzai-*.md` — the six workflow skill files

Does not modify any existing project files. Safe to run on a project that already has `AGENTS.md`, existing `.skills/` files, or any other configuration.

If `.kbz/` already exists, `init` reports the existing version and offers `--update-skills` to refresh skill files to the current version.

---

## 6. Onboarding an Existing Project

### 6.1 The Problem

`kanbanzai init` will often be run on a project that already exists. That project may have years of design documents, specifications, meeting notes, and decisions scattered across one or more directories. It may have an established `docs/`, `wiki/`, `design/`, or `rfcs/` directory. It may not use `work/` at all.

Kanbanzai must not assume a particular document directory structure. It must not clobber existing files. And it must not silently import hundreds of documents that were never intended to be workflow documents — README files, API documentation, changelogs, and code comments are not design documents.

### 6.2 Document Location: `work/` as the Default

Kanbanzai uses `work/` as the default root for workflow documents. This is a deliberate nudge, not an arbitrary choice.

Workflow documentation is voluminous and in-progress by nature: hundreds of design drafts, specification iterations, planning documents, research notes, and decision logs accumulate over the life of a project. This material is essential to the workflow but is not the same as the project's user-facing documentation — the clean, polished, publicly visible `docs/` that describes the software to its users.

Keeping these two concerns separate gives teams a clean `docs/` folder they can be proud of, while workflow noise is contained in `work/` where it belongs. Agents, tools, and context assembly all look in `work/` by default, so nothing is lost by the separation.

The recommended layout within `work/`:

| Directory | Document type | Contents |
|---|---|---|
| `work/design/` | design | Design documents, architecture proposals, policy documents |
| `work/spec/` | specification | Formal specifications with acceptance criteria |
| `work/plan/` | dev-plan | Implementation plans, decision logs, progress tracking |
| `work/research/` | research | Background research, analysis, exploration |
| `work/reports/` | report | Review reports, audit reports, post-implementation reviews |

This layout is a recommendation. Kanbanzai does not enforce it. The document path recorded in `.kbz/state/documents/` is a relative path from the repository root and can be anything. What matters is the document type recorded at registration time, not the path.

Projects that prefer a different structure — `rfcs/`, `docs/design/`, or a flat layout — can configure their document roots explicitly (see §6.3) and the tool will work equally well.

### 6.3 Document Roots in Configuration

To help agents and tools know where to find documents without scanning the entire repository, `config.yaml` records the project's document directories. The default, written by `kanbanzai init`, reflects the `work/` layout:

```yaml
version: "2"
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    - path: work/spec
      default_type: specification
    - path: work/plan
      default_type: dev-plan
    - path: work/research
      default_type: research
    - path: work/reports
      default_type: report
```

Projects that use a different layout override this in `config.yaml`. For example, a project that keeps workflow documents in `docs/`:

```yaml
documents:
  roots:
    - path: docs/design
      default_type: design
    - path: docs/specs
      default_type: specification
```

This configuration:

- Tells agents where to look when asked to find or import documents
- Provides a default type for `batch_import_documents` when no explicit type is given
- Tells the viewer where documents live in the repository
- Is used by `context_assemble` to surface relevant document context

For new projects, `init` writes the default `work/`-based roots and creates the directories. For existing projects, `init` asks where the project's documents live and records the answer instead.

### 6.4 What `init` Does and Does Not Do

`init` sets up infrastructure. It does not make knowledge decisions.

**`init` does:**
- Create `.kbz/config.yaml` with schema version, default prefix, and recorded document roots
- Install `.skills/kanbanzai-*.md`
- Ask where documents currently live (interactively, or via flags)
- Record those paths in `config.yaml`

**`init` does not:**
- Automatically scan and import existing documents
- Decide which documents are workflow documents
- Modify existing files
- Create document records in `.kbz/state/documents/`

Automatic import is the wrong tool for this job. A project with 200 markdown files may have 20 that are relevant workflow documents. An agent, working with the human, can make that distinction. A batch import script cannot.

### 6.5 The Onboarding Path for Existing Projects

After running `init`, the getting-started skill guides the user through document import:

1. Run `kanbanzai init` — infrastructure is ready
2. Open your AI editor and start a session
3. The agent reads the skills and knows the document roots from `config.yaml`
4. Ask the agent: *"I have existing documents in `docs/`. Help me import the relevant ones."*
5. The agent browses the directory, reads document contents, and proposes what to import and with what type
6. The human confirms, adjusts, or skips
7. The agent calls `doc_record_submit` for each confirmed document

This is consistent with the document-centric model: the human owns the decision about what is a workflow document. The agent does the mechanical work of reading, classifying, and registering.

For projects where the answer is simply "import everything in this directory", `batch_import_documents` remains available as a shortcut, with the human specifying the path and default type.

---

## 7. The `.kbz` Schema as a Public Interface

### 7.1 Current Status

The `.kbz` directory structure and YAML file formats are currently an implementation detail. They are tested (round-trip serialisation), versioned (via `config.yaml`), and stable, but they are not documented as a contract for external consumers.

For 1.0, they become a contract.

### 7.2 What the Schema Covers

The public schema includes:

- Directory layout under `.kbz/state/`
- File naming conventions for each entity type
- Required and optional YAML fields per entity type
- Valid values for enumerated fields (status, severity, type, etc.)
- Lifecycle state machines: valid states and valid transitions
- Referential integrity rules (which fields reference other entity IDs)
- Document record format
- Knowledge entry format
- Config file format

### 7.3 The Schema Library

For consumers written in Go — including the viewer — Kanbanzai makes its canonical type definitions, field validation, and YAML parsing available for import. External Go projects can depend on these directly rather than reimplementing `.kbz` parsing independently.

The exposed interface covers read-oriented access: parsing a `.kbz` directory, enumerating entities, resolving references, reading document records. Write operations — lifecycle enforcement, referential integrity checks, ID allocation — are not part of the public interface and remain internal to the Kanbanzai binary.

For consumers not written in Go, a JSON Schema document is generated from the Go types as a build artifact and published alongside each release. This provides a machine-readable schema definition that any language can validate against, without making Go a requirement.

The exact packaging (whether types live in a separate module or in exported packages of the main module) is an implementation decision.

### 7.4 Versioning and Compatibility

The schema version is recorded in `.kbz/config.yaml`. The schema module is versioned with semantic versioning. The compatibility policy:

- **Patch versions**: bug fixes, clarifications, no field changes
- **Minor versions**: new optional fields, new entity types, new valid status values — backward compatible
- **Major versions**: removed fields, renamed fields, changed field semantics, changed lifecycle rules — breaking

A Kanbanzai binary will refuse to operate on a `.kbz` directory whose schema version is newer than it understands, and will offer a migration command when operating on an older schema version.

---

## 8. Editor Independence

Kanbanzai's MCP server communicates over stdio using the MCP protocol. This is editor-agnostic by design. Any MCP-capable client can use it.

For 1.0, the documentation must cover setup for the editors most likely to be used by the target audience:

- Zed (current development environment)
- Claude Desktop
- VS Code with GitHub Copilot or Claude extension
- Cursor

The MCP server configuration format (`.mcp.json` or equivalent) varies by editor. Kanbanzai's configuration should not assume any particular editor. The `init` command generates a configuration snippet for the user's current editor if it can be detected, otherwise provides instructions for each supported editor.

Agent behaviour instructions are delivered through skills (see §4), not through editor-specific configuration files. This ensures consistent agent behaviour regardless of which editor is in use.

---

## 9. Documentation

The `docs/` directory, reserved through all previous phases, is populated for 1.0.

Required documentation:

| Document | Audience | Purpose |
|---|---|---|
| Getting started | New users | Installation, editor configuration, `kanbanzai init`, first plan — end to end in one place |
| Workflow overview | Human collaborators | The stage gate model, what each stage produces, how humans and agents collaborate |
| Schema reference | Tool builders, advanced users | Complete `.kbz` format reference |
| MCP tool reference | Agent developers | All MCP tools, parameters, return values |
| Configuration reference | All users | `config.yaml` fields, prefix registry, local settings |

Installation, basic configuration, editor setup guides, and the getting-started tutorial are combined into the single Getting Started document. A new user should not need to read multiple documents before they can do anything useful.

Documentation is written for humans, not agents. It lives in `docs/` and is published alongside the binary.

---

## 10. Hardening

1.0 must be robust enough that a first-time user does not hit a wall.

**Error messages**: All user-facing errors explain what went wrong and what to do next. Technical internals (YAML field names, Go type names, stack traces) do not appear in user-facing output.

**Clean-machine behaviour**: All features are tested on a machine with no pre-existing Kanbanzai state. No assumptions about accumulated configuration, cached data, or pre-existing `.kbz/` directories.

**Edge case handling in `init`**: Running `init` in a non-git directory, a directory with existing `.kbz/`, a directory with no write permission, or a directory with conflicting `.skills/` files — all produce clear, actionable errors rather than silent failures or partial state.

**CLI help and discoverability**: `kanbanzai --help` and `kanbanzai <command> --help` are sufficient for a new user to understand available commands and their purpose without reading documentation.

**Partial state recovery**: If an operation is interrupted (disk full, process killed, network failure), the tool detects and reports the partial state on next invocation rather than silently operating on corrupted data.

---

## 11. What 1.0 Does Not Include

The following are explicitly deferred:

- **A web viewer or dashboard** — this is a separate product (see §3)
- **Homebrew or other package manager distribution** — GitHub Releases is sufficient for alpha
- **Multi-platform GUI or app packaging** — deferred to the viewer project
- **GitLab, Bitbucket, or other platform support** — GitHub only for 1.0
- **Semantic search or embedding-based retrieval** — not in scope
- **Hosted or SaaS deployment** — self-hosted only
- **Write operations through any interface other than MCP and CLI** — the viewer is read-only by design
- **Authentication or authorisation** — delegated to the git hosting platform and deployment environment

---

## 12. The Viewer Project as Validation

The viewer is not part of Kanbanzai 1.0, but it is the first proof that 1.0 works.

The viewer project will:

1. Start in a fresh git repository with no prior Kanbanzai state
2. Run `kanbanzai init` — testing the installation and onboarding experience
3. Use the schema library to read `.kbz` entity state — testing the public schema interface
4. Be managed using Kanbanzai itself — testing the self-management workflow end-to-end
5. Be developed by agents using the installed skills — testing that skills are sufficient for a new project

If the viewer project can be started, run, and completed using only the public 1.0 interface, 1.0 is ready.

This is the 1.0 acceptance criterion at the highest level.