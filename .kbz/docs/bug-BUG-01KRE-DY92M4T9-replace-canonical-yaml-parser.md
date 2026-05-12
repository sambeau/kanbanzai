# BUG-01KRE-DY92M4T9: Replace custom canonical YAML parser with yaml.v3

## Bug Type
Architecture / Technical Debt

## Severity
Medium

## Priority
Medium

## Summary
Entity state files (`.kbz/state/features/*.yaml`, `.kbz/state/batches/*.yaml`, etc.) are parsed by a hand-rolled canonical YAML parser in `internal/storage/entity_store.go`. This parser doesn't support standard YAML idioms — specifically inline `- key: value` list items with continuation fields — causing parse failures on valid YAML.

The rest of the project uses `gopkg.in/yaml.v3` for all other YAML parsing. Only entity state files use the custom parser.

## Observed Behaviour
Three feature files created in the enforcement-mechanisms worktree use a compact list format:

```yaml
overrides:
  - from_status: dev-planning
    reason: "..."
    timestamp: "..."
    to_status: developing
```

The custom parser fails with `unexpected indentation at line 13` when encountering this format. The hotfix (#1) patches the custom parser to handle this specific case, but the fundamental issue remains: a hand-rolled parser will always lag behind standard library compatibility.

## Expected Behaviour
All valid YAML formats for entity state files should parse correctly, including:
- Inline `- key: value` list items with continuation fields
- Quoted strings containing colons
- Multi-line scalars (literal `|` and folded `>` blocks)
- YAML anchors and aliases
- Any other valid YAML construction used in entity files

---

## 1. Plan / Design

### Problem
`internal/storage/entity_store.go` contains `MarshalCanonicalYAML` and `UnmarshalCanonicalYAML` — custom functions that write and read a strict YAML subset line-by-line. The marshal side is intentionally custom (to enforce deterministic field ordering for git-friendly diffs), but the unmarshal side duplicates YAML parsing logic that `gopkg.in/yaml.v3` already handles correctly.

### Proposed Solution
Replace the custom `UnmarshalCanonicalYAML` with a `yaml.v3`-based implementation while keeping the canonical `MarshalCanonicalYAML` for writing.

### Approach
1. **Parse with `yaml.v3.Node`** — Unmarshal into a `yaml.Node` tree, then walk the tree to populate `map[string]any`. This avoids needing typed structs while gaining full YAML compatibility.
2. **Keep `MarshalCanonicalYAML`** — The writer stays custom because it enforces field ordering per entity type and produces deterministic output for git diffs.
3. **No struct changes** — `EntityRecord` continues to use `map[string]any` for fields. The change is purely in the unmarshal function.

### Files Affected
| File | Change |
|------|--------|
| `internal/storage/entity_store.go` | Replace `UnmarshalCanonicalYAML` body; keep `MarshalCanonicalYAML`; keep all helper functions for `Marshal`. |
| `internal/storage/entity_store.go` | Remove `parseMapping`, `parseList`, `parseScalar`, `countIndent`, `splitNonEmptyLines` — no longer needed. |
| `go.mod` / `go.sum` | `gopkg.in/yaml.v3` is already a dependency; no changes needed. |

### Risks
- Regression: entity files written by `MarshalCanonicalYAML` must round-trip through the new unmarshaler without data loss or type changes.
- Edge cases: quoted strings, timestamps, booleans, integers, `null`, and empty maps/lists must all survive the round-trip.

### Design Decisions
- Use `yaml.v3.Node` decoder rather than `yaml.v3.Decoder` with structs, because entity records are schema-less `map[string]any`.
- Convert `yaml.Node` values to Go types explicitly (string, int, float64, bool, nil, map[string]any, []any) matching the existing canonical parser's type output.
- Preserve the existing `FileHash` computation (SHA-256 of raw bytes before unmarshal).

---

## 2. Specification

### Functional Requirements

**REQ-001: Standard YAML compatibility**
The unmarshaler MUST accept all YAML formats that `yaml.v3` considers valid, including:
- Inline list items: `- key: value` followed by continuation fields
- Quoted strings with colons and special chars
- Multi-line literal blocks (`|`)
- Multi-line folded blocks (`>`)
- YAML anchors and aliases
- Mixed indentation styles

**REQ-002: Round-trip fidelity**
For any entity file written by `MarshalCanonicalYAML`, reading it back with the new `UnmarshalCanonicalYAML` MUST produce the same `map[string]any` values (accounting for type representation).

**REQ-003: Same return type**
The return type MUST remain `map[string]any` with the same Go type conventions:
| YAML type | Go type |
|-----------|---------|
| string | `string` |
| integer | `int` |
| float | `float64` |
| boolean | `bool` |
| null | `nil` |
| mapping | `map[string]any` |
| sequence | `[]any` |

**REQ-004: Error reporting**
Parse errors MUST include the line number and descriptive message, matching or exceeding the current parser's error quality.

**REQ-005: Performance**
The new implementation MUST NOT be significantly slower than the current custom parser for the expected workload (< 500 entity files, each < 10KB).

### Acceptance Criteria

- **AC-001**: All ~240 existing feature files parse without error.
- **AC-002**: All batch, plan, task, bug, and decision entity files parse without error.
- **AC-003**: The three previously-failing feature files (those with inline `- key: value` list format) parse without error.
- **AC-004**: Round-trip test: for each entity file, `Marshal → Unmarshal → Marshal` produces identical output (byte-for-byte).
- **AC-005**: `go test ./internal/storage/...` passes with existing tests.
- **AC-006**: Round-trip preserves Go type fidelity: booleans remain booleans, integers remain integers, null remains nil, strings remain strings.
- **AC-007**: Error messages include line numbers for malformed input.
- **AC-008**: `go vet ./internal/storage/...` passes cleanly.

---

## 3. Dev-Plan

### Task 1: Implement `yaml.v3.Node`-to-`map[string]any` converter
- Create `yamlNodeToMap(node *yaml.Node) (any, error)` that recursively converts a `yaml.Node` tree to Go types.
- Handle: DocumentNode, MappingNode, SequenceNode, ScalarNode.
- For ScalarNode: convert to string, int, float64, bool, or nil based on Tag and Value.

**Files:** `internal/storage/yaml_unmarshal.go` (new file)
**Dependencies:** `gopkg.in/yaml.v3` (already a dependency)

### Task 2: Replace `UnmarshalCanonicalYAML` body
- Rewrite `UnmarshalCanonicalYAML` to use `yaml.Unmarshal` into `yaml.Node`, then call the converter.
- Keep the function signature identical: `func UnmarshalCanonicalYAML(content string) (map[string]any, error)`.
- Remove `parseMapping`, `parseList`, `parseScalar`, `countIndent`, `splitNonEmptyLines`.

**Files:** `internal/storage/entity_store.go`
**Depends on:** Task 1

### Task 3: Round-trip and regression tests
- Add table-driven test in `internal/storage/entity_store_test.go` covering:
  - Inline list items: `- key: value` followed by continuation fields
  - Quoted strings with colons
  - Multi-line literal blocks
  - Integers, floats, booleans, null
  - Empty mappings and sequences
  - Round-trip: `Marshal → Unmarshal → Marshal` produces identical output
- Run against all real entity files in the project (golden test).

**Files:** `internal/storage/entity_store_test.go`
**Depends on:** Task 2

### Task 4: Validation against all entity files
- Write a one-shot test or CLI command that loads every entity file in `.kbz/state/` and verifies they all parse.
- This could be integrated into the existing test suite or run as a separate validation step.

**Files:** `internal/storage/entity_store_test.go`
**Depends on:** Task 2

### Task 5: Clean up dead code
- Remove `parseMapping`, `parseList`, `parseScalar`, `countIndent`, `splitNonEmptyLines` functions from `entity_store.go`.
- Verify `go build ./...` still passes.

**Files:** `internal/storage/entity_store.go`
**Depends on:** Task 2

---

## 4. Testing

### Unit Tests

| Test | Description |
|------|-------------|
| `TestUnmarshal_SimpleMapping` | Flat key-value pairs with all scalar types |
| `TestUnmarshal_NestedMapping` | Nested `key:\n  sub: value` |
| `TestUnmarshal_ListOfScalars` | `tags:\n  - a\n  - b` |
| `TestUnmarshal_ListOfMaps` | `items:\n  - key: val\n    k2: v2` |
| `TestUnmarshal_InlineListItem` | `list:\n  - key: val\n    sub: v` (the bug case) |
| `TestUnmarshal_QuotedStrings` | Strings with colons, quotes, special chars |
| `TestUnmarshal_MultiLineLiteral` | `|` block scalars |
| `TestUnmarshal_Empty` | Empty map, empty list |
| `TestUnmarshal_ScalarTypes` | int, float, bool, null, string |
| `TestRoundTrip_AllEntityTypes` | For each entity type (feature/batch/plan/task/bug/decision), marshal → unmarshal → marshal produces identical output |
| `TestRoundTrip_LiveFiles` | Parse every file in `.kbz/state/features/`, `.kbz/state/batches/`, etc. and verify no errors |

### Manual Verification
```bash
# List all features — must include all ~240 without errors
kbz entity list features

# Single get on previously-failing files
kbz entity get bug --id BUG-01KRE-DY92M4T9

# Full test suite
go test ./internal/storage/... -count=1
```

### Regression Risk
The most important regression test: every existing entity file must round-trip without data loss. Since `MarshalCanonicalYAML` stays the same, the main risk is the unmarshal side producing different Go values for the same YAML content. The round-trip test (`Marshal(original) → Unmarshal → Marshal` vs. `Marshal`) catches this.
