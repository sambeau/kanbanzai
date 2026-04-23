| Field  | Value                                          |
|--------|------------------------------------------------|
| Date   | 2026-04-23                                     |
| Status | Draft                                          |
| Author | sambeau                                        |
| Plan   | P32-doc-intel-classification-pipeline-hardening |

## Related Work

### Prior documents consulted

- `work/research/doc-intel-recurring-issues-investigation.md` — root-cause analysis; Cluster C4 (CLI identity bug) and Cluster C3 (fixed implementation bugs / MCP struct audit follow-on) are the direct inputs for this document
- `work/design/p28-doc-intel-polish-workflow-reliability.md` — P28 introduced the `classification_nudge` and related doc-intel enhancements; the JSON struct fix in that plan is the reference point for the audit scope

### Constraining decisions

- `config.ResolveIdentity(raw string) (string, error)` already exists and is used by `worktree create`, `merge`, `kbz import`, and `checkpoint create`. The CLI `doc register` command is the only registration path that bypasses it.
- The `Classification` struct and `ConceptIntroEntry` struct were fixed in P28 (added `json:"..."` tags alongside `yaml:"..."` tags). The audit checks whether the same class of defect exists elsewhere.
- MCP tool parameters are decoded from JSON via `json.Unmarshal`. Any exported struct field without an explicit `json:` tag relies on Go's default camelCase→CamelCase matching, which fails silently for snake_case parameter names sent by the MCP client.

### Open questions from prior work

None. Both fixes are well-scoped by the investigation. No research gaps remain.

## Problem and Motivation

### Fix 1 — CLI doc register identity resolution

The `kbz doc register` CLI command requires `--by <user>` to be supplied explicitly. When omitted, the `createdBy` variable is passed as an empty string to `docSvc.SubmitDocument`, which rejects it with a `created_by is required` error. However, the MCP `doc(action: "register")` tool succeeds without `created_by` because it calls `config.ResolveIdentity("")`, which reads `.kbz/local.yaml` and falls back to `git config user.name`.

The retrospective documents six consecutive failed attempts to register a document via the CLI before the agent switched to the MCP tool. The failure was purely a missing `ResolveIdentity` call — the identity was resolvable from context throughout.

This asymmetry between CLI and MCP is a friction source that causes agents to avoid the CLI, and causes human users to receive unhelpful errors when their identity is already configured.

### Fix 2 — MCP parameter struct JSON tag audit

MCP tool parameters arrive as JSON and are decoded via `json.Unmarshal`. Go's default JSON decoder maps JSON field names to struct field names by matching case-insensitively — but only for exact or all-lowercase matches. Snake_case JSON keys (e.g. `section_path`) do not match CamelCase Go fields (e.g. `SectionPath`) without an explicit `json:"section_path"` tag.

The `Classification` struct had exactly this bug: the `section_path` JSON parameter was silently deserialized into the zero value because the field was named `SectionPath` with only a `yaml:"section_path"` tag. This caused every classify call to fail silently during the P28 Layer 3 pilot — the struct was populated, but the section path was always empty.

That specific struct was fixed, but the same pattern may exist in other MCP parameter structs. Structs that carry `yaml:` tags are the highest-risk population because they were likely written for YAML state storage first and later reused as JSON deserialization targets. A systematic audit is needed to confirm no other structs carry the same silent vulnerability.

Without the audit, a future MCP parameter refactor could reintroduce the same class of bug, and it would again be invisible until a pilot or production run surfaces it.

## Design

### Fix 1 — CLI doc register identity resolution

One change in `cmd/kanbanzai/doc_cmd.go`, function `runDocRegister`:

After the flag-parsing loop, replace the direct pass-through of `createdBy` with a `config.ResolveIdentity(createdBy)` call:

```
createdBy, err = config.ResolveIdentity(createdBy)
if err != nil {
    return err
}
```

This matches the pattern already used in `runWorktreeCreate`, `runMergeRun`, and `runImport`. The resolved value is then passed to `docSvc.SubmitDocument`. The `--by` flag continues to work as an explicit override; omitting it now resolves from context rather than producing an error.

No changes to `service.SubmitDocument` or any other layer are required. The service already accepts a resolved identity string; this fix simply ensures the CLI always passes one.

### Fix 2 — MCP parameter struct JSON tag audit

The audit has two parts:

**Part A — Manual inspection.** Identify all exported Go structs in the `internal/mcp/` package (and any packages it imports for parameter binding) that:
1. Have one or more `yaml:` struct tags, **and**
2. Are used as the target of `json.Unmarshal` or are populated from `req.Params.Arguments.(map[string]any)` via manual field extraction.

For each identified struct, verify every exported field has an explicit `json:` tag. Add missing tags. Snake_case is the canonical JSON field name convention for MCP parameters (consistent with the existing fixed structs).

**Part B — Regression test.** Add a Go test (in `internal/mcp/` or a dedicated `_test` file) that encodes a representative set of MCP parameter structs to JSON using their `json:` tags and then decodes them, asserting that all fields round-trip correctly. This test will fail if a future struct gains a `yaml:` tag without a corresponding `json:` tag, catching the regression before it reaches a pilot run.

An alternative to the test is a `go vet`-compatible linter check, but the test approach is simpler, requires no external tooling, and is directly runnable in CI via `go test ./internal/mcp/...`.

### Scope boundary

These two fixes share no code paths and have no ordering dependency on each other or on the pipeline enrichment features (FEAT-1, FEAT-2, FEAT-3). They can be implemented as a single small task or two independent tasks. There are no new tool actions, no new MCP response fields, and no changes to the state store schema.

## Alternatives Considered

### Alternative A (Fix 1): Add ResolveIdentity to SubmitDocument at the service layer

Move the `ResolveIdentity` call into `service.SubmitDocument` so that both CLI and MCP paths resolve identity in one place.

**Trade-offs:** Single source of truth for identity resolution. But `SubmitDocument` currently takes a plain string and delegates identity concerns to the caller — consistent with how all other service methods work. Moving resolution into the service would make the service depend on `config`, which is a layer violation.

**Rejected.** The existing pattern (callers resolve, service receives a resolved string) is correct. The fix belongs at the CLI call site.

### Alternative B (Fix 1): Make --by optional with a warning instead of auto-resolving

When `--by` is omitted, print a warning and use `"unknown"` as the identity.

**Trade-offs:** No config dependency. But it produces low-quality state records and degrades the audit trail. The identity is already resolvable from `.kbz/local.yaml` or git config — ignoring it serves no purpose.

**Rejected.** Auto-resolution is strictly better than a fallback to `"unknown"`.

### Alternative C (Fix 2): Use go vet / staticcheck instead of a test

Write a `go vet` analyser or staticcheck rule that flags structs with `yaml:` tags but missing `json:` tags.

**Trade-offs:** Catches the issue at compile/lint time rather than test time. But writing a custom analyser is significantly more work than a table-driven round-trip test, and the project does not currently have custom vet analysers. A test achieves the same safety guarantee with far less effort.

**Rejected** in favour of the simpler test approach.

### Alternative D (Fix 2): Audit only and add no regression protection

Perform the audit, fix any found issues, and rely on code review to catch future regressions.

**Trade-offs:** Zero ongoing maintenance. But the original bug survived code review in P28. A mechanical check is more reliable than reviewer attention for this class of structural defect.

**Rejected.** The audit finds today's bugs; the regression test prevents tomorrow's.

## Decisions

**Decision:** Place the `config.ResolveIdentity` call in `runDocRegister` at the CLI call site, not in the service layer.
**Context:** All other CLI commands that require identity resolution follow the pattern of resolving at the call site and passing a resolved string to the service.
**Rationale:** Preserves the existing layer boundary. The service receives a resolved identity; it does not resolve one.
**Consequences:** The fix is a one-line addition at the CLI level. The MCP path is unchanged. If a new CLI command is added in future that also registers documents, it must also call `ResolveIdentity` — but this is consistent with how all existing commands are written and is documented by example.

---

**Decision:** Use a round-trip Go test (not a custom linter) as the regression guard for the JSON tag audit.
**Context:** The project has no existing custom `go vet` analysers. The test approach is idiomatic and runnable in CI without additional tooling.
**Rationale:** The test provides equivalent safety with a fraction of the implementation cost.
**Consequences:** The regression guard runs at `go test` time, not at compile time. This is acceptable — CI runs tests on every PR.

---

**Decision:** Scope the JSON tag audit to structs in `internal/mcp/` that carry `yaml:` tags.
**Context:** The known failure mode is structs that were written for YAML state storage and later used as JSON deserialization targets. Structs without `yaml:` tags were written for JSON from the start and are lower risk.
**Rationale:** Targeted audit reduces the scope to the highest-risk population without requiring an exhaustive review of every struct in the codebase.
**Consequences:** Structs outside `internal/mcp/` that might have the same pattern are not covered. This is an acceptable residual risk — MCP parameter structs are the only structs that are both YAML-tagged and JSON-decoded in the hot path.
```

Now let me register both documents: