# Design: Bootstrapping Gap Remediation

| Field | Value |
|-------|-------|
| Date | 2025-07-17 |
| Status | Draft |
| Author | System Architect (AI) |
| Informed by | `work/P47-bootstrapping-gap-remediation/report-bootstrapping-gap-audit.md` |
| Supersedes | `work/_project/design-fresh-install-experience.md` (updated to reflect current 3.0 pipeline reality) |

---

## 1. Problem Statement

`kbz init` produces a project that cannot activate the 3.0 context assembly pipeline. The pipeline requires `stage-bindings.yaml`, which requires `.kbz/skills/`, which requires `.kbz/roles/` — none of which are installed. The pipeline silently degrades to legacy 2.0 mode. Consumer projects never experience progressive disclosure, vocabulary merging, anti-pattern rendering, or orchestration metadata.

The audit identified 9 gaps: 2 Critical (G1: `stage-bindings.yaml`, G2: task-execution skills), 4 High (G3: roles, G4: AGENTS.md content, G5: upgrade path, G6: copilot-instructions divergence), 2 Medium (G7: stale seeds, G8: codebase-memory skills), and 1 Low (G9: project-local refs).

## 2. Design Principles

### DP-1: Single Source of Truth for Init Artifacts

Every artifact installed by `kbz init` must have exactly one canonical source: the embedded filesystem in `internal/kbzinit/`. Never have a file exist in `.kbz/` or `.agents/` without a corresponding embedded source. This eliminates the dual-source bug by construction.

### DP-2: Co-install Interdependent Artifacts

If artifact A references artifact B, installing A without B creates a broken state. `stage-bindings.yaml` references skills; skills are loaded from `.kbz/skills/`. These must be installed atomically — all or nothing.

### DP-3: Version-Aware Update for All Managed Files

Every installed file that is managed (not user-owned) must carry a version marker and support the same create/update/skip/conflict resolution logic already implemented for `.agents/skills/kanbanzai-*`. No partial upgrade paths.

### DP-4: Consumer Project Is the Primary Use Case

The Kanbanzai project's self-bootstrapping is secondary. Design decisions prioritize the consumer experience. If the Kanbanzai project needs special local files, those files should explicitly declare themselves as project-local.

### DP-5: Graceful Degradation for Optional Dependencies

codebase-memory-mcp is optional. References to it must be conditional. The system must work fully without it. The generated AGENTS.md should document it as optional with clear setup instructions.

## 3. Design — Remediation Architecture

### 3.1 New embedded filesystem structure

```
internal/kbzinit/
├── init.go                     # (existing)
├── skills.go                   # (existing — .agents/skills/ installer)
├── roles.go                    # (extended — from 2 roles to all 18)
├── agents_md.go               # (extended — enriched content)
├── config_writer.go           # (unchanged)
├── mcp_config.go              # (unchanged)
├── git.go                     # (unchanged)
├── stage-bindings.yaml        # NEW — embedded binding file
├── task_skills.go             # NEW — .kbz/skills/ installer
├── codebase_skills.go         # NEW — .github/skills/ installer (optional, G8)
├── skills/
│   ├── agents/                # (existing)
│   ├── design/                # (existing)
│   ├── ...                    # (existing — 7 more workflow skills)
│   └── task-execution/        # NEW — 20 SKILL.md files
│       ├── write-design/
│       ├── write-spec/
│       └── ...
└── roles/
    ├── base.yaml              # (existing)
    ├── reviewer.yaml          # (existing)
    ├── architect.yaml         # NEW
    ├── spec-author.yaml       # NEW
    └── ...                    # NEW — 14 more role files
```

### 3.2 Installation flow (revised)

The new `kbz init` flow for a fresh project:

1. Create `.kbz/config.yaml` (existing)
2. Create `work/*/` directories (existing)
3. Install `.agents/skills/kanbanzai-*` (existing)
4. **Install `.kbz/stage-bindings.yaml`** (NEW — G1)
5. **Install `.kbz/skills/*` task-execution skills** (NEW — G2)
6. **Install `.kbz/roles/*` all role files** (NEW — G3)
7. Write `.mcp.json` and `.zed/settings.json` (existing)
8. Write `AGENTS.md` and `.github/copilot-instructions.md` (existing, enhanced — G4/G6)

### 3.3 Detailed fix for each gap

#### G1: Install stage-bindings.yaml

**New file:** `internal/kbzinit/stage-bindings.yaml`
**Content:** The canonical binding file from `.kbz/stage-bindings.yaml` (currently 10 stage bindings)
**New code in** `init.go`:

```go
//go:embed stage-bindings.yaml
var embeddedStageBindings []byte

const stageBindingsVersion = 1

func (i *Initializer) installStageBindings(kbzDir string) error {
    destPath := filepath.Join(kbzDir, "stage-bindings.yaml")
    // Version-aware create/update/skip logic using a YAML comment marker
    // "# kanbanzai-managed: true\n# kanbanzai-version: N"
    // Same pattern as installOneSkill in skills.go
}
```

**Called from:** `runNewProject()` and `runExistingProject()` after config write, before skills install (since skills are loaded by the binding).

**Version marker scheme:** Use YAML comment markers on line 1-2:
```yaml
# kanbanzai-managed: true
# kanbanzai-version: 1
```

#### G2: Install task-execution skills

**New directory:** `internal/kbzinit/skills/task-execution/`
**New file:** `internal/kbzinit/task_skills.go`
**Content:** Copy all 20 skill directories from `.kbz/skills/` with their `SKILL.md` files.

**Install path:** `.kbz/skills/<name>/SKILL.md`
**Version scheme:** Reuse the same managed marker pattern from `skills.go` (`# kanbanzai-managed:` and `# kanbanzai-version:` in YAML frontmatter).

**Skill list:**
```
write-design, write-spec, write-dev-plan, decompose-feature,
orchestrate-development, implement-task, review-code, orchestrate-review,
review-plan, write-research, update-docs, orchestrate-doc-pipeline,
write-docs, edit-docs, check-docs, style-docs, copyedit-docs,
audit-codebase, prompt-engineering, write-skill
```

**Design decision:** All task-execution skills are managed (version-aware update). The `transformSkillContent` function from `skills.go` can be reused directly since the frontmatter format is identical.

#### G3: Install all role files

**New directory:** `internal/kbzinit/roles/`
**Extended file:** `internal/kbzinit/roles.go`

**Role classification:**

| Role | Install strategy |
|------|-----------------|
| `base` | Scaffold (create once, never overwrite) |
| `reviewer` | Managed (version-aware update) — existing |
| `architect` | Managed |
| `spec-author` | Managed |
| `implementer` | Managed |
| `implementer-go` | Managed |
| `orchestrator` | Managed |
| `reviewer-conformance` | Managed |
| `reviewer-quality` | Managed |
| `reviewer-security` | Managed |
| `reviewer-testing` | Managed |
| `researcher` | Managed |
| `documenter` | Managed |
| `doc-pipeline-orchestrator` | Managed |
| `doc-editor` | Managed |
| `doc-checker` | Managed |
| `doc-stylist` | Managed |
| `doc-copyeditor` | Managed |

**Design decision:** All roles except `base` are managed. `base` is user-owned (project conventions). `reviewer` is the inheritance base for reviewer sub-types and must be managed because sub-types depend on it.

**Install path:** `.kbz/roles/<id>.yaml` (new location; the old `.kbz/context/roles/` is the legacy location)

**Version marker:** Same YAML inline marker as reviewer.yaml: `kanbanzai-managed: "true"` with `version: "<version>"`.

#### G4: Enrich generated AGENTS.md

**Modified file:** `internal/kbzinit/agents_md.go`
**Version bump:** `agentsMDVersion` from 2 to 3

**New content additions to `agentsMDContent`:**
- "Task-Execution Skills" table: 17 rows mapping skill → `.kbz/skills/` path → stage
- "Roles" table: 17 rows mapping role → `.kbz/roles/` path → stage
- "Stage Bindings" section: "The authoritative mapping is in `.kbz/stage-bindings.yaml`. Read it to find your role and skill for the current workflow stage."

**Also update `copilotInstructionsContent`:**
- Add: "Read `.kbz/stage-bindings.yaml` before starting any workflow stage to find the correct role and skill."
- Keep under the 25-line limit by being terse.

#### G5: Extend --update-skills

**Modified file:** `internal/kbzinit/init.go` `Run()` method

**Current `--update-skills` path:**
```go
if opts.UpdateSkills {
    i.detectStaleMCPConfigs(gitRoot)
    i.installSkills(gitRoot)        // .agents/skills/ only
    i.updateManagedRoles(kbzDir)    // reviewer.yaml only
}
```

**New `--update-skills` path:**
```go
if opts.UpdateSkills {
    i.detectStaleMCPConfigs(gitRoot)
    i.installSkills(gitRoot)           // .agents/skills/ workflow skills
    i.installStageBindings(kbzDir)     // NEW — G1
    i.installTaskSkills(kbzDir)        // NEW — G2
    i.installAllRoles(kbzDir)          // NEW — all managed roles
}
```

**Design decision:** Keep the flag name `--update-skills` for backward compatibility. It now updates more than skills, but the spirit is the same: "update the managed artifacts." If confusion arises, add `--update` as an alias later.

#### G6: Reconcile copilot-instructions divergence

**Two-part approach:**

1. **Enrich generated `copilotInstructionsContent`** in `agents_md.go` with a condensed skill/role quick-reference (mirroring G4 but under 25 lines).

2. **Trim project's `.github/copilot-instructions.md`** to reference only artifacts that exist after init, OR add a header comment: `<!-- kanbanzai-project: this file is hand-maintained for kanbanzai's own development and contains project-local references not present in generated installs -->`.

The generated file is the canonical consumer version. The project file is kanbanzai-specific. Mark this clearly.

#### G7: CI staleness check for embedded skills

**New test file:** `internal/kbzinit/skills_consistency_test.go`

```go
func TestEmbeddedSkillsMatchAgentSkills(t *testing.T) {
    for _, name := range skillNames {
        embedded, _ := embeddedSkills.ReadFile("skills/" + name + "/SKILL.md")
        onDisk, _ := os.ReadFile("../../.agents/skills/kanbanzai-" + name + "/SKILL.md")
        // Normalize version markers to "dev" before comparing
        assert.Equal(t, normalizeMarkers(embedded), normalizeMarkers(onDisk))
    }
}
```

Run as part of `go test ./internal/kbzinit/...` (not integration-only).

#### G8: Codebase-memory skills

**Decision: Defer to separate install.**

The 5 codebase-memory skills are tightly coupled to `codebase-memory-mcp` which is a separate tool. Installing them by default would create files that reference tools that don't exist unless the user also installs codebase-memory-mcp.

**Instead:**
- Update generated AGENTS.md to say: "If your project uses codebase-memory-mcp, copy the graph tool skills from the kanbanzai repository's `.github/skills/` directory into your project."
- These skills could become a separate `kbz init --with-codebase-memory` flag in a follow-up.

#### G9: No action needed

Project-local artifacts (`refs/`, `work/templates/`) are correctly not installed. Add a note in the project's AGENTS.md clarifying this.

## 4. Implementation Sequence

### Phase 1: Critical path (G1 + G2, co-required)

1. Create `internal/kbzinit/stage-bindings.yaml` — copy from `.kbz/stage-bindings.yaml`
2. Create `internal/kbzinit/task_skills.go` with embed directive and install function
3. Create `internal/kbzinit/skills/task-execution/` with all 20 skill directories
4. Add `installStageBindings` and `installTaskSkills` calls to `runNewProject`, `runExistingProject`, and `--update-skills`
5. Test: `kbz init` on a fresh git repo, verify pipeline activates

### Phase 2: Role completion (G3)

6. Create `internal/kbzinit/roles/` with all 16 role files
7. Extend `installRoles` in `roles.go` to iterate embedded roles
8. Test: verify all roles load via `RoleStore.LoadAll()`

### Phase 3: Documentation enrichment (G4 + G6)

9. Update `agentsMDContent` and `copilotInstructionsContent` in `agents_md.go`
10. Bump `agentsMDVersion` to 3
11. Trim project's `copilot-instructions.md` to avoid confusion

### Phase 4: Quality (G7 + G8)

12. Add `skills_consistency_test.go`
13. Decide G8 disposition and document

## 5. Prevention — Avoiding Future Dual-Source Bugs

### 5.1 Structural prevention

**The `go:embed` invariant:** If a file exists in `.kbz/`, `.agents/`, or `.github/` under the Kanbanzai project and it should also exist in consumer projects after `kbz init`, there must be a corresponding `go:embed` + install function in `internal/kbzinit/`. This is enforceable: a CI check can walk `.kbz/` and `.agents/` (excluding state files), cross-reference `internal/kbzinit/`, and flag files that exist only in the project.

**Explicit project-local markers:** Files that are intentionally project-local (like `refs/*`, `work/templates/*`, the project's hand-maintained `copilot-instructions.md`) should declare themselves. A simple convention: `<!-- project-local: true -->` in Markdown files, `# project-local: true` in YAML files.

### 5.2 Process prevention

**The "init path required" rule:** Every feature that adds files to `.kbz/`, `.agents/`, or `.github/` (that are not project-local state or config) must include the corresponding init logic in the same or immediately following batch. This should be documented in the project's AGENTS.md.

**Staleness check in CI:** The test from G7 should be extended to cover all embedded files — not just skills, but also stage-bindings and roles. Any PR that modifies a file under `.kbz/skills/`, `.kbz/roles/`, `.kbz/stage-bindings.yaml`, or `.agents/skills/kanbanzai-*/` must pass the staleness check.

### 5.3 Testing prevention

**End-to-end init test:** Add an integration test that:
1. Creates a fresh git repo
2. Runs `kbz init`
3. Verifies every expected file exists at the right path
4. Verifies the pipeline activates (load stage-bindings, resolve a role, load a skill)
5. Verifies `--update-skills` updates managed files

This is the single most effective guard against future asymmetry.

## 6. Risks and Trade-offs

| Risk | Mitigation |
|------|-----------|
| Binary size increase from embedding 20 SKILL.md files + 16 YAML files | Skills are ~2-5 KB each, roles ~1-2 KB each. Total ~150 KB. Negligible for a Go binary. |
| User customizations to managed skills overwritten on update | Version-aware logic skips files without managed markers. Users who customize manage skills must remove the marker. This is documented behavior. |
| stage-bindings.yaml version conflicts | Version marker allows detection. Mismatch produces a warning, not a failure. Users can set `--skip-skills` to bypass. |
| Consumer projects with pre-existing .kbz/ from older kanbanzai | `runExistingProject` handles this. First-time init on existing project creates missing files; re-run updates managed files. |

## 7. Success Criteria

1. `kbz init` on a fresh git repo creates a project where the 3.0 pipeline activates
2. `handoff` returns context with `assembly_path: pipeline-3.0` (not `legacy-2.0`)
3. All 10 stage bindings resolve their roles and skills without "not found" errors
4. `kbz init --update-skills` updates all managed artifacts on an existing project
5. The generated AGENTS.md contains task-execution skill and role reference tables
6. CI catches embedded seed drift
