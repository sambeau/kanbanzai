# Specification: Batch Classification Activation

| Field       | Value |
|-------------|-------|
| Feature     | FEAT-01KPNNYYXQSYW — Batch Classification Activation |
| Design ref  | `work/design/doc-intel-enhancement-design.md` §5 |
| Status      | Draft |
| Author      | spec-author |

---

## Overview

This specification defines the batch classification activation feature, which enables
efficient Layer 3 classification of the existing document corpus and prevents the
classified corpus from going stale. It introduces a single code change — a nudge
message in the `doc(action: "register")` response — and formalises a batch
classification workflow protocol as a documented convention for agents to follow when
classifying documents at scale. The feature activates capabilities that already exist
in `doc_intel(action: "classify")` and `doc_intel(action: "pending")` but have never
been exercised.

## Scope

### In scope

- Classification nudge message added to `doc(action: "register")` response
- Batch classification workflow protocol (documented convention)
- Classification priority ordering convention (by document type)
- Classification-on-registration convention (agent responsibility, not enforcement)

### Out of scope

- New MCP tool actions or parameters (Design §5.5: no new tool or action)
- Server-side enforcement of classification after registration
- Automatic/server-triggered classification
- Changes to the `doc_intel(action: "classify")` or `doc_intel(action: "pending")` actions
- Changes to the classification storage or Layer 3 data model

## Functional Requirements

### FR-001: Classification nudge on register response

The `doc(action: "register")` response MUST include a `classification_nudge` field
containing a human-readable message that tells the agent how to classify the newly
registered document.

The nudge message MUST include:

1. A statement that Layer 3 classification is pending
2. The document ID for use in subsequent `doc_intel` calls
3. The two-step workflow: call `guide` first, then `classify`

**Traceability:** Design §5.4, §8.3

### FR-002: Nudge includes document ID

The classification nudge MUST interpolate the registered document's actual ID (e.g.
`DOC-01KP...`) so the agent can use it directly without extracting it from elsewhere
in the response.

**Traceability:** Design §5.4

### FR-003: Nudge present for both single and batch registration

The classification nudge MUST appear in the response for single-document registration.
For batch registration (via the `documents` array), each individual result MUST include
its own nudge with that document's ID.

**Traceability:** Design §5.4

### FR-004: Nudge is informational only

The classification nudge MUST NOT change the response schema in a breaking way. It
MUST be an additional field alongside the existing `document` object. Existing fields
(`document`, `warnings`) MUST NOT be removed or renamed.

**Traceability:** Design §5.5

### FR-005: Batch classification workflow protocol

The project SHOULD document a batch classification workflow protocol that describes:

1. Start by calling `doc_intel(action: "pending")` to get unclassified document IDs
2. Select a batch of documents to classify
3. For each document: call `guide`, read sections, produce classifications, call `classify`
4. Repeat until the pending list is empty or the agent's budget is exhausted

This protocol uses existing tools only. It is a convention, not enforced by code.

**Traceability:** Design §5.3

### FR-006: Classification priority ordering

The batch classification protocol SHOULD define the following priority ordering for
document types:

| Priority | Document type    | Rationale                                          |
|----------|------------------|----------------------------------------------------|
| 1        | Specifications   | Most structured, highest value from classification |
| 2        | Designs          | Narrative + decisions, good concept extraction targets |
| 3        | Dev-plans        | Task-oriented, lower classification value          |
| 4        | Research/reports | Lowest priority                                    |

Agents MAY deviate from this ordering when context warrants it (e.g. classifying a
design that is actively needed for a search query).

**Traceability:** Design §5.3 (step 2)

### FR-007: Classification-on-registration convention

When an agent registers a document via `doc(action: "register")` and has the
document content in context, the agent SHOULD classify it immediately by following
the `guide` → `classify` workflow described in the nudge.

This is a workflow convention. The server MUST NOT reject registrations that are not
followed by classification. The server MUST NOT block or delay the register response
to wait for classification.

**Traceability:** Design §5.4

## Non-Functional Requirements

### NFR-001: Nudge message size

The nudge message SHOULD be concise — no more than three lines of text — to avoid
inflating MCP response payload size when processing batch registrations.

### NFR-002: No performance impact on registration

Adding the nudge field MUST NOT measurably increase the latency of `doc(action: "register")`.
The nudge is a static string template with ID interpolation only.

## Acceptance Criteria

- [ ] **FR-001:** `doc(action: "register")` response contains a `classification_nudge`
  string field that mentions Layer 3 classification, the document ID, and the
  `guide` → `classify` workflow
- [ ] **FR-002:** The nudge text contains the actual registered document ID (not a
  placeholder or generic string)
- [ ] **FR-003 (single):** Registering a single document returns a response with
  `classification_nudge` present
- [ ] **FR-003 (batch):** Registering multiple documents via `documents` array returns
  each individual result with its own `classification_nudge`
- [ ] **FR-004:** Existing response fields (`document.id`, `document.path`,
  `document.type`, `document.title`, `document.status`, `warnings`) remain unchanged
- [ ] **FR-005:** Batch classification workflow protocol is documented (in skill file,
  reference doc, or protocol description)
- [ ] **FR-006:** Protocol documentation lists the priority ordering: specifications →
  designs → dev-plans → research/reports
- [ ] **FR-007:** Registering a document without classifying it succeeds without error
  (convention, not enforcement)
- [ ] **NFR-001:** Nudge message is three lines or fewer
- [ ] **NFR-002:** Registration latency is not measurably affected (no new I/O or
  computation in the nudge path)

## Dependencies and Assumptions

### Dependencies

- `doc_intel(action: "classify")` — must accept classifications and store them correctly.
  Currently implemented in `doc_intel_tool.go:docIntelClassifyAction`.
- `doc_intel(action: "pending")` — must return the list of unclassified document IDs.
  Currently implemented in `doc_intel_tool.go:docIntelPendingAction`.
- `doc_intel(action: "guide")` — must return the outline, conventional roles, entity refs,
  and content hash needed for classification submission.
- Layer 1–2 indexing via `IngestDocument` on registration — must continue to work as-is.

### Assumptions

- The `classify`, `pending`, and `guide` actions are functionally correct. This feature
  does not modify them.
- The existing `docRegisterOne` function in `doc_tool.go` is the single code path for
  constructing the register response (both single and batch paths delegate to it).
- Agents can read and act on the nudge message without additional prompting infrastructure.
