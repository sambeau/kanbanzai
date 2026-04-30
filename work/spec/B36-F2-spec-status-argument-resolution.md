# Specification: Status Command Argument Resolution

| Field  | Value                                   |
|--------|-----------------------------------------|
| Date   | 2026-04-30                              |
| Status | approved |
| Author | spec-author                             |

---

## Related Work

- **Design:** `work/_project/design-kbz-cli-and-binary-rename.md` — §5.1 (command interface and exit codes), §5.2 (file path resolution), §5.3 (path resolution for `kbz doc approve`). This spec formalises the requirements stated in those sections.
- **B36-F3** (status rendering — human/plain formats) and **B36-F4** (status rendering — JSON format) depend on the routing and resolution logic defined here. F3 and F4 implement the rendering layer that this feature routes to; they must not be implemented before F2 is complete.

---

## Overview

`kbz status` currently accepts no arguments and always shows a project overview. This feature extends the command to accept an optional `<target>` argument — a file path, a full or display-format entity ID, or a plan prefix — and routes the request to the appropriate service-layer query. It also extends `kbz doc approve` to accept a file path in addition to a document ID.

The feature is concerned exclusively with argument parsing, disambiguation, path-to-record lookup, and routing. Output rendering is out of scope (see B36-F3, B36-F4).

---

## Scope

**In scope:**
- Parsing `kbz status [<target>] [--format <fmt>]`
- Disambiguation of `<target>` (file path vs entity ID vs plan prefix)
- File path → document record lookup; unregistered-document response
- Entity ID → service layer routing (task, feature, bug, plan, batch)
- Plan prefix resolution (e.g. `P1` → plan entity)
- Exit code semantics for all resolution outcomes
- Parsing `kbz doc approve <path-or-id> [--by <user>]`
- File path → document record ID resolution for `kbz doc approve`
- Error messages for all failure modes

**Out of scope:**
- Output rendering for any format (human, plain, JSON) — B36-F3, B36-F4
- Changes to the project overview (no-target) data or rendering
- `kbz doc register` path changes
- Multi-target support (`kbz status path1 path2`)
- `--watch` mode
- Title-based resolution for `kbz doc approve`

---

## Functional Requirements

### Command interface

- **FR-001:** `kbz status` MUST accept an optional positional argument `<target>`. When omitted, the command MUST produce the existing project overview behaviour unchanged.

- **FR-002:** `kbz status` MUST accept a `--format` flag (short form `-f`) with exactly three valid values: `human`, `plain`, and `json`. When `--format` is omitted, `human` MUST be the default.

- **FR-003:** `kbz status` MUST NOT accept any flags other than `--format`/`-f`. Any unrecognised flag MUST produce a usage error and exit with code 2.

- **FR-004:** `kbz status` MUST NOT accept more than one positional argument. If more than one positional argument is provided, the command MUST produce a usage error and exit with code 2.

### Disambiguation

- **FR-005:** When `<target>` contains a `/` character OR ends with `.md` or `.txt`, the command MUST treat `<target>` as a file path without consulting the entity ID pattern.

- **FR-006:** When `<target>` does not satisfy FR-005 and matches a known entity ID pattern (full ULID-based ID or display-format ID such as `FEAT-042`, `BUG-007`, `P1-my-plan`, `B2-my-batch`, `TASK-001`), the command MUST treat `<target>` as an entity ID.

- **FR-007:** When `<target>` matches neither FR-005 nor FR-006, the command MUST first attempt entity ID lookup. If that succeeds, the result is used. If entity ID lookup fails, the command MUST attempt file path lookup. If file path lookup also fails, the command MUST exit with code 1 and print a descriptive error message.

- **FR-008:** When `<target>` matches the pattern of a bare plan prefix (e.g. `P1`, `P2`) — one or two uppercase letters followed by one or more digits, with no slug — the command MUST resolve it to the plan entity with that prefix and route to the plan view. If no such plan exists, the command MUST exit with code 1.

### File path resolution

- **FR-009:** When `<target>` is treated as a file path (FR-005 or FR-007), the command MUST first check whether the file exists on disk. If the file does not exist on disk, the command MUST exit with code 1.

- **FR-010:** When the file exists on disk, the command MUST look up the file path in the document record store using an exact, repo-relative path match. The path comparison MUST be case-sensitive and MUST NOT resolve symlinks or normalise separators beyond making the path repo-relative.

- **FR-011:** When no document record is found for a file that exists on disk, the command MUST produce an unregistered-document response and exit with code 0. The unregistered-document response MUST include: the file path, a statement that the file is not registered, and a suggested `kbz doc register` command with `--type` and `--title` placeholders.

- **FR-012:** When a document record is found, the command MUST route to the document view followed by the owner entity view if an owner entity is recorded on the document. If no owner entity is recorded, only the document view is shown.

- **FR-013:** File paths provided as `<target>` MUST be accepted as relative paths from the repository root. The command MUST normalise relative paths (e.g. stripping a leading `./`) before performing a document record lookup.

### Entity ID routing

- **FR-014:** When `<target>` is treated as an entity ID, the command MUST route to the service-layer query appropriate for the entity type: feature, task, bug, batch, or plan.

- **FR-015:** When `<target>` is a display-format entity ID (e.g. `FEAT-042`), the command MUST resolve it to the full internal ID before performing the service-layer query.

- **FR-016:** When the entity ID (full or display-format) does not correspond to any existing entity, the command MUST exit with code 1 and print a descriptive error message identifying the ID that was not found.

### Exit codes

- **FR-017:** The command MUST exit with code 0 when a query completes successfully, including when the result is an unregistered-document response (FR-011).

- **FR-018:** The command MUST exit with code 1 when: a file path argument refers to a file that does not exist on disk; an entity ID is not found; the state store cannot be read; or any other system-level error occurs.

- **FR-019:** The command MUST exit with code 2 when: an unrecognised flag is provided; `--format` is given an unrecognised value; more than one positional argument is provided; or any other usage error occurs.

- **FR-020:** When `--format` is given a value other than `human`, `plain`, or `json`, the command MUST exit with code 2 and print a message listing the valid values.

### `kbz doc approve` path resolution

- **FR-021:** `kbz doc approve` MUST accept a file path as its first argument in addition to a document ID. A file path argument is identified by the same rule as FR-005: it contains `/` or ends with `.md` or `.txt`.

- **FR-022:** When a file path is provided to `kbz doc approve`, the command MUST look up the document record for that path using the same exact, repo-relative match as FR-010. If no record is found, the command MUST exit with code 1 and print an error message indicating the file is not registered.

- **FR-023:** When a file path is provided to `kbz doc approve` and a document record is found, the command MUST proceed with approval using the resolved document ID. Observable behaviour (output, exit code, `--by` flag handling) MUST be identical to the existing ID-based flow.

- **FR-024:** The existing `kbz doc approve <id>` form (passing a document ID directly) MUST continue to work without any change in behaviour.

---

## Non-Functional Requirements

- **NFR-001:** The disambiguation logic (FR-005 through FR-008) MUST be implemented as a single, deterministic decision sequence that executes without any I/O before determining the resolution strategy. The determination of "is this a file path?" or "is this an entity ID pattern?" MUST be based on lexical rules only.

- **NFR-002:** File path normalisation (FR-013) MUST NOT require the working directory to be the repository root. The command MUST resolve paths relative to the repository root regardless of where the user invokes it.

- **NFR-003:** Error messages for missing files and missing entities MUST include the value the user provided so the user can identify a typo without re-reading their command.

- **NFR-004:** The `--format` flag value MUST be passed through to the rendering layer without modification. Argument parsing and rendering are separate concerns; this feature MUST NOT interpret the format value beyond validating it.

---

## Acceptance Criteria

- [ ] **AC-001:** Given `kbz status work/design/foo.md`, when the argument is parsed, then the command routes to file path resolution without consulting the entity ID matcher.

- [ ] **AC-002:** Given `kbz status notes.txt`, when the argument is parsed, then the command routes to file path resolution.

- [ ] **AC-003:** Given `kbz status FEAT-042`, when the argument is parsed, then the command routes to entity ID lookup.

- [ ] **AC-004:** Given `kbz status P1-my-plan`, when the argument is parsed, then the command routes to entity ID lookup.

- [ ] **AC-005:** Given `kbz status P1` and a plan with prefix `P1` exists, when the command runs, then it produces the plan view for that plan and exits 0.

- [ ] **AC-006:** Given `kbz status P99` and no plan with prefix `P99` exists, when the command runs, then it exits 1 with an error message referencing `P99`.

- [ ] **AC-007:** Given `kbz status sometoken` where `sometoken` matches no entity ID pattern, when entity ID lookup fails and file path lookup also fails, then the command exits 1 with a descriptive error.

- [ ] **AC-008:** Given `kbz status work/design/nonexistent.md` and the file does not exist on disk, when the command runs, then it exits 1 with an error message referencing the path.

- [ ] **AC-009:** Given `kbz status work/design/unregistered.md` and the file exists on disk but has no document record, when the command runs, then it exits 0 and the output includes the phrase "not registered" and a suggested `kbz doc register` command.

- [ ] **AC-010:** Given `kbz status work/design/registered.md` and the file has a document record with an owner feature, when the command runs, then it exits 0 and routes to both the document view and the owner feature view.

- [ ] **AC-011:** Given `kbz status work/design/registered.md` and the file has a document record with no owner entity, when the command runs, then it exits 0 and routes to the document view only (no entity view).

- [ ] **AC-012:** Given `kbz status ./work/design/foo.md` (leading `./`), when the command runs, then it performs the same document record lookup as `kbz status work/design/foo.md`.

- [ ] **AC-013:** Given `kbz status FEAT-999` and no feature with display ID `FEAT-999` exists, when the command runs, then it exits 1 with an error message containing `FEAT-999`.

- [ ] **AC-014:** Given `kbz status FEAT-042` and the feature exists, when the command runs, then it exits 0 and routes to the feature view.

- [ ] **AC-015:** Given `kbz status BUG-007` and the bug exists, when the command runs, then it exits 0 and routes to the bug view.

- [ ] **AC-016:** Given `kbz status --format xml`, when the command runs, then it exits 2 and prints a message listing `human`, `plain`, and `json` as the valid values.

- [ ] **AC-017:** Given `kbz status --unknown-flag`, when the command runs, then it exits 2 with a usage error.

- [ ] **AC-018:** Given `kbz status FEAT-042 FEAT-043`, when the command runs, then it exits 2 with a usage error.

- [ ] **AC-019:** Given `kbz status` with no target and the project overview succeeds, when the command runs, then it exits 0.

- [ ] **AC-020:** Given `kbz status` and the state store cannot be read, when the command runs, then it exits 1.

- [ ] **AC-021:** Given `kbz doc approve work/design/foo.md` and no document record exists for that path, when the command runs, then it exits 1 with an error message stating the file is not registered.

- [ ] **AC-022:** Given `kbz doc approve work/design/foo.md` and a document record exists for that path, when the command runs, then it approves the document and exits 0, producing the same output as `kbz doc approve <resolved-id>`.

- [ ] **AC-023:** Given `kbz doc approve DOC-0012` where `DOC-0012` exists, when the command runs, then behaviour is unchanged from the current implementation.

- [ ] **AC-024:** Given `kbz doc approve work/design/foo.md --by alice` and a document record exists for that path, when the command runs, then the approval is recorded with `alice` as the approver, identical to `kbz doc approve <resolved-id> --by alice`.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001–AC-007 | Unit test | Parse a representative set of target strings and assert the disambiguation outcome (file path vs entity ID vs plan prefix) without performing I/O. |
| AC-008 | Integration test | Invoke `kbz status` with a path to a file that does not exist; assert exit code 1 and error message content. |
| AC-009 | Integration test | Invoke `kbz status` with a path to a real file absent from the document record store; assert exit code 0 and output contains "not registered" and a `kbz doc register` suggestion. |
| AC-010–AC-011 | Integration test | Invoke `kbz status` with paths to documents with and without owner entities; assert routing outcome via stub renderer or output sentinel. |
| AC-012 | Unit test | Pass `./work/design/foo.md` through path normalisation; assert the normalised result equals `work/design/foo.md`. |
| AC-013–AC-015 | Integration test | Invoke `kbz status` with existing and non-existing entity IDs; assert exit codes and routing. |
| AC-016–AC-018 | Unit test | Parse flag/argument combinations that should produce usage errors; assert exit code 2 and message content. |
| AC-019–AC-020 | Integration test | Invoke `kbz status` with no target in healthy and broken state store; assert exit codes. |
| AC-021–AC-024 | Integration test | Invoke `kbz doc approve` with path and ID arguments, registered and unregistered; assert exit codes, output, and stored approval state. |

---

## Dependencies and Assumptions

- **DEP-001:** The document record service must support exact lookup by repo-relative file path. If `DocumentService` does not already expose this as a single method, the service layer must be extended before this feature can be implemented. (See design §7.2.)

- **DEP-002:** The entity ID display-format pattern (e.g. `FEAT-042`) must be codified in a single location that the disambiguation logic can consult without duplicating the pattern.

- **DEP-003:** B36-F3 (human/plain rendering) and B36-F4 (JSON rendering) depend on this feature's routing output. These features must not be started until the routing interface defined here is stable.

- **ASM-001:** File paths provided by the user are assumed to be relative to the repository root. The command does not need to support absolute paths.

- **ASM-002:** Plan prefix patterns are of the form `[A-Z]{1,2}[0-9]+` (one or two uppercase letters followed by one or more digits). This pattern does not overlap with entity ID display-format patterns and can be matched lexically.

- **ASM-003:** The `--format` flag value is opaque to the resolution layer; the rendering layer is responsible for interpreting it. Any future format values added by F3/F4 do not require changes to the argument parsing defined here, provided they are added to the validation allow-list.

- **ASM-004:** `kbz doc approve` path resolution uses the same lexical rule as `kbz status` (FR-005): an argument containing `/` or ending in `.md`/`.txt` is treated as a file path. Document IDs do not match this pattern so there is no ambiguity.
