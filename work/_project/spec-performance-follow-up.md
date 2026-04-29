| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28                     |
| Status | Draft                          |
| Author | Claude Sonnet (via sambeau)    |
| Design | PROJECT/design-design-performance-follow-up |

## Overview

This specification implements the performance follow-up design described in
`work/_project/design-performance-follow-up.md`
(PROJECT/design-design-performance-follow-up).

## Scope

In scope: cache-backed `entityExists()`, pre-loaded cascade data passing,
and tests for both changes. These close two gaps left by P29: the
`entityExists()` method not consulting the SQLite cache, and redundant
entity deserialisation in the finish cascade.

Out of scope: worktree store caching (deferred per P29 Decision 4),
tool-level batch endpoints (separate concern), and any change to the YAML
flat-file canonical store or MCP tool signatures.

## Functional Requirements

- **REQ-001:** `EntityService.entityExists()` SHALL consult
  `Cache.EntityExists()` when the cache is non-nil and warm for the entity
  type, returning the cache's result without touching the filesystem.
- **REQ-002:** When the cache is nil, cold, or returns an error, `entityExists()`
  SHALL fall back to the existing filesystem logic unchanged.
- **REQ-003:** `EntityService` SHALL provide internal pre-loaded variants of
  `CheckAllTasksTerminal` and `CheckAllFeaturesTerminal` that accept
  parent-indexed entity maps instead of calling `List()`.
- **REQ-004:** The lifecycle hook orchestrating the finish cascade SHALL load
  all tasks and features once, build parent-indexed maps, and pass them through
  the `MaybeAutoAdvanceFeature` → `MaybeAutoAdvancePlan` chain using the
  pre-loaded variants.
- **REQ-005:** The existing public signatures of `CheckAllTasksTerminal` and
  `CheckAllFeaturesTerminal` SHALL be preserved for external callers.
- **REQ-006:** If a pre-loaded map does not contain the expected parent key,
  the pre-loaded variant SHALL fall back to calling `List()` for that entity type.

## Non-Functional Requirements

- **REQ-NF-001:** `entityExists()` with a warm cache must complete in under 1ms
  (single indexed SQL query) on hardware matching a MacBook Pro M-series.
- **REQ-NF-002:** The finish cascade must perform at most two entity-type loads
  (one for tasks, one for features) regardless of cascade depth.
- **REQ-NF-003:** `kanbanzai status` wall-clock time must not regress from the
  current baseline of ~0.7s for ~900 entities.
- **REQ-NF-004:** All existing tests in `internal/service/` must continue to
  pass without modification.

## Constraints

- **The YAML flat-file store remains canonical.** The SQLite cache is derived
  and disposable per `workflow-design-basis.md` §7.1. No feature may depend on
  cache presence for correctness.
- **Filesystem fallback must be preserved** for `entityExists()`. The cache
  fast-path is an optimisation, not a replacement.
- **No MCP tool signature changes.** This specification covers internal service
  layer changes only.
- **Worktree store is out of scope** (per P29 Decision 4).
- **Tool-level batching is out of scope** (separate design space).

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a warm cache containing entity FEAT-001, when
  `entityExists("feature", "FEAT-001")` is called, then it returns true
  without calling `filepath.Glob`.
- **AC-002 (REQ-002):** Given a nil cache, when `entityExists("task", "T-001")`
  is called, then it falls back to `filepath.Glob` and returns the correct result.
- **AC-003 (REQ-002):** Given a warm cache whose `EntityExists()` returns an
  error, when `entityExists()` is called, then it falls back to the filesystem
  and returns the correct result.
- **AC-004 (REQ-005):** Given an existing caller of
  `CheckAllTasksTerminal(featureID)`, when called, the method returns the same
  result as before the change using the same public signature.
- **AC-005 (REQ-003, REQ-004):** Given a task completion triggers the finish
  cascade, when `MaybeAutoAdvanceFeature` checks task terminal status, then it
  uses the pre-loaded task map rather than calling `List("task")`.
- **AC-006 (REQ-006):** Given a pre-loaded task map that does not contain the
  parent feature ID, when the pre-loaded variant checks tasks, then it falls
  back to `List("task")` and returns the correct result.
- **AC-007 (REQ-NF-003):** Given a project with ~900 entities, when
  `kanbanzai status` is run, then it completes in under 1 second wall-clock time.
- **AC-008 (REQ-NF-004):** When `go test ./internal/service/...` is run, all
  existing tests pass without modification.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: mock warm cache, call `entityExists`, assert no filesystem access and correct result |
| AC-002 | Test | Unit test: nil cache, call `entityExists`, assert filesystem fallback produces correct result |
| AC-003 | Test | Unit test: cache returns error from `EntityExists()`, assert filesystem fallback |
| AC-004 | Test | Existing `entity_children_test.go` tests pass without modification |
| AC-005 | Test | Unit test: build pre-loaded map, call cached variant, assert `List()` is not invoked |
| AC-006 | Test | Unit test: build incomplete pre-loaded map, call cached variant, assert `List()` fallback |
| AC-007 | Demo | Manual: `time kanbanzai status` on the kanbanzai project (~900 entities), verify <1s |
| AC-008 | Test | Automated: `go test ./internal/service/... -count=1` in CI, verify zero failures |
