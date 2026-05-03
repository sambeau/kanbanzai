# Bootstrapping Gap Audit: Dogfooding Asymmetry in Kanbanzai

| Field  | Value             |
|--------|-------------------|
| Date   | 2025-07-17        |
| Status | Draft             |
| Author | System Architect (AI) |

## Executive Summary

This audit identified **9 gaps** across the Kanbanzai bootstrapping pipeline — **2 Critical (A), 4 High (B), 2 Medium (C), and 1 Low (D)**. The root cause is systematic: `kbz init` installs only `.agents/skills/kanbanzai-*` workflow skills (9 files), `base.yaml` + `reviewer.yaml` roles (2 files), config files, and editor integration — but it does **not** install `stage-bindings.yaml`, any task-execution skills under `.kbz/skills/` (20 skills), or the remaining 16 role files under `.kbz/roles/`. The result is that the 3.0 context assembly pipeline silently degrades to legacy mode on every consumer project because it requires `stage-bindings.yaml` to activate. Furthermore, the project's hand-maintained `copilot-instructions.md` and `AGENTS.md` contain an extensive table of 17 task-execution skills, 16 roles, and codebase-memory graph skills — none of which exist after a fresh install. The top recommendation is to embed and install all task-execution skills and `stage-bindings.yaml`, then embed the full role catalog.

## Methodology

Followed the 7-step procedure:

1. **Mapped the init pipeline.** Read `internal/kbzinit/init.go`, `skills.go`, `roles.go`, `agents_md.go`, `config_writer.go`, and `mcp_config.go` in full. Identified every artifact type `kbz init` installs: `.kbz/config.yaml`, `work/*/` directories, `.agents/skills/kanbanzai-*/` (9 workflow skills), `.kbz/context/roles/base.yaml` (scaffold), `.kbz/context/roles/reviewer.yaml` (managed), `.mcp.json`, `.zed/settings.json`, `AGENTS.md`, `.github/copilot-instructions.md`. Noted what it does NOT install: `stage-bindings.yaml`, `.kbz/skills/` task-execution skills, the remaining 16 roles, `.github/skills/` codebase-memory skills, `refs/`, `work/templates/`.

2. **Mapped runtime requirements.** Read `internal/context/pipeline.go`, `internal/skill/loader.go`, `internal/context/role_store.go`, `internal/context/role_resolve.go`, `internal/binding/loader.go`, and `internal/mcp/server.go`. The pipeline reads from: `.kbz/stage-bindings.yaml` (bindings), `.kbz/skills/<name>/SKILL.md` (skills), `.kbz/roles/<id>.yaml` (roles, with fallback to `.kbz/context/roles/`). The pipeline is only constructed when `stage-bindings.yaml` exists; when nil, `handoff` falls back to legacy 2.0 assembly.

3. **Diffed the two maps.** Compared the install set from step 1 against the runtime requirements from step 2 for each artifact type. Categorized each gap by severity.

4. **Audited upgrade paths.** Examined `Run()`, `runExistingProject()`, `--update-skills` path, `transformSkillContent`, and `installOneSkill`. Verified that `--update-skills` only touches `.agents/skills/` and managed roles — not `stage-bindings.yaml` or `.kbz/skills/`.

5. **Audited references.** Read `stage-bindings.yaml` (10 stage bindings referencing 17 skills and 15 roles), the project's `AGENTS.md` (references to `refs/*`, `work/templates/*`, `.github/skills/*`), the project's `.github/copilot-instructions.md` (full skill/role tables), and cross-referenced every name against the install set.

6. **Audited codebase-memory-mcp integration.** Checked references in AGENTS.md, generated AGENTS.md, skill files, and context assembly. The generated AGENTS.md treats it as optional ("If your project uses codebase-memory-mcp..."). The skill files are project-local only. The system gracefully handles absence.

7. **Compiled this report.**

## Gap Table

| ID | Category | Artifact | Symptom | Fix |
|----|----------|----------|---------|-----|
| G1 | A | `.kbz/stage-bindings.yaml` | 3.0 context pipeline never activates; `handoff` always falls back to legacy 2.0 assembly | Embed and install `stage-bindings.yaml` during init |
| G2 | A | `.kbz/skills/*` (20 task-execution skills) | If G1 were fixed, pipeline step 6 would fail with "skill not found" for every stage | Embed all 20 skills in `internal/kbzinit/skills/task-execution/`, add install step |
| G3 | B | `.kbz/roles/*` (16 role files beyond base+reviewer) | Pipeline step 5 fails with "role not found" for every stage; consumer agents have no role vocabulary | Embed all 16 role files, add install step with version-aware logic |
| G4 | B | Generated AGENTS.md lacks skill/role tables | Consumer agents have no reference mapping stages → skills → roles | Add task-execution skill and role reference tables to generated AGENTS.md |
| G5 | B | `--update-skills` doesn't update stage-bindings or .kbz/skills/ | Existing projects that upgrade `kbz` binary never get new stage bindings or task skills | Extend `--update-skills` to include stage-bindings.yaml and .kbz/skills/ update logic |
| G6 | B | Project copilot-instructions.md diverges from generated version | Project file references skills/roles that don't exist after init; generated version is too sparse | Reconcile: embed an enriched generated version; trim project file to match reality |
| G7 | C | Embedded `.agents/skills/` seeds may be stale vs. project versions | Fresh installs could get outdated skill content if seeds aren't updated | Add CI check that embedded skills match `.agents/skills/` counterparts |
| G8 | C | `.github/skills/` (codebase-memory skills) not installed by init | Generated AGENTS.md mentions them as optional, but skill files are never installed | Either embed the codebase-memory skills or document them as separate install |
| G9 | D | `refs/` and `work/templates/` referenced in project AGENTS.md but not installed | These are project-local development references; no consumer impact | No action needed — document that these are kanbanzai-project-only artifacts |

## Detailed Findings

### G1: stage-bindings.yaml missing after kbz init

**Category:** A (Critical)
**Artifact:** `.kbz/stage-bindings.yaml`
**Referenced by:**
- `internal/mcp/server.go` lines 133–145: loads `stage-bindings.yaml` to construct the 3.0 pipeline; if absent, `pipeline` stays `nil`
- `internal/mcp/handoff_tool.go` `tryPipeline()` lines 242–269: checks `pipeline == nil` and falls back to legacy 2.0 assembly
- `internal/mcp/server.go` lines 64–80: loads stage-bindings for section validation provider (gracefully degrades)
- `internal/gate/router.go`: uses registry cache from binding path (gracefully falls back to hardcoded gates)
**Symptom:** The 3.0 context assembly pipeline never activates on consumer projects. `handoff` always falls back to legacy 2.0 assembly. The pipeline's progressive disclosure, vocabulary merging, anti-pattern rendering, orchestration metadata, and token budgeting are all skipped. Consumers get a strictly inferior experience compared to this project's self-development.
**Root cause:** `stage-bindings.yaml` exists in this project's `.kbz/` directory but has no corresponding embedded source in `internal/kbzinit/` and no install step in `init.go`. The `kbz init` path was never built for it — it was added directly to `.kbz/` during Kanbanzai's self-development and the install path was never created.
**Fix:**
1. Create `internal/kbzinit/stage-bindings.yaml` with the canonical stage bindings content
2. Add `//go:embed stage-bindings.yaml` directive
3. Add `installStageBindings(kbzDir string)` function with version-aware create/update/skip logic
4. Call it from both `runNewProject` and `runExistingProject`
5. Add to `--update-skills` path

### G2: Task-execution skills missing after kbz init

**Category:** A (Critical)
**Artifact:** `.kbz/skills/*` — 20 skill directories: `write-design`, `write-spec`, `write-dev-plan`, `decompose-feature`, `orchestrate-development`, `implement-task`, `review-code`, `orchestrate-review`, `review-plan`, `write-research`, `update-docs`, `orchestrate-doc-pipeline`, `write-docs`, `edit-docs`, `check-docs`, `style-docs`, `copyedit-docs`, `audit-codebase`, `prompt-engineering`, `write-skill`
**Referenced by:**
- `internal/skill/loader.go` `SkillStore.Load()`: reads from `.kbz/skills/<name>/SKILL.md`
- `internal/context/pipeline.go` `stepLoadSkill()` line 391: calls `p.Skills.Load(skillName)` with skill names from `stage-bindings.yaml`
- `stage-bindings.yaml`: references 17 unique skill names across 10 stages
**Symptom:** If G1 were fixed (stage-bindings installed), the pipeline would activate and then immediately fail at step 6 with `"skill 'write-design' not found: check that the skill directory exists in .kbz/skills/"`. Currently masked because G1 prevents the pipeline from activating at all.
**Root cause:** These 20 skills exist in this project's `.kbz/skills/` (added manually during self-development) but have no embedded source in `internal/kbzinit/skills/` — which currently only contains 9 `kanbanzai-*` workflow skills installed to `.agents/skills/`. The install path was never built.
**Fix:**
1. Create `internal/kbzinit/skills/task-execution/` directory with all 20 SKILL.md files
2. Add `//go:embed skills/task-execution` directive
3. Add `installTaskSkills(kbzDir string)` function that writes to `.kbz/skills/<name>/SKILL.md`
4. Apply the same version-aware create/update/skip logic using the managed marker pattern
5. Wire into `runNewProject` and `runExistingProject` (under `!opts.SkipSkills`)

### G3: Role files missing after kbz init

**Category:** B (High)
**Artifact:** `.kbz/roles/*` — 16 role files beyond `base.yaml` and `reviewer.yaml`: `architect.yaml`, `spec-author.yaml`, `implementer.yaml`, `implementer-go.yaml`, `orchestrator.yaml`, `reviewer-conformance.yaml`, `reviewer-quality.yaml`, `reviewer-security.yaml`, `reviewer-testing.yaml`, `researcher.yaml`, `documenter.yaml`, `doc-pipeline-orchestrator.yaml`, `doc-editor.yaml`, `doc-checker.yaml`, `doc-stylist.yaml`, `doc-copyeditor.yaml`
**Referenced by:**
- `internal/context/role_store.go` `RoleStore.Load()`: reads from `.kbz/roles/<id>.yaml`
- `internal/context/pipeline.go` `stepResolveRole()` line 353: calls `p.Roles.Resolve(roleID)` with role IDs from `stage-bindings.yaml`
- `stage-bindings.yaml`: references 15 unique role IDs across 10 stages
**Symptom:** If G1 and G2 were fixed, the pipeline would fail at step 5 with `"failed to resolve role 'architect': role 'architect' not found"`. Currently masked by G1. Even without the pipeline, consumer agents that try to follow `stage-bindings.yaml` instructions to "read the role" find only `base` and `reviewer`.
**Root cause:** `roles.go` only installs `base.yaml` (scaffold only) and `reviewer.yaml` (managed). The remaining 16 roles were added directly to this project's `.kbz/roles/` and never given an embedded source or install step.
**Fix:**
1. Create `internal/kbzinit/roles/` directory with all 16 role YAML files
2. Add `//go:embed roles` directive
3. Extend `installRoles()` to iterate all embedded roles, using existing version-aware pattern for managed roles, and skip-if-exists for project-owned roles
4. Determine which roles are "managed" (updated on version change) vs "scaffold" (created once, never overwritten)

### G4: Generated AGENTS.md lacks skill/role reference tables

**Category:** B (High)
**Artifact:** Generated `AGENTS.md` (from `internal/kbzinit/agents_md.go`)
**Referenced by:** Consumer agents reading AGENTS.md for orientation
**Symptom:** The generated AGENTS.md is ~50 lines. It has a "Skills Reference" table listing only the 9 `kanbanzai-*` workflow skills. It has no task-execution skill table, no role table, and no stage-to-skill-to-role mapping. Consumer agents must read `stage-bindings.yaml` directly to discover which skill and role to use for each stage.
**Root cause:** The generated AGENTS.md content in `agentsMDContent` was written before the task-execution skill and role system was built. It was never updated to include the expanded catalog.
**Fix:**
1. Add a "Task-Execution Skills" table to `agentsMDContent` mapping each skill name to its path and stage
2. Add a "Roles" table mapping each role name to its file and stage
3. Add a "Stage Bindings" section explaining that `.kbz/stage-bindings.yaml` is the source of truth
4. Increment `agentsMDVersion` to 3 so existing managed files get the update

### G5: --update-skills doesn't update stage-bindings or .kbz/skills/

**Category:** B (High)
**Artifact:** `internal/kbzinit/init.go` `Run()` method, `--update-skills` path (lines 93–104)
**Referenced by:** `kbz init --update-skills` command
**Symptom:** When a consumer upgrades their `kbz` binary (e.g., `go install`), running `kbz init --update-skills` only updates `.agents/skills/kanbanzai-*` files and managed roles. It does NOT update `stage-bindings.yaml` or `.kbz/skills/` task-execution skills. Even after fixing G1 and G2, consumers on older installs would need to delete `.kbz/` and re-init — losing their config.
**Root cause:** The `--update-skills` path was designed before stage-bindings and task-execution skills existed. It only calls `installSkills()` (which writes to `.agents/skills/`) and `updateManagedRoles()`. It needs to be extended.
**Fix:**
1. Add `installStageBindings(kbzDir)` call in the `--update-skills` path
2. Add `installTaskSkills(kbzDir)` call in the `--update-skills` path
3. Add `installAllRoles(kbzDir)` call (or extend `updateManagedRoles`)
4. Consider renaming `--update-skills` to `--update` or adding a broader `--update-all` flag

### G6: Project copilot-instructions.md diverges from generated version

**Category:** B (High)
**Artifact:** `.github/copilot-instructions.md` (project version: 150+ lines) vs. generated version (25 lines in `copilotInstructionsContent`)
**Referenced by:** GitHub Copilot in this project (reads project version); consumer Copilot (reads generated version)
**Symptom:** The project version contains detailed tables of 17 skills and 16 roles, codebase-memory skill references, template paths, and reference file paths — none of which exist after a fresh install. Meanwhile, the generated version is sparse. The project version is actively misleading if it were shipped to consumers.
**Root cause:** The project's file evolved organically during Kanbanzai's self-development. The generated content was written as a minimal bootstrap and never reconciled with the project version. This is the dual-source bug.
**Fix:**
1. Enrich the generated `copilotInstructionsContent` with skill/role reference tables (mirroring G4)
2. Trim the project's `copilot-instructions.md` to reference only artifacts that exist after init, or treat it as kanbanzai-project-only and mark it as such
3. Increment `agentsMDVersion` to 3 so existing managed files get the update

### G7: Embedded skill seeds may be stale

**Category:** C (Medium)
**Artifact:** `internal/kbzinit/skills/*/SKILL.md` (9 files)
**Referenced by:** `kbz init` and `kbz init --update-skills`
**Symptom:** If the embedded seed hasn't been updated after a skill change, fresh installs get an old version. The dual-write rule (documented in AGENTS.md) requires manual discipline and has no automated verification.
**Root cause:** The dual-write rule is a human process: "When you modify any file under .agents/skills/kanbanzai-*/, check whether a corresponding file exists under internal/kbzinit/skills/. If one exists, apply the same change." There is no CI or test enforcement.
**Fix:**
1. Add a CI check or `go test` that compares embedded skill content vs. `.agents/skills/kanbanzai-*/` content (ignoring version markers)
2. The test should normalize the managed marker and version lines, then assert byte-identity

### G8: Codebase-memory skills not installed

**Category:** C (Medium)
**Artifact:** `.github/skills/codebase-memory-*/` (5 skill directories)
**Referenced by:** Project's AGENTS.md and copilot-instructions.md: "Before using any graph tool ... read the relevant SKILL.md file below"
**Symptom:** The generated AGENTS.md mentions codebase-memory-mcp as optional, but the `.github/skills/` directory is never created by init. If a consumer adds codebase-memory-mcp, they have no skill files teaching agents how to use graph tools.
**Root cause:** The codebase-memory skills were added to this project's `.github/skills/` for its own use but never given an init install path. They are treated as project-local even though they describe generic graph tool usage patterns.
**Fix:**
1. Embed the 5 codebase-memory skill files in `internal/kbzinit/`
2. Add an install step that writes them to `.github/skills/` (always-install since they're harmless without the MCP server)
3. Update generated AGENTS.md to point to the installed location
4. Alternatively: document that these are kanbanzai-project-only and consumers should get them from the kanbanzai repo separately

### G9: refs/ and work/templates/ are project-local only

**Category:** D (Low)
**Artifact:** `refs/*.md` (13 files), `work/templates/*.md` (3 files)
**Referenced by:** Project's AGENTS.md: "See `refs/go-style.md`", "see `refs/testing.md`", "use these when producing specifications, plans, or reviews: `work/templates/...`"
**Symptom:** None for consumers — these are kanbanzai-project-specific development references and should NOT be installed by init. They are correctly treated as project-local.
**Root cause:** Not a gap — these are intentionally project-local.
**Fix:** No action needed. Consider adding a comment in the project's AGENTS.md noting that `refs/` and `work/templates/` are kanbanzai-internal and not part of the consumer install.

## Upgrade Path Analysis

| Artifact | `kbz init` (new project) | `kbz init` (existing project) | `kbz init --update-skills` |
|----------|--------------------------|-------------------------------|---------------------------|
| `.agents/skills/kanbanzai-*/` | Creates all 9 | Version-aware create/update/skip | Updates managed files only |
| `.kbz/config.yaml` | Creates | Creates if absent; validates if present | No-op |
| `.kbz/context/roles/base.yaml` | Creates if absent | Creates if absent | No-op (never overwritten) |
| `.kbz/context/roles/reviewer.yaml` | Creates | Version-aware update | Updates if managed + older version |
| `.mcp.json` | Creates | Version-aware create/update/skip | No-op (stale detection runs) |
| `.zed/settings.json` | Creates | Version-aware (skip if .zed/ absent) | No-op (stale detection runs) |
| `AGENTS.md` | Creates | Version-aware (managed marker) | No-op |
| `.github/copilot-instructions.md` | Creates | Version-aware (managed marker) | No-op |
| `work/*/` | Creates dirs + .gitkeep | Creates if first-time init | No-op |
| `work/README.md` | Creates if absent | Creates if first-time init | No-op |
| **`.kbz/stage-bindings.yaml`** | **NOT INSTALLED** | **NOT INSTALLED** | **NOT UPDATED** |
| **`.kbz/skills/*` (20 skills)** | **NOT INSTALLED** | **NOT INSTALLED** | **NOT UPDATED** |
| **`.kbz/roles/*` (16 roles)** | **2 of 18 installed** | **2 of 18 installed** | **Only reviewer updated** |
| **`.github/skills/`** | **NOT INSTALLED** | **NOT INSTALLED** | **NOT UPDATED** |

## Recommendations

**Must fix (Critical — blocks pipeline activation on consumer projects):**

1. **G1 + G2 together.** These are co-requisites. Installing `stage-bindings.yaml` without skills (or vice versa) would break the pipeline. Must be done in the same change. Priority order: embed the 20 task-execution skills, embed `stage-bindings.yaml`, add install steps, test fresh init end-to-end.

**Should fix (High — blocks role resolution if pipeline activates):**

2. **G3.** Embed all 16 role files. Can be done after G1+G2 since the roles load lazily (only when a stage is entered).
3. **G5.** Extend `--update-skills` to cover stage-bindings, task skills, and all roles. This is the upgrade path for existing consumers.
4. **G4.** Enrich generated AGENTS.md with skill/role reference tables. Low-effort, high-impact for consumer agent onboarding.
5. **G6.** Reconcile copilot-instructions.md divergence. Align project file with what init actually installs.

**Nice to fix (Medium — quality and staleness):**

6. **G7.** Add CI check for embedded skill staleness. Prevents silent drift.
7. **G8.** Decide whether to embed codebase-memory skills or document them as separate install.

**Defer (Low):**

8. **G9.** No action needed. Document project-local artifacts in project AGENTS.md.
