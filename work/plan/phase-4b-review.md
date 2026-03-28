# Phase 4b Post-Implementation Review

| Document | Phase 4b Post-Implementation Review          |
|----------|----------------------------------------------|
| Status   | Active                                        |
| Created  | 2026-03-25T20:12:31Z                                    |
| Author   | AI review agent                               |
| Related  | `work/spec/phase-4b-specification.md`         |
|          | `work/plan/phase-4b-implementation-plan.md`   |

---

## 1. Overview

This document records the findings of a post-implementation review of Phase 4b. The review covered:

- Specification conformance (all §16 acceptance criteria)
- Implementation correctness and code quality
- Test coverage adequacy
- CLI and MCP interface documentation
- Workflow document currency

All tests pass with `go test -race ./...`. The implementation is architecturally sound, well-structured, and largely conforms to the specification. However, the review identified two critical defects that mean Phase 4b acceptance criteria are **not yet fully met**, along with several medium and minor items.

---

## 2. Findings Summary

| ID    | Severity | Category           | Title                                                              | AC at risk          |
|-------|----------|--------------------|--------------------------------------------------------------------|---------------------|
| R4B-1 | Critical | Bug                | Incident YAML field ordering non-deterministic in storage          | §16.6               |
| R4B-2 | Critical | Missing impl       | `kbz incident` CLI command group entirely absent                   | §16.6, §14          |
| R4B-3 | Medium   | Bug                | `review_task_output` tool declares `output_files` as `WithObject`  | §16.3               |
| R4B-4 | Medium   | Missing impl       | `decomposition.max_tasks_per_feature` config key not implemented   | §15                 |
| R4B-5 | Medium   | Test gap           | No canonical YAML fixture or round-trip storage test for Incident  | §16.6               |
| R4B-6 | Medium   | Test gap           | `TestServer_ListTools` does not cover Phase 4b tools               | —                   |
| R4B-7 | Minor    | Documentation      | `ServerVersion` and CLI `usageText` still say "Phase 3"            | —                   |
| R4B-8 | Minor    | Documentation      | `feature` and `task` commands absent from top-level CLI help       | —                   |
| R4B-9 | Minor    | Spec defect        | §8.2 step 1 typo: says `needs-rework`, should say `needs-review`   | —                   |
| R4B-10| Minor    | Code quality       | `incidentUpdateTool`/`incidentLinkBugTool` bypass `jsonResult()`   | —                   |
| R4B-11| Minor    | Code quality       | `conflict_tools.go` accesses `Params.Arguments` directly           | —                   |

---

## 3. Detailed Findings

### R4B-1 — Incident YAML field ordering non-deterministic (Critical)

**File:** `internal/storage/entity_store.go` — `fieldOrderForEntityType`

`fieldOrderForEntityType` has no `case` for `"incident"`. When it returns `nil`, the
`orderedKeys` function falls through to the `extras` branch, which sorts all fields
**alphabetically**. Incident YAML files are therefore written in alphabetical order, not
the canonical order defined in §11.2 of the spec.

The spec's canonical field order is:

```
id, slug, title, status, severity, reported_by, detected_at, triaged_at, mitigated_at,
resolved_at, affected_features, linked_bugs, linked_rca, summary, created, created_by, updated
```

Alphabetical order produces a different sequence (e.g. `affected_features` first, `created`
second, `detected_at` third), so the discrepancy is observable on disk.

This violates P1-DEC-008 (deterministic canonical serialisation) and directly blocks AC
§16.6: "Round-trip serialisation of Incident entity in canonical field order produces
identical output."

**Fix required:**

1. Add a `case string(model.EntityKindIncident):` block to `fieldOrderForEntityType` with
   the canonical field slice from §11.2.
2. Add a `testdata/entities/incident.yaml` canonical fixture file.
3. Add an `"incident"` entry to `TestCanonicalYAML_FixturesRoundTrip` in
   `internal/storage/entity_store_test.go`.

---

### R4B-2 — `kbz incident` CLI command group entirely absent (Critical)

**File:** `cmd/kanbanzai/main.go`

The spec §11.6 and §14 require three CLI commands:

```
kbz incident create --slug --title --severity --summary --reported_by
kbz incident list [--status] [--severity]
kbz incident show <incident-id>
```

The `run()` switch has no `case "incident":`. No handler function, no usage text, and no
help documentation exists for these commands. The MCP tools are fully implemented and
tested; the CLI gap is the only missing piece.

**Fix required:**

1. Create `cmd/kanbanzai/incident_cmd.go` with `runIncident`, `runIncidentCreate`,
   `runIncidentList`, and `runIncidentShow` functions.
2. Add `case "incident":` to `run()` in `main.go`.
3. Add `incident` to `usageText`.
4. Add CLI tests in `cmd/kanbanzai/main_test.go`.

---

### R4B-3 — `output_files` MCP parameter declared as `WithObject` (Medium)

**File:** `internal/mcp/review_tools.go`, line ~31

```go
mcp.WithObject("output_files",
    mcp.Description("Paths of files produced or modified by this task (array of strings)"),
),
```

The spec §8.2 defines `output_files` as `[]string`. The MCP tool schema declaration uses
`mcp.WithObject` (a JSON object/map schema), not `mcp.WithArray`. An MCP client inspecting
the tool schema may present this parameter incorrectly. The handler code already handles it
as an array and functions correctly end-to-end, but the schema declaration is wrong and
should be fixed before this interface is used by external clients.

**Fix required:** Change `mcp.WithObject("output_files", ...)` to `mcp.WithArray("output_files", ...)`.

---

### R4B-4 — `decomposition.max_tasks_per_feature` config key missing (Medium)

**File:** `internal/config/config.go`

The spec §15 defines:

```yaml
decomposition:
  max_tasks_per_feature: 20   # soft limit; proposals above this produce a warning
```

No `DecompositionConfig` struct exists and the key is absent from `Config`. The
`decompose_feature` service does not enforce the limit or emit a warning when a proposal
exceeds 20 tasks. For large or complex features this means the orchestrator receives no
signal that a decomposition may be too coarse.

**Fix required:**

1. Add a `DecompositionConfig` struct with a `MaxTasksPerFeature int` field.
2. Add it to `Config` under `decomposition`.
3. Add a default of 20 and validation (non-negative).
4. Emit a warning in `generateProposal` when `len(tasks) > maxTasksPerFeature`.

---

### R4B-5 — No canonical Incident YAML fixture or storage round-trip test (Medium)

**Files:** `testdata/entities/`, `internal/storage/entity_store_test.go`

Every other entity type has a canonical YAML fixture (`plan.yaml`, `epic.yaml`,
`feature.yaml`, `task.yaml`, `bug.yaml`, `decision.yaml`) used by
`TestCanonicalYAML_FixturesRoundTrip`. There is no `incident.yaml`.

Without this fixture:

- There is no authoritative reference for what a canonical Incident file looks like.
- `TestCanonicalYAML_FixturesRoundTrip` does not exercise the Incident serialisation path.
- AC §16.6 ("round-trip serialisation in canonical field order") has no machine-checked proof.

This finding is coupled with R4B-1: fixing the field ordering and adding the fixture together
constitute the full fix.

**Fix required:** Blocked on R4B-1. Once R4B-1 is fixed, add `testdata/entities/incident.yaml`
containing a complete canonical Incident in the correct field order, and add the corresponding
test case to `TestCanonicalYAML_FixturesRoundTrip`.

---

### R4B-6 — `TestServer_ListTools` does not cover Phase 4b tools (Medium)

**File:** `internal/mcp/server_test.go`

`TestServer_ListTools` checks 12 entity tools and 12 Phase 4a tools (24 total) but does not
register or verify the 9 Phase 4b tools:

```
decompose_feature, decompose_review, slice_analysis,
review_task_output, conflict_domain_check,
incident_create, incident_update, incident_list, incident_link_bug
```

The live `NewServer()` exposes all of these; the test server does not. This means there is
no test that would catch a tool accidentally being dropped from the server registration.

**Fix required:** Either extend `TestServer_ListTools` to use a full `NewServer()` instance
(or a helper that registers all production tools) and assert against the complete expected
tool list, or — if the scoped test server is intentional — add a separate
`TestServer_ListTools_Phase4b` that specifically verifies Phase 4b tool registration.

---

### R4B-7 — `ServerVersion` and CLI `usageText` still say "Phase 3" (Minor)

**Files:** `internal/mcp/server.go`, `cmd/kanbanzai/main.go`

```go
// server.go
ServerVersion = "phase-3-dev"

// main.go usageText
Phase 3 workflow kernel CLI.
```

Both are stale after Phase 4b. The version string is returned in the MCP server handshake,
so MCP clients may display incorrect version information.

**Fix required:** Update `ServerVersion` to `"phase-4b"` (or the appropriate version
string for this release point). Update `usageText` to reflect the current development phase.

---

### R4B-8 — `feature` and `task` commands absent from top-level CLI help (Minor)

**File:** `cmd/kanbanzai/main.go` — `usageText`

The `feature` and `task` command groups are implemented and dispatched correctly, but are
not listed in `usageText`. A user running `kbz help` will not discover `kbz feature decompose`
or `kbz task review`. This will be compounded once `incident` is added (R4B-2).

**Fix required:** Add `feature`, `task`, and `incident` to `usageText` with one-line
descriptions consistent with the other entries.

---

### R4B-9 — Spec §8.2 step 1 has a typo: `needs-rework` should be `needs-review` (Minor)

**File:** `work/spec/phase-4b-specification.md`, §8.2

> "Load the task. If status is not `active`, `done`, or `needs-rework`, return an error."

The acceptance criteria §16.3 AC 7 contradicts this:

> "`review_task_output` on a task already in `needs-review` or `done` returns findings
> without triggering a state transition"

The code correctly accepts `needs-review` (consistent with the acceptance criteria). The
spec body has a typo: `needs-rework` should read `needs-review`. The word `needs-rework` in
this sentence is never a valid input — tasks in `needs-rework` must transition to `active`
before re-review is meaningful.

**Fix required:** Correct `needs-rework` to `needs-review` in §8.2 step 1.

---

### R4B-10 — `incidentUpdateTool` and `incidentLinkBugTool` bypass `jsonResult()` (Minor)

**File:** `internal/mcp/incident_tools.go`

`incidentCreateTool` uses the `jsonResult()` helper (consistent with all other tools), but
`incidentUpdateTool` and `incidentLinkBugTool` marshal their responses manually:

```go
data, err := json.Marshal(result)
...
return mcp.NewToolResultText(string(data)), nil
```

This is a minor inconsistency: `jsonResult()` adds the `createResultWithDisplay` wrapper
for create results, but plain `GetResult` and `ListResult` values are passed through
`jsonResult()` in other tools and come out correctly. The manual path is functionally
equivalent here, but introduces maintenance risk if `jsonResult()` ever gains behaviour
(e.g. formatting, error normalisation).

**Fix required:** Refactor `incidentUpdateTool` and `incidentLinkBugTool` to use
`jsonResult(result)`.

---

### R4B-11 — `conflict_tools.go` accesses `request.Params.Arguments` directly (Minor)

**File:** `internal/mcp/conflict_tools.go`

```go
args, ok := request.Params.Arguments.(map[string]any)
```

Every other tool in the codebase uses `request.GetArguments()`. The direct access is
functionally equivalent but inconsistent and couples the code to the internal transport
representation.

**Fix required:** Replace with `request.GetArguments()`.

---

## 4. Items Not Found (Positive Findings)

The following areas were checked and found to be correctly implemented:

- All six Phase 4b feature tracks have MCP tools registered in `server.go`
- `DependencyUnblockingHook` is correctly wired via `CompositeTransitionHook`
- Failure isolation in the hook is proven by `TestDependencyUnblockingHook_FailureIsolation`
- `complete_task` always returns `unblocked_tasks` (empty array, never omitted)
- `not-planned` and `duplicate` terminal states satisfy dependency checks (tests present)
- Incident lifecycle transition table matches spec §11.2 exactly, including the override paths
- `INC` prefix is registered in both `TypePrefix` and `EntityKindFromPrefix`
- `EntityKindIncident` is defined and wired through `ValidateRecord` and `ValidateTransition`
- `rework_reason` field is in the Task schema with `omitempty`, correct canonical field position, and round-trip test
- `rework_reason` is cleared on `needs-rework → active` transition
- `RCA` document type is registered in `AllDocumentTypes`
- `incidents.rca_link_warn_after_days` config key exists with default of 7 and zero-disables behaviour
- `Phase4bHealthChecker` is registered in `server.go`
- `CheckUnlinkedResolvedIncidents` health check is tested against all relevant cases
- `conflict_domain_check` correctly requires ≥2 task IDs
- `work_queue --conflict-check` annotation is implemented and works inline
- `decompose_feature` returns a proposal without writing any tasks
- `decompose_review` detects gaps, oversized tasks, and dependency cycles
- All decompose and review MCP tools have integration tests
- Slice analysis returns name, outcomes, layers, estimate, depends_on, and rationale per slice
- CLI `kbz feature decompose` and `kbz task review` are implemented and wired
- CLI `kbz queue --conflict-check` is implemented and wired
- Phase 1 document store removal (`internal/document/`) is complete (§16.7 fully checked)

---

## 5. Remediation Priority Order

Resolve in this order to unblock AC verification:

1. **R4B-1** — Fix incident field ordering in storage (prerequisite for R4B-5)
2. **R4B-5** — Add incident fixture and round-trip test (depends on R4B-1)
3. **R4B-2** — Implement `kbz incident` CLI command group
4. **R4B-3** — Fix `output_files` tool parameter declaration
5. **R4B-4** — Add `decomposition` config section
6. **R4B-6** — Extend `TestServer_ListTools` to cover Phase 4b tools
7. **R4B-7 / R4B-8** — Update version string and help text
8. **R4B-9** — Fix spec typo
9. **R4B-10 / R4B-11** — Minor code quality tidying

---

## 6. Gate Assessment

The Phase 5 gate condition (all §16 acceptance criteria met, `go test -race ./...` clean, no
blocking health check errors) is **not yet satisfied** due to R4B-1 and R4B-2.

Once R4B-1 through R4B-6 are resolved, all §16 acceptance criteria will be demonstrably met
and the gate can be declared open.