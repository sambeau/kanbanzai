# Specification: Document Structural Checks and Quality Hooks (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J26RSB6 (document-structural-checks)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §10.4, §10.5, §16.1 Q3, §16.2 Q2
**Status:** Draft

---

## Overview

This specification defines automated structural checks that run when a feature transitions past a document-producing stage gate, a quality evaluation structural hook that enables agent-side document quality assessment without adding LLM capabilities to the server, and a promotion mechanism that graduates individual checks from non-blocking warnings to hard gates based on observed reliability. The structural checks use the existing document intelligence parser (`internal/docint/`) to verify required sections, cross-references, and acceptance criteria in registered documents. The quality evaluation hook adds a `quality_evaluation` record to the document model that the server can verify structurally — existence and score threshold — without making any LLM calls.

---

## Scope

### In scope

- Structural check definitions: required sections per document type, cross-reference validation, acceptance criteria detection
- Integration point with the feature lifecycle transition mechanism (runs at stage gates as an additional validation layer)
- Warning mode (non-blocking, logged) as the initial enforcement level for all structural checks
- Hard gate mode (blocking) as the promoted enforcement level
- Per-check promotion from warning to hard gate based on consecutive clean passes
- Demotion from hard gate back to warning on false positive detection
- Promotion state persistence
- The `quality_evaluation` record schema and its attachment to document records
- The structural gate on document approval: server checks evaluation exists and score meets threshold
- Actionable error messages for structural check failures (per design §6, WP-3)
- Hardcoded default required-section definitions per document type (`design`, `specification`, `dev-plan`)

### Explicitly excluded

- LLM-as-judge evaluation logic — that is an agent-side skill (design §10.5, §16.1 Q3); only the structural hook is specified here
- Document template content and authoring skills — those belong to the skills redesign
- The document intelligence parser itself — already exists in `internal/docint/`
- Gate enforcement mechanism for lifecycle transitions — that is the mandatory-stage-gates feature (FEAT-01KN5-8J24S2XW); this feature adds checks that run AT gates
- Binding registry integration for reading required sections from `document_template` structures — that is the binding-registry-gate-integration feature (FEAT-01KN5-8J27H83N); this feature uses hardcoded defaults
- Review report parsing for the `reviewing → done` gate — that is specified separately under the review-rework-loop feature
- Quality evaluation rubric content, prompt design, or dimension weighting — those are agent-side skill concerns

---

## Functional Requirements

### Structural Check Execution

**FR-001:** The system MUST execute structural checks when a feature transitions past a document-producing stage. Structural checks MUST fire at the following gates, examining the approved document that satisfied the document prerequisite for that gate:

| Gate transition | Document checked | Checks applied |
|---|---|---|
| `designing → specifying` | Approved design document | Required sections |
| `specifying → dev-planning` | Approved specification document | Required sections, cross-reference to design |
| `dev-planning → developing` | Approved dev-plan document; approved specification document | Required sections (dev-plan), acceptance criteria present (specification) |

**Acceptance criteria:**
- Transitioning a feature from `designing` to `specifying` triggers structural checks on the approved design document
- Transitioning a feature from `specifying` to `dev-planning` triggers structural checks on the approved specification document
- Transitioning a feature from `dev-planning` to `developing` triggers structural checks on the approved dev-plan document and the approved specification document
- Structural checks run regardless of whether the transition is a single step or part of an `advance` sequence
- If no approved document is found for a gate, the structural check step is skipped (the document prerequisite gate is a separate concern that handles missing documents)

---

**FR-002:** Structural checks MUST use the existing document intelligence parser (`internal/docint/ParseStructure`) to obtain the section tree of the document being checked. The check compares the parsed section headings against the required section list for the document type.

**Acceptance criteria:**
- The structural checker calls `ParseStructure` on the document content and inspects the returned `[]Section` tree
- Section matching is case-insensitive and uses substring containment (e.g., a heading "Functional Requirements and Constraints" satisfies the required section "Functional Requirements")
- Only top-level and second-level headings (levels 1–3 in the Markdown heading hierarchy) are considered for required section matching
- A document with no parseable sections (empty or non-Markdown content) reports all required sections as missing

---

### Required Section Definitions

**FR-003:** The system MUST define required sections per document type as hardcoded defaults. The following sections are required for each document type that participates in stage gate checks:

**Design documents** (`design`):
1. A section whose heading contains "overview" OR "purpose" OR "summary"
2. A section whose heading contains "design" (excluding the document title itself)

**Specification documents** (`specification`):
1. A section whose heading contains "overview"
2. A section whose heading contains "scope"
3. A section whose heading contains "functional requirements"
4. A section whose heading contains "acceptance criteria"

**Dev-plan documents** (`dev-plan`):
1. A section whose heading contains "overview"
2. A section whose heading contains "task" (to match "Tasks", "Task Breakdown", "Task List", etc.)

**Acceptance criteria:**
- A design document with an "Overview" section and a "Design Principles" section passes the required sections check
- A design document with only a "Purpose" section and no design-related section fails with one missing section reported
- A specification document with "Overview", "Scope", "Functional Requirements", and "Acceptance Criteria" sections passes
- A specification document missing "Scope" fails and the failure message names "scope" as the missing required section
- A dev-plan document with "Overview" and "Task Breakdown" sections passes
- A dev-plan document with only "Overview" and no task-related section fails
- The document's own H1 title heading is excluded from matching (to prevent the title "Design: Foo" from satisfying the "design" requirement)

---

**FR-004:** Required section definitions MUST be organized as a lookup table keyed by document type string, returning an ordered list of section requirements. Each section requirement specifies a human-readable label (for error messages) and one or more match keywords. A section satisfies a requirement if its heading contains any of the requirement's keywords (case-insensitive substring match).

**Acceptance criteria:**
- The lookup table returns the correct section requirements for document types `design`, `specification`, and `dev-plan`
- The lookup table returns an empty list for document types that have no structural check definitions (e.g., `research`, `report`, `policy`)
- Each section requirement has a label and at least one keyword
- The section requirement for "overview" in specification documents can be satisfied by headings titled "Overview", "1. Overview", "Project Overview", or "## Overview"

---

### Cross-Reference Validation

**FR-005:** At the `specifying → dev-planning` gate, the system MUST verify that the approved specification document contains a cross-reference to a design document. A cross-reference is satisfied if any of the following are true:
1. The document's `CrossDocLinks` (extracted by `internal/docint/ExtractPatterns`) include a link whose target path matches the path of an approved design document owned by the same feature or its parent plan
2. The document's `EntityRefs` include a reference to a document ID (`DOC-...`) that resolves to an approved design document owned by the same feature or its parent plan
3. The document's front matter or content contains a Markdown link or backtick-quoted path to an approved design document

**Acceptance criteria:**
- A specification containing `[design](work/design/my-design.md)` where `work/design/my-design.md` is the path of an approved design document owned by the feature passes the cross-reference check
- A specification containing `` `work/design/my-design.md` `` where that path matches an approved design document passes
- A specification with no links or references to any design document fails the cross-reference check
- A specification that references a design document owned by the feature's parent plan passes (plan-level design documents satisfy feature-level gates)
- The cross-reference check uses the document intelligence extractor's existing `CrossDocLink` and `EntityRef` output — it does not re-parse the document

---

### Acceptance Criteria Detection

**FR-006:** At the `dev-planning → developing` gate, the system MUST verify that the approved specification document contains at least one acceptance criterion. An acceptance criterion is detected if any of the following are true:
1. The document contains a section whose heading matches the conventional role keyword "acceptance criteria" (as defined in `internal/docint/taxonomy.go` `conventionalRoleKeywords`)
2. The document's `ConventionalRoles` (from the extractor) include at least one entry with role `requirement` associated with a heading containing "acceptance criteria"
3. The document contains a section whose heading contains "acceptance" and whose content is non-empty (word count > 0)

**Acceptance criteria:**
- A specification with a section titled "Acceptance Criteria" containing text passes
- A specification with a section titled "## Acceptance Criteria" containing bullet points passes
- A specification with a section titled "Acceptance Criteria" but no content beneath it (word count = 0) fails
- A specification with no section containing "acceptance" in its heading fails
- The check examines the specification document, not the dev-plan document (the spec is the document that should carry acceptance criteria)

---

### Check Result Structure

**FR-007:** Each structural check execution MUST produce a structured result containing:
- `check_type` — one of: `required_sections`, `cross_reference`, `acceptance_criteria`
- `gate` — the transition gate where the check ran (e.g., `designing→specifying`)
- `document_id` — the document record ID that was checked
- `document_type` — the type of the document checked
- `passed` — boolean indicating whether the check passed
- `mode` — `warning` or `hard_gate`, indicating the current enforcement level
- `details` — when failed: a list of specific issues (e.g., missing section names, missing cross-reference target)

**Acceptance criteria:**
- A passing required-sections check returns `passed: true` with empty details
- A failing required-sections check returns `passed: false` with details listing each missing section by its human-readable label
- A failing cross-reference check returns `passed: false` with details indicating the expected reference target
- The check result includes the enforcement mode so callers can distinguish warnings from hard gate failures

---

### Warning and Hard Gate Modes

**FR-008:** All structural checks MUST start in `warning` mode. In warning mode, a failing check MUST:
1. Be included in the transition response as a `structural_warnings` field
2. NOT block the transition — the feature proceeds to the next state regardless
3. Be logged for observability (the specific check, document, and failure details)

**Acceptance criteria:**
- A feature transitions from `designing` to `specifying` successfully even when the design document fails structural checks, when all checks are in warning mode
- The transition response includes a `structural_warnings` array containing the check results for any failing checks
- The transition response does NOT include `structural_warnings` when all checks pass
- Warning-mode check results are available for downstream observability (action pattern logging)

---

**FR-009:** When a structural check has been promoted to `hard_gate` mode, a failing check MUST block the transition. The transition MUST fail with an actionable error message following the format defined in design §6.2, §10.4:

```
Cannot transition FEAT-{id} to "{target_state}": {document_type} document {doc_id}
failed structural check "{check_type}": {specific_failure_description}.

To resolve:
1. Read the current document: doc(action: "content", id: "{doc_id}")
2. {specific_remediation_instruction}
3. Re-register the document if modified: doc(action: "refresh", id: "{doc_id}")
```

**Acceptance criteria:**
- A feature transition blocked by a hard-gate structural check returns an error, not a success with warnings
- The error message names the feature ID, target state, document ID, and check type
- The error message includes specific remediation steps referencing MCP tool calls
- For a missing-sections failure, the remediation instruction lists the missing section names: "Add the missing sections: {section_labels}"
- For a cross-reference failure, the remediation instruction names the expected reference: "Add a reference to the design document: {design_doc_path}"
- For an acceptance-criteria failure, the remediation instruction says: "Add an 'Acceptance Criteria' section to the specification with at least one verifiable criterion"
- A hard-gate check that passes does not produce warnings or errors

---

**FR-010:** Hard-gate structural check failures MUST be overridable using the same override mechanism as other stage gate prerequisites (`override: true` with `override_reason`). The override policy for structural checks follows the gate's existing override policy (default: `agent` for 3.0).

**Acceptance criteria:**
- A transition blocked by a hard-gate structural check succeeds when `override: true` and `override_reason` are provided
- The override is logged and flagged by the `health` tool, consistent with other gate overrides
- Warning-mode checks do not require or accept overrides (they are non-blocking by definition)

---

### Promotion and Demotion

**FR-011:** The system MUST track the pass/fail history of each structural check type independently for the purpose of promotion decisions. The tracking unit is a tuple of `(check_type, document_type)` — for example, `(required_sections, specification)` and `(required_sections, design)` are tracked separately.

**Acceptance criteria:**
- The system maintains a separate pass counter for `(required_sections, design)`, `(required_sections, specification)`, `(required_sections, dev-plan)`, `(cross_reference, specification)`, and `(acceptance_criteria, specification)`
- A pass on `(required_sections, specification)` does not affect the counter for `(required_sections, design)`
- The counter tracks consecutive features where the check produced no false positives, not total passes

---

**FR-012:** A structural check MUST be promoted from `warning` mode to `hard_gate` mode when 5 consecutive features pass through the relevant gate with the check producing zero false positives. A "pass with zero false positives" means either: (a) the check passed (the document satisfied the check), or (b) the check failed and the failure was a true positive (the document genuinely lacked the required structure).

**Acceptance criteria:**
- After 5 consecutive features pass the `(required_sections, specification)` check at the `specifying → dev-planning` gate without a false positive, the check is promoted to hard gate mode
- After 4 consecutive clean passes, the check remains in warning mode
- The counter resets to 0 when a false positive is reported (FR-014)
- The counter increments only when a feature actually transitions through the relevant gate (features that skip the gate via override or are cancelled do not affect the counter)

---

**FR-013:** A structural check in `hard_gate` mode MUST be demoted back to `warning` mode when a false positive is detected. A false positive occurs when the structural check rejects a document that is structurally valid — i.e., the parser incorrectly determined that a required section was missing or a cross-reference was absent. Upon demotion, the consecutive pass counter MUST reset to 0.

**Acceptance criteria:**
- Reporting a false positive on a hard-gate check immediately demotes it to warning mode
- The consecutive pass counter resets to 0 upon demotion
- A demoted check behaves identically to any other warning-mode check
- The demotion event is logged for observability

---

**FR-014:** The system MUST provide a mechanism for reporting false positives on structural checks. This mechanism MUST accept the check type, document type, and a brief description of the false positive. False positive reporting is a manual action (invoked by an agent or human when they observe a structural check incorrectly rejecting a valid document).

**Acceptance criteria:**
- A false positive can be reported by specifying `check_type` and `document_type`
- Reporting a false positive on a warning-mode check resets the consecutive pass counter to 0 but does not change the mode (it is already `warning`)
- Reporting a false positive on a hard-gate check demotes it to warning and resets the counter
- The false positive report is persisted (not lost on server restart)

---

**FR-015:** Promotion state MUST be persisted in a file within the `.kbz/` directory so it survives server restarts. The promotion state file MUST contain, for each tracked `(check_type, document_type)` tuple:
- `mode` — current enforcement mode: `warning` or `hard_gate`
- `consecutive_clean` — current count of consecutive clean passes (0–5+)
- `promoted_at` — timestamp of last promotion (null if never promoted)
- `demoted_at` — timestamp of last demotion (null if never demoted)
- `false_positive_count` — cumulative count of reported false positives

**Acceptance criteria:**
- The promotion state file is written to `.kbz/structural-check-state.yaml`
- On server startup, the file is read and the current mode for each check is restored
- If the file does not exist (fresh project), all checks default to `warning` mode with `consecutive_clean: 0`
- Modifying the file manually (e.g., setting a check to `hard_gate`) takes effect on the next transition that consults that check
- The file format is YAML consistent with other `.kbz/` state files

---

### Quality Evaluation Structural Hook

**FR-016:** The document record model (`model.DocumentRecord`) MUST be extended with an optional `quality_evaluation` field. The quality evaluation record schema MUST contain:

| Field | Type | Required | Description |
|---|---|---|---|
| `overall_score` | float64 (0.0–1.0) | Yes | Aggregate quality score |
| `pass` | boolean | Yes | Whether the evaluation meets the pass threshold |
| `evaluated_at` | timestamp | Yes | When the evaluation was performed |
| `evaluator` | string | Yes | Identifier of the evaluating model (e.g., `claude-sonnet-4-20250514`) |
| `dimensions` | map[string]float64 | Yes | Per-dimension scores; keys are dimension names, values are 0.0–1.0 |

**Acceptance criteria:**
- A document record can be stored and loaded with a `quality_evaluation` field populated
- A document record with no quality evaluation has `quality_evaluation` as null/omitted in YAML
- The `overall_score` field is validated to be in the range 0.0–1.0 inclusive
- Each dimension score is validated to be in the range 0.0–1.0 inclusive
- The `evaluator` field is a non-empty string
- The `evaluated_at` field is a valid timestamp
- The `dimensions` map is non-empty (at least one dimension)
- The schema supports arbitrary dimension names (not restricted to the four dimensions from §10.5) to allow rubric evolution via skill updates

---

**FR-017:** The system MUST provide a mechanism to attach a quality evaluation to a document record. This mechanism accepts a document ID and a quality evaluation record, and persists the evaluation on the document record. The quality evaluation can be attached to a document in any status (`draft` or `approved`).

**Acceptance criteria:**
- Attaching a quality evaluation to a draft document succeeds and persists the evaluation
- Attaching a quality evaluation to an approved document succeeds and persists the evaluation
- Attaching a quality evaluation to a document that already has one replaces the previous evaluation
- The document's `updated` timestamp is refreshed when a quality evaluation is attached
- Attaching a quality evaluation to a non-existent document ID returns a descriptive error

---

**FR-018:** The document approval gate (`doc(action: "approve")`) MUST support an optional quality evaluation prerequisite. When enabled, the approval gate checks that:
1. A `quality_evaluation` record exists on the document
2. The `quality_evaluation.pass` field is `true`
3. The `quality_evaluation.overall_score` is greater than or equal to a configurable threshold (default: 0.7)

The quality evaluation prerequisite MUST be **disabled by default** for 3.0. It can be enabled via a configuration flag in `.kbz/config.yaml` (e.g., `require_quality_evaluation: true`).

**Acceptance criteria:**
- With `require_quality_evaluation: false` (the default), `doc(action: "approve")` succeeds on a document with no quality evaluation, consistent with current behavior
- With `require_quality_evaluation: true`, `doc(action: "approve")` on a document with no quality evaluation returns an actionable error: "Document {id} requires a quality evaluation before approval. Attach one using doc(action: 'evaluate', ...)"
- With `require_quality_evaluation: true`, `doc(action: "approve")` on a document whose quality evaluation has `pass: false` returns an error identifying the failing dimensions
- With `require_quality_evaluation: true`, `doc(action: "approve")` on a document whose quality evaluation has `pass: true` and `overall_score >= threshold` succeeds
- The threshold is configurable in `.kbz/config.yaml` under a `quality_evaluation_threshold` key (default: 0.7)
- The threshold configuration is read from the config file; changing it does not require a server restart (it is re-read on each approval check)

---

**FR-019:** The quality evaluation prerequisite on document approval MUST produce an actionable error message on failure, following the error template from design §6.2:

```
Cannot approve document {doc_id}: quality evaluation required but {reason}.

To resolve:
1. Run the quality evaluation skill on the document.
2. Attach the result: doc(action: "evaluate", id: "{doc_id}", evaluation: {...})
3. Retry approval: doc(action: "approve", id: "{doc_id}")
```

Where `{reason}` is one of:
- "no quality evaluation found" — evaluation not yet attached
- "quality evaluation failed (overall_score: {score}, threshold: {threshold})" — score below threshold
- "quality evaluation did not pass (pass: false)" — evaluation explicitly marked as failing

**Acceptance criteria:**
- The error message for a missing evaluation includes the `doc(action: "evaluate")` tool call syntax
- The error message for a failing score includes the actual score and the threshold
- The error message for `pass: false` includes the dimension scores so the agent knows what to improve

---

### MCP Tool Integration

**FR-020:** The `doc` MCP tool MUST expose a new `evaluate` action that attaches a quality evaluation to a document record. The action accepts:
- `id` (required) — document record ID
- `evaluation` (required) — quality evaluation object matching the schema in FR-016

**Acceptance criteria:**
- `doc(action: "evaluate", id: "DOC-xxx", evaluation: {...})` attaches the evaluation to the document
- Missing `id` returns a validation error
- Missing `evaluation` returns a validation error
- An `evaluation` object missing required fields (`overall_score`, `pass`, `evaluated_at`, `evaluator`, `dimensions`) returns a validation error naming the missing fields
- An `evaluation` with `overall_score` outside 0.0–1.0 returns a validation error

---

**FR-021:** The structural check results MUST be included in the response of `entity(action: "transition")` and `entity(action: "transition", advance: true)` when structural checks are executed. The response MUST include a `structural_checks` field containing the list of check results (FR-007 schema). Warning-mode failures appear in the response but do not prevent the transition from succeeding.

**Acceptance criteria:**
- A transition response includes `structural_checks` when checks were executed, regardless of pass/fail
- A transition response omits `structural_checks` when no checks applied (e.g., the gate has no structural checks defined)
- In `advance` mode, structural checks are reported for each intermediate gate that was crossed
- Warning-mode failures are clearly distinguishable from hard-gate failures in the response

---

## Non-Functional Requirements

**NFR-001:** Structural checks MUST complete within 100ms per document for documents up to 100KB. The checks use the existing `ParseStructure` function which is already optimized for this size range; the additional overhead is section-heading comparison against a small required-sections list.

**NFR-002:** The structural check mechanism MUST not modify the document content or document record during check execution. Checks are read-only operations against the parsed document structure and the document record metadata.

**NFR-003:** The quality evaluation record MUST be backward-compatible with existing document records. Documents created before this feature MUST load successfully with `quality_evaluation` as null/absent. No migration of existing document records is required.

**NFR-004:** The promotion state file (`.kbz/structural-check-state.yaml`) MUST use atomic writes consistent with other `.kbz/` state files (via `internal/fsutil/` atomic write utilities) to prevent corruption on concurrent access.

**NFR-005:** The structural check system MUST be testable without the binding registry. Hardcoded defaults provide complete functionality; the binding registry integration (a separate feature) adds configurability but is not a prerequisite.

**NFR-006:** The quality evaluation schema MUST not constrain dimension names to a fixed set. The four dimensions from the design (completeness, consistency, testability, factual_accuracy) are the initial set used by agent-side skills, but the server schema accepts any string keys. This allows rubric evolution via skill updates without server changes, consistent with the architectural decision in §16.1 Q3.

---

## Acceptance Criteria

| Requirement | Verification method |
|---|---|
| FR-001 (check execution at gates) | Integration test: transition a feature through all document-producing gates and verify structural checks fire at each |
| FR-002 (parser integration) | Unit test: parse a document with known sections, run the checker, verify matches |
| FR-003 (required section definitions) | Unit test: verify the hardcoded defaults for each document type match the specification |
| FR-004 (section requirement lookup) | Unit test: verify lookup returns correct requirements per document type and empty list for unchecked types |
| FR-005 (cross-reference validation) | Unit test: create documents with and without cross-references, verify detection using docint extractor output |
| FR-006 (acceptance criteria detection) | Unit test: create specifications with and without acceptance criteria sections, verify detection |
| FR-007 (check result structure) | Unit test: verify result struct contains all required fields |
| FR-008 (warning mode) | Integration test: configure all checks as warnings, trigger a failing check, verify transition succeeds with warnings in response |
| FR-009 (hard gate mode error) | Integration test: promote a check to hard gate, trigger a failure, verify transition is blocked with actionable error |
| FR-010 (override for hard gates) | Integration test: trigger a hard-gate failure, retry with override, verify transition succeeds |
| FR-011 (per-check tracking) | Unit test: increment counters for different check types independently, verify isolation |
| FR-012 (promotion at threshold) | Unit test: simulate 5 consecutive clean passes, verify mode changes to hard_gate |
| FR-013 (demotion on false positive) | Unit test: promote a check, report false positive, verify demotion to warning |
| FR-014 (false positive reporting) | Unit test: report false positive, verify counter reset and mode change |
| FR-015 (promotion state persistence) | Unit test: write state, restart (reload from file), verify state is preserved |
| FR-016 (quality evaluation schema) | Unit test: marshal/unmarshal document records with and without quality_evaluation |
| FR-017 (attach evaluation) | Unit test: attach evaluation to document, reload, verify evaluation is persisted |
| FR-018 (approval gate prerequisite) | Integration test: enable quality evaluation requirement, attempt approval without evaluation, verify error; attach passing evaluation, verify approval succeeds |
| FR-019 (approval error messages) | Unit test: verify error messages contain document ID, score, threshold, and remediation steps |
| FR-020 (doc evaluate action) | Integration test: call `doc(action: "evaluate")` via MCP tool, verify evaluation is attached |
| FR-021 (transition response inclusion) | Integration test: transition with structural checks, verify `structural_checks` field in response |

---

## Dependencies and Assumptions

**Dependencies:**
1. The document intelligence parser (`internal/docint/ParseStructure`, `ExtractPatterns`) exists and correctly parses Markdown section structure, cross-document links, entity references, and conventional roles
2. The document service (`internal/service/DocumentService`) provides document lookup by owner and type, and document content access
3. The feature lifecycle transition mechanism (`internal/service/advance.go`, `prereq.go`) provides the integration point where structural checks are invoked
4. The atomic file write utilities (`internal/fsutil/`) are available for promotion state persistence
5. The project configuration loader (`internal/config/`) supports reading new configuration keys from `.kbz/config.yaml`

**Assumptions:**
1. The mandatory-stage-gates feature (FEAT-01KN5-8J24S2XW) will make stage gates mandatory for all transitions (not just `advance`). This specification assumes structural checks fire at all transitions through document-producing gates, regardless of whether that feature is implemented first. If stage gates remain advance-only, structural checks fire only during advance sequences.
2. The binding registry feature (FEAT-01KN5-8J27H83N) is not a prerequisite. Hardcoded section requirements provide complete functionality. When the binding registry is available, a future enhancement can read `document_template.required_sections` from bindings instead of hardcoded defaults.
3. Document content is available on disk at the path recorded in the document record at the time of transition. If the file has been moved or deleted, the structural check reports an error rather than silently passing.
4. The quality evaluation skill (an agent-side concern) will use the `doc(action: "evaluate")` tool to attach evaluations. The server does not validate the semantic correctness of the evaluation — it trusts the agent to produce honest evaluations, consistent with the architectural decision in §16.1 Q3.
5. The threshold of 5 consecutive clean features for promotion (§16.2 Q2) is appropriate for structural checks (section headings, cross-references). The implementing agent should revisit this if early usage reveals the threshold is too low (false promotions) or too high (checks never promote).