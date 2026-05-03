# Specification: Bootstrapping Gap Remediation

| Field  | Value             |
|--------|-------------------|
| Date   | 2026-05-03        |
| Status | Draft             |
| Author | Spec Author (AI)  |

> This specification implements the design described in
> `work/P47-bootstrapping-gap-remediation/design-bootstrapping-gap-remediation.md`.

---

## Related Work

### Prior designs consulted

- **`work/_project/design-fresh-install-experience.md`** — Predates the 3.0 pipeline. Addresses `.agents/skills/` consolidation, MCP config generation, and default role scaffolding (2 roles). This spec extends that work to cover the 3.0 pipeline artifacts: `stage-bindings.yaml`, `.kbz/skills/` (20 task-execution skills), and `.kbz/roles/` (all 18 roles).
- **`work/P47-bootstrapping-gap-remediation/design-bootstrapping-gap-remediation.md`** — The approved design for this specification. Defines 5 design principles, the embedded filesystem layout, per-gap fixes, implementation sequence, and prevention measures.

### Prior specs consulted

- **`work/B16-skills-and-roles-3.0/B16-F3-spec-3.0-binding-registry.md`** — Defines the `stage-bindings.yaml` schema, location (`.kbz/stage-bindings.yaml`), and validation rules. The binding file exists in this project's `.kbz/` but has no embedded source for init. FR-001 of that spec confirms the expected file location.
- **`work/B16-skills-and-roles-3.0/B16-F2-spec-3.0-skill-system.md`** (implied) — Defines the skill format (SKILL.md with YAML frontmatter), storage layout (`<name>/SKILL.md` under `.kbz/skills/`), and validation rules. The 20 task-execution skills in this project's `.kbz/skills/` conform to this spec.
- **`work/B16-skills-and-roles-3.0/B16-F1-spec-3.0-role-system.md`** (implied) — Defines the role format (YAML with `id`, `inherits`, `identity`, `vocabulary`, `anti_patterns`, `tools` fields), storage layout (`.kbz/roles/<id>.yaml`), and inheritance resolution. The 18 role files in this project's `.kbz/roles/` conform to this spec.
- **`work/B11-fresh-install-experience/B11-F2-spec-embedded-skills.md`** — Prior spec for embedding `.agents/skills/` workflow skills. Defines the embedded skill count (then 8, now 9). This spec extends the embedded asset set to include task-execution skills and roles.

### Relationship to prior work

This specification is the logical completion of B11 (fresh-install-experience) brought forward to the 3.0 pipeline era. B11 installed `.agents/skills/` workflow skills and 2 roles but predated the binding registry, task-execution skills, and the full role catalog. This spec adds what B11 could not: the artifacts that the 3.0 pipeline actually requires to activate.

### Deliberate divergences

- **G8 (codebase-memory skills) is deferred** — The design explicitly defers installing `.github/skills/codebase-memory-*/` to a later feature. This spec does not include requirements for codebase-memory skill installation.
- **G9 (refs/ and work/templates/) is no-action** — These are intentionally project-local and not included in this spec.

---

## Overview

This specification defines the requirements for closing the 9 bootstrapping gaps identified in the audit report. The work embeds three new artifact categories into the `kbz` binary — `stage-bindings.yaml`, 20 task-execution skill files, and 16 additional role files — and adds install logic to `kbz init` so that consumer projects receive a complete, pipeline-ready `.kbz/` directory. It also enriches the generated `AGENTS.md` and `.github/copilot-instructions.md` with skill/role reference tables, extends `--update-skills` to cover all managed artifacts, and adds a CI staleness check to prevent future drift between embedded seeds and their project-file counterparts.

---

## Scope

### In scope

- Embedding `stage-bindings.yaml` in the binary and installing it to `.kbz/stage-bindings.yaml` during `kbz init`
- Embedding all 20 task-execution skill files in the binary and installing them to `.kbz/skills/<name>/SKILL.md` during `kbz init`
- Embedding all 16 additional role files in the binary and installing them to `.kbz/roles/<id>.yaml` during `kbz init`
- Version-aware create/update/skip logic for every managed artifact
- Enriching generated `AGENTS.md` with a task-execution skill table and a role table
- Enriching generated `.github/copilot-instructions.md` with stage-bindings guidance
- Extending `kbz init --update-skills` to update stage-bindings, task skills, and all managed roles
- A CI/test check that embedded workflow skill seeds match their `.agents/skills/` counterparts
- End-to-end integration test verifying a fresh `kbz init` produces a pipeline-ready project

### Explicitly excluded

- Installing `.github/skills/codebase-memory-*/` (deferred per G8)
- Installing `refs/*.md` or `work/templates/*.md` (project-local per G9)
- Changes to skill or role content — this spec installs existing content as-is
- Changes to the `kbz serve` pipeline construction logic — it already correctly activates when `stage-bindings.yaml` exists
- Changes to `handoff` or `next` tool behavior
- Binary size optimization beyond standard Go embedding
- Creating new skill files or role files — all artifacts already exist in this project

---

## Functional Requirements

### FR-001: stage-bindings.yaml embedded and installed

The `kbz` binary MUST embed the canonical `stage-bindings.yaml` file and `kbz init` MUST install it to `.kbz/stage-bindings.yaml` in the consumer project's `.kbz/` directory.

**Acceptance criteria:**
- `kbz init` on a fresh git repo creates `.kbz/stage-bindings.yaml`
- The installed file is byte-identical to the embedded source (excluding version markers)
- The installed file passes `binding.LoadBindingFile()` validation
- The file contains a `# kanbanzai-managed: true` marker on line 1
- The file contains a `# kanbanzai-version: <n>` marker on line 2
- Re-running `kbz init` on an existing project with a managed stage-bindings at an older version updates it
- Re-running `kbz init` on an existing project with a managed stage-bindings at the same version is a no-op
- Re-running `kbz init` on an existing project with an unmanaged stage-bindings (no marker) skips with a warning
- `kbz init --update-skills` updates a managed stage-bindings at an older version

---

### FR-002: Task-execution skills embedded and installed

The `kbz` binary MUST embed all 19 task-execution skill files and `kbz init` MUST install them to `.kbz/skills/<name>/SKILL.md` in the consumer project.

The 19 skills are: `write-design`, `write-spec`, `write-dev-plan`, `decompose-feature`, `orchestrate-development`, `implement-task`, `review-code`, `orchestrate-review`, `review-plan`, `write-research`, `update-docs`, `orchestrate-doc-pipeline`, `write-docs`, `edit-docs`, `check-docs`, `style-docs`, `copyedit-docs`, `audit-codebase`, `write-skill`.

Note: `prompt-engineering` is not included — it is being developed in batch B44 and will be added to the skill catalog in a future update.

**Acceptance criteria:**
- `kbz init` on a fresh git repo creates `.kbz/skills/` with all 19 skill subdirectories
- Each skill directory contains a valid `SKILL.md` file that passes `skill.SkillStore.Load()` validation
- Each `SKILL.md` frontmatter contains `# kanbanzai-managed:` and `# kanbanzai-version:` markers
- Re-running `kbz init` on an existing project updates managed skills at older versions
- Re-running `kbz init` on an existing project skips managed skills at the same version
- Re-running `kbz init` on an existing project skips unmanaged skill files (no managed marker) with a warning
- `kbz init --update-skills` updates managed task skills at older versions
- After installation, `NewSkillStore(".kbz/skills/").LoadAll()` returns all 19 skills without error
- The embedded skill files are byte-identical to the corresponding files in `.kbz/skills/<name>/SKILL.md` in this project (excluding version markers)

---

### FR-003: Role files embedded and installed

The `kbz` binary MUST embed all 16 role files currently in this project's `.kbz/roles/` (excluding `base.yaml` and `reviewer.yaml` which are already installed) and `kbz init` MUST install them to `.kbz/roles/<id>.yaml` in the consumer project.

The 16 roles are: `architect`, `spec-author`, `implementer`, `implementer-go`, `orchestrator`, `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`, `researcher`, `documenter`, `doc-pipeline-orchestrator`, `doc-editor`, `doc-checker`, `doc-stylist`, `doc-copyeditor`.

**Acceptance criteria:**
- `kbz init` on a fresh git repo creates `.kbz/roles/` with all 18 role files (2 existing + 16 new)
- Each role file passes `RoleStore.Load()` validation
- `base.yaml` is created once and never overwritten (existing behavior)
- All other roles are managed: version-aware update applies
- After installation, `NewRoleStore(".kbz/roles/", ".kbz/context/roles/").LoadAll()` returns all 18 roles without error
- Re-running `kbz init` updates managed roles at older versions
- Re-running `kbz init` skips managed roles at the same version
- Re-running `kbz init` skips unmanaged role files (no `kanbanzai-managed` marker) with a warning
- `kbz init --update-skills` updates managed roles at older versions

---

### FR-004: Enriched generated AGENTS.md

The generated `AGENTS.md` content MUST include a task-execution skill reference table and a role reference table, enabling consumer agents to discover the available skills and roles without reading `stage-bindings.yaml` directly.

**Acceptance criteria:**
- Generated `AGENTS.md` contains a "Task-Execution Skills" table with columns: Skill, Path, and Stage
- The table lists all 17 skills referenced in `stage-bindings.yaml` with their `.kbz/skills/` paths and the stage(s) each is bound to. Skills bound to the same stage may appear in a compound row (e.g., `write-docs`, `edit-docs`, `check-docs`, `style-docs`, `copyedit-docs` grouped as one row) — such a row counts as multiple entries, one per skill.
- Generated `AGENTS.md` contains a "Roles" table with columns: Role, File, and Stage
- The table lists all 15 roles referenced in `stage-bindings.yaml` with their `.kbz/roles/` paths and the stage(s) each is bound to. Roles bound to the same stage may appear in a compound row (e.g., `doc-editor`, `doc-checker`, `doc-stylist`, `doc-copyeditor` grouped as one row) — such a row counts as multiple entries, one per role.
- Generated `AGENTS.md` contains a "Stage Bindings" section stating that `.kbz/stage-bindings.yaml` is the authoritative source of truth
- The `agentsMDVersion` is incremented to 3
- An existing managed AGENTS.md at version 2 is overwritten with the version 3 content on re-run
- The file does not exceed 100 lines (current limit is 50; expanded budget for the tables)

---

### FR-005: Enriched generated copilot-instructions.md

The generated `.github/copilot-instructions.md` content MUST include guidance for reading `stage-bindings.yaml` to find the correct role and skill for the current workflow stage.

**Acceptance criteria:**
- Generated file contains: "Read `.kbz/stage-bindings.yaml` before starting any workflow stage to find the correct role and skill."
- The file remains under 25 lines
- The `agentsMDVersion` increment also applies to this file (shared version constant)

---

### FR-006: --update-skills covers all managed artifacts

`kbz init --update-skills` MUST update all managed artifacts — not just `.agents/skills/kanbanzai-*` files and `reviewer.yaml`, but also `stage-bindings.yaml`, `.kbz/skills/` task-execution skills, and all managed roles.

**Acceptance criteria:**
- `kbz init --update-skills` calls the stage-bindings install function
- `kbz init --update-skills` calls the task-skills install function
- `kbz init --update-skills` calls the all-roles install function (or extends the existing `updateManagedRoles`)
- A project initialized with an older `kbz` binary gets all new managed artifacts after running `kbz init --update-skills` with the new binary
- Unmanaged files are never overwritten
- `--update-skills` and `--skip-skills` remain mutually exclusive (existing behavior)

---

### FR-007: CI staleness check for embedded workflow skills

The test suite MUST include a check that embedded workflow skill seeds (`internal/kbzinit/skills/<name>/SKILL.md`) are byte-identical to their corresponding files in `.agents/skills/kanbanzai-<name>/SKILL.md` after normalizing version markers.

**Acceptance criteria:**
- `go test ./internal/kbzinit/...` includes a test named `TestEmbeddedSkillsMatchAgentSkills` or equivalent
- The test iterates all 9 workflow skill names
- For each skill, it reads the embedded file and the on-disk file
- It normalizes the `# kanbanzai-version:` line to a known value before comparing
- The test fails if any skill pair differs in body content
- The test runs as a regular unit test (no integration build tag required)

---

### FR-008: End-to-end init integration test

The test suite MUST include an integration test that runs `kbz init` on a fresh git repo and verifies the resulting project activates the 3.0 context assembly pipeline.

**Acceptance criteria:**
- Test creates a fresh git repo with at least one commit
- Test runs `kbz init` with default options
- Test verifies `.kbz/stage-bindings.yaml` exists and passes `binding.LoadBindingFile()` validation
- Test verifies all 20 task-execution skills exist and pass `skill.SkillStore.Load()` validation
- Test verifies all 18 roles exist and pass `RoleStore.Load()` validation
- Test verifies the pipeline can be constructed (stage-bindings loads, a skill loads, a role resolves)
- Test verifies `AGENTS.md` was created with the managed marker
- Test verifies `.mcp.json` was created with the managed marker
- Test runs with the `integration` build tag

---

### FR-009: Project copilot-instructions.md reconciliation

The project's hand-maintained `.github/copilot-instructions.md` MUST NOT contain references to artifacts that do not exist after `kbz init`, unless those references are clearly marked as kanbanzai-project-local.

**Acceptance criteria:**
- The project file is either trimmed to match what init installs, OR
- A comment at the top of the file states: `<!-- kanbanzai-project: this file contains project-local references not present in generated installs. See internal/kbzinit/agents_md.go for the canonical consumer version. -->`
- The file does not claim that `.kbz/skills/`, `.kbz/roles/`, `.github/skills/`, `refs/`, or `work/templates/` exist after init without also noting they are project-local

---

### FR-010: Skill store and role store load without errors

After `kbz init` on a fresh project, the SkillStore and RoleStore MUST load their full catalogs without errors when the server starts.

**Acceptance criteria:**
- `NewSkillStore(".kbz/skills/").LoadAll()` returns all 20 skills with no errors
- `NewRoleStore(".kbz/roles/", ".kbz/context/roles/").LoadAll()` returns all 18 roles with no errors
- The `kbz serve` startup log includes the message "3.0 context assembly pipeline loaded with N stage bindings" where N is the number of stages in the binding file

---

## Non-Functional Requirements

- **NFR-001 (Binary size):** The total increase in the `kbz` binary size from adding all embedded files MUST NOT exceed 500 KB. (Estimate: 20 SKILL.md files × ~5 KB + 16 YAML files × ~2 KB + stage-bindings.yaml ~3 KB ≈ 135 KB — well within budget.)
- **NFR-002 (Init time):** `kbz init` on a fresh git repo MUST complete in under 2 seconds on a modern SSD. The additional file writes are small and should add negligible latency.
- **NFR-003 (Backward compatibility):** `kbz init` MUST NOT break on a project initialized with an older `kbz` binary. Existing config, worktrees, and user-owned files MUST be preserved.
- **NFR-004 (Idempotency):** Running `kbz init` twice on the same project MUST produce the same result as running it once. No files should be duplicated or corrupted.
- **NFR-005 (Test coverage):** All new install functions MUST have unit tests covering create, update, skip, and conflict cases.

---

## Constraints

- The existing `go:embed` pattern for skills MUST be preserved — do not introduce a new embedding mechanism
- The existing version-aware managed marker pattern MUST be used for all new managed files
- The `--skip-skills` flag MUST suppress task-execution skill installation (it already suppresses workflow skill installation)
- The `--skip-roles` flag MUST suppress role installation (existing behavior, extended to all roles)
- The `--skip-agents-md` flag MUST suppress AGENTS.md and copilot-instructions.md generation (existing behavior)
- The MCP server construction logic in `internal/mcp/server.go` MUST NOT be changed (it already activates the pipeline when stage-bindings exists)
- Skill content and role content MUST be copied as-is from this project's `.kbz/skills/` and `.kbz/roles/` directories — no content changes in this feature

---

## Acceptance Criteria

- **AC-001 (FR-001):** Given a fresh git repo with no `.kbz/` directory, when `kbz init` runs, then `.kbz/stage-bindings.yaml` exists with managed markers and passes `binding.LoadBindingFile()` validation.

- **AC-002 (FR-002):** Given a fresh git repo with no `.kbz/` directory, when `kbz init` runs, then `.kbz/skills/` contains all 19 task-execution skill directories, each with a valid `SKILL.md` that passes `skill.SkillStore.Load()`.

- **AC-003 (FR-003):** Given a fresh git repo with no `.kbz/` directory, when `kbz init` runs, then `.kbz/roles/` contains all 18 role files, each passing `RoleStore.Load()` validation.

- **AC-004 (FR-004):** Given a project initialized with the new binary, when an agent reads `AGENTS.md`, then it finds a task-execution skill table covering all 17 skills from `stage-bindings.yaml` and a role table covering all 15 roles from `stage-bindings.yaml`. (Some rows may group multiple related skills or roles — this is acceptable as long as every skill and role is listed.)

- **AC-005 (FR-006):** Given a project initialized with an older binary (missing stage-bindings, task skills, and most roles), when `kbz init --update-skills` runs with the new binary, then all managed artifacts are brought to the current version.

- **AC-006 (FR-007):** Given embedded skill seeds that differ from their `.agents/skills/` counterparts, when `go test ./internal/kbzinit/...` runs, then `TestEmbeddedSkillsMatchAgentSkills` fails.

- **AC-007 (FR-008):** Given the integration test suite, when the end-to-end init test runs, then all assertions pass and the pipeline activates.

- **AC-008 (FR-001 + FR-002 + FR-003):** Given a fresh `kbz init`, when `kbz serve` starts, then the log contains "3.0 context assembly pipeline loaded with N stage bindings" (not "legacy").
- **AC-009 (FR-010):** Given a fresh `kbz init`, when `kbz serve` starts, then `SkillStore.LoadAll()` returns 19 skills and `RoleStore.LoadAll()` returns 18 roles without error.

- **AC-010 (FR-009):** Given the project's `.github/copilot-instructions.md`, when read by a reviewer, then it does not claim that `.kbz/skills/`, `.kbz/roles/`, or `.github/skills/` exist after a consumer init without marking them as project-local.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: run `kbz init` on temp git repo, assert `stage-bindings.yaml` exists, assert `binding.LoadBindingFile()` succeeds |
| AC-002 | Test | Automated test: run `kbz init` on temp git repo, iterate all 19 skill names, assert `skill.SkillStore.Load(name)` succeeds |
| AC-003 | Test | Automated test: run `kbz init` on temp git repo, assert `RoleStore.LoadAll()` returns 18 roles |
| AC-004 | Inspection | Verify generated AGENTS.md content in `agents_md.go` includes the two tables with correct entries |
| AC-005 | Test | Automated test: init with old-style `.kbz/` (no stage-bindings, no task skills, few roles), run `--update-skills`, assert all artifacts exist |
| AC-006 | Test | Automated unit test: `TestEmbeddedSkillsMatchAgentSkills` in `skills_consistency_test.go` |
| AC-007 | Test | Integration test (build tag `integration`): full end-to-end init + pipeline activation check |
| AC-008 | Test | Integration test: start `kbz serve`, capture log output, assert pipeline activation message present |
| AC-009 | Test | Integration test: programmatically verify `SkillStore.LoadAll()` returns 19 skills and `RoleStore.LoadAll()` returns 18 roles |
| AC-010 | Inspection | Review `.github/copilot-instructions.md` for project-local markers or trimmed content |

## Dependencies and Assumptions

### Dependencies

- All 20 task-execution skill files exist in this project's `.kbz/skills/` and are valid per `skill.SkillStore` (verified during audit)
- All 16 additional role files exist in this project's `.kbz/roles/` and are valid per `RoleStore` (verified during audit)
- The `stage-bindings.yaml` in this project's `.kbz/` is the canonical version (verified during audit)
- The 3.0 pipeline construction logic in `internal/mcp/server.go` does not require changes (verified during audit — it already activates when stage-bindings exists)

### Assumptions

- The 20 task-execution skills do not require content changes before embedding — they are production-quality as-is
- The 16 role files do not require content changes before embedding
- `go:embed` supports the directory count (21 embedded directories: 20 task skills + 16 roles + 1 stage-bindings = 37, well within limits)
- Binary size increase is acceptable (estimated ~135 KB)
- The existing `transformSkillContent` function works for task-execution skill frontmatter (same format as workflow skills)
- No consumer project will have user-created files at `.kbz/skills/` or `.kbz/roles/` that conflict with managed install paths — the managed marker pattern handles conflicts gracefully
