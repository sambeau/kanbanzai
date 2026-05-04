# Dev-Plan: Hash-Anchored Edit Tool

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | AI architect                   |

## Overview

This plan implements the hash-anchored edit tool specified in
`work/P42-hash-anchored-edit-tool/P42-spec-hash-anchored-edit-tool.md` (DOC-`P42-hash-anchored-edit-tool/spec-p42-spec-hash-anchored-edit-tool`). It covers all 23 requirements across 5 task groups: hash computation, hash-tagged read, hash-validated edit, backward compatibility and error modes, and integration testing.

**In scope:** A `hash_tag` parameter on `read_file`, a `hash_validate` + `hash_ref` mechanism on `edit_file`, SHA-256 hashing truncated to 2 hex characters, backward compatibility with existing callers, and a separate internal package for hash logic enabling future extraction.

**Out of scope:** Hash-tagging on `get_code_snippet` (deferred synergy with `codebase-memory-mcp`), a standalone `hash_edit` tool (future extraction), and any change to `write_file`.

## Task Breakdown

### Task 1: Implement hash computation package

- **Description:** Create `internal/hashvalidate/` package with SHA-256 line-hashing. Expose `HashLine(content string) string` returning a 2-char uppercase hex hash tag. The hash must exclude trailing newlines and be deterministic within the process. Include unit tests verifying determinism, newline exclusion, and uniform distribution across the 256-value space.
- **Deliverable:** `internal/hashvalidate/hash.go` + `internal/hashvalidate/hash_test.go`
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** REQ-HC-001, REQ-HC-002, REQ-HC-003, REQ-NF-003

### Task 2: Add hash-tagged output to read_file

- **Description:** Add an optional `hash_tag` boolean parameter to the `read_file` MCP tool. When `true`, prefix each line with `{line_number}#{hash}| ` where line numbers are 1-based absolute positions, hashes come from `HashLine()`, and the separator is `|`. Line numbers must be left-padded to at least 4 characters for vertical alignment. When `hash_tag` is absent or `false`, output is unchanged. Handle blank lines (empty content still gets hash-tag prefix). Handle `start_line`/`end_line` ranges with correct absolute line numbering.
- **Deliverable:** Modified `read_file` tool handler + tests
- **Depends on:** Task 1
- **Effort:** Medium
- **Spec requirement:** REQ-HR-001, REQ-HR-002, REQ-HR-003, REQ-HR-004, REQ-HR-005, REQ-HR-006, REQ-NF-001, REQ-NF-002

### Task 3: Add hash-validated edits to edit_file

- **Description:** Add an optional `hash_validate` boolean parameter to the `edit_file` MCP tool. When `true`, each edit must provide a `hash_ref` field in format `{line_number}#{hash}`. The tool reads the current file, computes the hash of the referenced line, and compares. Match → apply the edit with `new_text`. Mismatch → reject with error containing line number, expected hash, and actual hash. Line number out of range → reject with error. When `hash_validate` is absent or `false`, current fuzzy-match behavior applies unchanged. When `hash_validate: true` and an edit lacks `hash_ref`, reject with "hash_ref required" error.
- **Deliverable:** Modified `edit_file` tool handler + tests
- **Depends on:** Task 1
- **Effort:** Medium
- **Spec requirement:** REQ-HE-001, REQ-HE-002, REQ-HE-003, REQ-HE-004, REQ-HE-005, REQ-HE-006, REQ-HE-007

### Task 4: Backward compatibility, error modes, and schema updates

- **Description:** Ensure backward compatibility: existing callers of `read_file` and `edit_file` are unaffected (zero regression in existing tests). Make `hash_ref` optional at the schema level (only validated when `hash_validate: true`). Document hash collision acceptance (1/256 per line — edit may apply to wrong line; no detection logic). Verify file-not-found behavior is unchanged for hash-validated edits. Update MCP tool schemas to declare new parameters.
- **Deliverable:** Schema updates, regression test suite pass, error-mode test cases
- **Depends on:** Task 2, Task 3
- **Effort:** Small
- **Spec requirement:** REQ-BC-001, REQ-BC-002, REQ-ERR-001, REQ-ERR-002

### Task 5: Integration tests and verification

- **Description:** Write end-to-end integration tests covering the full read-then-edit flow: (a) read with hashes → edit with matching hashes → verify edit applied correctly; (b) read with hashes → modify file externally → edit with stale hashes → verify rejection with correct error details; (c) read with hashes → truncate file externally → edit with out-of-range hash → verify line-not-found error. Verify all 23 spec requirements are covered by tests. Run full test suite to confirm zero regressions.
- **Deliverable:** Integration test file(s), verification traceability matrix
- **Depends on:** Task 4
- **Effort:** Medium
- **Spec requirement:** REQ-HR-001 through REQ-ERR-002 (cross-cutting verification of all 23 requirements)

## Dependency Graph

```
Task 1 (hash package) ──┬── Task 2 (hash-tagged read) ──┐
                        │                                ├── Task 4 (backward compat + errors) ── Task 5 (integration tests)
                        └── Task 3 (hash-validated edit) ┘

Parallel groups: [Task 2, Task 3]
Critical path: Task 1 → Task 2 → Task 4 → Task 5
```

Task 1 is the foundation — both Task 2 and Task 3 depend on it but not on each other, so they can run in parallel. Task 4 ties the two together with backward compatibility checks and schema updates, and Task 5 verifies the whole system end-to-end.

## Risk Assessment

### Risk 1: Hash collision at 2-char truncation

- **Probability:** Low (~1/256 per line per edit)
- **Impact:** Low — collision means a stale edit could slip through, but the consequence is a fuzzy-match fallback (current behavior), not data corruption
- **Mitigation:** Document accepted risk. Monitor collision rate in practice. If collisions cause visible edit failures, upgrade to 4-char hashes (1/65,536) in a follow-up task — the `HashLine` function signature accepts a length parameter.
- **Affected tasks:** Task 1, Task 3

### Risk 2: Schema change breaks existing MCP clients

- **Probability:** Low — new parameters are optional booleans with safe defaults
- **Impact:** Medium — if the schema change is incompatible, existing `read_file`/`edit_file` callers would break
- **Mitigation:** Add parameters as optional fields only. Run full test suite before merging. Verify `hash_tag` absent and `hash_validate` absent both produce current behavior.
- **Affected tasks:** Task 2, Task 3, Task 4

### Risk 3: Performance regression on large files

- **Probability:** Low — SHA-256 per line is fast; 10K lines target is ≤10ms
- **Impact:** Low — even if slightly over budget, hash-tagging is opt-in; callers who don't use it pay zero cost
- **Mitigation:** Performance test in Task 2 verifies the 10ms budget. If exceeded, optimize (pre-allocate buffer, compute hashes in parallel for very large files).
- **Affected tasks:** Task 1, Task 2

### Risk 4: Hash determinism across file reads in same process

- **Probability:** Very low — SHA-256 of identical content with same newline handling is deterministic by definition
- **Impact:** Medium — if hashes change between read and edit within the same session, valid edits would be falsely rejected
- **Mitigation:** Task 1 includes determinism unit test. Task 5 end-to-end test validates read-then-edit with no external modification.
- **Affected tasks:** Task 1, Task 3, Task 5

## Interface Contracts

### HashLine function contract (Task 1 → Tasks 2, 3)

- **Signature:** `func HashLine(content string) string`
- **Input:** A line's content as a string, excluding the trailing newline character.
- **Output:** A 2-character uppercase hexadecimal hash tag string (e.g., `"AB"`, `"3F"`).
- **Contract:** The function must be deterministic within a single process — the same input must always produce the same output. The hash must be derived from SHA-256 truncated to 2 hex characters. Trailing newlines in the input must be stripped before hashing.
- **Consumer tasks:** Task 2 (hash-tagged read) and Task 3 (hash-validated edit) both call `HashLine()`.

### read_file hash_tag parameter contract (Task 2 → Tasks 4, 5)

- **Parameter:** `hash_tag` (boolean, optional, default `false`).
- **When `false` or absent:** Output is identical to current `read_file` behavior — plain text, no hash tags. No schema change visible to callers who omit the parameter.
- **When `true`:** Each line is prefixed with `{line_number}#{hash}| ` where `{line_number}` is 1-based, left-padded to ≥4 chars, `{hash}` is the output of `HashLine()`, and `|` is a literal separator. Absolute line numbering applies regardless of `start_line`/`end_line` range.
- **Consumer tasks:** Task 4 verifies backward compatibility. Task 5 tests the read-then-edit flow.

### edit_file hash_validate parameter contract (Task 3 → Tasks 4, 5)

- **Parameter:** `hash_validate` (boolean, optional, default `false`).
- **When `false` or absent:** Current fuzzy-match behavior applies unchanged.
- **When `true`:** Each edit object in the `edits` array must include a `hash_ref` field in format `{line_number}#{hash}`. The tool reads the current file, computes `HashLine()` on the referenced line, and compares. Match → apply `new_text`. Mismatch → error with line number, expected hash, and actual hash. Line out of range → error.
- **hash_ref schema:** The `hash_ref` field is required when `hash_validate: true` and optional at the schema level (not required at the top-level schema).
- **Consumer tasks:** Task 4 verifies backward compatibility and error modes. Task 5 tests end-to-end flows.

## Traceability Matrix

| Spec Requirement | Task(s) | Verification |
|-----------------|---------|-------------|
| REQ-HC-001 (SHA-256, 2-char hex) | Task 1 | Unit test: determinism |
| REQ-HC-002 (newline exclusion) | Task 1 | Unit test: "abc" vs "abc\n" |
| REQ-HC-003 (process determinism) | Task 1 | Unit test: double-hash |
| REQ-HR-001 (hash_tag parameter) | Task 2 | Unit test: format check |
| REQ-HR-002 (format: line#hash|content) | Task 2 | Unit test: format regex |
| REQ-HR-003 (1-based line numbers) | Task 2 | Unit test: verify numbering |
| REQ-HR-004 (backward compatible read) | Task 2 | Regression test |
| REQ-HR-005 (blank lines get hash) | Task 2 | Unit test: blank line |
| REQ-HR-006 (absolute numbering with range) | Task 2 | Unit test: start_line/end_line |
| REQ-HE-001 (hash_validate parameter) | Task 3 | Integration test |
| REQ-HE-002 (hash_ref format) | Task 3 | Unit test: format validation |
| REQ-HE-003 (hash comparison logic) | Task 3 | Integration test: match/mismatch |
| REQ-HE-004 (mismatch error format) | Task 3 | Integration test: error content |
| REQ-HE-005 (line out of range) | Task 3 | Integration test: truncated file |
| REQ-HE-006 (backward compatible edit) | Task 3 | Regression test |
| REQ-HE-007 (hash_ref required when validate on) | Task 3 | Unit test: missing hash_ref |
| REQ-BC-001 (zero regressions) | Task 4 | Full test suite |
| REQ-BC-002 (hash_ref optional at schema) | Task 4 | Unit test: schema validation |
| REQ-ERR-001 (collision accepted) | Task 4 | Unit test: collision scenario |
| REQ-ERR-002 (file not found unchanged) | Task 4 | Unit test: nonexistent file |
| REQ-NF-001 (≤10ms hash overhead) | Task 2 | Performance test: 10K lines |
| REQ-NF-002 (fixed-width line numbers) | Task 2 | Inspection: output format |
| REQ-NF-003 (separate hash package) | Task 1 | Inspection: package layout |

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-HR-001 (hash-tagged format) | Unit test: 3-line file → verify `{line}#{hash}| {content}` | Task 2 |
| AC-HR-002 (absolute line numbering with range) | Unit test: `start_line: 10, end_line: 15` → verify lines 10–15 | Task 2 |
| AC-HR-003 (backward compatible read) | Regression test: `read_file` without `hash_tag` → identical output | Task 2 |
| AC-HR-004 (blank line handling) | Unit test: file with blank line → hash tag prefix on blank line | Task 2 |
| AC-HC-001 (hash determinism) | Unit test: hash same content twice → identical output | Task 1 |
| AC-HC-002 (newline exclusion) | Unit test: "abc" and "abc\n" → same hash | Task 1 |
| AC-HE-001 (matching hash → edit applied) | Integration test: read with hash → edit with matching hash → success | Task 3, Task 5 |
| AC-HE-002 (mismatched hash → rejected) | Integration test: read → modify file → edit with stale hash → error with details | Task 3, Task 5 |
| AC-HE-003 (line out of range) | Integration test: read → truncate file → edit with out-of-range hash → error | Task 3, Task 5 |
| AC-HE-004 (backward compatible edit) | Regression test: `edit_file` without `hash_validate` → fuzzy-match unchanged | Task 3 |
| AC-HE-005 (missing hash_ref rejected) | Unit test: `hash_validate: true` + no `hash_ref` → error | Task 3 |
| AC-BC-001 (no test regressions) | Full test suite → zero failures | Task 4 |
| AC-BC-002 (hash_ref optional at schema) | Unit test: `hash_validate: false` + no `hash_ref` → accepted | Task 4 |
| AC-ERR-001 (hash collision accepted) | Unit test: construct collision → edit applied without crash | Task 4 |
| AC-ERR-002 (file not found unchanged) | Unit test: hash-validated edit on nonexistent file → standard error | Task 4 |
| AC-NF-001 (≤10ms hash overhead) | Performance test: 10K-line file → hash time ≤ 10ms | Task 2 |
| AC-NF-002 (fixed-width line numbers) | Inspection: review output → ≥4 char left-padded line numbers | Task 2 |
| AC-NF-003 (separate hash package) | Inspection: verify `internal/hashvalidate/` exists | Task 1 |
