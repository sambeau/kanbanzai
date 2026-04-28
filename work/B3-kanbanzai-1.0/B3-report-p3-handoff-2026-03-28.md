# P3 Handoff Summary — 2026-03-28T18:09Z

| Document | P3 Handoff Summary                     |
|----------|----------------------------------------|
| Plan     | P3-kanbanzai-1.0 (Kanbanzai 1.0)      |
| Date     | 2026-03-28                             |
| Context  | End-of-session handoff for continuity  |

---

## 1. Goal

Get kanbanzai ready for its first public release, then use it to bootstrap a new project (the workflow viewer). The viewer project is the 1.0 acceptance criterion — fresh repo, `kanbanzai init`, use only the public interface (design doc §12).

## 2. What Was Done This Session

### Research and triage
- **Full release readiness assessment** written at `work/research/release-readiness-assessment.md` (registered as `PROJECT/research-release-readiness-assessment`).
- Analysed AGENTS.md: ~60% project-specific (stays), ~25% product-facing duplicating skills (remove), ~15% mixed.
- Triaged all P3 feature branches — discovered most work is done on unmerged branches.
- Identified the dependency chain between branches.

### Bug fix
- **Plan creation bug fixed** — `CreatePlan` in `internal/service/plans.go` now uses `config.LoadOrDefault()` instead of `config.Load()`, so plans can be created in fresh projects without `.kbz/config.yaml`. Tests added and passing. Committed to main.

### New features created
- **FEAT-01KMTSPAV34HR (agents-md-cleanup)** — separate project-specific from product-facing content in AGENTS.md. Spec approved (14 ACs), 4 tasks created.
- **FEAT-01KMTSPEE6BXS (release-infrastructure)** — module path change to `github.com/sambeau/kanbanzai`, Go version evaluation, README updates. Spec approved (12 ACs), 3 tasks created.

### Key decision recorded
- GitHub owner/org: **sambeau** → `github.com/sambeau/kanbanzai`

## 3. P3 Feature Status

### Track A: Independent (no blockers)

| Feature | ID | Status | Notes |
|---|---|---|---|
| **public-schema-interface** | FEAT-01KMKRQV025FA | `developing` | 5/5 tasks done on branch `feature/FEAT-01KMKRQV025FA-public-schema-interface`. **Clean merge** to main. Ready for review + merge. |
| **user-documentation** | FEAT-01KMKRQVKBPRX | `dev-planning` | 5/5 tasks done, content already on main. Needs lifecycle transitions to `done`. |
| **agents-md-cleanup** | FEAT-01KMTSPAV34HR | `developing` | 4 tasks queued. Spec: `work/spec/agents-md-cleanup.md`. |
| **release-infrastructure** | FEAT-01KMTSPEE6BXS | `developing` | 3 tasks queued. Spec: `work/spec/release-infrastructure.md`. |

### Track B: Dependency chain (skills-content is critical path)

```
skills-content (clean merge, needs rework first)
  └─ merged into → init-command (conflicts with main in .kbz/state/ and cmd/kanbanzai/main.go)
                      ├─ merged into → binary-distribution (inherits conflicts)
                      └─ merged into → hardening (inherits conflicts)
```

| Feature | ID | Status | Notes |
|---|---|---|---|
| **skills-content** | FEAT-01KMKRQSD1TKK | `needs-rework` | All 6 tasks done but review found **8 blockers**: stale 1.0 tool names (~20 refs across all 6 skill files), fabricated lifecycle states in `references/lifecycle.md`. Review report at `work/reviews/review-FEAT-01KMKRQSD1TKK-skills-content.md`. Fix = update tool names to 2.0, rewrite lifecycle reference from `internal/validate/lifecycle.go`. |
| **init-command** | FEAT-01KMKRQRRX3CC | `developing` | All 6 tasks done on branch. Merged skills-content. After skills fix: rebase on main, resolve conflicts (entity YAML + main.go), re-embed fixed skills, merge. |
| **hardening** | FEAT-01KMKRQWF0FCH | `developing` | 5/5 tasks done on branch. Merged init-command. After init-command merges: rebase, resolve conflicts, merge. |
| **binary-distribution** | FEAT-01KMKRQT9QCPR | `developing` | 4/5 tasks done on branch (goreleaser, GitHub Actions, install.sh built). 1 task `needs-review` (release-validation: push test tag). Merged init-command. After init-command merges: rebase, resolve conflicts, merge. |

## 4. Task Breakdown

### agents-md-cleanup tasks

| Task | ID | Status | Depends on |
|---|---|---|---|
| remove-redundant-sections | TASK-01KMTT3VY4R5C | `queued` | — |
| update-terminology | TASK-01KMTT3VYM1AA | `queued` | — |
| move-principles-to-skills | TASK-01KMTT3VZ2ZXQ | `queued` | — |
| verify-coherence | TASK-01KMTT3VZFXT8 | `queued` | All 3 above |

### release-infrastructure tasks

| Task | ID | Status | Depends on |
|---|---|---|---|
| module-path-change | TASK-01KMTT47PA5P5 | `queued` | — |
| go-version-evaluation | TASK-01KMTT47PSTBS | `queued` | — |
| update-readme-and-docs | TASK-01KMTT47Q719Y | `queued` | module-path-change |

## 5. Recommended Execution Order

### Phase 1: Parallel quick wins
1. **Merge public-schema-interface** — clean merge, no conflicts. Review the branch diff, merge, transition feature to done.
2. **Transition user-documentation** through lifecycle — all content is on main already, just needs entity state updates.
3. **Execute module-path-change** — mechanical find-replace of all Go imports from `kanbanzai/internal/...` to `github.com/sambeau/kanbanzai/internal/...`. Update `go.mod`. Single atomic commit. Run `go build ./...` and `go test ./...` to verify.
4. **Execute go-version-evaluation** — check if any Go 1.25.0 features are used. Lower if possible.

### Phase 2: AGENTS.md cleanup (parallel with Phase 1)
5. Execute the three independent agents-md-cleanup tasks (remove, update terminology, move to skills), then the verification task.

### Phase 3: Skills-content fix (critical path)
6. **Fix skills-content** — update ~20 stale 1.0 tool names to 2.0 equivalents across all 6 skill files. Rewrite `.agents/skills/kanbanzai-workflow/references/lifecycle.md` with correct states from `internal/validate/lifecycle.go`. The review report has the specific findings.
7. Merge skills-content to main (clean merge).

### Phase 4: Merge the chain
8. Rebase init-command on main, resolve conflicts, merge.
9. Rebase hardening on main, resolve conflicts, merge.
10. Rebase binary-distribution on main, resolve conflicts, merge.
11. Execute release-validation (push a test tag, verify artefacts).

### Phase 5: Final
12. Update README and docs (depends on module path being done).
13. Final verification — run full test suite, check all features done.

## 6. Key Files and Locations

| What | Path |
|---|---|
| P3 design document | `work/design/kanbanzai-1.0.md` |
| Release readiness research | `work/research/release-readiness-assessment.md` |
| agents-md-cleanup spec | `work/spec/agents-md-cleanup.md` |
| release-infrastructure spec | `work/spec/release-infrastructure.md` |
| Skills-content review report | `work/reviews/review-FEAT-01KMKRQSD1TKK-skills-content.md` |
| Six kanbanzai skill files | `.agents/skills/kanbanzai-*/SKILL.md` |
| AGENTS.md | `AGENTS.md` (project root) |
| Plan creation bug fix | `internal/service/plans.go` line 54 |
| Lifecycle definitions (for skills fix) | `internal/validate/lifecycle.go` |

## 7. Decisions Made

| Decision | Detail |
|---|---|
| GitHub org/owner | `sambeau` → `github.com/sambeau/kanbanzai` |
| Use existing P3 plan | No new plan created; two new features added to P3 |
| Plan creation fix | `LoadOrDefault()` instead of `Load()` — committed to main |
| Both new specs approved | 14 ACs (agents-md) + 12 ACs (release-infra) |

## 8. Open Questions for Human

1. **Go version**: Do you have a preference for minimum Go version, or should the agent evaluate purely on technical grounds (what compiles)?
2. **Binary distribution validation**: The release-validation task requires pushing a `v1.0.0-alpha.1` tag. This should wait until all other work is merged. Confirm timing when ready.
3. **Skills-content rework scope**: The review found the feature has no specification document (only a design doc). Should a spec be written before the rework, or is fixing the review findings sufficient?

## 9. Commits Made This Session

```
66d53f5 docs: add release readiness assessment research report
4479591 fix(FEAT-01KMKRQRRX3CC): use LoadOrDefault in CreatePlan for fresh projects
6d59d01 workflow(P3-kanbanzai-1.0): add agents-md-cleanup and release-infrastructure features with specs
2152307 workflow(P3-kanbanzai-1.0): approve specs, create tasks for agents-md-cleanup and release-infrastructure
```

All on `main` branch. No uncommitted changes.