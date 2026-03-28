# Documentation Currency Health Check Specification

| Document | Documentation Currency Health Check Specification |
|----------|---------------------------------------------------|
| Status   | Draft                                             |
| Created  | 2026-03-28T16:10:00Z                              |
| Plan     | P10-review-and-doc-currency                       |
| Feature  | FEAT-01KMTJH8Z2AQ5 (doc-currency-health-check)   |
| Design   | work/plan/P10-review-and-doc-currency-plan.md §7  |

---

## 1. Purpose

This specification defines the acceptance criteria for a new health check category that detects stale references in agent-facing documentation. The check has two tiers: tool name validation (detecting references to tools that no longer exist) and plan completion documentation verification (detecting missing AGENTS.md updates and unapproved specs after a plan ships).

---

## 2. Background

Stale tool references in `.skills/*.md` and `AGENTS.md` survived through five completed plans (Kanbanzai 2.0, P6, P7, P8, P9) without detection. Every agent that read the SKILL files during that period received instructions referencing tools that had been removed in Kanbanzai 2.0. Similarly, AGENTS.md Project Status and Scope Guard sections were found to be missing plan completion entries during P7, P8, and P9 reviews — each time caught only by a human reviewer, not by any automated check.

The health check system already supports extensibility via the `AdditionalHealthChecker` pattern used by `Phase3HealthChecker`, `Phase4aHealthChecker`, and `Phase4bHealthChecker`. This specification adds a new checker following the same pattern.

---

## 3. Tier 1: Tool Name Validation

### 3.1 Scope

Scan the following files for tool name references:

- All `.md` files in `.skills/` (currently: `code-review.md`, `document-creation.md`, `plan-review.md`, `README.md`)
- `AGENTS.md` (project root)

### 3.2 Tool Name Extraction

Extract candidate tool names from markdown content using two patterns:

1. **Backtick-wrapped function-call syntax:** Matches of the form `` `tool_name(` `` or `` `tool_name` `` where the name consists of lowercase letters, digits, and underscores. Examples: `` `doc_record_submit` ``, `` `batch_import_documents(` ``, `` `context_assemble(` ``.

2. **MCP action invocation syntax:** Matches of the form `tool(action: "...")` or `tool(action: ...)` where `tool` is a known tool name pattern. The tool name portion (before the parenthesis) is the candidate. Examples: `doc(action: "refresh")` extracts `doc`; `entity(action: "list")` extracts `entity`.

### 3.3 Known Tool Set

The checker receives the set of registered MCP tool names at construction time. For the current Kanbanzai 2.0 server, this is the 22-tool set: `status`, `entity`, `next`, `finish`, `handoff`, `health`, `server_info`, `decompose`, `estimate`, `conflict`, `knowledge`, `profile`, `worktree`, `merge`, `pr`, `branch`, `cleanup`, `doc`, `doc_intel`, `incident`, `checkpoint`, `retro`.

### 3.4 Matching Rules

A candidate tool name is flagged as stale if ALL of the following are true:

1. It is not present in the known tool set.
2. It looks like a plausible tool name — at least two characters, consists of `[a-z][a-z0-9_]*`.
3. It is not on the exclusion list (see §3.5).

### 3.5 Exclusion List

Some identifiers in backticks are not tool names (e.g., `` `go test` ``, `` `git status` ``, `` `goimports` ``, `` `kbz` ``, `` `kanbanzai` ``, `` `.kbz/` ``). The checker maintains a static exclusion list of common non-tool identifiers. This list is defined in the implementation and does not need to be exhaustive — the matching rules in §3.4 already filter out most non-tool content (multi-word strings, paths, etc.).

The exclusion list should include at minimum:
- Common CLI tools: `go`, `git`, `grep`, `cat`, `find`, `sed`, `make`, `shasum`
- Project-specific terms: `kbz`, `kanbanzai`, `goimports`, `go_fmt`, `go_vet`, `go_test`
- Format identifiers: `yaml`, `json`, `utf`, `lf`
- Go keywords and common patterns that appear in backticks in documentation

### 3.6 Output

For each stale tool reference, emit a health **warning** (not error) with:
- Category: `doc_currency`
- Message: `stale tool reference "{name}" in {file_path}` (file path relative to repo root)
- Entity type: `documentation`

Warnings, not errors, because a stale reference may be an intentional historical mention rather than a broken instruction.

---

## 4. Tier 2: Plan Completion Documentation Checklist

### 4.1 Scope

For each plan entity in `done` status, verify that project-level documentation reflects its completion.

### 4.2 Checks

**Check 1 — AGENTS.md Project Status:**
Read `AGENTS.md` from the repository root. Search for the plan's slug (e.g., `mcp-discoverability` for P9) in the Project Status section. If the slug does not appear anywhere in the file, emit a warning.

**Check 2 — AGENTS.md Scope Guard:**
Search the Scope Guard section of `AGENTS.md` for the plan's slug or plan ID prefix (e.g., `P9`). If neither appears in the Scope Guard section, emit a warning.

**Check 3 — Spec Document Status:**
For each feature under the plan in `done` status, check associated specification documents. If any spec document owned by the plan or its features has status `draft` (not `approved`), emit a warning.

### 4.3 Section Detection

To locate the "Project Status" and "Scope Guard" sections in AGENTS.md, search for markdown headings containing those strings (e.g., `## Project Status`, `## Scope Guard`). The section content runs from the heading to the next heading of equal or lesser depth.

### 4.4 Output

For each finding, emit a health **warning** with:
- Category: `doc_currency`
- Message format:
  - Check 1: `plan "{plan_id}" is done but not mentioned in AGENTS.md Project Status`
  - Check 2: `plan "{plan_id}" is done but not mentioned in AGENTS.md Scope Guard`
  - Check 3: `spec document "{doc_id}" is still in draft status but plan "{plan_id}" is done`
- Entity type: `plan` (checks 1–2) or `document` (check 3)

### 4.5 Plans Without Specs

Some plans have features that do not require specification documents (e.g., pure documentation work). Check 3 only applies to spec documents that exist and are associated with the plan's features. Missing specs are not flagged — that is a different concern handled by feature lifecycle gates.

---

## 5. Implementation

### 5.1 Architecture

Implement as an `AdditionalHealthChecker` function in `internal/mcp/doc_currency_health.go`. The checker is constructed in the MCP server layer with access to:

- The registered tool name set (from the MCP server's tool registry)
- The repository root path (for reading `.skills/` and `AGENTS.md`)
- The entity service (for listing plans, features, and their status)
- The document service (for checking spec document status)

The checker returns a `*validate.HealthReport` that is merged into the main health report by the existing `mergeHealthReports` function.

### 5.2 Registration

Register the checker alongside existing checkers in `internal/mcp/server.go`, following the same pattern as `Phase3HealthChecker`, `Phase4aHealthChecker`, and `Phase4bHealthChecker`.

### 5.3 File Layout

| File | Purpose |
|------|---------|
| `internal/mcp/doc_currency_health.go` | Checker implementation |
| `internal/mcp/doc_currency_health_test.go` | Tests |

---

## 6. Acceptance Criteria

| # | Criterion |
|---|-----------|
| C.1 | Health check detects a backtick-wrapped tool name in `.skills/*.md` that is not in the MCP tool registry |
| C.2 | Health check detects a backtick-wrapped tool name in `AGENTS.md` that is not in the MCP tool registry |
| C.3 | Health check does not flag tool names that ARE in the registry (e.g., `status`, `entity`, `doc`) |
| C.4 | Health check does not flag excluded identifiers (e.g., `go`, `git`, `kbz`) |
| C.5 | Health check detects a plan in `done` state with no mention in AGENTS.md Project Status section |
| C.6 | Health check detects a plan in `done` state with no mention in AGENTS.md Scope Guard section |
| C.7 | Health check detects a feature spec document in `draft` status when the parent plan is `done` |
| C.8 | Health check does not flag plans that are not in `done` state |
| C.9 | Health check does not flag plans in `done` state that ARE mentioned in AGENTS.md |
| C.10 | The new checker is registered via the `AdditionalHealthChecker` pattern |
| C.11 | Findings are emitted as warnings (not errors) with category `doc_currency` |
| C.12 | `go test -race ./...` passes |

---

## 7. Test Requirements

### 7.1 Tier 1 Tests

Table-driven tests using temporary directories with synthetic `.skills/` and `AGENTS.md` files:

- `TestDocCurrencyHealth_DetectsStaleToolName` — a SKILL file contains `batch_import_documents` (not in registry); warning emitted.
- `TestDocCurrencyHealth_IgnoresValidToolName` — a SKILL file contains `doc(action: "refresh")`; no warning.
- `TestDocCurrencyHealth_IgnoresExcludedNames` — AGENTS.md contains `` `go test` ``, `` `git status` ``, `` `kbz` ``; no warning.
- `TestDocCurrencyHealth_DetectsInAgentsMD` — AGENTS.md contains `context_assemble`; warning emitted.
- `TestDocCurrencyHealth_MultipleStaleRefs` — file with several stale names emits one warning per name.

### 7.2 Tier 2 Tests

Tests using the entity and document service test infrastructure:

- `TestDocCurrencyHealth_DetectsMissingProjectStatus` — plan in `done`, slug absent from AGENTS.md Project Status; warning emitted.
- `TestDocCurrencyHealth_DetectsMissingScopeGuard` — plan in `done`, slug absent from Scope Guard; warning emitted.
- `TestDocCurrencyHealth_DetectsDraftSpec` — plan `done`, child feature `done`, spec document `draft`; warning emitted.
- `TestDocCurrencyHealth_IgnoresActivePlan` — plan in `active`; no warning for missing AGENTS.md mention.
- `TestDocCurrencyHealth_PassesWhenMentioned` — plan in `done`, slug present in both sections; no warning.

### 7.3 Integration

- `TestDocCurrencyHealth_RegisteredInHealthTool` — verify the checker is included when calling the `health` MCP tool (the output should include the `doc_currency` category when findings exist).