---
name: kanbanzai-documents
description: >
  Use this skill whenever you create, edit, register, or approve any document in a
  Kanbanzai-managed project, including files in work/design/, work/spec/, work/plan/,
  work/dev/, work/research/, work/report/, work/review/, work/retro/, or any configured
  document root. Activate even when document registration is not the primary goal of
  the task — any file creation in a document root requires registration.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill covers document types, where to place them, how to register them, and the
approval workflow. Every document placed in a configured root must be registered.

## Document Types and Locations

| Type | Directory | Character | When to use |
|---|---|---|---|
| `design` | `work/design/` | Discursive prose | What to build and why — alternatives considered, decisions made, rationale recorded. No code. No acceptance criteria. |
| `specification` | `work/spec/` | Terse and formal | Verifiable acceptance criteria distilled from an approved design. No code, no implementation notes, no prose that does not directly support a testable criterion. |
| `plan` | `work/plan/` | Structured reference | Project planning: roadmaps, scope, decision logs |
| `dev-plan` | `work/dev/` | Coordination | Task breakdown, dependency graph, parallelism analysis, file ownership, estimates. No implementation code. Interface stubs that define contracts between tasks are acceptable. Uses the approved specification as its basis. |
| `research` | `work/research/` | Exploratory | Analysis, exploration, background reading |
| `report` | `work/report/` | Evaluative | Audit reports, post-mortems, general reports |
| `report` | `work/review/` | Evaluative | Review findings: bugs, deviations from spec, verdict |
| `retrospective` | `work/retro/` | Reflective | Retrospective synthesis documents |

Design decisions belong in `work/design/`, not `work/plan/`. If a document contains
architecture, API shapes, data models, or technology choices, it is a design document.

Acceptance criteria belong in `work/spec/`, not `work/design/`. If you are writing
verifiable pass/fail criteria, you are writing a specification, not a design.

Task breakdowns and dependency graphs belong in `work/dev/`, not `work/spec/`. A
specification that contains code or implementation notes has absorbed dev-plan content.
A dev-plan that contains implementation code has absorbed implementing-agent work —
write the interface, not the implementation.

## Registration Procedure

Every document placed in a configured root must be registered immediately after creation:

    doc(action="register", path="work/design/my-doc.md", type="design", title="Human-readable title")

To batch-import an entire directory (idempotent, safe to repeat):

    doc(action="import", path="work")

## Approval Workflow

Documents follow a three-status lifecycle: **draft → approved → superseded**.

- **Draft:** created, not yet approved. Cannot be used as a stage gate basis.
- **Approved:** approved by a human. An approved design document allows Feature entity
  creation. An approved specification allows task decomposition.
- **Superseded:** replaced by a newer document. The superseding document becomes the
  authoritative basis.

Approval can be verbal; record it immediately:

    doc(action="approve", id="DOC-...")

## Drift and Refresh

If a document is edited after registration, its content hash becomes stale. Attempting
to approve a drifted document will fail.

Check drift status:

    doc(action="get", id="DOC-...")

Refresh before approving:

    doc(action="refresh", id="DOC-...")

## Supersession

When a document is replaced by a newer version:

1. Create and register the new document.
2. Call `doc(action="supersede", id="old-DOC-...", superseded_by="new-DOC-...")`.

Do not silently amend an approved document. Any edit to an approved document requires
creating a new document and superseding the old one.

---

## Gotchas

**Forgot to register:** The document is invisible to document intelligence, entity
extraction, approval workflow, and health checks. Use `doc(action="import", path="work")`
as a safety net to catch any unregistered files.

**Editing after registration:** The content hash becomes stale and approval will fail.
Always call `doc` with action: `refresh` after editing, before approving.

**Design content in the wrong place:** Design decisions placed in `work/plan/` instead
of `work/design/` bypass the approval gate. The system cannot enforce what it cannot find.

**Blind retries:** If a tool call fails, read the error message before retrying. Most
failures are caused by drift, missing records, or invalid state — not transient errors.

---

## Dates and Timestamps

Always call `now()` to get the current UTC datetime before writing any date field in
document content. Never guess or invent a date.

    now(timezone="utc")
    # returns e.g. "2026-03-26T15:44:49Z"

Use full UTC ISO 8601 format (`YYYY-MM-DDTHH:MM:SSZ`) in document metadata headers:

    | Created | 2026-03-26T15:44:49Z |
    | Updated | 2026-03-26T15:44:49Z |

Or for design-doc frontmatter style:

    - Date: 2026-03-26T15:44:49Z

This format is unambiguous and lets any viewer convert to local time. The same UTC ISO
8601 format is used by entity records in `.kbz/state/` — keeping document content
consistent with entity metadata makes the project's timeline easy to reason about.

---

## Related

- `kanbanzai-workflow` — how document approval gates interact with stage progression
- `kanbanzai-agents` — task dispatch and knowledge contribution