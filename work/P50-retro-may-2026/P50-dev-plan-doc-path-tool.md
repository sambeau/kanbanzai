# Dev-Plan: Document Path Tool

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | Draft                          |
| Author | architect                      |

## Overview

This dev-plan implements the document path tool spec:
`work/P50-retro-may-2026/P50-spec-doc-path-tool.md`
(DOC-`FEAT-01KQTNYN00HZA/spec-p50-spec-doc-path-tool`).

Adds a `path` action to the `doc` MCP tool that returns canonical file paths for documents
given a type and parent entity. Also adds a warning when `doc(action: "register")` receives
a path that doesn't match the canonical form.

## Task Breakdown

### T1: Implement path-generation function
- **Deliverable:** New function `canonicalDocPath(docType string, parentEntityID string) (string, error)` in `internal/mcp/` or `internal/service/`. Uses plan slug resolution to build paths following `work/{plan-slug}/{plan-id}-{type-abbrev}-{topic}.md` convention. Handles plan/batch/feature parent resolution.
- **Depends on:** nothing
- **Effort:** 2 (slug resolution + path construction)
- **Parallelisable:** yes

### T2: Add path action to doc MCP tool
- **Deliverable:** `doc(action: "path")` handler added to `internal/mcp/doc_tool.go`. Takes `type` and optional `parent` parameters. Returns canonical path string. Returns clear errors for missing parent or non-existent parent entity.
- **Depends on:** T1
- **Effort:** 1 (wire handler to path function)
- **Parallelisable:** no

### T3: Add prompt path support
- **Deliverable:** `doc(action: "path", type: "prompt")` returns paths under `work/{plan-slug}/prompts/` or `work/_project/prompts/`. No formal prompt document type — convention only.
- **Depends on:** T1
- **Effort:** 1 (special-case path logic)
- **Parallelisable:** yes (can run alongside T2)

### T4: Add register-time path warning
- **Deliverable:** `doc(action: "register")` handler updated to call `canonicalDocPath` and include a warning in the response when the provided path doesn't match. Warning is advisory — does not block registration.
- **Depends on:** T1, T2
- **Effort:** 1 (validation + warning)
- **Parallelisable:** no

### T5: Add document path tool tests
- **Deliverable:** Unit tests for: path with design type + plan parent, path with each type abbreviation, path with feature parent resolving to plan dir, path without parent (error), path with non-existent parent (error), register with wrong path (warning), prompt path
- **Depends on:** T2, T3, T4
- **Effort:** 2 (test suite)
- **Parallelisable:** no

## Dependency Graph

```
T1 ──┬── T2 ──┬── T4 ──┬── T5
     │        │        │
     └── T3 ──┘        │
                       │
```

T2 and T3 can run in parallel after T1. T4 depends on T2. T5 gates on T2, T3, T4.

## Interface Contracts

- **doc(action: "path")** takes `type` (required) and `parent` (optional) parameters
- Returns string path or structured error
- Prompt type uses convention-based paths without formal registration
- Register-time warning is in the response map, keyed as `path_warning`
- Existing doc actions unchanged

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-001 (canonical path) | T1, T2 |
| REQ-002 (type abbreviations) | T1 |
| REQ-003 (plan slug resolution) | T1 |
| REQ-004 (no parent → error) | T2 |
| REQ-005 (non-existent parent → error) | T2 |
| REQ-006 (register warning) | T4 |
| REQ-007 (prompt path) | T3 |
| REQ-NF-001 (constant time) | T5 |
| REQ-NF-002 (pure query) | T5 |
