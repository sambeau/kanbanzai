# Implementation Plan: Migration and Backward Compatibility

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PGXATW (migration-and-backward-compat)               |
| Spec    | `work/spec/3.0-migration.md`                                       |
| Design  | `work/design/skills-system-redesign-v2.md` §7, §11                |

---

## 1. Overview

This plan decomposes the migration and backward compatibility specification into assignable tasks for AI agents. The specification covers file restructuring (roles relocating from `.kbz/context/roles/` to `.kbz/roles/`, skills from `.skills/` to `.kbz/skills/`), renaming and splitting of existing roles and skills, creation of new files (`stage-bindings.yaml`), Go code changes to the `profile` and `handoff` tools for backward compatibility, and a retirement procedure for old files.

**Scope boundaries (from specification):**

- IN: File relocation, role restructuring (`base.yaml` migration, `developer.yaml` → `implementer-go.yaml`, `reviewer.yaml` → reviewer + 4 specialists), skill splitting (`code-review.md` → 2 skills, `plan-review.md` → 1 restructured skill, `document-creation.md` → 5 type-specific skills), `stage-bindings.yaml` creation, vocabulary and anti-pattern fields in all new files, backward compatibility for `profile` and `handoff` tools, old file retention and retirement procedure.
- OUT: Content of new roles and skills (covered by separate content specs), YAML/SKILL.md format schemas (covered by system specs), context assembly pipeline changes, gate enforcement mechanism, automated migration tooling.

**Key structural decisions:**

1. **Go code changes serialised first.** The `profile` tool extension (Task 1) must land before role files can be tested in their new locations. The `handoff` tool extension (Task 2) depends on Task 1's profile store changes. All file-level tasks (3–5) depend on the tools being able to read from new locations.
2. **File tasks parallelised after code.** Once the tools support both old and new locations, the three file-migration tasks (roles, skills, stage-bindings) are independent and can execute in parallel.
3. **Retirement is a deliberate final step.** Task 6 (retirement verification) runs only after all other tasks complete and all references are updated.

**Critical NFR:** The system must be functional at every intermediate migration state (NFR-001). No task may leave the system in a broken state. This means every task must preserve old paths while creating new ones.

---

## 2. Task Breakdown

### Task 1: Extend Profile Store and Profile Tool for Dual-Location Loading

**Objective:** Modify the `ProfileStore` in `internal/context/profile.go` and the `profile` MCP tool in `internal/mcp/profile_tool.go` to support loading role files from both `.kbz/roles/` (new canonical location) and `.kbz/context/roles/` (old location, fallback). The store must check the new location first and fall back to the old location if the file is not found. The `Profile` struct must gain optional `Vocabulary`, `AntiPatterns`, and `Tools` fields so that new-format role files can be loaded without breaking old-format files that lack these fields. The `ResolvedProfile` must also gain these fields, and `resolve.go` must merge them during inheritance resolution. The `profile(action: "list")` must return roles from the new location after migration, falling back to the old location for unmigrated roles.

**Specification references:** FR-001 (role directory relocation), FR-012 (profile tool extension), FR-014 (unchanged subsystems — inheritance resolution algorithm itself unchanged, but extended for new fields), NFR-001 (incremental migration — system works with any mix of old/new locations), NFR-002 (code changes confined to FR-012 scope)

**Input context:**
- `internal/context/profile.go` — current `ProfileStore`, `Profile` struct, `Load`, `LoadAll`, `validateProfile`
- `internal/context/resolve.go` — current `ResolveChain`, `ResolveProfile` with leaf-level replace semantics
- `internal/mcp/profile_tool.go` — current `profileTool`, `profileListAction`, `profileGetAction`
- `work/spec/3.0-migration.md` FR-012 — backward compatibility acceptance criteria
- `work/design/skills-system-redesign-v2.md` §7 — new directory structure

**Output artifacts:**
- Modified `internal/context/profile.go`:
  - `Profile` struct gains `Vocabulary []string`, `AntiPatterns []AntiPattern`, `Tools []string` fields (all `omitempty`)
  - New `AntiPattern` struct with `Name`, `Detect`, `Because`, `Resolve` string fields
  - `ResolvedProfile` struct gains matching fields
  - `ProfileStore` gains a secondary root path (fallback) or accepts an ordered list of search paths
  - `Load(id)` checks new root first, then fallback root; returns first match
  - `LoadAll()` merges results from both roots, preferring new-root files when an ID exists in both
  - `validateProfile` unchanged for old-format files (new fields are optional)
- Modified `internal/context/resolve.go`:
  - `ResolveProfile` merges `Vocabulary`, `AntiPatterns`, and `Tools` using the same leaf-level replace semantics: leaf's slice replaces parent's slice if non-nil; nil inherits from parent
- Modified `internal/mcp/profile_tool.go`:
  - `profileGetAction` includes `vocabulary`, `anti_patterns`, and `tools` in response when present
  - `profileListAction` returns merged list from both roots
- New test file `internal/context/profile_dual_root_test.go` — tests dual-location loading, fallback behaviour, and new field parsing
- New test file `internal/context/resolve_new_fields_test.go` — tests vocabulary/anti-pattern inheritance merging

**Dependencies:** None — this is the first task.

**Interface contract (consumed by Tasks 2, 3):**

```go
// ProfileStore constructor — accepts primary (new) and fallback (old) roots.
// When fallbackRoot is empty, behaves identically to the current single-root store.
func NewProfileStore(primaryRoot, fallbackRoot string) *ProfileStore

// AntiPattern represents a structured anti-pattern entry in a role file.
type AntiPattern struct {
    Name    string `yaml:"name"`
    Detect  string `yaml:"detect"`
    Because string `yaml:"because"`
    Resolve string `yaml:"resolve"`
}

// Profile gains these optional fields (zero values for old-format files):
type Profile struct {
    // ... existing fields ...
    Vocabulary   []string      `yaml:"vocabulary,omitempty"`
    AntiPatterns []AntiPattern  `yaml:"anti_patterns,omitempty"`
    Tools        []string      `yaml:"tools,omitempty"`
}

// ResolvedProfile gains matching fields:
type ResolvedProfile struct {
    // ... existing fields ...
    Vocabulary   []string
    AntiPatterns []AntiPattern
    Tools        []string
}
```

---

### Task 2: Extend Handoff Tool for New Skill and Role Structures

**Objective:** Modify the `handoff` tool in `internal/mcp/handoff_tool.go` and the context assembly pipeline in `internal/context/assemble.go` to support the new attention-curve skill format and the new role fields (`vocabulary`, `anti_patterns`). When new-format skills are available, `handoff` must assemble context using attention-curve ordering (Vocabulary → Anti-Patterns → Procedure → Output Format). When only old-format skills exist, behaviour is unchanged. The tool must handle mixed states where some roles/skills are migrated and some are not.

**Specification references:** FR-013 (handoff tool extension), FR-014 (unchanged subsystems — MCP tool signatures unchanged), NFR-001 (functional at every intermediate state)

**Input context:**
- `internal/mcp/handoff_tool.go` — current `handoffTool`, `renderHandoffPrompt`, `assembleContext`
- `internal/context/assemble.go` — current `Assemble` function, `AssemblyInput`, `AssemblyResult`, `formatProfile`
- `internal/context/profile.go` — updated `Profile`/`ResolvedProfile` structs (from Task 1)
- `work/spec/3.0-migration.md` FR-013 — backward compatibility acceptance criteria
- `work/design/skills-system-redesign-v2.md` §3.2 — attention-curve ordering definition

**Output artifacts:**
- Modified `internal/context/assemble.go`:
  - `formatProfile` extended to render `Vocabulary` and `AntiPatterns` fields when present on the resolved profile
  - New helper `formatSkillContext(skillPath string) string` that reads a SKILL.md, detects whether it is new-format (has YAML frontmatter with `name` field) or old-format, and renders accordingly
  - When new-format skill is detected: render sections in attention-curve order (Vocabulary first, then Anti-Patterns, then Procedure, then Output Format)
  - When old-format skill is detected: render as plain markdown (current behaviour)
- Modified `internal/mcp/handoff_tool.go`:
  - `renderHandoffPrompt` includes skill context in the assembled prompt when a skill is associated with the task's stage
  - No change to the `handoff` tool's MCP parameter signature (FR-014)
- New test file `internal/context/assemble_skill_test.go` — tests skill detection, attention-curve rendering, old-format passthrough, and mixed-state handling
- New test file `internal/mcp/handoff_mixed_state_test.go` — integration tests for handoff with pre-migration, mid-migration, and post-migration role/skill combinations

**Dependencies:** Task 1 (profile store changes — the updated `Profile`/`ResolvedProfile` structs with vocabulary and anti-pattern fields)

**Interface contract (consumed by Task 3):**

The skill format detection uses this heuristic:
- New-format: file is `SKILL.md` inside a directory under `.kbz/skills/`, contains YAML frontmatter delimited by `---` with a `name:` field
- Old-format: file is a `.md` file under `.skills/`, no YAML frontmatter with `name:` field

No new MCP tool parameters are added. The `handoff` tool signature remains:
```
handoff(task_id: string, role?: string, instructions?: string)
```

---

### Task 3: Migrate Role Files to `.kbz/roles/`

**Objective:** Create the new `.kbz/roles/` directory and populate it with migrated role files. Migrate `base.yaml` with new vocabulary/anti-pattern fields. Create `implementer.yaml` (base implementation role) and `implementer-go.yaml` (inherits from `implementer`, carries Go-specific vocabulary — replaces `developer.yaml`). Create `reviewer.yaml` (base review role) and four specialist subtypes (`reviewer-conformance.yaml`, `reviewer-quality.yaml`, `reviewer-security.yaml`, `reviewer-testing.yaml`). All specialist roles inherit from `reviewer`. Do NOT delete old files in `.kbz/context/roles/` — they must remain for backward compatibility (FR-011, NFR-001).

**Specification references:** FR-001 (role directory relocation), FR-002 (base role restructuring), FR-003 (developer → implementer-go rename), FR-004 (reviewer split into 5 files), FR-009 (vocabulary fields in all roles), FR-010 (anti-pattern sections in all non-base roles), FR-011 (old directory retained), NFR-001 (incremental — system works with partially migrated roles), NFR-003 (directory structure matches design §7.3)

**Input context:**
- `.kbz/context/roles/base.yaml` — current base role file (content to preserve/migrate)
- `.kbz/context/roles/developer.yaml` — current developer role (content to map into implementer-go)
- `.kbz/context/roles/reviewer.yaml` — current reviewer role (content to distribute across reviewer + 4 specialists)
- `work/spec/3.0-migration.md` FR-002, FR-003, FR-004 — restructuring acceptance criteria
- `work/design/skills-system-redesign-v2.md` §7.3 — target directory structure
- `internal/context/profile.go` — updated Profile struct with new fields (from Task 1)

**Output artifacts:**
- New directory `.kbz/roles/` containing:
  - `base.yaml` — migrated from `.kbz/context/roles/base.yaml`, gains `vocabulary` (project-wide terms) and `anti_patterns` (empty list — per FR-010 exception for base)
  - `implementer.yaml` — NEW base implementation role (`id: implementer`, `inherits: base`, vocabulary with language-agnostic implementation terms, anti-patterns for common implementation failures)
  - `implementer-go.yaml` — migrated from `developer.yaml` (`id: implementer-go`, `inherits: implementer`, vocabulary with Go-specific terms, anti-patterns for Go-specific pitfalls)
  - `reviewer.yaml` — migrated from `.kbz/context/roles/reviewer.yaml` (`id: reviewer`, `inherits: base`, vocabulary with general review terms, anti-patterns for general review failures)
  - `reviewer-conformance.yaml` — NEW (`id: reviewer-conformance`, `inherits: reviewer`, conformance-specific vocabulary and anti-patterns)
  - `reviewer-quality.yaml` — NEW (`id: reviewer-quality`, `inherits: reviewer`, quality-specific vocabulary and anti-patterns)
  - `reviewer-security.yaml` — NEW (`id: reviewer-security`, `inherits: reviewer`, security-specific vocabulary and anti-patterns)
  - `reviewer-testing.yaml` — NEW (`id: reviewer-testing`, `inherits: reviewer`, testing-specific vocabulary and anti-patterns)
- Old files in `.kbz/context/roles/` remain UNCHANGED on disk

**Dependencies:** Task 1 (profile store must support new location and new fields before role files can be loaded and tested there)

**Verification:** After this task completes:
- `profile(action: "get", id: "implementer-go")` returns the new role with vocabulary and anti-patterns
- `profile(action: "get", id: "reviewer")` returns the new base review role (from `.kbz/roles/`, not `.kbz/context/roles/`)
- `profile(action: "get", id: "developer")` still returns the old developer role (from fallback location) — backward compatibility
- `profile(action: "list")` returns all roles from both locations, preferring new-location files

---

### Task 4: Restructure Skills into `.kbz/skills/` Directories

**Objective:** Create the new `.kbz/skills/` directory structure and populate it with restructured skill files. Split `code-review.md` into `review-code/SKILL.md` and `orchestrate-review/SKILL.md`. Restructure `plan-review.md` into `review-plan/SKILL.md`. Create stub directories for the five document authoring skills and three implementation skills (content to be filled by their respective features — this task creates the directory structure and minimal placeholder SKILL.md files so the directory layout matches the design). Do NOT delete old files in `.skills/` — they must remain for backward compatibility.

**Specification references:** FR-005 (code review skill split), FR-006 (plan review restructure), FR-007 (document creation skill split — directories only, content from separate feature), FR-009 (vocabulary fields in all skills), FR-010 (anti-pattern sections in all skills), FR-011 (old `.skills/` directory retained), NFR-001 (incremental), NFR-003 (directory structure matches design §7.3)

**Input context:**
- `.skills/code-review.md` — current code review skill (content to split into two skills)
- `.skills/plan-review.md` — current plan review skill (content to restructure)
- `.skills/document-creation.md` — current document creation skill (directories created, content from separate feature)
- `work/spec/3.0-migration.md` FR-005, FR-006, FR-007 — splitting and restructuring acceptance criteria
- `work/design/skills-system-redesign-v2.md` §3.2 — attention-curve SKILL.md format
- `work/design/skills-system-redesign-v2.md` §7.3 — target directory structure

**Output artifacts:**
- New directory `.kbz/skills/` containing:
  - `review-code/SKILL.md` — split from `code-review.md`: individual code review procedure, finding classification, vocabulary, anti-patterns, attention-curve format with frontmatter
  - `review-code/references/` — overflow content if needed
  - `orchestrate-review/SKILL.md` — split from `code-review.md`: multi-reviewer coordination, panel dispatch, result aggregation, attention-curve format with frontmatter
  - `orchestrate-review/references/` — overflow content if needed
  - `review-plan/SKILL.md` — restructured from `plan-review.md`: attention-curve format with frontmatter, gains vocabulary and anti-patterns sections
  - `review-plan/references/` — overflow content if needed
  - `write-design/` — directory created (SKILL.md content from document authoring skill content feature)
  - `write-spec/` — directory created
  - `write-dev-plan/` — directory created
  - `write-research/` — directory created
  - `update-docs/` — directory created
  - `implement-task/` — directory created (SKILL.md content from implementation skill content feature)
  - `orchestrate-development/` — directory created
  - `decompose-feature/` — directory created
- Old files in `.skills/` remain UNCHANGED on disk

**Dependencies:** Task 1 (profile store changes must be available so the handoff tool can detect new-format skills during testing). Task 2 is a soft dependency — the handoff tool extension makes new skills usable in context assembly, but the skill files can be created independently of the handoff code.

**Note on stub directories:** For the 8 skill directories whose content comes from other features (5 document authoring + 3 implementation), this task creates only the directory. It does NOT create placeholder SKILL.md files — those directories remain empty until the content features populate them. This avoids creating files that would need to be overwritten and ensures the content features have clean ownership of their output artifacts.

---

### Task 5: Create Stage Bindings File

**Objective:** Create `.kbz/stage-bindings.yaml` declaring the mapping from feature lifecycle stages to roles, skills, orchestration patterns, and document templates. This file has no predecessor in the current system — it is entirely new. It must cover all document-producing lifecycle stages: `designing`, `specifying`, `dev-planning`, `developing`, `reviewing`, `researching`, `documenting`.

**Specification references:** FR-008 (stage bindings creation), NFR-003 (directory structure)

**Input context:**
- `work/spec/3.0-migration.md` FR-008 — stage bindings acceptance criteria
- `work/design/skills-system-redesign-v2.md` §5.1, §5.2, §5.3 — role/skill/stage mappings per stage
- `work/design/skills-system-redesign-v2.md` §7.3 — file location
- `work/spec/3.0-document-authoring-skill-content.md` FR-002 — stage-to-skill mapping for authoring skills
- `work/spec/3.0-implementation-skill-content.md` FR-002 — stage-to-skill mapping for implementation skills

**Output artifacts:**
- `.kbz/stage-bindings.yaml` — stage binding declarations for all 7 document-producing stages, each entry containing:
  - `stage` — lifecycle stage name
  - `roles` — list of role IDs assigned to this stage
  - `skills` — list of skill names assigned to this stage
  - `orchestration_pattern` — topology (e.g., `orchestrator-workers` for `developing`, `single-agent` for others)
  - `document_template` — (for document-producing stages) required sections list matching the canonical definitions in the design §5.1

**Dependencies:** None for file creation. Tasks 3 and 4 are soft dependencies — the stage bindings reference role and skill names that those tasks create, but the YAML file itself can be written independently since it is declarative configuration. Validation that all referenced roles and skills exist happens at retirement time (Task 6).

---

### Task 6: Retirement Verification and Old File Removal

**Objective:** Verify that all retirement conditions from FR-015 are met, then remove the old `.skills/` skill files and update documentation references. This task is a deliberate, verifiable step — not an implicit side effect. The three conditions are: (1) all stage bindings reference new-format skills exclusively, (2) no `AGENTS.md` or `copilot-instructions.md` reference points to `.skills/` paths, (3) no active workflow or agent session depends on old-format skill files. After verification, remove the migrated skill files from `.skills/` (retain `.skills/README.md` updated to point to new locations). Update `AGENTS.md` and `.github/copilot-instructions.md` to reference new paths.

**Specification references:** FR-015 (retirement of old files), FR-011 (old directory retained UNTIL retirement), NFR-004 (condition-based retirement, not time-based)

**Input context:**
- `.kbz/stage-bindings.yaml` (from Task 5) — verify all entries reference new-format skills
- `AGENTS.md` — scan for `.skills/` path references
- `.github/copilot-instructions.md` — scan for `.skills/` path references
- `.kbz/skills/` directory (from Task 4) — verify all referenced skills exist
- `.kbz/roles/` directory (from Task 3) — verify all referenced roles exist
- `work/spec/3.0-migration.md` FR-015 — retirement conditions and acceptance criteria

**Output artifacts:**
- Retirement verification checklist (documented in commit message):
  1. ✅/❌ All stage bindings reference new-format skills exclusively
  2. ✅/❌ No `AGENTS.md` reference points to `.skills/` paths
  3. ✅/❌ No `.github/copilot-instructions.md` reference points to `.skills/` paths
  4. ✅/❌ All roles referenced by stage bindings exist in `.kbz/roles/`
  5. ✅/❌ All skills referenced by stage bindings exist in `.kbz/skills/`
- If ALL checks pass:
  - Remove `.skills/code-review.md`, `.skills/plan-review.md`, `.skills/document-creation.md`
  - Update `.skills/README.md` to note that skills have moved to `.kbz/skills/` with a directory listing
  - Update `AGENTS.md` to reference `.kbz/skills/` and `.kbz/roles/` paths
  - Update `.github/copilot-instructions.md` to reference new paths
- If ANY check fails:
  - Document which conditions are not met
  - Do NOT remove old files
  - Flag the task as blocked with a description of what remains

**Dependencies:** Tasks 3, 4, and 5 must ALL complete before this task begins. This is the final serialised step.

---

## 3. Dependency Graph

```
Task 1: Profile Store Extension (Go code)
  │
  ├──→ Task 2: Handoff Tool Extension (Go code)
  │       │
  │       └──→ (soft: enables skill context assembly testing)
  │
  ├──→ Task 3: Role File Migration ──────────┐
  ├──→ Task 4: Skill File Restructuring ─────┤
  └──→ Task 5: Stage Bindings File ──────────┤
                                              │
                                              ▼
                                     Task 6: Retirement
```

**Execution phases:**

1. **Phase 1 (serial):** Task 1 → Task 2. These are Go code changes with a compile-time dependency: Task 2 uses the updated `Profile` struct from Task 1.
2. **Phase 2 (parallel):** Tasks 3, 4, 5 execute concurrently. All three are file-creation tasks with no inter-dependencies. They require the Profile Store (Task 1) to support the new location, but do not depend on each other.
3. **Phase 3 (serial):** Task 6 runs only after Tasks 3, 4, and 5 all complete.

**Maximum parallelism:** 3 concurrent agents (during Phase 2).

**Critical path:** Task 1 → Task 2 → Task 3 (or 4) → Task 6.

---

## 4. Interface Contracts

### 4.1 ProfileStore Dual-Root Constructor (Task 1 → Tasks 2, 3)

```go
// NewProfileStore creates a store that searches primaryRoot first, then fallbackRoot.
// If fallbackRoot is "", only primaryRoot is searched (backward compatible).
func NewProfileStore(primaryRoot, fallbackRoot string) *ProfileStore
```

**Wiring location:** The call site that constructs the `ProfileStore` (in `cmd/` or server setup) must pass:
- `primaryRoot`: `.kbz/roles/`
- `fallbackRoot`: `.kbz/context/roles/`

### 4.2 Profile Struct Extensions (Task 1 → Tasks 2, 3)

```go
type AntiPattern struct {
    Name    string `yaml:"name"`
    Detect  string `yaml:"detect"`
    Because string `yaml:"because"`
    Resolve string `yaml:"resolve"`
}

// New fields on Profile (all omitempty for backward compatibility):
Vocabulary   []string      `yaml:"vocabulary,omitempty"`
AntiPatterns []AntiPattern  `yaml:"anti_patterns,omitempty"`
Tools        []string      `yaml:"tools,omitempty"`
```

### 4.3 Skill Format Detection Heuristic (Task 2 → Task 4)

Tasks 2 and 4 must agree on how new-format skills are distinguished from old-format:

| Property | New format | Old format |
|----------|-----------|------------|
| Location | `.kbz/skills/{name}/SKILL.md` | `.skills/{name}.md` |
| Frontmatter | YAML block delimited by `---` with `name:` field | No YAML frontmatter |
| Sections | Attention-curve ordered (`## Vocabulary` first) | Unstructured markdown |

Task 2 (handoff code) detects format by checking for YAML frontmatter with a `name:` field. Task 4 (skill files) must ensure all new SKILL.md files have this frontmatter.

### 4.4 Stage Bindings Schema (Task 5 → Task 6)

```yaml
# .kbz/stage-bindings.yaml
stages:
  - stage: "designing"
    roles: ["architect"]
    skills: ["write-design"]
    orchestration_pattern: "single-agent"
    document_template:
      required_sections:
        - "Problem and Motivation"
        - "Design"
        - "Alternatives Considered"
        - "Decisions"
```

Task 6 validates that every `roles` entry exists as a file in `.kbz/roles/` and every `skills` entry exists as a directory in `.kbz/skills/` with a `SKILL.md` file.

### 4.5 Role File Schema (Task 3 — consumed by Task 1's loader)

All role files in `.kbz/roles/` must conform to:

```yaml
id: "{role-id}"              # required, matches filename
inherits: "{parent-role-id}" # optional
description: "..."           # required
packages: [...]              # optional, existing field
conventions: ...             # optional, existing field
architecture: ...            # optional, existing field
vocabulary:                  # NEW — required (non-empty for all roles)
  - "term one"
  - "term two"
anti_patterns:               # NEW — required (may be empty for base.yaml only)
  - name: "Anti-Pattern Name"
    detect: "Observable signal"
    because: "Why harmful"
    resolve: "Corrective action"
tools:                       # NEW — optional
  - "tool_name"
```

---

## 5. Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | 1, 3 | Task 1: code support for new location; Task 3: files created there |
| FR-002 | 3 | `base.yaml` migrated with new fields |
| FR-003 | 3 | `developer.yaml` → `implementer-go.yaml` with `implementer` parent |
| FR-004 | 3 | `reviewer.yaml` → reviewer + 4 specialist subtypes |
| FR-005 | 4 | `code-review.md` → `review-code/SKILL.md` + `orchestrate-review/SKILL.md` |
| FR-006 | 4 | `plan-review.md` → `review-plan/SKILL.md` |
| FR-007 | 4 | `document-creation.md` → 5 type-specific skill directories (content from separate feature) |
| FR-008 | 5 | `.kbz/stage-bindings.yaml` created |
| FR-009 | 1, 3, 4 | Task 1: struct support; Task 3: role vocabulary; Task 4: skill vocabulary |
| FR-010 | 1, 3, 4 | Task 1: struct support; Task 3: role anti-patterns; Task 4: skill anti-patterns |
| FR-011 | 3, 4 | Both tasks preserve old files on disk |
| FR-012 | 1 | Profile tool extended for dual-location loading and new fields |
| FR-013 | 2 | Handoff tool extended for new skill/role structures |
| FR-014 | 1, 2 | Both tasks preserve existing tool signatures and subsystem behaviour |
| FR-015 | 6 | Retirement verification and conditional old file removal |
| NFR-001 | 1, 2, 3, 4 | Every task maintains system functionality; dual-location fallback ensures incremental safety |
| NFR-002 | 1, 2 | Code changes confined to `internal/context/` and `internal/mcp/` per FR-012 and FR-013 |
| NFR-003 | 3, 4, 5 | Directory structures match design §7.3 |
| NFR-004 | 6 | Retirement is condition-based (checklist), not time-based |