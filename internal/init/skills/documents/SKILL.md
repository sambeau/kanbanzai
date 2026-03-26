---
name: kanbanzai-documents
description: >
  Use this skill whenever you create, edit, register, or approve any document in a
  Kanbanzai-managed project, including files in work/design/, work/spec/, work/plan/,
  work/research/, or any configured document root. Activate even when document
  registration is not the primary goal of the task — any file creation in a document
  root requires registration.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill covers document types, where to place them, how to register them, and the
approval workflow. Every document placed in a configured root must be registered.

## Document Types and Locations

| Type | Directory | When to use |
|---|---|---|
| `design` | `work/design/` | Architecture, vision, approach decisions |
| `specification` | `work/spec/` | Acceptance criteria, binding contracts |
| `dev-plan` | `work/plan/` or `work/dev/` | Implementation plans, task breakdowns |
| `research` | `work/research/` | Analysis, exploration, background |
| `report` | `work/reports/` | Review reports, audits, post-mortems |
| `policy` | `work/design/` | Standing rules, process definitions |

Design decisions belong in `work/design/`, not `work/plan/`. If a document contains
architecture, API shapes, data models, or technology choices, it is a design document.

## Registration Procedure

Every document placed in a configured root must be registered immediately after creation:

    doc_record_submit(
      path="work/design/my-doc.md",
      type="design",
      title="Human-readable title"
    )

To batch-import an entire directory (idempotent, safe to repeat):

    batch_import_documents(path="work")

## Approval Workflow

Documents follow a three-status lifecycle: **draft → approved → superseded**.

- **Draft:** created, not yet approved. Cannot be used as a stage gate basis.
- **Approved:** approved by a human. An approved design document allows Feature entity
  creation. An approved specification allows task decomposition.
- **Superseded:** replaced by a newer document. The superseding document becomes the
  authoritative basis.

Approval can be verbal; record it immediately:

    doc_record_approve(id="DOC-...")

## Drift and Refresh

If a document is edited after registration, its content hash becomes stale. Attempting
to approve a drifted document will fail.

Check drift status:

    doc_record_get(id="DOC-...", check_drift=true)

Refresh before approving:

    doc_record_refresh(id="DOC-...")

## Supersession

When a document is replaced by a newer version:

1. Create and register the new document.
2. Call `doc_record_supersede(id="old-DOC-...", superseded_by="new-DOC-...")`.

Do not silently amend an approved document. Any edit to an approved document requires
creating a new document and superseding the old one.

---

## Gotchas

**Forgot to register:** The document is invisible to document intelligence, entity
extraction, approval workflow, and health checks. Use `batch_import_documents` as a
safety net to catch any unregistered files.

**Editing after registration:** The content hash becomes stale and approval will fail.
Always call `doc_record_refresh` after editing, before approving.

**Design content in the wrong place:** Design decisions placed in `work/plan/` instead
of `work/design/` bypass the approval gate. The system cannot enforce what it cannot find.

**Blind retries:** If a tool call fails, read the error message before retrying. Most
failures are caused by drift, missing records, or invalid state — not transient errors.

---

## Related

- `kanbanzai-workflow` — how document approval gates interact with stage progression
- `kanbanzai-agents` — task dispatch and knowledge contribution