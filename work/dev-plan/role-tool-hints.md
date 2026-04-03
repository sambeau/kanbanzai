# Role-Scoped Tool Hints — Development Plan

> Development plan for FEAT-01KNA-11F3BBMP
> Spec: work/spec/role-tool-hints.md
> Design: work/design/role-tool-hints.md

---

## Overview

This feature adds a configuration-driven mechanism for injecting machine-specific tool availability guidance into sub-agent prompts. Tool hints are keyed by role ID, stored in `config.yaml` (project) and `local.yaml` (machine), merged per-key with local overriding project, resolved through role inheritance, and injected into `handoff` prompts, `next` context output, and `health` diagnostics.

The implementation is small (~100 lines of Go excluding tests) and splits cleanly into three tasks: config+resolution logic, injection into prompt paths, and health surfacing.

---

## Task Breakdown

| Task ID | Name | Description | Dependencies | Files |
|---------|------|-------------|--------------|-------|
| TASK-01KNA-KX73CCHS | Tool Hints Config and Resolve | Add `ToolHints` field to both config structs, implement merge function and role inheritance resolution, wire merged map at server startup | None | `internal/config/config.go`, `internal/config/user.go`, `internal/config/tool_hints.go`, `internal/context/tool_hints.go`, `internal/mcp/server.go`, `internal/config/tool_hints_test.go`, `internal/context/tool_hints_test.go` |
| TASK-01KNA-KX76WWZ7 | Tool Hints Injection | Inject resolved hint as `## Available Tools` section in 3.0 pipeline, legacy 2.0 path, and `next` structured output | TASK-01KNA-KX73CCHS | `internal/context/pipeline.go`, `internal/mcp/handoff_tool.go`, `internal/mcp/next_tool.go`, + respective test files |
| TASK-01KNA-KX798DCB | Tool Hints Health Surfacing | Display merged hint map in `health` tool output | TASK-01KNA-KX73CCHS | `internal/mcp/health_tool.go`, `internal/mcp/health_tool_test.go` |

---

## Dependency Graph

```
TASK-01KNA-KX73CCHS (Config and Resolve)
    ├──► TASK-01KNA-KX76WWZ7 (Injection)
    └──► TASK-01KNA-KX798DCB (Health Surfacing)
```

Tasks 2 and 3 are independent of each other and can be worked in parallel once task 1 is complete.

---

## Interface Contracts

### Config structs

```go
// internal/config/config.go
type Config struct {
    // ... existing fields ...
    ToolHints map[string]string `yaml:"tool_hints,omitempty"`
}

// internal/config/user.go
type LocalConfig struct {
    // ... existing fields ...
    ToolHints map[string]string `yaml:"tool_hints,omitempty"`
}
```

### Merge function (`internal/config/tool_hints.go`)

```go
// MergeToolHints returns the effective tool hints map. Local hints override
// project hints on a per-key basis. Either or both inputs may be nil.
func MergeToolHints(project, local map[string]string) map[string]string
```

### Role inheritance resolution (`internal/context/tool_hints.go`)

```go
// ResolveToolHint returns the effective hint for the given role ID, walking
// the inheritance chain if no exact match exists. Returns "" if no hint resolves.
func ResolveToolHint(hints map[string]string, roleID string, resolver RoleResolver) string
```

### Injection points

- **3.0 pipeline** (`internal/context/pipeline.go`): `## Available Tools` section in `stepAssembleSections`, after Role, before Procedure
- **Legacy 2.0** (`internal/mcp/handoff_tool.go`): `## Available Tools` section in `renderHandoffPrompt`, before "Additional Instructions"
- **Next** (`internal/mcp/next_tool.go`): `tool_hint` string field in `nextContextToMap()` output
- **Health** (`internal/mcp/health_tool.go`): `tool_hints` section in health output showing merged map

### Section ordering with Phase 2

When both are present: `## Available Tools` → `## Code Graph` (FR-017).

---

## Traceability Matrix

| Requirement | Task | Acceptance Criteria |
|-------------|------|---------------------|
| FR-001, FR-002 (config parsing) | TASK-…CCHS | AC-001, AC-002 |
| FR-003 (opaque strings) | TASK-…CCHS | AC-002 |
| FR-004, FR-005, FR-006 (merge strategy) | TASK-…CCHS | AC-003, AC-004, AC-005 |
| FR-007, FR-008, FR-009 (role inheritance) | TASK-…CCHS | AC-006, AC-007, AC-008, AC-009 |
| FR-010, FR-011, FR-012 (3.0 pipeline injection) | TASK-…WWZ7 | AC-010, AC-011 |
| FR-013, FR-014 (legacy 2.0 injection) | TASK-…WWZ7 | AC-012, AC-013 |
| FR-015, FR-016 (next injection) | TASK-…WWZ7 | AC-014, AC-015 |
| FR-017 (coexistence with Phase 2) | TASK-…WWZ7 | AC-016 |
| FR-018, FR-019 (health surfacing) | TASK-…DCB | AC-017, AC-018 |
| FR-020, FR-021 (backward compatibility) | TASK-…CCHS, TASK-…WWZ7 | AC-019 |