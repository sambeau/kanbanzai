---
id: DOC-01KM8JVA98M33
type: implementation-plan
title: Phase 1 Remaining Work
status: submitted
feature: FEAT-01KM8JTF0MK91
created_by: human
created: 2026-03-21T16:14:58Z
updated: 2026-03-21T16:14:58Z
---
# Phase 1 Remaining Work

- Status: implementation plan
- Purpose: define the remaining work to complete Phase 1, organized into work packages with dependencies and execution order
- Date: 2026-07-25
- Based on:
  - `work/spec/phase-1-specification.md`
  - `work/spec/id-system-specification.md`
  - `work/spec/bootstrap-specification.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/plan/phase-1-decision-log.md`

---

## 1. Current State

The Phase 1 workflow kernel is substantially complete. All five entity types, lifecycle enforcement, CRUD operations, document support, health checks, validation, MCP server (20 tools), CLI, and SQLite cache are implemented and passing (292 tests, zero failures).

Three areas of work remain:

1. **ID system migration** — replace sequential IDs (`FEAT-001`, `E-001`, `FEAT-001.1`) with TSID13-based IDs and epic slugs, including the task field rename and test fixture updates.
2. **Bootstrap activation** — create initial records, register documents, commit `.kbz/` state.
3. **Decision log housekeeping** — mark resolved follow-ups, update superseded decisions, align doc references with implementation.

---

## 2. Work Packages

### WP1: TSID13 Core (`internal/id`)

Build the new ID generation primitives. This is the foundation — everything else depends on it.

**Scope:**

- Implement Crockford base32 encoder (encode/decode, canonical uppercase, case-insensitive decode).
- Implement TSID13 generator: 48-bit millisecond timestamp + 15-bit `crypto/rand` random, producing exactly 13 uppercase Crockford base32 characters.
- Implement epic slug validator per ID spec §8 (length, charset, no leading/trailing/consecutive hyphens, case normalisation).
- Implement unified allocator interface: accepts entity type, returns correctly formatted ID. Routes epics to slug validation, all others to TSID13 generation.
- Implement local collision check with 3-retry loop (ID spec §6.2). The allocator needs a function to check whether an ID exists in the store.
- Implement input normalisation: strip break hyphens, normalise case.

**Files:**

| File | Action |
|------|--------|
| `internal/id/tsid.go` | New — Crockford base32 codec and TSID13 generator |
| `internal/id/tsid_test.go` | New — encoding round-trips, timestamp ordering, uniqueness (10k in a loop), character set validation |
| `internal/id/slug.go` | New — epic slug validation per §8 |
| `internal/id/slug_test.go` | New — valid/invalid slug cases, normalisation |
| `internal/id/allocator.go` | Rewrite — unified allocator interface replacing scan-max-increment |
| `internal/id/allocator_test.go` | Rewrite — test routing for all 6 entity types, invalid type error, collision retry |

**Acceptance criteria (from ID spec):**

- §14.2 — TSID13 values are exactly 13 chars, Crockford base32, uppercase, time-sortable.
- §14.3 — 10,000 IDs in a tight loop produce no duplicates.
- §14.4 — collision check retries without caller-visible error.
- §14.5 — invalid epic slugs rejected with explanatory error.
- §14.11 — two IDs created ≥2ms apart sort lexicographically in chronological order.
- §14.12 — unified allocator routes all 6 types correctly, rejects invalid types.

**Estimated effort:** Medium-large. The Crockford encoder is straightforward; the allocator restructuring requires careful interface design for the collision-check callback.

---

### WP2: Entity Model Update (`internal/model`)

Update the entity model to reflect TSID13 ID formats and the task field rename.

**Scope:**

- Change the `EntityKind` constants: `EntityKindEpic` value from `"epic"` to whatever the model uses, and add the type prefix mapping (`Epic→EPIC`, `Feature→FEAT`, etc.) if not already present. Verify the 6-type routing table matches the ID spec §3.
- Add `EntityKindDocument` if not present — the unified allocator must handle all six types including Document.
- Rename `Task.Feature` to `Task.ParentFeature` with YAML tag `parent_feature` (ID spec §13.4).
- Update the deterministic field-order serialisation for Task to use the new field name.

**Files:**

| File | Action |
|------|--------|
| `internal/model/entities.go` | Edit — rename Task field, add Document entity kind if missing |
| `internal/model/entities_test.go` | Edit — update any tests referencing the old field name |

**Acceptance criteria:**

- Task YAML files serialize with `parent_feature:` not `feature:`.
- The model defines all six entity kinds used by the allocator.

**Estimated effort:** Small. Mechanical rename with ripple effects in WP3.

---

### WP3: Storage and Service Migration

Plumb the new IDs through the storage layer, service layer, and interfaces.

**Scope:**

This is the integration work — connecting the new allocator to entity creation, updating the filename parser, and wiring prefix resolution through the service layer into MCP and CLI.

**3a. Storage (`internal/storage`):**

- Update filename format to `{CANONICAL-ID}-{slug}.yaml` with TSID13 IDs.
- Update the filename parser: for TSID-based types, extract the 13-char TSID after the type-prefix hyphen; for epics, extract the slug portion of the ID.
- Retain recognition of legacy sequential filenames for robustness (ID spec §13.6).

**3b. Service (`internal/service`):**

- Replace calls to the old allocator with the unified allocator from WP1.
- For `CreateTask`: accept `parent_feature` instead of `feature`.
- Add prefix resolution: `GetEntity` and `UpdateEntity` (and `UpdateStatus`) must accept ID prefixes and resolve them via the store (ID spec §10).
- Prefix resolution must strip break hyphens from input, normalise case, and handle ambiguous/no-match cases.

**3c. Document service (`internal/document`):**

- Replace the in-memory counter closure with TSID13 generation using `DOC` prefix.
- Use the unified allocator (or at minimum the same TSID13 generator).

**3d. Validation (`internal/validate`):**

- Update ID format validation to accept TSID13-based IDs (and epic slug IDs) instead of sequential format.
- Update task validation: the required-field check must reference `parent_feature` not `feature`.
- Update cross-reference validation (health check): task→feature references use `parent_feature`.

**3e. MCP tools (`internal/mcp`):**

- `create_task` tool: rename the `feature` input parameter to `parent_feature`.
- All `get_entity`, `update_status`, `update_entity` tools: pass through prefix resolution (accept short IDs).
- Responses should include display-formatted IDs where appropriate (full display form with break hyphen).

**3f. CLI (`cmd/kanbanzai`):**

- `create task` command: rename the `--feature` flag to `--parent-feature`.
- All commands accepting `--id`: pass through prefix resolution.
- Output should use display-formatted IDs.

**Files:**

| File | Action |
|------|--------|
| `internal/storage/entity_store.go` | Edit — filename format, parser |
| `internal/storage/entity_store_test.go` | Edit — new filename format in tests |
| `internal/service/entities.go` | Edit — new allocator, prefix resolution, field rename |
| `internal/service/entities_test.go` | Edit — test prefix resolution, updated field names |
| `internal/document/service.go` | Edit — TSID13 allocation |
| `internal/document/service_test.go` | Edit — updated ID format in tests |
| `internal/validate/entity.go` | Edit — ID format rules, field name |
| `internal/validate/entity_test.go` | Edit — updated validation tests |
| `internal/validate/health.go` | Edit — cross-reference field name |
| `internal/validate/health_test.go` | Edit — updated health check tests |
| `internal/mcp/entity_tools.go` | Edit — parameter rename, prefix resolution |
| `internal/mcp/server_test.go` | Edit — updated tool parameter names |
| `cmd/kanbanzai/main.go` | Edit — flag rename, prefix resolution |
| `cmd/kanbanzai/main_test.go` | Edit — updated CLI tests |

**Acceptance criteria (from ID spec):**

- §14.1 — all six entity types can be created and receive valid IDs.
- §14.6 — canonical form used in files, references, filenames (no break hyphens stored).
- §14.7 — prefix matching works: unambiguous→resolve, ambiguous→candidates, none→error, strips hyphens, case-insensitive.
- §14.9 — filenames use `{CANONICAL-ID}-{slug}.yaml`, parser extracts ID and slug correctly.
- §14.10 — round-trip integrity (write→read→write produces identical files).

**Estimated effort:** Large. This is the widest-reaching change — it touches nearly every package. The prefix resolution logic is new behavior requiring careful integration.

**Dependencies:** WP1 (allocator), WP2 (model).

---

### WP4: Display Layer

Implement the display conventions for human-readable ID output.

**Scope:**

- Implement break hyphen insertion: for TSID-based IDs, insert `-` after the 5th TSID character in full and short display forms.
- Implement shortest-unique-prefix computation: given a set of entities of the same type, find the minimum prefix length (≥5 chars) that uniquely identifies each one.
- Wire display formatting into CLI output and MCP responses (listings, entity details).

**Files:**

| File | Action |
|------|--------|
| `internal/id/display.go` | New — break hyphen insertion, short display computation |
| `internal/id/display_test.go` | New — formatting tests, shortest-prefix computation |

Plus edits to CLI and MCP output formatting (may overlap with WP3 files).

**Acceptance criteria (from ID spec):**

- §14.8 — break hyphen at position 5 in full/short display; shortest unique prefix per type; never stored.

**Estimated effort:** Small-medium. The logic is self-contained; the wiring into output paths is the main integration work.

**Dependencies:** WP1 (needs TSID structure knowledge). Can proceed in parallel with WP3.

---

### WP5: Test Fixture Migration

Update all test data and test helpers to use TSID13-format IDs.

**Scope:**

- Define a set of well-known test constants in TSID13 format (e.g., `TestEpicID = "EPIC-TESTPROJECT"`, `TestFeatureID = "FEAT-01J3K7MXP3RT5"`, `TestTaskID = "TASK-01J3KZZZBB4KF"`, etc.) for use across all test files.
- Rewrite all YAML fixtures in `testdata/` directories with TSID13 IDs and `parent_feature` field for tasks.
- Update all test code that constructs entities with hard-coded sequential IDs.
- Add at least one legacy-ID recognition test: create a YAML file with a sequential ID (`FEAT-001`), read it, confirm no error (ID spec §13.6 / §14.13).

**Files:**

Every `*_test.go` and every `testdata/` file across `internal/storage`, `internal/service`, `internal/validate`, `internal/cache`, `internal/mcp`, `internal/document`.

**Acceptance criteria (from ID spec):**

- §14.13 — all fixtures use TSID13 format; at least one legacy recognition test passes.
- Full test suite passes with zero failures.

**Estimated effort:** Medium. Mechanical but wide-reaching. Best done as a single sweep after WP1–WP3 are integrated.

**Dependencies:** WP1, WP2, WP3 (the new code must exist before tests can target it).

---

### WP6: Specification and Decision Log Updates

Update documents that reference the old ID format or the old task field name. This is a documentation-only work package with no code changes.

**Scope:**

| Item | Document | Change |
|------|----------|--------|
| Task field name | `work/spec/phase-1-specification.md` §9.3 | Change `feature` to `parent_feature` in the minimum fields list |
| Task field name | `work/plan/phase-1-decision-log.md` P1-DEC-009 | Update Task table: `feature` → `parent_feature`, update `id` default from "feature-local sub-ID" to "TSID13" |
| Filename format | `work/plan/phase-1-decision-log.md` P1-DEC-006 | Update file layout examples to show TSID13 filename format |
| Superseded decision | `work/plan/phase-1-decision-log.md` P1-DEC-007 | Mark all follow-ups as superseded by P1-DEC-021 |
| Resolved follow-ups | `work/plan/phase-1-decision-log.md` | Strike through follow-ups that are now resolved by implementation or by the bootstrap/ID specs. Candidates: P1-DEC-009 (MCP alignment, enum values), P1-DEC-015 (retirement criteria), P1-DEC-016 (Go module, MCP library), P1-DEC-021 (after implementation completes) |

**Estimated effort:** Small. Straightforward text edits.

**Dependencies:** None — can be done at any time, but best done after WP1–WP3 are settled so the edits reflect the final state.

---

### WP7: Bootstrap Activation

Create initial records through the tool and begin self-hosting. Per the bootstrap spec, this does not depend on which allocator is active — it can use sequential or TSID13 IDs. However, doing it after ID migration avoids creating records that would need format migration later.

**Scope:**

- Create one Epic for Phase 1 (e.g., `EPIC-PHASE1`), status `active`.
- Create Features for the remaining and ongoing areas of Phase 1 work, linked to the Epic.
- Register all existing documents in `work/design/`, `work/spec/`, `work/plan/`, `work/research/`, `work/bootstrap/` through the tool's document operations.
- Run `kbz health` and `kbz validate` — both must pass with no errors.
- Verify creating a new Task, Bug, and Decision works correctly with proper cross-references.
- Commit `.kbz/` to Git (ensure it is not in `.gitignore`).
- Record bootstrap workflow retirement criteria as trackable items.

**Acceptance criteria (from bootstrap spec):**

- §9.1 — `.kbz/state/` contains at least one Epic with status `active`.
- §9.2 — at least one Feature linked to the Epic, status `open` or `in-progress`.
- §9.3 — all documents in the five `work/` subdirectories are registered and retrievable by ID.
- §9.4 — `kbz health` passes.
- §9.5 — `kbz validate` passes.
- §9.6 — `.kbz/` is committed to Git.
- §9.7 — new Task, Bug, Decision creation works with correct cross-references.
- §9.8 — retirement criteria are documented and trackable.

**Estimated effort:** Medium. The document registration is the bulk of the work — there are ~15–20 documents to register. The rest is straightforward CLI/MCP usage.

**Dependencies:** All code work packages (WP1–WP5) should be complete first so bootstrap records use final ID formats.

---

## 3. Dependency Graph

```
WP1: TSID13 Core ─────────┐
                           ├──► WP3: Storage & Service ──► WP5: Test Migration
WP2: Entity Model Update ─┘         │
                                     │
WP4: Display Layer ──────────────────┘  (can start after WP1, parallel with WP3)

WP6: Doc Updates ──────────────────────  (independent, anytime)

WP7: Bootstrap ────────────────────────  (after WP1–WP5 complete)
```

Critical path: **WP1 → WP3 → WP5 → WP7**.

WP2 is small and can be done as part of WP1 or WP3. WP4 can proceed in parallel with WP3. WP6 is independent.

---

## 4. Recommended Execution Order

### Phase A: Foundation (WP1 + WP2)

Build the new allocator and update the entity model. These are self-contained and can be fully tested in isolation before touching any other package.

1. Implement Crockford base32 codec with tests.
2. Implement TSID13 generator with tests (ordering, uniqueness, character set).
3. Implement epic slug validator with tests.
4. Implement unified allocator interface with collision-check callback.
5. Rename Task `Feature` → `ParentFeature` in model.
6. Add `EntityKindDocument` to model if missing.

**Gate:** All new unit tests pass. The old allocator tests will break — that's expected and addressed in Phase C.

### Phase B: Display (WP4)

Can proceed in parallel with Phase C if multiple agents are available.

1. Implement break hyphen insertion (format and strip).
2. Implement shortest-unique-prefix computation.
3. Unit test both.

**Gate:** Display formatting tests pass in isolation.

### Phase C: Integration (WP3)

The largest and most complex phase. Systematically update each layer.

Recommended order within WP3 (each step should compile and could be committed):

1. **Storage** — update filename format and parser. Add legacy filename recognition. At this point, existing tests will fail (they use old ID formats). That's expected.
2. **Validation** — update ID format validation, field name, cross-reference checks.
3. **Service** — wire new allocator, rename task field in create/update paths, add prefix resolution.
4. **Document service** — replace counter with TSID13 generation.
5. **MCP** — rename parameter, wire prefix resolution into get/update tools, add display formatting to responses.
6. **CLI** — rename flag, wire prefix resolution, add display formatting.

**Gate:** `go build ./...` succeeds. Individual package tests may still fail until fixtures are updated in Phase D.

### Phase D: Test Migration (WP5)

Sweep through all test files and fixtures.

1. Define well-known test ID constants.
2. Update `testdata/` fixtures.
3. Update test helpers and assertions.
4. Add legacy ID recognition test.
5. Run full suite: `go test ./...` — must pass with zero failures.
6. Run with race detector: `go test -race ./...` — must pass.

**Gate:** Full test suite green. This is the "it works" gate.

### Phase E: Documentation (WP6)

Update specs and decision log to match the implementation.

1. Update Phase 1 spec §9.3.
2. Update P1-DEC-009 Task table.
3. Update P1-DEC-006 filename examples.
4. Mark P1-DEC-007 follow-ups as superseded.
5. Strike through resolved follow-ups across the decision log.

**Gate:** Documents are internally consistent and match the code.

### Phase F: Bootstrap (WP7)

Activate self-hosting.

1. Create the Phase 1 Epic.
2. Create Features for ongoing work.
3. Register all existing documents.
4. Run health check and validate.
5. Test new entity creation (Task, Bug, Decision).
6. Commit `.kbz/` to Git.
7. Record retirement criteria.

**Gate:** All 8 bootstrap acceptance criteria (bootstrap spec §9.1–§9.8) pass.

---

## 5. Items Explicitly Not in Scope

The following were identified during the remaining-work survey but are **not Phase 1 work**:

| Item | Disposition |
|------|-------------|
| Concurrency model for lost updates and state machine violations | P1-DEC-021 follow-up. Phase 1 spec §19 requires only that the architecture not block future concurrency support. TSID13 IDs satisfy the ID-safety requirement. The broader concurrency design is Phase 2 work. |
| Rich server-side query/filtering | P1-DEC-020: Phase 1 is list-by-type only. Server-side filters are Phase 2 (cache layer). |
| Document-to-entity extraction automation | P1-DEC-019: agent-driven, not automated. The MCP tools exist; agent guidance is a Phase 2 concern. |
| Link resolution | P1-DEC-017: deferred to Phase 2. |
| Duplicate detection | P1-DEC-018: deferred to Phase 2. |
| Cache schema expansion for documents | P1-DEC-013 follow-up: Phase 2. |
| Agent orchestration | Phase 1 spec §5.2: explicitly excluded. |

---

## 6. Risk Notes

**ID migration is wide-reaching.** The field rename and ID format change touch nearly every package. The recommended approach is to make the core changes first (WP1, WP2), then integrate layer by layer (WP3), and fix all tests in a single sweep (WP5). Attempting to keep all tests green throughout the migration would require updating fixtures incrementally, which is more error-prone than a clean sweep.

**Bootstrap records use final IDs.** The bootstrap spec allows either allocator, but using TSID13 IDs for bootstrap records avoids a second migration. Complete ID migration before bootstrap activation.

**Decision log housekeeping is low-risk but high-volume.** There are ~20 follow-up items to resolve. Most are already addressed by implementation — they just need to be struck through. Batch this work rather than interleaving it with code changes.

---

## 7. Summary

| Phase | Work Package | Effort | Dependencies |
|-------|-------------|--------|--------------|
| A | WP1: TSID13 Core + WP2: Model Update | Medium-large | None |
| B | WP4: Display Layer | Small-medium | WP1 |
| C | WP3: Storage & Service Migration | Large | WP1, WP2 |
| D | WP5: Test Fixture Migration | Medium | WP1–WP3 |
| E | WP6: Spec & Decision Log Updates | Small | None (but best after WP3) |
| F | WP7: Bootstrap Activation | Medium | WP1–WP5 |

Critical path: **A → C → D → F**.

Phase B (display) and Phase E (docs) can proceed in parallel with the critical path when agent capacity allows.