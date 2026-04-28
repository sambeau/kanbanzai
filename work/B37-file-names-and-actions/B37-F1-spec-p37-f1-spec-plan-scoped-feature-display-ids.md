# P37-F1: Plan-scoped Feature Display IDs — Specification

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Date    | 2026-04-27T12:34:44Z                                               |
| Status | approved |
| Author  | sambeau                                                            |
| Feature | FEAT-01KQ7JDSVMP4E                                                 |
| Design  | P37 File Names and Actions — D2: Plan-scoped feature display IDs   |

---

## Overview

This specification defines a display ID layer for Kanbanzai features. Each feature receives a
short sequential identifier (`P{n}-F{m}`) scoped to its parent plan. The canonical
`FEAT-{TSID13}` identifier is preserved for all storage and cross-references. See Problem
Statement below for full background.

---

## Problem Statement

Feature identifiers in Kanbanzai are opaque 13-character TSIDs (e.g. `FEAT-01KMKRQRRX3CC`).
These identifiers cannot be spoken aloud, memorised, or referenced naturally in team
communication. Plan IDs (`P24`) are already short, sequential, and human-friendly; features
have no equivalent. As the project moves toward multi-person use, teams need speakable,
memorable feature identifiers that do not compromise the integrity guarantees of TSIDs for
internal storage and distributed generation.

The solution is a display ID layer: each feature receives a short sequential identifier
(`P24-F3`) scoped to its parent plan. The canonical `FEAT-{TSID13}` identifier is preserved
for all internal state, storage filenames, and cross-references. The display ID is stored on
disk and accepted as input in all entity tools, enabling human-friendly references without
breaking any existing integration.

---

## Requirements

## Functional Requirements

**REQ-001 — Plan counter field**
The Plan state file (`.kbz/state/plans/P{n}-{slug}.yaml`) MUST contain a `next_feature_seq`
field of type integer.

**REQ-002 — Plan counter initialisation**
When a new plan is created, `next_feature_seq` MUST be initialised to `1`.

**REQ-003 — Plan counter increment**
Each time a feature is successfully created under a plan, the plan's `next_feature_seq` MUST
be incremented by exactly 1.

**REQ-004 — Feature display_id field**
The Feature state file (`.kbz/state/features/FEAT-{TSID13}-{slug}.yaml`) MUST contain a
`display_id` field of type string.

**REQ-005 — Feature display_id format**
The `display_id` value MUST match the pattern `P{n}-F{m}` where `{n}` is the integer component
of the parent plan's ID (e.g. `37` for plan `P37-file-names-and-actions`) and `{m}` is the
sequence value allocated to this feature (the value of `next_feature_seq` read from the plan
before the increment).

**REQ-006 — Counter write ordering**
During feature creation, the plan's `next_feature_seq` MUST be written with its incremented
value BEFORE the feature state file is written to disk. This ordering ensures that a process
crash between the two writes leaves a gap in the sequence (the skipped sequence number is
unused but the counter has advanced) rather than producing a duplicate display_id (two features
holding the same `P{n}-F{m}` value). Gaps in the sequence are acceptable; duplicate display_ids
within a plan are not.

**REQ-007 — CreateFeature requires parent plan**
The `CreateFeature` operation MUST fail with a descriptive error message when no parent plan is
supplied. No feature state file may be written in this case.

**REQ-008 — CreateFeature allocates display_id**
When a feature is created, `CreateFeature` MUST execute the following steps in order:

1. Read the current value of `next_feature_seq` from the parent plan state file (call it N).
2. Compute `display_id` as `P{plan-number}-F{N}`.
3. Write the plan state file with `next_feature_seq` set to N+1.
4. Write the feature state file with `display_id` set to `P{plan-number}-F{N}`.

If step 3 fails, `CreateFeature` MUST return an error and MUST NOT write the feature state
file. If step 4 fails, the plan counter has already been incremented; the sequence number N is
considered consumed and MUST NOT be reused.

**REQ-009 — Input resolution: P{n}-F{m} pattern**
The entity resolution layer MUST recognise input matching the pattern `P{n}-F{m}` (where `n`
and `m` are positive integers, e.g. `P37-F1`) and resolve it to the canonical `FEAT-{TSID13}`
ID of the matching feature before dispatching to any entity operation.

**REQ-010 — Resolution is case-insensitive**
`P{n}-F{m}` pattern matching MUST be case-insensitive. The inputs `P37-F1`, `p37-f1`, and
`P37-f1` MUST all resolve to the same feature.

**REQ-011 — All entity tools accept P{n}-F{m}**
The following entity operations MUST accept a `P{n}-F{m}` identifier wherever a feature ID is
accepted as input:

- `entity get` — returns the feature matching the display_id.
- `entity update` — applies updates to the feature matching the display_id.
- `entity transition` — transitions the feature matching the display_id.
- `entity list` — when a `P{n}-F{m}` value is supplied as an ID filter, returns the matching
  feature.

**REQ-012 — MCP output shows display_id**
MCP tool responses that include a feature's identity MUST include the `display_id` field
(`P{n}-F{m}` form) for any feature that has one. The `display_id` MUST be present as a named
field in the response payload; it is the primary human-facing identifier. The TSID-derived
break-hyphen form (`FEAT-01KMR-X1SEQV49`) MAY remain present but MUST NOT be the sole
identifier shown.

**REQ-013 — CLI output shows display_id**
CLI output (table views and detail views) that currently shows feature identifiers MUST display
the `display_id` value (`P{n}-F{m}`) for any feature that has the field populated. The raw TSID
break-hyphen form MUST NOT be shown as the primary identifier when `display_id` is available.

**REQ-014 — Migration: backfill display_id**
A migration operation MUST assign `display_id` values to all existing features that do not have
one. Within each plan, features MUST be assigned sequence numbers in ascending order of their
`created` timestamp (oldest feature receives the lowest sequence number).

**REQ-015 — Migration: set plan counter after backfill**
After backfilling all features under a plan, the plan's `next_feature_seq` MUST be set to
(count of features that received a display_id during backfill) + 1, ensuring that the next
feature creation allocates the next unused sequence number.

---

## Non-Functional Requirements

**REQ-NF-001 — Resolution performance**
Resolution of a `P{n}-F{m}` identifier to its canonical `FEAT-{TSID13}` MUST complete within
100 ms on a repository containing 1,000 features.

**REQ-NF-002 — No duplicate display_ids within a plan**
Within a single plan, no two features MAY share the same `display_id` value. The write ordering
in REQ-006 enforces this in the single-writer case. In the concurrent multi-writer case (two
developers creating features from different Git clones simultaneously), a Git merge conflict on
`next_feature_seq` is the expected and sufficient resolution mechanism; automatic conflict
resolution is not required.

**REQ-NF-003 — Backward compatibility: canonical TSID input**
Canonical `FEAT-{TSID13}` identifiers (e.g. `FEAT-01KMKRQRRX3CC`) MUST continue to work as
input in all entity tools, unchanged from current behaviour.

**REQ-NF-004 — Backward compatibility: break-hyphen TSID input**
The TSID break-hyphen display form (e.g. `FEAT-01KMK-RQRRX3CC`) MUST continue to work as input
in all entity tools, unchanged from current behaviour.

**REQ-NF-005 — No state filename changes**
This feature MUST NOT change the filenames of any existing `.kbz/state/` files. The canonical
filename for a feature state file remains `FEAT-{TSID13}-{slug}.yaml`.

---

## Scope

- The canonical feature identifier is `FEAT-{TSID13}`. The `display_id` is a human-facing
  convenience layer; it does not replace the TSID for storage, cross-references, or distributed
  ID generation.
- The sequence counter is per-plan, not global or per-user.
- Git merge conflict resolution is the expected and sufficient mechanism for handling concurrent
  counter writes by two developers working from separate clones; no automatic conflict resolution
  is required by this specification.
- All 145 existing features already have a populated `parent` field pointing to a plan. The
  migration has no orphan features to handle and need not define behaviour for that case.
- The `display_id` is stored persistently on disk so that the mapping from `P{n}-F{m}` to
  `FEAT-{TSID13}` survives process restarts without requiring a full scan of all feature files
  at startup.
- Sequence numbers are allocated monotonically increasing and are never reused, even if a gap
  arises from a crash between REQ-006 write steps. Gap-filling is not permitted.

---

## Acceptance Criteria

**AC-001 (REQ-001, REQ-002):**
Given a `CreatePlan` call completes successfully,
when the resulting plan state file is read from disk,
then the YAML contains a `next_feature_seq` field with integer value `1`.

**AC-002 (REQ-003):**
Given a plan state file contains `next_feature_seq: N`,
when a feature is successfully created under that plan,
then reading the plan state file from disk shows `next_feature_seq: N+1`.

**AC-003 (REQ-004, REQ-005):**
Given a feature is created under plan `P37-file-names-and-actions` at the moment when the
plan's `next_feature_seq` is `3`,
when the feature state file is read from disk,
then the YAML contains the field `display_id: P37-F3`.

**AC-004 (REQ-006, REQ-NF-002):**
Given a test harness that injects a fault immediately after writing the plan's incremented
`next_feature_seq` but before writing the feature state file,
when the test inspects all feature state files under the plan,
then no feature with `display_id: P{n}-F{N}` exists, and the plan's `next_feature_seq`
equals `N+1`, confirming a sequence gap rather than a duplicate allocation.

**AC-005 (REQ-007):**
Given a `CreateFeature` call that supplies no `parent` plan field,
when the operation executes,
then it returns an error containing a message that identifies the missing plan as the cause,
and no feature state file is written to disk.

**AC-006 (REQ-008):**
Given a plan with `next_feature_seq: 5` (plan number `37`),
when a feature is created under that plan,
then the plan state file shows `next_feature_seq: 6`, the feature state file contains
`display_id: P37-F5`, and both writes are observable on disk before `CreateFeature` returns.

**AC-007 (REQ-009):**
Given a feature with canonical ID `FEAT-01KMKRQRRX3CC` has `display_id: P24-F3`,
when `entity get` is called with id `P24-F3`,
then the response contains the same entity data as calling `entity get` with id
`FEAT-01KMKRQRRX3CC`.

**AC-008 (REQ-010):**
Given the same feature as AC-007,
when `entity get` is called with id `p24-f3` (all lowercase),
then the response is identical to the response for `P24-F3`.

**AC-009 (REQ-011 — entity get):**
Given a feature with `display_id: P37-F1`,
when `entity get` is called with id `P37-F1`,
then the response returns that feature's full state.

**AC-010 (REQ-011 — entity update):**
Given a feature with `display_id: P37-F2`,
when `entity update` is called with id `P37-F2` and a new `summary` value,
then the feature's state file on disk reflects the updated summary.

**AC-011 (REQ-011 — entity transition):**
Given a feature with `display_id: P37-F3` in `ready` status,
when `entity transition` is called with id `P37-F3` and target status `active`,
then the feature's status is updated to `active` in its state file.

**AC-012 (REQ-011 — entity list):**
Given a feature with `display_id: P37-F1`,
when `entity list` is called with `P37-F1` supplied as an ID filter,
then the response contains exactly the feature with `display_id: P37-F1`.

**AC-013 (REQ-012):**
Given a feature with `display_id: P37-F1`,
when any MCP tool response includes a reference to that feature,
then the JSON payload contains a `display_id` field with value `P37-F1`.

**AC-014 (REQ-013):**
Given a feature that has `display_id: P24-F3`,
when the CLI `entity get` command is invoked for that feature,
then the output displays `P24-F3` as the feature identifier and does not display only the
TSID break-hyphen form as the primary identifier.

**AC-015 (REQ-014):**
Given three existing features under plan `P24` with no `display_id` field, with creation
timestamps T1 < T2 < T3,
when the migration operation runs,
then the feature created at T1 has `display_id: P24-F1`, the feature created at T2 has
`display_id: P24-F2`, and the feature created at T3 has `display_id: P24-F3`.

**AC-016 (REQ-015):**
Given the migration scenario from AC-015 where 3 features were backfilled under plan `P24`,
when the plan state file is read after migration,
then it contains `next_feature_seq: 4`.

**AC-017 (REQ-NF-001):**
Given a repository fixture containing 1,000 features distributed across multiple plans,
when `entity get P37-F1` is called,
then the response is returned in 100 ms or less (measured wall-clock time, excluding I/O
startup).

**AC-018 (REQ-NF-003):**
Given a feature with canonical ID `FEAT-01KMKRQRRX3CC` and `display_id: P24-F3`,
when `entity get FEAT-01KMKRQRRX3CC` is called,
then the response is identical to calling `entity get P24-F3`.

**AC-019 (REQ-NF-004):**
Given the same feature,
when `entity get` is called with the TSID break-hyphen form (e.g. `FEAT-01KMK-RQRRX3CC`),
then the response is identical to calling `entity get P24-F3`.

**AC-020 (REQ-NF-005):**
Given the migration from AC-015 has completed,
when the `.kbz/state/features/` directory is listed,
then every filename matches the pattern `FEAT-{TSID13}-{slug}.yaml` and no filenames differ
from their pre-migration values.

---

## Verification Plan

| Criterion | Method             | Description                                                                                                                              |
|-----------|--------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Unit test          | Call `CreatePlan`; read written YAML file; assert `next_feature_seq` field exists with value `1`.                                        |
| AC-002    | Unit test          | Prepare plan stub with `next_feature_seq: N`; call `CreateFeature`; re-read plan YAML; assert value is `N+1`.                           |
| AC-003    | Unit test          | Prepare plan with `next_feature_seq: 3` (plan number 37); call `CreateFeature`; read feature YAML; assert `display_id: P37-F3`.          |
| AC-004    | Unit test          | Inject a fault (panic/error) after plan write and before feature write; scan all feature files; assert none holds the expected display_id; assert plan counter is N+1. |
| AC-005    | Unit test          | Call `CreateFeature` with no `parent` field; assert error is returned containing a message about the missing plan; assert no file written. |
| AC-006    | Unit test          | Prepare plan with `next_feature_seq: 5`; call `CreateFeature`; assert plan shows `6` and feature shows `P{n}-F5` before the call returns. |
| AC-007    | Integration test   | Load fixture with a known `display_id: P24-F3` → `FEAT-01KMKRQRRX3CC` mapping; call `entity get P24-F3`; assert response equals `entity get FEAT-01KMKRQRRX3CC`. |
| AC-008    | Integration test   | Same fixture; call `entity get p24-f3`; assert response equals `entity get P24-F3`.                                                      |
| AC-009    | Integration test   | Call `entity get P37-F1` against a fixture; assert feature data returned.                                                                |
| AC-010    | Integration test   | Call `entity update P37-F2` with a new summary; read state file from disk; assert summary field updated.                                  |
| AC-011    | Integration test   | Call `entity transition P37-F3` to `active`; read state file; assert status is `active`.                                                 |
| AC-012    | Integration test   | Call `entity list` with ID filter `P37-F1`; assert exactly one result returned with `display_id: P37-F1`.                                |
| AC-013    | Integration test   | Call any MCP tool that returns a feature; parse JSON response; assert `display_id` key is present with expected `P{n}-F{m}` value.        |
| AC-014    | Integration test   | Run CLI `entity get` for a feature with a `display_id`; capture stdout; assert `P{n}-F{m}` form is present as the displayed identifier.  |
| AC-015    | Migration test     | Prepare fixture: plan P24, three features with no `display_id`, timestamps T1 < T2 < T3; run migration; assert display_ids assigned in timestamp order. |
| AC-016    | Migration test     | Following AC-015 migration; read plan P24 state file; assert `next_feature_seq: 4`.                                                      |
| AC-017    | Performance test   | Load 1,000-feature fixture; time `entity get P37-F1` wall-clock; assert result within 100 ms.                                            |
| AC-018    | Integration test   | Call `entity get FEAT-01KMKRQRRX3CC` (canonical TSID); assert response equals `entity get P24-F3`.                                       |
| AC-019    | Integration test   | Call `entity get` with break-hyphen TSID form; assert response equals `entity get P24-F3`.                                               |
| AC-020    | Migration test     | After migration; list `.kbz/state/features/`; assert all filenames are unchanged from their pre-migration values.                        |
```

Now let me write the file and register it: