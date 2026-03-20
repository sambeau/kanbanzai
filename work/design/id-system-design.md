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
- time-sortable
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

Time-sorted IDs with a random component do not require a central allocator, a lock file, a counter, or any form of coordination between agents. Two agents generating IDs independently, on different machines, in different worktrees, at the same millisecond, will produce different IDs with overwhelming probability.

This is the only ID strategy that is safe by construction for parallel work.

### 3.3 Human-friendliness is a display concern

Humans do not need short IDs. They need short *addressing*. Git commits have 40-character hex SHAs; nobody types all 40 characters. Users type `git show a1b2c3` and Git resolves the prefix. The identifier is long; the interface is short.

The same principle applies here. The canonical ID is moderately long. The CLI, the UI, and conversational references accept any unique prefix. The display layer shows the shortest unique prefix within the current project context.

### 3.4 Right-sized entropy

Kanbanzai is a small workflow database, not a global-scale distributed system. Standard ULIDs provide 80 bits of randomness per millisecond — enough to generate 10²⁴ unique IDs per millisecond without coordination. This is massive overkill for a system that creates a handful of entities per day.

The 48-bit millisecond timestamp already separates any two IDs created more than 1ms apart — which is virtually all of them. The random component only needs to handle the near-impossible case of two agents creating the same entity type in the exact same millisecond. Additionally, the local database provides a collision check as a safety net, allowing immediate retry on the rare collision.

Three characters of Crockford base32 randomness (15 bits = 32,768 values per millisecond) is more than sufficient. Even without a collision check (cross-worktree scenario), the probability of two agents creating the same entity type in the exact same millisecond AND drawing the same 1-in-32,768 random value is negligible.

This produces compact 13-character IDs (10 timestamp + 3 random) that are short enough to recognise at a glance, while remaining collision-safe for any realistic workflow scenario.

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

### 4.2 Features, Bugs, Decisions — Type prefix + compact time-sorted ID

| Property     | Value                                     |
| ------------ | ----------------------------------------- |
| Format       | `{TYPE}-{TSID13}`                         |
| Examples     | `FEAT-01J3K7MXP3RT5`, `BUG-01J4AR7WHN4F2`, `DEC-01J3KABCDE7MX` |
| Type prefix  | `FEAT`, `BUG`, `DEC`                      |
| TSID13       | 13-character Crockford base32: 10-char timestamp + 3-char random |
| Uniqueness   | 48-bit ms timestamp + 15-bit random; local collision check with retry |
| Entropy      | 15 bits per millisecond (32,768 values)   |

Rationale: Features, bugs, and decisions are created by both humans and agents, sometimes in parallel across worktrees. Sequential IDs are unsafe here. The compact time-sorted ID provides collision safety without coordination, time-sortability, and a reasonable length for filenames and references. The type prefix preserves immediate recognition of what kind of entity an ID refers to.

### 4.3 Tasks — Type prefix + compact time-sorted ID (independent)

| Property     | Value                                     |
| ------------ | ----------------------------------------- |
| Format       | `TASK-{TSID13}`                           |
| Example      | `TASK-01J3KZZZBB4KF`                     |
| Uniqueness   | Same as §4.2                              |
| Parent       | Stored as a metadata field (`parent_feature`), not embedded in the ID |

Rationale: Tasks use the same ID format as other entities for consistency. One format means one generator, one parser, one set of tests, one thing to document. 15 bits of randomness per millisecond is adequate even for burst creation of many tasks — and the local collision check provides a safety net.

The parent feature relationship is tracked in entity metadata, not in the ID, because:

- IDs must be stable; relationships are data that can change
- Embedding the parent would make task IDs dependent on feature IDs
- Independent task IDs allow reassignment between features without breaking references

### 4.4 Format summary

| Entity   | Format            | Example                 | Created by | ID length |
| -------- | ----------------- | ----------------------- | ---------- | --------- |
| Epic     | `EPIC-{SLUG}`     | `EPIC-IDS`              | Human      | Variable  |
| Feature  | `FEAT-{TSID13}`   | `FEAT-01J3K7MXP3RT5`   | Human/AI   | 17 chars  |
| Bug      | `BUG-{TSID13}`    | `BUG-01J4AR7WHN4F2`    | Human/AI   | 16 chars  |
| Decision | `DEC-{TSID13}`    | `DEC-01J3KABCDE7MX`    | Human/AI   | 16 chars  |
| Task     | `TASK-{TSID13}`   | `TASK-01J3KZZZBB4KF`   | AI         | 17 chars  |
| Document | `DOC-{TSID13}`    | `DOC-01J3K7MXP3RT5`    | AI         | 16 chars  |

---

## 5. Compact Time-Sorted ID Specification (TSID13)

### 5.1 Structure

A TSID13 is a 13-character string encoded in Crockford base32, composed of two parts:

| Component  | Characters | Bits | Content                                |
| ---------- | ---------- | ---- | -------------------------------------- |
| Timestamp  | 1–10       | 48   | Milliseconds since Unix epoch (unsigned) |
| Random     | 11–13      | 15   | Cryptographic random                   |
| **Total**  | **13**     | **63** | Effectively 65 bits (13 × 5), 2 unused |

The encoding uses 13 characters × 5 bits = 65 bits total. The timestamp occupies 48 bits (with the top 2 bits of the first character always zero for current dates), and the random component occupies 15 bits. This leaves 2 bits unused in the encoding, which are set to zero.

### 5.2 Encoding

- **Alphabet:** Crockford base32 (`0123456789ABCDEFGHJKMNPQRSTVWXYZ`)
- **Canonical case:** Uppercase
- **Case sensitivity:** Case-insensitive for matching and input; normalised to uppercase for storage
- **Sortability:** TSID13 values sort lexicographically by creation time (timestamp is most significant)

### 5.3 Timestamp

The timestamp is the number of milliseconds since the Unix epoch (1970-01-01T00:00:00Z), encoded as the most significant 50 bits of the first 10 Crockford base32 characters (48 bits of timestamp, 2 high bits zero).

This is the same timestamp encoding used by the ULID specification, ensuring compatibility with ULID tooling for the timestamp portion.

The maximum representable timestamp is 2⁴⁸ − 1 milliseconds ≈ year 10889. Overflow is not a concern.

### 5.4 Random component

The random component is 15 bits generated from a cryptographically secure random source (`crypto/rand`). It is encoded as the final 3 Crockford base32 characters.

For each ID generated, a fresh random value is produced. There is no monotonic increment within the same millisecond — this is unnecessary given the low ID generation rate and the local collision check.

### 5.5 Collision safety

The collision model for TSID13:

- Two IDs generated **more than 1ms apart:** guaranteed unique (different timestamps)
- Two IDs generated **within the same millisecond:** 1-in-32,768 collision probability per pair
- **With local collision check:** the system detects the collision and regenerates the random component immediately; effective collision probability is zero within a single process
- **Cross-worktree (no shared check):** collision requires same entity type + same millisecond + same 15-bit random value; negligible probability for any realistic workflow

### 5.6 Go implementation

```
import "crypto/rand"

func generateTSID13() string {
    // 48-bit millisecond timestamp
    now := uint64(time.Now().UnixMilli())

    // 15-bit random
    var rb [2]byte
    crypto_rand.Read(rb[:])
    random := uint16(rb[0])<<8 | uint16(rb[1])
    random &= 0x7FFF // mask to 15 bits

    // Encode as 13 Crockford base32 characters
    // First 10 chars: timestamp (most significant)
    // Last 3 chars: random
    return encodeCrockford(now, random)
}
```

The `encodeCrockford` function encodes the timestamp as 10 characters and the random value as 3 characters using the Crockford base32 alphabet. The implementation must produce uppercase output and accept case-insensitive input.

---

## 6. Display Conventions

### 6.1 The break hyphen

TSID-based IDs are displayed with a break hyphen after the fifth TSID character to create a natural visual boundary between the "short form" and the "tail":

| Canonical (stored)         | Displayed                    |
| -------------------------- | ---------------------------- |
| `FEAT-01J3K7MXP3RT5`      | `FEAT-01J3K-7MXP3RT5`       |
| `BUG-01J4AR7WHN4F2`       | `BUG-01J4A-R7WHN4F2`        |

The break hyphen is a **display convention only**. It is not part of the canonical ID and is not stored in YAML files, filenames, or entity references. The system inserts it when displaying IDs and strips it when accepting input.

**Why position 5:** Five TSID characters encode the most significant 25 bits of the millisecond timestamp, providing ~9.3 hours of resolution. In a typical project cadence (features created hours or days apart), the first five characters are almost always unique. The resulting short form — `FEAT-01J3K` at 10 characters — is comparable in length to a JIRA key.

### 6.2 Prefix matching

The CLI and MCP operations accept any unique prefix of an ID:

- `kbz get FEAT-01J3K` — resolves to the unique feature matching that prefix
- `kbz get FEAT-01J3` — also works if unambiguous
- `kbz get FEAT-01J3K-7MX` — break hyphens in input are accepted and stripped

Resolution rules:

1. Strip the type prefix and all hyphens from the input; this yields the TSID prefix
2. Find all entities of that type whose TSID starts with the given prefix
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

If a new feature with a TSID starting with `01J3KZ...` is later created, the first feature's display lengthens to `FEAT-01J3K7` to remain unambiguous.

In YAML files and git references, the full canonical ID is always used. Short prefixes are a human interface convenience, not a storage format.

---

## 7. Canonical Form and Storage

### 7.1 Canonical ID form

The canonical form of an ID — used in YAML entity files, filenames, cross-references, commit messages, and any persistent storage — does **not** include the break hyphen:

- Epic: `EPIC-IDS`
- Feature: `FEAT-01J3K7MXP3RT5`
- Bug: `BUG-01J4AR7WHN4F2`
- Decision: `DEC-01J3KABCDE7MX`
- Task: `TASK-01J3KZZZBB4KF`
- Document: `DOC-01J3K7MXP3RT5`

### 7.2 Case handling

Canonical form is uppercase. Matching and input parsing are case-insensitive. The system normalises input to uppercase before storage or lookup.

### 7.3 Filename format

Entity files use the format `{CANONICAL-ID}-{slug}.yaml`:

- `EPIC-IDS-id-system-redesign.yaml`
- `FEAT-01J3K7MXP3RT5-profile-editing.yaml`
- `TASK-01J3KZZZBB4KF-implement-id-allocator.yaml`

The canonical ID contains no break hyphen, and the TSID portion is always exactly 13 characters. The parser identifies the slug boundary by:

1. Stripping the `.yaml` extension
2. Extracting the known type prefix (before the first hyphen)
3. For TSID-based types: the next 13 characters after the first hyphen are the TSID; the remainder (after the subsequent hyphen) is the slug
4. For epics: everything after `EPIC-` up to the slug boundary is the epic slug identifier

### 7.4 YAML references

Cross-references between entities use the full canonical ID:

```
id: FEAT-01J3K7MXP3RT5
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

- The `internal/id` package is replaced with a TSID13-based allocator (for FEAT, BUG, DEC, TASK, DOC) and a slug-based allocator (for EPIC)
- The `internal/storage` package filename parser is updated to handle the new 13-character TSID format, with legacy sequential ID recognition retained for robustness
- The `internal/service` entity creation methods use the new allocator
- The `internal/document` ID allocation (currently an in-memory counter closure) is updated to use TSID13 with a `DOC` type prefix
- The `internal/validate` package ID validation rules are updated to accept the new formats
- Prefix resolution is added to the service layer for get/update operations
- A local collision check is added: on ID generation, verify the ID does not already exist in the local store; if it does, regenerate the random component and retry

### 10.3 Document IDs

Document IDs follow the same pattern as entity IDs:

- Format: `DOC-{TSID13}`
- Example: `DOC-01J3K7MXP3RT5`
- The Phase 1 in-memory counter closure is replaced with TSID13 generation

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

1. **Compact time-sorted IDs (TSID13) replace sequential IDs** for all entity types except epics. The format is 10 characters of Crockford base32 timestamp + 3 characters of random (48-bit ms + 15-bit random = 13 characters total). The sequential scan-max-increment strategy is retired.
2. **Epics use human-chosen slugs** rather than any form of generated ID. Uniqueness is enforced at creation and detected at merge.
3. **Tasks have independent IDs** (`TASK-{TSID13}`), not hierarchical IDs (`FEAT-xxx.n`). The parent feature relationship is metadata.
4. **One ID format for all generated entities.** Features, bugs, decisions, tasks, and documents all use the same TSID13 format. Consistency over marginal entropy differentiation.
5. **The break hyphen is display-only.** The canonical stored form does not include it. The display layer adds it at position 5 of the TSID.
6. **Prefix matching is the human interface model.** Humans address entities by shortest unique prefix; the system resolves prefixes to full canonical IDs.
7. **Provenance is metadata, not identity.** Creator information is stored in entity fields and git history, not embedded in IDs.
8. **Local collision check with retry.** On ID generation, the system checks the local store for duplicates and regenerates the random component if a collision is detected. This provides a safety net beyond the statistical guarantees.