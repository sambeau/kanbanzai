# Design: doc register rejects non-canonical paths

## Overview

Modify `doc(action: register)` in `internal/mcp/doc_tool.go` to validate that the supplied path matches the canonical path from `doc(action: path)`.

## Design

Add a validation check in the register handler: after computing the canonical path via `doc(action: path)`, compare against the user-supplied path. If they differ, reject with an error message that includes the canonical path.

The change is a single-file edit to `internal/mcp/doc_tool.go` with corresponding test additions.

## Alternatives Considered

- **Post-hoc fixer**: Let registration proceed and fix paths later. Rejected — silent acceptance creates confusion and extra round-trips.
- **Client-side only**: Have the agent always call `doc(action: path)` first. Rejected — agents forget; server-side enforcement is more reliable.

## Dependencies

None. This is a self-contained validation addition.
