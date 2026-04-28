# Release Readiness Assessment

| Document | Release Readiness Assessment |
|----------|------------------------------|
| Status   | Draft                        |
| Created  | 2026-03-28                   |
| Updated  | 2026-03-28                   |
| Related  | `work/spec/init-command.md`  |
|          | `work/design/init-command.md` |
|          | `work/design/kanbanzai-1.0.md` |

## 1. Purpose

This document assesses what needs to happen before kanbanzai can be used in an external project — specifically, the workflow viewer. The assessment covers four areas:

1. **Init command** — can a user run `kbz init` in a fresh project and get a working setup?
2. **AGENTS.md separation** — is kanbanzai's own AGENTS.md disentangled from the product-facing content that ships to users?
3. **Distribution** — can an external user install the binary without cloning the repo?
4. **MCP server readiness** — does the MCP server work correctly in a project that has no pre-existing `.kbz/` state?

## 2. Terminology

The "bootstrap-workflow" vs "kbz-workflow" distinction in AGENTS.md is outdated. Kanbanzai now manages its own development using its own tools — the bootstrap period is over. The terminology should be updated to reflect the current reality:

- **"The kanbanzai project"** — the Go codebase, its conventions, its AGENTS.md, its design documents. What contributors work on.
- **"The kanbanzai product"** — the MCP server, CLI binary, and skill files that get installed into someone else's project. What users consume.

The "Two Workflows" section in AGENTS.md should be replaced with a brief note acknowledging that kanbanzai is self-hosting: the project uses its own product for workflow management.

## 3. AGENTS.md Analysis

The current AGENTS.md is approximately 60% project-specific, 25% product-facing (redundant with skills), and 15% mixed. This needs to be separated so that the project's AGENTS.md contains only project-specific conventions, and product-facing guidance lives exclusively in skills.

### Should stay (project-specific)

These sections are specific to contributing to the kanbanzai codebase and do not belong in skills:

- Overview
- Naming Conventions
- Repository Structure
- Document Reading Order
- Key Design Documents by Topic (table)
- Decision-Making Rules
- Scope Guard
- YAML Serialisation Rules
- Build and Test Commands
- Go Code Style
- File Organisation
- Dependencies
- Testing conventions
- Codebase Knowledge Graph (`codebase-memory-mcp`) configuration
- Delegating to Sub-Agents (project-specific parts)

### Should be removed (already duplicated in skills)

These sections are fully covered by existing skill files and should be removed from AGENTS.md to eliminate duplication drift:

| Section | Lines (approx.) | Covered by skill |
|---------|-----------------|------------------|
| Workflow Stage Gates | ~108 | `kanbanzai-workflow`, `kanbanzai-planning`, `kanbanzai-design`, `kanbanzai-agents` |
| Document Creation Workflow | ~27 | `kanbanzai-documents` |
| Git Rules commit format table | ~20 | `kanbanzai-agents` (verbatim) |
| Emergency Brake | ~12 | `kanbanzai-workflow` |

### Needs to move to skills (product-facing, not yet covered)

Two principles in AGENTS.md are product-facing guidance that should apply to any project using kanbanzai, but are not yet captured in any skill file:

- **"Communicating With Humans"** — reference documents by name, not decision IDs. Currently in AGENTS.md §"Communicating With Humans". Should be added to the `kanbanzai-agents` skill.
- **"Documentation Accuracy"** — code is truth; spec is intent; surface conflicts to humans. Currently in AGENTS.md §"Documentation Accuracy". Should be added to the `kanbanzai-agents` skill.

### "Two Workflows" section

Replace the current "Two Workflows" section with a brief self-hosting note, e.g.:

> Kanbanzai manages its own development. The `.kbz/` state, skills, and context profiles in this repository are live — not test fixtures. Changes to workflow behaviour are reflected immediately in how agents work on this project.

## 4. Init Command Status

Feature `FEAT-01KMKRQRRX3CC` is in `developing` status. Three of six tasks are complete:

| Task | Status | Notes |
|------|--------|-------|
| CLI skeleton + flag parsing | ✅ Done | |
| New-project config generation | ✅ Done | |
| Existing-project detection | ✅ Done | |
| Skill file embedding | 🔲 Ready | Blocked by skills-content |
| Skill update logic | 🔲 Ready | |
| Atomicity + integration tests | 🔲 Ready | |

### Blocking dependency

The init command is blocked by `FEAT-01KMKRQSD1TKK` (skills-content), which is in `needs-rework` status. The situation:

- All six skill files are authored and on disk at `.agents/skills/kanbanzai-*/SKILL.md`.
- The feature has not been formally completed through its lifecycle.
- Zero `//go:embed` directives exist in the codebase — the skill content is not yet wired into the binary.

Until the skills-content feature is resolved, the init command cannot embed skill files into new projects.

## 5. MCP Server Behaviour in Fresh Projects

Testing was performed to determine whether the MCP server functions correctly in a project with no pre-existing `.kbz/` directory.

### Works out of the box

- **Server starts cleanly.** `LoadOrDefault()` falls back to `DefaultConfig()` silently when no config file exists.
- **All read operations return empty results**, not errors. Listing entities, querying knowledge, checking status — all return empty collections gracefully.
- **Most entity creation works.** Directory auto-creation via `os.MkdirAll` means `.kbz/state/` subdirectories are created on first write.
- **No hard-coded paths.** Import type mappings use relative globs (`*/design/*`), so document intelligence works regardless of project location.

### Critical bug: Plan creation fails without config file

`CreatePlan` calls `config.Load()` instead of `config.LoadOrDefault()`. In a fresh project with no `.kbz/config.yaml`, this returns an error. Plan creation is typically the first entity operation a user performs in a new project.

**Impact:** The very first meaningful action a user takes in a fresh project fails.

**Fix:** Change `config.Load()` to `config.LoadOrDefault()` in `CreatePlan`. This is a one-line change.

**Minimum viable workaround:** Manually create `.kbz/config.yaml` with a version field and at least one prefix entry. This is not acceptable as a user experience.

## 6. Distribution Gaps

There is currently no mechanism for an external user to install kanbanzai without cloning the repository and building from source.

| Gap | Impact |
|-----|--------|
| No goreleaser configuration | No automated binary builds |
| No GitHub Actions CI/CD | No release pipeline |
| Module path is `kanbanzai` (no domain prefix) | `go install` from remote will not work — Go requires a domain-qualified module path |
| Requires Go 1.25.0 | Bleeding-edge version; barrier for external users who may be on 1.23 or 1.24 |
| Builds are manual via `make build` | No reproducible release process |

## 7. Recommended Work Items

Priority-ordered list of work needed before kanbanzai can be used in an external project:

| Priority | Item | Effort | Notes |
|----------|------|--------|-------|
| 1 | Fix Plan creation bug (`LoadOrDefault()`) | Trivial | One-line fix. Unblocks fresh-project usage. |
| 2 | Resolve skills-content feature (`FEAT-01KMKRQSD1TKK`) | Small | Content exists on disk; needs lifecycle completion. |
| 3 | Finish three remaining init tasks | Medium | Embedding, update logic, atomicity + integration tests. |
| 4 | Clean up AGENTS.md | Medium | Remove product-facing content, update terminology, replace "Two Workflows" section. |
| 5 | Fix module path for `go install` support | Small | Requires choosing a domain-qualified path and updating all imports. |
| 6 | Set up goreleaser + GitHub Actions | Medium | Enables binary distribution via GitHub Releases. |
| 7 | Evaluate Go version requirement | Small | Determine whether 1.25.0 features are essential or if 1.23/1.24 is viable. |

Items 1–3 are the critical path. Items 4–7 improve the experience but are not strict blockers for a first external use.

## 8. Context Profiles and Skills

### Context profiles

Three context profiles exist in `.kbz/context/roles/`:

- `base`
- `developer`
- `reviewer`

These are not part of the init command scope. In a fresh external project, they would need to be created manually or added to the init command's output. This is a gap but not a blocker — the MCP server functions without context profiles.

### Skill files

Six skill files are authored and on disk:

| Skill | Path | Extra references |
|-------|------|-----------------|
| `kanbanzai-getting-started` | `.agents/skills/kanbanzai-getting-started/SKILL.md` | — |
| `kanbanzai-workflow` | `.agents/skills/kanbanzai-workflow/SKILL.md` | `references/lifecycle.md` |
| `kanbanzai-planning` | `.agents/skills/kanbanzai-planning/SKILL.md` | — |
| `kanbanzai-design` | `.agents/skills/kanbanzai-design/SKILL.md` | `references/design-quality.md` |
| `kanbanzai-documents` | `.agents/skills/kanbanzai-documents/SKILL.md` | — |
| `kanbanzai-agents` | `.agents/skills/kanbanzai-agents/SKILL.md` | — |

### Managed marker mismatch

There is a discrepancy between the spec and the actual skill files regarding the managed marker format:

- **Spec says:** YAML comments — `# kanbanzai-managed: ...`
- **Actual files use:** YAML fields — `metadata.kanbanzai-managed: "true"`

This needs to be reconciled before the init command ships, since the update logic depends on detecting the managed marker to determine whether it is safe to overwrite a skill file.