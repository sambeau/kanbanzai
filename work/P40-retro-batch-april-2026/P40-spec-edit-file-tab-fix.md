# Specification: Fix edit_file Tab Whitespace Corruption

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | Draft                         |
| Author | Spec Author                   |
| Feature | FEAT-01KQGBA8WD0GJ           |
| Bug    | BUG-01KQGB83FXVKV             |
| Design | P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements |

---

## Problem Statement

The `edit_file` tool's fuzzy matching engine normalizes whitespace during
comparison but does not preserve the original indentation style when writing
replacements. When editing tab-indented Go source files, replacement text is
written with spaces instead of tabs, corrupting struct field alignment and
breaking compilation.

This was observed during B39-F1 implementation: three consecutive `edit_file`
calls to add `"write_file"` to a `PrimaryTools` string slice all produced the
same corruption — the replacement line was space-indented while surrounding
lines remained tab-indented, and subsequent struct fields lost their tab
alignment.

This is a correctness bug in the `edit_file` tool's edit mode, related to
Theme 10 (edit_file reliability) and extends the work done in B40-F3 (atomic
multi-edit).

**Scope inclusion:** The `edit_file` tool's edit mode (fuzzy match + replace
path). The write mode (full file overwrite) does not have this issue.

**Scope exclusion:** The `edit_file` tool's worktree resolution (B39-F2).
Multi-edit atomicity (B40-F3). The stale-buffer issue (separate bug).

---

## Requirements

### Functional Requirements

- **REQ-001:** When `edit_file` in edit mode replaces `old_text` in a file
  containing tab-indented lines, the replacement `new_text` MUST preserve the
  indentation style (tabs vs spaces) of the matched `old_text` in the original
  file. If the matched text was tab-indented, the replacement must be
  tab-indented at the same level.

- **REQ-002:** Lines surrounding the replaced text MUST retain their original
  indentation unchanged. A single-line replacement MUST NOT affect the
  indentation of adjacent lines.

### Non-Functional Requirements

- **REQ-NF-001:** The fix MUST NOT change the behaviour of `edit_file` for
  space-indented files. Existing fuzzy matching behaviour for non-tab files is
  preserved.

- **REQ-NF-002:** The fix MUST NOT change the behaviour of `edit_file` in
  write mode (full file overwrite). Only the edit mode is affected.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a Go source file with tab-indented lines, when
  `edit_file` replaces one tab-indented line with new content, then the
  replacement line is tab-indented at the same level and all surrounding lines
  retain their original tab indentation.

- **AC-002 (REQ-002):** Given a Go struct literal with tab-indented fields,
  when `edit_file` replaces one field value, then no other fields lose their
  tab indentation or become space-indented.

- **AC-003 (REQ-NF-001):** Given a space-indented file (e.g., YAML, Markdown),
  when `edit_file` performs a replacement, then behaviour is identical to
  pre-fix — space indentation is preserved as before.

- **AC-004 (REQ-NF-002):** Given a call to `edit_file` in write mode, when the
  call executes, then behaviour is identical to pre-fix — write mode is
  unchanged.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Create tab-indented Go file, replace one line, verify replacement is tab-indented and surroundings unchanged |
| AC-002 | Test | Create Go struct with tab-indented fields, replace one field value, verify other fields retain tabs |
| AC-003 | Test | Run existing space-indented file tests; all pass |
| AC-004 | Test | Run existing write-mode tests; all pass |
