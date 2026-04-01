# Specification: Knowledge Auto-Surfacing

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PEF817 (knowledge-auto-surfacing)                    |
| Design  | `work/design/skills-system-redesign-v2.md` §6.3                   |

---

## 1. Overview

This specification defines the automatic surfacing of relevant knowledge entries during context assembly. Instead of requiring agents to manually query the knowledge base, the context assembly pipeline (step 7) automatically selects and includes knowledge entries based on file path matching, tag matching, explicit "always" inclusion, and recency weighting. Surfaced entries appear at a fixed position in the assembled context (position 8, the rising-attention zone), are formatted as actionable "Always/Never X BECAUSE Y" directives, and are capped at 10 entries to preserve the empirically validated n=5-beats-n=19 lean constraint.

---

## 2. Scope

### 2.1 In Scope

- Four matching criteria for selecting knowledge entries: file path matching, tag matching, explicit "always" entries, and recency weighting.
- Scoring and ranking of matched entries by recency-weighted confidence score.
- Hard cap of 10 auto-surfaced entries per assembly.
- Logging of excluded entries when the cap is exceeded.
- Output formatting of surfaced entries as "Always/Never X BECAUSE Y" directives.
- Placement of surfaced entries at position 8 in the assembled context.
- Health recommendation for compaction when the cap is routinely hit.
- Integration with the context assembly pipeline at step 7.

### 2.2 Out of Scope

- Changes to the knowledge entry schema, contribution workflow, or storage format (these are the existing knowledge system's responsibility).
- The context assembly pipeline itself (specified in the Context Assembly Pipeline feature; this spec defines the subsystem invoked at step 7).
- Manual knowledge querying via `knowledge(action: "list")` — that existing capability is unchanged.
- Knowledge compaction logic — only the recommendation trigger is in scope; the compaction mechanism itself is an existing `knowledge` tool capability.
- Token budget estimation of surfaced entries — that is handled by the assembly pipeline at step 9.

---

## 3. Functional Requirements

### FR-001: File Path Matching

The auto-surfacing subsystem MUST match knowledge entries whose scope overlaps with the task's file paths. A knowledge entry scoped to a directory (e.g., `internal/storage/`) MUST match any task that lists a file within that directory or its subdirectories.

**Acceptance criteria:**
- A task with file path `internal/storage/yaml.go` matches a knowledge entry scoped to `internal/storage/`.
- A task with file path `internal/mcp/handoff.go` does NOT match a knowledge entry scoped to `internal/storage/`.
- A task with no file paths produces no file-path-based matches.

---

### FR-002: Tag Matching

The auto-surfacing subsystem MUST match knowledge entries whose tags overlap with the current role's domain tags. A knowledge entry tagged `security` MUST be matched when the resolved role carries a `security` domain tag.

**Acceptance criteria:**
- A role with domain tags `[security, go]` matches knowledge entries tagged `security` and entries tagged `go`.
- A role with domain tags `[testing]` does NOT match a knowledge entry tagged `security`.
- A role with no domain tags produces no tag-based matches.

---

### FR-003: Explicit "Always" Entries

The auto-surfacing subsystem MUST include all knowledge entries tagged `always` or scoped to `project` in every context assembly, regardless of the task's file paths or the role's domain tags.

**Acceptance criteria:**
- A knowledge entry tagged `always` appears in the surfaced results for any task and any role.
- A knowledge entry with scope `project` appears in the surfaced results for any task and any role.
- An entry that is both tagged `always` and matches by file path is not duplicated in the results.

---

### FR-004: Recency Weighting

The auto-surfacing subsystem MUST weight matched entries by recency, preferring recently confirmed entries over older ones. The recency weight MUST be derived from the entry's last confirmation timestamp. Entries that have never been confirmed MUST receive a lower recency weight than confirmed entries.

**Acceptance criteria:**
- Given two entries with equal confidence scores, the entry confirmed more recently ranks higher.
- An entry confirmed today ranks higher than an entry confirmed 30 days ago, all else being equal.
- An entry in `contributed` status (never confirmed) ranks lower than a `confirmed` entry with the same confidence score.

---

### FR-005: Recency-Weighted Confidence Score

The auto-surfacing subsystem MUST compute a composite score for each matched entry that combines the entry's confidence score with its recency weight. This composite score MUST be used to rank entries for selection when the cap is applied.

**Acceptance criteria:**
- A high-confidence entry confirmed recently scores higher than a high-confidence entry confirmed long ago.
- A high-confidence stale entry can still outscore a low-confidence recent entry (confidence is not fully overridden by recency).
- The scoring function is deterministic: the same inputs always produce the same score.

---

### FR-006: Entry Cap

The auto-surfacing subsystem MUST return at most 10 entries. When more than 10 entries match across all matching criteria, the subsystem MUST select the top 10 by recency-weighted confidence score and exclude the rest.

**Acceptance criteria:**
- When 15 entries match, exactly 10 are surfaced and 5 are excluded.
- When 10 or fewer entries match, all matched entries are surfaced.
- When 0 entries match, the subsystem returns an empty result.

---

### FR-007: Excluded Entry Logging

When entries are excluded due to the cap (FR-006), the auto-surfacing subsystem MUST log the IDs and topics of excluded entries. The log MUST be accessible to the orchestrator so that excluded entries can be requested explicitly if needed.

**Acceptance criteria:**
- When 15 entries match and 5 are excluded, the log contains 5 entries each with an ID (e.g., `KE-...`) and topic.
- When all matched entries fit within the cap, no exclusion log is produced.
- The exclusion log is included in the assembly pipeline's diagnostic output, not in the assembled context itself.

---

### FR-008: Output Formatting

Surfaced knowledge entries MUST be formatted as "Always/Never X BECAUSE Y" directives in the assembled context. Each entry MUST include the actionable directive and the rationale. Entries that do not have a BECAUSE clause in their content MUST still be included, formatted with the entry content as the directive and an empty rationale.

**Acceptance criteria:**
- An entry with content "Always use table-driven tests for validation functions BECAUSE they make it trivial to add new cases" is rendered as that exact text.
- An entry with content "Never use global variables for configuration" is rendered with the directive text followed by no BECAUSE clause (or "BECAUSE" followed by the entry's rationale if available in metadata).
- All surfaced entries follow the same format pattern.

---

### FR-009: Position in Assembled Context

Surfaced knowledge entries MUST appear at position 8 in the assembled context — after the output format and examples section (position 7) and before the evaluation criteria section (position 9). This places entries in the rising-attention zone near the end of context.

**Acceptance criteria:**
- In assembled context with all sections present, the knowledge entries section appears immediately after the examples section and immediately before the evaluation criteria section.
- If the examples section is absent, the knowledge entries section still appears before the evaluation criteria section.

---

### FR-010: Within-Section Ordering

Within the knowledge entries section, entries MUST be ordered so that the most relevant entry (highest recency-weighted confidence score) appears LAST, exploiting recency bias.

**Acceptance criteria:**
- The last entry in the knowledge section has the highest recency-weighted confidence score among all surfaced entries.
- The first entry in the knowledge section has the lowest recency-weighted confidence score among all surfaced entries.

---

### FR-011: Deduplication Across Matching Criteria

A knowledge entry that matches via multiple criteria (e.g., matches by file path AND by tag AND is tagged `always`) MUST appear only once in the surfaced results. The entry's score is computed once regardless of how many criteria it matched.

**Acceptance criteria:**
- A knowledge entry scoped to `internal/storage/` and tagged `always`, for a task involving `internal/storage/yaml.go`, appears exactly once in the surfaced results.
- The total count of surfaced entries never exceeds 10, even if many entries match via multiple criteria.

---

### FR-012: Health Compaction Recommendation

The `health` tool MUST flag a recommendation to compact the knowledge base for a given scope when the auto-surfacing cap (10 entries) is routinely exceeded. "Routinely" is defined as the cap being hit on 3 or more consecutive assemblies for the same scope.

**Acceptance criteria:**
- After 3 consecutive assemblies for scope `internal/storage/` where more than 10 entries matched, the `health` tool includes a recommendation to compact knowledge for that scope.
- After 2 consecutive cap-hits followed by an assembly below the cap, the counter resets and no recommendation is produced.
- The recommendation identifies the scope and the number of consecutive cap-hits.

---

### FR-013: Omission When No Entries Match

When the auto-surfacing subsystem returns zero entries, the knowledge section MUST be entirely omitted from the assembled context. No empty section heading, placeholder, or explanatory text MUST appear.

**Acceptance criteria:**
- Assembled context for a task with no matching knowledge entries contains no knowledge section heading or content.
- The section ordering of other sections is unaffected by the omission.

---

### FR-014: Filtering of Retired Entries

The auto-surfacing subsystem MUST NOT surface knowledge entries in `retired` status. Only entries in `contributed`, `confirmed`, or `disputed` status are eligible for matching.

**Acceptance criteria:**
- A `retired` knowledge entry scoped to a matching file path does not appear in surfaced results.
- A `confirmed` knowledge entry and a `retired` knowledge entry for the same scope — only the `confirmed` entry is surfaced.

---

## 4. Non-Functional Requirements

### NFR-001: Surfacing Latency

The auto-surfacing subsystem MUST complete matching, scoring, and selection within 500 milliseconds for a knowledge base of up to 500 entries. The subsystem operates on in-memory or local-filesystem data only.

**Acceptance criteria:**
- Benchmark test with 500 knowledge entries, 10 task file paths, and 5 role tags completes in under 500 milliseconds.

---

### NFR-002: Deterministic Results

Given identical inputs (same task file paths, same role tags, same knowledge base state), the auto-surfacing subsystem MUST return identical results in identical order. The subsystem MUST NOT depend on map iteration order or non-deterministic tie-breaking.

**Acceptance criteria:**
- Running the subsystem twice with the same inputs produces identical surfaced entries in identical order.

---

### NFR-003: Backward Compatibility with Existing Knowledge System

The auto-surfacing subsystem MUST NOT modify, delete, or alter any knowledge entries. It is a read-only consumer of the existing knowledge base. The existing `knowledge(action: "list")` tool MUST continue to function unchanged.

**Acceptance criteria:**
- After auto-surfacing executes, the knowledge base state is identical to before execution.
- `knowledge(action: "list")` returns the same results before and after auto-surfacing runs.

---

### NFR-004: Graceful Degradation

If the knowledge base is unavailable or empty, the auto-surfacing subsystem MUST return an empty result without error. The context assembly pipeline MUST proceed normally, omitting the knowledge section.

**Acceptance criteria:**
- Assembly succeeds with no knowledge section when the knowledge base directory is missing.
- Assembly succeeds with no knowledge section when the knowledge base contains zero entries.
- No error is returned to the caller in either case.

---

## 5. Acceptance Criteria

| Requirement | Verification Method |
|-------------|---------------------|
| FR-001 | Unit test: file path overlap matching with positive and negative cases |
| FR-002 | Unit test: role domain tags match knowledge entry tags |
| FR-003 | Unit test: entries tagged `always` or scoped `project` included for any task |
| FR-004 | Unit test: recently confirmed entries rank above older ones at equal confidence |
| FR-005 | Unit test: composite score combines confidence and recency; deterministic output |
| FR-006 | Unit test: 15 matches → 10 surfaced; 8 matches → 8 surfaced; 0 matches → 0 |
| FR-007 | Unit test: excluded entries logged with IDs and topics; no log when under cap |
| FR-008 | Unit test: entries formatted as "Always/Never X BECAUSE Y" directives |
| FR-009 | Integration test: knowledge section appears at position 8 in assembled context |
| FR-010 | Unit test: highest-scored entry appears last in knowledge section |
| FR-011 | Unit test: entry matching multiple criteria appears exactly once |
| FR-012 | Unit test: health recommendation after 3 consecutive cap-hits; reset after below-cap assembly |
| FR-013 | Integration test: no knowledge section heading in output when 0 entries match |
| FR-014 | Unit test: retired entries excluded from matching |
| NFR-001 | Benchmark test: 500 entries, 10 file paths, 5 tags completes in < 500ms |
| NFR-002 | Test: two identical runs produce byte-identical surfaced entry lists |
| NFR-003 | Test: knowledge base state unchanged after auto-surfacing execution |
| NFR-004 | Test: empty/missing knowledge base → empty result, no error |

---

## 6. Dependencies and Assumptions

### Dependencies

- **Context Assembly Pipeline feature (FEAT-01KN5-88PE43M6):** This subsystem is invoked at step 7 of the assembly pipeline. The pipeline must provide the task's file paths, the resolved role (with domain tags), and an insertion point for the knowledge section at position 8.
- **Existing knowledge system (`internal/knowledge/`):** The subsystem reads knowledge entries via the existing knowledge store. Entries must have: content, scope, tags, status, confidence score, and a last-confirmed timestamp (or equivalent).
- **Role System feature:** Roles must expose domain tags that are used for tag matching (FR-002).

### Assumptions

- Knowledge entries have a `scope` field that can be a file path prefix (e.g., `internal/storage/`) or the string `project`. This is an existing capability of the knowledge system.
- Knowledge entries have a `tags` field (list of strings). This is an existing capability.
- Knowledge entries have a `confidence` score (0.0–1.0) and a timestamp indicating last confirmation. These are existing fields in the knowledge entry schema.
- The role schema includes a mechanism for domain tags (either the existing `tags` field or a dedicated `domain_tags` field). If the role schema does not yet include domain tags, this feature depends on the Role System feature adding them.
- The "routinely exceeded" threshold for health compaction recommendations (FR-012) requires a persistent counter per scope. This counter is maintained by the assembly pipeline or the health subsystem across assemblies.