# Agent Onboarding and Skill Discovery Specification

| Document | Agent Onboarding and Skill Discovery Specification |
|----------|-----------------------------------------------------|
| Status   | Draft                                               |
| Created  | 2026-03-30                                          |
| Plan     | P12-agent-onboarding                                |
| Design   | `work/design/agent-onboarding.md`                   |
| Related  | `work/spec/init-command.md` (supersedes §3.3 re AGENTS.md exclusion) |
|          | `work/design/fresh-install-experience.md` (P11)     |
|          | `.agents/skills/kanbanzai-getting-started/SKILL.md` |
|          | `.agents/skills/kanbanzai-workflow/SKILL.md`        |

---

## 1. Purpose

This specification defines the acceptance criteria for closing the skill discovery
gap identified in first real-world usage of kanbanzai. Agents with kanbanzai MCP
tools available did not use them because the skills explaining the workflow protocol
were installed to `.agents/skills/` — a location not read by all editor platforms.

This specification covers five features:

| ID | Label | Scope |
|---|---|---|
| A | agents-md | Generate `AGENTS.md` on `kbz init` |
| B | copilot-instr | Generate `.github/copilot-instructions.md` on `kbz init` |
| C | skill-updates | Add "use MCP tools" rule to existing skills |
| D | spec-skill | New `kanbanzai-specification` skill |
| E | mcp-breadcrumbs | Orientation field in `status` and `next` responses |

---

## 2. Supersession

This specification supersedes the following clause in `work/spec/init-command.md` §3.3:

> `AGENTS.md` — never created or modified by `init`.

After this work, `kbz init` will create and version-manage `AGENTS.md`. The remainder
of the init command specification is unaffected.

---

## 3. Feature A: AGENTS.md Generation

### 3.1 File location and name

`kbz init` MUST write `AGENTS.md` at the project root (the directory containing `.kbz/`).

### 3.2 Content requirements

The generated `AGENTS.md` MUST contain all of the following sections, in order:

1. **Managed marker** — an HTML comment on the first line:
   `<!-- kanbanzai-managed: v1 -->`
   This marker is invisible to agents reading the file as Markdown but machine-parseable
   by the version-aware conflict logic.

2. **Title** — a level-1 heading identifying this as agent instructions for a
   kanbanzai-managed project.

3. **Before You Do Anything** — three numbered steps:
   - Call `status` to see the current project state
   - Call `next` to see the work queue
   - Read `.agents/skills/kanbanzai-getting-started/SKILL.md` for full orientation

4. **Rules** — at minimum, these three rules:
   - Use kanbanzai MCP tools (`status`, `next`, `entity`, `doc`, `finish`) for all
     workflow operations. Do not create or modify entities or documents by writing
     files directly.
   - Follow the stage gates: Planning → Design → Features → Specification →
     Dev plan → Implementation. Skipping forward is not allowed.
   - Human approval is required at stage gates. When in doubt, ask.

5. **Skills reference** — a table listing all installed skills with their name and
   a brief "when to read" description. The table MUST include every skill installed
   by `kbz init` at the time of generation.

### 3.3 Content constraints

The generated file MUST NOT:
- Exceed 50 lines (including blank lines and the marker comment)
- Duplicate the content of any skill file — it is a pointer, not a policy document
- Include project-specific conventions (language, framework, etc.)

### 3.4 Version-aware conflict logic

On `kbz init`, the AGENTS.md writer MUST apply the following logic:

| Condition | Action |
|---|---|
| File does not exist | Create with generated content |
| File exists, first line is `<!-- kanbanzai-managed: vN -->` where N < current | Overwrite with generated content |
| File exists, first line is `<!-- kanbanzai-managed: vN -->` where N >= current | No-op (already current) |
| File exists, no managed marker on first line | Skip. Print warning: "AGENTS.md exists and is not managed by kanbanzai. Add the kanbanzai workflow section manually. See docs/getting-started.md." |

### 3.5 CLI flag

A new `--skip-agents-md` boolean flag MUST be added to `kbz init`. When set:
- AGENTS.md is not written
- `.github/copilot-instructions.md` is not written (Feature B is gated on this flag too)

Default: `false`.

### 3.6 Init integration

AGENTS.md MUST be written in both `runNewProject` and `runExistingProject` code paths,
at the same point in the sequence where `.mcp.json` is written (after skills, before
the sentinel file). The `--skip-agents-md` flag MUST gate both Feature A and Feature B
writes.

### 3.7 Idempotency

Running `kbz init` twice on the same project with no changes between runs MUST NOT
modify AGENTS.md (no file write, no stdout message).

### 3.8 Acceptance criteria

- AC-A1: `kbz init` on a new project creates `AGENTS.md` at the project root.
- AC-A2: The generated file starts with `<!-- kanbanzai-managed: v1 -->` on line 1.
- AC-A3: The file contains the "Before You Do Anything" section with `status`, `next`,
  and skill path references.
- AC-A4: The file contains the three rules (MCP tools, stage gates, human approval).
- AC-A5: The file contains a skills reference table listing all installed skills.
- AC-A6: The file does not exceed 50 lines.
- AC-A7: Running `kbz init` on a project with an existing kanbanzai-managed AGENTS.md
  at the current version does not modify the file.
- AC-A8: Running `kbz init` on a project with an existing kanbanzai-managed AGENTS.md
  at an older version overwrites it.
- AC-A9: Running `kbz init` on a project with an existing non-managed AGENTS.md prints
  a warning and does not modify the file.
- AC-A10: `kbz init --skip-agents-md` does not create AGENTS.md.
- AC-A11: The generated content is embedded in the binary (not read from disk at runtime).

---

## 4. Feature B: Copilot Instructions Generation

### 4.1 File location and name

`kbz init` MUST write `.github/copilot-instructions.md` inside the project root.

### 4.2 Directory creation

If the `.github/` directory does not exist, it MUST be created. If it already exists,
only the instructions file is written or updated — no other files in `.github/` are
modified.

### 4.3 Content requirements

The generated file MUST contain:

1. **Managed marker** — `<!-- kanbanzai-managed: v1 -->` on line 1.
2. **Title** — a level-1 heading identifying these as Copilot instructions.
3. **Pointer to AGENTS.md** — an explicit instruction to read `AGENTS.md` in the
   project root before doing any work.
4. **Quick reference** — the three essential first steps (call `status`, call `next`,
   read the getting-started skill) and the "use MCP tools" rule.

### 4.4 Content constraints

The file MUST NOT:
- Exceed 25 lines
- Duplicate the AGENTS.md content — it is a redirect, not a copy
- Contain any kanbanzai-specific rules that are not also in AGENTS.md

### 4.5 Version-aware conflict logic

Same pattern as Feature A §3.4, applied to `.github/copilot-instructions.md`:

| Condition | Action |
|---|---|
| File does not exist | Create with generated content |
| File exists, managed marker present, version < current | Overwrite |
| File exists, managed marker present, version >= current | No-op |
| File exists, no managed marker | Skip with warning |

### 4.6 CLI flag

Feature B is gated on the same `--skip-agents-md` flag as Feature A. There is no
separate flag for copilot instructions alone.

### 4.7 Acceptance criteria

- AC-B1: `kbz init` on a new project creates `.github/copilot-instructions.md`.
- AC-B2: The `.github/` directory is created if it does not exist.
- AC-B3: The generated file starts with `<!-- kanbanzai-managed: v1 -->` on line 1.
- AC-B4: The file contains an explicit instruction to read `AGENTS.md`.
- AC-B5: The file does not exceed 25 lines.
- AC-B6: An existing non-managed `.github/copilot-instructions.md` is not modified;
  a warning is printed.
- AC-B7: `kbz init --skip-agents-md` does not create the file.
- AC-B8: If `.github/` already exists with other files, only `copilot-instructions.md`
  is affected — no other files in `.github/` are created, modified, or deleted.

---

## 5. Feature C: Skill Content Updates

### 5.1 Getting-started skill update

In `.agents/skills/kanbanzai-getting-started/SKILL.md`, a new rule MUST be added to
the "Session Start" section (between the existing steps and the "Related" section):

**Title:** "Use kanbanzai MCP tools for all workflow operations"

**Content must convey:** Do not create documents by writing files directly with
`edit_file` or equivalent — this bypasses lifecycle enforcement, document registration,
and health checks. Use `doc` to register and manage documents. Use `entity` to create
and transition entities. If MCP tools are unavailable, report the issue rather than
falling back to direct file writes.

### 5.2 Workflow skill update

In `.agents/skills/kanbanzai-workflow/SKILL.md`, a new item MUST be added to the
"Emergency Brake" bullet list:

**Content must convey:** Stop and ask the human when you are about to create a
document in `work/` or an entity in `.kbz/state/` without using the corresponding
kanbanzai MCP tool (`doc`, `entity`). Direct file writes bypass lifecycle enforcement.

### 5.3 Content constraints

- The updates MUST be additions, not replacements — existing skill content is preserved.
- The added text MUST be consistent in tone and formatting with the surrounding content.
- No other sections of either skill file are modified.

### 5.4 Embedding

Since skills are embedded in the binary, the source files under `internal/kbzinit/skills/`
are the files to modify. Changes take effect on the next `kbz init` or
`kbz init --update-skills` run in target projects.

### 5.5 Acceptance criteria

- AC-C1: The getting-started skill contains a rule about using MCP tools instead of
  direct file writes.
- AC-C2: The workflow skill's Emergency Brake section includes direct file writes as
  a condition to stop and ask.
- AC-C3: All existing content in both skills is preserved — no deletions or
  reorganisation.
- AC-C4: The updated skills pass the doc-currency health checker (no stale tool
  references introduced).
- AC-C5: `kbz init --update-skills` on a target project updates both skill files to
  the new version.

---

## 6. Feature D: Specification Skill

### 6.1 File location

A new skill MUST be created at:
`.agents/skills/kanbanzai-specification/SKILL.md`

It MUST be added to the `skillNames` list in `internal/kbzinit/skills.go` and embedded
in the binary.

### 6.2 YAML frontmatter

The skill MUST include the standard YAML frontmatter with:
- `name: kanbanzai-specification`
- `description:` — a description following the same activation-trigger pattern used
  by other skills (when to activate, what questions it answers)
- `metadata.kanbanzai-managed: "true"`
- `metadata.version: "0.2.0"` (matching the current skill version)

### 6.3 Required sections

The skill MUST contain all of the following sections:

#### 6.3.1 Purpose
Guide the specification process from an approved design to an approved specification,
ready for dev plan and task decomposition.

#### 6.3.2 When to Use
- After a design document has been approved
- When writing or reviewing acceptance criteria
- When assessing whether a specification is ready for approval
- When the design leaves implementation questions that the spec must resolve

#### 6.3.3 Roles
- **Human:** Product Owner — owns acceptance criteria, decides what "done" means,
  approves the specification.
- **Agent:** Specification Writer — proposes structure, drafts testable criteria,
  identifies edge cases, flags ambiguities.

#### 6.3.4 The Specification Process
An iterative flow: design is approved → agent drafts the spec → human reviews →
agent revises → repeat until no open questions → human approves → workflow moves
to dev plan.

#### 6.3.5 What a Good Specification Contains
At minimum:
- Scope (what is being specified and what is excluded)
- Acceptance criteria (each independently testable and verifiable)
- Constraints (performance, compatibility, security requirements)
- Out-of-scope items (explicit exclusions to prevent scope creep)

#### 6.3.6 Acceptance Criteria Quality Bar
Every acceptance criterion in the specification MUST be:
- **Testable** — an unambiguous pass/fail determination is possible
- **Independent** — verifiable without reference to other criteria
- **Specific** — names the exact behaviour, input, output, or state

The skill MUST state that "it works correctly" is not an acceptance criterion.

#### 6.3.7 The Approved Specification Invariant
A specification is ready for approval when:
- All acceptance criteria meet the quality bar
- No unresolved questions remain
- Scope matches the approved design (no additions, no omissions)
- A single direction is chosen (no alternatives)

#### 6.3.8 Relationship to Design
The specification operationalises the design. The design says *what to build and why*.
The specification says *what to verify and how to know it's done*. If the spec
contradicts the design, the design governs — surface the conflict to the human.

#### 6.3.9 Gotchas
At minimum:
- Forgetting to register the specification document
- Approving with ambiguous criteria (the downstream agent will interpret them
  differently than intended)
- Adding scope not in the approved design (the emergency brake should fire)
- Editing an approved specification (same rules as editing an approved design —
  supersede, don't silently modify)

#### 6.3.10 Related
Links to: `kanbanzai-design`, `kanbanzai-workflow`, `kanbanzai-documents`,
`kanbanzai-agents`.

### 6.4 Content constraints

- The skill MUST NOT exceed 200 lines (consistent with other skills).
- The skill MUST follow the same structure, tone, and formatting conventions as
  `kanbanzai-design/SKILL.md`.
- The skill MUST NOT duplicate content from the workflow skill's stage gates table —
  it should reference it.

### 6.5 Acceptance criteria

- AC-D1: The file exists at `.agents/skills/kanbanzai-specification/SKILL.md` in the
  embedded skills and is installed by `kbz init`.
- AC-D2: The skill appears in the `skillNames` list in `internal/kbzinit/skills.go`.
- AC-D3: The YAML frontmatter contains the correct name, description, and managed
  metadata.
- AC-D4: All required sections from §6.3 are present.
- AC-D5: The acceptance criteria quality bar section explicitly states that "it works
  correctly" is not an acceptance criterion.
- AC-D6: The file does not exceed 200 lines.
- AC-D7: The skill references `kanbanzai-design`, `kanbanzai-workflow`,
  `kanbanzai-documents`, and `kanbanzai-agents` in its Related section.
- AC-D8: The generated AGENTS.md (Feature A) lists this skill in its skills reference
  table.
- AC-D9: The skill passes the doc-currency health checker.

---

## 7. Feature E: MCP Orientation Breadcrumbs

### 7.1 Status tool (project overview)

When `status` is called with no `id` parameter (project overview mode), the response
object MUST include an `orientation` field:

```json
{
  "orientation": {
    "message": "This is a kanbanzai-managed project. For workflow guidance, read .agents/skills/kanbanzai-getting-started/SKILL.md",
    "skills_path": ".agents/skills/"
  }
}
```

The `orientation` field MUST always be present in the project overview response. It
is not conditional on project state.

### 7.2 Next tool (empty queue)

When `next` is called with no `id` parameter and the work queue is empty (zero ready
tasks), the response object MUST include the same `orientation` field as §7.1.

When the queue is non-empty, the `orientation` field MUST NOT be included — the agent
is already working and does not need orientation.

### 7.3 Next tool (claim mode)

When `next` is called with an `id` parameter (claim mode), the `orientation` field
MUST NOT be included. The context packet returned by claim mode already contains
assembled instructions.

### 7.4 Field structure

The `orientation` field MUST be a JSON object with exactly two keys:
- `message` (string): a one-sentence instruction directing the agent to the
  getting-started skill.
- `skills_path` (string): the relative path to the skills directory.

### 7.5 Backward compatibility

The `orientation` field is additive — it is a new top-level key in an existing
response object. Existing consumers that do not expect this field MUST NOT break.
No existing fields are modified or removed.

### 7.6 Acceptance criteria

- AC-E1: `status` with no `id` returns a response containing the `orientation` field.
- AC-E2: The `orientation.message` string references the getting-started skill by its
  file path.
- AC-E3: The `orientation.skills_path` value is `.agents/skills/`.
- AC-E4: `next` with no `id` and an empty queue returns the `orientation` field.
- AC-E5: `next` with no `id` and a non-empty queue does NOT return the `orientation`
  field.
- AC-E6: `next` with an `id` (claim mode) does NOT return the `orientation` field.
- AC-E7: Existing fields in `status` and `next` responses are unchanged.

---

## 8. Integration Acceptance Criteria

These criteria verify the features work together as a coherent system.

- AC-INT-1: After `kbz init` on a fresh project, an agent that reads `AGENTS.md`
  can determine (a) that kanbanzai MCP tools should be used, (b) the stage gate
  order, and (c) where to find detailed procedures.
- AC-INT-2: After `kbz init` on a fresh project, an agent using GitHub Copilot
  that reads `.github/copilot-instructions.md` is directed to `AGENTS.md` and
  subsequently to the skills.
- AC-INT-3: An agent that calls `status` without reading any files receives the
  orientation breadcrumb pointing to the skills.
- AC-INT-4: The specification skill appears in both the generated AGENTS.md skills
  table and in the workflow skill's stage gates progression (as a reference).
- AC-INT-5: `kbz init --skip-agents-md --skip-skills` produces a project with
  MCP tools available but no AGENTS.md, no copilot instructions, and no skills.
  The orientation breadcrumb in `status` remains the sole discovery path.

---

## 9. Documentation Updates

### 9.1 Getting-started guide

`docs/getting-started.md` MUST be updated to:
- Mention that `kbz init` creates `AGENTS.md` and `.github/copilot-instructions.md`
- Add these files to the directory structure diagram in the "Initialising a project"
  section
- Note the `--skip-agents-md` flag

### 9.2 Init command specification

`work/spec/init-command.md` §3.3 is superseded by this document regarding AGENTS.md.
A note SHOULD be added to that section pointing to this specification. The existing
init spec is not otherwise modified.

### 9.3 MCP tool reference

`docs/mcp-tool-reference.md` SHOULD be updated to document the `orientation` field
in `status` and `next` responses.

---

## 10. Out of Scope

- Generating `.cursorrules`, `.windsurfrules`, or other editor-specific instruction files.
- Changing the `.agents/skills/` install location.
- MCP server reliability improvements (editor-level concern, not a kanbanzai concern).
- Multi-project MCP server support.
- Generating project-specific conventions in AGENTS.md (language, framework, etc.).
- Modifying the `kanbanzai-planning` or `kanbanzai-agents` skills.

---

## 11. Verification Approach

### 11.1 Unit tests

Each feature MUST have unit tests covering the acceptance criteria:

- **Feature A:** Test the AGENTS.md writer with new project, existing managed file
  (current version, older version), existing non-managed file, and `--skip-agents-md`.
- **Feature B:** Test the copilot instructions writer with the same matrix as Feature A,
  plus `.github/` directory creation when absent.
- **Feature C:** Verify that the updated skill source files contain the required text.
  The doc-currency health checker must pass.
- **Feature D:** Verify that the specification skill is embedded, installed, and contains
  all required sections. The doc-currency health checker must pass.
- **Feature E:** Test `status` project overview includes `orientation`. Test `next`
  with empty queue includes `orientation`, non-empty queue excludes it, and claim
  mode excludes it.

### 11.2 Integration test

An end-to-end test MUST run `kbz init` on a temporary directory and verify that:
- `AGENTS.md` exists and contains the managed marker
- `.github/copilot-instructions.md` exists and references AGENTS.md
- `.agents/skills/kanbanzai-specification/SKILL.md` exists
- The getting-started and workflow skills contain the new MCP-tools rule
- A subsequent `kbz init` run does not modify any of these files (idempotency)