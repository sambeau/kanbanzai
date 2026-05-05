# Design: Handoff Pipeline Unification

**Plan ID:** P51-handoff-pipeline-unification
**Status:** Shaping
**Parent:** P50 (Retrospective Fixes — May 2026)

## Overview

The `handoff` MCP tool currently has two context assembly paths: a 3.0 pipeline (roles, skills, vocabulary, anti-patterns, knowledge, code graph) and a legacy 2.0 fallback (basic spec sections, knowledge, file paths — no role/skill/vocabulary). The fallback activates silently when a stage binding is missing, the pipeline is unconfigured, or the task has no parent feature. In practice, all three conditions indicate misconfiguration, not a valid reason to degrade.

Additionally, the orchestrator can bypass `handoff` entirely and manually compose prompts for `spawn_agent`. While P44 will solve this definitively by making dispatch a single tool call, this plan removes the dual-path confusion now so that when P44 arrives, there is only one assembly path to internalize.

## Goals

1. Remove the legacy 2.0 fallback path from `handoff` — hard errors instead of silent degradation
2. Fix sub-agent role routing so `handoff(task_id: "TASK-xxx", role: "implementer-go")` correctly resolves the `implementer-go` role and `implement-task` skill (not the orchestrator's role/skill)
3. Remove the dead code: `assembleContext`, `renderHandoffPrompt`, `buildLegacyResponse`, and the `asmInput`/`assembledContext` types
4. Simplify the `handoffTool` function signature — drop parameters that are now pipeline-only
5. Update `renderHandoffPrompt` callers in tests

## Non-Goals

- Not changing the `next` tool — it uses the same `assembleContext`, but `next` returns structured JSON, not a rendered prompt. `next` can be unified in a follow-up.
- Not building a `dispatch_task` tool (that's P44)
- Not changing the pipeline itself — just making it unconditional
- Not fixing the plan numbering reuse (P1 reused instead of P51) — tracked separately

## Design

### Current state (dual-path)

```
handoff(task_id, role, instructions)
  ├── tryPipeline(pipeline, ...)
  │   ├── pipeline != nil?                   → no  → fallback
  │   ├── parentFeature != ""?               → no  → fallback
  │   ├── binding for feature stage?         → no  → fallback
  │   └── YES → pipeline.Run(input)          → 3.0 prompt
  └── fallback: assembleContext + renderHandoffPrompt → 2.0 prompt
```

### Target state (pipeline-only)

```
handoff(task_id, role, instructions)
  └── pipeline.Run(input)                    → 3.0 prompt
      ├── pipeline == nil?                   → ERROR "stage-bindings.yaml not loaded"
      ├── parentFeature == ""?               → ERROR "task has no parent feature"
      ├── no binding for stage?              → ERROR "no binding for stage 'X'"
      └── role/skill resolution fails?       → ERROR (already the case)
```

### Changes

#### 1. `internal/mcp/handoff_tool.go`

- Remove `tryPipeline` function — `pipeline.Run` is called directly
- Remove `buildLegacyResponse` function
- Remove `renderHandoffPrompt` function
- Remove fallback code block (lines ~195-218)
- Simplify `handoffTool` signature: drop `profileStore`, `knowledgeSvc`, `intelligenceSvc`, `docRecordSvc`, `mergedToolHints`, `roleStore`, `worktreeStore` — all now supplied through the pipeline
- Update `HandoffTools` signature to match
- Update `buildPipelineResponse` to be the only response path

#### 2. `internal/mcp/assembly.go`

- Remove `assembleContext` function
- Remove `asmInput` type
- Remove `assembledContext` type
- Remove `asmSpecSection`, `asmKnowledgeEntry`, `asmFileEntry`, `asmTrimmedEntry`, `asmExperimentNudge`, `asmDocPointer` types
- Remove `asmExtractSpecSections`, `asmExtractCriteria`, `asmLoadKnowledge`, `asmLoadSiblingKnowledge`, `asmLoadDocumentPointers`, `asmExtractFiles`, `asmLoadExperimentNudge`
- Remove `ValidateFeatureStage` (if only used by the legacy path)
- Remove stage-aware guidance text constants (`asmReviewRubricText`, `asmTestExpectText`, `asmImplGuidanceText`, `asmPlanGuidanceText`)

#### 3. `internal/mcp/server.go`

- Update `HandoffTools(...)` call to pass only `entitySvc` and `pipeline`
- Remove `profileStore`, `knowledgeSvc`, `intelligenceSvc`, `docRecordSvc`, `mergedToolHints`, `roleStore`, `worktreeStore` from the handoff wiring

#### 4. Sub-agent role routing fix

In `internal/context/pipeline.go`, `stepLoadSkill` currently resolves skills as follows:

```go
// If the caller specifies a role and the binding declares sub-agents, check
// whether the caller's role prefix-matches a sub-agent role entry.
if state.Input.Role != "" && state.Binding.SubAgents != nil {
    for i, subRole := range state.Binding.SubAgents.Roles {
        if strings.HasPrefix(state.Input.Role, subRole) {
            skillName = state.Binding.SubAgents.Skills[i]
            break
        }
    }
}
// Fall back to the binding's primary skill.
if skillName == "" && len(state.Binding.Skills) > 0 {
    skillName = state.Binding.Skills[0]
}
```

When the orchestrator calls `handoff(task_id: "TASK-xxx")` without a role, `state.Input.Role` is empty, so it falls through to `state.Binding.Skills[0]` = `"orchestrate-development"`. This means the sub-agent gets the orchestrator's skill instead of the implementer's.

But when the orchestrator calls `handoff(task_id: "TASK-xxx", role: "implementer-go")`, the prefix match should work: `"implementer-go"` has prefix `"implementer"` which matches the sub-agent role. This should already be working correctly.

**However**, `stepResolveRole` also runs: it uses `state.Input.Role` if set, otherwise `state.Binding.Roles[0]` = `"orchestrator"`. So if the orchestrator doesn't pass a role, the sub-agent gets the orchestrator role AND skill. If the orchestrator passes `role: "implementer-go"`, it gets the implementer role AND skill. This is correct behavior.

**The fix** is in the `orchestrate-development` skill — it should always pass `role: "implementer-go"` (or the appropriate sub-agent role) when calling `handoff`. The skill already says this implicitly ("Always use `handoff(task_id: "TASK-xxx")`"), but it should explicitly include the role parameter.

**Additional pipeline change:** When `state.Binding.SubAgents != nil` and `state.Input.Role` is empty, `stepLoadSkill` should use the sub-agent skill rather than the binding's primary skill. The sub-agent role should also be used for role resolution. This is a pipeline-level fix:

- `stepResolveRole`: if no caller role and sub_agents exist, use `sub_agents.roles[0]`
- `stepLoadSkill`: if no caller role and sub_agents exist, use `sub_agents.skills[0]`

This ensures the default behavior (no explicit role) gives the sub-agent the right role and skill.

#### 5. Test fixes

- `internal/mcp/handoff_tool_test.go` — tests that reference legacy assembly need updating. Tests that verify the fallback conditions should instead verify hard errors.
- `internal/mcp/server_test.go` — `HandoffTools` signature change
- `internal/mcp/assembly_test.go` — remove or repurpose
- Any other test files calling `HandoffTools` or `assembleContext`

### Files affected

| File | Change |
|------|--------|
| `internal/mcp/handoff_tool.go` | Major: remove legacy path, simplify signature |
| `internal/mcp/assembly.go` | Major: remove legacy assembly code |
| `internal/mcp/server.go` | Minor: update `HandoffTools` call |
| `internal/context/pipeline.go` | Minor: fix sub-agent role/skill defaults |
| `internal/mcp/handoff_tool_test.go` | Update tests |
| `internal/mcp/assembly_test.go` | Remove or repurpose |
| `internal/mcp/next_tool.go` | May still reference `assembleContext` — leave for follow-up |

### Risk assessment

- **Risk:** `next` tool still uses `assembleContext`. **Mitigation:** Leave `assembly.go` types/functions that `next` needs; only remove handoff-specific code. Or extract the shared parts into a separate file before removing the rest.
- **Risk:** Tests may rely on the legacy path for setup. **Mitigation:** Check before removal.
- **Risk:** The pipeline requires `stage-bindings.yaml` to be present and valid. **Mitigation:** This is already a requirement for the server to start — the pipeline is nil if bindings can't be loaded, and the server logs a warning. The hard error will make this visible.

## Dependencies

- Requires `stage-bindings.yaml` to be present and valid (already the case)
- Requires all working stages to have bindings (already the case)
- No dependency on P44 — this is a prerequisite cleanup for P44, not dependent on it

## Alternatives Considered

### Keep the fallback, add a visible warning

**Idea:** Keep the legacy path but add a prominent warning in the response metadata when it's used.

**Reject:** The warning would be visible to the orchestrator but not actionable — the orchestrator can't fix a missing binding or pipeline config. Warnings that can't be acted on become noise. And the orchestrator would still receive a degraded prompt without role/skill/vocabulary context, defeating the purpose.

### Merge handoff output into next

**Idea:** Have `next` return the assembled prompt alongside the structured context, eliminating the separate `handoff` call.

**Defer:** This is a good idea but changes the `next` contract (currently returns structured JSON). It's a larger change that should be its own feature. The dual-path problem is narrower and more urgent.

### Build dispatch_task now (P44 Phase 1)

**Idea:** Skip the cleanup and go straight to building `dispatch_task`.

**Reject:** P44 is a feasibility design, not an implementation plan. Building provider integrations, API key management, and a dispatch loop is a large feature with its own risks. This plan removes the dual-path confusion as a prerequisite so that when P44 is built, there's only one assembly path to internalize.

## Open Questions

1. Should `next` also be unified in this plan, or left for a follow-up? (Recommend: follow-up — `next` has its own output format and callers.)
2. Should the `handoff` tool rename to reflect that it's now pipeline-only? (Recommend: no — the tool name is stable; it's the implementation that's changing.)
