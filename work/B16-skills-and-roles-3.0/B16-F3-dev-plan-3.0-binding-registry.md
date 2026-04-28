# Implementation Plan: Binding Registry (Kanbanzai 3.0)

**Specification:** `work/spec/3.0-binding-registry.md`
**Feature:** FEAT-01KN5-88PDPE8V (binding-registry)
**Design reference:** `work/design/skills-system-redesign-v2.md` §3.3

---

## Overview

This plan decomposes the binding registry specification into assignable tasks for AI agents. The binding registry creates the declarative mapping from workflow stages to roles, skills, orchestration patterns, prerequisites, and constraints via a single `.kbz/stage-bindings.yaml` file. It is the decision table that orchestrators consult to determine who does what at each stage.

The work divides into four layers: model definition (Go structs for all binding fields including nested structures), YAML loading with duplicate key detection, validation (stage validity, field rules, cross-references), and the stage lookup API. Validation is the heaviest layer given the number of nested structures (`prerequisites`, `sub_agents`, `document_template`) and cross-cutting rules (orchestration↔sub_agents consistency, role fallback).

### Scope boundaries (from specification)

- **In scope:** File location/format, YAML schema for binding entries (all fields), one-stage-one-binding invariant, stage lookup API, role fallback, validation against role/skill formats, validation against known lifecycle stages, `sub_agents`/`document_template`/`prerequisites` nested structures, multi-error reporting
- **Out of scope:** Prerequisite enforcement at transition time, context assembly, orchestration execution, script execution, MCP tool filtering, binding content, document template enforcement at stage gates

---

## Task Breakdown

### Task 1: Binding model definition

**Objective:** Define the Go structs for the complete binding registry schema — the top-level file structure, individual binding entries, and all nested structures (`prerequisites`, `sub_agents`, `document_template`). Implement field-level validation for each struct that accumulates errors.

**Specification references:** FR-001 (top-level `stage_bindings` mapping), FR-002 (binding entry fields), FR-003 (orchestration enum), FR-004 (roles format), FR-005 (skills format), FR-009 (prerequisites structure), FR-010 (sub_agents structure), FR-011 (document_template structure), FR-013 (max_review_cycles), FR-014 (document_type), FR-015 (multi-error reporting), NFR-002 (strict parsing)

**Input context:**
- `internal/context/profile.go` — `idRegexp` for role ID format validation (FR-004 reuses this: lowercase alphanumeric with hyphens, 2–30 chars)
- Spec §FR-001 for top-level structure: `stage_bindings` mapping with stage name keys
- Spec §FR-002 for complete field list with required/optional designations
- Spec §FR-003 for orchestration enum: `single-agent`, `orchestrator-workers`
- Spec §FR-009 for prerequisites: `documents` (list of `{type, status}`), `tasks` (`min_count` XOR `all_terminal`)
- Spec §FR-010 for sub_agents: `roles`, `skills`, `topology` (only `parallel`), `max_agents`
- Spec §FR-011 for document_template: `required_sections` (non-empty), `cross_references`, `acceptance_criteria_format`
- The project uses `gopkg.in/yaml.v3` with `KnownFields(true)` for strict parsing

**Output artifacts:**
- New file `internal/binding/model.go` containing `BindingFile`, `StageBinding`, `Prerequisites`, `DocumentPrereq`, `TaskPrereq`, `SubAgents`, `DocumentTemplate` structs and `validateBinding` function
- New file `internal/binding/model_test.go` with table-driven tests covering: all required fields present, each required field missing individually, invalid orchestration enum, empty roles/skills lists, invalid role ID format, invalid skill name format, sub_agents with single-agent orchestration (error), sub_agents missing required fields, prerequisites with both min_count and all_terminal (error), document_template with empty required_sections (error), max_review_cycles < 1 (error), empty document_type (error), unknown field rejected

**Dependencies:** None — this is the foundation task.

**Interface contract (shared with Tasks 2, 3, 4, 5):**

```go
// StageBinding represents a single stage's binding configuration.
// Strict parsing: unknown fields are rejected (NFR-002).
type StageBinding struct {
    Description      string            `yaml:"description"`
    Orchestration    string            `yaml:"orchestration"`
    Roles            []string          `yaml:"roles"`
    Skills           []string          `yaml:"skills"`
    HumanGate        bool              `yaml:"human_gate"`
    DocumentType     *string           `yaml:"document_type,omitempty"`
    Prerequisites    *Prerequisites    `yaml:"prerequisites,omitempty"`
    Notes            string            `yaml:"notes,omitempty"`
    EffortBudget     string            `yaml:"effort_budget,omitempty"`
    MaxReviewCycles  *int              `yaml:"max_review_cycles,omitempty"`
    SubAgents        *SubAgents        `yaml:"sub_agents,omitempty"`
    DocumentTemplate *DocumentTemplate `yaml:"document_template,omitempty"`
}

// Prerequisites declares what must be true before entering the stage.
type Prerequisites struct {
    Documents []DocumentPrereq `yaml:"documents,omitempty"`
    Tasks     *TaskPrereq      `yaml:"tasks,omitempty"`
}

// DocumentPrereq is a single document prerequisite declaration.
type DocumentPrereq struct {
    Type   string `yaml:"type"`
    Status string `yaml:"status"`
}

// TaskPrereq declares task completion prerequisites.
// Exactly one of MinCount or AllTerminal may be set, not both.
type TaskPrereq struct {
    MinCount    *int  `yaml:"min_count,omitempty"`
    AllTerminal *bool `yaml:"all_terminal,omitempty"`
}

// SubAgents declares the worker configuration for orchestrator-workers stages.
type SubAgents struct {
    Roles    []string `yaml:"roles"`
    Skills   []string `yaml:"skills"`
    Topology string   `yaml:"topology"`
    MaxAgents *int    `yaml:"max_agents,omitempty"`
}

// DocumentTemplate declares required structure for documents produced in a stage.
type DocumentTemplate struct {
    RequiredSections         []string `yaml:"required_sections"`
    CrossReferences          []string `yaml:"cross_references,omitempty"`
    AcceptanceCriteriaFormat string   `yaml:"acceptance_criteria_format,omitempty"`
}

// BindingFile is the top-level structure of stage-bindings.yaml.
type BindingFile struct {
    StageBindings map[string]*StageBinding `yaml:"stage_bindings"`
}

// validateBinding checks all field-level invariants for a single binding entry
// and accumulates errors. stageName is included in error messages.
func validateBinding(b *StageBinding, stageName string) []error
```

---

### Task 2: YAML loader with duplicate key detection

**Objective:** Implement the loader that reads `.kbz/stage-bindings.yaml`, detects duplicate stage keys (which standard YAML parsers silently accept with last-writer-wins), and decodes the file into the `BindingFile` struct with strict field parsing.

**Specification references:** FR-001 (file location, `stage_bindings` top-level key), FR-007 (one-stage-one-binding, duplicate detection), FR-015 (multi-error reporting), NFR-002 (strict parsing), NFR-003 (standard YAML 1.2)

**Input context:**
- Spec §FR-007: duplicate stage keys must be detected — standard `yaml.Unmarshal` uses last-writer-wins, so the `yaml.Node` API is required for duplicate detection
- Spec dependencies §4: "Duplicate key detection requires using the `yaml.Node` API rather than direct unmarshalling"
- `gopkg.in/yaml.v3` — `yaml.Node` with `Kind == yaml.MappingNode` exposes key-value pairs as alternating entries in `Content`, allowing duplicate key scanning before structured decoding
- Spec §FR-001: missing `stage_bindings` top-level key is an error; `stage_bindings` not being a mapping is an error

**Output artifacts:**
- New file `internal/binding/loader.go` containing `loadBindingFile` function
- New file `internal/binding/loader_test.go` with tests for: valid file loads, missing `stage_bindings` key error, `stage_bindings` is a list (not mapping) error, duplicate stage key detected and reported, unknown top-level key rejected, empty `stage_bindings` mapping loads (zero bindings), file not found error

**Dependencies:** Task 1 (needs `BindingFile`, `StageBinding` structs)

**Interface contract (shared with Tasks 3, 4, 5):**

```go
// loadBindingFile reads and decodes stage-bindings.yaml from the given path.
// It detects duplicate stage keys via the yaml.Node API before structured
// decoding with KnownFields(true). Returns all parse/structural errors accumulated.
func loadBindingFile(path string) (*BindingFile, []error)
```

---

### Task 3: Stage and cross-reference validation

**Objective:** Implement the validation layer that checks stage names against known lifecycle stages, validates all binding entries, enforces the orchestration↔sub_agents consistency rule, and implements the role fallback mechanism for references to non-existent roles.

**Specification references:** FR-006 (stage names must be valid), FR-010 (sub_agents only with orchestrator-workers), FR-012 (role fallback: strip last hyphen segment, log warning), FR-015 (multi-error reporting)

**Input context:**
- `internal/model/entities.go` — `FeatureStatus` constants for lifecycle stages: `designing`, `specifying`, `dev-planning`, `developing`, `reviewing`; plus non-lifecycle stages from spec: `researching`, `documenting`, `plan-reviewing`
- Spec §FR-006: stage keys must be from the known set; unknown stage names are errors
- Spec §FR-010: `sub_agents` present with `orchestration: single-agent` is an error
- Spec §FR-012: role fallback by removing last hyphen-delimited segment (`reviewer-security` → `reviewer`); if neither exists, log warning but do not fail; warning must include role ID and stage name
- The role fallback requires checking whether a role file exists — this function receives a role existence checker (interface or function) so it can be tested without filesystem fixtures

**Output artifacts:**
- New file `internal/binding/validate.go` containing `validateBindingFile` function and `validStages` set
- New file `internal/binding/validate_test.go` with tests for: valid stage names pass, invalid stage name error with valid-stage list in message, sub_agents with single-agent error, sub_agents with orchestrator-workers passes, role exists (no warning), role fallback to parent (warning), role unresolvable (warning but no error), all errors across multiple bindings accumulated in single pass

**Dependencies:** Task 1 (needs model structs and `validateBinding`), Task 2 (needs `loadBindingFile` output)

**Interface contract (shared with Tasks 4, 5):**

```go
// RoleChecker tests whether a role ID has a corresponding role file.
// Implemented by RoleStore.Exists from the role system.
type RoleChecker func(id string) bool

// ValidationResult holds errors (blocking) and warnings (non-blocking) separately.
type ValidationResult struct {
    Errors   []error
    Warnings []string
}

// validStages is the canonical set of stage names accepted by the binding registry.
var validStages map[string]bool

// validateBindingFile checks stage names, cross-references, and consistency rules
// across all binding entries. roleChecker may be nil (skips role fallback checks).
func validateBindingFile(bf *BindingFile, roleChecker RoleChecker) *ValidationResult
```

---

### Task 4: Binding registry and stage lookup API

**Objective:** Implement the `BindingRegistry` type that loads the file, runs all validation, builds an in-memory index for O(1) stage lookup, and exposes the public API for querying bindings by stage name.

**Specification references:** FR-008 (stage lookup function), NFR-001 (< 200ms load), NFR-004 (O(1) lookup after load)

**Input context:**
- `internal/context/profile.go` — `ProfileStore` pattern with `Load` / `LoadAll` for filesystem-backed stores
- Spec §FR-008: lookup by stage name, error if stage not found
- Spec §NFR-004: O(1) average lookup after load — the `BindingFile.StageBindings` is already a `map[string]*StageBinding`, so this is naturally satisfied; the registry wraps it with validation status
- `internal/mcp/server.go` — future integration point (the registry will be constructed in the MCP server setup, similar to how `ProfileStore` is constructed at line ~72)

**Output artifacts:**
- New file `internal/binding/registry.go` containing `BindingRegistry` with `Load`, `Lookup`, `Stages`, and `Warnings` methods
- New file `internal/binding/registry_test.go` with tests for: lookup existing stage returns binding, lookup missing stage returns error with stage name, lookup empty string returns error, `Stages()` returns all stage names, `Warnings()` returns role fallback warnings, load with validation errors returns error

**Dependencies:** Task 1 (model), Task 2 (loader), Task 3 (validation)

**Interface contract (shared with Task 5, future context assembly):**

```go
// BindingRegistry is the loaded and validated binding registry.
// After successful Load, stage lookup is O(1).
type BindingRegistry struct { /* unexported fields */ }

// NewBindingRegistry creates an unloaded registry.
// bindingPath is the path to stage-bindings.yaml.
// roleChecker is optional; if provided, enables role fallback validation.
func NewBindingRegistry(bindingPath string, roleChecker RoleChecker) *BindingRegistry

// Load reads, parses, and validates the binding file.
// Returns an error if there are any validation errors.
// Warnings (e.g., role fallback) are stored and accessible via Warnings().
func (r *BindingRegistry) Load() error

// Lookup returns the binding for the given stage name.
// Returns an error if the stage has no binding.
func (r *BindingRegistry) Lookup(stage string) (*StageBinding, error)

// Stages returns the sorted list of all stage names with bindings.
func (r *BindingRegistry) Stages() []string

// Warnings returns any non-fatal warnings from the last Load.
func (r *BindingRegistry) Warnings() []string
```

---

### Task 5: Integration tests and benchmarks

**Objective:** Write end-to-end tests that verify the complete binding registry pipeline from a realistic `stage-bindings.yaml` fixture through the `BindingRegistry` API, including cross-reference validation with a mock role checker. Add benchmarks for NFR-001.

**Specification references:** NFR-001 (< 200ms load with 15 bindings), NFR-004 (O(1) lookup), all FRs (end-to-end coverage)

**Input context:**
- `internal/testutil/` — shared test helpers
- Spec NFR-001: load benchmark with up to 15 stage bindings
- Spec NFR-004: lookup benchmark confirming O(1) — time does not grow with binding count
- All specification acceptance criteria — this task validates the full pipeline against representative fixtures

**Output artifacts:**
- New file `internal/binding/integration_test.go` — tests with a realistic `stage-bindings.yaml` fixture covering:
  - Complete valid binding file with all 8 feature lifecycle + non-lifecycle stages
  - Binding with all optional fields populated (sub_agents, document_template, prerequisites, effort_budget, max_review_cycles, notes, document_type)
  - Binding with only required fields (minimal valid entry)
  - Duplicate stage key detection across the full file
  - Invalid stage name in otherwise valid file
  - Role fallback: mock roleChecker returns false for `reviewer-security`, true for `reviewer`
  - Prerequisites with mutually exclusive `min_count` and `all_terminal`
  - Sub_agents with single-agent orchestration (cross-field error)
  - Multiple errors across multiple stages accumulated
  - Stage lookup after successful load (all stages resolve)
  - Stage lookup for missing stage returns error
- Benchmark `BenchmarkLoadBindingFile` with 10 stage bindings (NFR-001)
- Benchmark `BenchmarkLookupStage` confirming constant-time lookup (NFR-004)

**Dependencies:** Tasks 1, 2, 3, 4 (needs the full pipeline assembled)

---

## Dependency Graph

```
Task 1: Binding model definition
  │
  ├──► Task 2: YAML loader with duplicate key detection
  │       │
  │       └──► Task 3: Stage & cross-reference validation
  │               │
  │               └──► Task 4: Binding registry & lookup API
  │                       │
  └─────────────────────► Task 5: Integration tests & benchmarks
```

**Parallelism opportunities:**
- Task 1 is the serial bottleneck — it must complete first
- Tasks 2 and 3 are largely serialized because Task 3 needs the loaded `BindingFile` from Task 2. However, the `validStages` set and `validateBindingFile` logic in Task 3 could be developed in parallel with Task 2 if coding against the Task 1 model structs directly — the dependency is on the data structure, not the loader output
- **Tasks 2 and 3 could execute in parallel** if Task 3 writes validation logic against the model structs (from Task 1) and Task 2 writes the YAML→struct loading; they integrate when Task 4 wires the pipeline together
- Task 4 must wait for Tasks 2 and 3 (it composes their outputs)
- Task 5 must wait for Task 4

**Recommended execution:** 1 → (2 ∥ 3) → 4 → 5

---

## Interface Contracts

### Contract A: Binding model types (Task 1 → Tasks 2, 3, 4, 5)

The `BindingFile`, `StageBinding`, `Prerequisites`, `DocumentPrereq`, `TaskPrereq`, `SubAgents`, and `DocumentTemplate` structs defined in Task 1 are the shared data model. All downstream tasks depend on these types being stable. The struct definitions in the Task 1 interface contract section are authoritative.

### Contract B: Loader output (Task 2 → Tasks 3, 4)

The `loadBindingFile` function returns a `*BindingFile` and accumulated parse errors. Task 3's validation operates on the loaded `BindingFile`. Task 4's registry calls the loader and then passes the result to validation. The function signature in the Task 2 interface contract section is authoritative.

### Contract C: Validation API (Task 3 → Task 4)

The `validateBindingFile` function and `ValidationResult` type are called by the registry (Task 4) after loading. The `RoleChecker` function type allows the registry to inject role existence checking without a direct dependency on the role system's `RoleStore`. The `validStages` set is also used by the skill system's stage validation (shared canonical source). The signatures in the Task 3 interface contract section are authoritative.

### Contract D: Registry API (Task 4 → Task 5, future context assembly)

The `BindingRegistry` with `Load`, `Lookup`, `Stages`, and `Warnings` methods is the public API consumed by integration tests, the MCP server setup, and the future context assembly pipeline. The signatures in the Task 4 interface contract section are authoritative.

### Cross-feature contract: RoleChecker

The `RoleChecker` function type (`func(id string) bool`) is the interface between the binding registry and the role system. The role system's `RoleStore.Exists` method (defined in the role system plan, Task 2) satisfies this contract. The binding registry must not import `RoleStore` directly — it accepts the function via dependency injection.

---

## Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | Task 2 | File location, `stage_bindings` top-level mapping |
| FR-002 | Task 1 | Binding entry field definitions, required/optional |
| FR-003 | Task 1 | Orchestration enum: `single-agent`, `orchestrator-workers` |
| FR-004 | Task 1 | Roles list: non-empty, role ID format |
| FR-005 | Task 1 | Skills list: non-empty, skill name format |
| FR-006 | Task 3 | Stage names must be from the valid set |
| FR-007 | Task 2 | Duplicate key detection via `yaml.Node` API |
| FR-008 | Task 4 | Stage lookup function, error on missing stage |
| FR-009 | Task 1 | Prerequisites structure: documents, tasks, min_count XOR all_terminal |
| FR-010 | Task 1, 3 | Sub_agents structure (Task 1), orchestration consistency check (Task 3) |
| FR-011 | Task 1 | Document_template structure: required_sections non-empty |
| FR-012 | Task 3 | Role fallback: strip last hyphen segment, log warning |
| FR-013 | Task 1 | max_review_cycles: positive integer ≥ 1 |
| FR-014 | Task 1 | document_type: non-empty when present |
| FR-015 | Task 1, 2, 3 | Multi-error reporting throughout the pipeline |
| NFR-001 | Task 5 | Benchmark: full load < 200ms with 15 bindings |
| NFR-002 | Task 1, 2 | Strict parsing: unknown fields rejected |
| NFR-003 | Task 1 | Standard YAML 1.2 only |
| NFR-004 | Task 4, 5 | O(1) lookup via map index; benchmark confirmation |