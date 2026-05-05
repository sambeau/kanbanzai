# Specification: Document Path Tool

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | spec-author                    |

## Overview

This specification implements the document path tool described in
`work/P50-retro-may-2026/P50-design-retro-may-2026.md`
(DOC-`P50-retro-may-2026/design-p50-design-retro-may-2026`), Feature 3.

Document placement conventions are documented in `AGENTS.md` and
KE-01KQSER9N0QHY but enforced only by agent memory. Agents sometimes place
documents in incorrect directories because they guess paths rather than
following the convention. A `path` action on the `doc` MCP tool returns the
canonical path for a document given its type and parent entity, making the
convention enforceable by tooling.

## Scope

**In scope:**
- `doc(action: "path")` that returns a canonical file path for a given document type and optional parent entity
- Path validation warning when `doc(action: "register")` is called with a path that does not match the canonical form

**Out of scope:**
- Adding a `prompt` document type to the `doc` tool schema
- Enforcing path conventions at registration time (warning only)
- Renaming existing documents to match conventions

## Functional Requirements

- **REQ-001:** Given a document type (`design`, `specification`, `dev-plan`,
  `research`, `report`) and a parent entity ID with a known slug, when
  `doc(action: "path")` is called, then the tool must return the canonical
  file path following the convention:
  `work/{plan-slug}/{plan-id}-{type-abbrev}-{topic-slug}.md`.
- **REQ-002:** The type abbreviations must follow the document-to-topic
  map conventions: `design` → `design`, `specification` → `spec`,
  `dev-plan` → `dev-plan`, `research` → `research`, `report` → `report`.
- **REQ-003:** When the parent entity is a plan, the path must use the
  plan's slug as the directory component. When the parent entity is a
  batch or feature, the path must resolve upward to find the owning plan's
  slug.
- **REQ-004:** When no parent entity is provided, the tool must return an
  error with a clear message: "Cannot determine path: no parent entity
  provided. Specify a parent plan, batch, or feature ID."
- **REQ-005:** When the parent entity does not exist, the tool must return
  an error with a clear message: "Cannot determine path: parent entity
  {id} not found."
- **REQ-006:** When `doc(action: "register")` is called with a path that
  does not match the canonical form returned by `doc(action: "path")` for
  the same type and parent, the tool must include a warning in its
  response indicating the mismatch and the expected canonical path.
- **REQ-007:** For a `prompt` file (which has no formal document type in
  the schema), `doc(action: "path", type: "prompt")` must return a path
  under `work/{plan-slug}/prompts/` when a parent plan exists, or
  `work/_project/prompts/` when no parent is specified.

## Non-Functional Requirements

- **REQ-NF-001:** `doc(action: "path")` must complete in constant time —
  no file I/O beyond entity lookup for the parent.
- **REQ-NF-002:** The path action must not modify any state — it is a
  pure query.

## Constraints (Scope Exclusions)

- The `doc` tool's existing actions (`register`, `approve`, `get`,
  `content`, `list`, etc.) must not change behaviour.
- Path generation must use the same slug resolution logic that
  `entity(action: "get")` uses — no duplicate slug extraction code.
- This specification does NOT cover a `prompt` document type. The `path`
  action handles prompt placement as a convention, not a registered type.
- The register-time path warning is advisory only and must not block
  registration.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given `doc(action: "path", type: "design", parent:
  "P50-retro-may-2026")`, then the returned path is
  `work/P50-retro-may-2026/P50-design-{topic}.md` where `{topic}` is
  derived from the parent's slug.
- **AC-002 (REQ-002):** Given each document type (`design`,
  `specification`, `dev-plan`, `research`, `report`), when `path` is
  called, then the returned path uses the correct type abbreviation in the
  filename.
- **AC-003 (REQ-003):** Given a parent feature under batch B49 under plan
  P50, when `doc(action: "path")` is called with that feature as parent,
  then the returned path uses P50's slug as the directory component.
- **AC-004 (REQ-004):** Given `doc(action: "path", type: "design")` with
  no parent, then the tool returns an error stating that a parent entity
  is required.
- **AC-005 (REQ-005):** Given `doc(action: "path", parent: "P999-nonexist")`,
  then the tool returns an error stating the parent entity was not found.
- **AC-006 (REQ-006):** Given `doc(action: "register", path:
  "work/wrong-dir/spec.md", type: "specification", parent: "P50")`, then
  the response includes a warning that the path does not match the
  canonical form and shows the expected path.
- **AC-007 (REQ-007):** Given `doc(action: "path", type: "prompt", parent:
  "P50-retro-may-2026")`, then the returned path is
  `work/P50-retro-may-2026/prompts/{slug}.md`.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: call path with design type and plan parent, assert correct path |
| AC-002 | Test | Table-driven test: call path with each type, assert correct abbreviation |
| AC-003 | Test | Unit test: call path with feature parent, assert plan-level directory |
| AC-004 | Test | Unit test: call path without parent, assert error returned |
| AC-005 | Test | Unit test: call path with non-existent parent, assert error returned |
| AC-006 | Test | Integration test: register with wrong path, assert warning in response |
| AC-007 | Test | Unit test: call path with prompt type, assert correct prompts directory |
| REQ-NF-001 | Test | Benchmark doc(action: path) with cold and warm caches, assert response time is O(1) regardless of repository size |
| REQ-NF-002 | Test | Verify doc(action: path) does not produce any writes to .kbz/state/ or .kbz/index/ (call then check git status for no changes) |
