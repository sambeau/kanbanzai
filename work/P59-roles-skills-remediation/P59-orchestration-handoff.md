# P59 Orchestration Handoff

**Session date:** 2026-05-08  
**Status:** Blocked — `handoff` tool timing out; implementation dispatch not yet started  
**Plan:** P59-roles-skills-remediation  
**Batches:** B58, B59, B60, B61, B62 (all `active`)

---

## What was accomplished this session

1. **All 5 specs approved** (batch `doc(action: approve, ids: [...])`) — auto-cascaded all features from `specifying` → `dev-planning`

2. **5 dev-plans written and tasks decomposed** — 5 architect sub-agents ran in parallel, each wrote a dev-plan, registered + approved it, and created tasks manually (decompose propose was not usable — AC format incompatibility with `**AC-001 (REQ-001):**` style). All features auto-advanced to `developing`.

3. **5 worktrees created** — one per feature, all `active`.

4. **15 Wave 1 tasks claimed** via `next()` — all transitioned `ready → active`. These are the ready-frontier tasks (no unmet dependencies). They remain in `active` status awaiting implementation.

5. **Dispatch blocked** — `handoff` tool times out on every call (single, batched, sequential). Root cause under investigation. Server was rebuilt and restarted with no improvement. The `status()`, `entity()`, `server_info`, and `index_status` tools all respond normally.

---

## Feature state

| Feature | ID | Status | Worktree branch |
|---------|-----|--------|-----------------|
| B58 — Constraint Card & Stage-Binding Hydration | `FEAT-01KR3MDJ7AV37` | developing | `feature/FEAT-01KR3MDJ7AV37-constraint-card-stage-binding-hydration` |
| B59 — High-Violation MCP Rule Invariants | `FEAT-01KR3MDSZKAFG` | developing | `feature/FEAT-01KR3MDSZKAFG-high-violation-mcp-rule-invariants` |
| B60 — Generated Role/Skill Registry Surfaces | `FEAT-01KR3MEJGGMT5` | developing | `feature/FEAT-01KR3MEJGGMT5-generated-role-skill-registry-surfaces` |
| B61 — Instruction Corpus Cleanup | `FEAT-01KR3MES5ZJ11` | developing | `feature/FEAT-01KR3MES5ZJ11-instruction-corpus-cleanup` |
| B62 — Runtime Discovery Surfaces | `FEAT-01KR3MEYRQ9RG` | developing | `feature/FEAT-01KR3MEYRQ9RG-runtime-discovery-surfaces` |

---

## Task graph — all features

### B58 · FEAT-01KR3MDJ7AV37 · Constraint Card and Stage-Binding Hydration

Spec: `work/B58-inject-constraint-card-stage-binding-hydration/B58-F1-spec-constraint-card-stage-binding-hydration.md`  
Dev-plan: `work/B58-inject-constraint-card-stage-binding-hydration/plan/B58-F1-dev-plan.md`

```
Wave 1 (parallel, both ACTIVE):
  T1  TASK-01KR3PY208N1W  Constraint Registry model loader and initial YAML      3 sp
  T3  TASK-01KR3PYCQ2E6W  Stage-Binding Hydration Payload extractor              2 sp

Wave 2 (after T1):
  T2  TASK-01KR3PYR0BR41  Constraint Card Renderer                               5 sp

Wave 3 (parallel, after T2+T3):
  T4  TASK-01KR3PZ4PJEJA  Inject constraint card and stage binding into next      3 sp
  T5  TASK-01KR3PZJNS6GS  Inject constraint card and stage binding into handoff   3 sp

Wave 4 (after T4+T5):
  T6  TASK-01KR3PZWD77W3  Golden tests validation suite and regression tests      5 sp
```

Critical path: T1 → T2 → T4 → T6

---

### B59 · FEAT-01KR3MDSZKAFG · High-Violation MCP Rule Invariants

Spec: `work/B59-enforce-high-violation-rules-mcp-invariants/B59-F1-spec-high-violation-mcp-rule-invariants.md`  
Dev-plan: `work/B59-enforce-high-violation-rules-mcp-invariants/plan/B59-F1-dev-plan.md`

```
Wave 1 (ACTIVE — sole gate):
  T1  TASK-01KR3Q03TDBQE  Define the invariant catalog package                    3 sp
      internal/invariants/catalog.go — five codes INV-001..005, RefusalResponse, Format()

Wave 2 (parallel, after T1):
  T2  TASK-01KR3Q0ANRJ6K  Enforce registered-entity invariant in next+handoff     2 sp
  T3  TASK-01KR3Q0CQW50P  Enforce orphaned-workflow-state invariant in next        3 sp
  T4  TASK-01KR3Q0FB91YP  Add shell-read warning to task-context assembly          2 sp
  T5  TASK-01KR3Q0W95KWT  Prose de-duplication in orchestrator role and skill      2 sp

Wave 3 (after T2+T3+T4+T5):
  T6  TASK-01KR3Q16BFSES  Invariant boundary tests                                 5 sp
```

Critical path: T1 → any of T2–T5 → T6  
**Note:** T5 removes `spawn_agent` from `orchestrator.yaml`. T6 test for AC-002 should be `pending` until P44's `dispatch_task` is registered.

---

### B60 · FEAT-01KR3MEJGGMT5 · Generated Role and Skill Registry Surfaces

Spec: `work/B60-unify-role-skill-registries/B60-F1-spec-generated-role-skill-registry-surfaces.md`  
Dev-plan: `work/B60-unify-role-skill-registries/plan/B60-F1-dev-plan.md`

```
Wave 1 (parallel, both ACTIVE — no shared types):
  T1  TASK-01KR3PVZZ72EP  Registry extraction model — extractor and model
      internal/registry/model.go, extractor.go, extractor_test.go
  T2  TASK-01KR3PWF4DY61  Markdown region parser and writer
      internal/registry/region.go, region_test.go

Wave 2 (after T1):
  T3  TASK-01KR3PWT48NSD  Registry content renderer

Wave 3 (after T1+T2+T3):
  T4  TASK-01KR3PX9J6DW5  kbz docs CLI command — sync and check subcommands

Wave 4 (after T4):
  T5  TASK-01KR3PXKS1BZ5  Marker placement, initial sync, and CI target
```

Critical path: T1 → T3 → T4 → T5

---

### B61 · FEAT-01KR3MES5ZJ11 · Instruction Corpus Cleanup

Spec: `work/B61-tidy-contradictions-stray-files-corpus-size/B61-F1-spec-instruction-corpus-cleanup.md`  
Dev-plan: `work/B61-tidy-contradictions-stray-files-corpus-size/plan/B61-F1-dev-plan.md`

```
Wave 1 (all 5 fully parallel, all ACTIVE — strictly disjoint files):
  T1  TASK-01KR3PZCRF7BD  Replace stray SKILL.md files with directory indexes      1 sp
      .kbz/skills/SKILL.md, .agents/skills/SKILL.md, two README files
  T2  TASK-01KR3PZCRGN24  Align top-level instruction files on no-stash rule       1 sp
      .github/copilot-instructions.md
  T3  TASK-01KR3PZCRJEPB  Reduce over-budget skills and relocate long examples     3 sp
      orchestrate-development/SKILL.md, kanbanzai-agents/SKILL.md,
      internal/kbzinit/skills/agents/SKILL.md (dual-write), reference files
  T4  TASK-01KR3PZCRKKTC  Audit and fix base role tool access                      2 sp
      .kbz/roles/base.yaml + role files lacking explicit grep/search_graph
  T5  TASK-01KR3PZCRM6GW  De-duplicate anti-pattern prose between skills/roles     2 sp
      skill SKILL.md files with duplicated anti-pattern prose

Wave 2 (after all T1–T5):
  T6  TASK-01KR3PZCRNGQJ  Measure corpus size and verify invariants                1 sp
```

---

### B62 · FEAT-01KR3MEYRQ9RG · Runtime Discovery Surfaces

Spec: `work/B62-discover-runtime-instruction-surfaces/B62-F1-spec-runtime-discovery-surfaces.md`  
Dev-plan: `work/B62-discover-runtime-instruction-surfaces/plan/B62-F1-dev-plan.md`

```
Wave 1 (all independent, all ACTIVE):
  T1  TASK-01KR3PZXXR68X  Create .claude/skills/ wrappers and generator            3 sp
      7 wrapper files + generator script under .claude/skills/
  T2  TASK-01KR3Q0ASDH5N  Create OPENAI.md redirect to AGENTS.md                  1 sp
      OPENAI.md at repo root, <=20 lines
  T3  TASK-01KR3Q0BNMN27  Add P59 rule text to Kanbanzai tool descriptions         2 sp
      next_tool.go, handoff_tool.go, entity_tool.go, worktree_tool.go, pr_tool.go
  T4  TASK-01KR3Q0FHXC05  Document DeepSeek host loading behaviour                 2 sp
      refs/sub-agents.md — DeepSeek section
  T6  TASK-01KR3Q0TF64S0  Create Cursor rules shim (optional)                      1 sp
      .cursor/rules/kanbanzai.mdc

Wave 2 (after T1):
  T5  TASK-01KR3Q0TB4VS7  Implement wrapper drift check CI target                   2 sp
```

Critical path: T1 → T5  
**Note on T3:** Use `TODO(P59-B2)` comment if invariant codes from B59-T1 are not yet merged.

---

## The handoff tool timeout issue

**Symptom:** `handoff(task_id, role: "implementer-go")` returns `Context server request timeout` on every call — single, batched, parallel, sequential. Server rebuilt and restarted: no improvement. All other MCP tools respond normally.

**Suspected causes (in priority order):**

1. **Conflict detection O(n²) overload** — 15 tasks simultaneously `active` across 5 features. The conflict checker may be running ~105 pair-wise code graph comparisons. Most likely cause.

2. **Doc-intel spec section loading timeout** — specs and dev-plans registered in the same session; doc_intel index may not have processed them, causing the pipeline to hang waiting for doc content.

3. **Knowledge assembly size** — `next()` calls returned 64KB each (at `byte_budget` ceiling); `handoff` assembles even more context on top.

**What to check in server logs:**
- Which pipeline step hangs (spec section loading, knowledge assembly, code graph traversal, or conflict check)
- Current timeout threshold (30s? 60s?)

---

## Resumption checklist

When `handoff` is fixed, resume here:

1. Verify fix: `handoff(task_id: "TASK-01KR3Q03TDBQE", role: "implementer-go")` should return a prompt without timeout
2. Run `status(id: "FEAT-01KR3MDSZKAFG")` to confirm the 15 Wave 1 tasks are still `active` (no state drift from the break)
3. Dispatch all 15 Wave 1 tasks in parallel: `handoff` → `spawn_agent` for each
4. As each Wave 1 task completes, immediately dispatch newly-unblocked Wave 2 tasks per the dependency graphs above
5. After all tasks terminal: dispatch one `reviewer-conformance` agent per feature
6. Transition each feature: `reviewing` → `done`, merge branches, remove worktrees, produce completion report

**Wave 1 tasks to dispatch (all currently `active`):**

| Task ID | Feature | Summary |
|---------|---------|---------|
| `TASK-01KR3PY208N1W` | B58 | Constraint Registry model loader |
| `TASK-01KR3PYCQ2E6W` | B58 | Stage-Binding Hydration Payload extractor |
| `TASK-01KR3Q03TDBQE` | B59 | Invariant catalog package (critical blocker) |
| `TASK-01KR3PVZZ72EP` | B60 | Registry extraction model |
| `TASK-01KR3PWF4DY61` | B60 | Markdown region parser and writer |
| `TASK-01KR3PZCRF7BD` | B61 | Replace stray SKILL.md files |
| `TASK-01KR3PZCRGN24` | B61 | Align top-level instruction files |
| `TASK-01KR3PZCRJEPB` | B61 | Reduce over-budget skills |
| `TASK-01KR3PZCRKKTC` | B61 | Audit and fix base role tool access |
| `TASK-01KR3PZCRM6GW` | B61 | De-duplicate anti-pattern prose |
| `TASK-01KR3PZXXR68X` | B62 | .claude/skills/ wrappers and generator |
| `TASK-01KR3Q0ASDH5N` | B62 | OPENAI.md redirect |
| `TASK-01KR3Q0BNMN27` | B62 | Tool description P59 rule text |
| `TASK-01KR3Q0FHXC05` | B62 | DeepSeek host loading documentation |
| `TASK-01KR3Q0TF64S0` | B62 | Cursor rules shim (optional) |
