# ID System Design

- Status: design
- Purpose: design for a collision-safe, human-friendly ID system suitable for parallel multi-agent work
- Date: 2026-03-20
- Basis:
  - `workflow-design-basis.md` §11 (ID strategy)
  - `phase-1-specification.md` §13 (ID requirements)
  - `phase-1-decision-log.md` P1-DEC-007 (Phase 1 ID strategy, superseded by this design)
- Notes:
  - This design supersedes the Phase 1 sequential ID strategy
  - The Phase 1 implementation used temporary sequential IDs (`FEAT-001`, `BUG-002`, etc.) to prove the system; this document defines the permanent ID system
  - Migration timing is now, while no production state exists

---

## 1. Purpose

This document defines the ID system for all Kanbanzai workflow entities. The system must produce identifiers that are:

- unique without central coordination
- safe for parallel work across multiple agents, worktrees, and branches
- human-friendly at the levels where humans interact
- machine-friendly at the levels where agents interact
- time-sortable where possible
- URL-safe
- stable (never change after creation)

---

## 2. Problem Statement

The Phase 1 ID system uses sequential scan-max-increment allocation:

1. Scan the filesystem for existing entity files
2. Parse the highest numeric ID
3. Return max + 1

This fails in every concurrent scenario:

- **Same worktree, two agents:** Both scan, both see the same max, both allocate the same ID. One silently overwrites the other.
- **Separate worktrees:** Both agents independently allocate the same sequential ID in their own branches. Guaranteed filename collision at merge that cannot be auto-resolved.
- **Unmerged branches:** IDs are not "claimed" until committed, merged, and pushed. Any work on unmerged branches risks collision with no detection until merge time.

The Phase 1 decision log (P1-DEC-007) explicitly acknowledged this as open follow-up work and required that the ID strategy remain replaceable.

---

## 3. Design Principles

### 3.1 IDs identify; metadata describes

An ID's sole job is to uniquely and stably identify an entity. Structural relationships (parent feature, assigned agent), provenance (who created it), and classification (component, severity) belong in entity metadata fields — not in the ID.

Embedding metadata in IDs creates coupling between the identifier and facts that can change. A bug attributed to the frontend turns out to be a backend issue. A task reassigned to a different feature. If the metadata is in the ID, the ID must change, which breaks every reference.

### 3.2 Entropy eliminates coordination

High-entropy random IDs (ULIDs, UUIDs) do not require a central allocator, a lock file, a counter, or any form of coordination between agents. Two agents generating IDs independently, on different machines, in different worktrees, at the same millisecond, will produce different IDs with overwhelming probability.

This is the only ID strategy that is safe by construction for parallel work.

### 3.3 Human-friendliness is a display concern

Humans do not need short IDs. They need short *addressing*. Git commits have 40-character hex SHAs; nobody types all 40 characters. Users type `git show a1b2c3` and Git resolves the prefix. The identifier is long; the interface is short.

The same principle applies here. The canonical ID can be long. The CLI, the UI, and conversational references accept any unique prefix. The display layer shows the shortest unique prefix within the current project context.

### 3.4 The entropy gradient

Not all entities need the same amount of entropy. The entity hierarchy has a natural gradient:

- **Epics** are few, human-created, and human-discussed. Collision risk is negligible.
- **Features and bugs** are moderate in number, created by both humans and agents. They sit on the human-AI boundary.
- **Tasks and decisions** are many, primarily agent-created and agent-managed. Collision risk is highest.

The ID format follows this gradient: human-chosen identifiers at the top, full machine-generated entropy at the bottom.

---

## 4. ID Format by Entity Type

### 4.1 Epics — Human-chosen slugs

| Property     | Value                                     |
| ------------ | ----------------------------------------- |
| Format       | `EPIC-{SLUG}`                             |
| Examples     | `EPIC-IDS`, `EPIC-CONCURRENCY`, `EPIC-CONTEXT-ASSEMBLY` |
| Slug rules   | Uppercase alphanumeric and hyphens, 2–20 characters, no leading/trailing/double hyphens, URL-safe |
| Uniqueness   | Enforced at creation time; health check detects duplicates after merge |
| Entropy      | None — human-chosen, meaningful, memorable |

Rationale: There are few epics in any project (typically 5–30 over a project lifetime). They are created one at a time by humans, in conversation or design documents. The probability of two humans independently choosing the same epic slug on different branches is extremely low, and if it happens, it almost certainly represents the same epic and should be merged manually.

### 4.2 Features, Bugs, Decisions — Type prefix + ULID

| Property     | Value                                     |
| ------------ | ----------------------------------------- |
| Format       | `{TYPE}-{ULID}`                           |
| Examples     | `FEAT-01J3K7MXP3RTE5K9Z2QFHVWA`, `BUG-01J4AR7WHN4F2DX8T12QAZBYC`, `DEC-01J3KABCDEFGH1234567890WX` |
| Type prefix  | `FEAT`, `BUG`, `DEC`                      |
| ULID         | 26-character Crockford base32, standard ULID format |
| Uniqueness   | 128-bit ULID — collision probability is negligible (2⁻¹²⁸) |
| Entropy      | Full — 48-bit millisecond timestamp + 80-bit cryptographic random |

Rationale: Features, bugs, and decisions are created by both humans and agents, sometimes in parallel across worktrees. Sequential IDs are unsafe here. Full ULIDs provide collision safety by construction with no coordination needed. The type prefix preserves immediate recognition of what kind of entity an ID refers to.

### 4.3 Tasks — Type prefix + ULID (independent)

| Property     | Value                                     |
| ------------ | ----------------------------------------- |
| Format       | `TASK-{ULID}`                             |
| Example      | `TASK-01J3KZZZBBBBCCCCDDDDEEEEQQ`        |
| Uniqueness   | 128-bit ULID                              |
| Parent       | Stored as a metadata field (`parent_feature`), not embedded in the ID |

Rationale: Tasks are the most numerous entity type and the most likely to be created concurrently by multiple agents. They require maximum entropy. The parent feature relationship is tracked in entity metadata, not in the ID, because:

- IDs must be stable; relationships are data that can change
- Embedding the parent would make task IDs dependent on feature IDs
- Independent task IDs allow reassignment between features without breaking references

### 4.4 Format summary

| Entity   | Format           | Example                                    | Created by | Entropy  |
| -------- | ---------------- | ------------------------------------------ | ---------- | -------- |
| Epic     | `EPIC-{SLUG}`    | `EPIC-IDS`                                 | Human      | None     |
| Feature  | `FEAT-{ULID}`    | `FEAT-01J3K7MXP3RTE5K9Z2QFHVWA`           | Human/AI   | 128-bit  |
| Bug      | `BUG-{ULID}`     | `BUG-01J4AR7WHN4F2DX8T12QAZBYC`            | Human/AI   | 128-bit  |
| Decision | `DEC-{ULID}`     | `DEC-01J3KABCDEFGH1234567890WX`            | Human/AI   | 128-bit  |
| Task     | `TASK-{ULID}`    | `TASK-01J3KZZZBBBBCCCCDDDDEEEEQQ`          | AI         | 128-bit  |

---

## 5. ULID Specification

All ULID-based IDs use standard ULIDs as defined by the [ULID specification](https://github.com/ulid/spec):

- **Encoding:** Crockford base32 (`0123456789ABCDEFGHJKMNPQRSTVWXYZ`)
- **Length:** 26 characters
- **Structure:** 10-character timestamp (48-bit millisecond Unix epoch) + 16-character randomness (80-bit)
- **Monotonicity:** Within the same millisecond, ULIDs must be monotonically increasing (use a monotonic entropy source)
- **Sortability:** ULIDs sort lexicographically by creation time
- **Case:** Canonical form is uppercase; matching is case-insensitive (Crockford property)

### 5.1 Go implementation

Use `github.com/oklog/ulid/v2` with a monotonic entropy source:

```
entropy := ulid.Monotonic(cryptorand.Reader, 0)
id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
```

The entropy source must be safe for concurrent use. `ulid.Monotonic` wraps a reader with a mutex and provides monotonic ordering within the same millisecond.

---

## 6. Display Conventions

### 6.1 The break hyphen

ULID-based IDs are displayed with a break hyphen after the fifth ULID character to create a natural visual boundary between the "short form" and the "entropy tail":

| Canonical (stored)                     | Displayed                                |
| -------------------------------------- | ---------------------------------------- |
| `FEAT-01J3K7MXP3RTE5K9Z2QFHVWA`      | `FEAT-01J3K-7MXP3RTE5K9Z2QFHVWA`       |
| `BUG-01J4AR7WHN4F2DX8T12QAZBYC`       | `BUG-01J4A-R7WHN4F2DX8T12QAZBYC`        |

The break hyphen is a **display convention only**. It is not part of the canonical ID and is not stored in YAML files, filenames, or entity references. The system inserts it when displaying IDs and strips it when accepting input.

**Why position 5:** Five ULID characters encode the most significant 25 bits of the millisecond timestamp, providing ~9.3 hours of resolution. In a typical project cadence (features created hours or days apart), the first five characters are almost always unique. The resulting short form — `FEAT-01J3K` at 10 characters — is comparable in length to a JIRA key.

### 6.2 Prefix matching

The CLI and MCP operations accept any unique prefix of an ID:

- `kbz get FEAT-01J3K` — resolves to the unique feature matching that prefix
- `kbz get FEAT-01J3` — also works if unambiguous
- `kbz get FEAT-01J3K-7MX` — break hyphens in input are accepted and stripped

Resolution rules:

1. Strip the type prefix and all hyphens from the input; this yields the ULID prefix
2. Find all entities of that type whose ULID starts with the given prefix
3. If exactly one match: use it
4. If multiple matches: return all candidates and ask the user to be more specific
5. If no match: error

### 6.3 Display length

When listing entities, the display layer shows the shortest unique prefix among all entities of the same type within the project. This is computed, not stored, and changes as entities are added:

```
$ kbz list features
FEAT-01J3K  profile-editing       active
FEAT-01J4A  auth-overhaul         planning
FEAT-01J5D  id-redesign           active
```

If a new feature `FEAT-01J3KZ...` is later created, the first feature's display lengthens to `FEAT-01J3K7` to remain unambiguous.

In YAML files and git references, the full canonical ID is always used. Short prefixes are a human interface convenience, not a storage format.

---

## 7. Canonical Form and Storage

### 7.1 Canonical ID form

The canonical form of an ID — used in YAML entity files, filenames, cross-references, commit messages, and any persistent storage — does **not** include the break hyphen:

- Epic: `EPIC-IDS`
- Feature: `FEAT-01J3K7MXP3RTE5K9Z2QFHVWA`
- Bug: `BUG-01J4AR7WHN4F2DX8T12QAZBYC`
- Decision: `DEC-01J3KABCDEFGH1234567890WX`
- Task: `TASK-01J3KZZZBBBBCCCCDDDDEEEEQQ`

### 7.2 Case handling

Canonical form is uppercase. Matching and input parsing are case-insensitive. The system normalises input to uppercase before storage or lookup.

### 7.3 Filename format

Entity files use the format `{CANONICAL-ID}-{slug}.yaml`:

- `EPIC-IDS-id-system-redesign.yaml`
- `FEAT-01J3K7MXP3RTE5K9Z2QFHVWA-profile-editing.yaml`
- `TASK-01J3KZZZBBBBCCCCDDDDEEEEQQ-implement-ulid-allocator.yaml`

The canonical ID contains no break hyphen, and the ULID portion is always exactly 26 characters. The parser identifies the slug boundary by:

1. Stripping the `.yaml` extension
2. Extracting the known type prefix (before the first hyphen)
3. For ULID-based types: the next 26 characters after the first hyphen are the ULID; the remainder (after the subsequent hyphen) is the slug
4. For epics: everything between the first hyphen and the slug-start is the epic slug — which is also the full ID suffix

### 7.4 YAML references

Cross-references between entities use the full canonical ID:

```
id: FEAT-01J3K7MXP3RTE5K9Z2QFHVWA
title: Profile editing
parent_epic: EPIC-IDS
status: active
```

Short prefixes are never stored in YAML. This ensures references are unambiguous regardless of what other entities exist in the project.

---

## 8. Epic Slug Rules

Epic slugs must be:

- 2–20 characters in length
- Composed of uppercase ASCII letters, digits, and hyphens (`A-Z`, `0-9`, `-`)
- URL-safe (no spaces, no special characters)
- No leading hyphen, no trailing hyphen, no consecutive hyphens
- Unique within the project (enforced at creation time)

Input normalisation: the system accepts lowercase input and converts to uppercase. Spaces are rejected (not silently converted to hyphens) to avoid ambiguity.

Epic slugs are chosen by humans to be meaningful and memorable. Good epic slugs are short, descriptive, and unambiguous:

- ✓ `IDS`, `CONCURRENCY`, `CONTEXT`, `AUTH`
- ✗ `EPIC-NUMBER-ONE`, `STUFF`, `A`

---

## 9. Provenance

Creator identity is not embedded in IDs. It is tracked through:

1. **Entity metadata:** A `created_by` field in the entity YAML records who or what created the entity.
2. **Git history:** The commit that introduced the entity file records the author.
3. **Display augmentation:** The CLI and UI can show provenance alongside IDs in listings without polluting the identifier itself.

This separation ensures that IDs are stable even when ownership or responsibility changes, and that provenance information can be corrected without modifying the identifier.

---

## 10. Migration

### 10.1 Timing

Migration from the Phase 1 sequential ID system happens now. There is no production state to migrate — the `.kbz/state/` directory is empty following the Phase 1 cleanup. The sequential ID allocator is replaced entirely.

### 10.2 Code changes

- The `internal/id` package is replaced with a ULID-based allocator (for FEAT, BUG, DEC, TASK) and a slug-based allocator (for EPIC)
- The `internal/storage` package filename parser is updated to handle both the new format and legacy sequential IDs (for robustness, even though no legacy data exists)
- The `internal/service` entity creation methods use the new allocator
- The `internal/document` ID allocation (currently an in-memory counter closure) is updated to use ULIDs with a `DOC` type prefix
- The `internal/validate` package ID validation rules are updated to accept the new formats
- Prefix resolution is added to the service layer for get/update operations

### 10.3 Document IDs

Document IDs follow the same pattern as entity IDs:

- Format: `DOC-{ULID}`
- Example: `DOC-01J3K7MXP3RTE5K9Z2QFHVWA`
- The Phase 1 in-memory counter closure is replaced with ULID generation

---

## 11. Relationship to Concurrency Design

This ID system is a necessary precondition for safe concurrent work, but it is not sufficient on its own. IDs that cannot collide eliminate one class of concurrency bug (duplicate identity), but other concurrency concerns remain:

- **Lost updates:** Two agents reading and modifying the same entity file (read-modify-write without locking)
- **State machine violations:** Two agents transitioning the same entity through incompatible state changes
- **Cache coherence:** Multiple processes with stale derived caches

These concerns are addressed by the concurrency and worktree design (separate document). The ID system's contribution is that it removes ID allocation from the set of operations that require coordination — agents can create entities freely and independently without risk of identity collision.

---

## 12. Design Decisions

This document establishes the following decisions, which should be recorded in the decision log:

1. **ULIDs replace sequential IDs** for all entity types except epics. The sequential scan-max-increment strategy is retired.
2. **Epics use human-chosen slugs** rather than any form of generated ID. Uniqueness is enforced at creation and detected at merge.
3. **Tasks have independent IDs** (`TASK-{ULID}`), not hierarchical IDs (`FEAT-xxx.n`). The parent feature relationship is metadata.
4. **The break hyphen is display-only.** The canonical stored form does not include it. The display layer adds it at position 5 of the ULID.
5. **Prefix matching is the human interface model.** Humans address entities by shortest unique prefix; the system resolves prefixes to full canonical IDs.
6. **Provenance is metadata, not identity.** Creator information is stored in entity fields and git history, not embedded in IDs.
7. **Document IDs follow the same ULID pattern** (`DOC-{ULID}`), replacing the Phase 1 in-memory counter.