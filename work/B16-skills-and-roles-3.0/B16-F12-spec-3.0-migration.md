# Specification: Migration and Backward Compatibility

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PGXATW (migration-and-backward-compat)               |
| Design  | `work/design/skills-system-redesign-v2.md` §7, §11                |

---

## 1. Overview

This specification defines the file restructuring, renaming, splitting, and backward compatibility requirements for migrating from the current skills and roles system to the Kanbanzai 3.0 skills/roles architecture. It covers the relocation of role files from `.kbz/context/roles/` to `.kbz/roles/`, the renaming and restructuring of existing roles, the splitting of monolithic skills into type-specific skills, the introduction of new files (`stage-bindings.yaml`, vocabulary fields, anti-pattern sections), the backward compatibility contract that preserves access to old paths during the transition period, and the retirement procedure for old files once all references are updated.

---

## 2. Scope

### 2.1 In Scope

- File relocation: role files from `.kbz/context/roles/` to `.kbz/roles/`.
- Role restructuring: `base.yaml`, `developer.yaml` → `implementer-go.yaml`, `reviewer.yaml` → `reviewer.yaml` + 4 specialist subtypes.
- Skill splitting: `.skills/code-review.md` → `review-code/SKILL.md` + `orchestrate-review/SKILL.md`; `.skills/plan-review.md` → `review-plan/SKILL.md`; `.skills/document-creation.md` → `write-design/`, `write-spec/`, `write-dev-plan/`, `write-research/`, `update-docs/`.
- New file creation: `.kbz/stage-bindings.yaml`, vocabulary fields in all roles and skills, anti-pattern sections in all roles and skills.
- Backward compatibility: `.skills/` directory retained during transition; `profile` and `handoff` tools extended to support new structures.
- Retirement: procedure and conditions for removing old files.
- Enumeration of what stays the same (unchanged subsystems).

### 2.2 Explicitly Excluded

- The content of new roles and skills (covered by role content, document authoring skill content, implementation skill content, and review skill content specifications).
- The YAML schema for role files (covered by the role system specification).
- The SKILL.md file format schema (covered by the skill system specification).
- The binding registry schema and `stage-bindings.yaml` internal structure (covered by the binding registry specification).
- Context assembly pipeline changes (covered by the context assembly pipeline specification).
- Gate enforcement mechanism and stage transition logic (covered by the workflow specification).
- Automated migration tooling or scripts that perform the file moves programmatically.

---

## 3. Functional Requirements

### FR-001: Role Directory Relocation

All role files MUST be stored at `.kbz/roles/{id}.yaml` in the new system. The current location `.kbz/context/roles/` MUST NOT be the canonical location for any role file after migration is complete.

**Acceptance criteria:**
- Every role file required by the 3.0 system exists under `.kbz/roles/`
- No stage binding, skill `roles` field, or `profile` tool invocation references `.kbz/context/roles/` as the canonical source after migration
- The `.kbz/roles/` directory contains all role files listed in the design §7.3 directory structure

---

### FR-002: Base Role Restructuring

The file `.kbz/context/roles/base.yaml` MUST be migrated to `.kbz/roles/base.yaml`. The migrated file MUST retain all existing fields and MUST gain the following new fields: `vocabulary` (list of strings), `anti_patterns` (list of objects), and `tools` (list of strings, optional). Existing field values that are compatible with the new schema MUST be preserved; fields that are incompatible MUST be restructured to conform to the new role schema.

**Acceptance criteria:**
- `.kbz/roles/base.yaml` exists and conforms to the 3.0 role schema
- The `vocabulary` field is present and contains a non-empty list of project-wide terms
- The `anti_patterns` field is present (may be empty for the base role if anti-patterns are carried by inheriting roles)
- Existing base role content (identity, inheritance-related fields) is preserved or mapped to equivalent new-schema fields
- `.kbz/context/roles/base.yaml` is no longer required by any 3.0 component after migration

---

### FR-003: Developer Role Rename and Restructure

The file `.kbz/context/roles/developer.yaml` MUST be migrated to `.kbz/roles/implementer-go.yaml`. The `id` field MUST change from `developer` to `implementer-go`. The migrated file MUST gain `vocabulary` and `anti_patterns` fields with Go-specific content. The role MUST inherit from `implementer` (the base implementation role), not directly from `base`.

**Acceptance criteria:**
- `.kbz/roles/implementer-go.yaml` exists with `id: implementer-go`
- The `inherits` field references `implementer` (not `base` or `developer`)
- A base `implementer` role exists at `.kbz/roles/implementer.yaml` that `implementer-go` inherits from
- The `vocabulary` field contains Go-specific terms
- `.kbz/context/roles/developer.yaml` is no longer required by any 3.0 component after migration
- No role file or stage binding references the identifier `developer`

---

### FR-004: Reviewer Role Split

The file `.kbz/context/roles/reviewer.yaml` MUST be migrated to `.kbz/roles/reviewer.yaml` (base review role) plus four specialist subtypes: `reviewer-conformance.yaml`, `reviewer-quality.yaml`, `reviewer-security.yaml`, `reviewer-testing.yaml`. Each specialist MUST inherit from `reviewer`. Each specialist MUST carry dimension-specific vocabulary and anti-patterns that the base `reviewer` role does not carry.

**Acceptance criteria:**
- `.kbz/roles/reviewer.yaml` exists as the base review role
- All four specialist files exist: `reviewer-conformance.yaml`, `reviewer-quality.yaml`, `reviewer-security.yaml`, `reviewer-testing.yaml`
- Each specialist's `inherits` field references `reviewer`
- Each specialist has a non-empty `vocabulary` field with terms specific to its review dimension
- Each specialist has a non-empty `anti_patterns` field with anti-patterns specific to its review dimension
- The base `reviewer.yaml` does not duplicate vocabulary or anti-patterns that are specific to a single specialist dimension
- `.kbz/context/roles/reviewer.yaml` is no longer required by any 3.0 component after migration

---

### FR-005: Code Review Skill Split

The file `.skills/code-review.md` MUST be migrated into two new skills: `.kbz/skills/review-code/SKILL.md` (the individual code review skill) and `.kbz/skills/orchestrate-review/SKILL.md` (the review orchestration skill). Both MUST follow the attention-curve SKILL.md format. The review procedure and finding classification content MUST go to `review-code`. The multi-reviewer coordination, panel dispatch, and result aggregation content MUST go to `orchestrate-review`.

**Acceptance criteria:**
- `.kbz/skills/review-code/SKILL.md` exists and follows the attention-curve section ordering
- `.kbz/skills/orchestrate-review/SKILL.md` exists and follows the attention-curve section ordering
- `review-code` contains the procedure for evaluating code against a specification and classifying findings
- `orchestrate-review` contains the procedure for dispatching specialist reviewers and aggregating results
- No single skill file contains both individual review procedure and multi-reviewer orchestration logic
- The combined content of both new skills covers all procedural content from the original `.skills/code-review.md`

---

### FR-006: Plan Review Skill Restructure

The file `.skills/plan-review.md` MUST be migrated to `.kbz/skills/review-plan/SKILL.md`. The content MUST be restructured to follow the attention-curve SKILL.md format (Vocabulary → Anti-Patterns → Checklist → Procedure → Output Format → Examples → Evaluation Criteria → Questions This Skill Answers).

**Acceptance criteria:**
- `.kbz/skills/review-plan/SKILL.md` exists and follows the attention-curve section ordering
- The skill carries a `vocabulary` section (not present in the current `.skills/plan-review.md`)
- The skill carries an `anti_patterns` section (not present in the current `.skills/plan-review.md`)
- The procedural content from `.skills/plan-review.md` is preserved in the `## Procedure` section
- The skill has a YAML frontmatter block with all required fields (`name`, `description`, `triggers`, `roles`, `stage`, `constraint_level`)

---

### FR-007: Document Creation Skill Split by Type

The file `.skills/document-creation.md` MUST be migrated into five type-specific authoring skills under `.kbz/skills/`: `write-design/SKILL.md`, `write-spec/SKILL.md`, `write-dev-plan/SKILL.md`, `write-research/SKILL.md`, `update-docs/SKILL.md`. Each MUST follow the attention-curve SKILL.md format and carry type-specific vocabulary, anti-patterns, and output templates.

**Acceptance criteria:**
- All five skill directories exist under `.kbz/skills/` with a `SKILL.md` file in each
- Each skill follows the attention-curve section ordering
- Each skill's vocabulary is specific to its document type (not generic document-creation vocabulary)
- Each skill's output format defines the template for its document type
- The original `.skills/document-creation.md` generic procedure is not replicated verbatim in any single new skill
- No new authoring skill is generic across document types

---

### FR-008: New File — Stage Bindings

A new file `.kbz/stage-bindings.yaml` MUST be created as part of the migration. This file MUST declare the mapping from feature lifecycle stages to roles, skills, orchestration patterns, and document templates. This file has no predecessor in the current system.

**Acceptance criteria:**
- `.kbz/stage-bindings.yaml` exists after migration
- The file contains entries for all document-producing lifecycle stages (`designing`, `specifying`, `dev-planning`, `developing`, `reviewing`, `researching`, `documenting`)
- Each entry references the role(s) and skill(s) that the design (§5.1, §5.2, §5.3) assigns to that stage
- The file is loadable by the binding registry component (schema conformance verified by the binding registry spec)

---

### FR-009: New Content — Vocabulary Fields

Every role file under `.kbz/roles/` and every SKILL.md under `.kbz/skills/` MUST carry vocabulary content that did not exist in the current system. For roles, the `vocabulary` field MUST be a non-empty YAML list. For skills, the `## Vocabulary` section MUST be a non-empty list of domain-specific terms. No role or skill file in the 3.0 system is permitted to have an empty or missing vocabulary payload.

**Acceptance criteria:**
- Every `.yaml` file in `.kbz/roles/` has a `vocabulary` field containing at least one term
- Every `SKILL.md` file in `.kbz/skills/*/` has a `## Vocabulary` section containing at least one term
- A lint or validation pass across all role and skill files confirms zero files with empty vocabulary

---

### FR-010: New Content — Anti-Pattern Sections

Every role file under `.kbz/roles/` (except `base.yaml`, where anti-patterns are optional) and every SKILL.md under `.kbz/skills/` MUST carry anti-pattern content that did not exist in the current system. For roles, the `anti_patterns` field MUST be a list of objects with `name`, `detect`, `because`, and `resolve` keys. For skills, the `## Anti-Patterns` section MUST contain named anti-patterns with Detect, BECAUSE, and Resolve sub-fields.

**Acceptance criteria:**
- Every non-base `.yaml` file in `.kbz/roles/` has an `anti_patterns` field containing at least one entry
- `base.yaml` has the `anti_patterns` field present (may be an empty list)
- Every `SKILL.md` file in `.kbz/skills/*/` has a `## Anti-Patterns` section containing at least one anti-pattern
- Each anti-pattern entry (in both roles and skills) has all four required fields: name, detect, because, resolve

---

### FR-011: Backward Compatibility — Old Skills Directory Retention

The `.skills/` directory at the project root MUST be retained during the migration period. Agents that read `.skills/code-review.md`, `.skills/plan-review.md`, or `.skills/document-creation.md` directly MUST continue to find those files at their current paths until retirement (FR-015).

**Acceptance criteria:**
- The `.skills/` directory and its files continue to exist on disk after the new `.kbz/skills/` structure is created
- Reading `.skills/code-review.md` returns valid content (either the original file or a redirect/symlink to the new location)
- Reading `.skills/plan-review.md` returns valid content
- Reading `.skills/document-creation.md` returns valid content
- No migration step deletes or empties the `.skills/` directory or its files

---

### FR-012: Backward Compatibility — Profile Tool Extension

The `profile(action: "get")` tool MUST be extended (not replaced) to support the new role location and schema. Existing callers that request a profile by its current ID (e.g., `profile(action: "get", id: "reviewer")`) MUST continue to receive a valid response. The tool MUST additionally support loading roles from `.kbz/roles/` and resolving the new schema fields (`vocabulary`, `anti_patterns`, `tools`).

**Acceptance criteria:**
- `profile(action: "get", id: "reviewer")` returns a valid response both before and after migration
- `profile(action: "get", id: "implementer-go")` returns the new role after migration
- The tool resolves vocabulary and anti-pattern fields from the new schema when loading from `.kbz/roles/`
- The tool does not break for callers that do not use or expect the new fields
- `profile(action: "list")` returns roles from the new location after migration

---

### FR-013: Backward Compatibility — Handoff Tool Extension

The `handoff` tool MUST be extended (not replaced) to support the new role and skill structures. The tool MUST assemble context using the new attention-curve ordering when the new structures are available, while continuing to function with the old structures during the transition period.

**Acceptance criteria:**
- `handoff` produces valid output when only old-format roles and skills exist (pre-migration state)
- `handoff` produces valid output when new-format roles and skills exist (post-migration state)
- `handoff` assembles context with attention-curve ordering when new-format skills are available
- The tool does not require all roles and skills to be migrated simultaneously — it handles a mixed state where some files are in the old format and some in the new

---

### FR-014: Unchanged Subsystems

The following subsystems MUST remain unchanged by the migration. The migration MUST NOT modify their behaviour, interfaces, or data formats:

1. **Profile inheritance mechanism** (`internal/context/resolve.go`) — extended to handle new fields, but the inheritance resolution algorithm itself is unchanged.
2. **Feature lifecycle state machine** (`internal/validate/lifecycle.go`) — the binding registry maps TO existing states; it does not change them.
3. **Document types and stages** — the binding registry maps FROM existing document types; it does not change them.
4. **Knowledge system** — unchanged except for the addition of auto-surfacing (a separate concern).
5. **MCP tool surface** — no tools are added, removed, or have their signatures changed. Tool filtering is additive (selecting a subset), not modifying.

**Acceptance criteria:**
- The feature lifecycle state machine accepts the same set of states and transitions before and after migration
- All existing document types (`design`, `specification`, `dev-plan`, `research`, `report`) remain valid and function identically
- The `doc`, `entity`, `knowledge`, `finish`, `next`, and `handoff` MCP tools retain their existing parameter signatures
- Profile inheritance resolution (child inherits and overrides parent fields) continues to work identically for existing role hierarchies
- No existing MCP tool is removed or has required parameters added

---

### FR-015: Retirement of Old Files

Once all of the following conditions are met, the old `.skills/` files MUST be retired (deleted or archived): (1) all stage bindings reference new-format skills exclusively, (2) no AGENTS.md or copilot-instructions reference points to `.skills/` paths, (3) no active workflow or agent session depends on old-format skill files. The retirement MUST be a deliberate, verifiable step — not an implicit side effect of migration.

**Acceptance criteria:**
- A documented checklist exists for verifying retirement readiness (the three conditions above)
- Retirement does not occur until all three conditions are verified
- After retirement, the `.skills/` directory at the project root no longer contains the migrated skill files (`code-review.md`, `plan-review.md`, `document-creation.md`)
- After retirement, no component in the system fails due to missing `.skills/` files
- The `.skills/README.md` may be retained or updated to point to the new locations

---

## 4. Non-Functional Requirements

**NFR-001:** The migration MUST be executable incrementally. It MUST be possible to migrate roles and skills one at a time rather than requiring an atomic all-at-once cutover. At every intermediate state (some files migrated, others not), the system MUST remain functional.

**Acceptance criteria:**
- The system functions correctly with only `base.yaml` migrated and all other files in their old locations
- The system functions correctly with roles fully migrated but skills still in `.skills/`
- The system functions correctly with skills fully migrated but `.skills/` directory still present
- No intermediate migration state causes tool errors, missing context, or broken stage transitions

---

**NFR-002:** The migration MUST NOT require changes to any files outside of `.kbz/`, `.skills/`, `.github/copilot-instructions.md`, and `AGENTS.md`. Application source code in `cmd/`, `internal/`, and `pkg/` is modified only as part of the tool extension work (FR-012, FR-013), not as a migration step.

**Acceptance criteria:**
- No migration step modifies files in `cmd/`, `internal/`, or `pkg/` beyond what is required by FR-012 and FR-013
- No migration step modifies `go.mod`, `go.sum`, or test fixtures unrelated to role/skill loading
- Configuration changes are confined to `.kbz/`, `.skills/`, and documentation reference files

---

**NFR-003:** The new `.kbz/roles/` and `.kbz/skills/` directory structures MUST follow the layout specified in the design §7.3. Each skill directory MUST contain at minimum a `SKILL.md` file and MAY contain `references/` and `scripts/` subdirectories.

**Acceptance criteria:**
- The directory tree under `.kbz/roles/` matches the file listing in design §7.3
- The directory tree under `.kbz/skills/` matches the directory listing in design §7.3
- Every skill directory contains a `SKILL.md` file
- No skill directory contains files outside of `SKILL.md`, `references/`, and `scripts/`

---

**NFR-004:** The backward compatibility period (during which both old and new paths are functional) MUST last until retirement conditions (FR-015) are explicitly verified. There is no time-based expiration — compatibility is condition-based, not date-based.

**Acceptance criteria:**
- No automated process removes old files based on elapsed time
- Old files are removed only after the FR-015 checklist is satisfied
- The backward compatibility contract is documented and referenced in AGENTS.md or equivalent

---

## 5. Acceptance Criteria

The requirements above include inline acceptance criteria. The following are system-level acceptance criteria for the feature as a whole:

1. **Directory structure:** After migration, `.kbz/roles/` contains all role files and `.kbz/skills/` contains all skill directories as specified in design §7.3.
2. **No orphan references:** No stage binding, skill frontmatter, tool invocation, or documentation reference points to a path that does not exist.
3. **Backward compatibility:** During the transition period, agents using old paths (`.skills/code-review.md`, `.kbz/context/roles/developer.yaml`, etc.) continue to receive valid content.
4. **Tool continuity:** `profile(action: "get")` and `handoff` work correctly in pre-migration, mid-migration, and post-migration states.
5. **Unchanged subsystems:** The feature lifecycle state machine, document types, knowledge system, and MCP tool signatures are identical before and after migration.
6. **Clean retirement:** After retirement conditions are met, old files are removed and the system functions without them.
7. **Incremental safety:** The migration can be executed one file or one group at a time without breaking system functionality at any intermediate point.

---

## 6. Dependencies and Assumptions

### Dependencies

- **Role system specification:** The new role YAML schema must be defined before role files can be migrated to the new format. The role system spec defines the target schema that migrated files must conform to.
- **Skill system specification:** The SKILL.md file format (frontmatter fields, attention-curve section ordering) must be defined before skills can be restructured. The skill system spec defines the target format.
- **Binding registry specification:** The `stage-bindings.yaml` schema must be defined before the file can be created (FR-008). The binding registry spec defines the required structure.
- **Role and skill content specifications:** The content for new and restructured roles and skills must be authored as part of separate features (document authoring skill content, implementation skill content, review skill content, role content). This migration spec defines WHERE files go; the content specs define WHAT they contain.
- **Profile tool implementation:** The `profile` tool's extension (FR-012) depends on the role system schema being finalized so the loader can be updated.
- **Handoff tool implementation:** The `handoff` tool's extension (FR-013) depends on both the role system and skill system schemas being finalized.

### Assumptions

- The existing `.kbz/context/roles/` files (`base.yaml`, `developer.yaml`, `reviewer.yaml`) are the only role files that require migration. No other role files exist in the current system.
- The existing `.skills/` files (`code-review.md`, `plan-review.md`, `document-creation.md`) are the only skill files that require migration. The `.skills/README.md` is informational and does not require functional migration.
- The profile inheritance mechanism in `internal/context/resolve.go` can be extended to merge `vocabulary` and `anti_patterns` fields without breaking existing inheritance chains.
- The `context assembly function` in `internal/context/assemble.go` can be extended to support attention-curve ordering without changing its function signature or breaking existing callers.
- Git history for migrated files is acceptable to lose at the file level (moves appear as delete + create in Git); content provenance is tracked by the design document and migration commit messages, not by Git file history.
- The `.skills/` directory retention (FR-011) uses actual files on disk (not symlinks or redirects) unless the implementation team determines symlinks are more maintainable. Either approach satisfies the requirement as long as agents reading old paths receive valid content.