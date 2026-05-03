---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: kanbanzai-documents
description: >
  Use when creating, editing, registering, or approving documents in a
  Kanbanzai-managed project. Activates for document types, placement,
  registration with doc action register, approval workflow, content drift,
  doc action refresh, or any question about document status, ownership,
  or lifecycle. Use whenever any markdown file is created or modified in a
  configured document root — even if the edit seems minor, registration
  and refresh rules still apply.
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

## Vocabulary

| Term | Definition |
|------|------------|
| **document record** | The YAML metadata file in `.kbz/state/documents/` that tracks a document's type, status, content hash, and ownership |
| **content hash** | A hash of the document's file contents, recorded at registration and checked at approval to ensure what was reviewed is what gets approved |
| **document type** | One of `design`, `specification`, `dev-plan`, `research`, `report`, or `policy` — determines how the system treats the document in lifecycle gates |
| **document approval** | Calling `doc(action: "approve")` to transition a document from `draft` to `approved`, binding the approval to the current content hash |
| **document registration** | Calling `doc(action: "register")` to create a document record, making the document visible to the workflow system |
| **supersession** | Replacing an approved document with a newer version via `doc(action: "supersede")`, marking the old document as `superseded` |
| **document drift** | The state where a registered document's file contents no longer match its stored content hash, typically after editing |
| **hash refresh** | Calling `doc(action: "refresh")` to update a document record's content hash to match current file contents |
| **document owner** | The parent Plan or Feature ID that a document is associated with, set via the `owner` field at registration |
| **document chain** | The succession history of a document — its predecessors and replacements, retrieved via `doc(action: "chain")` |

---

## Document Creation Checklist

Copy this checklist when creating any new document:

- [ ] Determined the correct document type (design, specification, dev-plan, research, report, policy)
- [ ] Placed the file in the correct directory for its type
- [ ] Registered the document with `doc(action: register, path: "...", type: "...", title: "...", owner: "...")`
- [ ] Verified registration with `doc(action: get, path: "...")`
- [ ] Committed both the document file and the registration record together
- [ ] Classified the document with doc_intel(action: "classify") if content was in context

---

## Document Types and Locations

Documents live in entity-scoped directories under the project's document
directory (typically `work/`). Execution work goes in batch folders; strategic
work goes in plan folders; project-level documents go in `work/_project/`.

### Canonical filename template

All work documents follow one of these templates:

```
{entity-id}-{type}[-{slug}].md              # batch or plan level
{entity-id}-{feature-seq}-{type}[-{slug}].md  # feature-scoped (under a batch)
```

Where:
- `{entity-id}` is the owning entity's ID:
  - `B{n}` for batches (e.g. `B24`) — execution containers that hold features
  - `P{n}` for plans (e.g. `P1`) — strategic containers that hold batches and child plans
- `{feature-seq}` is `F{m}` (e.g. `F3`)
- `{type}` is the document type prefix (`design`, `spec`, `dev-plan`, `review`,
  `report`, `research`, `retro`, `proposal`)
- `[-{slug}]` is an optional lowercase-kebab-case human description

Each identifier appears **exactly once** in the filename. The feature ID
(`F3`) is sufficient because it is batch-scoped — do not repeat the batch ID
or document type.

**Examples:**

| Document | Filename |
|---|---|
| Batch 24's design | `B24-design-auth-system.md` |
| Batch 24, Feature 3's spec | `B24-F3-spec-oauth-flow.md` |
| Batch 24, Feature 3's dev-plan | `B24-F3-dev-plan-oauth-flow.md` |
| Plan 1's design | `P1-design-social-platform.md` |
| Project-level research | `research-ai-orchestration.md` (in `work/_project/`) |

**Character rules:**
- Filenames are lowercase throughout, including the entity ID and feature
  sequence: `b24-f3-spec-oauth-flow.md`
- Exception: the entity and feature prefixes use uppercase `B`/`P` and `F`
  for visual distinction: `B24-F3-spec-oauth-flow.md`
- Slugs use only `[a-z0-9-]`
- No spaces, no underscores, no uppercase in slugs

### Document types

| Type prefix | Type (register as) | When to use |
|---|---|---|
| `design` | `design` | Architecture, vision, approach decisions |
| `spec` | `specification` | Acceptance criteria, binding contracts |
| `dev-plan` | `dev-plan` | Implementation plans, task breakdowns |
| `review` | `review` | Formal review reports |
| `report` | `report` | Internal analyses, evaluations, status |
| `research` | `research` | External research, technology comparisons |
| `retro` | `retrospective` | Retrospectives |
| `proposal` | `proposal` | Early-stage proposals before formal design |

**Note:** `specification` and `retrospective` are accepted as synonyms for the
`spec` and `retro` types respectively when registering. The stored type is
normalised to the short form.

### Folder placement

| Scope | Directory |
|---|---|
| Batch-level documents | `work/{BatchID}-{batch-slug}/` |
| Feature-scoped documents | `work/{BatchID}-{batch-slug}/` (same folder) |
| Plan-level documents | `work/{PlanID}-{plan-slug}/` |
| Project-level documents | `work/_project/` |

**Placement rule:** design content goes in design documents, not in planning
documents. A document whose content contains "Decision:",
"Architecture:", or "Technology Choice:" should be a `design` type.

---

## Registration

Every document placed in a configured root must be registered with the system
immediately after creation. Unregistered documents are invisible to document
intelligence, entity extraction, approval workflow, and health checks.

### Registering a single document

```
doc(
  action="register",
  path="work/B24-auth-system/B24-design-auth-system.md",
  type="design",
  title="Human-Readable Title"
)
```

For a feature-scoped document:

```
doc(
  action="register",
  path="work/B24-auth-system/B24-F3-spec-oauth-flow.md",
  type="specification",
  title="OAuth Flow Specification"
)
```

The system generates a document ID from the path (e.g.
`FEAT-01ABC/spec-oauth-flow`). The filename must follow the canonical
template — each identifier (batch ID, feature ID, document type) appears
exactly once.

### Batch import

To catch unregistered documents or register many at once:

```
doc(action="import", path="work")
```

This is idempotent — already-registered documents are skipped. Safe to run
at any time as a consistency check.

### Verify registration

```
doc(action="get", id="FEAT-01ABC/spec-oauth-flow")
```

A document is properly registered when a YAML record exists in
`.kbz/state/documents/` and `doc` action: `get` returns its metadata.

---

## Classification (Layer 3)

After registering a document, classify it immediately if you have the document
content in context. Do not defer classification to a batch run.

The `doc register` response includes a `classification_nudge` — agents MUST
follow it before moving to the next task.

**Rationale:** Layer 3 classification enables concept search, role-based
retrieval, and semantic guides. Documents deferred to batch runs accumulate as
a growing backlog that is never fully cleared.

### When to run this protocol

- At the start of a new session, to ensure the corpus is up to date.
- After a batch registration (`doc(action: "import")`), to catch newly added documents.
- Periodically, to maintain classification coverage across the corpus.

### Classification-on-registration convention

When you register a document via `doc(action: "register")` and you have the document's
content in context, **classify it immediately** — do not wait for a batch run.
The register response includes a `classification_nudge` that tells you exactly which
`doc_intel` calls to make.

### Priority ordering

Process documents in this order to maximise classification value per unit of effort:

| Priority | Document type | Rationale |
|----------|--------------|-----------|
| 1 | Specifications | Most structured, highest value |
| 2 | Designs | Narrative + decisions |
| 3 | Dev-plans | Task-oriented, lower value |
| 4 | Research/reports | Lowest priority |

You MAY deviate from this ordering when context warrants it (e.g. a design needed
immediately for a concept search query).

**Be concise — no commentary between documents, just run the tools. Report only final counts.**

**Atomicity guarantee:** `classify` calls commit to the persistent index as they succeed — a batch failure does not roll back previously classified documents. After any batch failure, `doc_intel(action: "pending")` is the authoritative ground truth for which documents have already been classified. **Always call `doc_intel(action: "pending")` before re-dispatching a failed batch** to avoid re-classifying documents that were already successfully classified.

### Step-by-step procedure

1. **Get the pending list.** Call `doc_intel(action: "pending")` to retrieve all
   document IDs that have no Layer 3 classifications yet.

2. **Select a batch.** Choose documents to classify in the current session, applying
   the priority ordering above. Size the batch to your available context budget.

3. **For each document in the batch:**
   1. Call `doc_intel(action: "guide", id: "DOC-xxx")` to get the section outline,
      conventional roles, entity refs, and content hash.
   2. Read the sections you need (use `doc_intel(action: "section", ...)` if the
      document is large and you only need specific sections).
   3. Produce a classification object for each section in the outline, assigning a
      role from the taxonomy below.
   4. Call `doc_intel(action: "classify", id: "DOC-xxx", content_hash: "...", model_name: "...", model_version: "...", classifications: "[...]")`
      to submit the classifications.

   **Classification object format** (`classifications` is a JSON-encoded string):
   ```json
   [
     {"section_path": "1",   "role": "narrative",    "confidence": "high"},
     {"section_path": "1.1", "role": "requirement",  "confidence": "high"},
     {"section_path": "1.2", "role": "rationale",    "confidence": "medium"}
   ]
   ```

   **Valid roles** (choose the one that best describes the section's primary content):

   | Role | When to use |
   |------|-------------|
   | `requirement` | Acceptance criteria, must/shall/must-not statements, ACs |
   | `decision` | A choice that was made, an ADR, a "we will do X" statement |
   | `rationale` | Explanation of *why* — motivation, problem statement, purpose |
   | `constraint` | Scope boundaries, in-scope/out-of-scope, deferred items, exclusions |
   | `assumption` | Things assumed true that haven't been verified |
   | `risk` | Identified risks or mitigations |
   | `question` | Open questions, TBDs |
   | `definition` | Glossary entries, term definitions, data schemas |
   | `example` | Worked examples, sample code, sample payloads |
   | `alternative` | Options considered but not chosen |
   | `narrative` | Everything else — introductions, summaries, front matter, file lists |

   **Valid confidence values:** `"high"`, `"medium"`, `"low"`

   **Optional fields per object:** `summary` (one-line description), `concepts_intro`
   (array of concept names introduced), `concepts_used` (array of concept names referenced).

4. **Repeat** until the pending list is empty or your context budget is exhausted.
   Re-call `doc_intel(action: "pending")` to confirm progress.

### Anti-patterns

- **Skipping `guide` before `classify`.** The `guide` response provides the content
  hash required by `classify`. Never construct the content hash manually.
- **Classifying without reading sections.** Classification requires understanding the
  content. Do not assign roles without reading the relevant section text.
- **Classifying the whole corpus in one pass.** Work in batches sized to your context
  window. Quality drops when context is saturated.

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

### Auto-Approve for Agent-Authored Documents

For agent-authored documents of types `dev-plan`, `research`, and `report`,
registration and approval can be combined in a single call:

```
doc(action: "register", type: "dev-plan", auto_approve: true, path: "...", title: "...", owner: "...")
```

This registers the document and immediately approves it. The `auto_approve`
flag is only honoured for types in the whitelist (`dev-plan`, `research`,
`report`). Design documents and specifications always require explicit human
approval.

---

## Drift and Refresh

When a document is registered, the system records a content hash. If the
file is edited after registration, the hash becomes stale — this is called
**drift**.

- **`doc approve` auto-refreshes on hash mismatch.** If the file has drifted
  since registration, `doc` action: `approve` automatically updates the hash
  before approving — it no longer fails on mismatch.
- **`doc refresh` is still available** for checking or correcting drift
  outside the approval path (e.g. to detect whether a document has changed
  since it was last registered).
- **If an approved document is edited**, the approval is effectively void.
  The content no longer matches what was approved. Notify the human and
  re-approve after review.

---

## Supersession

When a document is replaced by a newer version:

1. Create and register the new document.
2. Call `doc` action: `supersede` on the old document, linking it to the new one.
3. The old document's status becomes `superseded`.

Superseded documents remain in the repository as historical records. They are
no longer authoritative.

---

## Moving Documents

To move a document file to a new path:

1. Call `doc(action: "move", id: "DOC-xxx", new_path: "work/new-location/file.md")`
2. The system moves the file, updates the record's path, recomputes the content hash, and commits atomically.
3. If the new path implies a different document type, the type is updated automatically.

**Note:** Approval status, owner, and cross-references are preserved across moves.

---

## Deleting Documents

To delete a document:

1. Call `doc(action: "delete", id: "DOC-xxx")` for draft documents.
2. For approved documents, add `force: true`: `doc(action: "delete", id: "DOC-xxx", force: true)`
3. The system removes the file, clears the entity's document reference, and commits atomically.

**Note:** This operation is irreversible. Consider supersession instead if historical preservation matters.

---

## Anti-Patterns

### Unregistered Document

- **Detect:** A design, spec, or plan document exists on disk but has no
  document record in `.kbz/state/documents/`.
- **BECAUSE:** Unregistered documents are invisible to the workflow system —
  stage gates can't check prerequisites, and other agents can't discover the
  document through MCP tools.
- **Resolve:** Call `doc(action: "register")` immediately after creating any
  document.

### Stale Content Hash

- **Detect:** A registered document's file contents no longer match its stored
  content hash (visible via `doc(action: "refresh")`).
- **BECAUSE:** The file was edited after registration, so the stored hash no
  longer matches disk.
- **Note:** `doc(action: "approve")` now auto-refreshes the hash before
  approving, so this no longer blocks approval. Use `doc(action: "refresh")`
  explicitly if you need to check or update drift outside the approval path.

### Silent Supersession

- **Detect:** An approved document is edited directly instead of creating a
  replacement and superseding.
- **BECAUSE:** The approval is tied to the content hash. Editing voids the
  approval silently — downstream consumers see "approved" but the content
  has drifted.
- **Resolve:** Create a new document, register it, and call
  `doc(action: "supersede")` on the old one.

---

## Gotchas

- **Forgot to register.** If you create a file in `work/` and forget to call
  `doc` action: `register`, the document is invisible to the system. Run
  `doc(action="import", path="work")` as a safety net if unsure.
- **Design in the wrong place.** Design decisions belong in design
  documents, not planning documents. If a dev-plan contains architecture
  decisions, move that content to a design document.
- **Registering with the wrong type.** A specification registered as `design`
  won't be found when the system checks for specification prerequisites.
  Check the Document Types table — the type determines how the system treats
  the document in lifecycle gates.
- **Tool call fails.** If `doc` action: `register` or `doc` action: `approve`
  returns an error, read the message — it explains the problem. Do not retry
  with the same arguments. Fix the underlying issue first.

---

## Commit Discipline

Document registration and approval are automatically committed by the MCP
tools. Manual commits for document files are no longer required.

If you create or edit a document file outside of an MCP tool (e.g. writing
content directly to disk), commit the file itself using the standard commit
format — the registration record will be handled by the tool when you call
`doc(action: "register")`.

---

## Output Templates

When producing specific document types, use these templates as structural
guides:

- **Specifications:** `work/templates/specification-prompt-template.md`
- **Implementation plans:** `work/templates/implementation-plan-prompt-template.md`
- **Reviews:** `work/templates/review-prompt-template.md`

---

## Evaluation Criteria

| # | Question | Weight |
|---|----------|--------|
| 1 | Was every new document registered via `doc(action: "register")` immediately after creation? | required |
| 2 | Was `doc(action: "approve")` called immediately when approval was signalled? | required |
| 3 | Were edited approved documents handled via supersession rather than direct edit? | high |
| 4 | Was `doc(action: "refresh")` used to resolve content hash mismatches before re-approval? | high |

---

## Questions This Skill Answers

- How do I register a new document?
- How do I approve a document?
- What do I do when approval fails with a hash mismatch?
- How do I supersede an old document?
- What document types does the system support?
- When should I call `doc(action: "refresh")`?
- How do I check if all specs for a plan are approved?

---

## Related

- `kanbanzai-getting-started` — session orientation
- `kanbanzai-workflow` — stage gates that depend on document approval
- `write-design` — the design process that produces design documents