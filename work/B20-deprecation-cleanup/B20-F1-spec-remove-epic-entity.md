# Specification: Remove Epic Entity

- Feature: FEAT-01KN85HMQP2CX
- Status: draft

---

## Overview

Remove the deprecated `Epic` entity type and all associated code. Epic was
replaced by Plan in Phase 2 and no live state exists.

## Scope

All Go source, tests, and testdata in this repository. No external systems,
no data migration, no user-facing behaviour changes.

## Functional Requirements

- FR-1: The `Epic` struct and all `EpicStatus` constants are deleted from `internal/model/entities.go`.
- FR-2: `EntityKindEpic` is deleted; the string `"epic"` is no longer a valid entity kind.
- FR-3: `CreateEpic` / `CreateEpicInput` are deleted from the service layer and the `entityService` interface.
- FR-4: The `"epic"` and `"epics"` cases are removed from the CLI.
- FR-5: The `EPIC-` prefix is removed from ID allocation, display, and `EntityKindFromPrefix`.
- FR-6: Epic lifecycle, entry state, terminal state, and required-field entries are removed from the validate package.
- FR-7: The epic `case` is removed from: storage field ordering, cache `extractParentRef`, service `RebuildCache`, `HealthCheck`, `parseRecordIdentity`, `recordFromEntity`, `epicFields`, health check ref resolution, and the MCP entity/estimate tool descriptions.
- FR-8: The `EPIC-[A-Za-z0-9]+` pattern is removed from the document-intelligence entity-ref extractor.
- FR-9: The legacy `Epic string` field is removed from the `Feature` struct, along with the `"epic"` parent fallback in `extractParentRefFromState`.
- FR-10: `testdata/entities/epic.yaml` is deleted; `testdata/entities/feature.yaml` is updated to use a Plan ID as parent.
- FR-11: All tests that exist solely to exercise epic behaviour are deleted; tests that incidentally use epic as fixture data are updated to use a different entity type.

## Non-Functional Requirements

- NFR-1: `go test ./...` passes after all changes.
- NFR-2: No new `//nolint` or `//noinspection` directives are introduced to suppress errors caused by the removal.

## Acceptance Criteria

- [ ] `grep -r "EntityKindEpic\|EpicStatus\|CreateEpic\|\"epic\"\|EPIC-" --include="*.go" .` returns no matches outside of comments in historical design documents.
- [ ] `go build ./...` succeeds with no errors.
- [ ] `go test ./...` passes with no failures.
- [ ] No `epics/` directory exists under `.kbz/state/`.
- [ ] `testdata/entities/epic.yaml` does not exist.