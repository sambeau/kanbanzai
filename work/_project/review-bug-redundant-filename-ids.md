# Review: BUG-01KQEWZ8YJ1FN — Agent filenames have redundant repeated IDs

**Reviewer:** sambeau
**Date:** 2026-05-07
**Commit:** `81f2ad06`
**Type:** Documentation-only bug fix (skill files)
**Verdict:** **approved**

## Scope

4 skill files updated across 2 locations (user-facing `.agents/` and distributed `internal/kbzinit/`):

| File | Type |
|------|------|
| `.agents/skills/kanbanzai-documents/SKILL.md` | Agent-facing skill |
| `.agents/skills/kanbanzai-getting-started/SKILL.md` | Agent-facing skill |
| `internal/kbzinit/skills/documents/SKILL.md` | Distributed copy (source of truth for fresh installs) |
| `internal/kbzinit/skills/getting-started/SKILL.md` | Distributed copy |

Plus metadata-only updates to 5 task state files.

## Bug description

Agents were generating document filenames that repeated entity ID, feature ID, and document type multiple times (e.g., `B37-F1-dev-plan-p37-f1-dev-plan-plan-scoped-feature-display-ids.md`). Each identifier appeared 2–3 times.

## Fix summary

Updated the canonical filename template in all 4 skill files to:

1. **Entity-scoped template**: `{entity-id}-{type}[-{slug}]` — one template for batch/plan level, one for feature-scoped documents under a batch.
2. **Clear entity ID distinction**: `B{n}` for batches (execution containers), `P{n}` for plans (strategic containers).
3. **Feature scoping**: Features belong to batches, so feature-scoped filenames use the batch prefix (`B24-F3-spec-oauth-flow.md`).
4. **Concrete examples**: Replaced ambiguous P37 examples with clear batch-scoped examples.
5. **Each identifier appears exactly once** — explicitly stated and reinforced with examples.

## Findings

### Blocking: None

All requirements from the bug report are addressed:

| Requirement | Status | Evidence |
|---|---|---|
| Each identifier appears exactly once | ✅ | Template states: "Each identifier appears **exactly once** in the filename" with explicit "do not repeat" guidance |
| Plan ID prefix should be P{n} not B{n} | ✅ | Template distinguishes `B{n}` (batches) from `P{n}` (plans) with definitions |
| Feature ID is sufficient without repeating plan/batch | ✅ | "The feature ID (`F3`) is sufficient because it is batch-scoped — do not repeat the batch ID or document type" |

### Non-blocking: None

## Review unit breakdown

Single review unit (`doc-skill-files`): all 4 skill files changed together. The `.agents/` and `internal/kbzinit/` copies must remain in sync — verified they contain identical content.

## Verdict

**approved** — documentation-only fix, all requirements met, both copies in sync.
