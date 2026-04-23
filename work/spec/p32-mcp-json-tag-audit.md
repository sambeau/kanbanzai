| Field  | Value                                              |
|--------|----------------------------------------------------|
| Date   | 2026-04-23                                         |
| Status | Draft                                              |
| Author | spec-author                                        |
| Plan   | P32-doc-intel-classification-pipeline-hardening    |
| Feature | FEAT-01KPX5CW4R82P mcp-parameter-struct-json-audit |

> This specification implements the design described in
> `work/design/p32-independent-fixes.md` — Fix 2: MCP parameter struct JSON tag audit.

## Problem Statement

MCP tool parameters are transmitted as JSON and decoded via `json.Unmarshal`. Go's default JSON decoder performs case-insensitive matching, but snake_case JSON keys (e.g. `section_path`) do not match CamelCase Go field names (e.g. `SectionPath`) unless an explicit `json:"section_path"` tag is present. When the tag is absent, the field is silently populated with its zero value — no error is returned, no warning is emitted.

Structs in `internal/mcp/` that carry `yaml:` tags are the highest-risk population: they were written for YAML state storage first and later reused as JSON deserialization targets. The `Classification` struct had exactly this defect during the P28 Layer 3 pilot — `SectionPath` was always empty because only a `yaml:"section_path"` tag was present. `Classification` and `ConceptIntroEntry` were fixed in P28.

This specification covers the systematic audit of all remaining MCP parameter structs in `internal/mcp/` that carry `yaml:` tags, the addition of any missing `json:` tags, and the introduction of a round-trip regression test that will catch this class of defect mechanically going forward.

**Included in scope:**
- Exported struct fields in structs within `internal/mcp/` that carry one or more `yaml:` tags and are used as targets of `json.Unmarshal` or populated from MCP `req.Params.Arguments`
- A round-trip regression test in `internal/mcp/` covering representative structs from the audited population

**Explicitly excluded from scope:**
- `Classification` and `ConceptIntroEntry` — already fixed in P28; do not respecify
- Structs outside `internal/mcp/` — considered lower risk (not both YAML-tagged and JSON-decoded)
- Any changes to MCP tool behaviour, response shapes, or state store schema
- Custom `go vet` analysers or staticcheck rules

---

## Requirements

### Functional Requirements

- **REQ-001:** Every exported field in every struct within `internal/mcp/` that (a) carries at least one `yaml:` struct tag and (b) is used as a JSON deserialization target must have an explicit `json:"<snake_case_field_name>"` tag.

- **REQ-002:** The snake_case field name used in each `json:` tag must match the snake_case form of the field's `yaml:` tag name, maintaining consistency between YAML and JSON representations.

- **REQ-003:** A Go test must exist in `internal/mcp/` that encodes a representative set of MCP parameter structs to JSON using their `json:` tags and then decodes that JSON back into a fresh struct instance, asserting that every exported field retains its original non-zero value after the round-trip.

- **REQ-004:** The regression test must be structured so that adding a new struct with `yaml:` tags to `internal/mcp/` without adding `json:` tags to its exported fields causes the test to fail. This may be achieved either by a reflection-based check that enumerates all fields of a registered struct and asserts each has a non-empty `json:` tag, or by an explicit table of structs-under-test that is authoritative and must be updated when new `yaml:`-tagged structs are introduced.

- **REQ-005:** The regression test must be runnable via `go test ./internal/mcp/...` with no additional flags or environment variables.

### Non-Functional Requirements

- **REQ-NF-001:** The audit must produce zero false negatives — every exported field in a `yaml:`-tagged struct that is a JSON deserialization target must be inspected. Manual inspection is acceptable; partial inspection is not.

- **REQ-NF-002:** The regression test must complete in under 1 second under normal `go test` execution (no network calls, no I/O, pure in-memory struct operations).

- **REQ-NF-003:** No existing MCP tool behaviour, wire format, or state file format must change as a result of adding `json:` tags. Adding an explicit `json:` tag to a field that was previously decoded via Go's case-insensitive default matching is a no-op for well-formed callers; this must be verified by confirming the tag value matches what callers already send.

---

## Constraints

- **Do not modify** `Classification` or `ConceptIntroEntry` — these were corrected in P28 and are not in scope.
- **Do not change** the snake_case naming convention already established by the fixed structs. All new `json:` tags must follow the same `snake_case` convention.
- **Do not add** new MCP tool actions, change MCP response payloads, or alter state store schema as part of this fix.
- **Do not introduce** external test dependencies. The regression test must use only the Go standard library (`encoding/json`, `reflect`, `testing`).
- The audit is bounded to `internal/mcp/`. Structs in other packages are out of scope even if they also carry `yaml:` tags.
- The design mandates a Go test approach (not a custom linter). A `go vet` analyser is explicitly out of scope for this feature.

---

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given the complete set of structs in `internal/mcp/` that carry `yaml:` tags and are used as JSON deserialization targets (excluding `Classification` and `ConceptIntroEntry`), when each exported field is inspected, then every such field has an explicit `json:"<snake_case_name>"` tag whose value matches the corresponding `yaml:` tag name.

- **AC-002 (REQ-003):** Given a representative set of MCP parameter structs populated with non-zero values for every exported field, when the struct is marshalled to JSON and then unmarshalled into a zero-value instance of the same type, then the unmarshalled instance is equal to the original (i.e. no field is lost or zeroed).

- **AC-003 (REQ-004):** Given a struct in `internal/mcp/` that carries `yaml:` tags but has one or more exported fields missing `json:` tags, when `go test ./internal/mcp/...` is executed, then the regression test fails with an output that identifies the offending struct and field.

- **AC-004 (REQ-005):** Given a clean checkout of the repository with no environment customisation, when `go test ./internal/mcp/...` is run, then the regression test passes with exit code 0 and completes without error.

- **AC-005 (REQ-NF-002):** Given the full `internal/mcp/` test suite, when `go test -v -run TestJSONTagRoundTrip ./internal/mcp/...` is executed, then the test function completes in under 1 second.

- **AC-006 (REQ-NF-003):** Given the existing MCP integration tests or manual verification, when MCP tool calls that exercise the audited structs are made with their existing snake_case parameter names, then the tool calls produce the same results as before the `json:` tags were added.

---

## Verification Plan

| Criterion | Method      | Description                                                                                                                                           |
|-----------|-------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Inspection  | Code review: enumerate all structs in `internal/mcp/` with `yaml:` tags; verify every exported field has a `json:` tag with the correct snake_case name. |
| AC-002    | Test        | Automated: `TestJSONTagRoundTrip` encodes each representative struct and decodes it; `reflect.DeepEqual` asserts the round-tripped value equals the original. |
| AC-003    | Test        | Automated: introduce a synthetic struct with a missing `json:` tag in the test file (or use the reflection-based field check) and assert the test fails; remove after confirming failure mode. Alternatively, the reflection-based approach in the production test body covers this directly. |
| AC-004    | Test        | CI: `go test ./internal/mcp/...` exits 0 on the main branch after the fix is merged.                                                                  |
| AC-005    | Test        | Timing: run `go test -v -run TestJSONTagRoundTrip -count=1 ./internal/mcp/...` and confirm elapsed time is under 1 second.                            |
| AC-006    | Inspection  | Code review: confirm each new `json:` tag value matches the snake_case name that MCP clients already send (cross-reference tool parameter definitions). |