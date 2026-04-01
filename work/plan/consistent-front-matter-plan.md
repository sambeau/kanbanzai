# Consistent Front Matter Implementation Plan

| Field | Value |
|-------|-------|
| Author | Sam Phillips |
| Created | 2026-07-18 |

Related:
- `work/spec/consistent-front-matter.md` (specification)
- `work/design/consistent-front-matter.md` (design document)
- `internal/docint/extractor.go` (extractFrontMatter function)
- `internal/docint/types.go` (FrontMatter struct)
- `internal/validate/health.go` (CheckDocumentHealth function)

---

## 1. Overview

This plan decomposes the Consistent Front Matter specification into assignable tasks for AI agents. The specification defines four features:

| Feature | Summary | Tasks |
|---------|---------|-------|
| **B** — Pipe-Table Parser | Extend `extractFrontMatter` to recognise pipe-table metadata | T1 |
| **C** — Health Checks | Add front matter hygiene checks to `CheckDocumentHealth` | T2 |
| **D** — Document Migration | Strip store-managed fields from existing managed documents | T3 |
| **A** — Front Matter Standard | The rules themselves — verified by T1, T2, T3 collectively | (no standalone task) |

Feature A is a specification of rules and conventions. It has no standalone implementation — its acceptance criteria are satisfied by the parser (Feature B), the health checks (Feature C), and the migration (Feature D).

---

## 2. Dependency Graph

```
T1: Pipe-Table Parser
        │
        ├──────────────┐
        ▼              ▼
T2: Health Checks    T3: Document Migration
```

- **T1** has no dependencies — it extends an existing function with new parsing logic.
- **T2** depends on **T1** — the health checks call `extractFrontMatter`, which must handle pipe-table format before the checks can validate documents using that format.
- **T3** depends on **T1** — the migration must parse both bullet-list and pipe-table documents to identify and strip prohibited fields.
- **T2** and **T3** are independent of each other and MAY run in parallel once T1 is complete.

---

## 3. Interface Contract

Tasks T2 and T3 both depend on `extractFrontMatter`. This is the shared interface:

### Function signature (unchanged)

```go
// extractFrontMatter parses front matter from document content.
// Returns nil if no front matter is found.
func extractFrontMatter(content []byte) *FrontMatter
```

### FrontMatter struct (unchanged — spec §5.6)

```go
type FrontMatter struct {
    Type    string            `yaml:"type,omitempty"`
    Status  string            `yaml:"status,omitempty"`
    Date    string            `yaml:"date,omitempty"`
    Related []string          `yaml:"related,omitempty"`
    Extra   map[string]string `yaml:"extra,omitempty"`
}
```

### Field mapping contract for pipe-table format (spec §5.4)

| Pipe-table field name | Target | Matching |
|-----------------------|--------|----------|
| `Author` | `Extra["author"]` | Case-insensitive |
| `Created` | `Date` | Case-insensitive |
| `Date` | `Date` | Case-insensitive |
| `Status` | `Status` | Case-insensitive |
| `Type` | `Type` | Case-insensitive |
| Any other field | `Extra[field_name]` | Preserved-case key |

T2 and T3 agents can rely on this contract to write their code without waiting for T1's implementation details.

### Prohibited field names (used by T2 and T3)

For health checks and migration, the following field names are prohibited in managed documents (case-insensitive matching):

`status`, `type`, `owner`, `approved`, `approved by`, `approved at`, `updated`

These map to the `FrontMatter` struct as follows:
- `Status` → `FrontMatter.Status` (non-empty means present)
- `Type` → `FrontMatter.Type` (non-empty means present)
- `Owner`, `Approved`, `Approved by`, `Approved at`, `Updated` → keys in `FrontMatter.Extra` (case-insensitive key search)

---

## 4. Task T1: Pipe-Table Parser

### Objective

Extend `extractFrontMatter` in `internal/docint/extractor.go` to recognise and parse pipe-table style front matter, while preserving all existing bullet-list parsing behaviour.

### Specification references

- §5.2 (pipe-table detection)
- §5.3 (pipe-table parsing rules)
- §5.4 (field mapping)
- §5.5 (Related list extraction after pipe-table)
- §5.6 (FrontMatter struct unchanged)
- §5.7 (backward compatibility)
- §5.8 (acceptance criteria 1–8)

### Input context

- **Read:** `internal/docint/extractor.go` — the existing `extractFrontMatter` and `parseFrontMatterLines` functions. Understand the heading-detection and blank-line-skipping preamble (lines 240–267 of extractor.go), which locates the first non-blank line after the H1 heading. This preamble is the insertion point for the format detection branch.
- **Read:** `internal/docint/types.go` lines 40–46 — the `FrontMatter` struct. This struct must not be modified.
- **Read:** `internal/docint/extractor_test.go` — existing front matter tests (`TestExtractPatterns_FrontMatter`, `TestExtractPatterns_FrontMatter_WithBasis`, `TestExtractPatterns_NoFrontMatter`, `TestExtractPatterns_NoFrontMatter_NoHeading`). These must all continue to pass.
- **Consult:** Spec §5.4 field mapping table — this is the authoritative reference for how pipe-table field names map to `FrontMatter` fields.

### Output artifacts

- **Modify:** `internal/docint/extractor.go`
  - Add a format detection branch in `extractFrontMatter`: if first non-blank line after heading starts with `|`, delegate to a new `parsePipeTableFrontMatter` function. If it starts with `- `, continue with existing bullet-list logic.
  - Implement `parsePipeTableFrontMatter(lines []string, startIdx int) *FrontMatter` — parses pipe-table rows, skips the header-separator row, extracts key-value pairs, and maps them per the §5.4 table.
  - Handle the `Related:` block appearing after the pipe-table and before the `---` separator, extracting it into `FrontMatter.Related`.
- **Modify:** `internal/docint/extractor_test.go` — add tests:
  - Pipe-table with `Author` and `Created` fields → `Extra["author"]` and `Date` populated.
  - Pipe-table with `Author`, `Created`, and a `Related:` list → `Related` slice populated.
  - Pipe-table with extra fields (`Reviewer`, `Verdict`) → stored in `Extra`.
  - Pipe-table with `Status` and `Type` → `FrontMatter.Status` and `FrontMatter.Type` populated.
  - Pipe-table with case variation (`AUTHOR`, `created`, `Status`) → correctly mapped.
  - Pipe-table with header + separator row only (no data rows) → returns `nil`.
  - Header separator row (`|---|---|`) not parsed as a field entry.
  - Existing bullet-list tests all pass unchanged.

### Dependencies

None.

### Verification

```
go test ./internal/docint/ -run TestExtract -v
go test ./internal/docint/ -v          # full suite — confirm nothing broken
```

---

## 5. Task T2: Health Checks

### Objective

Add three front matter hygiene checks to the existing `CheckDocumentHealth` function in `internal/validate/health.go`: prohibited field detection, stale status echo, and missing provenance.

### Specification references

- §6.1 (scope — extends CheckDocumentHealth)
- §6.2 (parse on demand, not from index)
- §6.3 (prohibited field check — warning severity, message format)
- §6.4 (stale status echo — warning severity, message format, dual-warning behaviour)
- §6.5 (missing provenance — warning severity, both-absent trigger)
- §6.6 (no front matter is not an error)
- §6.7 (acceptance criteria 1–13)

### Input context

- **Read:** `internal/validate/health.go` lines 217–330 — the `CheckDocumentHealth` function and `DocumentInfo` struct. Understand the existing check pattern: iterate over docs, extract fields from `d.Fields`, append to `report.Errors` or `report.Warnings`.
- **Read:** `internal/validate/doc_health_test.go` — existing test patterns. Tests use `loadAll`/`entityExists`/`checkContentHash` function stubs and `validDocFields` helpers.
- **Read:** `internal/docint/extractor.go` — the `extractFrontMatter` function signature and return type. This is the function T2 will call.
- **Read:** `internal/docint/types.go` lines 40–46 — the `FrontMatter` struct.
- **Consult:** The interface contract in §3 of this plan for prohibited field names and how they map to `FrontMatter` fields.

### Output artifacts

- **Modify:** `internal/validate/health.go`
  - Add an import of `"github.com/your-org/kanbanzai/internal/docint"` (or whatever the correct import path is — check `go.mod`).
  - Extend `CheckDocumentHealth` signature to accept a new parameter: `readFileContent func(path string) ([]byte, error)`. This function reads the document file content for front matter parsing. The existing content-hash check already implies the file is readable; this parameter makes the content available to the front matter checks without reading the file twice.
  - After the existing Check 4 (approved documents), add Check 5 (front matter hygiene):
    1. Call `readFileContent(docPath)` to get file bytes (skip if error — file-missing already reported by Check 1).
    2. Call `docint.ExtractFrontMatter(content)` (note: `extractFrontMatter` is currently unexported — it needs to be exported or a wrapper added. See design note below).
    3. If result is `nil`, skip front matter checks for this document.
    4. Check for each prohibited field. Emit a warning per field found.
    5. If `Status` is present and non-empty, compare case-insensitively with store status. If mismatched, emit the stale-status warning.
    6. If no `Author` key in `Extra` (case-insensitive) AND `Date` is empty, emit the missing-provenance warning.
  - **Design note on export:** `extractFrontMatter` is currently unexported. The cleanest approach is to add an exported wrapper `ParseFrontMatter(content []byte) *FrontMatter` in `internal/docint/extractor.go` that delegates to `extractFrontMatter`. This avoids renaming the existing function and breaking internal call sites. Alternatively, simply rename `extractFrontMatter` → `ExtractFrontMatter` and update the one internal caller in `ExtractPatterns`. Either approach is acceptable.
- **Modify:** `internal/validate/doc_health_test.go` — add tests covering all 13 acceptance criteria from spec §6.7. Follow the existing test pattern: construct `DocumentInfo` with `Fields` map, provide stubs for the new `readFileContent` parameter, and assert on the returned `report.Warnings`.
- **Modify:** All callers of `CheckDocumentHealth` — update to pass the new `readFileContent` parameter. Search for `CheckDocumentHealth(` to find all call sites (likely in `internal/health/` or `internal/mcp/`).

### Dependencies

**T1 must be complete.** The health checks call `extractFrontMatter`, which must handle pipe-table format to correctly check documents using the new template.

### Verification

```
go test ./internal/validate/ -run TestCheckDocumentHealth -v
go test ./internal/... -v              # confirm no breakage from signature change
go test -race ./...                    # full suite
```

---

## 6. Task T3: Document Migration

### Objective

Update all documents currently registered in the `.kbz` document record store to comply with the front matter standard: strip prohibited fields, ensure provenance fields are present, and refresh content hashes.

### Specification references

- §7.1 (scope — all managed documents)
- §7.2 (migration actions per document)
- §7.3 (commit discipline — single atomic commit)
- §7.4 (content hash refresh)
- §7.5 (acceptance criteria 1–6)

### Input context

- **Read:** `internal/docint/extractor.go` — the `extractFrontMatter` function (as extended by T1) to understand parsing for both formats.
- **Run:** `kbz health` (or `kanbanzai serve` → `health` tool) to get the current list of managed documents and their paths.
- **Run:** `doc list` to enumerate all managed document records, noting their `path` and `status` fields.
- **Consult:** Spec §4.2 for the complete list of prohibited field names.
- **Consult:** Spec §7.2 for the git-history inference commands for `Author` and `Created`.

### Migration procedure

For each managed document file:

1. **Read** the file content.
2. **Identify** the front matter format (bullet-list or pipe-table) and its line boundaries.
3. **Remove** all lines corresponding to prohibited fields:
   - In bullet-list format: lines starting with `- Status:`, `- Type:`, `- Owner:`, `- Approved:`, `- Approved by:`, `- Approved at:`, `- Updated:` (case-insensitive). Also remove any indented continuation lines belonging to removed fields.
   - In pipe-table format: table rows where the first column matches a prohibited field name (case-insensitive).
4. **Check** for `Author` and `Created`. If either is absent:
   - Infer `Author` from git history: `git log --format='%an' --reverse -- <path> | head -1`
   - Infer `Created` from git history: `git log --format='%aI' --reverse -- <path> | head -1` (truncate to `YYYY-MM-DD`)
   - Add the missing field(s) to the front matter block, preserving the document's existing format (bullet-list or pipe-table).
5. **Write** the modified file, preserving all content outside the front matter block byte-for-byte.
6. **Do not** convert bullet-list documents to pipe-table format (permitted but not required — keep migration minimal and safe).

After all files are modified:

7. **Commit** all changes as a single atomic commit:
   ```
   docs: strip store-managed metadata from document front matter

   Remove Status, Type, and Owner fields from managed documents.
   These fields are tracked in the .kbz document record store.
   Add Author and Created provenance where absent (inferred from git history).
   ```
8. **Refresh** content hashes for all modified documents: run `doc refresh` for each document ID, or use batch refresh if available.

### Output artifacts

- **Modify:** All managed document `.md` files in `work/` — front matter blocks only. Prose content is untouched.
- **Update:** `.kbz/state/documents/` — content hashes refreshed to match modified files.

### Dependencies

**T1 must be complete.** The migration must correctly parse both bullet-list and pipe-table front matter to identify prohibited fields.

T3 does NOT depend on T2. The migration is a file-editing activity, not a health-check activity. However, running `health` after migration (with T2's checks in place) is a useful verification step.

### Verification

After the migration commit:

```
kbz health                            # zero prohibited-field warnings, zero stale-status warnings
git diff HEAD~1 --stat                # confirm only work/ files and .kbz/state/documents/ changed
```

Manual spot-checks:
- Pick 3 documents that had `Status:` in their front matter — confirm it is removed.
- Pick 3 documents that had no `Author` — confirm `Author` was added from git history.
- Pick 1 document in each format (bullet-list, pipe-table) — confirm prose content is identical.

---

## 7. Execution Summary

| Task | Feature | Files modified | Depends on | Parallel? |
|------|---------|----------------|------------|-----------|
| T1 | B — Parser | `internal/docint/extractor.go`, `internal/docint/extractor_test.go` | — | Start immediately |
| T2 | C — Health | `internal/validate/health.go`, `internal/validate/doc_health_test.go`, callers of `CheckDocumentHealth` | T1 | After T1 |
| T3 | D — Migration | `work/**/*.md`, `.kbz/state/documents/**` | T1 | After T1, parallel with T2 |

**Critical path:** T1 → T2 (or T1 → T3). Total serial depth: 2 tasks.

**Parallelism:** T2 and T3 can run concurrently. They modify entirely different files — T2 modifies `internal/validate/` (Go code), T3 modifies `work/` (Markdown documents) and `.kbz/state/` (YAML records). No file conflicts are possible.

---

## 8. Traceability Matrix

| Spec Requirement | Task |
|------------------|------|
| §4.1 Permitted front matter fields | T1 (parser recognises them), T3 (migration preserves them) |
| §4.2 Prohibited front matter fields | T2 (health checks detect them), T3 (migration removes them) |
| §4.3 Cross-references | T1 (parser extracts Related list from pipe-table docs) |
| §4.4 Standard template format | T1 (parser handles pipe-table format) |
| §4.5 Bullet-list backward compatibility | T1 (existing tests pass), T3 (migration handles both formats) |
| §4.6 Unmanaged documents exempt | T2 (health checks skip unmanaged docs) |
| §4.7 Non-metadata prose fields | No task needed (no system change) |
| §4.8 Registration tooling is read-only | No task needed (no change to registration) |
| §5.2 Pipe-table detection | T1 |
| §5.3 Pipe-table parsing rules | T1 |
| §5.4 Field mapping | T1 |
| §5.5 Related list extraction | T1 |
| §5.6 FrontMatter struct unchanged | T1 (verified by compilation) |
| §5.7 Backward compatibility | T1 (existing tests pass) |
| §6.1 Health check scope | T2 |
| §6.2 Parse on demand | T2 |
| §6.3 Prohibited field check | T2 |
| §6.4 Stale status echo | T2 |
| §6.5 Missing provenance | T2 |
| §6.6 No front matter is not an error | T2 |
| §7.1 Migration scope | T3 |
| §7.2 Migration actions | T3 |
| §7.3 Commit discipline | T3 |
| §7.4 Content hash refresh | T3 |

---

## 9. Scope Boundaries

Carried forward from the specification (§9):

- No changes to the `.kbz` document record schema.
- No changes to `doc register`, `doc import`, or `doc approve` tool behaviour.
- No automated file rewriting triggered by registration tooling.
- No front matter rules for files outside `work/`.
- No changes to the `DocumentIndex` schema.
- The document-creator skill is out of scope for this plan (it is a skill file, not code).