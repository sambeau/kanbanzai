# Implementation Plan: Skill System (Kanbanzai 3.0)

**Specification:** `work/spec/3.0-skill-system.md`
**Feature:** FEAT-01KN5-88PDBW85 (skill-system)
**Design reference:** `work/design/skills-system-redesign-v2.md` §3.2, §5

---

## Overview

This plan decomposes the skill system specification into assignable tasks for AI agents. The skill system creates a new loading and validation pipeline for `.kbz/skills/{skill-name}/SKILL.md` files — Markdown documents with YAML frontmatter that encode procedures, output formats, and evaluation criteria for specific types of work.

The work divides into three layers: frontmatter schema and validation, body section parsing and validation, and the directory-level loader that ties everything together (references, scripts, size limits). Each layer builds on the previous one, but frontmatter and body parsing are internally independent and can be developed in parallel once a thin shared model is established.

### Scope boundaries (from specification)

- **In scope:** Directory layout, SKILL.md frontmatter schema, body section names and ordering, dual-register description, constraint level enum, reference/script file conventions, 500-line size limit, evaluation criteria format, loader, validation
- **Out of scope:** Skill content/catalog, context assembly, script execution runtime/sandboxing, skill selection/trigger matching, binding registry integration, role system schema, evaluation pass execution

---

## Task Breakdown

### Task 1: Skill model and frontmatter validation

**Objective:** Define the Go structs for the skill frontmatter schema and implement multi-error validation for all frontmatter field-level rules. This includes the dual-register `Description` type, constraint level enum, and format validation for `triggers`, `roles`, `stage`, and `name`.

**Specification references:** FR-001 (name format), FR-002 (frontmatter fields), FR-003 (description dual-register), FR-004 (triggers), FR-005 (roles format), FR-006 (stage), FR-007 (constraint_level enum), FR-017 (multi-error reporting), NFR-003 (strict parsing)

**Input context:**
- `internal/context/profile.go` — `idRegexp` pattern for role ID validation (FR-005 reuses this format for role references)
- Spec §FR-001 for skill name format: lowercase alphanumeric with hyphens, 2–40 characters
- Spec §FR-002 for the full field list
- Spec §FR-006 for valid stage names — derived from feature lifecycle stages in `internal/model/entities.go` (`designing`, `specifying`, `dev-planning`, `developing`, `reviewing`) plus non-lifecycle stages (`researching`, `documenting`, `plan-reviewing`)
- Spec §FR-007 for constraint level enum: `low`, `medium`, `high`
- The project uses `gopkg.in/yaml.v3` with `KnownFields(true)` for strict parsing

**Output artifacts:**
- New file `internal/skill/model.go` containing `Skill`, `SkillFrontmatter`, `SkillDescription`, `BodySection` structs and `validateFrontmatter` function
- New file `internal/skill/model_test.go` with table-driven tests for all frontmatter validation rules
- Tests must cover: all required fields present, each required field missing individually, invalid name format, empty triggers, invalid role ID format, invalid stage, invalid constraint level, unknown frontmatter field rejected, description as plain string rejected, empty expert/natural

**Dependencies:** None — this is the foundation task.

**Interface contract (shared with Tasks 2, 3, 4, 5):**

```go
// SkillDescription is the dual-register description (expert + natural).
type SkillDescription struct {
    Expert  string `yaml:"expert"  json:"expert"`
    Natural string `yaml:"natural" json:"natural"`
}

// SkillFrontmatter is the YAML frontmatter parsed from SKILL.md.
// Strict parsing: unknown fields are rejected (NFR-003).
type SkillFrontmatter struct {
    Name            string           `yaml:"name"`
    Description     SkillDescription `yaml:"description"`
    Triggers        []string         `yaml:"triggers"`
    Roles           []string         `yaml:"roles"`
    Stage           string           `yaml:"stage"`
    ConstraintLevel string           `yaml:"constraint_level"`
}

// BodySection represents a parsed ## section from the SKILL.md body.
type BodySection struct {
    Heading string // the heading text (e.g., "Vocabulary")
    Content string // everything between this heading and the next ## heading
}

// Skill is the fully parsed and validated representation of a skill directory.
type Skill struct {
    Frontmatter    SkillFrontmatter
    Sections       []BodySection
    ReferencePaths []string // paths relative to the skill directory
    ScriptPaths    []string // paths relative to the skill directory
    SourcePath     string   // absolute path to the SKILL.md file
}

// validStages is the canonical set of stage names accepted by the skill loader.
var validStages map[string]bool

// validateFrontmatter checks all frontmatter field invariants and accumulates errors.
func validateFrontmatter(fm *SkillFrontmatter, expectedName string) []error
```

---

### Task 2: SKILL.md frontmatter parser

**Objective:** Implement the parser that splits a SKILL.md file into its YAML frontmatter block and Markdown body, decodes the frontmatter using strict YAML parsing, and returns both parts for downstream validation.

**Specification references:** FR-002 (frontmatter delimited by `---`), FR-014 (500-line limit), FR-017 (structured representation), NFR-002 (human-readable format), NFR-003 (strict parsing)

**Input context:**
- Spec §FR-002: SKILL.md begins with `---` delimited YAML frontmatter
- Spec §FR-014: 500-line limit on the entire file (including frontmatter)
- Spec dependencies §4: line-by-line scanning for `## ` headings is sufficient for body parsing
- The project uses `gopkg.in/yaml.v3`; strict parsing requires `yaml.NewDecoder` + `decoder.KnownFields(true)`

**Output artifacts:**
- New file `internal/skill/parse.go` containing `parseSKILLMD` function
- New file `internal/skill/parse_test.go` with tests for: valid frontmatter+body split, missing opening `---`, missing closing `---`, 500-line file (passes), 501-line file (error with counts), empty body, frontmatter-only file

**Dependencies:** Task 1 (needs `SkillFrontmatter` struct for YAML decoding)

**Interface contract (shared with Tasks 3, 4):**

```go
// parsedSKILLMD holds the raw parse result before section-level validation.
type parsedSKILLMD struct {
    Frontmatter SkillFrontmatter
    BodyRaw     string // raw Markdown body after the closing ---
    LineCount   int    // total lines in the file
}

// parseSKILLMD reads a SKILL.md file, splits frontmatter from body,
// decodes frontmatter with strict YAML parsing, and enforces the 500-line limit.
// Returns all parse errors accumulated in a single pass.
func parseSKILLMD(data []byte) (*parsedSKILLMD, []error)
```

---

### Task 3: Body section parser and ordering validation

**Objective:** Implement the parser that extracts `##` sections from the Markdown body and validates section presence, ordering (attention-curve sequence), and content rules (non-empty vocabulary, anti-pattern structure, evaluation criteria format, checklist conditional requirement).

**Specification references:** FR-008 (section ordering), FR-009 (required sections), FR-010 (vocabulary non-empty), FR-011 (anti-pattern detect/because), FR-012 (checklist conditional on constraint_level), FR-013 (evaluation criteria with weights), FR-017 (multi-error reporting)

**Input context:**
- Spec §FR-008 for the canonical section order: Vocabulary → Anti-Patterns → Checklist → Procedure → Output Format → Examples → Evaluation Criteria → Questions This Skill Answers
- Spec §FR-009 for required sections: Vocabulary, Anti-Patterns, Procedure, Output Format, Evaluation Criteria, Questions This Skill Answers (Checklist and Examples are optional)
- Spec §FR-011: anti-pattern entries need "Detect" and "BECAUSE" labels (case-insensitive) — validation warnings, not errors
- Spec §FR-012: checklist required when `constraint_level` is `low` or `medium`
- Spec §FR-013: evaluation criteria must have gradable questions with weight values (`required`, `high`, `medium`, `low`)
- Spec dependencies §4: line-by-line scanning for `## ` headings; sub-headings (`###`, `####`) do not affect ordering

**Output artifacts:**
- New file `internal/skill/sections.go` containing `parseSections` and `validateSections` functions
- New file `internal/skill/sections_test.go` with tests for: correct order passes, out-of-order sections error, missing required section errors, unknown heading produces warning, checklist required for low/medium, checklist optional for high, empty vocabulary body error, anti-pattern missing detect/because produces warnings, evaluation criteria with valid weights passes, evaluation criteria empty body errors, sub-headings within sections ignored

**Dependencies:** Task 1 (needs `BodySection` struct and `SkillFrontmatter.ConstraintLevel` for checklist rule)

**Interface contract (shared with Task 4):**

```go
// ValidationMessage is either an error or a warning from section validation.
type ValidationMessage struct {
    Level   string // "error" or "warning"
    Message string
}

// parseSections extracts ## sections from raw Markdown body.
// Returns sections in document order. Sub-headings are included in parent section content.
func parseSections(bodyRaw string) []BodySection

// validateSections checks section presence, ordering, and content rules.
// constraintLevel is needed for the checklist conditional requirement (FR-012).
func validateSections(sections []BodySection, constraintLevel string) []ValidationMessage
```

---

### Task 4: Skill directory loader

**Objective:** Implement the top-level loader that reads a skill directory (`.kbz/skills/{skill-name}/`), parses SKILL.md, discovers `references/` and `scripts/` contents, validates reference constraints, and assembles the full `Skill` struct. Also implement `LoadAll` for listing all skills.

**Specification references:** FR-001 (directory structure, name match), FR-015 (references: markdown-only, one-level-deep, orphan warning), FR-016 (scripts: output-only, paths not content), FR-017 (structured representation, multi-error), NFR-001 (< 100ms single skill), NFR-004 (30 skills listing < 500ms)

**Input context:**
- `internal/context/profile.go` — `ProfileStore` pattern with `Load` / `LoadAll` for filesystem-backed loading
- Spec §FR-001: directory name must match frontmatter `name`
- Spec §FR-015: `references/` contains only `.md` files; reference files must be linked from SKILL.md (one level deep); orphaned files produce warnings; files referencing other reference files is out-of-scope for the loader to detect (content concern)
- Spec §FR-016: `scripts/` contains executable files; loader records paths only, does not load content into context
- Spec §FR-017: all errors reported in a single pass

**Output artifacts:**
- New file `internal/skill/loader.go` containing `SkillStore` with `Load` and `LoadAll` methods
- New file `internal/skill/loader_test.go` with tests using `t.TempDir()` for: valid skill directory loads, name-directory mismatch error, missing SKILL.md error, references with non-markdown file warning, orphaned reference file warning, scripts directory discovered, empty skills directory returns empty list, multiple validation errors accumulated

**Dependencies:** Task 1 (model structs), Task 2 (frontmatter parser), Task 3 (section parser/validator)

**Interface contract (shared with Task 5):**

```go
// SkillStore reads skill definitions from the filesystem.
type SkillStore struct { /* unexported fields */ }

// NewSkillStore creates a SkillStore rooted at the given directory (.kbz/skills/).
func NewSkillStore(root string) *SkillStore

// Load reads, parses, and validates a single skill by name.
// Returns the Skill and any validation warnings. Returns an error if
// the skill has validation errors (warnings alone do not cause an error).
func (s *SkillStore) Load(name string) (*Skill, []ValidationMessage, error)

// LoadAll reads and validates all skills in the root directory.
// Returns all successfully loaded skills and any per-skill errors.
func (s *SkillStore) LoadAll() ([]*Skill, error)
```

---

### Task 5: Integration tests and benchmarks

**Objective:** Write end-to-end tests that verify the complete skill loading pipeline from filesystem fixtures through the `SkillStore` API, including edge cases across all layers. Add benchmarks for NFR-001 and NFR-004.

**Specification references:** NFR-001 (< 100ms single skill), NFR-004 (30 skills < 500ms listing), all FRs (end-to-end coverage)

**Input context:**
- `internal/testutil/` — shared test helpers
- Spec NFR-001: single skill load benchmark (400 lines, 3 references, 2 scripts)
- Spec NFR-004: 30-skill directory listing benchmark
- All specification acceptance criteria — this task validates the full pipeline against representative fixtures

**Output artifacts:**
- New file `internal/skill/integration_test.go` — tests with realistic SKILL.md fixtures covering:
  - Complete valid skill with all optional sections
  - Minimal valid skill (required sections only)
  - Skill at exactly 500 lines (boundary)
  - Skill at 501 lines (boundary error)
  - Skill with constraint_level low missing checklist (error)
  - Skill with constraint_level high missing checklist (ok)
  - Multiple validation errors in one file accumulated
  - Reference and script subdirectory handling
- Benchmark `BenchmarkLoadSingleSkill` with a representative fixture (NFR-001)
- Benchmark `BenchmarkLoadAllSkills` with 30 skill directories (NFR-004)

**Dependencies:** Tasks 1, 2, 3, 4 (needs the full pipeline assembled)

---

## Dependency Graph

```
Task 1: Skill model & frontmatter validation
  │
  ├──► Task 2: Frontmatter parser
  │       │
  ├──► Task 3: Body section parser & ordering validation
  │       │
  │       ├───┐
  │       │   ▼
  │       └──► Task 4: Skill directory loader
  │               │
  └───────────────┤
                  ▼
           Task 5: Integration tests & benchmarks
```

**Parallelism opportunities:**
- Task 1 is the serial bottleneck — it must complete first
- **Tasks 2 and 3 can execute in parallel** after Task 1 completes — they are independent (frontmatter parsing vs body section parsing), connected only by the shared model from Task 1
- Task 4 must wait for both Tasks 2 and 3 (it composes their outputs)
- Task 5 must wait for Task 4

**Recommended execution:** 1 → (2 ∥ 3) → 4 → 5

---

## Interface Contracts

### Contract A: Skill model types (Task 1 → Tasks 2, 3, 4, 5)

The `Skill`, `SkillFrontmatter`, `SkillDescription`, and `BodySection` structs defined in Task 1 are the shared data model. All downstream tasks depend on these types. The struct definitions in the Task 1 interface contract section are authoritative.

### Contract B: Parse output (Task 2 → Task 4)

The `parseSKILLMD` function returns a `parsedSKILLMD` containing the decoded frontmatter and raw body string. Task 4's loader calls this function and feeds the raw body into Task 3's section parser. The function signature in the Task 2 interface contract section is authoritative.

### Contract C: Section validation (Task 3 → Task 4)

The `parseSections` and `validateSections` functions are called by the loader (Task 4) after `parseSKILLMD` splits the file. `validateSections` needs the `constraintLevel` from the parsed frontmatter to enforce the checklist conditional requirement. The `ValidationMessage` type is used throughout the pipeline for warnings that don't block loading. The signatures in the Task 3 interface contract section are authoritative.

### Contract D: SkillStore API (Task 4 → Task 5, future binding registry integration)

The `SkillStore` with `Load` and `LoadAll` methods is the public API consumed by integration tests and future callers. The signatures in the Task 4 interface contract section are authoritative.

---

## Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | Task 1, 4 | Name format validation (Task 1), directory-name match (Task 4) |
| FR-002 | Task 1, 2 | Field definitions (Task 1), frontmatter parsing (Task 2) |
| FR-003 | Task 1 | Description dual-register: expert + natural subfields |
| FR-004 | Task 1 | Triggers: non-empty list of strings |
| FR-005 | Task 1 | Roles: non-empty list, role ID format validation |
| FR-006 | Task 1 | Stage: must be a known lifecycle or non-lifecycle stage |
| FR-007 | Task 1 | Constraint level enum: low, medium, high |
| FR-008 | Task 3 | Section ordering: attention-curve sequence validation |
| FR-009 | Task 3 | Required sections: Vocabulary, Anti-Patterns, Procedure, Output Format, Evaluation Criteria, Questions This Skill Answers |
| FR-010 | Task 3 | Vocabulary section non-empty body |
| FR-011 | Task 3 | Anti-pattern detect/because structure (warnings) |
| FR-012 | Task 3 | Checklist required for low/medium constraint level |
| FR-013 | Task 3 | Evaluation criteria: gradable questions with weights |
| FR-014 | Task 2 | 500-line limit on SKILL.md file |
| FR-015 | Task 4 | References: markdown-only, orphan warnings |
| FR-016 | Task 4 | Scripts: paths only, output not source in context |
| FR-017 | Task 1, 2, 3, 4 | Multi-error reporting throughout the pipeline |
| NFR-001 | Task 5 | Benchmark: single skill < 100ms |
| NFR-002 | Task 1 | Human-readable, standard Markdown + YAML frontmatter |
| NFR-003 | Task 1, 2 | Strict parsing: unknown frontmatter fields rejected |
| NFR-004 | Task 5 | Benchmark: 30 skills listing < 500ms |