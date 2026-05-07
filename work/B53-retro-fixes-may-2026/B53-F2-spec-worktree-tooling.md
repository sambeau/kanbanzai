# Specification: Worktree File-Writing Tooling

**Feature:** FEAT-01KR003J3ATM2
**Tier:** retro_fix
**Design:** FEAT-01KR003J3ATM2/design-b53-f2-design-worktree-tooling

## Overview

Update implement-task skill and copilot instructions to prefer kanbanzai_edit_file/write_file with entity_id for worktree file writing, replacing terminal-based workarounds.

## Scope

- `.kbz/skills/implement-task/SKILL.md`
- `.github/copilot-instructions.md`
- Documentation-only change

## Functional Requirements

- **FR-001:** `implement-task/SKILL.md` must document `kanbanzai_edit_file(entity_id: ..., mode: ..., ...)` as the primary method for editing files in a worktree
- **FR-002:** `implement-task/SKILL.md` must document `write_file(entity_id: ..., path: ..., content: ...)` as the primary method for creating new files in a worktree
- **FR-003:** Terminal-based workarounds (python3 -c, heredoc) must be deprecated in the skill's guidance
- **FR-004:** `.github/copilot-instructions.md` must note that `kanbanzai_edit_file` and `write_file` accept `entity_id` for worktree targeting

## Acceptance Criteria

- [ ] AC-001: `implement-task/SKILL.md` contains guidance to use kanbanzai_edit_file for worktree edits
- [ ] AC-002: `implement-task/SKILL.md` contains guidance to use write_file for worktree file creation
- [ ] AC-003: Terminal workarounds are marked as deprecated or removed from the guidance
- [ ] AC-004: `.github/copilot-instructions.md` mentions entity_id support for worktree file operations
