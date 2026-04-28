# Specification: Freshness Tracking for Skills and Roles

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PETZQE (freshness-tracking)                          |
| Design  | `work/design/skills-system-redesign-v2.md` §6.4                   |

---

## 1. Overview

This specification defines the freshness tracking mechanism for role and skill files consumed by the context assembly pipeline. Roles (`.kbz/roles/*.yaml`) and skills (`.kbz/skills/*/SKILL.md`) are operational context read on every task — stale content silently misdirects every agent that receives it. This feature adds `last_verified` metadata to both file types, extends the `health` tool to flag staleness against a configurable window, surfaces staleness warnings in assembled context metadata, and provides a mechanism to update the `last_verified` timestamp. The system does NOT block context assembly for stale files; it warns.

---

## 2. Scope

### 2.1 In Scope

- `last_verified` field on role files (`.kbz/roles/*.yaml`): schema addition, reading, and updating.
- `last_verified` field on skill files (`.kbz/skills/*/SKILL.md`): frontmatter addition, reading, and updating.
- `health` tool extension to detect and report stale roles and skills.
- Configurable staleness window with a default of 30 days.
- Staleness warnings in assembled context metadata (via the context assembly pipeline).
- A refresh mechanism to update the `last_verified` timestamp on a role or skill.
- Alignment with the existing `doc(action: "refresh")` freshness concept for documents.

### 2.2 Out of Scope

- Blocking context assembly for stale roles or skills. The design explicitly states stale files remain usable.
- Automatic content validation or correction of stale files. Freshness tracking records when a human or agent last confirmed the content is current; it does not assess whether the content is actually correct.
- Changes to the role schema beyond the `last_verified` field.
- Changes to the skill file structure beyond the `last_verified` frontmatter field.
- Freshness tracking for stage binding files (bindings are not individually verified in this feature).
- The context assembly pipeline itself (specified in the Context Assembly Pipeline feature).
- Document freshness tracking via `doc(action: "refresh")` — that is an existing capability; this feature extends the same concept to roles and skills.

---

## 3. Functional Requirements

### FR-001: Role `last_verified` Field

Role files (`.kbz/roles/*.yaml`) MUST support a `last_verified` field at the top level of the YAML document. The field value MUST be an ISO 8601 timestamp (e.g., `2025-07-30T14:00:00Z`).

**Acceptance criteria:**
- A role file with `last_verified: 2025-07-30T14:00:00Z` parses without error and the timestamp is accessible to the health and assembly subsystems.
- A role file without a `last_verified` field parses without error; the field is treated as absent (never verified).

---

### FR-002: Skill `last_verified` Field

Skill files (`.kbz/skills/*/SKILL.md`) MUST support a `last_verified` field in YAML frontmatter. The field value MUST be an ISO 8601 timestamp.

**Acceptance criteria:**
- A skill file with frontmatter containing `last_verified: 2025-07-30T14:00:00Z` parses without error and the timestamp is accessible to the health and assembly subsystems.
- A skill file without a `last_verified` field in its frontmatter parses without error; the field is treated as absent (never verified).

---

### FR-003: Health Tool — Stale Role Detection

The `health` tool MUST flag any role file in `.kbz/roles/` whose `last_verified` timestamp is older than the configured staleness window. The flag MUST identify the role file, the `last_verified` date, and how many days past the window it is.

**Acceptance criteria:**
- A role with `last_verified` 45 days ago and a 30-day window is flagged with a message identifying the role, the last-verified date, and that it is 15 days overdue.
- A role with `last_verified` 10 days ago and a 30-day window is NOT flagged.

---

### FR-004: Health Tool — Stale Skill Detection

The `health` tool MUST flag any skill file in `.kbz/skills/` whose `last_verified` timestamp is older than the configured staleness window. The flag MUST identify the skill directory, the `last_verified` date, and how many days past the window it is.

**Acceptance criteria:**
- A skill with `last_verified` 60 days ago and a 30-day window is flagged with a message identifying the skill, the last-verified date, and that it is 30 days overdue.
- A skill with `last_verified` 25 days ago and a 30-day window is NOT flagged.

---

### FR-005: Health Tool — Never-Verified Detection

The `health` tool MUST flag any role or skill file that has no `last_verified` field. The flag MUST indicate that the file has never been verified and MUST recommend setting an initial `last_verified` value.

**Acceptance criteria:**
- A role file with no `last_verified` field is flagged with a message containing "never verified" (or equivalent) and a recommendation to verify.
- A skill file with no `last_verified` field in its frontmatter is flagged with the same pattern.
- The flag is distinct from the "stale" flag (FR-003/FR-004) — it indicates absence rather than expiry.

---

### FR-006: Configurable Staleness Window

The staleness window MUST be configurable. The default value MUST be 30 days. The configuration MUST accept a positive integer representing the number of days.

**Acceptance criteria:**
- With the default configuration (30 days), a role verified 31 days ago is flagged as stale.
- With the window configured to 60 days, a role verified 31 days ago is NOT flagged as stale.
- A window value of 0 or negative MUST be rejected with an error.

---

### FR-007: Staleness Warning in Health Report

The `health` tool output MUST include a dedicated section or category for role and skill freshness. Stale and never-verified entries MUST be grouped separately from other health findings.

**Acceptance criteria:**
- The health report contains a section identifiable as role/skill freshness (by heading or category label).
- Stale roles and stale skills appear within that section.
- Never-verified roles and skills appear within that section.
- Roles and skills that are within the freshness window do NOT appear in this section.

---

### FR-008: Staleness Warning in Assembled Context Metadata

When the context assembly pipeline assembles context using a stale or never-verified role or skill, the assembled output MUST include a metadata warning indicating the staleness. The warning MUST identify which file (role, skill, or both) is stale and its `last_verified` date (or "never" if absent).

**Acceptance criteria:**
- Assembled context using a role verified 45 days ago (with a 30-day window) contains a metadata warning identifying the role and its last-verified date.
- Assembled context using a skill that has never been verified contains a metadata warning identifying the skill as "never verified."
- Assembled context using a role and skill both within the freshness window contains no staleness warning.
- Assembled context using a stale role and a fresh skill contains a warning for the role only.

---

### FR-009: Stale Files Do Not Block Assembly

The context assembly pipeline MUST NOT refuse to assemble context when a role or skill is stale or never-verified. Assembly MUST proceed normally with the stale content included, and the staleness warning (FR-008) attached as metadata.

**Acceptance criteria:**
- Calling `handoff` with a task that resolves to a stale role produces assembled context (not an error).
- Calling `handoff` with a task that resolves to a never-verified skill produces assembled context (not an error).
- The assembled context content (sections 1–10) is identical whether the role/skill is fresh or stale; only the metadata differs.

---

### FR-010: Refresh Mechanism for Roles

There MUST be a mechanism to update the `last_verified` field on a role file to the current timestamp. The mechanism MUST write the ISO 8601 timestamp to the role's YAML file without altering any other fields.

**Acceptance criteria:**
- After refreshing a role, the role file's `last_verified` field contains a timestamp within 1 second of the current time.
- All other fields in the role YAML file are unchanged after the refresh.
- Refreshing a role that had no `last_verified` field adds the field with the current timestamp.

---

### FR-011: Refresh Mechanism for Skills

There MUST be a mechanism to update the `last_verified` field on a skill file's frontmatter to the current timestamp. The mechanism MUST write the ISO 8601 timestamp to the skill's frontmatter without altering the skill's Markdown content or other frontmatter fields.

**Acceptance criteria:**
- After refreshing a skill, the skill file's frontmatter `last_verified` field contains a timestamp within 1 second of the current time.
- The skill's Markdown content (everything after the frontmatter closing `---`) is unchanged after the refresh.
- All other frontmatter fields are unchanged after the refresh.
- Refreshing a skill that had no `last_verified` frontmatter field adds the field with the current timestamp.

---

### FR-012: Conceptual Alignment with Document Refresh

The refresh mechanism for roles and skills MUST follow the same conceptual model as `doc(action: "refresh")` for documents: the action records that a human or agent has reviewed the content and confirmed it is current. The refresh MUST NOT imply automated validation of the content.

**Acceptance criteria:**
- The refresh mechanism for roles and skills is invocable in the same manner as `doc(action: "refresh")` — by specifying the target entity (role name or skill name).
- The mechanism's documentation or help text describes the action as confirming that the content has been reviewed and is current.

---

### FR-013: Health Report Summary Counts

The `health` tool MUST include summary counts for role and skill freshness: the number of fresh roles, stale roles, never-verified roles, fresh skills, stale skills, and never-verified skills.

**Acceptance criteria:**
- A project with 3 roles (2 fresh, 1 stale) and 4 skills (3 fresh, 0 stale, 1 never-verified) produces a health summary showing: 2 fresh roles, 1 stale role, 0 never-verified roles, 3 fresh skills, 0 stale skills, 1 never-verified skill.
- The counts are accurate and sum to the total number of role and skill files present.

---

### FR-014: Staleness Window Source

The staleness window configuration MUST be read from a project-level configuration location (e.g., `.kbz/config.yaml` or equivalent). If no configuration is present, the default of 30 days MUST be used.

**Acceptance criteria:**
- A project with no staleness window configuration uses 30 days as the window.
- A project with the staleness window set to 14 days uses 14 days.
- The configuration value is read once per `health` tool invocation and once per context assembly, not cached across invocations.

---

## 4. Non-Functional Requirements

### NFR-001: Zero Impact on Assembly Performance

The staleness check during context assembly (FR-008, FR-009) MUST add no more than 10 milliseconds to assembly time. The check reads the `last_verified` field from files already loaded by the pipeline and compares against the current time.

**Acceptance criteria:**
- Benchmark test: context assembly with staleness checking enabled completes within 10 milliseconds of assembly without staleness checking.

---

### NFR-002: Refresh Atomicity

The refresh mechanism MUST write the `last_verified` field atomically. If the write fails (e.g., disk full, permission error), the original file MUST be preserved unchanged.

**Acceptance criteria:**
- Simulated write failure leaves the original role/skill file intact and unchanged.
- No partial writes (e.g., truncated YAML or corrupted frontmatter) result from a failed refresh.

---

### NFR-003: Backward Compatibility — Existing Files Without `last_verified`

Existing role and skill files that do not contain a `last_verified` field MUST continue to parse and function. The absence of the field MUST NOT cause errors in any tool: `health`, `handoff`, or any role/skill loading path.

**Acceptance criteria:**
- A role file created before this feature (no `last_verified` field) loads successfully in all tools.
- A skill file created before this feature (no `last_verified` in frontmatter) loads successfully in all tools.
- The `health` tool reports these as "never verified" (FR-005) rather than erroring.

---

### NFR-004: Timestamp Precision

The `last_verified` timestamp MUST be stored with at least second-level precision in ISO 8601 format with UTC timezone designator (e.g., `2025-07-30T14:30:00Z`). Sub-second precision is permitted but not required.

**Acceptance criteria:**
- A refreshed `last_verified` value matches the pattern `YYYY-MM-DDTHH:MM:SSZ` (with optional fractional seconds).
- Timestamps are in UTC regardless of the system's local timezone.

---

## 5. Acceptance Criteria

| Requirement | Verification Method |
|-------------|---------------------|
| FR-001 | Unit test: role YAML with and without `last_verified` parses correctly |
| FR-002 | Unit test: skill frontmatter with and without `last_verified` parses correctly |
| FR-003 | Unit test: role 45 days old flagged with 30-day window; role 10 days old not flagged |
| FR-004 | Unit test: skill 60 days old flagged with 30-day window; skill 25 days old not flagged |
| FR-005 | Unit test: role/skill with no `last_verified` flagged as "never verified" |
| FR-006 | Unit test: custom window of 60 days; role at 31 days not flagged; default window flags at 31 days |
| FR-007 | Integration test: health report contains a freshness section with stale and never-verified entries |
| FR-008 | Integration test: assembled context metadata contains staleness warning for stale role; no warning for fresh role |
| FR-009 | Integration test: `handoff` with stale role/skill returns assembled context (not an error) |
| FR-010 | Unit test: refresh role sets `last_verified` to current time; other fields unchanged |
| FR-011 | Unit test: refresh skill sets frontmatter `last_verified` to current time; content and other fields unchanged |
| FR-012 | Code review: refresh mechanism documentation describes it as confirming content currency |
| FR-013 | Unit test: health summary counts match actual fresh/stale/never-verified role and skill files |
| FR-014 | Unit test: absent config uses 30-day default; present config with 14 days uses 14 days |
| NFR-001 | Benchmark test: staleness check adds < 10ms to assembly |
| NFR-002 | Unit test: simulated write failure leaves original file intact |
| NFR-003 | Integration test: pre-existing role/skill files without `last_verified` load without error in all tools |
| NFR-004 | Unit test: refreshed timestamp matches ISO 8601 UTC pattern |

---

## 6. Dependencies and Assumptions

### Dependencies

- **Role System feature:** The role YAML schema must accept an optional `last_verified` field at the top level. This feature adds the field; the Role System feature defines the rest of the schema.
- **Skill System feature:** The skill SKILL.md frontmatter must accept an optional `last_verified` field. This feature adds the field; the Skill System feature defines the rest of the schema.
- **Existing `health` tool (`internal/mcp/health.go` or equivalent):** The staleness checks (FR-003 through FR-007, FR-013) extend the existing health reporting infrastructure. The health tool must support adding new check categories.
- **Context Assembly Pipeline feature (FEAT-01KN5-88PE43M6):** Staleness warnings in assembled context (FR-008) require the assembly pipeline to accept and include metadata warnings. The pipeline must expose a mechanism for freshness checks to attach warnings to the output.
- **Existing `doc(action: "refresh")` pattern:** The refresh mechanism is modelled after the document refresh action. Familiarity with that pattern informs the interface design.

### Assumptions

- Role files are stored at `.kbz/roles/*.yaml` and skill files at `.kbz/skills/*/SKILL.md`. If the Role System or Skill System features change these paths, this specification must be updated accordingly.
- The system clock provides accurate UTC time for timestamp generation and staleness comparison.
- YAML parsing preserves field ordering and comments when writing the `last_verified` field. If the YAML library does not preserve comments, this is an acceptable limitation documented in the refresh mechanism.
- The number of role and skill files in a typical project is small (fewer than 50 each), so iterating all files for health checks is acceptable without indexing.