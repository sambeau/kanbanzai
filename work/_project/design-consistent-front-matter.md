# Consistent Front Matter

| Field | Value |
|-------|-------|
| Author | Sam Phillips |
| Created | 2026-07-18 |

---

## 1. Problem

Front matter across documents in `work/` is inconsistent, often stale, and cannot be trusted. Three specific issues:

1. **Duplication.** Metadata such as `Status`, `Type`, and `Owner` appears in document front matter *and* in the `.kbz` document record store. The two are never synchronised.

2. **Drift.** Documents regularly show `Status: draft` in their front matter while the store records them as `approved`. There is no mechanism to reconcile the two, and no tooling warns about the divergence.

3. **Format fragmentation.** Early documents (Phase 1–2 era) use bullet-list metadata. Later documents (Phase 3+) use pipe-table metadata. A handful use neither. Status values alone span 20+ variants (`draft`, `Draft`, `design basis`, `protocol draft`, `policy draft`, `draft design`, `design draft`, `design proposal`, `proposal`, `specification draft`, `plan draft`, `active`, `superseded`, `historical — research trail`, and more). There is no controlled vocabulary.

The result is that front matter is noise — humans reading `.md` files see metadata that may or may not reflect reality, and agents have no reliable way to extract workflow state from the file itself.

---

## 2. Design Principle

**Single source of truth.** Every piece of metadata lives in exactly one place:

- **Mutable workflow state** lives in the `.kbz` document record store. The store is the authority for status, type, ownership, approval, and supersession lifecycle.
- **Immutable provenance** lives in the document file. The file records facts that were true at creation time and will never change.

The document intelligence pipeline (`docint`) may parse and index in-file metadata for analysis, but it is never treated as a source of truth for lifecycle decisions.

---

## 3. What Stays in the File

A managed document's front matter contains only immutable provenance fields:

| Field | Required | Description |
|-------|----------|-------------|
| Author | Recommended | Who wrote the document (may differ from `created_by` in the store, which records who registered it) |
| Created | Recommended | Date the document was first written, ISO 8601 date (`2026-07-18`) |

The document title is the H1 heading — it is not repeated in the front matter table.

### Cross-references

Documents may include a `Related` list after the front matter table. These are the author's stated cross-references at time of writing — informational context, not authoritative relationship data. The store and docint cross-doc links are the authoritative relationship graph.

```/dev/null/related-example.md#L1-5
Related:
- `work/design/workflow-design-basis.md` §2, §3
- `work/spec/phase-1-specification.md` §6
- `work/design/document-intelligence-design.md` §5
```

Cross-references may become stale as referenced documents are superseded or moved. This is acceptable — they represent the author's intent at time of writing, not a live dependency graph.

---

## 4. What Lives Only in the Store

The following fields are managed exclusively by the `.kbz` document record store and MUST NOT appear in a managed document's front matter:

| Field | Store field | Rationale |
|-------|-------------|-----------|
| Status | `status` | Set by lifecycle transitions (`draft` → `approved` → `superseded`). The #1 source of drift today. |
| Type | `type` | Set at registration, inferred from path conventions by `doc import`. |
| Owner | `owner` | Plan or feature association. Changes as scope evolves. |
| Approved by | `approved_by` | Set by the `doc approve` action. |
| Approved at | `approved_at` | Set by the `doc approve` action. |
| Updated | `updated` | Bumped automatically on any record mutation. |

Fields like `Purpose` that appear in many existing documents are neither prohibited nor required — they are ordinary prose that happens to be near the top of the file. They are not metadata and need no special treatment.

---

## 5. Standard Template

### New document (pipe-table format)

```/dev/null/template.md#L1-10
# Document Title

| Field | Value |
|-------|-------|
| Author | Name or identity |
| Created | 2026-07-18 |

---

(body begins here)
```

### New document with cross-references

```/dev/null/template-with-refs.md#L1-14
# Document Title

| Field | Value |
|-------|-------|
| Author | Name or identity |
| Created | 2026-07-18 |

Related:
- `work/design/some-design.md` §3
- `work/spec/some-spec.md`

---

(body begins here)
```

The pipe-table format is the standard. Bullet-list metadata in existing documents is accepted by the `docint` parser and does not need to be converted, but all new documents use pipe-tables.

---

## 6. Unmanaged Documents

Documents that have not yet been registered in the `.kbz` store (drafts in progress, scratch notes, proposals) may contain additional metadata fields that are destined for the store — for example, a `Status: draft` line that signals intent to the human reader.

Once a document is registered via `doc register` or `doc import`, these store-managed fields become redundant. The recommended workflow:

1. Author writes document with whatever front matter is useful during drafting.
2. Agent or human registers the document in the store.
3. Store-managed fields (`Status`, `Type`, `Owner`) are stripped from the file in the same commit that registers it.

Step 3 is a convention enforced by skills and health checks, not by the registration tooling itself. The `doc register` and `doc import` actions do not modify document files — they are read-only with respect to the `.md` content.

---

## 7. Health Check Integration

The existing `CheckDocumentHealth` function in `internal/validate/health.go` already validates document records against the store (file existence, content hash drift, orphaned owners, approval field completeness). Front matter hygiene checks extend this naturally.

### Proposed checks

| Check | Severity | Condition |
|-------|----------|-----------|
| **Prohibited field in managed document** | Warning | A document registered in the store contains `Status`, `Type`, `Owner`, `Approved`, or `Approved by` in its parsed front matter. |
| **Stale status echo** | Warning | A managed document's in-file `Status` value does not match the store's `status` field. (Subset of the above, but with a more specific message.) |
| **Missing provenance** | Info | A managed document has no `Author` or `Created` field in its front matter. Not an error — provenance is recommended, not required. |

### Implementation approach

`CheckDocumentHealth` already receives each document's store fields and file path. The front matter checks need the parsed `FrontMatter` from the docint index. Two options:

**Option A — Parse on demand.** Read the file and call `extractFrontMatter` during the health check. Simple, no new dependencies, but reads every managed document file during health checks.

**Option B — Use the docint index.** Load the `DocumentIndex` for each managed document (already persisted in `.kbz/index/`). The `FrontMatter` field is already populated by Layer 2 extraction. Avoids re-reading files but introduces a dependency on the index being current.

Option A is simpler and more reliable — health checks should not depend on index freshness. The file read is already implicit in the content-hash check. The `extractFrontMatter` function is fast (string scanning, no allocation-heavy parsing).

### Check logic (pseudocode)

```/dev/null/pseudocode.go#L1-22
// For each managed document with a readable file:
fm := extractFrontMatter(fileContent)
if fm == nil {
    // No front matter — nothing to check.
    continue
}

// Prohibited fields in managed documents.
prohibitedFields := []string{"status", "type", "owner"}
for _, field := range prohibitedFields {
    if fm has field {
        report warning: "%s: front matter contains '%s' which is managed by the store"
    }
}

// Stale status echo (more specific message).
if fm.Status != "" && fm.Status != storeStatus {
    report warning: "%s: in-file status '%s' does not match store status '%s'"
}

// Missing provenance (informational).
if fm has no Author and no Created {
    report info: "%s: front matter has no author or created date (recommended)"
}
```

### Reporting

The `health` MCP tool already aggregates `HealthReport` errors and warnings across all entity types. Front matter warnings appear in the same report alongside existing document health checks. No new MCP tool surface is needed.

---

## 8. Docint Parser Implications

The `extractFrontMatter` function currently parses bullet-list metadata and populates `FrontMatter.Type`, `FrontMatter.Status`, `FrontMatter.Date`, `FrontMatter.Related`, and `FrontMatter.Extra`. It does not parse pipe-table format.

This design requires two changes:

1. **Extend the parser to recognise pipe-table metadata.** New documents use pipe-tables. The parser should handle both formats — bullet-list for backward compatibility, pipe-table as the standard going forward. The detection heuristic is straightforward: if the first non-blank line after the heading starts with `|`, parse as pipe-table; if it starts with `- `, parse as bullet-list.

2. **No changes to what's stored.** The `FrontMatter` struct is sufficient. Pipe-table fields map to the same struct: `Author` → `Extra["author"]`, `Created` → `Date`. The `Related` list continues to be extracted from the related block if present.

The parsed front matter remains a Layer 2 concern in the docint pipeline. It continues to be stored in `DocumentIndex.FrontMatter` and remains available for health checks and `TraceEntity` sort ordering.

---

## 9. Migration

### Phase 1 — New documents (immediate)

All new documents follow the template in §5. This is enforced by convention and by agent skills (the document creator skill described in §10).

### Phase 2 — Strip stale fields from managed documents (single commit)

For every document currently registered in the `.kbz` store:

1. Parse the front matter.
2. Remove `Status` and any other store-managed fields.
3. Ensure `Author` and `Created` are present (infer from git history if necessary).
4. Normalise to pipe-table format (optional — bullet-list is accepted, but new-format consistency is preferred for documents being touched anyway).

This is a single cleanup commit touching all affected files. The commit message:

```/dev/null/commit-msg.txt#L1-3
docs: strip store-managed metadata from document front matter

Remove Status, Type, and Owner fields from managed documents.
These fields are tracked in the .kbz document record store.
```

### Phase 3 — Health check enforcement (ongoing)

Once the health check integration (§7) is in place, any reintroduction of prohibited fields is surfaced as a warning in the `health` report. This provides ongoing enforcement without blocking workflows.

---

## 10. Document Creator Skill

A `document-creator` skill should codify the template and rules for agents creating new documents. The skill covers:

1. **Template.** Use the pipe-table format from §5.
2. **Prohibited fields.** Do not include `Status`, `Type`, `Owner`, or approval metadata in front matter.
3. **Registration.** After creating a document, register it in the store via `doc register`.
4. **Cross-references.** Include a `Related` list when the document references other documents. Use relative paths with optional section references.

The skill is a `.skills/` file, not runtime code. It is included in agent context via the standard skill discovery mechanism.

---

## 11. Scope and Non-Goals

This design covers:

- Front matter format standardisation for documents in `work/`.
- Rules for what metadata belongs in-file vs. in the store.
- Health check integration for ongoing enforcement.
- Migration plan for existing documents.

This design does not cover:

- Changes to the `.kbz` document record schema.
- Changes to the `doc register`, `doc import`, or `doc approve` MCP tool behaviour (these remain read-only with respect to document file content).
- Automated file rewriting by registration tooling (this was considered and rejected — too much magic, too many edge cases with formatting preservation).
- Front matter in files outside `work/` (e.g., `AGENTS.md`, `README.md`, skill files).