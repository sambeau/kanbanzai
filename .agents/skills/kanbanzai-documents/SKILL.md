---
name: kanbanzai-documents
description: >
  Use when creating, editing, registering, or approving documents in a
  Kanbanzai-managed project. Activates for document types, placement,
  registration with doc_record_submit, approval workflow, content drift,
  doc_record_refresh, or any question about document status, ownership,
  or lifecycle.
metadata:
  kanbanzai-managed: "true"
  version: "0.1.0"
---

# SKILL: Kanbanzai Documents

## Purpose

Describe document types, where to place them, how to register them with the
system, and the approval workflow that makes them authoritative.

## When to Use

- When creating any new document in a configured document root
- When registering a document with `doc_record_submit`
- When approving a document or checking whether it is ready for approval
- When editing a document that has already been registered
- When unsure which document type or directory to use

---

## Document Types and Locations

Documents live in configured roots under the project's document directory
(typically `work/`). Each root has a default type:

| Type | Typical directory | When to use |
|---|---|---|
| `design` | `work/design/` | Architecture, vision, approach decisions |
| `specification` | `work/spec/` | Acceptance criteria, binding contracts |
| `dev-plan` | `work/dev/` or `work/plan/` | Implementation plans, task breakdowns |
| `research` | `work/research/` | Analysis, exploration, background |
| `report` | `work/reports/` | Review reports, audits, post-mortems |

The actual roots and default types are defined in `.kbz/config.yaml` under
`documents.roots`. Check the project configuration if the defaults above do
not match.

**Placement rule:** design content goes in design documents, not in planning
documents. A document in `work/plan/` that contains "Decision:",
"Architecture:", or "Technology Choice:" is a sign that design work is being
done in the wrong place.

---

## Registration

Every document placed in a configured root must be registered with the system
immediately after creation. Unregistered documents are invisible to document
intelligence, entity extraction, approval workflow, and health checks.

### Registering a single document

```
doc_record_submit(
  path="work/design/my-document.md",
  type="design",
  title="Human-Readable Title"
)
```

The `type` must match the document root. The system generates a document ID
from the path (e.g., `PROJECT/design-my-document`).

### Batch import

To catch unregistered documents or register many at once:

```
batch_import_documents(path="work")
```

This is idempotent — already-registered documents are skipped. Safe to run
at any time as a consistency check.

### Verify registration

```
doc_record_get(id="PROJECT/design-my-document")
```

A document is properly registered when a YAML record exists in
`.kbz/state/documents/` and `doc_record_get` returns its metadata.

---

## Approval Workflow

Documents follow a three-status lifecycle: **draft → approved → superseded**.

1. Agent or human creates the document file.
2. Agent registers it with `doc_record_submit` — status becomes `draft`.
3. Human reviews the document and signals approval (verbally: "Approved",
   "LGTM", or equivalent).
4. Agent calls `doc_record_approve` — status becomes `approved`.
5. Approved documents are the authoritative basis for downstream work:
   - Approved design → features can be created
   - Approved specification → tasks can be decomposed

A draft document is a working document. An approved document is a contract.

---

## Drift and Refresh

When a document is registered, the system records a content hash. If the
file is edited after registration, the hash becomes stale — this is called
**drift**.

- **Approving a drifted document will fail.** The system requires the
  content hash to match.
- **After editing a registered document**, call `doc_record_refresh` to
  update the hash before requesting approval.
- **If an approved document is edited**, the approval is effectively void.
  The content no longer matches what was approved. Notify the human and
  re-approve after review.

The drift mechanism exists to ensure that what was reviewed is what gets
approved. Do not bypass it.

---

## Supersession

When a document is replaced by a newer version:

1. Create and register the new document.
2. Call `doc_record_supersede` on the old document, linking it to the new one.
3. The old document's status becomes `superseded`.

Superseded documents remain in the repository as historical records. They are
no longer authoritative.

---

## What Agents Must Not Do

- Do not create Plan or Feature entities referencing a document that is still
  in `draft` status. Design documents must be approved first.
- Do not edit an approved document without notifying the human. The approval
  becomes void when content drifts.
- Do not forget to register. Creating a file and forgetting `doc_record_submit`
  is the single most common mistake.
- Do not place design content in planning documents. Design decisions belong
  in `work/design/`, not `work/plan/`.

---

## Commit Discipline

When creating a new document, commit both the file and its registration
record together:

```
git add work/design/my-document.md .kbz/state/documents/
git commit -m "docs(my-document): create design document for feature X"
```

When batch-importing, commit the new records:

```
git add .kbz/state/documents/
git commit -m "workflow(PROJECT): register new documents with system"
```

---

## Related

- `kanbanzai-getting-started` — session orientation (includes document
  awareness)
- `kanbanzai-workflow` — stage gates that depend on document approval
- `kanbanzai-design` — the design process that produces design documents