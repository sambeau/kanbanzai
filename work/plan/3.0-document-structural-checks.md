# Implementation Plan: Document Structural Checks and Quality Hooks

**Feature:** FEAT-01KN5-8J26RSB6 (document-structural-checks)
**Specification:** `work/spec/3.0-document-structural-checks.md`
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §10.4, §10.5, §16.1 Q3, §16.2 Q2

---

## Scope Boundaries

Carried forward from the specification:

- **In scope:** Structural check definitions (required sections, cross-references, acceptance criteria detection), integration with feature lifecycle transitions at document-producing stage gates, warning mode (non-blocking) and hard gate mode (blocking) enforcement, per-check promotion/demotion with state persistence, `quality_evaluation` record schema on document records, `doc(action: "evaluate")` MCP action, optional quality evaluation approval gate, actionable error messages for all failures.
- **Out of scope:** LLM-as-judge evaluation logic (agent-side skill), document template content, the document intelligence parser itself (already exists in `internal/docint/`), gate enforcement mechanism for lifecycle transitions (mandatory-stage-gates feature FEAT-01KN5-8J24S2XW), binding registry integration for reading required sections (FEAT-01KN5-8J27H83N), review report parsing, quality evaluation rubric content or prompt design.

---

## Dependency on Mandatory Stage Gates

This feature adds checks that run AT gates. The gate enforcement mechanism itself is provided by the mandatory-stage-gates feature (FEAT-01KN5-8J24S2XW). Task 4 of this plan integrates structural checks into the transition path established by that feature. If mandatory-stage-gates is not yet merged, Task 4 must integrate with the existing `AdvanceFeatureStatus` / `CheckFeatureGate` code paths in `internal/service/advance.go` and `prereq.go`, and may need adjustment when mandatory-stage-gates lands.

---

## Task Breakdown

### Task 1: Required Section Definitions Data Module

**Objective:** Create a self-contained data module that defines required sections per document type as a lookup table. Each entry specifies a human-readable label and one or more match keywords. The lookup returns an ordered list for known types (`design`, `specification`, `dev-plan`) and an empty list for all other types.

**Specification references:** FR-003, FR-004

**Input context:**
- `work/spec/3.0-document-structural-checks.md` §Required Section Definitions — the exact section requirements per document type
- `internal/model/entities.go` — `DocumentType` constants (`DocumentTypeDesign`, `DocumentTypeSpecification`, `DocumentTypeDevPlan`)
- `refs/go-style.md` — naming conventions, package design

**Output artifacts:**
- New file `internal/structural/sections.go`:
  - `SectionRequirement` struct with `Label string` and `Keywords []string`
  - `RequiredSections(docType string) []SectionRequirement` — returns the ordered list for the given document type, or nil for unrecognised types
  - Hardcoded defaults for `design` (overview/purpose/summary + design), `specification` (overview + scope + functional requirements + acceptance criteria), `dev-plan` (overview + task)
- New file `internal/structural/sections_test.go`:
  - Table-driven tests verifying correct requirements returned for `design`, `specification`, `dev-plan`
  - Test that unknown types (`research`, `report`, `policy`, `""`) return an empty/nil list
  - Test that each requirement has a non-empty label and at least one keyword

**Dependencies:** None — this is a pure data module with no external dependencies.

**Interface contract:** The `SectionRequirement` type is the shared contract used by Task 2 (check engine):

```
// SectionRequirement defines a required section for structural checking.
type SectionRequirement struct {
    Label    string   // Human-readable name for error messages (e.g. "overview")
    Keywords []string // Any keyword match satisfies (case-insensitive substring)
}

// RequiredSections returns the ordered list of required sections for a document type.
// Returns nil for document types with no structural check definitions.
func RequiredSections(docType string) []SectionRequirement
```

---

### Task 2: Structural Check Engine

**Objective:** Implement the three structural check types — required sections, cross-reference validation, and acceptance criteria detection — as pure functions that accept parsed document data and return structured check results. The engine uses the existing `docint.ParseStructure` and `docint.ExtractPatterns` output; it does not parse documents itself. Each check produces a `CheckResult` (FR-007 schema).

**Specification references:** FR-002, FR-005, FR-006, FR-007

**Input context:**
- `internal/docint/parser.go` — `ParseStructure` returns `[]Section` with `Level`, `Title`, `WordCount`, `Children`
- `internal/docint/extractor.go` — `ExtractPatterns` returns `ExtractResult` with `CrossDocLinks`, `EntityRefs`, `ConventionalRoles`
- `internal/docint/types.go` — `Section`, `CrossDocLink`, `EntityRef`, `ConventionalRole` structs
- `internal/docint/taxonomy.go` — `conventionalRoleKeywords` includes `"acceptance criteria"` mapped to `RoleRequirement`
- `internal/structural/sections.go` (Task 1) — `RequiredSections`, `SectionRequirement`
- `internal/model/entities.go` — `DocumentRecord` struct (Path, Type, Owner, Status fields)

**Output artifacts:**
- New file `internal/structural/checks.go`:
  - `CheckResult` struct with fields: `CheckType string`, `Gate string`, `DocumentID string`, `DocumentType string`, `Passed bool`, `Mode string`, `Details []string`
  - `CheckRequiredSections(sections []docint.Section, docType string, docID string, gate string) CheckResult` — flattens sections to level 1–3, matches headings against `RequiredSections(docType)` using case-insensitive substring containment, excludes the H1 title heading from matching
  - `CheckCrossReference(extractResult docint.ExtractResult, approvedDesignPaths []string, approvedDesignIDs []string, docID string, gate string) CheckResult` — checks `CrossDocLinks` target paths and `EntityRefs` for `DOC-` IDs against the provided approved design documents
  - `CheckAcceptanceCriteria(sections []docint.Section, conventionalRoles []docint.ConventionalRole, docID string, gate string) CheckResult` — checks for a section heading containing "acceptance" with non-empty content (word count > 0), or a `ConventionalRole` entry with role `requirement` associated with a heading containing "acceptance criteria"
- New file `internal/structural/checks_test.go`:
  - **Required sections tests:**
    - Design doc with "Overview" and "Design Principles" sections → passes
    - Design doc with only "Purpose" and no design section → fails, details names missing section
    - Spec doc with all four required sections → passes
    - Spec doc missing "Scope" → fails, details includes "scope"
    - Dev-plan with "Overview" and "Task Breakdown" → passes
    - Dev-plan with "Overview" only → fails
    - Document titled "Design: Foo" — the H1 title must not satisfy the "design" requirement
    - Empty/non-Markdown content → all required sections reported as missing
    - Heading "Functional Requirements and Constraints" satisfies "functional requirements" keyword
  - **Cross-reference tests:**
    - Spec with `[design](work/design/my-design.md)` where that path is in approved list → passes
    - Spec with backtick-quoted `` `work/design/my-design.md` `` where path matches → passes
    - Spec with `DOC-xxx` entity ref where ID is in approved list → passes
    - Spec with no links → fails
    - Spec referencing a design from parent plan → passes (parent plan paths in approved list)
  - **Acceptance criteria tests:**
    - Spec with "Acceptance Criteria" heading containing bullet points → passes
    - Spec with "Acceptance Criteria" heading but empty content (word count 0) → fails
    - Spec with no "acceptance" heading → fails

**Dependencies:** Task 1 (required section definitions).

**Interface contract with Task 4:** The `CheckResult` struct is the shared schema used by the gate integration layer. The `Mode` field is populated by Task 4 (the engine does not know about promotion state); the engine sets `Mode` to `""` (callers fill it in):

```
// CheckResult is the structured outcome of a single structural check.
type CheckResult struct {
    CheckType    string   `json:"check_type"`    // "required_sections", "cross_reference", "acceptance_criteria"
    Gate         string   `json:"gate"`          // e.g. "designing→specifying"
    DocumentID   string   `json:"document_id"`
    DocumentType string   `json:"document_type"`
    Passed       bool     `json:"passed"`
    Mode         string   `json:"mode"`          // "warning" or "hard_gate" (set by caller)
    Details      []string `json:"details"`       // specific failure descriptions
}
```

---

### Task 3: Promotion State Persistence

**Objective:** Implement the promotion/demotion state machine and its persistence layer. Each `(check_type, document_type)` tuple is tracked independently. Checks start in `warning` mode and promote to `hard_gate` after 5 consecutive clean passes. False positive reporting demotes a hard-gate check back to warning and resets the counter. State persists in `.kbz/structural-check-state.yaml`.

**Specification references:** FR-011, FR-012, FR-013, FR-014, FR-015

**Input context:**
- `internal/fsutil/atomic.go` — `WriteFileAtomic` for safe state persistence
- `internal/config/config.go` — pattern for reading `.kbz/` state files
- `refs/go-style.md` — YAML serialisation rules (block style, deterministic field order, no flow style)

**Output artifacts:**
- New file `internal/structural/promotion.go`:
  - `CheckKey` struct with `CheckType string` and `DocumentType string`
  - `PromotionEntry` struct with fields: `Mode string`, `ConsecutiveClean int`, `PromotedAt *time.Time`, `DemotedAt *time.Time`, `FalsePositiveCount int`
  - `PromotionState` struct holding a `map[CheckKey]PromotionEntry` and the file path
  - `LoadPromotionState(stateRoot string) (*PromotionState, error)` — reads `.kbz/structural-check-state.yaml`, returns defaults (all warning, zero counters) if file does not exist
  - `(*PromotionState) Save() error` — atomic write to the state file
  - `(*PromotionState) GetMode(key CheckKey) string` — returns `"warning"` or `"hard_gate"` for the given check
  - `(*PromotionState) RecordPass(key CheckKey)` — increments consecutive clean counter; promotes to `hard_gate` if counter reaches 5
  - `(*PromotionState) RecordFalsePositive(key CheckKey, description string)` — demotes to `warning` if currently `hard_gate`, resets counter to 0, increments `FalsePositiveCount`
- New file `internal/structural/promotion_test.go`:
  - Fresh state: all checks default to `warning` with `consecutive_clean: 0`
  - After 5 consecutive `RecordPass` calls on `(required_sections, specification)` → mode becomes `hard_gate`, `promoted_at` is set
  - After 4 passes → still `warning`
  - Pass on `(required_sections, specification)` does not affect `(required_sections, design)` — counters are independent
  - Reporting false positive on `hard_gate` check → demotes to `warning`, counter resets, `demoted_at` set, `false_positive_count` incremented
  - Reporting false positive on `warning` check → counter resets, mode unchanged, `false_positive_count` incremented
  - Round-trip: save state, load from file, verify all entries preserved
  - Missing file on load → returns default state (no error)
  - Manual file edit (set a check to `hard_gate`) → takes effect on next load
  - Counter only increments on `RecordPass`, not on skipped/overridden transitions

**Dependencies:** None — this is an independent persistence module.

**Interface contract with Task 4:** Task 4 calls `GetMode` before running checks (to set the `Mode` field on `CheckResult`), and calls `RecordPass` or uses the check outcome to update state after each gate execution:

```
// CheckKey identifies a structural check for promotion tracking.
type CheckKey struct {
    CheckType    string // "required_sections", "cross_reference", "acceptance_criteria"
    DocumentType string // "design", "specification", "dev-plan"
}

func (ps *PromotionState) GetMode(key CheckKey) string
func (ps *PromotionState) RecordPass(key CheckKey)
func (ps *PromotionState) RecordFalsePositive(key CheckKey, description string)
```

**Interface contract with Task 4 (false positive MCP surface):** Task 4 exposes false positive reporting via the `entity` tool or a dedicated structural check tool action. The service-layer function signature is:

```
func (ps *PromotionState) RecordFalsePositive(key CheckKey, description string)
```

---

### Task 4: Gate Integration, Enforcement, and Transition Response

**Objective:** Wire the structural check engine and promotion state into the feature lifecycle transition path. At each document-producing gate, determine which checks apply, execute them, look up their enforcement mode, and either include warnings in the response (warning mode) or block the transition (hard gate mode). Enrich the transition response with `structural_checks` results per FR-021. Support overrides for hard-gate failures. Expose false positive reporting as an MCP action.

**Specification references:** FR-001, FR-008, FR-009, FR-010, FR-021, FR-014 (MCP surface)

**Input context:**
- `internal/structural/checks.go` (Task 2) — `CheckRequiredSections`, `CheckCrossReference`, `CheckAcceptanceCriteria`, `CheckResult`
- `internal/structural/promotion.go` (Task 3) — `PromotionState`, `GetMode`, `RecordPass`, `RecordFalsePositive`
- `internal/service/advance.go` — `AdvanceFeatureStatus`, the loop that walks features through gates
- `internal/service/prereq.go` — `CheckFeatureGate`, `GateResult`, `stageDocMapping`, `checkDocumentGate`
- `internal/mcp/entity_tool.go` — `entityTransitionAction`, `entityAdvanceFeature` — where transition responses are built
- `internal/service/documents.go` — `DocumentService` for loading document content and listing documents by owner/type
- `internal/docint/parser.go` — `ParseStructure`
- `internal/docint/extractor.go` — `ExtractPatterns`
- `internal/model/entities.go` — `Feature` struct, `DocumentRecord`

**Output artifacts:**
- New file `internal/structural/gate.go`:
  - `GateCheckConfig` struct defining which checks apply at each gate (the FR-001 table):
    - `designing→specifying`: required_sections on design doc
    - `specifying→dev-planning`: required_sections + cross_reference on spec doc
    - `dev-planning→developing`: required_sections on dev-plan + acceptance_criteria on spec
  - `RunGateChecks(gate string, feature *model.Feature, docSvc DocumentLookup, promotionState *PromotionState) ([]CheckResult, error)` — orchestrates: identifies which checks to run for the gate, loads document content, calls `ParseStructure` / `ExtractPatterns`, runs check functions, sets `Mode` from promotion state, calls `RecordPass` for passing checks
  - `FormatCheckError(result CheckResult) string` — produces the actionable error message per FR-009 template
  - `DocumentLookup` interface — minimal interface for loading document records and content (testable without full `DocumentService`)
- Modified `internal/service/advance.go`:
  - After each successful gate transition in the `AdvanceFeatureStatus` loop, call `structural.RunGateChecks` for the gate just crossed
  - Collect `[]CheckResult` across all gates traversed
  - If any hard-gate check fails, stop the advance and return an error with `FormatCheckError`
  - Accumulate warning-mode failures into the `AdvanceResult` (add a new `StructuralChecks []structural.CheckResult` field)
- Modified `internal/mcp/entity_tool.go`:
  - In `entityTransitionAction`: after a single-step feature transition, call `RunGateChecks` for the relevant gate (if any)
  - If hard-gate failure: return error response with actionable message
  - If warning-mode failures: include `structural_checks` in the response map
  - In `entityAdvanceFeature`: propagate `StructuralChecks` from `AdvanceResult` into the response map
  - Support override: when `override: true` and `override_reason` are provided, bypass hard-gate structural check failures (consistent with the override mechanism from mandatory-stage-gates)
- New file `internal/structural/gate_test.go`:
  - Integration-style test: mock document lookup, run gate checks for each gate type, verify correct checks executed
  - Warning mode: failing check included in results with `mode: "warning"`, no error returned
  - Hard gate mode: failing check returns error with actionable message
  - Override: hard-gate failure with override succeeds
  - Advance mode: checks reported for each intermediate gate crossed
  - No structural checks for gates outside the FR-001 table (e.g., `developing→reviewing`)
  - Missing approved document for a gate → structural check step skipped (doc prerequisite is a separate concern)
- New MCP surface for false positive reporting (in `doc_tool.go` or a new `structural_tool.go`):
  - Action accepting `check_type`, `document_type`, and `description`
  - Calls `PromotionState.RecordFalsePositive` and persists

**Dependencies:** Task 2 (check engine), Task 3 (promotion state).

**Interface contract — `AdvanceResult` extension:**

```
type AdvanceResult struct {
    FinalStatus      string
    AdvancedThrough  []string
    StoppedReason    string
    StructuralChecks []structural.CheckResult // NEW: collected across all gates
}
```

**Interface contract — transition response shape (FR-021):**

```
{
  "entity": { ... },
  "structural_checks": [
    {
      "check_type": "required_sections",
      "gate": "designing→specifying",
      "document_id": "FEAT-xxx/my-design",
      "document_type": "design",
      "passed": false,
      "mode": "warning",
      "details": ["missing required section: design"]
    }
  ]
}
```

**Actionable error message format (FR-009):**

```
Cannot transition FEAT-{id} to "{target_state}": {document_type} document {doc_id}
failed structural check "{check_type}": {specific_failure_description}.

To resolve:
1. Read the current document: doc(action: "content", id: "{doc_id}")
2. {specific_remediation_instruction}
3. Re-register the document if modified: doc(action: "refresh", id: "{doc_id}")
```

---

### Task 5: Quality Evaluation Schema and Attachment

**Objective:** Extend `model.DocumentRecord` with an optional `quality_evaluation` field and implement the service-layer method to attach an evaluation to a document record. The evaluation is validated (score ranges, required fields) and persisted on the document record. This is the data foundation for Tasks 6 and 7.

**Specification references:** FR-016, FR-017

**Input context:**
- `internal/model/entities.go` — `DocumentRecord` struct (L454–470), current fields
- `internal/service/documents.go` — `DocumentService`, `ApproveDocument`, document store load/save patterns
- `refs/go-style.md` — YAML serialisation rules

**Output artifacts:**
- Modified `internal/model/entities.go`:
  - New `QualityEvaluation` struct:
    - `OverallScore float64` (`yaml:"overall_score"`)
    - `Pass bool` (`yaml:"pass"`)
    - `EvaluatedAt time.Time` (`yaml:"evaluated_at"`)
    - `Evaluator string` (`yaml:"evaluator"`)
    - `Dimensions map[string]float64` (`yaml:"dimensions"`)
  - New field on `DocumentRecord`: `QualityEvaluation *QualityEvaluation` (`yaml:"quality_evaluation,omitempty"`)
- Modified `internal/service/documents.go`:
  - New `AttachQualityEvaluation(input AttachEvaluationInput) (DocumentResult, error)` method on `DocumentService`
  - `AttachEvaluationInput` struct: `ID string`, `Evaluation QualityEvaluation`
  - Validation: `overall_score` in [0.0, 1.0], each dimension score in [0.0, 1.0], `evaluator` non-empty, `dimensions` non-empty (at least one entry), `evaluated_at` is valid
  - Loads the document record, sets `QualityEvaluation`, updates `Updated` timestamp, saves
  - Works on both `draft` and `approved` documents
  - Replaces any existing evaluation
  - Returns error for non-existent document ID
- Modified `internal/model/entities_test.go`:
  - YAML round-trip test: document record with quality evaluation serialises and deserialises correctly
  - YAML round-trip test: document record without quality evaluation omits the field
  - Backward compatibility: existing document YAML (no `quality_evaluation` field) loads with `QualityEvaluation` as nil
- New tests in `internal/service/documents_test.go`:
  - Attach evaluation to draft document → succeeds, evaluation persisted
  - Attach evaluation to approved document → succeeds
  - Attach evaluation replacing an existing one → new evaluation replaces old
  - `Updated` timestamp refreshed after attachment
  - Non-existent document ID → descriptive error
  - `overall_score` > 1.0 → validation error
  - `overall_score` < 0.0 → validation error
  - Dimension score out of range → validation error
  - Empty `evaluator` → validation error
  - Empty `dimensions` map → validation error
  - Arbitrary dimension names accepted (not restricted to a fixed set)

**Dependencies:** None — this is a model and service layer change with no structural check dependency.

**Interface contract:** The `QualityEvaluation` struct is the shared type used by Tasks 6 and 7:

```
type QualityEvaluation struct {
    OverallScore float64            `yaml:"overall_score"`
    Pass         bool               `yaml:"pass"`
    EvaluatedAt  time.Time          `yaml:"evaluated_at"`
    Evaluator    string             `yaml:"evaluator"`
    Dimensions   map[string]float64 `yaml:"dimensions"`
}
```

The service method signature used by Task 6:

```
func (s *DocumentService) AttachQualityEvaluation(input AttachEvaluationInput) (DocumentResult, error)
```

---

### Task 6: Doc Evaluate MCP Action

**Objective:** Add a new `evaluate` action to the `doc` MCP tool that accepts a document ID and a quality evaluation object, validates the input, and delegates to the `DocumentService.AttachQualityEvaluation` method. This provides the agent-facing surface for attaching quality evaluations.

**Specification references:** FR-020

**Input context:**
- `internal/mcp/doc_tool.go` — existing action dispatch pattern (`docTool` function, action routing at L50–111), `docArgStr` helper, `docRecordToMap`
- `internal/service/documents.go` (Task 5) — `AttachQualityEvaluation`, `AttachEvaluationInput`
- `internal/model/entities.go` (Task 5) — `QualityEvaluation` struct

**Output artifacts:**
- Modified `internal/mcp/doc_tool.go`:
  - New case in the action dispatch: `"evaluate"` → `docEvaluateAction(docSvc)`
  - New function `docEvaluateAction(docSvc *service.DocumentService) ActionHandler`:
    - Extracts `id` (required) and `evaluation` (required) from args
    - Parses `evaluation` map into `model.QualityEvaluation` struct (extracting `overall_score`, `pass`, `evaluated_at`, `evaluator`, `dimensions`)
    - Returns validation error naming missing fields if any required field is absent
    - Calls `docSvc.AttachQualityEvaluation`
    - Returns `{"document": docRecordToMap(result)}` on success
  - Update `docRecordToMap` to include `quality_evaluation` in the output map when present
- New test in `internal/mcp/doc_tool_test.go` (or integration test):
  - Successful evaluate call → evaluation attached, response includes document with evaluation
  - Missing `id` → validation error
  - Missing `evaluation` → validation error
  - Evaluation missing required fields → validation error naming the fields
  - `overall_score` out of range → validation error

**Dependencies:** Task 5 (quality evaluation schema and service method).

---

### Task 7: Quality Evaluation Approval Gate

**Objective:** Add an optional quality evaluation prerequisite to `doc(action: "approve")`. When enabled via `.kbz/config.yaml`, approval checks that a quality evaluation exists, `pass` is `true`, and `overall_score` meets the configurable threshold. Disabled by default. Produces actionable error messages on failure per FR-019.

**Specification references:** FR-018, FR-019

**Input context:**
- `internal/service/documents.go` — `ApproveDocument` method (L296+), current approval flow
- `internal/config/config.go` — `Config` struct (L146–175), `Load`/`LoadFrom` patterns
- `internal/model/entities.go` (Task 5) — `DocumentRecord.QualityEvaluation`, `QualityEvaluation` struct

**Output artifacts:**
- Modified `internal/config/config.go`:
  - New `QualityEvaluationConfig` struct:
    - `RequireForApproval bool` (`yaml:"require_quality_evaluation"`) — default `false`
    - `Threshold float64` (`yaml:"quality_evaluation_threshold"`) — default `0.7`
  - New field on `Config`: `QualityEvaluation QualityEvaluationConfig` (`yaml:"quality_evaluation,omitempty"`)
- Modified `internal/service/documents.go`:
  - In `ApproveDocument`, after existing validation (status == draft, content hash match), add quality evaluation gate check:
    - Read config (re-read on each call per FR-018 AC — threshold changes take effect without restart)
    - If `require_quality_evaluation` is `false` → skip (current behaviour preserved)
    - If `true`: check `record.QualityEvaluation != nil`, `record.QualityEvaluation.Pass == true`, `record.QualityEvaluation.OverallScore >= threshold`
    - On failure: return actionable error per FR-019 template
  - Three error message variants:
    - No evaluation: `"Cannot approve document {id}: quality evaluation required but no quality evaluation found."`
    - Score below threshold: `"Cannot approve document {id}: quality evaluation required but quality evaluation failed (overall_score: {score}, threshold: {threshold})."`
    - Pass is false: `"Cannot approve document {id}: quality evaluation required but quality evaluation did not pass (pass: false)."`
    - Each includes remediation steps: run skill, attach via `doc(action: "evaluate")`, retry approval
    - Pass-false error includes dimension scores so the agent knows what to improve
- Modified `internal/config/config_test.go`:
  - Test that config without `quality_evaluation` section loads with defaults (`false`, `0.7`)
  - Test that config with `require_quality_evaluation: true` and `quality_evaluation_threshold: 0.8` loads correctly
- New tests in `internal/service/documents_test.go`:
  - `require_quality_evaluation: false` (default): approve succeeds without evaluation (backward compatible)
  - `require_quality_evaluation: true`, no evaluation → error with "no quality evaluation found" and tool call hint
  - `require_quality_evaluation: true`, evaluation with `pass: false` → error with dimension scores
  - `require_quality_evaluation: true`, evaluation with `pass: true` but `overall_score` < threshold → error with score and threshold
  - `require_quality_evaluation: true`, evaluation with `pass: true` and `overall_score` >= threshold → approval succeeds
  - Threshold change in config takes effect on next approval (no restart needed)

**Dependencies:** Task 5 (quality evaluation schema on DocumentRecord).

---

## Dependency Graph

```
Layer 0 (no dependencies — can run in parallel):
  ┌──────────┐    ┌──────────┐    ┌──────────┐
  │  Task 1  │    │  Task 3  │    │  Task 5  │
  │ Required │    │Promotion │    │ Quality  │
  │ Sections │    │  State   │    │  Eval    │
  │   Data   │    │Persistence│   │  Schema  │
  └────┬─────┘    └────┬─────┘    └──┬───┬───┘
       │               │             │   │
Layer 1:               │             │   │
  ┌────▼─────┐         │        ┌────▼┐ ┌▼────────┐
  │  Task 2  │         │        │Task 6│ │ Task 7  │
  │ Check    │         │        │ Doc  │ │ Approval│
  │ Engine   │         │        │ Eval │ │  Gate   │
  └────┬─────┘         │        │Action│ │         │
       │               │        └──────┘ └─────────┘
Layer 2:               │
  ┌────▼───────────────▼┐
  │       Task 4        │
  │  Gate Integration   │
  │  & Enforcement      │
  └─────────────────────┘
```

**Maximum parallelism:**
- Layer 0: Tasks 1, 3, 5 execute simultaneously (3-wide)
- Layer 1: Tasks 2, 6, 7 execute simultaneously (3-wide), Task 3 may still be in progress
- Layer 2: Task 4 executes after Tasks 2 and 3 complete

---

## Traceability Matrix

| Requirement | Task(s) | Verification |
|---|---|---|
| FR-001 (check execution at gates) | Task 4 | Integration test: transition through all document-producing gates, verify checks fire |
| FR-002 (parser integration) | Task 2 | Unit test: parse document, run checker, verify section matches |
| FR-003 (required section definitions) | Task 1 | Unit test: verify hardcoded defaults match specification |
| FR-004 (section requirement lookup) | Task 1 | Unit test: lookup returns correct requirements per type, empty for unchecked types |
| FR-005 (cross-reference validation) | Task 2 | Unit test: documents with/without cross-references, verify detection |
| FR-006 (acceptance criteria detection) | Task 2 | Unit test: specifications with/without acceptance criteria sections |
| FR-007 (check result structure) | Task 2 | Unit test: verify result struct contains all required fields |
| FR-008 (warning mode) | Task 4 | Integration test: warning-mode failure → transition succeeds, warnings in response |
| FR-009 (hard gate mode error) | Task 4 | Integration test: hard-gate failure → transition blocked with actionable error |
| FR-010 (override for hard gates) | Task 4 | Integration test: hard-gate failure with override → transition succeeds |
| FR-011 (per-check tracking) | Task 3 | Unit test: independent counters for different (check_type, document_type) tuples |
| FR-012 (promotion at threshold) | Task 3 | Unit test: 5 consecutive clean passes → mode changes to hard_gate |
| FR-013 (demotion on false positive) | Task 3 | Unit test: promote, report false positive, verify demotion |
| FR-014 (false positive reporting) | Task 3, Task 4 | Unit test: report false positive, verify counter reset; MCP surface test |
| FR-015 (promotion state persistence) | Task 3 | Unit test: write state, reload, verify preserved |
| FR-016 (quality evaluation schema) | Task 5 | Unit test: marshal/unmarshal with and without quality_evaluation |
| FR-017 (attach evaluation) | Task 5 | Unit test: attach to draft/approved, replace existing, verify persistence |
| FR-018 (approval gate prerequisite) | Task 7 | Integration test: enable requirement, test all approval scenarios |
| FR-019 (approval error messages) | Task 7 | Unit test: verify error messages contain doc ID, score, threshold, remediation steps |
| FR-020 (doc evaluate action) | Task 6 | Integration test: MCP tool call attaches evaluation |
| FR-021 (transition response inclusion) | Task 4 | Integration test: structural_checks field in transition response |
| NFR-001 (performance) | Task 2 | Benchmark test: structural checks on 100KB document complete within 100ms |
| NFR-002 (read-only checks) | Task 2 | Code review: check functions take parsed data, do not write |
| NFR-003 (backward compatibility) | Task 5 | Unit test: existing document YAML loads with nil quality_evaluation |
| NFR-004 (atomic writes) | Task 3 | Code review: uses `fsutil.WriteFileAtomic` |
| NFR-005 (no binding registry dependency) | Task 1 | Design: hardcoded defaults, no external dependency |
| NFR-006 (flexible dimension names) | Task 5 | Unit test: arbitrary dimension keys accepted |