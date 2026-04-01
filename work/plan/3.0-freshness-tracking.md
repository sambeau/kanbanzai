# Implementation Plan: Freshness Tracking for Skills and Roles

| Field          | Value                                                    |
|----------------|----------------------------------------------------------|
| Specification  | `work/spec/3.0-freshness-tracking.md`                    |
| Feature        | FEAT-01KN5-88PETZQE (freshness-tracking)                 |
| Design         | `work/design/skills-system-redesign-v2.md` §6.4          |
| Status         | Draft                                                    |

---

## 1. Overview

This plan decomposes the freshness tracking specification into implementable tasks. The feature adds `last_verified` metadata to role YAML files and skill SKILL.md frontmatter, extends the health system to detect and report stale or never-verified roles/skills, surfaces staleness warnings in assembled context metadata, and provides a refresh mechanism to update `last_verified` timestamps.

The implementation builds on:
- The health check infrastructure in `internal/health/` (category-based checks, severity levels, formatting, summary counts)
- The context assembly pipeline in `internal/mcp/assembly.go` (shared by `next` and `handoff` tools)
- The profile loading system in `internal/context/profile.go`
- The config system in `internal/config/config.go`
- The existing `doc(action: "refresh")` pattern in `internal/service/documents.go` and `internal/mcp/doc_tool.go`

**Dependencies:** The Role System and Skill System features define the base schemas for role YAML and skill frontmatter. This feature adds the `last_verified` field to those schemas. Tasks in this plan that touch schema parsing depend on those base schemas existing.

**Scope boundaries (from specification):**
- IN: `last_verified` field, health detection, configurable window, assembly warnings, refresh mechanism
- OUT: Blocking assembly for stale files, automatic content validation, stage binding freshness, the assembly pipeline itself

---

## 2. Task Breakdown

### Task 1: Staleness Configuration and Core Detection Logic

**Objective:** Add the configurable staleness window to `Config` and implement pure functions that classify a `last_verified` timestamp as fresh, stale, or never-verified against a given window. These functions are the shared kernel used by both health checks (Task 2) and assembly warnings (Task 3).

**Specification references:** FR-006, FR-014, NFR-003, NFR-004

**Input context:**
- `internal/config/config.go` — `Config` struct, `DefaultConfig()`, existing config sections like `BranchTrackingConfig` as a pattern
- `internal/health/health.go` — `Severity` type, `Issue` struct (understand the data model the detection results feed into)
- Specification §FR-006: window must be a positive integer (days), default 30, reject 0 or negative
- Specification §FR-014: read from project config, default if absent, read once per invocation

**Output artifacts:**
- Modified `internal/config/config.go`: new `FreshnessConfig` struct with `StalenessWindowDays int` field (default 30), added to `Config` struct, wired into `DefaultConfig()`
- New file `internal/health/freshness.go`: pure functions for staleness classification:
  - `FreshnessStatus` type (fresh/stale/never-verified)
  - `ClassifyFreshness(lastVerified time.Time, isZero bool, window int, now time.Time) FreshnessStatus` — returns the classification
  - `StalenessDetail` struct holding the classification, days overdue (if stale), and the `last_verified` value
  - `ValidateStalenessWindow(days int) error` — rejects 0 or negative
- New file `internal/health/freshness_test.go`: unit tests covering:
  - Fresh (within window), stale (past window), never-verified (zero time)
  - Custom window (60 days) where 31 days is fresh
  - Default window (30 days) where 31 days is stale
  - Window validation: 0, negative, positive
  - Backward compatibility: zero-value `last_verified` treated as never-verified, not an error
  - Timestamp precision: function accepts any `time.Time`, does not enforce format (format enforcement is at parse/write boundaries)
- Modified `internal/config/config.go` test (if config tests exist): verify default staleness window is 30

**Dependencies:** None — this is a leaf task with no upstream dependencies.

---

### Task 2: Health Check Category for Stale Roles and Skills

**Objective:** Add a new health check category that scans `.kbz/roles/*.yaml` and `.kbz/skills/*/SKILL.md` files, reads their `last_verified` field, classifies each using the core detection logic from Task 1, and produces health issues (warnings for stale, warnings for never-verified). Include summary counts (fresh/stale/never-verified) for roles and skills in the health output.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-007, FR-013, NFR-003

**Input context:**
- `internal/health/categories.go` — existing check functions (`CheckWorktree`, `CheckBranch`, `CheckKnowledgeStaleness`, etc.) as the pattern for new category checks
- `internal/health/check.go` — `RunHealthCheck()`, `CheckOptions` struct — where the new category is wired in
- `internal/health/format.go` — `FormatHealthResult`, `CountBySeverity`, `Summary` — formatting functions that will automatically pick up the new category
- `internal/health/health.go` — `CategoryResult`, `Issue`, `NewCategoryResult`, `AddIssue`
- `internal/mcp/health_tool.go` — `Phase3HealthChecker`, `mergeHealthResult` — where the new check is invoked and merged into the health report
- `internal/context/profile.go` — `Profile` struct, `ProfileStore.Load()`, `ProfileStore.LoadAll()` — role loading (the `last_verified` field will be parsed here or in a helper)
- Specification §FR-001, §FR-002: `last_verified` is an ISO 8601 timestamp at top level of role YAML / in skill frontmatter
- Specification §FR-003, §FR-004: flag must identify the file, last-verified date, and days overdue
- Specification §FR-005: never-verified is a distinct flag from stale, with "never verified" wording
- Specification §FR-007: dedicated category/section in health output
- Specification §FR-013: summary counts for fresh/stale/never-verified roles and skills

**Output artifacts:**
- New file `internal/health/freshness_check.go`:
  - `CheckRoleFreshness(rolesDir string, window int, now time.Time) CategoryResult` — scans `rolesDir/*.yaml`, parses `last_verified` from each, classifies, emits issues
  - `CheckSkillFreshness(skillsDir string, window int, now time.Time) CategoryResult` — scans `skillsDir/*/SKILL.md`, parses `last_verified` from frontmatter, classifies, emits issues
  - `FreshnessSummary` struct with counts: `FreshRoles`, `StaleRoles`, `NeverVerifiedRoles`, `FreshSkills`, `StaleSkills`, `NeverVerifiedSkills`
  - `ComputeFreshnessSummary(rolesDir, skillsDir string, window int, now time.Time) FreshnessSummary`
  - Helper to parse `last_verified` from a YAML file (role) and from Markdown frontmatter (skill)
- New file `internal/health/freshness_check_test.go`: unit tests covering:
  - Role 45 days old with 30-day window → stale, message includes role name, date, "15 days overdue"
  - Role 10 days old with 30-day window → not flagged
  - Skill 60 days old with 30-day window → stale, 30 days overdue
  - Skill 25 days old → not flagged
  - Role/skill with no `last_verified` → "never verified" warning (distinct from stale)
  - Custom window (60 days): role at 31 days not flagged
  - Summary counts accuracy: 3 roles (2 fresh, 1 stale), 4 skills (3 fresh, 1 never-verified)
  - Malformed `last_verified` (not a valid timestamp) → warning, not crash
  - Empty roles/skills directories → OK result, no issues
- Modified `internal/mcp/health_tool.go`: wire `CheckRoleFreshness` and `CheckSkillFreshness` into `Phase3HealthChecker` (or create a new checker), reading staleness window from `config.Config.Freshness.StalenessWindowDays`. Add summary counts to the health report output.

**Dependencies:** Task 1 (staleness configuration and core detection logic) must complete first — this task calls `ClassifyFreshness` and reads `FreshnessConfig` from config.

**Interface contract with Task 1:** This task calls:
- `ClassifyFreshness(lastVerified time.Time, isZero bool, window int, now time.Time) FreshnessStatus`
- `ValidateStalenessWindow(days int) error`
- Reads `config.Config.Freshness.StalenessWindowDays` (type `int`, default `30`)

**Interface contract with Task 3:** The `last_verified` parsing helpers created here (for role YAML and skill frontmatter) must be reusable by Task 3 for reading freshness during assembly. Export them as:
- `ParseRoleLastVerified(path string) (time.Time, error)` — returns zero time if field absent
- `ParseSkillLastVerified(skillDir string) (time.Time, error)` — returns zero time if field absent

---

### Task 3: Staleness Warnings in Assembled Context Metadata

**Objective:** Extend the context assembly pipeline to check whether the resolved role and any referenced skills are stale or never-verified, and attach metadata warnings to the assembled output. Stale content is NOT blocked — assembly proceeds normally with warnings attached.

**Specification references:** FR-008, FR-009, NFR-001, NFR-003

**Input context:**
- `internal/mcp/assembly.go` — `assembledContext` struct, `asmInput` struct, `assembleContext()` function — the shared assembly pipeline
- `internal/mcp/handoff_tool.go` — `renderHandoffPrompt()`, response JSON construction — where warnings surface in handoff output
- `internal/mcp/next_tool.go` (if it exists) — where warnings surface in next output
- `internal/context/profile.go` — `ProfileStore`, `Profile` struct — role loading path
- Specification §FR-008: warning must identify which file is stale, its `last_verified` date or "never"
- Specification §FR-009: assembly must NOT refuse; content identical whether fresh or stale; only metadata differs
- Specification §NFR-001: staleness check must add < 10ms (reads already-loaded fields, compares to `time.Now()`)

**Output artifacts:**
- Modified `internal/mcp/assembly.go`:
  - New field on `assembledContext`: `stalenessWarnings []asmStalenessWarning`
  - New type `asmStalenessWarning` with fields: `FileType string` ("role" or "skill"), `Name string`, `LastVerified string` (ISO 8601 or "never"), `DaysOverdue int`
  - New field on `asmInput`: `stalenessWindowDays int` (read from config by caller)
  - Logic in `assembleContext()` after profile resolution: check role's `last_verified` using `ParseRoleLastVerified` and `ClassifyFreshness`; for each skill referenced by the profile, check `ParseSkillLastVerified` and `ClassifyFreshness`. Append warnings to `actx.stalenessWarnings`.
- Modified `internal/mcp/handoff_tool.go`:
  - Pass `stalenessWindowDays` from config into `asmInput`
  - In `renderHandoffPrompt()`: if `actx.stalenessWarnings` is non-empty, render a `### Staleness Warnings` section listing each warning
  - In response JSON `context_metadata`: include `staleness_warnings` array
- Modified `internal/mcp/assembly_test.go` or new `internal/mcp/assembly_freshness_test.go`:
  - Test: stale role (45 days, 30-day window) → warning in `assembledContext.stalenessWarnings` with role name and date
  - Test: never-verified skill → warning with "never" as last-verified
  - Test: fresh role + fresh skill → empty `stalenessWarnings`
  - Test: stale role + fresh skill → warning for role only
  - Test: assembly still succeeds (returns non-nil result) with stale/never-verified inputs
  - Test: assembled content sections are identical regardless of staleness (only metadata differs)
- Benchmark test (can be in the same test file): verify staleness check adds < 10ms to assembly

**Dependencies:** Task 1 (core detection logic) and Task 2 (parsing helpers `ParseRoleLastVerified`, `ParseSkillLastVerified`) must complete first.

**Interface contract with Task 2:** This task calls:
- `ParseRoleLastVerified(path string) (time.Time, error)` from `internal/health/freshness_check.go`
- `ParseSkillLastVerified(skillDir string) (time.Time, error)` from `internal/health/freshness_check.go`

**Interface contract with Task 1:** This task calls:
- `ClassifyFreshness(lastVerified time.Time, isZero bool, window int, now time.Time) FreshnessStatus`

---

### Task 4: Refresh Mechanism for Roles and Skills

**Objective:** Implement a mechanism to update the `last_verified` field on a role YAML file or skill SKILL.md frontmatter to the current UTC timestamp. The write must be atomic (write-to-temp-then-rename) and must not alter any other fields or content. Expose the mechanism through the `profile` MCP tool (new `refresh` action).

**Specification references:** FR-010, FR-011, FR-012, NFR-002, NFR-004

**Input context:**
- `internal/context/profile.go` — `Profile` struct, `ProfileStore` — role file structure and loading
- `internal/mcp/profile_tool.go` — `profileTool()`, action dispatch — where the refresh action is added
- `internal/mcp/doc_tool.go` — `docRefreshAction()` — pattern for the refresh action (conceptual alignment per FR-012)
- `internal/service/documents.go` — `RefreshContentHash()` — existing refresh implementation as a model
- Specification §FR-010: write ISO 8601 to role YAML without altering other fields; add field if absent
- Specification §FR-011: write ISO 8601 to skill frontmatter without altering Markdown content or other frontmatter; add field if absent
- Specification §FR-012: same conceptual model as `doc(action: "refresh")` — records human/agent review confirmation
- Specification §NFR-002: atomic write — temp file + rename; failed write preserves original
- Specification §NFR-004: UTC, second precision, `YYYY-MM-DDTHH:MM:SSZ` format

**Output artifacts:**
- New file `internal/context/refresh.go`:
  - `RefreshRoleLastVerified(path string, now time.Time) error` — reads role YAML, sets/updates `last_verified`, writes atomically
  - `RefreshSkillLastVerified(skillDir string, now time.Time) error` — reads skill SKILL.md, parses frontmatter, sets/updates `last_verified` in frontmatter, writes atomically preserving Markdown content
  - Internal helper `atomicWriteFile(path string, data []byte) error` — write to temp in same dir, then `os.Rename`
- New file `internal/context/refresh_test.go`:
  - Test: refresh role with existing `last_verified` → updated to current time, other fields unchanged
  - Test: refresh role without `last_verified` → field added, other fields unchanged
  - Test: refresh skill with existing `last_verified` → updated, Markdown content unchanged, other frontmatter unchanged
  - Test: refresh skill without `last_verified` → field added, content unchanged
  - Test: timestamp format matches `YYYY-MM-DDTHH:MM:SSZ` pattern and is UTC
  - Test: simulated write failure (read-only directory) → original file preserved
  - Test: timestamp within 1 second of provided `now` value
- Modified `internal/mcp/profile_tool.go`:
  - New action `refresh` in the `profile` tool action dispatch
  - Parameters: `id` (role name) or `skill` (skill name) — exactly one required
  - Calls `RefreshRoleLastVerified` or `RefreshSkillLastVerified`
  - Response includes the new `last_verified` value, confirmation message aligned with FR-012 wording ("Content reviewed and confirmed current")
  - Tool description updated to list `refresh` as a valid action
- New file `internal/mcp/profile_refresh_test.go` (or added to `internal/mcp/profile_tool_test.go`):
  - Test: refresh action for role → success, `last_verified` set
  - Test: refresh action for skill → success, `last_verified` set
  - Test: refresh action with neither id nor skill → error
  - Test: refresh action for nonexistent role → error

**Dependencies:** None for the core refresh logic. However, the parsing helpers from Task 2 (`ParseRoleLastVerified`, `ParseSkillLastVerified`) may be useful for verification in tests. Task 4 can run in parallel with Tasks 2 and 3 since it writes files rather than reading classification results. The only soft dependency is on the `last_verified` field format agreed in the interface contracts.

---

## 3. Dependency Graph

```
Task 1: Config + Core Detection Logic
  │
  ├──→ Task 2: Health Check Category (depends on Task 1)
  │       │
  │       └──→ Task 3: Assembly Staleness Warnings (depends on Task 1 + Task 2)
  │
  └──→ Task 4: Refresh Mechanism (no hard dependency — parallel with Task 2/3)
```

**Parallelism opportunities:**
- **Task 1** runs first (no dependencies).
- **Task 2** and **Task 4** can run in parallel after Task 1 completes. Task 4 has no hard dependency on Task 1's runtime output — it writes timestamps rather than classifying them — but shares the timestamp format contract.
- **Task 3** must wait for both Task 1 and Task 2 (needs core detection functions and parsing helpers).

**Execution order (2 waves):**

| Wave | Tasks | Notes |
|------|-------|-------|
| 1    | Task 1 | Foundation: config + detection logic |
| 2    | Task 2, Task 4 (parallel) | Health checks + refresh mechanism |
| 3    | Task 3 | Assembly integration (needs Task 1 + Task 2 outputs) |

---

## 4. Interface Contracts

### Contract A: Staleness Classification (Task 1 → Tasks 2, 3)

```go
// Package: internal/health

// FreshnessStatus classifies a last_verified timestamp.
type FreshnessStatus int

const (
    StatusFresh         FreshnessStatus = iota
    StatusStale
    StatusNeverVerified
)

// ClassifyFreshness returns the staleness classification for a given timestamp.
// If isZero is true, the file has no last_verified field (never verified).
// window is the staleness threshold in days (must be positive).
func ClassifyFreshness(lastVerified time.Time, isZero bool, window int, now time.Time) FreshnessStatus

// StalenessDetail provides classification with context for health messages.
type StalenessDetail struct {
    Status      FreshnessStatus
    DaysOverdue int       // 0 if fresh or never-verified
    LastVerified time.Time // zero if never-verified
}

// ValidateStalenessWindow returns an error if days is not positive.
func ValidateStalenessWindow(days int) error
```

### Contract B: Config Field (Task 1 → Tasks 2, 3, 4)

```go
// Package: internal/config

type FreshnessConfig struct {
    StalenessWindowDays int `yaml:"staleness_window_days,omitempty"`
}

// Added to Config struct:
// Freshness FreshnessConfig `yaml:"freshness,omitempty"`

// DefaultConfig sets Freshness.StalenessWindowDays = 30
```

### Contract C: Parsing Helpers (Task 2 → Task 3)

```go
// Package: internal/health

// ParseRoleLastVerified reads the last_verified field from a role YAML file.
// Returns zero time if the field is absent. Returns error only for I/O or parse failures.
func ParseRoleLastVerified(path string) (time.Time, error)

// ParseSkillLastVerified reads the last_verified field from a skill's SKILL.md frontmatter.
// skillDir is the path to the skill directory (e.g., .kbz/skills/my-skill).
// Returns zero time if the field is absent.
func ParseSkillLastVerified(skillDir string) (time.Time, error)
```

### Contract D: Assembly Staleness Warnings (Task 3, internal)

```go
// Package: internal/mcp

type asmStalenessWarning struct {
    FileType    string // "role" or "skill"
    Name        string // role ID or skill directory name
    LastVerified string // ISO 8601 timestamp or "never"
    DaysOverdue int    // 0 if never-verified
}

// Added to assembledContext:
// stalenessWarnings []asmStalenessWarning

// Added to asmInput:
// stalenessWindowDays int
```

### Contract E: Timestamp Format (all tasks)

All `last_verified` values written or read use ISO 8601 UTC format: `2025-07-30T14:30:00Z`. The Go format string is `time.RFC3339` (which produces second-precision UTC when the input is in UTC). Sub-second precision is accepted on read but not produced on write.

---

## 5. Traceability Matrix

| Requirement | Task(s) | Verification |
|-------------|---------|--------------|
| FR-001: Role `last_verified` field | Task 2 (parse), Task 4 (write) | Unit test: role YAML with/without field parses correctly |
| FR-002: Skill `last_verified` field | Task 2 (parse), Task 4 (write) | Unit test: skill frontmatter with/without field parses correctly |
| FR-003: Stale role detection | Task 1 (classify), Task 2 (health check) | Unit test: 45-day role flagged, 10-day role not flagged |
| FR-004: Stale skill detection | Task 1 (classify), Task 2 (health check) | Unit test: 60-day skill flagged, 25-day skill not flagged |
| FR-005: Never-verified detection | Task 1 (classify), Task 2 (health check) | Unit test: missing field → "never verified" warning |
| FR-006: Configurable staleness window | Task 1 (config + validation) | Unit test: custom 60-day window, default 30-day, reject 0/negative |
| FR-007: Health report freshness section | Task 2 (category) | Integration test: health report contains freshness category |
| FR-008: Assembly staleness warning | Task 3 (assembly) | Integration test: stale role → metadata warning; fresh → no warning |
| FR-009: Stale files don't block assembly | Task 3 (assembly) | Integration test: handoff with stale role returns context, not error |
| FR-010: Refresh mechanism for roles | Task 4 (refresh) | Unit test: refresh sets timestamp, other fields unchanged |
| FR-011: Refresh mechanism for skills | Task 4 (refresh) | Unit test: refresh sets timestamp, content/frontmatter unchanged |
| FR-012: Conceptual alignment with doc refresh | Task 4 (MCP action) | Code review: description says "confirmed current" |
| FR-013: Health summary counts | Task 2 (summary) | Unit test: counts match actual file states |
| FR-014: Staleness window source | Task 1 (config) | Unit test: absent config → 30 days; present config → custom value |
| NFR-001: Zero assembly performance impact | Task 3 (benchmark) | Benchmark test: < 10ms overhead |
| NFR-002: Refresh atomicity | Task 4 (atomic write) | Unit test: simulated failure preserves original |
| NFR-003: Backward compatibility | Task 1 (classify), Task 2 (parse), Task 3 (assembly) | Integration test: files without field load without error |
| NFR-004: Timestamp precision | Task 4 (write) | Unit test: output matches `YYYY-MM-DDTHH:MM:SSZ` |