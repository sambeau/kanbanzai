# Phase 2b Implementation Plan

- Status: active
- Purpose: define the work breakdown, sequencing, and constraints for Phase 2b implementation
- Date: 2026-03-24
- Based on:
  - `work/spec/phase-2b-specification.md`
  - `work/spec/phase-2-specification.md`
  - `work/design/machine-context-design.md`
  - `work/plan/phase-2-decision-log.md` (P2-DEC-001 through P2-DEC-004)
  - `work/plan/phase-1-implementation-plan.md` (format reference)

---

## 1. Purpose

This document defines the concrete implementation plan for Phase 2b: context management, knowledge persistence, and agent capabilities. It provides sequencing, work breakdown, and constraints so that implementation can proceed without ambiguity about what to build, in what order, and why.

---

## 2. Phase 2b Outcome

When Phase 2b is complete, the system will:

1. auto-resolve user identity from git config or local config — no more `created_by: human`
2. store persistent knowledge entries contributed by agents, with lifecycle and confidence tracking
3. deduplicate knowledge contributions using topic matching and Jaccard similarity
4. define context profiles as named YAML bundles with CSS-style inheritance
5. assemble targeted context packets for agents, composing profiles, knowledge, and design fragments within a byte budget
6. report knowledge usage at task completion, driving the confidence feedback loop
7. suggest entity links from free-text and surface potential duplicates at entity creation time
8. provide structured extraction guidance for turning approved documents into entities
9. import existing project documents in batch for practical onboarding
10. validate all new state through extended health checks

---

## 3. Planning Principles

### 3.1 Foundation first

User identity resolution is small and unblocks every creation operation. Build it first. Knowledge entries are the core data type — build them before anything that depends on them.

### 3.2 Each track delivers a working vertical

Every track produces a working capability with its own MCP tools, tests, and validation. MCP tools are built alongside their feature, not as a separate integration pass.

### 3.3 The assembly is the product

`context_assemble` is the keystone tool that agents will call at the start of every session. Everything before Track F exists to feed data into the assembler. Everything after Track F consumes the same data through different lenses. Get assembly right.

### 3.4 Start with YAML, add cache if needed

Knowledge entries and profiles are stored as Git-tracked YAML files, read by scanning directories. At realistic scales (tens to low hundreds of entries), this is fast enough. Do not add SQLite cache tables for knowledge queries unless profiling shows a bottleneck.

### 3.5 Agent UX over internal elegance

When making trade-offs, optimise for the agent's experience: fewer round-trips, actionable error messages, responses that include enough context to decide the next action without a follow-up call.

---

## 4. Deliverables

### 4.1 Product deliverables

- user identity resolution module (git config / local config chain)
- KnowledgeEntry entity type with storage, lifecycle, and confidence scoring
- knowledge contribution with deduplication (topic match + Jaccard similarity)
- usage reporting with auto-confirmation and auto-retirement side effects
- context profile storage, inheritance resolution, and validation
- context assembly algorithm with byte budgeting and priority trimming
- link resolution tool (entity suggestions from free-text)
- duplicate detection tool (candidate surfacing at entity creation)
- document extraction guide tool
- batch document import with path-based type inference
- extended health checks for knowledge and profile state
- MCP tools: 17 new operations per spec §17
- CLI commands: `kbz import`, `kbz knowledge`, `kbz profile`, `kbz context`

### 4.2 Instance deliverables for this project

- `.kbz/context/roles/base.yaml` — project-wide conventions profile
- `.kbz/context/roles/developer.yaml` — development conventions profile
- `.kbz/local.yaml` added to `.gitignore`
- existing `work/` documents imported as document records via batch import

### 4.3 Verification deliverables

- unit tests for all new packages
- round-trip serialisation tests for KnowledgeEntry and profile YAML
- lifecycle transition tests (valid and invalid)
- confidence scoring tests (Wilson formula verification)
- deduplication tests (exact match and Jaccard threshold)
- profile inheritance tests (resolution, cycle detection, missing reference)
- context assembly tests (budgeting, trimming, priority ordering)
- integration tests for MCP tool surface
- health check tests for new validation rules

---

## 5. Resolved Implementation Questions

The spec §21 lists five open questions. Resolved here for implementation:

### 5.1 Cache schema for knowledge entries

Start without cache tables. Scan `.kbz/state/knowledge/` YAML files directly. At realistic scales (< 500 entries), directory scanning completes in single-digit milliseconds. Add a `knowledge` SQLite table only if profiling shows a bottleneck — the cache is derived and rebuildable, so adding it later is non-breaking.

### 5.2 Context assembly performance

Same as §5.1. Assembly reads profiles (< 20 files) and knowledge entries (< 500 files) from disk. Profile this path during Track F implementation. If assembly latency exceeds 100ms, add cache-backed queries.

### 5.3 Stop word list for Jaccard deduplication

Use a minimal English stop word list (approximately 30 words): a, an, the, is, are, was, were, be, been, being, have, has, had, do, does, did, will, would, shall, should, may, might, can, could, in, on, of, for, to, and, or, but, not, with, at, by, from, as, it, this, that.

Hard-code the list as a package-level variable. No external dependencies, no configuration. The list can be extended later if deduplication proves too aggressive or too permissive.

### 5.4 Import path configuration

Use glob patterns under an `import` key in `.kbz/config.yaml`:

```yaml
import:
  type_mappings:
    - glob: "*/design/*"
      type: design
    - glob: "*/spec/*"
      type: specification
    - glob: "*/plan/*"
      type: report
    - glob: "*/research/*"
      type: research
```

The system provides built-in defaults matching the table in spec §14.2. Project config overrides or extends the defaults.

### 5.5 Profile creation tooling

No `kbz profile init` command in Phase 2b. Profiles are YAML files created by hand or by agents. A scaffolding command can follow later if adoption feedback demands it.

---

## 6. Recommended Implementation Order

### 6.1 Layer 1: User identity resolution

Implement:
- `.kbz/local.yaml` parsing
- git config fallback
- resolution chain module
- update all existing creation tools to use auto-resolution

Small, self-contained, unblocks every subsequent track. Ship this before touching anything else.

### 6.2 Layer 2: KnowledgeEntry core

Implement:
- entity type definition and storage
- deterministic YAML serialisation with canonical field order
- CRUD operations via MCP tools (`knowledge_contribute`, `knowledge_get`, `knowledge_list`, `knowledge_update`)
- lifecycle state machine (contributed → confirmed → disputed → stale → retired)

This is the core data type. Everything else reads or writes knowledge entries.

### 6.3 Layer 3: Confidence and deduplication

Implement:
- Wilson score lower bound computation
- confidence recomputation on count changes
- topic normalisation
- Jaccard similarity with stop word removal
- deduplication checks in `knowledge_contribute`
- `knowledge_confirm`, `knowledge_flag`, `knowledge_retire`, `knowledge_promote` tools

Builds directly on Layer 2. The dedup check runs inside the contribution flow.

### 6.4 Layer 4: Context profiles

Implement:
- profile file parsing and validation
- inheritance chain walking with cycle detection
- leaf-level replace resolution algorithm
- `profile_get` and `profile_list` MCP tools
- `.kbz/context/roles/` directory structure

Independent of Layers 2–3. Can be built in parallel with confidence/dedup work.

### 6.5 Layer 5: Usage reporting

Implement:
- `context_report` MCP tool
- use_count / miss_count updates
- confidence recomputation after reports
- auto-confirmation side effect (use_count ≥ 3, miss_count = 0)
- auto-retirement side effect (miss_count ≥ 2)

Ties the knowledge lifecycle to task completion. Depends on Layers 2–3.

### 6.6 Layer 6: Context assembly

Implement:
- assembly algorithm (profile + design context + Tier 2 + Tier 3 + task instructions)
- byte-based budgeting with configurable ceiling
- priority trimming (Tier 3 first, then Tier 2, then design context; profile and task never trimmed)
- confidence filtering per tier (Tier 2 ≥ 0.3, Tier 3 ≥ 0.5)
- `context_assemble` MCP tool
- assembly with and without task_id

The keystone. Depends on Layers 2–4 (knowledge entries and profiles). Consumes Phase 2a document intelligence for design context.

### 6.7 Layer 7: Agent capabilities

Implement:
- `suggest_links` — entity reference pattern matching against known entities
- `check_duplicates` — Jaccard similarity against existing entity titles/summaries
- `doc_extraction_guide` — combines structural analysis + classification + entity refs into guidance

Depends on Phase 2a entity store and document intelligence. Independent of Layers 2–6 (can be built in parallel with assembly).

### 6.8 Layer 8: Batch import

Implement:
- directory scanning with glob matching
- document type inference from path conventions
- idempotent submission (skip existing records)
- error collection without abort
- `batch_import_documents` MCP tool
- `kbz import` CLI command

Depends on Layer 1 (identity resolution) and Phase 2a document management. Independent of knowledge/profile/assembly layers.

### 6.9 Layer 9: Health checks, CLI, and integration testing

Implement:
- health checks for knowledge entry schema and confidence consistency
- health checks for profile inheritance resolution
- health checks for knowledge scope validity
- CLI commands: `kbz knowledge`, `kbz profile`, `kbz context`
- integration tests across the full MCP tool surface
- bootstrap usage on this project (import docs, create profiles)

Final layer. Depends on everything above.

---

## 7. Work Breakdown

### 7.1 Track A — User identity resolution

Goal:
Auto-resolve `created_by` from local config or git config when not explicitly provided.

Tasks:
- define `.kbz/local.yaml` schema (user.name)
- implement local config parser (read `.kbz/local.yaml`, extract user.name)
- implement git config fallback (shell out to `git config user.name`)
- implement resolution chain: explicit → local config → git config → error
- add `.kbz/local.yaml` to `.gitignore`
- update all MCP create/submit tool handlers to use resolution when `created_by` is empty
- update CLI create/submit commands to use resolution when `--created_by` is not provided
- update `approved_by` on `doc_record_approve` to use the same resolution chain

Outputs:
- `internal/config/user.go` (or similar) — identity resolution module
- updated MCP tool handlers
- updated CLI handlers
- tests for each resolution step and the chain as a whole

### 7.2 Track B — KnowledgeEntry entity model and storage

Goal:
Define the KnowledgeEntry entity type with storage, serialisation, and basic CRUD.

Tasks:
- define KnowledgeEntry struct with all fields from spec §6.3
- implement deterministic YAML serialisation with canonical field order (spec §6.4)
- implement storage in `.kbz/state/knowledge/`, one file per entry
- implement TSID13 ID generation with `KE-` prefix
- implement `knowledge_contribute` MCP tool (create, without dedup — added in Track C)
- implement `knowledge_get` MCP tool
- implement `knowledge_list` MCP tool with filtering (tier, scope, status, topic, tags, min_confidence)
- implement `knowledge_update` MCP tool (resets use_count, miss_count, confidence)
- implement lifecycle state machine (valid transitions per spec §7.2)
- implement transition enforcement (reject invalid transitions)

Outputs:
- `internal/model/knowledge.go` — KnowledgeEntry type definition
- `internal/storage/knowledge.go` — YAML read/write with canonical serialisation
- `internal/validate/knowledge.go` — lifecycle transitions and field validation
- `internal/mcp/knowledge_tools.go` — MCP tool handlers
- round-trip serialisation tests
- lifecycle transition tests (valid and invalid)

### 7.3 Track C — Knowledge contribution and deduplication

Goal:
Add deduplication checks to the contribution flow and implement knowledge management tools.

Tasks:
- implement topic normalisation (lowercase, hyphens, collapse, strip)
- implement Jaccard similarity over normalised word-sets with stop word removal
- implement exact topic match check (same topic + same scope → reject with pointer)
- implement near-duplicate check (Jaccard > 0.65 in same scope → reject with pointer)
- ensure rejection responses include conflicting entry's ID, topic, and content
- implement `knowledge_confirm` MCP tool
- implement `knowledge_flag` MCP tool (increment miss_count, recompute confidence, auto-retire at miss_count ≥ 2)
- implement `knowledge_retire` MCP tool (set deprecated_reason)
- implement `knowledge_promote` MCP tool (tier 3 → 2, set promoted_from)

Outputs:
- `internal/knowledge/dedup.go` — topic normalisation, Jaccard similarity, stop word list
- updated `knowledge_contribute` handler with dedup checks
- additional MCP tool handlers
- dedup tests (exact match, near-duplicate, below threshold, cross-scope independence)
- topic normalisation tests

### 7.4 Track D — Context profiles and inheritance

Goal:
Implement context profile storage, inheritance resolution, and validation.

Tasks:
- define ContextProfile struct with fields from spec §11.3
- implement profile YAML parsing from `.kbz/context/roles/`
- implement profile validation (required fields, id matches filename)
- implement inheritance chain walking (root ancestor → leaf)
- implement cycle detection (error on circular inherits references)
- implement missing reference detection (error on unresolvable inherits)
- implement leaf-level replace resolution (spec §11.6)
- implement `profile_get` MCP tool (resolved and raw modes)
- implement `profile_list` MCP tool
- create `.kbz/context/roles/` directory structure

Outputs:
- `internal/context/profile.go` — profile type, parsing, validation
- `internal/context/resolve.go` — inheritance resolution algorithm
- `internal/mcp/profile_tools.go` — MCP tool handlers
- inheritance tests (simple chain, deep chain, diamond avoidance, cycle detection, missing ref)
- resolution tests (scalar replace, list replace, absent field inheritance)

### 7.5 Track E — Usage reporting and feedback loop

Goal:
Implement usage reporting that updates knowledge entry confidence and triggers lifecycle side effects.

Tasks:
- implement Wilson score lower bound formula
- implement confidence recomputation (called after any use_count or miss_count change)
- implement `context_report` MCP tool
- implement report processing: increment use_count for `used` entries, update last_used
- implement report processing: increment miss_count for `flagged` entries
- implement auto-confirmation side effect (contributed → confirmed when use_count ≥ 3, miss_count = 0)
- implement auto-retirement side effect (→ retired when miss_count ≥ 2, set deprecated_reason)

Outputs:
- `internal/knowledge/confidence.go` — Wilson score computation
- updated `context_report` handler with full processing pipeline
- confidence formula tests (known input/output pairs, edge cases: n=0, n=1, all successes, all failures)
- auto-transition tests (confirmation threshold, retirement threshold)

### 7.6 Track F — Context assembly

Goal:
Implement the context assembly algorithm that composes targeted context packets for agents.

Tasks:
- implement assembly source collection (resolved profile, design fragments, Tier 2 entries, Tier 3 entries, task instructions)
- implement scope filtering (entries matching role's scope or `project`)
- implement confidence filtering (Tier 2 ≥ 0.3, Tier 3 ≥ 0.5)
- implement byte-based budgeting (default 30,720 bytes, configurable via parameter and project config)
- implement priority trimming (Tier 3 lowest-confidence first → Tier 2 → design context; profile and task never trimmed)
- implement assembly without task_id (profile + knowledge only)
- implement assembly with task_id (adds design context from doc intelligence + task entity)
- implement `context_assemble` MCP tool
- implement source annotations in assembled output (which entry came from where)

Outputs:
- `internal/context/assemble.go` — assembly algorithm
- `internal/mcp/context_tools.go` — MCP tool handlers
- assembly tests (under budget, over budget with trimming, empty knowledge store, no profile, with/without task)
- priority ordering tests (verify trimming order is correct)
- byte budget tests (verify output does not exceed ceiling)

### 7.7 Track G — Agent capabilities

Goal:
Implement link resolution, duplicate detection, and document extraction guidance.

Tasks:
- implement entity reference pattern scanning (ID patterns, slugs, title fragments)
- implement candidate matching against known entities (exact, prefix, fuzzy)
- implement `suggest_links` MCP tool (returns matches with text span, entity ID, title, match quality)
- implement entity similarity check (Jaccard over normalised title + summary word-sets, threshold 0.5)
- implement `check_duplicates` MCP tool (returns candidates with IDs, titles, similarity scores)
- implement extraction guide assembly (structural outline + section roles + entity-relevant prompts + existing entity refs)
- implement `doc_extraction_guide` MCP tool

Outputs:
- `internal/knowledge/links.go` — entity reference scanning and matching
- `internal/knowledge/duplicates.go` — entity similarity checking
- `internal/mcp/agent_capability_tools.go` — MCP tool handlers
- link resolution tests (exact ID match, prefix match, slug match, no match)
- duplicate detection tests (high similarity, low similarity, cross-type independence)
- extraction guide tests (document with classifications, document without classifications)

### 7.8 Track H — Batch document import

Goal:
Implement batch import of existing documents for project onboarding.

Tasks:
- implement directory scanning with glob pattern matching
- implement document type inference from path conventions (spec §14.2, configurable via §5.4)
- implement idempotent submission (skip files that already have document records, matched by path)
- implement error collection without aborting the batch (individual file errors do not stop the import)
- implement summary response (imported count, skipped count with reasons, error count with details)
- implement `batch_import_documents` MCP tool
- implement `kbz import` CLI command

Outputs:
- `internal/service/import.go` — batch import logic
- `internal/mcp/import_tools.go` — MCP tool handler
- updated CLI with `import` subcommand
- import tests (happy path, idempotent re-run, type inference, missing type, error handling)

### 7.9 Track I — Health checks, CLI, and integration testing

Goal:
Extend health checks for Phase 2b state, add remaining CLI commands, and run integration tests.

Tasks:
- add health check: KnowledgeEntry schema validation (all files in knowledge/)
- add health check: confidence consistency (confidence matches Wilson score for stored counts)
- add health check: lifecycle state validity (no impossible states)
- add health check: knowledge scope validity (scope references existing profile or is "project")
- add health check: profile inheritance resolution (all inherits refs resolve, no cycles)
- add health check: profile schema validation (all files in context/roles/)
- add CLI: `kbz knowledge list`, `kbz knowledge get`
- add CLI: `kbz profile list`, `kbz profile get` (with `--raw` flag)
- add CLI: `kbz context assemble --role <name> [--task <id>] [--max-bytes <n>]`
- run integration tests against the full Phase 2b MCP tool surface
- bootstrap usage: import this project's documents, create initial profiles, verify assembly

Outputs:
- updated health check module
- updated CLI with knowledge, profile, context subcommands
- integration test suite
- initial profiles for this project (base.yaml, developer.yaml)

---

## 8. Parallelism Opportunities

Some tracks have no dependencies on each other and can be built concurrently:

```
Track A (identity)
  │
  ├─→ Track B (KE core) ──→ Track C (dedup) ──→ Track E (reporting) ──→ Track F (assembly)
  │
  ├─→ Track D (profiles) ──────────────────────────────────────────────→ Track F (assembly)
  │
  ├─→ Track G (agent capabilities)      [independent — needs only Phase 2a]
  │
  └─→ Track H (batch import)            [independent — needs only Phase 2a + Track A]

Track I (health + CLI + integration) ── after all others
```

Specifically:
- **Track D** (profiles) can run in parallel with **Tracks B + C** (knowledge core + dedup)
- **Track G** (agent capabilities) can run in parallel with everything after Track A
- **Track H** (batch import) can run in parallel with Tracks B–F
- **Track F** (assembly) must wait for both Track E (reporting, which gives it knowledge data) and Track D (profiles)

If using multiple agents in parallel, assign them to non-overlapping package directories:
- Agent 1: `internal/config/` (Track A), then `internal/knowledge/` (Tracks B, C, E)
- Agent 2: `internal/context/` (Track D), then Track F (assembly, also in `internal/context/`)
- Agent 3: `internal/mcp/agent_capability_tools.go` (Track G), `internal/service/import.go` (Track H)

---

## 9. Implementation Constraints

### 9.1 No cache-first implementation

Build on YAML file scanning first. Add SQLite cache tables for knowledge queries only if profiling shows latency > 100ms for assembly at realistic scales. The cache is derived — adding it later is non-breaking.

### 9.2 No automated pruning

Per P2-DEC-001, TTL-based pruning is Phase 3. The `ttl_days` field is stored but no background process or scheduled task acts on it. Retirement is manual or triggered by miss_count thresholds.

### 9.3 No git anchoring

Per P2-DEC-001, git-anchor-based staleness detection is Phase 3. The `git_anchors` field is stored but no file-change monitoring acts on it.

### 9.4 No post-merge compaction

Per P2-DEC-001, post-merge compaction is Phase 3. Duplicate knowledge entries from parallel branches are handled manually.

### 9.5 No deep merge in profile inheritance

Per P2-DEC-002, leaf-level replace only. Do not implement list concatenation, map merging, or override markers. If a child defines a field, it replaces the parent's value completely.

### 9.6 No automatic entity extraction

Per P1-DEC-019, entity extraction is agent-driven. The `doc_extraction_guide` tool provides guidance — it does not create entities.

### 9.7 MCP tools are built with their feature

Each track delivers its own MCP tools. Do not defer MCP integration to a separate pass. A track is not complete until its tools are registered, tested, and documented.

---

## 10. Testing and Verification Plan

### 10.1 Core verification categories

| Category | What is verified |
|----------|-----------------|
| Schema | KnowledgeEntry and profile field validation, required fields, type constraints |
| Serialisation | Deterministic YAML output, round-trip fidelity (write → read → write → compare) |
| Lifecycle | State transitions (valid accepted, invalid rejected), auto-transitions |
| Confidence | Wilson formula correctness, recomputation triggers, tier filtering thresholds |
| Deduplication | Exact topic match, Jaccard threshold, cross-scope independence, rejection response content |
| Inheritance | Chain resolution, cycle detection, missing reference, leaf-level replace behaviour |
| Assembly | Source composition, byte budgeting, priority trimming, confidence filtering |
| Identity | Resolution chain (explicit → local → git → error), fallback behaviour |
| Import | Glob matching, type inference, idempotency, error isolation |
| Health | Detection of invalid state across all new entity types |

### 10.2 Test types

- **Unit tests** for all new packages. Table-driven where multiple cases exist.
- **Round-trip tests** for KnowledgeEntry and profile YAML serialisation.
- **Integration tests** for MCP tool handlers (create → query → mutate → verify).
- **Edge case tests** for Wilson formula (n=0, n=1, all successes, all failures, large n).
- **Boundary tests** for byte budgeting (exactly at ceiling, one byte over, empty store).

### 10.3 Fixture strategy

Test fixtures live in `testdata/` directories alongside test files. Use `t.TempDir()` for filesystem tests. Fixture knowledge entries and profiles are minimal — only the fields needed for the test.

### 10.4 Verification against acceptance criteria

Each acceptance criterion in spec §20 must be traceable to at least one test. The test function name should include a reference to the criterion section (e.g., `TestKnowledgeContribute_DeduplicateExactTopic` → §20.1).

---

## 11. Risks and Mitigations

### 11.1 Risk: Context assembly complexity

The assembly algorithm composes five data sources with budgeting, filtering, and trimming. The interaction between these concerns could produce subtle bugs (wrong trimming order, off-by-one in budget, missing source).

Mitigation: Build assembly incrementally. Start with profile-only assembly (trivial). Add knowledge entries. Add design context. Add budgeting last. Test at each step.

### 11.2 Risk: Deduplication false positives

Jaccard similarity at 0.65 may be too aggressive or too permissive depending on knowledge entry length and vocabulary. Short entries with common words may trigger false positives.

Mitigation: The threshold is a constant, easy to tune. Log rejected contributions with their similarity scores during early usage. Adjust if the false positive rate is high.

### 11.3 Risk: Design context integration

Context assembly pulls design fragments from the Phase 2a document intelligence layer. If the doc intelligence index is sparse (few documents classified), assembly produces thin design context.

Mitigation: Batch import (Track H) followed by agent-driven classification fills the index. Assembly degrades gracefully — it returns what it has, annotated with source provenance. An empty design context section is not an error.

### 11.4 Risk: Profile authoring burden

If writing profiles feels like overhead, adoption suffers. Profiles need to be valuable from the first file.

Mitigation: A single `base.yaml` profile with 5–10 convention strings is enough to start. The system works without any profiles (graceful degradation). The self-hosting bootstrap (Track I) validates that profile authoring is practical.

### 11.5 Risk: Scope creep into Phase 3

The knowledge lifecycle has a clear Phase 2b/Phase 3 boundary (P2-DEC-001), but implementation pressure may push toward adding "just a little" TTL pruning or compaction.

Mitigation: The implementation constraints (§9) are explicit. TTL, git anchoring, and compaction are named as out of scope. If an implementation need arises, surface it — do not add it speculatively.

---

## 12. Definition of Done for Phase 2b

Phase 2b is complete when:

1. all acceptance criteria in spec §20 are met (50+ checkpoints across 11 categories)
2. all MCP tools from spec §17 are implemented and tested
3. all CLI commands from spec §19 are implemented
4. all health checks from spec §18 are implemented
5. all tests pass with the race detector enabled (`go test -race ./...`)
6. the system has been used to import this project's documents and create initial profiles (self-hosting validation)
7. deterministic YAML serialisation passes round-trip tests for all new file types
8. no known bugs in contributed code (bugs found during implementation are fixed or tracked)

---

## 13. Post-Implementation Audit Remediation

A comprehensive audit of the Phase 2b implementation was conducted after all 9 tracks were complete. The audit verified spec compliance, code quality, test coverage, and documentation accuracy. All core functionality is implemented and working — these items are fixes and improvements identified during the review.

Items are prioritised as **must fix** (blocks "done" per §12), **should fix** (spec compliance or quality gap), or **nice to have** (improvement, not blocking).

### 13.1 Must fix

**R1 — `knowledge_contribute` missing identity auto-resolution**

Spec §15.3 lists `knowledge_contribute` as a tool where `created_by` auto-resolution applies. The implementation passes the raw value without calling `config.ResolveIdentity()`. Every other creation tool (`create_plan`, `create_feature`, `doc_record_submit`, `batch_import_documents`, etc.) correctly calls `ResolveIdentity`.

- File: `internal/mcp/knowledge_tools.go` (line ~55)
- Fix: call `config.ResolveIdentity(createdByRaw)` before passing to `ContributeInput`
- Spec: §15.3, §17.1
- Track: C (contribution)

**R2 — Phase 2b health checks not wired into `kbz health` CLI**

`CheckKnowledgeHealth` and `CheckProfileHealth` are wired into the MCP `health_check` tool (via `server.go`) but are not called by the CLI `kbz health` command. Spec §18.2 says health checks must be available via "the `health` MCP tool or `kbz health` CLI command."

- File: `cmd/kanbanzai/main.go` (`runHealth` function, line ~606)
- Fix: instantiate `KnowledgeService` and `ProfileStore`, call both Phase 2b health checkers, merge reports
- Spec: §18.2
- Track: I (health/CLI)

**R3 — Documentation not updated for Phase 2b**

All project documentation still describes the project as "Phase 2a complete" with Phase 2b as future work. Critical documents affected:

- `AGENTS.md` — project status, scope guard (lists Phase 2b features as forbidden), repository structure (missing `knowledge/` and `context/` packages), no Phase 2b spec reference
- `README.md` — no Phase 2b mention, missing `KnowledgeEntry` from entity model, no Phase 2b tools listed
- `bootstrap-workflow.md` — doesn't mention Phase 2b tools are available
- `machine-context-design.md` — status still says "draft design", phasing section is stale

Additionally, `work/plan/phase-2b-progress.md` does not exist and should be created to track completion status, following the pattern established by `phase-2a-progress.md`.

- Fix: update all four documents; create `phase-2b-progress.md`
- Spec: §12 (definition of done point 8 — no known bugs)
- Track: I (integration)

### 13.2 Should fix

**R4 — Nil-check `knowledgeSvc` in `Assemble()`**

`context.Assemble()` nil-checks `entitySvc` and `intelligenceSvc` but calls `knowledgeSvc.List()` unconditionally. If `knowledgeSvc` is nil, this panics. Inconsistent with the graceful degradation principle (spec §4.4).

- File: `internal/context/assemble.go` (line ~113)
- Fix: add nil guard on `knowledgeSvc`, skip knowledge entries if nil
- Spec: §4.4 (graceful degradation)
- Track: F (assembly)

**R5 — Profile ID regex accepts 1-character IDs**

Spec §11.3 says profile IDs must be 2–30 characters. The regex second alternative `^[a-z0-9]{1,2}$` matches single-character IDs like `"a"`. The test explicitly asserts this is valid — both must be corrected.

- File: `internal/context/profile.go` (line ~16), `internal/context/profile_test.go`
- Fix: change `{1,2}` to `{2}` in the regex; update the test to expect `{"a", false}`
- Spec: §11.3
- Track: D (profiles)

**R6 — Add tests for `links.go` and `duplicates.go`**

`ScanEntityRefs`, `EntityTypeFromID` (link resolution), and `FindDuplicateCandidates` (duplicate detection) have zero unit tests. These are the algorithmic cores of two of the three agent capabilities.

- Files: `internal/knowledge/links.go`, `internal/knowledge/duplicates.go`
- Fix: add test files with cases for: no refs, single ref, multiple ref types, dedup, Plan ID matching, `KE-` prefix, empty input, threshold boundary, result ordering
- Spec: §20.7 (agent capabilities acceptance criteria)
- Track: G (agent capabilities)

**R7 — Add `glob` parameter to `batch_import_documents` and `kbz import`**

Spec §14.1 and §17.5 define an optional glob pattern parameter for filtering which files to import. Neither the MCP tool nor the CLI command implements it — all `.md` files in the directory are imported unconditionally.

- Files: `internal/mcp/import_tools.go`, `internal/service/import.go`, `cmd/kanbanzai/main.go`
- Fix: add `glob` parameter to MCP tool and `--glob` flag to CLI; filter matched files before import
- Spec: §14.1, §17.5, §19.1
- Track: H (batch import)

### 13.3 Nice to have

**R8 — Make `reason` required on `knowledge_flag` and `knowledge_retire`**

Spec §17.1 places `reason` in the required parameters column for both tools. The implementation treats it as optional via `GetString("reason", "")`.

- File: `internal/mcp/knowledge_tools.go` (lines ~218, ~243)
- Fix: use `RequireString("reason")` instead of `GetString`
- Spec: §17.1
- Track: B (knowledge core)

**R9 — System-generated `deprecated_reason` on auto-retirement**

Spec §7.3 says auto-retirement must set `deprecated_reason` to a system-generated message. The implementation uses the caller's flag reason (which may be empty).

- Files: `internal/service/knowledge.go` (`Flag` method line ~223, `ContextReport` method line ~279)
- Fix: always set a system message like `"auto-retired: miss_count reached 2"`, optionally appending the flag reason
- Spec: §7.3
- Track: E (usage reporting)

**R10 — Design context priority label is misleading**

Design context items are assigned `Priority: "low"` despite being the highest-priority trimmable source (trimmed last, after T3 and T2). The label should reflect the actual trimming semantics.

- File: `internal/context/assemble.go` (line ~106)
- Fix: change `Priority: "low"` to `Priority: "normal"` or `"high"` for design items
- Spec: §12.3
- Track: F (assembly)

**R11 — Make `used` required on `context_report`**

Spec §17.3 places `used` in the required parameters column. The implementation treats it as optional. A report with no `used` and no `flagged` is a no-op.

- File: `internal/mcp/knowledge_tools.go` (line ~311)
- Fix: validate that `used` is non-empty, or document the no-op case as intentional
- Spec: §17.3
- Track: E (usage reporting)

---

## 14. Summary

Phase 2b implementation is organised into 9 tracks (A–I), sequenced by dependency:

| Track | Name | Depends on | Key outputs |
|-------|------|-----------|-------------|
| A | User identity resolution | — | `internal/config/user.go`, updated tool handlers |
| B | KnowledgeEntry core | A | entity type, storage, CRUD, lifecycle |
| C | Contribution and dedup | B | topic normalisation, Jaccard similarity, dedup flow |
| D | Context profiles | — | profile parsing, inheritance resolution, validation |
| E | Usage reporting | B, C | `context_report`, confidence updates, auto-transitions |
| F | Context assembly | D, E | assembly algorithm, budgeting, `context_assemble` |
| G | Agent capabilities | Phase 2a | link resolution, duplicate detection, extraction guide |
| H | Batch import | A, Phase 2a | directory scanning, type inference, `kbz import` |
| I | Health + CLI + integration | all | health checks, CLI commands, integration tests, bootstrap |

Tracks D, G, and H can run in parallel with the B → C → E chain. Track F is the convergence point. Track I is the final verification pass.

The plan resolves all five open implementation questions from spec §21 and defines explicit constraints against Phase 3 scope creep. Each track follows the same discipline: implement the feature, build its MCP tools, write its tests, move on.