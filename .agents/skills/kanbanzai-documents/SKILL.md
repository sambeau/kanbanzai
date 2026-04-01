---
name: kanbanzai-documents
description: >
  Use when creating, editing, registering, or approving documents in a
  Kanbanzai-managed project. Activates for document types, placement,
  registration with doc action register, approval workflow, content drift,
  doc action refresh, or any question about document status, ownership,
  or lifecycle. Use whenever any markdown file is created or modified in a
  configured document root — even if the edit seems minor, registration
  and refresh rules still apply.
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Documents

## Purpose

Document types, where to place them, how to register them with the system,
and the approval workflow that makes them authoritative.

## When to Use

- When creating any new document in a configured document root
- When registering a document with `doc` action: `register`
- When approving a document or checking whether it is ready for approval
- When editing a document that has already been registered
- When unsure which document type or directory to use

---

## Document Creation Checklist

Copy this checklist when creating any new document:

- [ ] Determined the correct document type (design, specification, dev-plan, research, report, policy)
- [ ] Placed the file in the correct directory for its type
- [ ] Registered the document with `doc(action: register, path: "...", type: "...", title: "...", owner: "...")`
- [ ] Verified registration with `doc(action: get, path: "...")`
- [ ] Committed both the document file and the registration record together

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
| `policy` | `work/design/` | Standing rules, process definitions |

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
doc(
  action="register",
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
doc(action="import", path="work")
```

This is idempotent — already-registered documents are skipped. Safe to run
at any time as a consistency check.

### Verify registration

```
doc(action="get", id="PROJECT/design-my-document")
```

A document is properly registered when a YAML record exists in
`.kbz/state/documents/` and `doc` action: `get` returns its metadata.

---

## Approval Workflow

Documents follow a three-status lifecycle: **draft → approved → superseded**.

1. Agent or human creates the document file.
2. Agent registers it with `doc` action: `register` — status becomes `draft`.
3. Human reviews the document and signals approval (verbally: "Approved",
   "LGTM", or equivalent).
4. Agent calls `doc` action: `approve` — status becomes `approved`.
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
- **After editing a registered document**, call `doc` action: `refresh` to
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
2. Call `doc` action: `supersede` on the old document, linking it to the new one.
3. The old document's status becomes `superseded`.

Superseded documents remain in the repository as historical records. They are
no longer authoritative.

---

## Anti-Patterns

**Creating documents without registering them.** An unregistered document is invisible to the workflow system — it won't appear in document gap analysis, won't be tracked for staleness, and won't participate in lifecycle gates. Every document in a configured document root must be registered.

**Editing an approved document without refreshing.** When you edit a document that has `approved` status, its content hash changes but the approval status doesn't automatically update. Call `doc(action: refresh)` after significant edits so the system detects the drift and demotes the document back to `draft` for re-approval.

**Registering with the wrong type.** A specification registered as `design` won't be found when the system checks for specification prerequisites. Check the Document Types table — the type determines how the system treats the document in lifecycle gates.

---

## Gotchas

- **Forgot to register.** If you create a file in `work/` and forget to call
  `doc` action: `register`, the document is invisible to the system — no approval
  workflow, no document intelligence, no health check coverage. This is the
  single most common mistake. Run `doc(action="import", path="work")` as a
  safety net if unsure.
- **Editing after approval.** If you edit an approved document, the approval
  is silently void — the content hash no longer matches. You must notify the
  human and re-approve. Do not assume the approval still holds.
- **Design in the wrong place.** Design decisions belong in `work/design/`,
  not `work/plan/`. If a planning document starts containing architecture
  decisions, move that content to a design document. See `kanbanzai-workflow`
  for the emergency brake rules.
- **Tool call fails.** If `doc` action: `register` or `doc` action: `approve`
  returns an error, read the message — it usually explains the problem (wrong
  type, drifted hash, document already exists). Do not retry with the same
  arguments. Fix the underlying issue first.

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

## Output Templates

When producing specific document types, use these templates as structural guides:

- **Specifications:** See `work/templates/specification-prompt-template.md` for required sections and example requirements
- **Implementation plans:** See `work/templates/implementation-plan-prompt-template.md` for task breakdown structure
- **Reviews:** See `work/templates/review-prompt-template.md` for finding format and severity levels

---

## Related

- `kanbanzai-getting-started` — session orientation
- `kanbanzai-workflow` — stage gates that depend on document approval
- `kanbanzai-design` — the design process that produces design documents