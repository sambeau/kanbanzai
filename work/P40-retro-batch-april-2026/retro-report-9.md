# Retro Report 9 — Bug Investigation: Truncated File from Merge Commit

**Date:** 2026-04-29
**Author:** Sam Phillips (sambeau)

## Task

Investigate a suspected damaged merge commit (`85f3e789`), identify what was
truncated, trace the root cause, and restore the damaged file.

## What went well

- **Git archaeology was straightforward.** The damage was reproducible from
  history — `git show` and `git diff` made it easy to pinpoint commit
  `befa7c3a` as the source. The feature branch tip and merge commit were
  byte-for-byte identical, so the merge itself was clean.

- **The scope was contained.** Only one file was damaged (`write-spec/SKILL.md`),
  and the parent commit had the full content intact. Recovery was a simple
  restore-plus-two-edits operation.

- **The `now` tool (the MCP server) was unnecessary for this task.** No entity
  lifecycle manipulation was needed — this was pure Git archaeology and file
  restoration, which stayed entirely in the terminal and editor.

## What didn't go well

- **No guardrails prevent skill file truncation.** The `befa7c3a` commit was
  flagged as `feat(skills): update plan/batch terminology in skill files` but
  its diff showed 310 deletions in a file that should have had 2 insertions.
  The system didn't catch that the diff was disproportionate to the stated
  intent.

- **The truncation happened mid-sentence** — the file ended at `### Vague
  Requirement` with no closing content. This is the kind of structural break
  that a trivial post-commit check (e.g., "does every SKILL.md have a
  Questions section?") would have caught.

- **The commit message described only the intended changes** ("add parent
  batch vocabulary entry"), not the actual diff. This made the commit look
  benign when it was destructive.

## What to improve

- **Skill file integrity check.** A pre-commit or CI hook that validates
  skill files have all required sections (Vocabulary, Anti-Patterns, Procedure,
  Output Format, Examples, Questions) would have caught this immediately.
  A `kbz validate --skills` or similar would be valuable.

- **Diff/stat ratio check.** If a commit's diff shows >100 line deletions in a
  file whose commit message claims a "one vocabulary entry" change, that's a
  red flag. A simple heuristic in a pre-push hook — "if deletions > N×
  insertions, warn" — could surface accidental truncations.

- **The orchestrator (if one was involved) should diff-review before committing.**
  If a sub-agent produced this commit, the orchestrator should have spotted
  the 310-line deletion in the stat output.

## Tools used

| Tool | How |
|------|-----|
| `git log`, `git show`, `git diff` | Tracing the damage through history |
| `wc -l`, `tail`, `head` | Spot-checking file completeness |
| `edit_file` (write mode) | Final file restoration |
| `terminal` | All git operations |

No Kanbanzai MCP tools (`entity`, `status`, `doc`, etc.) were used — this was
a pure Git investigation and file repair task.
