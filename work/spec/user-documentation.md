# User Documentation: Specification

| Document | User Documentation Specification |
|----------|----------------------------------|
| Status   | Draft                            |
| Created  | 2026-03-26T14:24:58Z                       |
| Updated  | 2026-03-26T14:24:58Z                       |
| Related  | `work/design/kanbanzai-1.0.md` §8, §9 |

---

## 1. Purpose

This specification defines the documentation deliverables required for the Kanbanzai 1.0 release. It establishes what must be written, for whom, to what standard, and how completeness is verified. The documentation set lives in `docs/` and is published alongside the binary.

---

## 2. Goals

- Give a new user everything they need to install Kanbanzai, configure their editor, and run their first plan — without reading multiple documents.
- Give experienced collaborators a clear model of how the stage-gate workflow operates and what each stage produces.
- Give tool builders and advanced users a precise, complete reference for the `.kbz` format and all MCP tools.
- Ensure every document is written for humans, not agents.
- Ensure documentation reflects the editor-agnostic nature of the MCP server.

---

## 3. Scope

### 3.1 In scope

- Five documents in `docs/`: Getting Started, Workflow Overview, Schema Reference, MCP Tool Reference, and Configuration Reference.
- Coverage of all four supported editor integrations (Zed, Claude Desktop, VS Code, Cursor) in the Getting Started guide.
- Documentation of all MCP tools, parameters, and return values current at the 1.0 release.
- Complete `.kbz` format reference including all entity types, document records, and state files.
- Complete `config.yaml` field reference including prefix registry and local settings.
- End-to-end getting-started tutorial covering installation through first plan creation.

### 3.2 Deferred

- Video walkthroughs or screencast guides.
- Translated (non-English) versions.
- API client libraries or SDK documentation.
- Documentation for editors beyond the four listed (e.g. Neovim, Emacs, JetBrains IDEs).
- Searchable documentation site or hosted docs portal.

### 3.3 Explicitly excluded

- Agent-facing instructions (these are delivered via skills, not documentation).
- Editor-specific configuration embedded in the MCP server itself.
- Documentation for features not shipped in 1.0.
- Internal architecture documentation (lives in `work/`, not `docs/`).

---

## 4. Design Principles

**Editor-agnostic.** Kanbanzai's MCP server communicates over stdio using the MCP protocol. It has no dependency on any particular editor. Documentation must reflect this: no document should imply that one editor is required or preferred. Editor-specific setup instructions are presented as parallel options, not a hierarchy.

**One document to start.** A new user should not need to read multiple documents before they can do anything useful. The Getting Started guide is self-contained: it covers installation, editor configuration, `kanbanzai init`, and the first plan in sequence. Links to reference documents are provided for users who want more depth, but the tutorial does not depend on them.

**Written for humans.** Documentation uses plain prose, not bullet lists of parameters. Concepts are explained before they are used. The reader is assumed to be a software developer with no prior knowledge of Kanbanzai.

**Testable by reading.** Every procedural step must be executable as written. If a reader follows the Getting Started guide literally, they should arrive at a working setup. If a reference document lists a field, that field must exist in the current codebase.

**Accurate at release.** Documentation is written against the 1.0 implementation, not design documents. If the implementation differs from an earlier design document, the published documentation reflects the implementation.

---

## 5. Documents Required

### 5.1 Getting Started

**Audience:** New users — developers encountering Kanbanzai for the first time.

**Location:** `docs/getting-started.md`

**Purpose:** Get a new user from zero to a working setup with their first plan created, in a single document.

**Content requirements:**

- **Installation.** How to obtain the `kanbanzai` binary: building from source (`go install`), and any prebuilt release options available at 1.0. System requirements (Go version, OS).
- **What Kanbanzai is.** A short (two to three paragraph) plain-language explanation of what the system does and how it differs from a conventional project management tool. Written for a sceptical developer, not a product pitch.
- **Editor integration.** Separate, clearly labelled subsections for each supported editor:
  - **Zed** — MCP server config block to add to Zed settings.
  - **Claude Desktop** — MCP server config block for `claude_desktop_config.json`.
  - **VS Code** — setup for both the GitHub Copilot extension and the Claude extension, clearly distinguished.
  - **Cursor** — MCP server config block for Cursor's MCP settings.
  - Each subsection must show the exact configuration snippet required, with placeholders for any user-specific values (e.g. binary path).
  - Each subsection must state how to verify the integration is working (e.g. which tool to invoke, what response to expect).
- **Initialising a project.** Running `kanbanzai init` in a repository: what it does, what files it creates, how to confirm it succeeded.
- **First plan tutorial.** A step-by-step walkthrough covering:
  1. Creating a plan.
  2. Creating a feature under the plan.
  3. Creating a task under the feature.
  4. Checking the work queue.
  - The tutorial uses concrete, consistent example names throughout (not abstract placeholders like `<your-plan-name>`).
- **Next steps.** Brief signposts to the Workflow Overview and Configuration Reference for users who want to go deeper.

### 5.2 Workflow Overview

**Audience:** Human collaborators — team members who will interact with the system through documents and conversation.

**Location:** `docs/workflow-overview.md`

**Purpose:** Explain the stage-gate model: what each stage is, what it produces, and how humans and agents collaborate at each stage.

**Content requirements:**

- **The collaboration model.** A plain-language description of the human-AI split: humans own intent (goals, priorities, approvals, direction); agents own execution (decomposition, implementation, verification, status tracking). This framing should be established before any stage is described.
- **Stage overview table.** A table listing all six stages (Planning, Design, Features, Specification, Dev Plan and Tasks, Implementation) with columns for: stage name, what triggers it, what it produces, and the human approval gate.
- **Per-stage detail.** For each stage:
  - What happens in this stage.
  - Who drives it (human-led, agent-led, or collaborative).
  - What the output artifact is and where it lives.
  - What the approval gate is and who must pass it before the next stage begins.
  - What agents should and should not do at this stage.
- **Document-centric interface.** Explain that humans interact with the system through documents (writing and reviewing design docs, specifications, plans) rather than by managing entities directly.
- **Common failure modes.** A short section on what goes wrong when stages are skipped — specifically: creating tasks without an approved specification, making architecture decisions without a design document, conflating agent execution context with human workflow documents.

### 5.3 Schema Reference

**Audience:** Tool builders and advanced users who need to read or write `.kbz` state files directly, write integrations, or debug entity state.

**Location:** `docs/schema-reference.md`

**Purpose:** Complete, authoritative reference for the `.kbz` directory structure and all YAML entity formats.

**Content requirements:**

- **Directory layout.** Annotated tree of the `.kbz/` directory structure, explaining the purpose of each subdirectory and file.
- **YAML serialisation rules.** The canonical serialisation constraints: block style, double-quoted strings only when required, deterministic field order, UTF-8, LF line endings, trailing newline, no tags/anchors/aliases.
- **Entity types.** For each entity type (Plan, Feature, Task, Bug, Decision, KnowledgeEntry, DocumentRecord, Worktree, HumanCheckpoint, Incident):
  - All fields with name, type, required/optional status, and description.
  - Valid values for enumerated fields (e.g. lifecycle statuses).
  - Example YAML snippet showing a complete, valid entity.
- **Lifecycle state machines.** For each entity type with a lifecycle, a diagram or table of valid status transitions with the conditions or events that trigger each.
- **ID format.** The ULID-based ID allocation scheme, ID string format per entity type, and how IDs are allocated.
- **Plan ID format.** The `{prefix}{number}-{slug}` format, prefix registry, and constraints on prefix characters.
- **Referential integrity rules.** Which fields must reference valid entities of which types.

### 5.4 MCP Tool Reference

**Audience:** Agent developers and tool builders who interact with Kanbanzai through its MCP interface.

**Location:** `docs/mcp-tool-reference.md`

**Purpose:** Complete reference for every MCP tool exposed by the Kanbanzai server: parameters, behaviour, return values, and error conditions.

**Content requirements:**

- **Transport and protocol.** Brief description of the stdio MCP transport, protocol version, and how to connect. Include a note that the server is editor-agnostic.
- **Tool organisation.** Tools grouped by domain (entity management, document intelligence, knowledge management, worktree operations, estimation, work queue, orchestration, etc.).
- **Per-tool entries.** For each tool:
  - Tool name.
  - One-sentence description.
  - Parameters table: name, type, required/optional, description, valid values or constraints.
  - Return value: structure and fields.
  - Error conditions: what errors can be returned and under what circumstances.
  - At least one example call and response.
- **Lifecycle operation constraints.** Which tools trigger lifecycle transitions, and what preconditions must be satisfied.
- **Idempotency notes.** For tools that are idempotent (e.g. `batch_import_documents`), this must be stated explicitly.

### 5.5 Configuration Reference

**Audience:** All users who need to configure or customise a Kanbanzai instance.

**Location:** `docs/configuration-reference.md`

**Purpose:** Complete reference for all configuration files: `config.yaml`, `local.yaml`, and the prefix registry.

**Content requirements:**

- **`config.yaml`.** All fields with name, type, default value, and description. Example showing a complete minimal configuration and an example showing advanced options.
- **Prefix registry.** How to declare, use, and retire Plan ID prefixes. Constraints (single non-digit Unicode character, at least one active prefix must remain). Example registry block.
- **`local.yaml`.** Per-machine settings (user identity, etc.), which fields are available, and that this file is not committed to version control.
- **Context profiles.** Location of context role profiles (`context/roles/`), how to create or override them, and how inheritance works.
- **Environment interaction.** Any environment variables that affect Kanbanzai behaviour.
- **Validation.** How to validate configuration (`kbz config validate` or equivalent) and what errors look like.
- **Migration.** How configuration evolves between versions: what changes between phases, and how to update an existing instance.

---

## 6. Quality Standards

### 6.1 Tone and voice

- Plain, direct prose. Active voice. Present tense for descriptions of how things work.
- Second person ("you") for procedural instructions.
- No marketing language, no hedging ("might", "could potentially").
- Technical terms introduced before use; jargon-free where possible.

### 6.2 Completeness

- Every MCP tool in the 1.0 release must appear in the MCP Tool Reference.
- Every field in every entity type must appear in the Schema Reference.
- Every `config.yaml` field must appear in the Configuration Reference.
- The Getting Started tutorial must be executable without consulting any other document.

### 6.3 Accuracy

- All code snippets, configuration examples, and command invocations must be tested against the 1.0 binary before the documentation is considered complete.
- No document may describe behaviour not present in the 1.0 implementation.
- If a design document and the implementation differ, the published documentation reflects the implementation, and the discrepancy is noted for the team.

### 6.4 Navigability

- Each document begins with a brief statement of its audience and purpose.
- Documents link to each other where relevant (e.g. Getting Started links to Configuration Reference for advanced setup).
- Section headings are descriptive, not generic ("Configuring the Prefix Registry", not "Configuration").

### 6.5 Testability for readers

- Procedural sections (installation, editor setup, init, tutorial) must be written so that a developer following the steps literally arrives at the stated outcome.
- Reference sections must be verifiable by inspection: a reader should be able to confirm any claim by examining the binary or YAML files.

---

## 7. Acceptance Criteria

1. A `docs/` directory exists at the repository root containing exactly the five required documents: `getting-started.md`, `workflow-overview.md`, `schema-reference.md`, `mcp-tool-reference.md`, and `configuration-reference.md`.

2. The Getting Started guide contains installation instructions, editor setup subsections for all four supported editors (Zed, Claude Desktop, VS Code, Cursor), `kanbanzai init` walkthrough, and a first-plan tutorial — all within a single document and in that order.

3. Each editor subsection in the Getting Started guide includes a complete, copy-pasteable configuration snippet and a step to verify the integration is working after setup.

4. The Workflow Overview describes all six stage gates, identifies the human approval gate for each stage, and explains the human-AI collaboration split before describing any individual stage.

5. The Schema Reference documents every entity type present in the 1.0 implementation, including all fields, valid values for enumerated fields, and an example YAML snippet for each type.

6. The Schema Reference includes the lifecycle state machine (valid transitions) for every entity type that has a lifecycle status field.

7. The MCP Tool Reference contains an entry for every tool exposed by the 1.0 MCP server, and each entry includes a parameters table, return value description, and at least one example call and response.

8. The Configuration Reference documents every field in `config.yaml` and `local.yaml` present in the 1.0 implementation, with type, default value, and description for each field.

9. All command-line invocations, configuration snippets, and YAML examples in all five documents have been executed or validated against the 1.0 binary and produce the described output.

10. No document uses agent-facing framing (e.g. instructions addressed to AI agents, tool call syntax) — all five documents are written for human readers.

11. The Getting Started tutorial can be followed from a clean directory with no prior Kanbanzai knowledge and results in a working instance with at least one plan, one feature, and one task visible in the work queue.

12. All five documents are registered with the Kanbanzai document intelligence system (`doc_record_submit`) with status `approved` prior to the 1.0 release being tagged.