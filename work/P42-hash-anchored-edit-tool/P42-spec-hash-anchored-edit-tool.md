# Specification: Hash-Anchored Edit Tool

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | Draft                          |
| Author | AI spec-author                 |

## Problem Statement

This specification implements the hash-anchored edit mechanism described in
`work/P42-hash-anchored-edit-tool/P42-design-hash-anchored-edit-tool.md` (DOC-`P42-hash-anchored-edit-tool/design-p42-design-hash-anchored-edit-tool`). The design introduces per-line content hashing that prevents stale-line edit corruption — the most common agent edit failure mode.

Today, when an agent reads a file and then edits it, there is no guarantee the file hasn't changed between the read and the edit. If another agent, process, or human modifies the file in the interim, the agent's edit can corrupt the file by applying changes to the wrong lines. This specification adds hash-tagged line output to `read_file` and hash-validated edit input to `edit_file`, so the system can detect stale reads and reject edits before corruption occurs.

**In scope:** A `hash_tag` parameter on `read_file` that returns each line tagged with a content hash. A `hash_validate` parameter on `edit_file` that accepts hash references and validates line content before applying edits. SHA-256 hashing truncated to 2 hex characters. Backward compatibility — existing `read_file` and `edit_file` behavior is unchanged when the new parameters are absent.

**Out of scope:** A separate `hash_read` or `hash_edit` tool. Hash-tagging on `get_code_snippet` (deferred synergy with `codebase-memory-mcp`). Any change to `write_file`. Any version control or conflict resolution mechanism.

## Requirements

### Functional Requirements

#### Hash-Tagged Read

- **REQ-HR-001:** `read_file` must accept a new optional boolean parameter `hash_tag`. When `hash_tag: true`, each line in the returned content must be prefixed with its line number, a hash tag, and a separator.
- **REQ-HR-002:** The hash-tagged line format must be: `{line_number}#{hash}| {content}`, where `{line_number}` is the 1-based line number, `{hash}` is a 2-character uppercase hex string, and `|` is the separator between metadata and content.
- **REQ-HR-003:** Line numbers in hash-tagged output must match the file's current line numbering at the time of the read. Line numbers are 1-based.
- **REQ-HR-004:** When `hash_tag` is absent or set to `false`, `read_file` must behave exactly as it does today — no hash tags, no format change.
- **REQ-HR-005:** Hash-tagged output must include every line of the requested range. The hash tag prefix must be present on every line, including blank lines.
- **REQ-HR-006:** When `hash_tag: true` and a `start_line`/`end_line` range is specified, line numbers in the hash tags must reflect the file's absolute line numbering, not an offset from the range start.

#### Hash Computation

- **REQ-HC-001:** The hash for each line must be computed as SHA-256 of the line's content, truncated to 2 uppercase hexadecimal characters.
- **REQ-HC-002:** The line content hashed must be the exact text of the line as stored on disk, excluding the trailing newline character. For a blank line, the content hashed is the empty string.
- **REQ-HC-003:** The hash function must be deterministic: the same line content must always produce the same hash tag for the duration of the process. Different processes may produce different hashes (salting is not required but permitted).

#### Hash-Validated Edit

- **REQ-HE-001:** `edit_file` must accept a new optional boolean parameter `hash_validate`. When `hash_validate: true`, each edit in the `edits` array must reference a hash tag via a `hash_ref` field instead of relying on text-based fuzzy matching.
- **REQ-HE-002:** The `hash_ref` field value must be in the format `{line_number}#{hash}`, matching the prefix format produced by `read_file` with `hash_tag: true`.
- **REQ-HE-003:** When `hash_validate: true` and an edit provides a `hash_ref`, the tool must: (a) read the current file at the specified path, (b) compute the hash of the line at the referenced line number, (c) compare the computed hash against the hash in `hash_ref`, (d) if they match, apply the edit using the `new_text` field, (e) if they do not match, reject the edit with an error.
- **REQ-HE-004:** When a hash mismatch occurs, the error message must include: the line number, the expected hash (from `hash_ref`), the actual hash (computed from current file content), and an indication that the file may have changed since the last read.
- **REQ-HE-005:** When `hash_validate: true` and the referenced line number exceeds the current file length (file was shortened), the tool must reject the edit with an error indicating the line no longer exists.
- **REQ-HE-006:** When `hash_validate` is absent or set to `false`, `edit_file` must behave exactly as it does today — text-based fuzzy matching with no hash validation.
- **REQ-HE-007:** When `hash_validate: true` and an edit does not provide a `hash_ref` field, the tool must reject that edit with an error indicating that `hash_ref` is required when hash validation is enabled.

#### Backward Compatibility

- **REQ-BC-001:** Existing callers of `read_file` and `edit_file` must be unaffected. No existing test must break as a result of this change.
- **REQ-BC-002:** The `hash_ref` field in edits must be optional at the schema level — only validated when `hash_validate: true`.

#### Error Modes

- **REQ-ERR-001:** When a hash collision occurs (different line content produces the same 2-char hash), the edit must be applied to the wrong line. This is an accepted risk at 1/256 probability per line. The system must not attempt to detect or prevent hash collisions in the initial implementation.
- **REQ-ERR-002:** When a file is not found during hash-validated edit, the tool must return the standard file-not-found error — identical to the current `edit_file` behavior.

### Non-Functional Requirements

- **REQ-NF-001:** Hash computation must not add perceptible latency to `read_file` calls. For files under 10,000 lines, the added cost of SHA-256 per line must be under 10ms total.
- **REQ-NF-002:** Hash-tagged output must be visually parseable by an agent. The format must use a fixed-width line number field (minimum 4 characters, left-padded with spaces) so that content aligns vertically across lines with different line number widths.
- **REQ-NF-003:** The hash validation logic must be implemented in a separate internal package from the existing fuzzy-match logic so that extraction of a standalone `hash_edit` tool is possible in the future without touching the fuzzy-match code.

## Constraints

- The hash function must be SHA-256, not MD5 or a non-cryptographic hash, to ensure uniform distribution across the truncated output space.
- Hash tags are ephemeral — they are valid only for the session in which the file was read. The system must not persist hash tags or use them for cross-session validation.
- The `edit_file` schema must not require `hash_ref` at the top level. It is only required when `hash_validate: true`. This preserves backward compatibility for callers who do not use hash validation.
- The `read_file` format must not change when `hash_tag` is absent. Existing agents that parse `read_file` output must see no difference.
- The implementation must not introduce new MCP tools. Enhancement is via parameters on existing `read_file` and `edit_file` tools.
- This specification does NOT cover: hash-tagging in `get_code_snippet` (deferred), a standalone `hash_edit` tool (deferred), or any change to `write_file`.

## Acceptance Criteria

#### Hash-Tagged Read

- **AC-HR-001 (REQ-HR-001, REQ-HR-002):** Given a file at `src/main.go` with 3 lines, when `read_file(path: "src/main.go", hash_tag: true)` is called, then the output contains three lines each prefixed with a line number, a 2-char uppercase hex hash tag, and a `|` separator.
- **AC-HR-002 (REQ-HR-003, REQ-HR-006):** Given a file with 50 lines, when `read_file(path: "file.txt", hash_tag: true, start_line: 10, end_line: 15)` is called, then the output shows lines 10–15 with line numbers 10 through 15 (not 1 through 6).
- **AC-HR-003 (REQ-HR-004):** Given a file, when `read_file(path: "file.txt")` is called without `hash_tag`, then the output is identical to the current `read_file` output — no hash tags, no format change.
- **AC-HR-004 (REQ-HR-005):** Given a file with a blank line at line 3, when `read_file(path: "file.txt", hash_tag: true)` is called, then line 3 is displayed with its hash tag prefix and separator followed by empty content (e.g., `   3#AB| `).

#### Hash Computation

- **AC-HC-001 (REQ-HC-001):** Given a line with content "function hello() {", when the hash is computed twice in the same process, then both hash tags are identical.
- **AC-HC-002 (REQ-HC-002):** Given a file where line 5 is "  return x;" (with trailing newline), when the hash is computed, then the newline character is excluded from the hash input.

#### Hash-Validated Edit

- **AC-HE-001 (REQ-HE-001, REQ-HE-003):** Given a file read with `hash_tag: true` showing line 22 with hash `#XJ` and content `"  return \"world\";"`, when `edit_file(path: "file.go", hash_validate: true, edits: [{hash_ref: "22#XJ", new_text: "  return \"hello, world\";"}])` is called and the file has not changed, then the edit is applied successfully.
- **AC-HE-002 (REQ-HE-003, REQ-HE-004):** Given a file read with `hash_tag: true` showing line 22 with hash `#XJ`, when the file is modified externally before `edit_file` is called with `hash_ref: "22#XJ"`, then the edit is rejected with an error containing the line number (22), the expected hash (`#XJ`), and the actual computed hash.
- **AC-HE-003 (REQ-HE-005):** Given a file read with `hash_tag: true` showing line 22, when the file is shortened to 15 lines externally before `edit_file` is called with `hash_ref: "22#XJ"`, then the edit is rejected with an error indicating line 22 no longer exists.
- **AC-HE-004 (REQ-HE-006):** Given a file, when `edit_file` is called without `hash_validate`, then the current fuzzy-match behavior applies and the edit succeeds or fails based on text matching, not hash matching.
- **AC-HE-005 (REQ-HE-007):** Given `edit_file` is called with `hash_validate: true` and an edit that has no `hash_ref` field, then that edit is rejected with an error indicating `hash_ref` is required.

#### Backward Compatibility

- **AC-BC-001 (REQ-BC-001):** Given the existing test suite, when all tests are run after the hash-anchored edit implementation, then all tests that passed before the change continue to pass.
- **AC-BC-002 (REQ-BC-002):** Given `edit_file` is called with `hash_validate: false` and edits that omit `hash_ref`, then the edits are accepted (hash_ref is optional at the schema level).

#### Error Modes

- **AC-ERR-001 (REQ-ERR-001):** Given a hash collision scenario (documented in test setup), when a hash-validated edit is submitted with a hash that matches the wrong line due to collision, then the edit is applied to the wrong line. This is documented as accepted behavior; the test verifies the system does not crash or error in an unexpected way.
- **AC-ERR-002 (REQ-ERR-002):** Given `edit_file(path: "nonexistent.go", hash_validate: true, edits: [{hash_ref: "1#AB", new_text: "x"}])`, then the tool returns a file-not-found error identical to the current `edit_file` behavior for missing files.

#### Non-Functional

- **AC-NF-001 (REQ-NF-001):** Given a file with 10,000 lines, when `read_file(path: "large.go", hash_tag: true)` is called, then the hash computation overhead is ≤ 10ms.
- **AC-NF-002 (REQ-NF-002):** Given a file with more than 9 lines but fewer than 100 lines, when `read_file(..., hash_tag: true)` is called, then line numbers are left-padded to at least 4 characters so that content after the `|` separator aligns vertically across all lines.
- **AC-NF-003 (REQ-NF-003):** Given the implementation source tree, when a reviewer inspects the code, then the hash validation logic resides in a separate package from the fuzzy-match logic (e.g., `internal/hashvalidate/` or similar).

## Verification Plan

| Requirement(s) | Method | Description |
|----------------|--------|-------------|
| REQ-HR-001, REQ-HR-002 | Test | Unit test: read a 3-line file with `hash_tag: true` → verify output format matches `{line}#{hash}| {content}` |
| REQ-HR-003 | Test | Unit test: verify line numbers in hash-tagged output are 1-based and match file line numbering |
| REQ-HR-004 | Test | Regression test: call `read_file` without `hash_tag` → verify output unchanged from pre-implementation behavior |
| REQ-HR-005 | Test | Unit test: file with blank line → verify blank line still gets hash tag prefix and separator |
| REQ-HR-006 | Test | Unit test: `start_line: 10, end_line: 15` with `hash_tag: true` → verify line numbers are 10–15 |
| REQ-HC-001, REQ-HC-003 | Test | Unit test: hash the same content twice → verify identical output (determinism test)
| REQ-HC-002 | Test | Unit test: verify trailing newline excluded from hash input (lines "abc" and "abc\n" produce same hash) |
| REQ-HE-001, REQ-HE-002, REQ-HE-003 | Test | Integration test: read with hash → edit with matching `hash_ref` in correct format → verify edit applied |
| REQ-HE-004 | Test | Integration test: read with hash → modify file externally → edit with stale hash → verify rejection with expected/actual hash in error |
| REQ-HE-005 | Test | Integration test: read with hash → truncate file externally → edit with now-out-of-range hash → verify line-not-found error |
| REQ-HE-006 | Test | Regression test: call `edit_file` without `hash_validate` → verify fuzzy-match behavior unchanged |
| REQ-HE-007 | Test | Unit test: `hash_validate: true` + edit missing `hash_ref` → verify rejected with "hash_ref required" error |
| REQ-BC-001 | Test | Full test suite run → verify zero regressions |
| REQ-BC-002 | Test | Unit test: `hash_validate: false` + edits without `hash_ref` → verify accepted (optional at schema level) |
| REQ-ERR-001 | Test | Unit test: construct two lines with same 2-char hash → verify edit applied without crash or unexpected error |
| REQ-ERR-002 | Test | Unit test: hash-validated edit on nonexistent file → verify standard file-not-found error |
| REQ-NF-001 | Test | Performance test: time `read_file` with `hash_tag: true` on a 10K-line file → verify hash overhead ≤ 10ms |
| REQ-NF-002 | Inspection | Review hash-tagged output format → verify fixed-width line number field (≥4 chars, left-padded) |
| REQ-NF-003 | Inspection | Review package layout → verify hash validation in separate package from fuzzy-match logic |
