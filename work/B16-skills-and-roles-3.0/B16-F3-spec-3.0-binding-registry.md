# Specification: Binding Registry (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-88PDPE8V (binding-registry)
**Design reference:** `work/design/skills-system-redesign-v2.md` §3.3
**Status:** Draft

---

## Overview

The binding registry defines the schema, storage, loader, validation, and lookup API for `stage-bindings.yaml` — the declarative mapping from workflow stages to the roles, skills, orchestration patterns, prerequisites, and constraints that govern each stage. Each workflow stage has exactly one binding, eliminating ambiguity about what context to assemble for an agent entering a given stage. The registry is the decision table that orchestrators and the context assembly pipeline consult to determine who does what, when, and under what conditions.

---

## Scope

### In scope

- File location and format (`.kbz/stage-bindings.yaml`)
- YAML schema for binding entries, including all fields defined in the design (§3.3)
- One-stage-one-binding invariant
- Stage lookup API: given a stage name, return its binding
- Fallback behavior for references to non-existent roles
- Validation of the binding file against role system and skill system schemas
- Validation that referenced stages correspond to known lifecycle stages
- The `sub_agents` nested structure for multi-agent stages
- The `document_template` nested structure for document-producing stages
- The `prerequisites` structure as a data declaration

### Explicitly excluded

- Prerequisite enforcement at transition time (this is the responsibility of the `entity(action: "transition")` tool; the binding registry only declares prerequisites)
- Context assembly (how bindings are used to construct agent prompts)
- Orchestration execution (how `single-agent` vs `orchestrator-workers` patterns are implemented at runtime)
- Script execution within skills referenced by bindings
- MCP tool filtering at runtime (the binding declares roles which declare tools; actual filtering is a context assembly concern)
- The content of specific stage bindings (the binding values for each stage are a content concern)
- Document template enforcement at stage gates (the binding declares templates; gate checking is a transition tool concern)

---

## Functional Requirements

**FR-001:** The binding registry MUST be stored as a single YAML file at `.kbz/stage-bindings.yaml`. The file MUST contain a top-level `stage_bindings` mapping where each key is a stage name and each value is a binding object.

**Acceptance criteria:**
- A file at `.kbz/stage-bindings.yaml` with a valid `stage_bindings` mapping loads successfully
- A file missing the `stage_bindings` top-level key returns a validation error identifying the missing key
- A file at a different location (e.g., `.kbz/config.yaml`) is not recognized as the binding registry
- A file where `stage_bindings` is not a mapping (e.g., a list) returns a validation error

---

**FR-002:** Each binding entry MUST support the following fields: `description` (string, required), `orchestration` (enum string, required), `roles` (list of strings, required), `skills` (list of strings, required), `document_type` (nullable string, optional), `human_gate` (boolean, required), `prerequisites` (object, optional), `notes` (string, optional), `effort_budget` (string, optional), `max_review_cycles` (integer, optional), `sub_agents` (object, optional), `document_template` (object, optional). No additional top-level fields within a binding entry are permitted.

**Acceptance criteria:**
- A binding entry with all required fields (`description`, `orchestration`, `roles`, `skills`, `human_gate`) and valid types loads successfully
- A binding entry missing `description` returns a validation error naming the missing field and the stage
- A binding entry missing `orchestration` returns a validation error naming the missing field and the stage
- A binding entry missing `roles` returns a validation error naming the missing field and the stage
- A binding entry missing `skills` returns a validation error naming the missing field and the stage
- A binding entry missing `human_gate` returns a validation error naming the missing field and the stage
- A binding entry containing an unrecognised field (e.g., `priority: high`) returns a validation error naming the unknown field and the stage
- A binding entry with only required fields and no optional fields loads successfully

---

**FR-003:** The `orchestration` field MUST be one of two enum values: `single-agent` or `orchestrator-workers`. This field declares the orchestration pattern for the stage.

**Acceptance criteria:**
- `orchestration: single-agent` loads successfully
- `orchestration: orchestrator-workers` loads successfully
- `orchestration: multi-agent` returns a validation error listing the valid enum values
- `orchestration: ""` (empty) returns a validation error

---

**FR-004:** The `roles` field MUST be a non-empty list of strings. Each string MUST conform to the role ID format (lowercase alphanumeric with hyphens, 2–30 characters) as defined by the role system specification.

**Acceptance criteria:**
- A `roles` list with one or more valid role ID strings loads successfully
- An empty `roles` list (`roles: []`) returns a validation error stating roles must be non-empty for the stage
- A `roles` entry that does not match the role ID format returns a validation error identifying the invalid entry and the stage

---

**FR-005:** The `skills` field MUST be a non-empty list of strings. Each string MUST conform to the skill name format (lowercase alphanumeric with hyphens, 2–40 characters) as defined by the skill system specification.

**Acceptance criteria:**
- A `skills` list with one or more valid skill name strings loads successfully
- An empty `skills` list (`skills: []`) returns a validation error stating skills must be non-empty for the stage
- A `skills` entry that does not match the skill name format returns a validation error identifying the invalid entry and the stage

---

**FR-006:** Each stage key in `stage_bindings` MUST correspond to a known workflow stage. Valid stages include the Phase 2 feature lifecycle stages (`designing`, `specifying`, `dev-planning`, `developing`, `reviewing`) and recognized non-lifecycle stages (`researching`, `documenting`, `plan-reviewing`). An unrecognised stage name MUST cause a validation error.

**Acceptance criteria:**
- A binding with key `designing` loads successfully
- A binding with key `specifying` loads successfully
- A binding with key `reviewing` loads successfully
- A binding with key `researching` loads successfully
- A binding with key `plan-reviewing` loads successfully
- A binding with key `nonexistent-stage` returns a validation error identifying the unrecognised stage name and listing valid stages

---

**FR-007:** Each stage MUST have exactly one binding (the one-stage-one-binding invariant). Duplicate stage keys in the YAML file MUST be detected and rejected. The loader MUST NOT silently use the last-writer-wins behavior of standard YAML parsers for duplicate keys.

**Acceptance criteria:**
- A file with unique stage keys loads successfully
- A file containing two entries with the same stage key returns a validation error identifying the duplicated stage name
- The error message explicitly names the duplicate stage rather than silently accepting one of the two definitions

---

**FR-008:** The loader MUST provide a stage lookup function: given a workflow stage name, return the corresponding binding. If no binding exists for the requested stage, the function MUST return an error identifying the missing stage.

**Acceptance criteria:**
- Looking up `reviewing` when a binding for `reviewing` exists returns that binding
- Looking up `designing` when no binding for `designing` exists returns an error containing the stage name
- Looking up an empty string returns an error

---

**FR-009:** The `prerequisites` field, when present, MUST be an object that may contain a `documents` subfield (list of document prerequisite objects) and/or a `tasks` subfield (task prerequisite object). A document prerequisite object MUST contain `type` (string, required) and `status` (string, required). A task prerequisite object MUST contain either `min_count` (integer) or `all_terminal` (boolean), but not both simultaneously. The prerequisites structure is a data declaration only — enforcement is performed by the transition tool, not by the binding registry.

**Acceptance criteria:**
- A `prerequisites` with `documents: [{type: design, status: approved}]` loads successfully
- A `prerequisites` with `tasks: {min_count: 1}` loads successfully
- A `prerequisites` with `tasks: {all_terminal: true}` loads successfully
- A `prerequisites` with `tasks: {min_count: 1, all_terminal: true}` returns a validation error stating only one of `min_count` or `all_terminal` may be specified
- A document prerequisite missing `type` returns a validation error
- A document prerequisite missing `status` returns a validation error
- An empty `prerequisites: {}` loads successfully (no prerequisites for the stage)
- A binding with no `prerequisites` field loads successfully (the field is optional)

---

**FR-010:** The `sub_agents` field, when present, MUST be an object containing: `roles` (list of strings, required within sub_agents), `skills` (list of strings, required within sub_agents), `topology` (enum string: `parallel`, required within sub_agents), and `max_agents` (nullable integer, optional). `sub_agents` MUST only be present when `orchestration` is `orchestrator-workers`. A binding with `orchestration: single-agent` that includes a `sub_agents` field MUST cause a validation error.

**Acceptance criteria:**
- A binding with `orchestration: orchestrator-workers` and a valid `sub_agents` object loads successfully
- A `sub_agents` object missing `roles` returns a validation error
- A `sub_agents` object missing `skills` returns a validation error
- A `sub_agents` object missing `topology` returns a validation error
- `sub_agents` with `topology: parallel` loads successfully
- `sub_agents` with `topology: sequential` returns a validation error listing the valid values
- `sub_agents` with `max_agents: 4` loads successfully
- `sub_agents` with `max_agents: null` loads successfully (no cap)
- `sub_agents` without a `max_agents` field loads successfully (defaults to no cap)
- A binding with `orchestration: single-agent` and a `sub_agents` field returns a validation error stating sub_agents is only valid for orchestrator-workers

---

**FR-011:** The `document_template` field, when present, MUST be an object containing: `required_sections` (list of strings, required within document_template), `cross_references` (list of strings, optional), and `acceptance_criteria_format` (string, optional). The `required_sections` list MUST be non-empty.

**Acceptance criteria:**
- A `document_template` with `required_sections: ["Problem Statement", "Requirements"]` loads successfully
- A `document_template` with an empty `required_sections: []` returns a validation error stating required_sections must be non-empty
- A `document_template` missing `required_sections` returns a validation error
- A `document_template` with only `required_sections` (no optional fields) loads successfully
- A `document_template` with all three fields loads successfully

---

**FR-012:** When a binding references a role ID in `roles` or `sub_agents.roles` that does not correspond to an existing role file in `.kbz/roles/`, the loader MUST fall back to the parent role (by removing the last hyphen-delimited segment from the role ID) and log a warning. If no parent role exists either, the loader MUST log a warning but MUST NOT fail — the binding remains valid with the unresolved role reference.

**Acceptance criteria:**
- A binding referencing `reviewer-security` when `reviewer-security.yaml` exists loads without warnings
- A binding referencing `reviewer-security` when `reviewer-security.yaml` does not exist but `reviewer.yaml` exists loads successfully with a warning identifying the missing role and the fallback
- A binding referencing `reviewer-security` when neither `reviewer-security.yaml` nor `reviewer.yaml` exists loads successfully with a warning identifying the unresolved role
- The warning message includes both the referenced role ID and the stage name
- The fallback is determined by removing the last hyphen-delimited segment: `reviewer-security` → `reviewer`, `implementer-go` → `implementer`

---

**FR-013:** The `max_review_cycles` field, when present, MUST be a positive integer (≥ 1). It declares the maximum number of review-rework cycles before escalation to a human checkpoint.

**Acceptance criteria:**
- `max_review_cycles: 3` loads successfully
- `max_review_cycles: 1` loads successfully
- `max_review_cycles: 0` returns a validation error stating the value must be at least 1
- `max_review_cycles: -1` returns a validation error
- A binding without `max_review_cycles` loads successfully (the field is optional)

---

**FR-014:** The `document_type` field, when present and non-null, MUST be a non-empty string identifying the type of document produced during the stage (e.g., `design`, `specification`, `dev-plan`, `report`, `research`). A `null` value or absent field indicates the stage does not produce a document.

**Acceptance criteria:**
- `document_type: design` loads successfully
- `document_type: null` loads successfully
- A binding without `document_type` loads successfully
- `document_type: ""` (empty string) returns a validation error stating document_type must be non-empty when present

---

**FR-015:** The loader MUST report all validation errors found across all binding entries in a single pass rather than stopping at the first error. Each error MUST identify the stage name where the error was found.

**Acceptance criteria:**
- A file with validation errors in two different stage bindings returns errors for both stages in a single response
- Each error message includes the stage name for identification
- A file with no errors loads successfully

---

## Non-Functional Requirements

**NFR-001:** Loading and validating the complete `stage-bindings.yaml` file (including cross-reference validation against existing roles) MUST complete in under 200ms on a standard development machine, assuming up to 15 stage bindings.

**Acceptance criteria:**
- Benchmark tests with a representative `stage-bindings.yaml` containing 10 stage bindings complete within the time bound

---

**NFR-002:** The `stage-bindings.yaml` schema MUST be forward-compatible. Unknown fields within a binding entry MUST be rejected (strict parsing) to prevent silent schema drift.

**Acceptance criteria:**
- A binding entry with a field not in the defined schema returns a validation error naming the unknown field

---

**NFR-003:** The binding registry file MUST be valid YAML that can be authored, reviewed, and diffed by humans without tooling. The schema MUST NOT require features beyond YAML 1.2 scalar types, sequences, and mappings.

**Acceptance criteria:**
- The example `stage-bindings.yaml` in the design document is valid under the specified schema
- No binding file requires YAML anchors, aliases, custom tags, or merge keys

---

**NFR-004:** The stage lookup function MUST be O(1) average time complexity after the initial file load. The loader MUST build an in-memory index keyed by stage name during the loading phase.

**Acceptance criteria:**
- After loading, looking up a stage binding by name does not require iterating over all bindings
- Benchmark tests confirm lookup time does not grow with the number of bindings

---

## Dependencies and Assumptions

1. **Role system spec (FEAT-01KN5-88PCVN4Y):** The binding registry references role IDs in its `roles` and `sub_agents.roles` fields. The role ID format (lowercase alphanumeric with hyphens, 2–30 characters) and the storage location (`.kbz/roles/`) are defined by the role system specification. Role fallback logic (FR-012) depends on the role inheritance convention where parent role IDs are derived by removing the last hyphen-delimited segment.
2. **Skill system spec (FEAT-01KN5-88PDBW85):** The binding registry references skill names in its `skills` and `sub_agents.skills` fields. The skill name format (lowercase alphanumeric with hyphens, 2–40 characters) and the storage location (`.kbz/skills/`) are defined by the skill system specification.
3. **Feature lifecycle stages:** Valid stage names are derived from the Phase 2 feature lifecycle statuses defined in `internal/model/entities.go` (`designing`, `specifying`, `dev-planning`, `developing`, `reviewing`) plus non-lifecycle stages (`researching`, `documenting`, `plan-reviewing`) that exist as recognized workflow activities. The canonical list of valid stage names must be defined as a constant set accessible to the binding registry validator.
4. **YAML parsing:** The project uses `gopkg.in/yaml.v3` for YAML parsing. Strict parsing requires `KnownFields(true)` on the decoder. Duplicate key detection requires using the `yaml.Node` API rather than direct unmarshalling, since the standard `Unmarshal` function silently accepts duplicate keys with last-writer-wins semantics.
5. **Transition tool integration:** The `prerequisites` structure defined in FR-009 is a data declaration consumed by the `entity(action: "transition")` tool. The binding registry loader validates the structure but does not enforce prerequisites — that responsibility lies with the transition tool, which reads the binding registry at transition time.
6. **Document type values:** The `document_type` values (e.g., `design`, `specification`, `dev-plan`, `report`, `research`) correspond to document types already used by the existing document system (`internal/mcp/doc_tool.go`). The binding registry does not define new document types; it references existing ones.