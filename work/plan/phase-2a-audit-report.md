# Phase 2a Audit Report and Remediation Plan

- Status: active
- Date: 2025-07-23
- Purpose: document findings from the Phase 2a implementation audit and define remediation tracks
- Related:
  - `work/spec/phase-2-specification.md`
  - `work/plan/phase-2a-progress.md`
  - `work/plan/phase-2-scope.md`
  - `work/plan/phase-1-audit-2-remediation.md`

---

## 1. Purpose

This document records the findings of a comprehensive audit of the Phase 2a implementation against the Phase 2 specification (§6–§22) and defines remediation tracks for each issue found.

The audit was conducted after the progress document (`phase-2a-progress.md`) claimed all 11 acceptance criteria (§22.1–§22.11) were met. The audit found that the implementation is substantial and generally well-structured, but contains bugs, spec deviations, test gaps, and stale documentation that must be addressed before Phase 2a can be considered complete.

---

## 2. Audit Scope and Method

The audit assessed:

1. **Spec compliance** — every requirement in §6–§22 checked against implementation
2. **Code correctness** — logic errors, dead code, semantic bugs
3. **Code quality** — idiomatic Go, naming, documentation, error handling
4. **Test coverage** — missing tests, flaky tests, test quality
5. **Documentation accuracy** — progress doc, README, AGENTS.md, bootstrap-workflow

Build and test health at time of audit:

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test -race ./...` — 12 packages tested, 11 pass, **1 failure** (`internal/service`)
- No race conditions detected

The single failure is `TestEntityService_ResolvePrefix` — a flaky test with hardcoded ULIDs that share a common prefix, causing ambiguous resolution.

---

## 3. Findings Summary

### 3.1 Bugs

| ID | Severity | Summary | Location |
|----|----------|---------|----------|
| B1 | High | Plan `done` incorrectly marked as terminal state — blocks `done → superseded/cancelled` | `internal/validate/lifecycle.go` |
| B2 | High | Feature `done` incorrectly marked as terminal state — makes `done → superseded` transition dead code | `internal/validate/lifecycle.go` |
| B3 | Medium | Feature entry state stuck at Phase 1 `draft` — `phase2FeatureEntryState` defined but unused | `internal/validate/lifecycle.go` |
| B4 | Medium | Optimistic locking bypassed in DocumentService — `DocumentToRecord()` drops `FileHash` | `internal/storage/document_store.go` |
| B5 | Medium | `ClassifiedAt` timestamp not set in `doc_classify` MCP handler — defaults to year 0001 | `internal/mcp/doc_intelligence_tools.go` |
| B6 | Low | `MatchConventionalRole` iterates over Go map — non-deterministic when heading matches multiple keywords | `internal/docint/taxonomy.go` |
| B7 | Low | `NormalizeConcept` produces double hyphens — breaks deduplication for some inputs | `internal/docint/concepts.go` |
| B8 | Low | `TestEntityService_ResolvePrefix` flaky — hardcoded ULIDs share common prefix | `internal/service/entities_test.go` |

### 3.2 Spec deviations

| ID | Spec § | Summary | Location |
|----|--------|---------|----------|
| S1 | §10.2 | `PrefixEntry` uses `Label` (yaml: `"label"`) instead of spec's `Name` (yaml: `"name"`) | `internal/config/config.go` |
| S2 | §10.2 | `PrefixEntry` missing optional `Description` field | `internal/config/config.go` |
| S3 | §8.2 | Feature service still uses `Epic`/`Plan` fields, not `Parent`/`DevPlan` — `featureFields()` writes `"epic"` and `"plan"` to YAML; `CreateFeature` requires `epic` input | `internal/service/entities.go`, `internal/mcp/entity_tools.go` |
| S4 | §8.2 | Feature `CreateFeature` doesn't accept `parent`, `design`, or `tags` parameters — Phase 2 Feature creation path incomplete | `internal/service/entities.go` |
| S5 | §18.1 | MCP document record tool names use `doc_record_` prefix instead of spec names (`submit_document`, `approve_document`, etc.) | `internal/mcp/doc_record_tools.go` |
| S6 | §18.1 | `create_plan` takes `prefix` + `slug` separately; spec says caller provides full `id` and system derives slug | `internal/mcp/plan_tools.go` |
| S7 | §18.1 | Generic entity tools (`get_entity`, `list_entities`, `update_status`, `update_entity`) don't support `plan` entity type in their enum | `internal/mcp/entity_tools.go` |
| S8 | §18.1 | Phase 1 tools not removed per spec directive: `update_document_body`, `extract_from_document`, `retrieve_document` still registered | `internal/mcp/document_tools.go` |
| S9 | §14.1 | `ListEntitiesFiltered` exists in service layer but no MCP tool exposes it | `internal/mcp/` |
| S10 | §12.2 | Layer 1 does not identify content blocks within sections (paragraphs, lists, tables, code blocks) | `internal/docint/parser.go` |
| S11 | §12.7 | 3 graph edge types missing: `DEPENDS_ON`, `SUPERSEDES`, `REFINES` | `internal/docint/graph.go` |
| S12 | §12.3 | `DOC-xxx` entity reference pattern missing from extractor | `internal/docint/extractor.go` |
| S13 | §15.4 | Index storage uses `yaml.Marshal` instead of canonical serializer — violates P1-DEC-008 | `internal/docint/store.go` |
| S14 | §15.4 | Index files don't use atomic writes (entity storage uses `fsutil` for write-to-temp-then-rename) | `internal/docint/store.go` |

### 3.3 Code quality issues

| ID | Summary | Location |
|----|---------|----------|
| Q1 | `parsePlanIDParts` in config duplicates `model.ParsePlanID` to avoid import cycles | `internal/config/config.go` |
| Q2 | `ConfigDir` constant defined but never used | `internal/config/config.go` |
| Q3 | `Config.Validate()` doesn't check that Label/Name is non-empty (spec says required) | `internal/config/config.go` |
| Q4 | `IsPlanID` checks ASCII digits only; `ValidatePrefix` uses `unicode.IsDigit` — inconsistent | `internal/model/entities.go` |
| Q5 | `display.go` functions don't handle Plan IDs — they'd be mangled by TSID display formatting | `internal/id/display.go` |
| Q6 | `RecordToDocument` signature claims error return but body never returns one | `internal/storage/document_store.go` |
| Q7 | `LINKS_TO` graph edge sets `ToType: "section"` when target is a document path | `internal/docint/graph.go` |
| Q8 | `Concept.Aliases` field declared but never populated by any code path | `internal/docint/types.go` |
| Q9 | `FindByRole` can return duplicates when Layer 2 and Layer 3 agree on a section's role | `internal/service/intelligence.go` |
| Q10 | Dead code: `planListResultsWithDisplay()` and `jsonResultAny()` never called | `internal/mcp/plan_tools.go` |
| Q11 | Misplaced doc comment: `DocumentExists` carries `SupersessionChain`'s comment | `internal/service/documents.go` |
| Q12 | Owner entity existence not validated — only format check, no referential integrity | `internal/service/documents.go` |

### 3.4 Test gaps

| ID | Summary | Severity |
|----|---------|----------|
| T1 | No MCP-level integration tests for any Phase 2a tools — `setupTestServer` only registers Phase 1 tools | High |
| T2 | Plan YAML round-trip test missing | Medium |
| T3 | Plan lifecycle transition tests missing (no Plan cases in `lifecycle_test.go`) | Medium |
| T4 | Phase 2 Feature lifecycle transition tests missing (`proposed → designing → specifying → ...`) | Medium |
| T5 | Phase 2 Feature fields round-trip test missing (`parent`, `design`, `dev_plan`, `tags`) | Medium |
| T6 | `UpdatePlanStatus` / `UpdatePlan` / `GetPlan` / `ListPlans` service tests missing | Medium |
| T7 | Optimistic lock conflict during `ApproveDocument` / `SupersedeDocument` untestable (blocked by B4) | Medium |
| T8 | `ClassifiedAt` through MCP handler code path untested (masked by test helper setting it explicitly) | Medium |
| T9 | `MatchConventionalRole` with headings containing multiple keywords untested | Low |
| T10 | `SupersedeDocument` when document is not `approved` — negative path untested | Low |
| T11 | `TestCreatePlan_Success` skipped with `t.Skip` — requires config setup | Low |
| T12 | Compile-time interface check `var _ model.Entity = model.Plan{}` missing | Low |

### 3.5 Documentation inaccuracies

| ID | Document | Summary |
|----|----------|---------|
| D1 | `phase-2a-progress.md` | Claims all 11 acceptance criteria met — at least 5 have bugs or missing implementations |
| D2 | `phase-2a-progress.md` | Omits all 7 bugs found in audit |
| D3 | `phase-2a-progress.md` | Lists incorrect tool names (e.g. `config_get` vs actual `get_project_config`) |
| D4 | `phase-2a-progress.md` | Claims Feature field renames complete — only done at struct level, not service/MCP layer |
| D5 | `phase-2a-progress.md` | Claims "all tests pass with race detector" — `ResolvePrefix` test fails |
| D6 | `README.md` | Phase 2a status section lists 5 items as remaining that are reported complete |
| D7 | `README.md` | Missing 15 MCP tools from tool listing (intelligence, query, migration tools) |
| D8 | `README.md` | Missing 4 packages from repository structure (`docint`, `core`, `fsutil`, `testutil`) |
| D9 | `AGENTS.md` | Missing `internal/docint/` and `.kbz/index/` from repository structure |
| D10 | `AGENTS.md` | Scope guard section is Phase 1 only — Phase 2a is now active |
| D11 | `bootstrap-workflow.md` | Still references Epics (deprecated; replaced by Plans) |
| D12 | `bootstrap-workflow.md` | Lists several items as "deferred until tool exists" that are now implemented |

---

## 4. Remediation Tracks

### 4.1 Track A1 — Critical bug fixes

**Goal:** Fix the two high-severity lifecycle bugs and the optimistic locking bypass.

**Tasks:**

1. **Fix B1: Remove Plan `done` from terminal states.**
   - File: `internal/validate/lifecycle.go`
   - Remove `PlanStatusDone` from `terminalStates[EntityPlan]`
   - Add `done → superseded` and `done → cancelled` to `allowedTransitions`
   - Add test cases in `lifecycle_test.go` for Plan `done → superseded` and `done → cancelled`

2. **Fix B2: Remove Feature `done` from terminal states.**
   - File: `internal/validate/lifecycle.go`
   - Remove `FeatureStatusDone` from `terminalStates[EntityFeature]`
   - Verify existing `done → superseded` entry in `allowedTransitions` is now reachable
   - Add `done → cancelled` to `allowedTransitions`
   - Add test cases in `lifecycle_test.go` for Feature `done → superseded` and `done → cancelled`

3. **Fix B3: Activate Phase 2 Feature entry state.**
   - File: `internal/validate/lifecycle.go`
   - Change `entryState[EntityFeature]` to use `phase2FeatureEntryState` (`"proposed"`)
   - Update `CreateFeature` to use `model.FeatureStatusProposed` instead of hardcoded `"draft"`
   - Add test for Feature creation starting at `proposed`

4. **Fix B4: Carry FileHash through DocumentToRecord.**
   - File: `internal/storage/document_store.go`
   - Add a `FileHash` parameter or field so that the service layer can pass the loaded hash through the model conversion
   - Verify `ApproveDocument` and `SupersedeDocument` now correctly engage optimistic locking
   - Add test for concurrent modification detection during approve/supersede

5. **Fix B8: Fix flaky ResolvePrefix test.**
   - File: `internal/service/entities_test.go`
   - Change test fixture ULIDs so that the "unambiguous" and "case_insensitive" sub-cases use IDs with distinct prefixes

**Outputs:**
- Lifecycle transitions match spec §9.1 and §9.2 exactly
- Optimistic locking works end-to-end for document operations
- All tests pass reliably

---

### 4.2 Track A2 — Medium-severity bug fixes

**Goal:** Fix the classification timestamp and determinism bugs.

**Tasks:**

1. **Fix B5: Set ClassifiedAt in doc_classify handler.**
   - File: `internal/mcp/doc_intelligence_tools.go`
   - Set `ClassifiedAt: time.Now().UTC()` when constructing `ClassificationSubmission`
   - Add test that verifies ClassifiedAt is non-zero when going through the handler code path

2. **Fix B6: Make MatchConventionalRole deterministic.**
   - File: `internal/docint/taxonomy.go`
   - Replace map iteration with an ordered slice for the keyword → role lookup
   - Add test case for heading containing multiple keywords (e.g. "Risk Assumptions")

3. **Fix B7: Collapse consecutive hyphens in NormalizeConcept.**
   - File: `internal/docint/concepts.go`
   - After space/underscore replacement, collapse runs of hyphens: `strings.Join(filterEmpty(strings.Split(name, "-")), "-")` or regexp
   - Update the test expectation for `"  spaces  everywhere  "` from `"spaces--everywhere"` to `"spaces-everywhere"`

**Outputs:**
- Classifications have correct timestamps
- Conventional role matching is deterministic
- Concept normalization handles whitespace variants correctly

---

### 4.3 Track A3 — Feature field rename completion

**Goal:** Complete the `epic` → `parent` and `plan` → `dev_plan` rename at all layers, and add missing Phase 2 fields to Feature creation.

**Tasks:**

1. **Update `featureFields()` to write Phase 2 field names.**
   - File: `internal/service/entities.go`
   - Change `"epic"` to `"parent"`, `"plan"` to `"dev_plan"` in the fields map
   - Add `"design"` and `"tags"` fields
   - Update canonical field ordering in `internal/storage/entity_store.go` if needed

2. **Update `CreateFeature` to accept Phase 2 inputs.**
   - File: `internal/service/entities.go`
   - Rename `Epic` to `Parent` in `CreateFeatureInput`
   - Accept `Design` and `Tags` optional inputs
   - Set initial status to `proposed` (per B3 fix)

3. **Update `create_feature` MCP tool to use Phase 2 field names.**
   - File: `internal/mcp/entity_tools.go`
   - Rename `epic` parameter to `parent`
   - Add optional `design` and `tags` parameters
   - Maintain backward compatibility if needed (accept both `epic` and `parent`)

4. **Update Feature YAML canonical field ordering per spec §8.2.**
   - File: `internal/storage/entity_store.go`
   - Ensure Feature field order matches: `id`, `slug`, `title`, `status`, `summary`, `parent`, `design`, `spec`, `dev_plan`, `acceptance`, `tags`, `created`, `created_by`, `updated`

5. **Add Phase 2 Feature round-trip tests (T5).**
   - Test YAML serialization with `parent`, `design`, `dev_plan`, `tags` fields

**Outputs:**
- Feature entities use Phase 2 field names in all layers
- CreateFeature accepts Phase 2 inputs
- YAML output matches spec §8.2 field ordering

---

### 4.4 Track A4 — Spec-compliant entity tool surface

**Goal:** Ensure generic entity tools support Plan, and expose filtered entity listing via MCP.

**Tasks:**

1. **Add `plan` to generic entity tool enums (S7).**
   - File: `internal/mcp/entity_tools.go`
   - Add `"plan"` to `get_entity`, `list_entities`, `update_status`, `update_entity` enum values
   - Wire Plan handling in each handler (delegate to PlanService)

2. **Create MCP tool for filtered entity listing (S9).**
   - File: `internal/mcp/query_tools.go` or `entity_tools.go`
   - Expose `ListEntitiesFiltered` via an MCP tool with status, type, tags, parent, and date range parameters
   - Or extend `list_entities` with optional filter parameters

3. **Remove deprecated Phase 1 document tools per spec (S8).**
   - File: `internal/mcp/document_tools.go`, `internal/mcp/server.go`
   - Remove `update_document_body`, `extract_from_document`, `retrieve_document` tool registrations
   - Retain `scaffold_document` and `validate_document` (spec says to keep these)

4. **Remove dead code (Q10).**
   - File: `internal/mcp/plan_tools.go`
   - Remove unused `planListResultsWithDisplay()` and `jsonResultAny()`

**Outputs:**
- Generic entity tools work with Plans
- Filtered entity listing available via MCP
- Deprecated tools removed
- No dead code

---

### 4.5 Track A5 — Config spec compliance

**Goal:** Align config struct with spec §10.2.

**Tasks:**

1. **Rename `PrefixEntry.Label` to `Name` (S1).**
   - File: `internal/config/config.go`
   - Rename `Label` field to `Name`, change yaml tag from `"label"` to `"name"`
   - Update all references across config.go, config_test.go, config_tools.go, plans.go, migration.go, etc.

2. **Add optional `Description` field to `PrefixEntry` (S2).**
   - File: `internal/config/config.go`
   - Add `Description string yaml:"description,omitempty"`
   - Expose in `add_prefix` tool and `get_prefix_registry` output

3. **Validate Name is non-empty (Q3).**
   - File: `internal/config/config.go`
   - Add check in `Validate()` that every prefix entry has a non-empty `Name`

4. **Remove unused `ConfigDir` constant (Q2).**
   - File: `internal/config/config.go`
   - Delete the unused constant

**Outputs:**
- Config YAML uses `name` field per spec
- Optional `description` supported
- Empty names rejected on validation

---

### 4.6 Track A6 — Document intelligence gaps

**Goal:** Address missing spec requirements in the document intelligence layers.

**Tasks:**

1. **Add `DOC-xxx` entity reference pattern (S12).**
   - File: `internal/docint/extractor.go`
   - Add `{regexp.MustCompile(`\bDOC-[A-Za-z0-9]+\b`), "document"}` to `entityPatterns`
   - Add test case for DOC references

2. **Fix `LINKS_TO` edge `ToType` (Q7).**
   - File: `internal/docint/graph.go`
   - Change `ToType: "section"` to `ToType: "document"` for LINKS_TO edges
   - Update graph tests

3. **Fix `FindByRole` deduplication (Q9).**
   - File: `internal/service/intelligence.go`
   - Deduplicate results when Layer 2 and Layer 3 agree on a section's role
   - Add test case

4. **Assess content block identification (S10) — decision required.**
   - Spec §12.2 requires content blocks (paragraphs, lists, tables, code blocks) identified within sections
   - Current implementation provides section-level granularity only
   - **Decision needed:** Is the current section-level granularity sufficient for Phase 2a, or must content blocks be implemented? If deferred, document the rationale.

5. **Assess missing graph edge types (S11) — decision required.**
   - Spec §12.7 defines 8 edge types; 5 are implemented (`CONTAINS`, `REFERENCES`, `LINKS_TO`, `INTRODUCES`, `USES`)
   - Missing: `DEPENDS_ON` (requires Layer 3 dependency classification), `SUPERSEDES` (derivable from supersession chain), `REFINES` (derivable from front matter)
   - **Decision needed:** Should `SUPERSEDES` be derived automatically from the supersession chain? Should `REFINES` be derived from front matter `Basis:` fields? Is `DEPENDS_ON` deferred until explicit fragment dependency classification exists?

6. **Use canonical serializer for index files (S13).**
   - File: `internal/docint/store.go`
   - Replace `yaml.Marshal(v)` with the project's canonical serializer (`storage.MarshalCanonicalYAML` or equivalent)
   - Ensure trailing newline, block style

7. **Use atomic writes for index files (S14).**
   - File: `internal/docint/store.go`
   - Use `fsutil.WriteFileAtomic` (or equivalent write-to-temp-then-rename) instead of `os.WriteFile`

8. **Fix misplaced doc comment (Q11).**
   - File: `internal/service/documents.go`
   - Move `SupersessionChain` comment to the correct function

9. **Remove unused `Concept.Aliases` field or document its purpose (Q8).**
   - File: `internal/docint/types.go`
   - Either remove the field (it's never populated) or add a TODO noting it's reserved for future use

**Outputs:**
- DOC references extracted from documents
- Graph edges semantically correct
- Index storage meets P1-DEC-008 deterministic serialization requirements
- Index writes are crash-safe

---

### 4.7 Track A7 — Test coverage

**Goal:** Fill the most critical test gaps.

**Tasks:**

1. **Add MCP-level integration tests for Phase 2a tools (T1).**
   - File: `internal/mcp/server_test.go` or new `phase2a_test.go`
   - Extend `setupTestServer` to register Phase 2a tools (plan, doc record, config, intelligence, query, migration)
   - Add at least one integration test per tool category: plan CRUD, document lifecycle, classification, config operations, queries

2. **Add Plan lifecycle transition tests (T3).**
   - File: `internal/validate/lifecycle_test.go`
   - Test all Plan transitions: `proposed → designing → active → done`, `done → superseded`, `done → cancelled`, `proposed → superseded`, invalid transitions

3. **Add Phase 2 Feature lifecycle transition tests (T4).**
   - File: `internal/validate/lifecycle_test.go`
   - Test all Phase 2 Feature transitions: `proposed → designing → specifying → dev-planning → developing → done`, shortcuts, backward transitions

4. **Add Plan YAML round-trip test (T2).**
   - Test write → read → write → compare for a Plan entity with all fields populated

5. **Add Plan service tests (T6).**
   - Tests for `UpdatePlanStatus`, `UpdatePlan`, `GetPlan`, `ListPlans` at the service level

6. **Add compile-time interface checks (T12).**
   - Add `var _ model.Entity = model.Plan{}` and `var _ model.Entity = model.DocumentRecord{}` in model tests

**Outputs:**
- Phase 2a tools have integration test coverage
- All lifecycle transitions tested
- YAML round-trip verified for Plans

---

### 4.8 Track A8 — Documentation correction

**Goal:** Bring all documentation into alignment with actual implementation state.

**Tasks:**

1. **Correct `phase-2a-progress.md` (D1–D5).**
   - Update status summary to reflect audit findings
   - Add all bugs to Known Issues section
   - Fix tool names to match actual registered names
   - Correct Feature field rename claim
   - Update test pass claim
   - Adjust acceptance criteria checkboxes to reflect actual state

2. **Update `README.md` (D6–D8).**
   - Update Phase 2a status section to reflect current state
   - Add missing packages to repository structure (`docint`, `core`, `fsutil`, `testutil`)
   - Add missing MCP tools to tool listing (intelligence, query, migration tools)

3. **Update `AGENTS.md` (D9–D10).**
   - Add `internal/docint/` to repository structure
   - Add `.kbz/index/` to `.kbz/` structure description
   - Update scope guard to cover Phase 2a (or note Phase 1 is complete)

4. **Update `bootstrap-workflow.md` (D11–D12).**
   - Replace Epic references with Plan
   - Update the "deferred until tool exists" list to acknowledge implemented items

**Outputs:**
- All workflow documentation accurately reflects implementation state
- New developers/agents get correct information from docs

---

### 4.9 Track A9 — Minor code quality fixes

**Goal:** Clean up minor quality issues that don't fit other tracks.

**Tasks:**

1. **Fix digit check inconsistency (Q4).**
   - File: `internal/model/entities.go`
   - Change `IsPlanID` digit check from ASCII range comparison to `unicode.IsDigit()` for consistency with `ValidatePrefix`

2. **Handle Plan IDs in display functions (Q5).**
   - File: `internal/id/display.go`
   - Add Plan ID handling — pass through unchanged since Plan IDs are already human-readable

3. **Fix `RecordToDocument` signature (Q6).**
   - File: `internal/storage/document_store.go`
   - Either return parsing errors or drop the error from the signature

4. **Add owner entity existence check (Q12).**
   - File: `internal/service/documents.go`
   - In `ValidateDocument`, check that the owner entity actually exists (not just ID format validation)

5. **Resolve `parsePlanIDParts` duplication (Q1).**
   - Consider extracting shared Plan ID parsing logic to a leaf package to eliminate the copy

**Outputs:**
- Consistent digit validation across codebase
- Display functions handle all entity types
- Document validation includes referential integrity

---

## 5. Execution Record

| Track | Description | Status |
|-------|-------------|--------|
| A1 | Critical bug fixes (lifecycle, locking, flaky test) | Not started |
| A2 | Medium-severity bug fixes (timestamp, determinism) | Not started |
| A3 | Feature field rename completion | Not started |
| A4 | Spec-compliant entity tool surface | Not started |
| A5 | Config spec compliance | Not started |
| A6 | Document intelligence gaps | Not started |
| A7 | Test coverage | Not started |
| A8 | Documentation correction | Not started |
| A9 | Minor code quality fixes | Not started |

---

## 6. Decisions Required

Two items in Track A6 require human decision before implementation:

| Item | Question | Options |
|------|----------|---------|
| S10 | Content block identification within sections | (a) Implement for Phase 2a, (b) Defer to Phase 2b with documented rationale |
| S11 | Missing graph edge types (DEPENDS_ON, SUPERSEDES, REFINES) | (a) Implement SUPERSEDES and REFINES (derivable from existing data), defer DEPENDS_ON, (b) Defer all three with documented rationale |

---

## 7. Estimated Scope

| Track | Estimated effort | Priority |
|-------|-----------------|----------|
| A1 — Critical bugs | Small | Must fix |
| A2 — Medium bugs | Small | Must fix |
| A3 — Feature field rename | Medium | Must fix |
| A4 — Entity tool surface | Medium | Should fix |
| A5 — Config compliance | Small | Should fix |
| A6 — Doc intelligence gaps | Medium–Large (depends on decisions) | Should fix (some items), Decision required (others) |
| A7 — Test coverage | Medium | Should fix |
| A8 — Documentation | Small | Must fix |
| A9 — Minor quality | Small | Nice to have |

**Recommended execution order:** A1 → A2 → A3 → A8 → A5 → A4 → A7 → A6 → A9

Rationale: Fix bugs first (A1, A2), then complete the field rename (A3) since other tracks depend on the correct field names. Update documentation (A8) early so subsequent changes are tracked accurately. Config (A5) and tool surface (A4) are straightforward. Tests (A7) should be written against the fixed code. Document intelligence (A6) is last because it may be partially deferred pending decisions.

---

## 8. Acceptance Criteria

Phase 2a remediation is complete when:

1. All bugs B1–B8 are fixed and have regression tests
2. All spec deviations S1–S9 are resolved (code matches spec)
3. Spec deviations S10–S14 are either resolved or have documented deferral decisions
4. All test gaps T1–T6 are filled (T7–T12 are nice-to-have)
5. All documentation items D1–D12 are corrected
6. `go build ./...`, `go vet ./...`, and `go test -race ./...` all pass clean
7. `phase-2a-progress.md` accurately reflects implementation state