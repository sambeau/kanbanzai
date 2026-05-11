# Specification: doc register rejects non-canonical paths

## Overview

`doc(action: register)` currently accepts any path silently, even when it doesn't match the canonical path. This causes files to be registered at wrong locations, requiring another agent to move them later.

## Scope

Single-file change to `internal/mcp/doc_tool.go` in the register handler.

## Functional Requirements

- [ ] FR-1: When `doc(action: register)` receives a path that differs from the canonical path (as returned by `doc(action: path)` for the same parent/type/title), the register action SHALL reject with an error message that includes the canonical path.
- [ ] FR-2: When the supplied path matches the canonical path, registration SHALL proceed as normal (no change to happy-path behavior).

## Non-Functional Requirements

- [ ] NFR-1: The error message SHALL include the canonical path so the agent can correct in a single round-trip.

## Acceptance Criteria

- [ ] AC-1: Registering a document at a non-canonical path returns an error containing the canonical path.
- [ ] AC-2: Registering a document at the canonical path succeeds (no regression).
- [ ] AC-3: Existing tests continue to pass.
