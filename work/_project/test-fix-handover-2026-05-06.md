# Test Fix Handover — 2026-05-06

## Context

After a mass worktree/branch cleanup, tests were run and failures were found across
multiple packages. This handover summarizes what was fixed, what's still broken,
and the investigation results for the next agent.

## Changes Made (on main, uncommitted)

### ✅ 1. `cmd/kbz/main_test.go` — `fakeDocService.LookupByPath` pointer fix
**Problem:** Interface `docService.LookupByPath` returns `(*service.DocumentResult, error)`
but the fake returned `(service.DocumentResult, error)` — build failure.
**Fix:** Changed `lookupResults` map to `map[string]*service.DocumentResult`, updated
return values to use `&service.DocumentResult{}` instead of `nil` (nil caused panics in
callers that access `.ID` without nil check).
**Status:** ✅ Fixed. 1 remaining failure (`TestReParentEntityUpdate`) is pre-existing.

### ✅ 2. `internal/mcp/status_tool_test.go` — scope assertion fix
**Problem:** Two tests expected `scope == "plan"` but `synthesisePlanEntity` now
returns `"strategic_plan"`.
**Fix:** Updated both assertions to `"strategic_plan"`.
**Status:** ✅ Fixed.

### ⚠️ 3. Embedded skill seed files — managed marker comments
**Problem:** `transformSkillContent` expects `# kanbanzai-managed: true` comment lines
in seed files, but seeds used YAML frontmatter `metadata: {kanbanzai-managed: "true"}`.
**Fix:** Added `# kanbanzai-managed: true` and `# kanbanzai-version: dev` comment lines
to 6 agent skill seeds (`agents`, `planning`, `workflow` — `design`, `documents`,
`getting-started` already had them). Also synced 4 task-execution skill seeds
(`implement-task`, `orchestrate-review`, `write-design`, `write-spec`) from
`.kbz/skills/` counterparts and added markers.
**Status:** ⚠️ `TestRun_NewProject_SkillFrontmatter` ✅ and
`TestEmbeddedTaskSkillsMatchProjectSkills` ✅ now pass. But 9 idempotency/update tests
still fail (second-run init can't find `# kanbanzai-managed:` in output files).
`TestPipelineReadiness_NewProject` also fails.

## Current Test State

| Package | Failures | Classification |
|---------|----------|---------------|
| `cmd/kbz` | 1 | Pre-existing (`TestReParentEntityUpdate`) |
| `internal/kbzinit` | 9 | 8 idempotency + 1 pipeline readiness |
| `internal/mcp` | ~30 | Mix — see below |
| `internal/service` | ~25 | Pre-existing |
| `internal/storage` | 4 | Pre-existing |
| All others (30 pkgs) | ✅ | Green |

Confirmed: `internal/service` and `internal/storage` failures are pre-existing —
`git diff main` shows no changes to those files.

## Key Investigation: The Plan/Batch Type Mismatch (UNRESOLVED)

This is the core problem causing ~20 of the MCP test failures. Test helpers like
`createEntityTestPlan` write entities with `Type: "plan"` which goes to the
`"plans/"` directory via `entityDirectory()`. But `loadPlan()` loads with
`EntityKindPlan = "batch"` which goes to `"batches/"`. The plan file can't be found.

```
Test helpers:     Type: "plan"  → entityDirectory("plan")  = "plans/"
Production code:  EntityKindPlan = "batch" → entityDirectory("batch") = "batches/"
                  loadPlan uses EntityKindPlan → looks in "batches/"
```

**Approaches tried (all reverted):**

1. **`loadPlan` fallback to `"plan"` type** — cascading failures in service tests.
2. **`entityDirectory` remap `"batch"` → `"plans"`** — broke storage tests.
3. **Test helpers → `"batch"` + dual-directory `ListPlans`/`listPlanIDs`** — caused
   service/storage regressions from dual-directory scanning.

The code is back to its original state for these files. The fix needs a more surgical
approach — perhaps updating ALL test plan-creation helpers to use `Type: "batch"`,
or fixing the entity tool's `"plan"` → `"batch"` routing, or adding a plan-resolution
layer that checks both directories.

## Suggested Approach

1. **Fix plan/batch mismatch** — update all test helpers that create plans
   (`createEntityTestPlan`, `createTestPlan`, `createFinishTestPlan`,
   `createHandoffPlan`, `createActivePlan`, `writeIntegrationPlan`,
   `writePlanRecord`, etc.) to use `Type: "batch"`. Then update `ListPlans`,
   `listPlanIDs`, and `ListStrategicPlans` to scan both `"plans/"` and
   `"batches/"` with dedup by ID. This avoids changing `loadPlan`.

2. **Fix kbzinit idempotency** — `transformSkillContent` replaces
   `# kanbanzai-managed: true` with `# kanbanzai-managed: do not edit...`.
   First run creates file with marker. Second run should find it via `hasLine`.
   Investigate why `hasLine` doesn't find it.

3. **Remaining MCP failures** — merge tests, plan gate, handoff stage validation,
   doc tool tests. Mostly pre-existing, unrelated to git cleanup.

## Files Modified

- `cmd/kbz/main_test.go` — `fakeDocService` pointer fix
- `internal/mcp/status_tool_test.go` — scope assertion fix
- `internal/kbzinit/skills/agents/SKILL.md` — added managed markers
- `internal/kbzinit/skills/planning/SKILL.md` — added managed markers
- `internal/kbzinit/skills/workflow/SKILL.md` — added managed markers
- `internal/kbzinit/skills/task-execution/implement-task/SKILL.md` — synced + markers
- `internal/kbzinit/skills/task-execution/orchestrate-review/SKILL.md` — synced + markers
- `internal/kbzinit/skills/task-execution/write-design/SKILL.md` — synced + markers
- `internal/kbzinit/skills/task-execution/write-spec/SKILL.md` — synced + markers
