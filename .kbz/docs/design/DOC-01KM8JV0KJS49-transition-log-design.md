---
id: DOC-01KM8JV0KJS49
type: design
title: Transition Log Design
status: submitted
feature: FEAT-01KM8JT7542GZ
created_by: human
created: 2026-03-21T16:14:48Z
updated: 2026-03-21T16:14:48Z
---
# Transition Log Design

- Status: design
- Purpose: design for recording lifecycle transition history on entities
- Date: 2026-07-24
- Phase: post-Phase 1 (Phase 1 derives transition history from Git; this design covers the dedicated transition log planned for later phases)
- Basis:
  - `workflow-design-basis.md` §9 (lifecycle state machines)
  - `phase-1-specification.md` §10 (lifecycle requirements)
  - `phase-1-decision-log.md` P1-DEC-010 (lifecycle transition graph)

---

## 1. Purpose

This document defines a transition log for lifecycle state changes on Kanbanzai entities. The goal is to provide a queryable, structured history of how entities moved through their lifecycles — who triggered each transition, when, and why.

Phase 1 derives transition history from Git commit history. This works for a single-agent, low-volume system but has limitations that become significant as the system scales to multi-agent use:

- If an agent changes status twice in one commit, the intermediate state is lost.
- Querying Git history for a specific entity's transitions is slow and requires parsing commit messages.
- Git history records file changes, not semantic transitions — extracting "who moved this from `active` to `needs-review` and why" requires convention-dependent parsing.
- Multiple agents working concurrently produce interleaved commit histories that are hard to read per-entity.

A dedicated transition log solves these problems by recording each transition as a discrete, structured event.

---

## 2. Design Principles

### 2.1 Append-only

Transition log entries are immutable once written. Corrections are modelled as new transitions, not edits to old entries. This preserves the audit trail.

### 2.2 Per-entity, not global

Each entity owns its own transition history. There is no single global log file. This keeps entity state self-contained and avoids contention on a shared resource.

### 2.3 Derived, not authoritative

The `status` field on the entity record remains the source of truth for current state. The transition log is a historical record. If the log and the current status disagree, the current status wins and the log should be corrected or flagged by health checks.

### 2.4 Optional in early phases

The transition log is additive — entities without transition history are valid. This allows gradual adoption and preserves backwards compatibility with Phase 1 entities that have no log.

---

## 3. Transition Entry Schema

Each entry in the transition log records a single lifecycle transition.

### 3.1 Required fields

| Field | Type | Description |
|---|---|---|
| `from` | string | The status before the transition |
| `to` | string | The status after the transition |
| `at` | timestamp | When the transition occurred (UTC, RFC 3339) |
| `by` | string | Who or what triggered the transition (human username or agent identifier) |

### 3.2 Optional fields

| Field | Type | Description |
|---|---|---|
| `reason` | string | Free-text explanation of why the transition was made |
| `commit` | string | Git commit SHA that contains this transition, if known |

---

## 4. Storage

### 4.1 On-entity storage

The transition log is stored as a `transitions` field on the entity record — a YAML sequence of transition entries in chronological order.

```
transitions:
  - from: queued
    to: ready
    at: "2026-07-20T10:30:00Z"
    by: agent-alpha
  - from: ready
    to: active
    at: "2026-07-21T09:15:00Z"
    by: agent-beta
    reason: "dependencies resolved, starting implementation"
  - from: active
    to: needs-review
    at: "2026-07-22T16:45:00Z"
    by: agent-beta
```

### 4.2 Rationale for on-entity storage

The alternative — storing transition history in a separate file or in the cache only — was considered and rejected:

- **Separate files** add filesystem complexity (one more file per entity) and make it harder to move or archive entities atomically.
- **Cache-only storage** makes history disposable, which contradicts the audit purpose.

On-entity storage keeps the entity self-contained: one file contains everything about the entity, including its history. The cost is that entity files grow over time, but lifecycle transitions are low-frequency events — a typical entity might accumulate 5–15 transitions over its lifetime, adding a modest amount of YAML.

### 4.3 Field ordering

The `transitions` field should appear last in the entity's canonical field order. It is historical metadata, not primary entity data, and placing it last keeps the entity's core fields visible at the top of the file.

### 4.4 Cache integration

The cache should index transition log entries to support efficient queries (e.g., "show all entities that transitioned to `needs-review` this week" or "show all transitions by agent-beta"). The cache representation is a derived index — if lost, it is rebuilt from entity files.

---

## 5. Recording Transitions

### 5.1 When to record

A transition log entry is created whenever the system processes a valid status change through the `update_status` operation (MCP or CLI). The entry is written atomically with the status field update — the entity file is never in a state where `status` has changed but the log entry is missing.

### 5.2 Entry state recording

When an entity is created, the system should record an initial transition entry with `from` set to an empty string (or a sentinel like `_created`) and `to` set to the entry state. This establishes the creation event in the log.

Example:

```
transitions:
  - from: ""
    to: reported
    at: "2026-07-20T08:00:00Z"
    by: human-sam
    reason: "filed during code review"
```

### 5.3 The `by` field

The `by` field records the identity of the actor that triggered the transition. In a multi-agent system, this distinguishes between:

- Human users (e.g., `human-sam`)
- AI agents (e.g., `agent-alpha`, `copilot-workspace`)
- System-initiated transitions (e.g., `system` for automated state changes, if any are introduced in future phases)

The exact identity format is not defined here — it should align with whatever identity model the system uses for `created_by`, `reported_by`, `decided_by`, and similar fields.

---

## 6. Querying Transition History

### 6.1 Per-entity history

The simplest query: retrieve an entity and read its `transitions` field. This gives the full chronological history of that entity's lifecycle.

### 6.2 Cross-entity queries

Through the cache index, the system should support:

- **By actor:** "Show all transitions triggered by agent-beta" — useful for reviewing an agent's work.
- **By state:** "Show all entities that entered `needs-review` in the last week" — useful for review queues.
- **By time range:** "Show all transitions in the last 24 hours" — useful for activity summaries.
- **By entity type:** "Show all Bug transitions" — useful for defect workflow analysis.

These are Phase 2+ query capabilities. Phase 1 cache-based queries are list-by-type only (P1-DEC-020).

---

## 7. Health Check Integration

When the transition log is present, health checks should verify:

1. **Consistency:** The last entry's `to` field matches the entity's current `status`.
2. **Legality:** Every `from → to` pair in the log is a legal transition per the entity's state machine (P1-DEC-010).
3. **Chronological order:** Entries are in ascending `at` order.
4. **No gaps:** Every entry's `from` matches the previous entry's `to` (the log forms a continuous chain).

Health check failures in the transition log should be reported as warnings, not errors, since the log is a derived historical record and the `status` field remains authoritative.

---

## 8. Migration From Phase 1

Phase 1 entities will not have a `transitions` field. When the transition log is introduced:

- Existing entities are valid without a `transitions` field. The field is optional.
- The system may attempt to backfill transition history from Git commit history for existing entities, but this is best-effort and not required.
- New transitions on existing entities will start recording from the point the feature is enabled, creating a partial log. This is acceptable — a partial log is better than no log.

---

## 9. Relationship to Concurrency Controls

The transition log interacts with the planned cache-based optimistic locking mechanism:

- When two agents attempt to transition the same entity simultaneously, the locking mechanism rejects the second write.
- The rejected agent sees the updated entity (with the first agent's transition logged) and can decide whether its intended transition is still valid.
- The transition log provides evidence for diagnosing contention — if an entity shows rapid transitions by different agents, it may indicate a coordination problem.

---

## 10. Open Questions

1. **Should the `by` field support structured identifiers?** A plain string is simple, but a structured identifier (e.g., `type: agent, name: alpha, session: xyz`) would enable richer queries. Likely overkill for now.

2. **Should transitions record the MCP operation or CLI command that triggered them?** This would aid debugging but adds coupling between the log format and the interface layer.

3. **Maximum log length?** Entity files grow with each transition. Should there be a cap, with older entries archived or summarised? Probably unnecessary — even a long-lived entity is unlikely to exceed a few dozen transitions.

4. **Should the initial creation entry use `from: ""` or a sentinel like `from: _created`?** An empty string is minimal; a sentinel is more explicit. Either works; the choice should be consistent.

---

## 11. Summary

- Phase 1 derives transition history from Git. This is sufficient for single-agent use.
- Post-Phase 1, a `transitions` field on each entity records structured transition entries (from, to, at, by, reason).
- The log is append-only, per-entity, and derived (the `status` field remains authoritative).
- Entries are written atomically with status changes.
- The cache indexes transition entries for cross-entity queries.
- Health checks verify log consistency, legality, chronological order, and continuity.
- Existing Phase 1 entities remain valid without a transition log. The field is optional and adoption is gradual.