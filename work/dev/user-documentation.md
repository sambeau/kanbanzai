# User Documentation: Feature Dev-Plan

| Field    | Value                                          |
|----------|------------------------------------------------|
| Feature  | FEAT-01KMKRQVKBPRX — user-documentation        |
| Spec     | work/spec/user-documentation.md               |
| Status   | dev-planning                                   |
| Wave     | 4 (after core features are implemented)        |

---

## Overview

Write the five human-facing reference and tutorial documents that ship with Kanbanzai 1.0.
All documents live in `docs/` and are written against the final 1.0 implementation — not
design documents. No document is complete until it has been verified against the running
binary.

**Dependency:** This feature begins after `init-command`, `public-schema-interface`,
`hardening`, and `binary-distribution` are all done. Documents must describe the actual
1.0 behaviour, not anticipated behaviour.

---

## Tasks

### T1 — `getting-started-doc`
Write `docs/getting-started.md`.

Covers in order: installation (from source + prebuilt binaries), what Kanbanzai is
(2–3 plain-language paragraphs), editor integration for Zed / Claude Desktop / VS Code /
Cursor (each with a copy-pasteable config snippet and a verification step),
`kanbanzai init` walkthrough, and the first-plan tutorial (plan → feature → task →
work queue). Register with `doc_record_submit` on completion.

**Acceptance:** AC-2, AC-3 from the spec. Tutorial is executable from a clean directory.

---

### T2 — `workflow-overview-doc`
Write `docs/workflow-overview.md`.

Covers: human–AI collaboration model (humans own intent, agents own execution),
stage-overview table (all six stages: name, trigger, output, approval gate), per-stage
detail sections, the document-centric interface model, and a common failure modes section
(skipped stages, unauthorised architecture decisions, conflated agent/human context).
Register with `doc_record_submit` on completion.

**Acceptance:** AC-4 from the spec. All six stages described with human approval gates.

---

### T3 — `schema-reference-doc`
Write `docs/schema-reference.md`.

Covers: annotated `.kbz/` directory tree, YAML serialisation rules (block style,
double-quoted strings only when required, deterministic field order, LF + trailing newline,
no tags/anchors/aliases), per-entity-type field tables with required/optional status and
valid enum values plus an example YAML snippet, lifecycle state machine tables for all
entities with a status field, ULID-based ID format, Plan ID `{prefix}{number}-{slug}`
format and prefix registry, and referential integrity rules.
Register with `doc_record_submit` on completion.

**Acceptance:** AC-5, AC-6 from the spec.

---

### T4 — `mcp-tool-reference-doc`
Write `docs/mcp-tool-reference.md`.

Covers: stdio transport description and protocol version, tool organisation by domain
(entity management, document intelligence, knowledge management, worktree operations,
estimation, work queue, orchestration, etc.), and a complete per-tool entry for every
tool in the 1.0 MCP server — each entry includes tool name, one-sentence description,
parameters table (name / type / required / description / valid values), return value
structure, error conditions, and at least one example call + response.
Lifecycle operation constraints and idempotency notes included where applicable.
Register with `doc_record_submit` on completion.

**Acceptance:** AC-7 from the spec. Every 1.0 tool present with parameters and example.

---

### T5 — `configuration-reference-doc`
Write `docs/configuration-reference.md`.

Covers: `config.yaml` — all fields with type, default, and description, plus a minimal
example and an advanced example; prefix registry — declaration, use, retirement, and
constraints; `local.yaml` — per-machine fields, user identity, note that it is not
committed; context profiles — location (`context/roles/`), creation, inheritance;
environment variables; `kbz config validate` (or equivalent) and what errors look like;
migration between versions.
Register with `doc_record_submit` on completion.

**Acceptance:** AC-8 from the spec.

---

## Implementation Notes

- All five documents are independent of each other and can be written in any order,
  but T3 (Schema Reference) benefits from being written last so all entity fields are
  finalised.
- T4 (MCP Tool Reference) must be written against the live server — use `kbz` or the
  MCP inspector to verify every tool's parameters and return shape before writing them
  down.
- Every command invocation, config snippet, and YAML example must be tested against the
  1.0 binary before the document is submitted (spec §6.3, AC-9).
- None of the five documents should contain agent-facing language or tool call syntax
  (AC-10).
- Register each document individually with `doc_record_submit` as it is completed;
  do not batch-import at the end.

---

## Completion Criteria

All five documents exist in `docs/`, are registered with status `approved`, and satisfy
acceptance criteria AC-1 through AC-12 in `work/spec/user-documentation.md`.