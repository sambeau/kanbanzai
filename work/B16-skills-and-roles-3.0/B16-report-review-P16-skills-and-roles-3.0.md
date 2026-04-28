# Review: P16 — Skills and Roles 3.0

| Field    | Value                                      |
|----------|--------------------------------------------|
| Plan     | P16-skills-and-roles-3.0                   |
| Reviewer | Claude Sonnet 4.6                          |
| Date     | 2026-04-02T02:04:53Z                       |
| Verdict  | Pass with findings                         |

---

## Summary

P16 delivers the evidence-based skills and roles system described in `work/design/skills-system-redesign-v2.md`: a YAML role schema with vocabulary, anti-patterns, and inheritance; a SKILL.md format with frontmatter and attention-curve section ordering; a binding registry mapping workflow stages to role+skill combinations; a 10-step context assembly pipeline; knowledge auto-surfacing; and freshness tracking. The core infrastructure — role parsing, inheritance resolution, skill validation, binding registry loading, freshness tracking, and pipeline structure — is correct, well-tested, and merged. The main findings are one confirmed bug, two design gaps, and a cluster of content and lifecycle hygiene issues. No blocking defects prevent shipping, but the bug fix, CLI compatibility fix, and spec registration should be addressed before advancing the plan.

---

## Feature Status

| Feature | Slug | Status | Spec Conformance |
|---------|------|--------|-----------------|
| FEAT-01KN5-88PCVN4Y | role-system | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PDBW85 | skill-system | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PDPE8V | binding-registry | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PE43M6 | context-assembly-pipeline | developing (all tasks done) | ⚠️ Bug + missing progressive loading (see findings) |
| FEAT-01KN5-88PEF817 | knowledge-auto-surfacing | developing (**0 tasks**) | ⚠️ Implementation exists; lifecycle gap (see findings) |
| FEAT-01KN5-88PETZQE | freshness-tracking | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PF5P5Y | base-and-authoring-role-content | developing (all tasks done) | ⚠️ Minor content issues (see findings) |
| FEAT-01KN5-88PFG6GY | review-role-content | developing (all tasks done) | ⚠️ Minor content issues (see findings) |
| FEAT-01KN5-88PFWADY | document-authoring-skill-content | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PG7HA3 | implementation-skill-content | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PGJNM0 | review-skill-content | developing (all tasks done) | ✅ All criteria met |
| FEAT-01KN5-88PGXATW | migration-and-backward-compat | developing (all tasks done) | ⚠️ CLI path not updated (see findings) |

All features are in `developing` — none have been transitioned despite all tasks being complete.

---

## Spec Conformance Detail

### Feature: role-system

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | YAML schema with vocabulary, anti_patterns, tools, identity, inherits fields | ✅ | All fields present in `role.go` |
| 2 | Strict YAML parsing (no unknown fields) | ✅ | `KnownFields(true)` enforced |
| 3 | Inheritance resolution merges vocabulary and anti-pattern lists | ✅ | Parent first, child appended; leaf identity wins |
| 4 | Cycle detection | ✅ | `visited` map in `role_resolve.go` |
| 5 | Storage in `.kbz/roles/` with fallback to `.kbz/context/roles/` | ✅ | `role_store.go` checks new root first |
| 6 | Validation: id matches filename, identity ≤50 tokens, vocabulary non-empty | ✅ | All enforced in `validateRole` |

### Feature: skill-system

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | Frontmatter schema: name, description.expert/natural, triggers, roles, stage, constraint_level | ✅ | `SkillFrontmatter` in `internal/skill/model.go` |
| 2 | Attention-curve section ordering enforced | ✅ | `canonicalSectionOrder` in `sections.go` |
| 3 | Required sections validated | ✅ | All 6 required sections checked |
| 4 | Checklist required for low/medium constraint_level | ✅ | FR-012 implemented correctly |
| 5 | 500-line limit enforced | ✅ | In `parse.go` |
| 6 | Directory structure `.kbz/skills/*/SKILL.md` with `references/` and `scripts/` | ✅ | `loader.go` discovers both subdirs |

### Feature: binding-registry

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | stage-bindings.yaml schema: all §3.3 fields present | ✅ | `binding/model.go` |
| 2 | Stage lookup API | ✅ | `registry.go` |
| 3 | Duplicate key detection on load | ✅ | Two-pass decode in `loader.go` |
| 4 | Consistency validation: orchestration/sub_agents pairing, etc. | ✅ | `validate.go` |
| 5 | All 8 stages present in `.kbz/stage-bindings.yaml` | ✅ | designing, specifying, dev-planning, developing, reviewing, plan-reviewing, researching, documenting |

### Feature: context-assembly-pipeline

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | 10-step assembly pipeline | ✅ | Steps 0–10 present in `pipeline.go` |
| 2 | Token budget warn at 40%, refuse at 60% | ✅ | Constants correct; refuse tested |
| 3 | Progressive layer loading (L1→L2→L3→L4) | ❌ | Layer constants defined but not used for incremental loading; single post-assembly gate only |
| 4 | Task identity section uses task ID in heading | ❌ | Bug: `identityContent` uses `summary` twice; `taskID` extracted but never written to heading |
| 5 | Tool subset soft filtering | ✅ | `renderToolGuidance` in pipeline |
| 6 | Handoff tool extended | ✅ | `handoff_tool.go` wired end-to-end with fallback to legacy assembly |

### Feature: knowledge-auto-surfacing

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | File path matching | ✅ | `matchesFilePath` in `surfacer.go` |
| 2 | Tag matching | ✅ | `matchesTags` |
| 3 | Always-entries (scope=project or tag=always) | ✅ | `matchesAlways` |
| 4 | Recency weighting | ✅ | Delegated to `knowledge.RankAndCap` |
| 5 | Cap at 10 entries | ✅ | `defaultMaxSurfacedEntries = 10` |
| 6 | Position 8 in assembled context | ✅ | `PositionKnowledge = 8` |
| 7 | Role tag matching uses domain tags (e.g., reviewer-security → "security") | ❌ | Pipeline passes role ID as tag; `Role` struct has no domain `tags` field |
| 8 | Health compaction recommendation after consecutive cap-hits | ✅ | `CapTracker` + `capSaturationHealthChecker` |

### Feature: freshness-tracking

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | `last_verified` field on Role and SkillFrontmatter | ✅ | Both structs |
| 2 | Health tool flags stale roles and skills | ✅ | `freshnessHealthChecker` |
| 3 | Configurable window, default 30 days | ✅ | `cfg.Freshness.StalenessWindowDays` with fallback |
| 4 | Stale content remains usable (warnings only) | ✅ | Only appends to `StalenessWarnings` |
| 5 | Warnings in assembled context metadata | ✅ | `PipelineResult.MetadataWarnings` |
| 6 | `RefreshRole`/`RefreshSkill` actions | ✅ | `refresh.go` |

### Feature: base-and-authoring-role-content

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | 8 role files authored (base, architect, spec-author, implementer, implementer-go, researcher, documenter, orchestrator) | ✅ | All present in `.kbz/roles/` |
| 2 | All files have vocabulary, anti-patterns with 4 fields, tools, identity | ✅ | Validated by role loader |
| 3 | `base.yaml` identity is a job title | ❌ | Set to product description: `"Kanbanzai — Git-native workflow system for human-AI development"` |

### Feature: review-role-content

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | 5 reviewer role files (reviewer, reviewer-conformance, reviewer-quality, reviewer-security, reviewer-testing) | ✅ | All present |
| 2 | Inheritance chain correct | ✅ | All inherit from `reviewer` |
| 3 | `reviewer-security.yaml` tools list minimal (jig pattern) | ❌ | Tools list is identical copy of parent — no additive value; contradicts DP-10 |

### Feature: document-authoring-skill-content

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | 5 SKILL.md files (write-design, write-spec, write-dev-plan, write-research, update-docs) | ✅ | All present |
| 2 | All sections in canonical order | ✅ | |
| 3 | All anti-patterns have Detect/BECAUSE/Resolve | ✅ | |
| 4 | `write-spec` constraint_level appropriate for complexity | ⚠️ | Uses `high` (no checklist) despite 7-step procedure with validation loop; `medium` may be more appropriate |

### Feature: implementation-skill-content / review-skill-content

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | SKILL.md files authored for all specified skills | ✅ | implement-task, orchestrate-development, decompose-feature, review-code, orchestrate-review, review-plan all present |
| 2 | `review-code/SKILL.md` quality | ✅ | Strongest file — 15 vocabulary terms, all 5 anti-patterns from design, weighted evaluation criteria |

### Feature: migration-and-backward-compat

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| 1 | New file locations (`.kbz/roles/`, `.kbz/skills/`) | ✅ | Both populated |
| 2 | Old `.skills/` files retired | ✅ | Intentional (FR-015); README migration map left |
| 3 | `.kbz/context/roles/` legacy fallback in role store | ✅ | `role_store.go` dual-path |
| 4 | MCP `profile` tool uses new `RoleStore` | ✅ | `server.go` wired correctly |
| 5 | CLI `kanbanzai profile` command uses new paths | ❌ | Still hardcoded to `context/roles/`; new roles not found via CLI |
| 6 | Health checkers validate new roles | ❌ | `profileHealthChecker` uses old `ProfileStore`; `.kbz/roles/` invisible to health |

---

## Documentation Currency

| Check | Result |
|-------|--------|
| AGENTS.md Scope Guard | N/A — plan not yet done |
| Spec documents registered under P16 | ❌ 0 of 12 registered (files exist at `work/spec/3.0-*.md`) |
| `.skills/` old files retired | ✅ Intentional FR-015 retirement; README migration map present |
| `stage-bindings.yaml` notes | ❌ 6 of 8 stages missing rationale in `notes` field |

---

## Cross-Cutting Checks

| Check | Result |
|-------|--------|
| `go test ./...` | ✅ Pass |
| `go test -race ./...` | ⚠️ Races in `internal/service` (pre-existing; not introduced by P16) |
| `health()` errors | ⚠️ 4 pre-existing slug-format errors on plan IDs; dependency cycle on TASK-01KN5AJKNBEE2 |
| `git status` clean | ❌ Uncommitted changes in `internal/mcp/` (decompose_tool.go, entity_tool.go, finish_tool.go — P17 work in progress) |

The race conditions in `internal/service` involve the worktree hook and entity service — not the packages introduced by P16. They are pre-existing and tracked separately. The uncommitted MCP changes are P17 error-message improvements and should be committed or stashed.

---

## Findings

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| 1 | Significant | `internal/context/pipeline.go` stepBuildIdentity | **Bug:** `identityContent` formats `## Task: <summary>` using `summary` as both heading label and body. `taskID` is extracted but never used — heading should be `## Task: TASK-01JX...` |
| 2 | Significant | `internal/context/pipeline.go` stepSurfaceKnowledge | **Design gap:** Role tag matching passes role ID (e.g. `"reviewer-security"`) as the only tag. `Role` struct has no `Tags []string` field, so domain-tag routing (e.g., `reviewer-security` → `"security"`) is impossible. `SurfaceInput.SkillName` is also a dead field — populated but never consumed by the surfacer |
| 3 | Significant | `cmd/kanbanzai/main.go` runProfileList/runProfileGet | **CLI gap:** Profile CLI commands hardcoded to old `context/roles/` path; new roles at `.kbz/roles/` are not found. MCP `profile` tool is correctly wired. Health `profileHealthChecker` also uses old store, missing new roles in health validation |
| 4 | Minor | `internal/context/pipeline.go` | **Missing feature:** Progressive layer-based token budget management (§6.2: stop loading at each layer boundary) not implemented. Layer constants exist (`L1`–`L4`) but have no runtime effect. Single post-assembly refuse gate at 60% is the only enforcement |
| 5 | Minor | `.kbz/stage-bindings.yaml` | **Content gap:** `notes` field absent for 6 of 8 stages (designing, specifying, dev-planning, plan-reviewing, researching, documenting). Notes are the BECAUSE clauses explaining configuration choices — the primary rationale record for each stage's bindings |
| 6 | Minor | `.kbz/roles/base.yaml` | **Content issue:** `identity` field set to product description `"Kanbanzai — Git-native workflow system..."` rather than a job title (§3.1: "a real job title under 50 tokens"). Functionally harmless (leaf identity wins in resolution) but does not comply with the spec for the base role |
| 7 | Minor | `.kbz/roles/reviewer-security.yaml` | **Content issue:** `tools` list is an identical copy of `reviewer.yaml`'s tools list. After `mergeToolsUnion` the result equals the parent alone. Contradicts DP-10 (only add what the model doesn't know); field should be omitted or trimmed to security-specific tools |
| 8 | Minor | `internal/binding/validate.go` | **Validation hardness:** A binding referencing a non-existent role emits a warning, not an error. Typo'd role IDs silently load and fail at runtime when the agent attempts resolution. Should be a hard load error |
| 9 | Minor | `internal/skill/sections.go` | **Code quality:** (a) `!strings.HasPrefix(line, "### ")` guard in `parseSections` is dead code — any line matching `"## "` cannot match `"### "`. (b) Duplicate canonical section headings are not detected. (c) `constraint_level: high` means high autonomy (no checklist), which is counter-intuitive — content authors expecting "high" to mean "more constrained" will author incorrectly; deserves a comment |
| 10 | Minor | `.kbz/skills/write-spec/SKILL.md` | **Content question:** Uses `constraint_level: high` (no checklist) despite a 7-step procedure with a validate→fix→re-validate loop and external script invocation. `medium` (checklist required) may better reflect the actual constraint level |
| 11 | Informational | FEAT-01KN5-88PEF817 (knowledge-auto-surfacing) | **Lifecycle gap:** Feature has 0 tasks but implementation is complete (`surfacer.go`, `cap_tracker.go`, pipeline wiring). Work was done under other features' tasks. Needs retroactive tasks closed, or direct transition to `done` with a note |
| 12 | Informational | P16 document records | **Spec registration:** All 12 spec files exist at `work/spec/3.0-*.md` but none are registered as document records. The `developing → reviewing` gate checks for approved specs; without registration the gate cannot be automatically verified |

---

## Verdict

**Pass with findings.** The core infrastructure — role parsing, skill validation, binding registry, context assembly pipeline, freshness tracking — is correct and well-tested. All 12 features have their implementation merged. The plan can advance to `reviewing` once the following are addressed:

**Required before marking done:**
1. Fix the `taskID`/`summary` bug in `pipeline.go` (Finding #1 — clearly unintentional).
2. Update the CLI `profile` command to use the new `RoleStore` path (Finding #3).
3. Register the 12 spec documents so gate checks can function (Finding #12).
4. Resolve `knowledge-auto-surfacing` lifecycle gap — 0 tasks, full implementation (Finding #11).
5. Transition all 12 features from `developing` to `done`.

**Recommended before marking done:**
6. Fix `base.yaml` identity to be a job title (Finding #6) — one-line change.
7. Remove or trim `reviewer-security.yaml` duplicate tools (Finding #7) — one-line change.
8. Add `notes` to the 6 missing stages in `stage-bindings.yaml` (Finding #5).

**Deferred / follow-up:**
- Role domain tags (`Role.Tags []string`) and `SurfaceInput.SkillName` cleanup (Finding #2) — requires a design decision on how domain tags are expressed in the role schema.
- Progressive layer loading (Finding #4) — substantial feature; track as a follow-up if not targeted for 3.0.
- Binding validator hardness for missing roles (Finding #8).
- Race conditions in `internal/service` (pre-existing, not P16).