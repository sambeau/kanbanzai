# Specification: Cache Eviction on Entity Delete (FEAT-01KPXH0WSTHAM)

## Related Work

### Prior documents consulted

- **Design: State Store Read Path Performance (P29)** (`work/design/p29-state-store-read-performance.md`) — the parent design for this specification. Section 1.3 ("Component responsibilities") defines the consistency model and eviction requirement. Section 1.5 ("Decisions") states Decision 3: cache misses fall back to filesystem and the cache must not serve stale data for deleted entities.

### Decisions that constrain this specification

- **Decision 1 (design §1.5):** The SQLite cache is the read-acceleration layer for `List()` and `Get()`. Its correctness depends on write-through consistency — every mutation to entity state must be reflected in the cache before the tool call returns.
- **Decision 3 (design §1.5):** Cache miss falls back to filesystem. The cache must never serve stale data for a deleted entity. Any path that permanently removes an entity file from the filesystem MUST evict the corresponding cache row.

### Relationship to prior work

This specification covers the eviction half of the write-through consistency model established in P29. The upsert half (all create and update paths calling `cacheUpsertFromResult`) is already implemented and is out of scope here. This spec defines the invariant that governs current and future deletion paths.

---

## Overview

The SQLite entity cache (`internal/cache`) is maintained as a write-through cache: every create and update operation already upserts the affected row before the tool call returns. This specification covers the complementary eviction requirement: any code path that permanently removes an entity's canonical YAML file from the filesystem MUST also evict the corresponding row from the cache. At the time of writing, one such path exists in `EntityService` — the old-file removal during a slug rename in `UpdateEntity`. Because the entity ID is immutable, that path is already handled by the subsequent `cacheUpsertFromResult` call (the upsert overwrites the old row in place). No additional eviction is needed there. This specification also defines the invariant that any future hard-delete path added to `EntityService` must call `cache.Delete()` before or after removing the file.

---

## Scope

### In scope

- The eviction invariant: any `EntityService` method that permanently removes an entity's canonical YAML file MUST call `cache.Delete(entityType, id)` as part of the same operation.
- The slug-rename path in `UpdateEntity` — confirmation that it satisfies the invariant (via upsert, not eviction) and no additional change is required.
- Behaviour when `cache.Delete()` fails: the error MUST be logged but MUST NOT be surfaced to the caller or cause the parent operation to fail.
- `cache.Delete(entityType, id string) error` — the existing method on `internal/cache.Cache` that is the designated eviction API.

### Explicitly excluded

- Cache warm-up at server start (FEAT-01KPXGZXX8BJZ).
- Cache-first read paths in `Get()` and `List()` (FEAT-01KPXH0F5GFNV).
- Document entity deletion (`DocumentService.DeleteDocument`) — documents use a separate store and do not go through `EntityService` or its cache.
- Status transitions to terminal states (e.g. `done`, `cancelled`) — entities are not removed from the filesystem on status change; `cacheUpsertFromResult` is already called and is sufficient.
- Any changes to `internal/cache` internals beyond confirming `cache.Delete()` is the correct eviction API.

---

## Functional Requirements

**FR-001:** When an `EntityService` method permanently removes an entity's canonical YAML file from the filesystem, it MUST evict the corresponding row from the cache by calling `cache.Delete(entityType, id)` before or after the file removal within the same operation.

**Acceptance criteria:**
- After a method that removes an entity file completes, a subsequent `cache.LookupByID(entityType, id)` MUST return `found = false`.
- The eviction MUST occur in the same operation as the file removal — a caller MUST NOT observe a window where the file is gone but the cache still returns a hit.

---

**FR-002:** A `cache.Delete()` call that returns an error MUST NOT cause the parent `EntityService` operation to fail or return an error to the caller.

**Acceptance criteria:**
- If `cache.Delete()` returns a non-nil error, the parent method returns its normal success result.
- The error from `cache.Delete()` is logged at a non-fatal level (e.g. `log.Printf`).

---

**FR-003:** The slug-rename path in `UpdateEntity` — where the old YAML file is removed after a slug change — MUST satisfy the eviction invariant without a separate `cache.Delete()` call, because the subsequent `cacheUpsertFromResult` call overwrites the existing cache row in place (the primary key `(entity_type, id)` is immutable and unchanged by a slug rename).

**Acceptance criteria:**
- After `UpdateEntity` completes a slug rename, `cache.LookupByID(entityType, id)` returns the new slug, not the old one.
- No stale row with the old slug exists independently in the cache (since the primary key is `(entity_type, id)`, the upsert replaces the single row).

---

**FR-004:** When `EntityService` has no cache configured (`s.cache == nil`), any deletion path MUST complete normally without attempting to call `cache.Delete()`.

**Acceptance criteria:**
- Calling a deletion method on an `EntityService` with no cache set does not panic or return a cache-related error.

---

**FR-005:** Any future method added to `EntityService` that permanently removes an entity's canonical YAML file MUST call `cache.Delete(entityType, id)` as part of the same operation (FR-001 applies prospectively).

**Acceptance criteria:**
- This requirement is verified by code review at the time any such method is introduced.
- No automated test is required for this prospective requirement beyond the invariant tests covering FR-001.

---

## Non-Functional Requirements

**NFR-001:** The eviction call (`cache.Delete`) MUST add no observable latency to the parent operation beyond the SQLite DELETE query itself, which is O(1) by primary key.

**NFR-002:** Eviction MUST NOT require the caller to pass any additional parameters beyond what the deletion path already possesses (entity type and ID are always known at the point of file removal).

---

## Acceptance Criteria (summary)

| Requirement | Verification method |
|---|---|
| FR-001: Eviction on file removal | Unit test: call deletion method, assert `LookupByID` returns `found=false` |
| FR-002: Eviction failure is non-fatal | Unit test: inject failing cache, assert parent operation succeeds and logs error |
| FR-003: Slug rename satisfies invariant via upsert | Unit test: rename slug, assert `LookupByID` returns new slug |
| FR-004: No-cache path is safe | Unit test: nil cache, assert no panic |
| FR-005: Prospective invariant | Code review gate |

---

## Dependencies and Assumptions

- `cache.Delete(entityType, id string) error` already exists in `internal/cache/cache.go` and is the correct eviction API. No new cache methods are required by this specification.
- At the time of writing, no hard-delete method exists on `EntityService` beyond the slug-rename file removal in `UpdateEntity`, which is already handled correctly. FR-001 and FR-005 define the invariant for any path added in the future.
- This specification assumes that entity YAML files are the canonical source of truth and that cache rows are derived from them. The cache is non-authoritative and disposable; eviction correctness is a consistency property, not a durability property.
- FEAT-01KPXH0F5GFNV (cache-first reads) must be active for eviction to have user-visible effect, but this specification's invariants are valid and testable independently of whether the read fast path is enabled.