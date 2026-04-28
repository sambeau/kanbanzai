# Implementation Plan: Role System (Kanbanzai 3.0)

**Specification:** `work/spec/3.0-role-system.md`
**Feature:** FEAT-01KN5-88PCVN4Y (role-system)
**Design reference:** `work/design/skills-system-redesign-v2.md` §3.1, §4

---

## Overview

This plan decomposes the role system specification into assignable tasks for AI agents. The role system extends the existing `internal/context/` package to support a new role YAML schema with `identity`, `vocabulary`, `anti_patterns`, `tools`, and `inherits` fields. It replaces the current profile schema fields (`description`, `packages`, `conventions`, `architecture`) while maintaining backward compatibility during migration.

The work divides into four layers: model definition, storage/loading, inheritance resolution, and MCP tool integration. Validation is woven into each layer rather than isolated, following the existing pattern in `profile.go`.

### Scope boundaries (from specification)

- **In scope:** YAML schema, storage, inheritance resolution, validation, MCP tool extension, backward compatibility
- **Out of scope:** Context assembly, role content/taxonomy, MCP tool filtering at runtime, skill system, binding registry, migration tooling

---

## Task Breakdown

### Task 1: Role model and validation

**Objective:** Define the Go structs for the new role schema and implement multi-error validation for all field-level rules.

**Specification references:** FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-013, NFR-002, NFR-003

**Input context:**
- `internal/context/profile.go` — existing `Profile` struct, `idRegexp`, `validateProfile` function
- Spec §FR-002 for the field list: `id`, `inherits`, `identity`, `vocabulary`, `anti_patterns`, `tools`
- Spec §FR-004 for the 50-token identity limit (whitespace-delimited word count)
- Spec §FR-006 for anti-pattern structure: `name`, `detect`, `because`, `resolve`
- Spec §FR-013 for multi-error accumulation

**Output artifacts:**
- New file `internal/context/role.go` containing `Role`, `ResolvedRole`, `AntiPattern` structs and `validateRole` function
- New file `internal/context/role_test.go` with table-driven tests covering all validation rules
- The `Role` struct must use `yaml:"..." json:"..."` tags and strict parsing (`KnownFields(true)` via decoder)

**Dependencies:** None — this is the foundation task.

**Interface contract (shared with Tasks 2, 3, 4):**

```go
// AntiPattern represents a single anti-pattern entry in a role definition.
type AntiPattern struct {
    Name    string `yaml:"name"    json:"name"`
    Detect  string `yaml:"detect"  json:"detect"`
    Because string `yaml:"because" json:"because"`
    Resolve string `yaml:"resolve" json:"resolve"`
}

// Role is a role definition as loaded from a YAML file.
// Strict parsing: unknown fields are rejected (NFR-002).
type Role struct {
    ID           string        `yaml:"id"`
    Inherits     string        `yaml:"inherits,omitempty"`
    Identity     string        `yaml:"identity"`
    Vocabulary   []string      `yaml:"vocabulary"`
    AntiPatterns []AntiPattern `yaml:"anti_patterns,omitempty"`
    Tools        []string      `yaml:"tools,omitempty"`
}

// ResolvedRole is the result of walking the inheritance chain.
type ResolvedRole struct {
    ID           string
    Identity     string        // always from leaf role (FR-010)
    Vocabulary   []string      // parent ++ child concatenation (FR-010)
    AntiPatterns []AntiPattern // parent ++ child concatenation (FR-010)
    Tools        []string      // union, no duplicates (FR-010)
}

// validateRole checks all field-level invariants and accumulates errors.
// Returns nil if valid, or an error containing all validation failures.
func validateRole(r *Role, expectedID string) error
```

---

### Task 2: Role store and file loading

**Objective:** Implement the `RoleStore` that reads role YAML files from `.kbz/roles/`, with backward-compatible fallback to `.kbz/context/roles/`.

**Specification references:** FR-001, FR-008, FR-011, FR-013, NFR-004

**Input context:**
- `internal/context/profile.go` — existing `ProfileStore` with `Load`, `LoadAll` methods (use as pattern)
- `internal/core/` — `core.InstanceRootDir` for `.kbz/` path resolution
- Spec §NFR-004 for dual-location precedence: `.kbz/roles/` wins over `.kbz/context/roles/`
- Spec §FR-001 for id-filename match requirement
- Spec §FR-013 for multi-error reporting

**Output artifacts:**
- New file `internal/context/role_store.go` containing `RoleStore` with `Load`, `LoadAll`, and `Exists` methods
- New file `internal/context/role_store_test.go` with tests using `t.TempDir()` for filesystem fixtures
- Tests must cover: normal load, id-filename mismatch, missing file, fallback from old location, new location precedence, directory-not-exist returns empty list

**Dependencies:** Task 1 (needs `Role` struct and `validateRole`)

**Interface contract (shared with Tasks 3, 4):**

```go
// RoleStore reads role definitions from the filesystem.
// It checks .kbz/roles/ first, falling back to .kbz/context/roles/ (NFR-004).
type RoleStore struct { /* unexported fields */ }

// NewRoleStore creates a RoleStore. newRoot is .kbz/roles/, legacyRoot is
// .kbz/context/roles/. Either path may not exist on disk.
func NewRoleStore(newRoot, legacyRoot string) *RoleStore

// Load reads and validates a single role by ID.
func (s *RoleStore) Load(id string) (*Role, error)

// LoadAll reads and validates all roles from both locations (new takes precedence).
func (s *RoleStore) LoadAll() ([]*Role, error)

// Exists returns true if a role file exists for the given ID in either location.
func (s *RoleStore) Exists(id string) bool
```

---

### Task 3: Inheritance resolution for roles

**Objective:** Implement role-specific inheritance resolution with the new merge semantics: vocabulary concatenation, anti-pattern concatenation, tools union, identity not inherited.

**Specification references:** FR-008, FR-009, FR-010, NFR-001

**Input context:**
- `internal/context/resolve.go` — existing `ResolveChain` (cycle detection, chain walking) and `ResolveProfile` (leaf-replaces-parent semantics)
- The new role resolution has **different** merge semantics from the existing profile resolution: concatenation for lists instead of replacement, union for tools
- Spec §FR-009 for cycle detection (can reuse the `visited` map pattern from `ResolveChain`)
- Spec §FR-010 for exact merge rules per field

**Output artifacts:**
- New file `internal/context/role_resolve.go` containing `ResolveRoleChain` and `ResolveRole` functions
- New file `internal/context/role_resolve_test.go` with tests for: single role (no inheritance), two-level chain, three-level chain, cycle detection (direct and transitive), vocabulary concatenation order, anti-pattern concatenation order, tools union deduplication, identity always from leaf

**Dependencies:** Task 1 (needs `Role`, `ResolvedRole` structs), Task 2 (needs `RoleStore.Load`)

**Interface contract (shared with Task 4):**

```go
// ResolveRoleChain returns the inheritance chain from root to leaf (leaf is last).
// Returns an error if any inherits reference is missing or a cycle is detected.
func ResolveRoleChain(store *RoleStore, id string) ([]*Role, error)

// ResolveRole walks the inheritance chain and returns the fully resolved role.
// Merge semantics per FR-010:
//   - vocabulary: parent ++ child (concatenation)
//   - anti_patterns: parent ++ child (concatenation)
//   - tools: union (no duplicates)
//   - identity: leaf only (not inherited)
//   - id: leaf only
func ResolveRole(store *RoleStore, id string) (*ResolvedRole, error)
```

---

### Task 4: MCP profile tool update

**Objective:** Update the `profile` MCP tool to load roles from the new `RoleStore` and return the new schema fields (`identity`, `vocabulary`, `anti_patterns`, `tools`) instead of the old fields (`description`, `packages`, `conventions`, `architecture`).

**Specification references:** FR-011, FR-012

**Input context:**
- `internal/mcp/profile_tool.go` — existing tool handler with `profileListAction` and `profileGetAction`
- `internal/mcp/server.go` — where `ProfileStore` is constructed and passed to `ProfileTool` (line ~72: `profileRoot` and `profileStore` construction)
- The tool must continue to accept the same MCP parameters (`action`, `id`, `resolved`)
- `list` must return `id`, `inherits`, `identity` per entry (FR-011)
- `get` with `resolved: true` must return full merged role; `resolved: false` must return raw YAML definition (FR-012)

**Output artifacts:**
- Modified `internal/mcp/profile_tool.go` — update handler to use `RoleStore` instead of `ProfileStore`, update response maps
- Modified `internal/mcp/server.go` — construct `RoleStore` with both new and legacy roots, pass to `ProfileTool`
- Modified or new `internal/mcp/profile_tool_test.go` — tests for list and get actions with role schema responses

**Dependencies:** Task 2 (needs `RoleStore`), Task 3 (needs `ResolveRole`)

---

### Task 5: Integration tests and backward compatibility verification

**Objective:** Write end-to-end tests that verify the complete role loading pipeline from YAML files through MCP tool responses, including backward compatibility with the legacy `.kbz/context/roles/` location.

**Specification references:** NFR-001, NFR-004 (all backward compatibility acceptance criteria)

**Input context:**
- `internal/context/profile_test.go` and `internal/context/resolve_test.go` — existing test patterns
- `internal/mcp/profile_tool_test.go` — existing MCP tool test patterns (if present)
- `internal/testutil/` — shared test helpers
- NFR-004 acceptance criteria: new location precedence, legacy fallback, both-exist behavior

**Output artifacts:**
- New file `internal/context/role_integration_test.go` — tests that create temporary `.kbz/roles/` and `.kbz/context/roles/` directories, write role YAML files, and verify the full load→resolve→format pipeline
- Benchmark test for NFR-001: loading and resolving a 3-level chain under 50ms

**Dependencies:** Tasks 1, 2, 3 (needs the full role pipeline in place)

---

## Dependency Graph

```
Task 1: Role model & validation
  │
  ├──► Task 2: Role store & file loading
  │       │
  │       ├──► Task 3: Inheritance resolution
  │       │       │
  │       │       └──► Task 4: MCP profile tool update
  │       │               │
  │       └───────────────┤
  │                       ▼
  └──────────────► Task 5: Integration tests & backward compat
```

**Parallelism opportunities:**
- Task 1 is the serial bottleneck — it must complete first
- Tasks 2 and 3 could be developed in parallel against the Task 1 interface contract if Task 2's `Load` method is stubbed, but the dependency chain (Task 3 needs `RoleStore.Load`) makes serialization simpler
- The recommended execution order is: **1 → 2 → 3 → 4 → 5**
- Task 5 can begin as soon as Tasks 1–3 are complete (it does not depend on Task 4 for the core pipeline tests)

---

## Interface Contracts

### Contract A: Role struct (Task 1 → Tasks 2, 3, 4)

The `Role`, `ResolvedRole`, and `AntiPattern` structs defined in Task 1 are the shared data model. All downstream tasks depend on these types being stable. The struct definitions in the Task 1 interface contract section are authoritative.

### Contract B: RoleStore API (Task 2 → Tasks 3, 4)

`ResolveRoleChain` and `ResolveRole` (Task 3) call `RoleStore.Load` (Task 2). The MCP tool (Task 4) calls both `RoleStore.LoadAll` (for list) and `ResolveRole`/`RoleStore.Load` (for get). The method signatures in the Task 2 interface contract section are authoritative.

### Contract C: Resolve API (Task 3 → Task 4)

The MCP tool (Task 4) calls `ResolveRole` to produce the merged view. The function signature in the Task 3 interface contract section is authoritative.

---

## Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | Task 2 | ID-filename match, `.kbz/roles/` location |
| FR-002 | Task 1 | Schema definition, strict parsing |
| FR-003 | Task 1 | ID format validation (reuses `idRegexp`) |
| FR-004 | Task 1 | Identity non-empty, 50-token limit |
| FR-005 | Task 1 | Vocabulary non-empty list validation |
| FR-006 | Task 1 | Anti-pattern structure: four required fields |
| FR-007 | Task 1 | Tools list, duplicate detection |
| FR-008 | Task 2, 3 | Inherits reference must resolve |
| FR-009 | Task 3 | Cycle detection |
| FR-010 | Task 3 | Merge semantics: concat, union, identity not inherited |
| FR-011 | Task 4 | `profile(action: "list")` returns new fields |
| FR-012 | Task 4 | `profile(action: "get")` resolved vs raw |
| FR-013 | Task 1, 2 | Multi-error accumulation |
| NFR-001 | Task 5 | Benchmark: 3-level chain < 50ms |
| NFR-002 | Task 1 | Strict parsing, unknown fields rejected |
| NFR-003 | Task 1 | Standard YAML 1.2 only |
| NFR-004 | Task 2, 5 | Backward compat: dual-location, new wins |