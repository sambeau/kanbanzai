# Specification: Skill System (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-88PDBW85 (skill-system)
**Design reference:** `work/design/skills-system-redesign-v2.md` §3.2, §5
**Status:** Draft

---

## Overview

The skill system defines the directory structure, SKILL.md file format, frontmatter schema, body section ordering, reference and script conventions, and loading mechanism for agent skills. A skill encodes the procedure, output format, and evaluation criteria for a specific type of work. Skills are stored as directories under `.kbz/skills/{skill-name}/` with a mandatory `SKILL.md` file and optional `references/` and `scripts/` subdirectories. The loader parses frontmatter and validates body section presence and ordering against the attention-curve-optimized sequence defined by the design.

---

## Scope

### In scope

- Directory layout for skill packages (`.kbz/skills/{skill-name}/`)
- SKILL.md frontmatter YAML schema
- SKILL.md body section names and required ordering
- Dual-register description format (`expert` and `natural` subfields)
- Constraint level enum and its effect on checklist requirements
- Reference file constraints (depth, linking)
- Script file conventions (output-only context injection)
- Size limit for SKILL.md
- Evaluation criteria format (gradable questions with weights)
- Loader that parses frontmatter and validates body sections
- Validation rules for all schema fields

### Explicitly excluded

- The content of specific skills (the skill catalog is a content concern, not a schema concern)
- Context assembly (how skills are combined with roles and injected into agent prompts)
- Script execution runtime and sandboxing
- Skill selection and trigger matching algorithms
- Binding registry integration (specified separately in the binding registry spec)
- Role system schema (specified separately in the role system spec)
- Evaluation pass execution (the criteria format is specified here; the evaluation mechanism is an observability concern)

---

## Functional Requirements

**FR-001:** A skill MUST be stored as a directory at `.kbz/skills/{skill-name}/` containing a mandatory `SKILL.md` file. The `{skill-name}` directory name MUST be lowercase alphanumeric with hyphens permitted (not at start or end), between 2 and 40 characters. The `name` field in the SKILL.md frontmatter MUST match the directory name.

**Acceptance criteria:**
- A skill directory `review-code/` containing a valid `SKILL.md` whose frontmatter has `name: review-code` loads successfully
- A skill directory where the frontmatter `name` does not match the directory name returns a validation error identifying the mismatch
- A directory name with uppercase characters, leading hyphens, or length outside 2–40 returns a validation error
- A skill directory missing `SKILL.md` returns a validation error stating the file is required

---

**FR-002:** The SKILL.md file MUST begin with a YAML frontmatter block delimited by `---` lines. The frontmatter MUST contain the following fields: `name` (string, required), `description` (object with `expert` and `natural` subfields, required), `triggers` (list of strings, required), `roles` (list of strings, required), `stage` (string, required), `constraint_level` (enum string, required). No additional top-level frontmatter fields are permitted.

**Acceptance criteria:**
- A SKILL.md with all required frontmatter fields and valid types loads successfully
- A SKILL.md missing `name` returns a validation error naming the missing field
- A SKILL.md missing `description` returns a validation error naming the missing field
- A SKILL.md missing `triggers` returns a validation error naming the missing field
- A SKILL.md missing `roles` returns a validation error naming the missing field
- A SKILL.md missing `stage` returns a validation error naming the missing field
- A SKILL.md missing `constraint_level` returns a validation error naming the missing field
- A SKILL.md with an unrecognised frontmatter field returns a validation error naming the unknown field

---

**FR-003:** The `description` field MUST be an object containing exactly two subfields: `expert` (string, required, non-empty) and `natural` (string, required, non-empty). The `expert` description activates deep domain knowledge on direct invocation. The `natural` description provides casual trigger matching for ambiguous requests.

**Acceptance criteria:**
- A `description` with both `expert` and `natural` as non-empty strings loads successfully
- A `description` missing the `expert` subfield returns a validation error
- A `description` missing the `natural` subfield returns a validation error
- A `description` with an empty string for `expert` returns a validation error
- A `description` with an empty string for `natural` returns a validation error
- A `description` provided as a plain string instead of an object with subfields returns a validation error

---

**FR-004:** The `triggers` field MUST be a non-empty list of strings. Each trigger is a natural language phrase describing when this skill should be activated.

**Acceptance criteria:**
- A `triggers` list with one or more non-empty string entries loads successfully
- An empty `triggers` list (`triggers: []`) returns a validation error stating triggers must be non-empty
- A `triggers` entry that is not a string returns a validation error

---

**FR-005:** The `roles` field MUST be a non-empty list of strings. Each string MUST be a role ID (matching the role ID format: lowercase alphanumeric with hyphens, 2–30 characters). The `roles` field declares which roles are compatible with this skill.

**Acceptance criteria:**
- A `roles` list with valid role ID strings loads successfully
- An empty `roles` list (`roles: []`) returns a validation error stating roles must be non-empty
- A `roles` entry that does not match the role ID format returns a validation error identifying the invalid entry

---

**FR-006:** The `stage` field MUST be a non-empty string representing a workflow stage name. The value MUST correspond to a known feature lifecycle stage (e.g., `designing`, `specifying`, `dev-planning`, `developing`, `reviewing`) or a recognized non-lifecycle stage (e.g., `researching`, `documenting`, `plan-reviewing`).

**Acceptance criteria:**
- `stage: reviewing` loads successfully
- `stage: ""` (empty) returns a validation error
- `stage: nonexistent-stage` returns a validation error identifying the unrecognised stage name

---

**FR-007:** The `constraint_level` field MUST be one of three enum values: `low`, `medium`, or `high`. This field controls the freedom level of the skill's procedure and affects checklist requirements (see FR-012).

**Acceptance criteria:**
- `constraint_level: low` loads successfully
- `constraint_level: medium` loads successfully
- `constraint_level: high` loads successfully
- `constraint_level: extreme` returns a validation error listing the valid enum values

---

**FR-008:** The SKILL.md body (content after the frontmatter block) MUST contain Markdown sections using `##` level-2 headings. The following sections are recognized: `Vocabulary`, `Anti-Patterns`, `Checklist`, `Procedure`, `Output Format`, `Examples`, `Evaluation Criteria`, `Questions This Skill Answers`. When present, these sections MUST appear in this exact order (the attention-curve-optimized sequence). Unrecognised `##` headings MUST cause a validation warning (not an error) to allow forward-compatible extension.

**Acceptance criteria:**
- A SKILL.md with sections in the correct order loads successfully
- A SKILL.md where `Procedure` appears before `Anti-Patterns` returns a validation error identifying the out-of-order sections
- A SKILL.md where `Examples` appears before `Output Format` returns a validation error identifying the out-of-order sections
- A SKILL.md with an unrecognised `##` heading (e.g., `## Notes`) loads successfully but produces a validation warning
- Sub-headings (`###`, `####`) within a recognized section do not affect ordering validation

---

**FR-009:** The following body sections are required in every SKILL.md: `Vocabulary`, `Anti-Patterns`, `Procedure`, `Output Format`, `Evaluation Criteria`, `Questions This Skill Answers`. The `Checklist`, `Examples` sections are optional.

**Acceptance criteria:**
- A SKILL.md containing all six required sections (and no optional sections) loads successfully
- A SKILL.md missing `Vocabulary` returns a validation error naming the missing section
- A SKILL.md missing `Procedure` returns a validation error naming the missing section
- A SKILL.md missing `Evaluation Criteria` returns a validation error naming the missing section
- A SKILL.md missing `Questions This Skill Answers` returns a validation error naming the missing section
- A SKILL.md containing all required sections plus `Checklist` and `Examples` loads successfully

---

**FR-010:** The `Vocabulary` section MUST contain at least one term. The section body MUST NOT be empty.

**Acceptance criteria:**
- A `Vocabulary` section with a Markdown list of one or more terms passes validation
- A `Vocabulary` section with an empty body (only the heading, no content) returns a validation error

---

**FR-011:** Each entry in the `Anti-Patterns` section MUST contain a `Detect` field and a `BECAUSE` field (case-insensitive matching on the field labels). The `BECAUSE` clause explains why the anti-pattern is harmful, enabling the agent to generalise to adjacent cases.

**Acceptance criteria:**
- An anti-pattern subsection containing both "Detect" and "BECAUSE" labels passes validation
- An anti-pattern subsection missing "Detect" returns a validation warning identifying the incomplete entry
- An anti-pattern subsection missing "BECAUSE" returns a validation warning identifying the incomplete entry

---

**FR-012:** When `constraint_level` is `low` or `medium`, the `Checklist` section MUST be present in the SKILL.md body. When `constraint_level` is `high`, the `Checklist` section is optional.

**Acceptance criteria:**
- A skill with `constraint_level: low` and a `Checklist` section loads successfully
- A skill with `constraint_level: low` without a `Checklist` section returns a validation error stating checklist is required for the constraint level
- A skill with `constraint_level: medium` without a `Checklist` section returns a validation error
- A skill with `constraint_level: high` without a `Checklist` section loads successfully (no error)
- A skill with `constraint_level: high` with a `Checklist` section loads successfully (optional presence is allowed)

---

**FR-013:** The `Evaluation Criteria` section MUST contain criteria phrased as gradable questions. Each criterion MUST have an associated weight value. Valid weight values are: `required`, `high`, `medium`, `low`. This format supports LLM-as-judge automated evaluation passes.

**Acceptance criteria:**
- An `Evaluation Criteria` section with numbered questions each followed by a `Weight:` label and a valid value passes validation
- An `Evaluation Criteria` section with a criterion missing a weight value returns a validation warning
- An `Evaluation Criteria` section with zero criteria (empty body) returns a validation error

---

**FR-014:** The SKILL.md file MUST NOT exceed 500 lines (including frontmatter). Content exceeding this limit MUST be placed in the `references/` subdirectory.

**Acceptance criteria:**
- A SKILL.md file with 500 lines loads successfully
- A SKILL.md file with 501 lines returns a validation error stating the line limit and the actual line count
- A SKILL.md file with 499 lines loads successfully

---

**FR-015:** The `references/` subdirectory within a skill directory is optional. When present, it MUST contain only Markdown files. Reference files MUST be linked directly from SKILL.md (one level deep). Reference files MUST NOT reference other reference files (no transitive reference chains).

**Acceptance criteria:**
- A `references/` directory containing `.md` files loads successfully
- A `references/` directory containing a non-Markdown file (e.g., `.txt`, `.go`) returns a validation warning
- The loader records which reference files are linked from SKILL.md; a reference file not linked from SKILL.md produces a validation warning identifying the orphaned file

---

**FR-016:** The `scripts/` subdirectory within a skill directory is optional. When present, it contains executable scripts. Only script output is included in the agent's context window; script source code MUST NOT be injected into context.

**Acceptance criteria:**
- A `scripts/` directory with executable files loads successfully
- The loader's output representation for a skill distinguishes script paths (for execution) from script content (which is not loaded into context)

---

**FR-017:** The loader MUST parse a SKILL.md file into a structured representation containing: the parsed frontmatter fields, an ordered list of body sections (each with heading name and content), the list of reference file paths, and the list of script file paths. The loader MUST report all validation errors found in a single pass rather than stopping at the first error.

**Acceptance criteria:**
- Loading a valid skill returns a structured object with frontmatter fields, body sections in order, reference paths, and script paths
- Loading a skill with multiple validation errors (e.g., missing `Vocabulary` section and exceeding 500 lines) returns all errors in a single response
- The error response includes the skill name and file path for identification

---

## Non-Functional Requirements

**NFR-001:** Loading and validating a single skill (including frontmatter parsing, body section extraction, and reference/script discovery) MUST complete in under 100ms on a standard development machine, excluding filesystem I/O latency.

**Acceptance criteria:**
- Benchmark tests for skill loading with a representative SKILL.md (400 lines, 3 references, 2 scripts) complete within the time bound

---

**NFR-002:** The SKILL.md format MUST be human-readable and human-authorable without special tooling. Authors write standard Markdown with a YAML frontmatter block. The format MUST NOT require custom Markdown extensions, templating languages, or preprocessing steps.

**Acceptance criteria:**
- All example SKILL.md files in the design document are valid under the specified schema
- A SKILL.md file renders correctly in any standard Markdown viewer (GitHub, VS Code, etc.)

---

**NFR-003:** The frontmatter schema MUST be forward-compatible. Unknown frontmatter fields MUST be rejected (strict parsing) to prevent silent schema drift.

**Acceptance criteria:**
- A SKILL.md with a frontmatter field not in the defined schema returns a validation error naming the unknown field

---

**NFR-004:** The skill loader MUST handle a `.kbz/skills/` directory containing at least 30 skill directories without degradation. Listing all skills MUST complete in under 500ms.

**Acceptance criteria:**
- Benchmark tests with 30 skill directories complete the full listing within the time bound

---

## Dependencies and Assumptions

1. **Role system spec:** The `roles` field in SKILL.md frontmatter references role IDs defined by the role system (FEAT-01KN5-88PCVN4Y). The role ID format (lowercase alphanumeric with hyphens, 2–30 characters) is defined there. Cross-validation of role references against actually defined roles is a binding registry concern, not a skill loader concern — the skill loader validates format only.
2. **Feature lifecycle stages:** The `stage` field references workflow stages. Valid stage names are derived from the feature lifecycle statuses in `internal/model/entities.go` (e.g., `designing`, `specifying`, `dev-planning`, `developing`, `reviewing`) plus non-lifecycle stages (`researching`, `documenting`, `plan-reviewing`) that are defined in the binding registry.
3. **YAML parsing:** The project uses `gopkg.in/yaml.v3` for frontmatter parsing. Strict parsing requires `KnownFields(true)` on the decoder.
4. **Markdown parsing:** Body section extraction requires parsing `##` headings from Markdown. This does not require a full Markdown AST parser — line-by-line scanning for lines starting with `## ` is sufficient.
5. **Filesystem conventions:** Script executability is determined by the operating system's file permission model. The loader records script paths but does not execute scripts; execution is a runtime concern.
6. **500-line limit:** Line counting includes frontmatter delimiters and blank lines. The count is of newline-delimited lines in the raw file content.