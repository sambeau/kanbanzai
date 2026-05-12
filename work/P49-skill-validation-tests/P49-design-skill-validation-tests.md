| Field  | Value                                                          |
|--------|----------------------------------------------------------------|
| Date   | 2026-05-12                                                     |
| Status | Draft                                                          |
| Author | sambeau (architect role, write-design skill)                   |
| Plan   | P49 — Agent File Structural Verification                       |

## Related Work

- `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` (approved) — P59 addresses registry drift via CI guard (`make registry-check`), corpus de-duplication, and a single-source-of-truth generator. P59 B3 (Unify) and B4 (Tidy) overlap with P49's cross-reference-integrity and mirror-consistency goals. P49 must not duplicate what P59 already covers.
- `work/P66-test-governance/P66-design-test-governance.md` (approved) — P66 establishes the committed-YAML-record pattern for test health. P49 can adopt this pattern for a machine-readable structural-health record of the agent-file corpus.
- `work/P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md` (superseded) — P44's `dispatch_task`, when built, would enable outcome-based testing of skills (dispatch an agent with a skill, check output quality). This is a future capability: P49's current scope is structural verification; P44 unlocks behavioral validation as a follow-on.
- `work/P63-test-remediation/P63-design-test-remediation.md` (approved) — P63 established the project's test culture: all tests must pass, failing tests block new work. P49 inherits this standard.
- `internal/kbzinit/skills_consistency_test.go` (existing) — Already verifies that embedded skill seeds match `.agents/skills/` and `.kbz/skills/` counterparts, and that embedded roles match `.kbz/roles/` counterparts. P49 should not duplicate these; it should verify the upstream canonical files that the seeds are derived from.

## Problem and Motivation

### Original problem (May 3, 2026)

When P49 was created, the 46 files that control agent behavior (skills, roles, stage bindings, agent instructions) had **zero test coverage**. The claim was that ~200 lines of Go would have caught every regression in the April 29 plan/batch migration — regressions that manifested as missing fields, broken cross-references, and stale mirrors.

### What changed

Since P49 was created, the agent-file corpus has undergone massive rework:

1. **The corpus grew** — from ~46 files to ~61 files (26 skills, 22 roles, 6 agent skills, 5 GitHub skills, stage-bindings, copilot-instructions, CONVENTIONS.md).
2. **CONVENTIONS.md was introduced** — a formal skill authoring standard defining required frontmatter (6 fields), mandatory section ordering (8 sections), anti-pattern format (Detect/BECAUSE/Resolve), evaluation criteria format, and a 500-line budget.
3. **P59 shipped** — registry drift checks, de-duplication, and CI guard for registry mirrors now exist. This addresses some of the original P49 concerns (mirror consistency between copilot-instructions.md and stage-bindings.yaml).
4. **P42 validation skills were created** — `validate-spec`, `validate-plan`, and `validate-review` evaluate document quality at stage gates. These are behavioral validators, not structural verifiers.
5. **P66 test governance was designed** — establishes a committed-YAML-record pattern for machine-readable test health that P49 can adopt.

### What remains untested

Despite P59 and the existing consistency tests, several structural integrity concerns have **no test coverage**:

| Gap | Description | Example failure mode |
|-----|-------------|---------------------|
| **Cross-reference integrity** | Every role referenced in `stage-bindings.yaml` exists as a `.kbz/roles/*.yaml` file; every skill referenced exists as a `.kbz/skills/*/SKILL.md`; vice versa (no orphaned files that nothing references) | A skill is renamed but the stage-binding reference isn't updated; the skill silently stops being loaded |
| **Frontmatter validity** | Every SKILL.md has all 6 required frontmatter fields (`name`, `description.expert`, `description.natural`, `triggers`, `roles`, `stage`, `constraint_level`) with valid values | A `constraint_level` typo causes the pipeline to select the wrong procedure style |
| **CONVENTIONS.md compliance** | Section ordering matches spec, line count ≤ 500, anti-patterns have all three required sub-fields, vocabulary terms follow the 15-year-practitioner test | A section is reordered during an edit, breaking the U-curve attention model |
| **Role YAML validity** | Every role YAML has required top-level keys (`id`, `inherits`, `identity`, `vocabulary`, `anti_patterns`, `tools`); every `inherits` target exists | An inherits chain breaks silently; the role degrades to base |
| **Stage-binding completeness** | Every stage has all required fields (`description`, `orchestration`, `roles`, `skills`, `human_gate`); every sub-agent role/skill exists | A stage is added to `stage-bindings.yaml` referencing a skill that hasn't been written yet |
| **copilot-instructions.md mirror freshness** | The generated registry tables in `copilot-instructions.md` match the canonical source (already partially covered by P59's CI guard, but P59 only covers the table sections — not the narrative prose rules) | A critical rule is updated in a skill but the copilot-instructions summary diverges |

### Why this still matters

The April 29 migration regression was real. The P50/P55/P56/P57 evidence trail documents compounding incident cost from agents operating on rules that didn't reach their context. Structural integrity tests are the **cheapest layer of defense** — they catch regressions deterministically, in milliseconds, before they reach an agent's context window.

P59's registry drift checks catch *one class* of these problems (generated table divergence). The existing consistency tests catch *another* (embedded seed vs. canonical file). The gaps listed above remain unguarded.

## Design

### Rename: from "skill-validation-tests" to "agent-file-structural-verification"

The original name is misleading. In Kanbanzai's vocabulary, "validation" refers to quality evaluation against criteria (as done by `validate-spec`, `validate-plan`, `validate-review`). What P49 proposes is **verification**: deterministic structural checks against a known schema. The plan should be renamed to **"Agent File Structural Verification"** or **"skill-structural-tests"** to avoid confusion.

### Architecture: static analysis, not runtime tests

All checks are **static analysis** — they read files from disk and assert structural properties. No MCP server, no LLM evaluation, no agent dispatch. This keeps the design deterministic, fast (milliseconds in CI), and independent of P44.

The test suite lives in a new package: `internal/verify/agentfiles/` (or `internal/agentfiles/`). It reads the canonical files from the repository root at test time.

### Component: Structural health record

Adopting P66's pattern, the test suite can optionally produce a machine-readable YAML record committed to the repository:

```yaml
# .kbz/state/agent-files-health.yaml
last_check: "2026-05-12T20:36:00Z"
result: pass
violations: []  # empty on pass
summary: ""
```

This gives agents and humans a single point of truth for "are our agent control files structurally sound?" — without needing to run the test suite. The CI run updates this record; the record is consulted by `status()` and session-start checklists (analogous to P66's `test(action: "verify")`).

### Checks by category

#### C1: Cross-reference integrity

| Check | What it verifies |
|-------|-----------------|
| C1.1 | Every role in `stage-bindings.yaml` → exists as `.kbz/roles/<name>.yaml` |
| C1.2 | Every role that inherits → its `inherits` target exists |
| C1.3 | Every skill in `stage-bindings.yaml` → exists as `.kbz/skills/<name>/SKILL.md` |
| C1.4 | Every `.kbz/roles/*.yaml` → referenced in at least one stage binding (no orphans) |
| C1.5 | Every `.kbz/skills/*/SKILL.md` → referenced in at least one stage binding (no orphans) |
| C1.6 | Every sub-agent role/skill in stage bindings → exists (C1.1 + C1.3 already cover, but sub-agent references deserve explicit checking because they're easy to miss) |
| C1.7 | `.agents/skills/kanbanzai-*/SKILL.md` → has a corresponding embedded seed in `internal/kbzinit/skills/` (inverse of the existing consistency test) |

#### C2: SKILL.md frontmatter validity

| Check | What it verifies |
|-------|-----------------|
| C2.1 | Every SKILL.md has all 6 required frontmatter fields |
| C2.2 | `constraint_level` is one of: `low`, `medium`, `high` |
| C2.3 | `triggers` has at least 2 entries |
| C2.4 | `stage` value matches a key in `stage-bindings.yaml` |
| C2.5 | `roles` values are all valid role IDs that exist |
| C2.6 | `name` matches the directory name (e.g., `write-spec/SKILL.md` must have `name: write-spec`) |

#### C3: CONVENTIONS.md section-ordering compliance

| Check | What it verifies |
|-------|-----------------|
| C3.1 | Sections appear in the mandatory order: Vocabulary → Anti-Patterns → (Checklist) → Procedure → Output Format → Examples → Evaluation Criteria → Questions This Skill Answers |
| C3.2 | Line count ≤ 500 (warning, not failure — flag for human review) |
| C3.3 | Every anti-pattern has `Detect:`, `BECAUSE:`, and `Resolve:` sub-fields |
| C3.4 | Vocabulary section has 15–30 terms (advisory; flag below 10 or above 35) |
| C3.5 | Evaluation criteria section has 4–8 criteria with at least one `required` weight |
| C3.6 | Questions section has 5–10 entries |

#### C4: Role YAML structural validity

| Check | What it verifies |
|-------|-----------------|
| C4.1 | Every role YAML has required top-level keys: `id`, `inherits`, `identity`, `vocabulary`, `anti_patterns` |
| C4.2 | `id` matches the filename |
| C4.3 | `anti_patterns` entries have `name`, `detect`, `because`, `resolve` sub-fields |
| C4.4 | `tools` list contains only known MCP tool names (advisory warning on unknown names) |

#### C5: Stage-binding structural validity

| Check | What it verifies |
|-------|-----------------|
| C5.1 | `schema_version` is present and ≥ 2 |
| C5.2 | Every stage binding has `description`, `orchestration`, `roles`, `skills`, `human_gate` |
| C5.3 | `document_type` values (where present) are valid document types |
| C5.4 | `sub_agents.roles` and `sub_agents.skills` lists are non-empty when `orchestration` is `orchestrator-workers` or `pipeline-coordinator` |
| C5.5 | `prerequisites.documents` types are valid document types |
| C5.6 | `effort_budget` is present on every stage (P59 B1 depends on this for constraint card rendering) |

### What's explicitly out of scope

1. **Behavioral/outcome testing** — P49 does not dispatch agents and evaluate their output. That requires P44's `dispatch_task` and is a fundamentally different (and larger) project. When P44 ships, a follow-on plan can add outcome tests.
2. **Content quality evaluation** — P49 does not judge whether a skill's anti-patterns are correct, whether its vocabulary is complete, or whether its procedure produces good results. Those are the domain of the existing validation skills (`validate-spec`, etc.) and of human review.
3. **Duplicate of P59** — P49 does not regenerate registry tables or check generated-file drift. P59 already owns that. P49 checks the *canonical source files* that P59's generator consumes.
4. **`.github/skills/` codebase-memory skills** — These follow a different format (they're not Kanbanzai SKILL.md files with YAML frontmatter) and are outside the agent-file corpus scope. They can be added later if needed.
5. **`AGENTS.md` narrative consistency** — P59 D7 explicitly deferred AGENTS.md from registry generation. P49 similarly scopes out AGENTS.md content checks.

### Relationship to P44

P44, when built, enables a fundamentally different class of test: **outcome verification**. You could dispatch an agent with a specific skill, give it a controlled task, and evaluate whether its output meets the skill's evaluation criteria.

This is complementary to P49's structural checks, not a replacement:

| Layer | What | When | Cost |
|-------|------|------|------|
| **P49 (structural)** | Is the SKILL.md well-formed? Does every referenced role exist? | Every CI run | Milliseconds, deterministic |
| **P44-enabled (behavioral)** | Does an agent using this skill produce correct output? | Nightly / on-demand | Minutes, probabilistic |

The structural layer catches regressions that would silently degrade agent behavior (missing sections, broken cross-references, malformed frontmatter). The behavioral layer catches regressions where the structure is correct but the content produces wrong results.

P49 should ship first — it's cheaper, deterministic, and catches the class of regression that the April 29 migration demonstrated. P44-enabled outcome tests should follow as a separate plan.

### Implementation approach

A new Go test package at `internal/verify/agentfiles/` containing:

```
internal/verify/agentfiles/
├── cross_reference_test.go    # C1 checks
├── frontmatter_test.go         # C2 checks
├── conventions_test.go         # C3 checks
├── role_yaml_test.go           # C4 checks
├── stage_bindings_test.go      # C5 checks
├── health_record.go            # Optional: YAML record read/write
└── testdata/                   # Test fixtures if needed
```

Tests follow the standard `go test` pattern. They resolve the repository root from the test file's location (same pattern used by existing `skills_consistency_test.go`). They read canonical files directly — no MCP tools, no server startup.

Estimated size: ~300–400 lines of Go across the five test files (revised up from the original ~200 estimate to account for the larger corpus and CONVENTIONS.md checks).

## Alternatives Considered

### A1. Do nothing (status quo)

- **Trade-offs:** Zero engineering cost. P59 covers registry drift; existing consistency tests cover embedded seeds.
- **Why rejected:** The gaps in C1–C5 are real and unguarded. A malformed frontmatter or a broken cross-reference would silently degrade agent behavior with no CI signal. The April 29 regression class is still possible for categories P59 doesn't cover.

### A2. Expand P59's CI guard instead of creating a separate plan

- **Trade-offs:** Single CI target, simpler mental model.
- **Why rejected:** P59's scope is registry drift and de-duplication — it's about the *distribution* of rules, not the *structural integrity* of the canonical files. Mixing the two would complicate both. P59 checks "do the generated copies match the source?"; P49 checks "is the source itself well-formed?" These are different concerns with different failure modes.

### A3. Wait for P44 and build outcome tests instead

- **Trade-offs:** Skip structural tests; go straight to behavioral testing.
- **Why rejected:** Outcome tests are expensive, probabilistic, and depend on P44 shipping. Structural tests are cheap, deterministic, and can ship today. The two are complementary, not alternatives. Building structural tests first doesn't preclude adding outcome tests later — it provides the safety net while waiting for P44.

### A4. Make this a linter/CLI tool instead of a test suite

- **Trade-offs:** A `kanbanzai verify skills` command could be run ad-hoc. More flexible than CI-only tests.
- **Why deferred (not rejected):** A CLI tool is a natural evolution, but the immediate need is CI enforcement. Starting as Go tests keeps the implementation minimal. A CLI wrapper can be added later if the checks prove useful outside CI.

### A5. Use the P66 test-governance pattern (committed YAML health record)

- **Trade-offs:** Adds a committed file that must be updated after every relevant change. More ceremony than pure CI tests.
- **Why selected for optional inclusion:** The health record gives agents visibility into structural soundness without running tests. Adopting P66's pattern keeps consistency across the project. The record is optional — CI tests are the enforcement mechanism; the record is the visibility mechanism.

## Decisions

### D1. The plan is renamed to "Agent File Structural Verification."

- **Decision.** P49-skill-validation-tests → P49-agent-file-structural-verification.
- **Context.** "Validation" in Kanbanzai means quality evaluation (validate-spec, validate-plan, validate-review). This plan does structural verification — deterministic checks against a schema.
- **Rationale.** Using consistent vocabulary prevents confusion between this plan and the P42 validation pipeline.
- **Consequences.** The plan slug changes. Existing references (if any) need updating.

### D2. All checks are static analysis — no MCP server, no LLM evaluation.

- **Decision.** Tests read files from disk and assert structural properties using only the Go standard library and `gopkg.in/yaml.v3`.
- **Context.** Behavioral/outcome testing requires P44 and is a different project.
- **Rationale.** Static analysis is deterministic, fast, and has no dependencies on running infrastructure. It can run in CI as part of `go test ./...`.
- **Consequences.** The implementation is ~300–400 lines of Go. No new dependencies.

### D3. CONVENTIONS.md is treated as the authoritative schema for SKILL.md files.

- **Decision.** C3 checks validate against the CONVENTIONS.md spec (section ordering, anti-pattern format, line budget, vocabulary count).
- **Context.** CONVENTIONS.md was introduced after P49's creation and formalizes what was previously implicit.
- **Rationale.** CONVENTIONS.md is already the documented standard that skill authors follow. Tests should enforce the same standard.
- **Consequences.** Changes to CONVENTIONS.md may require test updates. This is desirable — it means the tests track the evolving standard.

### D4. C3.2 (line budget) and C3.4 (vocabulary count) are warnings, not failures.

- **Decision.** Exceeding the 500-line budget or having too few/many vocabulary terms produces a test warning (via `t.Log` or a soft-check flag), not a hard failure.
- **Context.** Hard-failing on line count would block merges for legitimate cases where a skill temporarily exceeds budget. The convention is a target, not a hard invariant.
- **Rationale.** Warnings surface the issue without blocking progress. The health record can distinguish warnings from failures.
- **Consequences.** CI stays green on line-budget violations; humans are nudged to address them.

### D5. The optional health record follows P66's YAML pattern.

- **Decision.** If implemented, `.kbz/state/agent-files-health.yaml` uses the same schema conventions as P66's `test-status.yaml`.
- **Context.** P66 established the committed-YAML-record pattern for machine-readable health.
- **Rationale.** Consistency across health records makes them easier for agents and humans to consume. A single pattern is easier to document.
- **Consequences.** The record is committed to git and updated by CI. It's consulted by `status()` and session-start checklists.

### D6. `.github/skills/` and `.agents/skills/` files are in scope only for cross-reference checks (C1.7), not for structural checks.

- **Decision.** C1.7 checks that `.agents/skills/` files have corresponding embedded seeds. C2–C5 checks apply only to `.kbz/skills/` and `.kbz/roles/`.
- **Context.** `.github/skills/` files follow a different format (codebase-memory skills, not Kanbanzai SKILL.md). `.agents/skills/` files are workflow guides, not task-execution skills.
- **Rationale.** Each skill family has different structural conventions. Mixing them would produce false positives.
- **Consequences.** The test suite has a clean boundary: `.kbz/` files get full structural checks; `.agents/` and `.github/` files get cross-reference-only checks.

## Dependencies

### Internal

- **`.kbz/stage-bindings.yaml`** — canonical source for role/skill references. Must be readable at test time.
- **`.kbz/roles/*.yaml`** — canonical role definitions. Tested for structural validity.
- **`.kbz/skills/*/SKILL.md`** — canonical skill definitions. Tested for frontmatter and section compliance.
- **`.kbz/skills/CONVENTIONS.md`** — schema for C3 checks.
- **`gopkg.in/yaml.v3`** — already in the project's dependency tree.

### External

- None. The test suite has no external dependencies.

### Sequencing

P49 is independent of all other in-flight plans. It can ship at any time. The only coordination needed:

- **P59:** Ensure P49's cross-reference checks don't duplicate P59's CI guard. P59 checks generated-file drift; P49 checks canonical-file integrity. No overlap.
- **P44:** When P44 ships, a follow-on plan can add outcome tests that consume P49's structural health record as a prerequisite ("don't bother running outcome tests if the skill files are structurally broken").

## Open Questions

1. **Should the health record be committed or generated?** P66's pattern commits the record. The alternative is to treat it as a build artifact. Committing gives git-tracked history of structural health; generating keeps the repo cleaner. Recommendation: commit (consistent with P66).
2. **Should C3.2 (line budget) fail CI or only warn?** Current design says warn-only. If the corpus stabilizes and all skills are under budget, this could be promoted to a hard failure. Revisit after P59 B4 (corpus cut) ships.
3. **Should the test suite be invoked from `make` or only from `go test`?** Both. Add a `make verify-agent-files` target that runs the specific test package. The existing `go test ./...` in CI will pick it up automatically.
4. **Should the `status()` MCP tool surface agent-file health?** If the health record is committed, `status()` can read and display it — analogous to P66's `test_health` block. This depends on the health record decision (Q1).
5. **What about `.github/copilot-instructions.md` narrative rule consistency?** The current design scopes this out (P59 already handles the registry table sections). Full narrative consistency checking is a natural language problem and belongs in a different plan (or in P59 if it proves needed).
