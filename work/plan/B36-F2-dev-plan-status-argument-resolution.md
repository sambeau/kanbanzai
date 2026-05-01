| Field  | Value                                   |
|--------|-----------------------------------------|
| Date   | 2026-04-30                              |
| Status | approved (amended: 2026-04-30)          |
| Author | architect                               |

## Overview

This dev-plan decomposes the specification `work/spec/B36-F2-spec-status-argument-resolution.md`
into eight vertical slices: argument parsing and validation, disambiguation of `<target>`,
file-path-to-document-record resolution, entity ID routing (including the FR-007 fallback probe),
exit code conformance, `kbz doc approve` path resolution, review-driven remediation, and integration
verification.

> **Amended 2026-04-30:** Added Tasks 7–8 to address blocking findings from the B36 batch
> conformance review (see `work/reviews/batch-review-b36-kbz-cli-and-status.md`). Task 7
> implements the FR-007 ResolveNone fallback probe. Task 8 fixes exit codes on entity/plan
> not-found paths (FR-008, FR-016).

## Scope

This plan implements the requirements defined in
work/spec/B36-F2-spec-status-argument-resolution.md (doc ID:
FEAT-01KQ2VFTWD1W2/spec-b36-f2-spec-status-argument-resolution). It covers
argument parsing, disambiguation, file path to document record resolution,
entity ID routing, kbz doc approve path resolution, and integration
verification for kbz status [target] [--format fmt].

It does not cover output rendering (B36-F3 human/plain, B36-F4 JSON),
changes to the project overview (no-target) rendering, multi-target support,
--watch mode, kbz doc register path changes, or title-based resolution
for kbz doc approve.

## Task Breakdown

### Task 1: Define Resolution Interfaces and Contracts

- Description: Create a shared ResolutionResult enum and a
  TargetResolver interface in a new internal/resolution package.
  The interface separates lexical disambiguation (no I/O) from
  concrete resolution (I/O). Codify the lexical-first rule from
  FR-005-FR-008: path heuristic to entity ID pattern match
  to bare plan prefix pattern to fallback entity lookup then
  path lookup. Add a Disambiguate(target string) ResolutionKind
  function and companion unit tests.
- Deliverable: internal/resolution/resolver.go,
  internal/resolution/resolver_test.go.
- Depends on: None.
- Effort: Medium.
- Spec requirement: FR-005, FR-006, FR-007, FR-008, NFR-001.

### Task 2: Wire --format Flag and Argument Validation into kbz status

- Description: Modify runStatus in cmd/kanbanzai/workflow_cmd.go
  to accept a single optional target argument and --format/-f flag.
  Validate the flag value against the allow-list (human, plain, json),
  reject unknown flags (exit 2), reject multiple positional arguments
  (exit 2), and reject invalid --format values (exit 2, with a message
  listing valid values). When target is omitted, preserve the existing
  project-overview behaviour unchanged.
- Deliverable: Modified cmd/kanbanzai/workflow_cmd.go and
  cmd/kanbanzai/workflow_cmd_test.go.
- Depends on: Task 1 (needs the Disambiguate function).
- Effort: Medium.
- Spec requirement: FR-001, FR-002, FR-003, FR-004, FR-019, FR-020, NFR-004.

### Task 3: Implement File Path Resolution

- Description: Implement path resolution per FR-009 to FR-013:
  normalise relative paths (strip ./), check file existence on disk
  (exit 1 if missing), look up the path in the document record store
  via exact repo-relative match. Add a LookupByPath method to
  DocumentService (DEP-001). When no record is found, print an
  unregistered-document response (exit 0) with the file path, not
  registered message, and a suggested kbz doc register command.
  When a record is found, route to the document view, then to the
  owner entity view if present.
- Deliverable: New method on DocumentService, modified
  cmd/kanbanzai/workflow_cmd.go, new cmd/kanbanzai/workflow_cmd_test.go.
- Depends on: Task 1, Task 2.
- Effort: Large.
- Spec requirement: FR-009, FR-010, FR-011, FR-012, FR-013, DEP-001.

### Task 4: Implement Entity ID Routing and Plan Prefix Resolution

- Description: Implement entity ID routing per FR-014 to FR-016:
  resolve display-format IDs to full IDs using existing
  id.StripBreakHyphens, id.NormalizeID, IsFeatureDisplayID, and
  ResolvePrefix. Route by type. For bare plan prefixes (no slug),
  scan the plans directory for a match. Exit 1 with a descriptive
  message when the entity is not found.
- Deliverable: Modified cmd/kanbanzai/workflow_cmd.go and tests.
- Depends on: Task 1, Task 2.
- Effort: Large.
- Spec requirement: FR-008, FR-014, FR-015, FR-016, ASM-002.

### Task 5: Implement kbz doc approve Path Resolution

- Description: Extend runDocApprove in cmd/kanbanzai/doc_cmd.go
  to accept a file path as its first argument using the same lexical
  rule as kbz status (FR-005). When a file path is provided, resolve
  it to a document record via the same exact repo-relative match
  (Task 3). If no record is found, exit 1. If a record is found,
  proceed with approval using the resolved document ID. Preserve
  all existing flags and the existing ID-based flow unchanged.
- Deliverable: Modified cmd/kanbanzai/doc_cmd.go and
  cmd/kanbanzai/doc_cmd_test.go.
- Depends on: Task 1, Task 3 (needs LookupByPath).
- Effort: Medium.
- Spec requirement: FR-021, FR-022, FR-023, FR-024.

### Task 6: Integration and Verification

- Description: Write integration tests covering all exit paths and
  verifying all 24 acceptance criteria. Verify exit codes 0, 1, and
  2 in their respective scenarios. Stub the rendering layer to assert
  routing decisions without depending on B36-F3/B36-F4.
- Deliverable: cmd/kanbanzai/status_cmd_integration_test.go,
  modified cmd/kanbanzai/doc_cmd_test.go.
- Depends on: Task 2, Task 3, Task 4, Task 5, Task 7, Task 8.
- Effort: Large.
- Spec requirement: All ACs; FR-017, FR-018, NFR-003.

### Task 7: Implement FR-007 ResolveNone Fallback Probe

- **Description:** The `ResolveNone` branch in `runStatus` currently returns an
  error immediately. Implement the two-step fallback probe required by FR-007:
  (1) attempt entity-ID lookup via `EntityService.Get`; if that fails,
  (2) attempt path lookup via `DocumentService.ListDocuments` filtered by path.
  Only after both fail should the command return an error. Each probe attempt
  must use the same service calls as their dedicated code paths (Task 3 and
  Task 4) so behaviour is consistent. The existing code comment "try entity
  first, then path, then give up" documents the intent — implement it.
- **Deliverable:** Modified `cmd/kbz/workflow_cmd.go` (`runStatus` ResolveNone
  branch, ~L130-134).
- **Depends on:** Task 2 (needs the runStatus structure), Task 3 (needs the path
  lookup pattern), Task 4 (needs the entity lookup pattern).
- **Effort:** Small.
- **Spec requirement:** FR-007, AC-007.

### Task 8: Fix Exit Codes on Entity/Plan Not-Found

- **Description:** `runStatusEntity` returns `nil` (exit 0) when `entitySvc.Get`
  fails; `runStatusPlanPrefix` returns `nil` (exit 0) when `GetPlan` fails. Both
  must instead return descriptive errors so the caller `runStatus` can exit with
  code 1 as required by FR-016 and FR-008 respectively. The error messages must
  include the unresolved ID/prefix. Update integration tests (Task 6) to assert
  exit code 1 for these paths. Also update the inline comment in
  `runStatusPlanPrefix` that currently reads "Plan not found — informational, exit
  0" to reflect the corrected behaviour.
- **Deliverable:** Modified `cmd/kbz/workflow_cmd.go` (`runStatusEntity` ~L230-238,
  `runStatusPlanPrefix` ~L435-439).
- **Depends on:** Task 4 (these are the entity/plan routing functions).
- **Effort:** Small.
- **Spec requirement:** FR-008, FR-016, AC-006, AC-013.

## Dependency Graph

Task 1 (no dependencies)
Task 2 -> depends on Task 1
Task 3 -> depends on Task 1, Task 2
Task 4 -> depends on Task 1, Task 2
Task 5 -> depends on Task 1, Task 3
Task 7 -> depends on Task 2, Task 3, Task 4
Task 8 -> depends on Task 4
Task 6 -> depends on Task 2, Task 3, Task 4, Task 5, Task 7, Task 8

Parallel groups: [Task 3, Task 4], [Task 7, Task 8]
Critical path: Task 1 -> Task 2 -> Task 4 -> Task 7 -> Task 6

## Risk Assessment

### Risk: LookupByPath Requires O(n) Scan

- Probability: Medium.
- Impact: Medium. Document count is typically small, but a linear scan
  on every kbz status invocation is suboptimal.
- Mitigation: Accept the linear scan for now; document count is low
  enough. The path to ID lookup can be cached in a follow-up if needed.
- Affected tasks: Task 3, Task 5.

### Risk: Display ID Resolution Ambiguity

- Probability: Low.
- Impact: Medium. ResolvePrefix can return ambiguous results.
- Mitigation: TSID has 13 characters of entropy; ambiguous matches
  are astronomically unlikely. ResolvePrefix already errors on ambiguity.
- Affected tasks: Task 4.

### Risk: FR-007 Fallback Probe Service Dependency

- Probability: Low.
- Impact: Medium. The ResolveNone probe needs both entity and document
  services available. If either service initialisation path differs from
  Task 3/Task 4, the probe could behave inconsistently.
- Mitigation: Reuse the same service initialisation patterns from Task 3
  (path lookup) and Task 4 (entity lookup) in the probe code. Integration
  tests in Task 6 verify consistency.
- Affected tasks: Task 7.

### Risk: Exit Code Change May Break Downstream Scripts

- Probability: Low.
- Impact: Low. The previous exit-0-on-not-found behaviour was a bug
  relative to the spec. Any scripts depending on it were relying on
  incorrect behaviour.
- Mitigation: Document the change in the batch completion summary.
  The spec has always required exit 1.
- Affected tasks: Task 8.

### Risk: DocumentService Interface Change Breaks MCP

- Probability: Low.
- Impact: High. Adding LookupByPath to DocumentService could interact
  unexpectedly with MCP code paths.
- Mitigation: LookupByPath is a read-only addition with no side effects.
  Verify MCP status tool still works after the change.
- Affected tasks: Task 3.

### Risk: Integration Test Fixture Complexity

- Probability: Medium.
- Impact: Medium. Integration tests need document records, entity state
  files, and plan directories.
- Mitigation: Reuse existing test helpers. Keep fixtures minimal:
  one doc, one feature, one plan.
- Affected tasks: Task 6.

## Verification Approach

All 24 ACs mapped to Tasks 1-6. Task 1 covers AC-001 through AC-004
(disambiguation unit tests). Task 2 covers AC-016 through AC-018
(flag/usage error tests). Task 3 covers AC-012 (path normalisation unit
test). Task 6 covers all remaining ACs as integration tests (AC-005
through AC-011, AC-013 through AC-015, AC-019 through AC-024).

## Interface Contracts

### ResolutionKind enum (Task 1)

```
type ResolutionKind int
const (
    KindFile            ResolutionKind = iota  // contains / or endswith .md/.txt
    KindEntityID                               // matches known entity ID patterns
    KindBarePlanPrefix                         // matches [A-Z]{1,2}[0-9]+ no slug
    KindAmbiguous                               // needs fallback probe
)
```

### Disambiguate(target string) ResolutionKind (Task 1)

Pure function, no I/O. Consumers call this first to decide resolution strategy.

### LookupByPath(path string) (DocumentResult, error) (Task 3)

New method on DocumentService. Case-sensitive exact repo-relative match.
Returns sentinel error when not found.

### runStatus signature (Task 2)

```
func runStatus(args []string, deps dependencies) error
```
Parses optional target and --format/-f flag. Exit code via error type.

### runDocApprove extended signature (Task 5)

```
func runDocApprove(args []string, deps dependencies) error
```
First arg may be file path (lexical FR-005 rule) or doc ID. Backward compatible.

### Renderer interface (Tasks 3, 4, 6)

```
type Renderer interface {
    RenderOverview(state ProjectOverview) string
    RenderDocumentView(doc DocumentResult) string
    RenderEntityView(entity GetResult) string
    RenderUnregistered(path string) string
    RenderError(err error) string
}
```
Passed via dependencies. Task 6 uses stub renderer for integration tests.

## Traceability Matrix

FR-001: Task 2, AC-019
FR-002: Task 2
FR-003: Task 2, AC-017
FR-004: Task 2, AC-018
FR-005: Task 1, Task 5, AC-001, AC-002
FR-006: Task 1, AC-003, AC-004
FR-007: Task 1, Task 4, Task 7, AC-007
FR-008: Task 1, Task 4, Task 8, AC-005, AC-006
FR-009: Task 3, AC-008
FR-010: Task 3, AC-009
FR-011: Task 3, AC-009
FR-012: Task 3, AC-010, AC-011
FR-013: Task 3, AC-012
FR-014: Task 4, AC-014, AC-015
FR-015: Task 4, AC-014
FR-016: Task 4, Task 8, AC-013
FR-017: Task 3, Task 4, Task 7, Task 8, Task 6, AC-005, AC-009 through AC-011, AC-014, AC-015, AC-019, AC-022, AC-023
FR-018: Task 3, Task 4, Task 7, Task 8, Task 6, AC-006 through AC-008, AC-013, AC-020, AC-021
FR-019: Task 2, AC-016 through AC-018
FR-020: Task 2, AC-016
FR-021: Task 5, AC-021, AC-022, AC-024
FR-022: Task 5, AC-021
FR-023: Task 5, AC-022, AC-024
FR-024: Task 5, AC-023
NFR-001: Task 1, AC-001 through AC-007
NFR-002: Task 3, AC-012
NFR-003: Task 2, Task 3, Task 4, Task 6, AC-006, AC-008, AC-013, AC-016, AC-017, AC-021
NFR-004: Task 2, AC-016
