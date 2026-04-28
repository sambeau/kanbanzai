# Specification: Role System (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-88PCVN4Y (role-system)
**Design reference:** `work/design/skills-system-redesign-v2.md` §3.1, §4
**Status:** Draft

---

## Overview

The role system defines the YAML schema, storage, inheritance resolution, validation, and loading mechanism for agent role files. A role establishes an agent's professional identity and domain expertise through a structured vocabulary payload, anti-pattern definitions, and an MCP tool subset declaration. Roles are stored as individual YAML files in `.kbz/roles/` and are loaded and resolved through an extension of the existing `internal/context/` profile system, replacing the current `.kbz/context/roles/` location and schema with the new role-specific structure.

---

## Scope

### In scope

- YAML schema for role definition files
- Storage location and file naming conventions
- Inheritance resolution mechanism (extending `internal/context/resolve.go`)
- Validation rules for role files
- Extension of the `profile` MCP tool to load from the new location and schema
- Backward compatibility contract with the existing `profile(action: "get")` and `profile(action: "list")` tools

### Explicitly excluded

- Context assembly (how roles are combined with skills and injected into agent prompts)
- The content of specific roles (the role taxonomy is a content concern, not a schema concern)
- MCP tool filtering at runtime (the `tools` field is declarative; enforcement is a separate concern)
- Skill system schema (specified separately in the skill system spec)
- Binding registry (specified separately in the binding registry spec)
- Migration tooling for converting existing context profiles to the new role format

---

## Functional Requirements

**FR-001:** A role file MUST be a YAML file stored at `.kbz/roles/{id}.yaml`, where `{id}` is the role's identifier. The `id` field inside the file MUST match the filename (without the `.yaml` extension).

**Acceptance criteria:**
- Loading a role file where the `id` field matches the filename succeeds
- Loading a role file where the `id` field does not match the filename returns a validation error identifying the mismatch
- A role file placed outside `.kbz/roles/` is not discovered by the loader

---

**FR-002:** A role file MUST conform to the following YAML schema with these top-level fields: `id` (string, required), `inherits` (string, optional), `identity` (string, required), `vocabulary` (list of strings, required), `anti_patterns` (list of objects, optional), `tools` (list of strings, optional). No additional top-level fields are permitted.

**Acceptance criteria:**
- A role file with all required fields and valid types loads successfully
- A role file missing `id` returns a validation error naming the missing field
- A role file missing `identity` returns a validation error naming the missing field
- A role file missing `vocabulary` returns a validation error naming the missing field
- A role file containing an unrecognised top-level field returns a validation error naming the unknown field

---

**FR-003:** The `id` field MUST be a lowercase alphanumeric string with hyphens permitted (not at start or end), between 2 and 30 characters. This matches the existing `idRegexp` validation in `internal/context/profile.go`.

**Acceptance criteria:**
- `id: reviewer-security` (valid) loads successfully
- `id: base` (valid, 4 chars) loads successfully
- `id: ab` (valid, 2 chars) loads successfully
- `id: A` (invalid, uppercase) returns a validation error
- `id: -bad` (invalid, leading hyphen) returns a validation error
- `id: this-id-is-way-too-long-for-the-limit` (invalid, >30 chars) returns a validation error

---

**FR-004:** The `identity` field MUST be a non-empty string under 50 tokens. The identity MUST be a real job title (e.g., "Senior application security engineer"), not a flattery-laden description.

**Acceptance criteria:**
- `identity: "Senior application security engineer"` loads successfully
- `identity: ""` (empty) returns a validation error
- An `identity` value exceeding 50 tokens returns a validation error identifying the token count violation
- Token counting uses whitespace-delimited word count as the approximation (each word ≈ 1 token)

---

**FR-005:** The `vocabulary` field MUST be a non-empty list of strings. Each string represents a domain-specific term or concept that routes the agent's knowledge activation.

**Acceptance criteria:**
- A `vocabulary` list with one or more string entries loads successfully
- An empty `vocabulary` list (`vocabulary: []`) returns a validation error stating vocabulary must be non-empty
- A `vocabulary` entry that is not a string returns a validation error

---

**FR-006:** Each entry in the `anti_patterns` list MUST be an object with exactly four string fields: `name` (required), `detect` (required), `because` (required), `resolve` (required). All four fields MUST be non-empty strings.

**Acceptance criteria:**
- An anti-pattern entry with all four fields as non-empty strings loads successfully
- An anti-pattern entry missing `name` returns a validation error identifying the missing field
- An anti-pattern entry missing `because` returns a validation error identifying the missing field
- An anti-pattern entry missing `detect` returns a validation error identifying the missing field
- An anti-pattern entry missing `resolve` returns a validation error identifying the missing field
- An anti-pattern entry with an empty string for any of the four fields returns a validation error
- An anti-pattern entry with an additional unrecognised field returns a validation error

---

**FR-007:** The `tools` field, when present, MUST be a list of strings. Each string represents an MCP tool name. Duplicate entries within the list MUST cause a validation error.

**Acceptance criteria:**
- A `tools` list with distinct string entries loads successfully
- A `tools` list containing duplicate entries (e.g., `[entity, entity]`) returns a validation error identifying the duplicate
- An absent `tools` field loads successfully (the field is optional)
- An empty `tools` list (`tools: []`) loads successfully

---

**FR-008:** The `inherits` field, when present, MUST reference a valid role `id` that can be loaded from the same `.kbz/roles/` directory. The referenced role MUST exist at resolution time.

**Acceptance criteria:**
- A role with `inherits: reviewer` loads successfully when `reviewer.yaml` exists in `.kbz/roles/`
- A role with `inherits: nonexistent` returns an error at resolution time identifying the missing parent role
- A role without an `inherits` field loads successfully (it is a root role)

---

**FR-009:** Inheritance resolution MUST detect and reject circular inheritance chains. If role A inherits from role B and role B inherits from role A (directly or transitively), the resolver MUST return an error identifying the cycle.

**Acceptance criteria:**
- A direct cycle (A inherits B, B inherits A) returns an error containing the word "cycle" and at least one of the role IDs involved
- A transitive cycle (A→B→C→A) returns an error containing the word "cycle"
- A valid chain (A→B→C where C has no `inherits`) resolves successfully

---

**FR-010:** Inheritance resolution MUST merge fields from parent to child as follows:
- `vocabulary`: parent's list followed by child's list (concatenation; parent entries first, child entries appended after)
- `anti_patterns`: parent's list followed by child's list (concatenation; parent entries first, child entries appended after)
- `tools`: union of parent and child lists (no duplicates in the resolved result)
- `identity`: NOT inherited — each role MUST define its own `identity` field; the resolved identity is always the leaf role's value
- `id`: always taken from the leaf role

**Acceptance criteria:**
- A child role with `vocabulary: [a, b]` inheriting from a parent with `vocabulary: [x, y]` resolves to `vocabulary: [x, y, a, b]`
- A child role with `anti_patterns: [child-ap]` inheriting from a parent with `anti_patterns: [parent-ap]` resolves to `anti_patterns: [parent-ap, child-ap]`
- A child role with `tools: [entity, grep]` inheriting from a parent with `tools: [entity, knowledge]` resolves to a tools list containing `entity`, `grep`, and `knowledge` exactly once each
- A child role's `identity` is always used in the resolved output, regardless of the parent's `identity` value
- Multi-level chains (grandparent→parent→child) resolve correctly with the same merge rules applied at each level

---

**FR-011:** The `profile` MCP tool with `action: "list"` MUST return all role files found in `.kbz/roles/`. Each entry in the result MUST include the role's `id`, `inherits` value (if any), and `identity`.

**Acceptance criteria:**
- Calling `profile(action: "list")` when `.kbz/roles/` contains two role files returns a list with two entries
- Each entry includes `id`, `inherits`, and `identity` fields
- Calling `profile(action: "list")` when `.kbz/roles/` does not exist returns an empty list without error

---

**FR-012:** The `profile` MCP tool with `action: "get", id: "<role-id>")` MUST load the role from `.kbz/roles/` and return it. When the `resolved` parameter is true (the default), the tool MUST return the fully inheritance-resolved role. When `resolved` is false, the tool MUST return the raw role definition without inheritance applied.

**Acceptance criteria:**
- `profile(action: "get", id: "reviewer-security")` with `resolved: true` returns the merged vocabulary, anti-patterns, and tools from the full inheritance chain
- `profile(action: "get", id: "reviewer-security")` with `resolved: false` returns only the fields defined in `reviewer-security.yaml`
- `profile(action: "get", id: "nonexistent")` returns an error stating the role was not found

---

**FR-013:** The loader MUST report all validation errors found in a single pass rather than stopping at the first error. The error response MUST include the role file path and all individual validation failures.

**Acceptance criteria:**
- A role file that is missing both `identity` and `vocabulary` returns an error listing both missing fields
- The error message includes the file path or role ID for identification

---

## Non-Functional Requirements

**NFR-001:** Loading and resolving a single role (including a 3-level inheritance chain) MUST complete in under 50ms on a standard development machine, excluding filesystem I/O latency.

**Acceptance criteria:**
- Benchmark tests for role loading and resolution with a 3-level chain complete within the time bound

---

**NFR-002:** The role file schema MUST be forward-compatible. Unknown fields in a role file MUST be rejected (strict parsing) to prevent silent schema drift.

**Acceptance criteria:**
- A role file with a field not in the defined schema (e.g., `extra_field: value`) returns a validation error

---

**NFR-003:** Role files MUST be valid YAML that can be authored and reviewed by humans without tooling. The schema MUST NOT require features beyond YAML 1.2 scalar types, sequences, and mappings.

**Acceptance criteria:**
- All example role files in the design document are valid under the specified schema
- No role file requires YAML anchors, aliases, custom tags, or merge keys

---

**NFR-004:** The role system MUST coexist with the existing context profile system during a migration period. If both `.kbz/roles/{id}.yaml` and `.kbz/context/roles/{id}.yaml` exist for the same `id`, the new `.kbz/roles/` location MUST take precedence.

**Acceptance criteria:**
- When a role exists in both locations, `profile(action: "get")` returns the role from `.kbz/roles/`
- When a role exists only in `.kbz/context/roles/`, `profile(action: "get")` falls back to loading from that location
- When a role exists only in `.kbz/roles/`, it loads successfully

---

## Dependencies and Assumptions

1. **Existing profile infrastructure:** The role system extends `internal/context/profile.go` and `internal/context/resolve.go`. The `ProfileStore`, `ResolveChain`, and `ResolveProfile` functions are the integration points. The new role schema replaces the fields (`description`, `packages`, `conventions`, `architecture`) with the new fields (`identity`, `vocabulary`, `anti_patterns`, `tools`), but the inheritance resolution mechanism (chain walking, cycle detection) is reused.
2. **MCP tool registration:** The `profile` tool is already registered in the MCP server. The tool handler in `internal/mcp/profile_tool.go` must be updated to use the new store location and return the new schema fields.
3. **YAML parsing:** The project uses `gopkg.in/yaml.v3` for YAML parsing. Strict parsing (rejecting unknown fields) requires `KnownFields(true)` on the decoder.
4. **Token counting for identity:** Token counting uses whitespace-delimited word count as a proxy. This is an approximation; exact tokenizer-specific counts are not required.
5. **Skill system and binding registry:** The role `id` values defined here are referenced by the skill system (`roles` field in SKILL.md frontmatter) and the binding registry (`roles` lists in stage bindings). Those specs depend on the `id` format defined in FR-003.