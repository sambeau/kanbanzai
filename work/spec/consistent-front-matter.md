# Consistent Front Matter Specification

| Field | Value |
|-------|-------|
| Author | Sam Phillips |
| Created | 2026-07-18 |

Related:
- `work/design/consistent-front-matter.md` (design document)
- `work/design/document-centric-interface.md` (document-centric human interface model)
- `work/design/document-intelligence-design.md` §5, §6 (structural analysis backend)

---

## 1. Purpose

This specification defines the acceptance criteria for consistent front matter across documents managed by the Kanbanzai system. It is the implementation contract derived from the design document (`work/design/consistent-front-matter.md`).

The specification covers four features:

| ID | Label | Scope |
|----|-------|-------|
| A | front-matter-standard | The rules for what metadata belongs in document files vs. the `.kbz` store |
| B | pipe-table-parser | Extend `extractFrontMatter` to recognise pipe-table metadata |
| C | health-checks | Front matter hygiene checks in `CheckDocumentHealth` |
| D | document-migration | Strip store-managed fields from existing managed documents |

---

## 2. Supersession

This specification introduces new requirements. It does not supersede any existing specification.

---

## 3. Definitions

| Term | Meaning |
|------|---------|
| **Managed document** | A document that has a corresponding record in the `.kbz` document record store (i.e., it has been registered via `doc register` or `doc import`). |
| **Unmanaged document** | A document file that has no corresponding record in the `.kbz` document record store. |
| **Front matter** | Structured metadata appearing between the H1 heading and the first `---` separator or first body section in a Markdown document. May be in bullet-list or pipe-table format. |
| **Provenance field** | A front matter field that records an immutable fact from the time of document creation (e.g., author, creation date). |
| **Store-managed field** | A metadata field whose authoritative value lives in the `.kbz` document record store and is mutated by workflow actions (e.g., status, type, owner, approval fields). |
| **Prohibited field** | A store-managed field that MUST NOT appear in a managed document's front matter. |

---

## 4. Feature A: Front Matter Standard

### 4.1 Permitted front matter fields

A managed document's front matter MAY contain only the following fields:

| Field | Required | Format | Description |
|-------|----------|--------|-------------|
| `Author` | Recommended | Free text | Who wrote the document. May differ from the store's `created_by` field, which records who registered the document. |
| `Created` | Recommended | ISO 8601 date (`YYYY-MM-DD`) | The date the document was first written. |

The document title MUST be the Markdown H1 heading. It MUST NOT be repeated as a front matter field.

### 4.2 Prohibited front matter fields

The following fields MUST NOT appear in a managed document's front matter. These are store-managed fields whose authoritative value lives in the `.kbz` document record store.

| Prohibited field | Corresponding store field |
|------------------|--------------------------|
| `Status` | `status` |
| `Type` | `type` |
| `Owner` | `owner` |
| `Approved` / `Approved by` | `approved_by` |
| `Approved at` | `approved_at` |
| `Updated` | `updated` |

Detection of prohibited fields MUST be case-insensitive. A field named `status`, `Status`, or `STATUS` is equally prohibited.

### 4.3 Cross-references

Documents MAY include a `Related` list after the front matter table and before the `---` separator. Cross-references are informational context — the author's stated references at time of writing. They are not authoritative relationship data.

The format MUST be a Markdown bullet list with each item being a backtick-quoted document path, optionally followed by section references:

```
Related:
- `work/design/some-design.md` §3
- `work/spec/some-spec.md`
```

Cross-references are not validated for accuracy or staleness. They represent the author's intent at time of writing.

### 4.4 Standard template format

New documents MUST use pipe-table format for front matter. The standard template is:

```
# Document Title

| Field | Value |
|-------|-------|
| Author | Name or identity |
| Created | YYYY-MM-DD |

---
```

Or with cross-references:

```
# Document Title

| Field | Value |
|-------|-------|
| Author | Name or identity |
| Created | YYYY-MM-DD |

Related:
- `path/to/document.md` §section

---
```

### 4.5 Bullet-list backward compatibility

Existing documents using bullet-list front matter do not need to be converted to pipe-table format. Both formats MUST be accepted by the `docint` parser. Only new documents are required to use pipe-table format.

### 4.6 Unmanaged documents

Unmanaged documents (not registered in the store) MAY contain any front matter fields, including store-managed fields such as `Status`. This permits drafts-in-progress to carry metadata useful to human readers before registration.

Once a document is registered, store-managed fields SHOULD be removed from the file. This is a convention enforced by skills and health checks, not by the registration tooling.

### 4.7 Non-metadata prose fields

Fields like `Purpose` that appear in many existing documents are neither prohibited nor subject to this specification. They are ordinary prose content that happens to be near the top of the file.

### 4.8 Registration tooling is read-only

The `doc register` and `doc import` actions MUST NOT modify document file content. They remain read-only with respect to `.md` files. Stripping of prohibited fields is a manual or skill-guided activity, not an automated side effect of registration.

### 4.9 Acceptance criteria

1. The standard template defines exactly two optional provenance fields: `Author` and `Created`.
2. The six store-managed fields listed in §4.2 are prohibited in managed documents (case-insensitive match).
3. Cross-references use the format specified in §4.3.
4. New documents use pipe-table format.
5. Bullet-list format is accepted in existing documents without requiring conversion.
6. Unmanaged documents are exempt from the prohibited-field rule.
7. `doc register` and `doc import` do not modify document file content.

---

## 5. Feature B: Pipe-Table Parser

### 5.1 Current state

The `extractFrontMatter` function in `internal/docint/extractor.go` currently parses only bullet-list style front matter (lines beginning with `- ` after the first heading). It populates the `FrontMatter` struct with `Type`, `Status`, `Date`, `Related`, and `Extra` fields.

### 5.2 Pipe-table detection

The parser MUST detect pipe-table front matter by examining the first non-blank line after the H1 heading. If the line starts with `|`, the parser MUST treat the block as pipe-table metadata. If it starts with `- `, the parser MUST treat it as bullet-list metadata (existing behaviour). If it starts with anything else, there is no front matter.

### 5.3 Pipe-table parsing rules

When a pipe-table is detected, the parser MUST:

1. Parse the table rows between the header row and the `---` separator (or next heading, or end of front matter block).
2. Ignore the header-separator row (the row containing `|---|---|` or similar).
3. Extract key-value pairs from the remaining rows, where the first column is the field name and the second column is the value.
4. Trim whitespace from both field names and values.
5. Match field names case-insensitively to map them into the `FrontMatter` struct.

### 5.4 Field mapping

Pipe-table fields MUST map to the existing `FrontMatter` struct as follows:

| Pipe-table field name | `FrontMatter` target | Matching |
|-----------------------|----------------------|----------|
| `Author` | `Extra["author"]` | Case-insensitive |
| `Created` | `Date` | Case-insensitive |
| `Date` | `Date` | Case-insensitive |
| `Status` | `Status` | Case-insensitive |
| `Type` | `Type` | Case-insensitive |
| Any other field | `Extra[field_name]` | Preserved-case key |

### 5.5 Related list extraction

If a `Related:` block appears after the pipe-table and before the `---` separator, the parser MUST extract it into `FrontMatter.Related` using the same logic as the existing bullet-list `Related` extraction.

### 5.6 FrontMatter struct

The `FrontMatter` struct in `internal/docint/types.go` MUST NOT change. Pipe-table metadata MUST be representable within the existing fields (`Type`, `Status`, `Date`, `Related`, `Extra`).

### 5.7 Backward compatibility

Existing bullet-list parsing behaviour MUST NOT change. All existing tests for `extractFrontMatter` MUST continue to pass without modification.

### 5.8 Acceptance criteria

1. `extractFrontMatter` correctly parses pipe-table front matter from a document using the standard template (§4.4).
2. `extractFrontMatter` correctly parses pipe-table front matter with a `Related` list following the table.
3. Bullet-list front matter parsing is unaffected — all existing tests pass.
4. Field mapping follows the table in §5.4 exactly.
5. A document with no front matter (no bullet-list or pipe-table after heading) returns `nil`.
6. A document with a pipe-table header separator row does not produce a spurious field entry.
7. Field name matching is case-insensitive.
8. Round-trip: a pipe-table document parsed and re-indexed produces the same `FrontMatter` struct.

---

## 6. Feature C: Health Checks

### 6.1 Scope

Front matter hygiene checks MUST be added to the existing `CheckDocumentHealth` function in `internal/validate/health.go`. No new MCP tool surface is required — the `health` tool already aggregates and reports all health check results.

### 6.2 Front matter parsing approach

Health checks MUST parse front matter on demand from the document file content using `extractFrontMatter`. Health checks MUST NOT depend on the `docint` index being current.

The file content for each managed document is already available during health checks (the content-hash check reads the file). Front matter parsing MUST reuse this file read.

### 6.3 Check: Prohibited field in managed document

| Property | Value |
|----------|-------|
| Severity | Warning |
| Applies to | Managed documents only |
| Condition | The parsed `FrontMatter` contains a `Status`, `Type`, or `Owner` field (case-insensitive). For `Status`, the check also covers any key in `Extra` matching `status` case-insensitively. Similarly for `Type` and `Owner` via `Extra` keys. Also covers `Approved`, `Approved by`, `Approved at`, and `Updated` via `Extra` keys. |
| Message format | `"{doc_id}: front matter contains '{field}' which is managed by the store"` |

For each prohibited field present, a separate warning MUST be emitted.

### 6.4 Check: Stale status echo

| Property | Value |
|----------|-------|
| Severity | Warning |
| Applies to | Managed documents where both in-file `Status` and store `status` are non-empty |
| Condition | The in-file `FrontMatter.Status` value (case-insensitive comparison) does not match the store's `status` field. |
| Message format | `"{doc_id}: in-file status '{in_file_status}' does not match store status '{store_status}'"` |

This check is a specialisation of §6.3. When the status field is present AND mismatched, both the §6.3 prohibited-field warning and the §6.4 stale-status warning MUST be emitted.

### 6.5 Check: Missing provenance

| Property | Value |
|----------|-------|
| Severity | Warning |
| Applies to | Managed documents only |
| Condition | The parsed `FrontMatter` has no `Author` (neither `Extra["author"]` nor `Extra["Author"]` nor any case-insensitive match) AND no `Date`/`Created` field. |
| Message format | `"{doc_id}: front matter has no author or created date (recommended for provenance)"` |

This check fires only when BOTH author and date are absent. A document with either one present does not trigger the warning.

### 6.6 No front matter is not an error

If `extractFrontMatter` returns `nil` for a managed document (no front matter present at all), no front matter checks are run for that document. The absence of front matter is not itself a warning or error — it simply means there is nothing to validate.

### 6.7 Acceptance criteria

1. A managed document with `Status` in its front matter produces a prohibited-field warning.
2. A managed document with `Type` in its front matter produces a prohibited-field warning.
3. A managed document with `Owner` in its front matter produces a prohibited-field warning.
4. A managed document with `Approved`, `Approved by`, `Approved at`, or `Updated` in its front matter produces a prohibited-field warning for each.
5. Prohibited field detection is case-insensitive (`status`, `Status`, `STATUS` all trigger).
6. A managed document where in-file status is `draft` and store status is `approved` produces both a prohibited-field warning and a stale-status warning.
7. A managed document with in-file status matching store status produces a prohibited-field warning (status is still prohibited) but no stale-status warning.
8. A managed document with no `Author` and no `Created`/`Date` field produces a missing-provenance warning.
9. A managed document with `Author` but no `Created` does NOT produce a missing-provenance warning.
10. A managed document with `Created` but no `Author` does NOT produce a missing-provenance warning.
11. A managed document with no front matter at all produces no front-matter-related warnings.
12. An unmanaged document (no store record) is not subject to any front matter health checks.
13. Front matter health check results appear in the standard `health` tool output alongside existing document health checks.

---

## 7. Feature D: Document Migration

### 7.1 Scope

All documents currently registered in the `.kbz` document record store MUST be updated to comply with the front matter standard defined in Feature A.

### 7.2 Migration actions per document

For each managed document:

1. Parse the front matter.
2. Remove all prohibited fields (§4.2): `Status`, `Type`, `Owner`, `Approved`, `Approved by`, `Approved at`, `Updated` — regardless of case.
3. If `Author` is absent, infer from git history (`git log --format='%an' --reverse -- <path> | head -1`) and add it.
4. If `Created` is absent, infer from git history (`git log --format='%aI' --reverse -- <path> | head -1`, truncated to date) and add it.
5. If the document already uses pipe-table format, preserve it. If it uses bullet-list format, it MAY be converted to pipe-table format when it is being modified anyway. Conversion is not required.

### 7.3 Commit discipline

All migration changes MUST be committed as a single atomic commit. The commit message MUST follow the project's git commit policy.

### 7.4 Content hash refresh

After modifying document files, the corresponding document records in the `.kbz` store MUST have their `content_hash` updated via `doc refresh` to reflect the new file content. Failure to do so will cause content-hash-drift warnings in the next health check.

### 7.5 Acceptance criteria

1. After migration, no managed document contains any prohibited field in its front matter.
2. After migration, `health` reports zero prohibited-field warnings and zero stale-status warnings.
3. After migration, every managed document has at least one of `Author` or `Created` in its front matter (satisfying the missing-provenance check).
4. After migration, all document record `content_hash` values match the files on disk.
5. The migration is a single git commit.
6. Existing document prose content is unmodified — only the front matter metadata block is changed.

---

## 8. Dependencies and Assumptions

| Dependency | Description |
|------------|-------------|
| `extractFrontMatter` function | Exists in `internal/docint/extractor.go`. Feature B extends it. Feature C depends on it. |
| `CheckDocumentHealth` function | Exists in `internal/validate/health.go`. Feature C extends it. |
| `FrontMatter` struct | Exists in `internal/docint/types.go`. Feature B maps pipe-table fields into it without schema changes. |
| `health` MCP tool | Already aggregates `HealthReport` results. No changes needed to surface Feature C warnings. |
| Document record store | The `.kbz` store schema is unchanged. Feature A defines a boundary; it does not modify the store. |
| Git history | Feature D relies on `git log` to infer `Author` and `Created` for documents that lack them. |

---

## 9. Out of Scope

- Changes to the `.kbz` document record schema.
- Changes to `doc register`, `doc import`, or `doc approve` tool behaviour (these remain read-only with respect to document file content).
- Automated file rewriting triggered by registration tooling.
- Front matter rules for files outside `work/` (e.g., `AGENTS.md`, `README.md`, `.skills/` files).
- Controlled vocabulary for in-file `Purpose`, `Related`, or other non-metadata prose fields.
- Changes to the document intelligence index schema (`DocumentIndex`).
- The document-creator skill file content (skill files are outside the scope of this specification; the skill is guided by the standard defined here but its content is not specified).

---

## 10. Verification Approach

### 10.1 Unit tests — Feature B (parser)

- Pipe-table front matter with `Author` and `Created` fields.
- Pipe-table front matter with `Author`, `Created`, and a `Related` list.
- Pipe-table front matter with extra fields (e.g., `Reviewer`, `Verdict`) stored in `Extra`.
- Pipe-table with store-managed fields (`Status`, `Type`) correctly populating `FrontMatter.Status` and `FrontMatter.Type`.
- Pipe-table with case variation in field names (`AUTHOR`, `created`, `Status`).
- Pipe-table with no data rows (header + separator only) returns `nil` or empty `FrontMatter`.
- Pipe-table header separator row is not parsed as a field.
- Bullet-list front matter continues to parse identically (existing tests unchanged).
- Document with no front matter returns `nil`.

### 10.2 Unit tests — Feature C (health checks)

- Managed document with `Status` in front matter → prohibited-field warning.
- Managed document with `status` (lowercase) in front matter → prohibited-field warning (case-insensitive).
- Managed document with `Type` in front matter → prohibited-field warning.
- Managed document with `Owner` in front matter → prohibited-field warning.
- Managed document with mismatched in-file and store status → both prohibited-field and stale-status warnings.
- Managed document with matching in-file and store status → prohibited-field warning only, no stale-status warning.
- Managed document with no `Author` and no `Created` → missing-provenance warning.
- Managed document with `Author` only → no missing-provenance warning.
- Managed document with `Created` only → no missing-provenance warning.
- Managed document with no front matter → no front-matter warnings.
- Managed document with only permitted fields (`Author`, `Created`) → no warnings.
- Front matter warnings coexist correctly with existing document health checks (content hash drift, orphaned owner, etc.).

### 10.3 Integration verification — Feature D (migration)

- After migration, `health` produces zero prohibited-field warnings.
- After migration, `health` produces zero stale-status warnings.
- After migration, `health` produces zero content-hash-drift warnings for migrated documents.
- Every managed document has `Author` or `Created` (or both) in its front matter.
- Document prose content is byte-identical outside the front matter block.