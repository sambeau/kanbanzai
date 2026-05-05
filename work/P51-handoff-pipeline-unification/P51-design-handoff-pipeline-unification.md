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

**The fix has two parts:**

1. **Skill documentation (immediate):** The `orchestrate-development` skill should explicitly instruct orchestrators to pass `role: "implementer-go"` (or the appropriate language-specific sub-agent role) when calling `handoff` for `spawn_agent` dispatch. The current instruction "Always use `handoff(task_id: "TASK-xxx")`" is silently incomplete — it produces orchestrator output when the consumer is an implementer. The corrected instruction should read: "Always use `handoff(task_id: "TASK-xxx", role: "implementer-go")` to generate sub-agent prompts."

2. **Pipeline default change (structural):** When `state.Binding.SubAgents != nil` and `state.Input.Role` is empty, default to the sub-agent role/skill rather than the primary binding role/skill. This makes the common case (sub-agent dispatch via `spawn_agent`) correct by default. The explicit `role` parameter would still override for cases where the orchestrator wants a different role. P50 proved that the current default (`orchestrator` role) is never the right answer when the output goes to `spawn_agent`.

**Concrete incident (P50, May 2026):** During fast-track implementation, the orchestrator called `handoff(task_id: "TASK-01KQTX5DXKRTH")` (and 11 other task IDs) without passing a `role` parameter. Every response came back with `assembly_path: "pipeline-3.0"` and `total_tokens: ~8900` — the pipeline was active and working. But the assembled prompt was the full `orchestrate-development` skill: 30+ anti-patterns (Premature Delegation, Context Bloat Without Offloading, Serial Dispatch of Independent Tasks...), a 6-phase orchestration procedure (Cohort Setup, Read Dev-Plan, Identify Parallel-Dispatchable Tasks, Dispatch Sub-Agents, Monitor Progress, Close-Out), vocabulary glossary, BAD/GOOD orchestration examples, and evaluation criteria for orchestrators.

The orchestrator recognized this as orchestrator training material — not implementer instructions — discarded all handoff output, and manually composed 12 custom implementer prompts (~400-600 tokens each) with task descriptions, worktree paths, file scopes, and commit formats. The manual prompts lacked spec sections, knowledge entries, code graph context, and role-grounded vocabulary that the pipeline would have included had it resolved the implementer role.

Root cause trace through the pipeline:
1. `handoff(task_id, role: "")` → `state.Input.Role` is empty
2. `stepResolveRole`: no caller role → falls back to `state.Binding.Roles[0]` = `"orchestrator"`
3. `stepLoadSkill`: no caller role → falls back to `state.Binding.Skills[0]` = `"orchestrate-development"`
4. Pipeline correctly produces orchestrator prompt — but the consumer is `spawn_agent`, not an orchestrator

The sub-agent role/skill resolution logic at `pipeline.go` (checking `state.Input.Role` against `state.Binding.SubAgents.Roles` prefixes) was never triggered because `state.Input.Role` was empty. The orchestrator had no way to discover that `role: "implementer-go"` was needed — the `orchestrate-development` skill says "Always use `handoff(task_id: "TASK-xxx")`" without mentioning the role parameter.

**Additional pipeline change:** When `state.Binding.SubAgents != nil` and `state.Input.Role` is empty, `stepLoadSkill` should use the sub-agent skill rather than the binding's primary skill. The sub-agent role should also be used for role resolution. This is a pipeline-level fix:

- `stepResolveRole`: if no caller role and sub_agents exist, use `sub_agents.roles[0]`
- `stepLoadSkill`: if no caller role and sub_agents exist, use `sub_agents.skills[0]`

This is not a hypothetical edge case — P50 demonstrated it as the default path. Every `handoff` call during fast-track implementation hit this fallback, and every sub-agent received the wrong role/skill as a result. The pipeline's sub-agent resolution logic exists and works correctly when triggered with an explicit role parameter — the fix is simply to make it the default.

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

## Findings from P50 Fast-Track Implementation

### Trimming visibility gap

During P50, the orchestrator received `next` responses with `byte_budget: 30720 / byte_usage: 30301` (98.6% full). The `trimmed` metadata listed entry counts and sizes but not **topics** — the orchestrator couldn't tell whether critical knowledge entries had been dropped. Several knowledge entries that would have been relevant (the handoff→spawn_agent loop not being closed, the retro-feature-already-implemented pattern) may have been trimmed without the orchestrator ever knowing.

**Design note:** The 30KB cap (`assemblyDefaultBudget = 30720`) is an MCP response size guard, not a context window limit. When P44's `dispatch_task` assembles prompts internally for provider API calls, this cap should not apply — the pipeline is building a provider request, not an MCP JSON response. The pipeline's own token budget system (`DefaultContextWindowTokens = 200_000`, warn at 40%, refuse at 60%) is the appropriate mechanism and should be recalibrated to current model windows (1M tokens).

**For P51 specifically:** The `trimmed` metadata in `next`/`handoff` responses should include topic-level detail (the topic string and whether the entry was tier-2 or tier-3) so the orchestrator can assess impact. Without this, trimming is invisible data loss.

### Repeated context across task claims

Each `next` call during P50 returned ~30KB of context, most of it identical across the 12 tasks (the same ~50 knowledge entries, the same implementation guidance, the same tool subset). The orchestrator called `next` 12 times and received essentially the same knowledge base 12 times.

This is a structural inefficiency: context assembly is per-task-claim, not per-session. If the orchestrator's context were assembled once per session and task claims were lightweight (task ID + spec section + file scope only), the token waste would be substantially lower. This is deferred to P44, where `dispatch_task` can maintain session-scoped context and only assemble task-specific sections per claim.

### finish summary limit (second retrospective)

The `finish` tool enforces a 500-character summary limit. The error message states the limit but it is not documented in the tool description or the handler flow. During fast-track close-out, the orchestrator hit this twice and had to learn the limit by failing.

**For P51:** Add the 500-character limit to the `finish` tool description. The current description says "Brief description of what was accomplished" — it should say "Brief description of what was accomplished (max 500 characters)." This is a one-line documentation change.

## Open Questions

1. Should `next` also be unified in this plan, or left for a follow-up? (Recommend: follow-up — `next` has its own output format and callers.)
2. Should the `handoff` tool rename to reflect that it's now pipeline-only? (Recommend: no — the tool name is stable; it's the implementation that's changing.)
3. **Trimming metadata:** Should `trimmed` entries include topic strings? (Recommend: yes — add a `topic` field to `asmTrimmedEntry` so the orchestrator can see what was dropped, not just how much. Low implementation cost, high debugging value.)
4. **finish summary limit:** Should the 500-character limit be documented in the tool description? (Recommend: yes — add "(max 500 characters)" to the summary parameter description. Also consider surfacing the limit in the error message itself so the orchestrator doesn't need to read tool descriptions to learn it.)

4. **edit_file reliability:** The `edit_file` tool failed repeatedly on design document sections during this session, requiring fallback to `python3` and `sed` for basic text insertion. The `old_text` matching could not find text that was demonstrably present in the file. This may be a token-limit issue on `old_text` or a fuzzy-matching bug. Needs investigation — design documents are a primary artifact type and should be editable with the primary editing tool.
5. **stale MCP binary:** The running `kbz serve` binary showed `git_sha: unknown` and a different path than the install record. The Makefile produces `kbz` but the editor MCP config expects `kanbanzai`. This mismatch means `go install` never updates the running binary. The `server_info` tool detects the problem but doesn't fix it. Consider adding a `rebuild` or `restart` capability, or aligning the binary names.
6. **test compilation errors:** `internal/mcp/` has pre-existing test compilation errors (redeclared functions, wrong `newServerWithConfig` signatures) from recent merges. These block running new tests and would block CI. They need to be fixed before any handoff/assembly changes are testable.
7. **plan numbering reuse:** Creating plans via `entity(action: "create", type: "strategic-plan")` reuses stale P1 numbers. P51 and P52 were both assigned P1, requiring manual state file renaming. Root cause: `listAllPlanIDs` scans `s.List("plan")` which reads batch plans, not strategic plans. The `NextPlanNumber` scan needs to include both plan types.
