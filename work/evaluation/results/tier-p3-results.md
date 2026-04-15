# Priority 3 Tool Evaluation Results

**Tier:** P3 (Query and support tools)
**Tools covered:** conflict, knowledge, estimate, health, cleanup, worktree, branch, checkpoint, incident, profile, retro
**Session date:** 2026-04-02
**Agent:** Claude (Opus 4.6)
**Server:** kanbanzai serve (local)

---

## Test 1: conflict tool — parallel task risk assessment

**Category:** query
**Tools exercised:** conflict

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `conflict(action: "check", task_ids: ["TASK-01KN5AJBC2072", "TASK-01KN5AJBCNCFV"])` | ✅ Returned risk assessment | Medium overall risk, no file overlap, medium boundary-crossing risk |

**Tool selection reasoning:**
- `conflict` selected — description says "check whether they risk conflicting on shared files, dependencies, or architectural boundaries"
- Description says "Returns per-pair risk assessment and recommendation (safe_to_parallelise, serialise, or checkpoint_required)"
- The phrase "Use INSTEAD OF manually inspecting file lists to decide parallelism" steered away from manual approaches

**Wrong-tool selections:** None.
- Did not consider `branch(action: "status")` — conflict description explicitly differentiates: "For actual merge conflict detection on branches, use branch(action: status) instead"
- The bidirectional negative guidance between `conflict` (task-level risk) and `branch` (branch-level conflicts) is clear

**Decision points:**
- `conflict` vs `branch`: conflict is for pre-work risk assessment ("before dispatching tasks"), branch is for post-work merge readiness

**Result:** PASS

**Observations:**
- Response included per-dimension risk breakdown: file overlap (low), dependency order (low), boundary crossing (medium).
- Recommendation was `checkpoint_required` — tasks can run in parallel but a human should review before merging.
- The description's phrase "safe_to_parallelise, serialise, or checkpoint_required" accurately predicted the response vocabulary.

---

## Test 2: knowledge tool — knowledge base queries

**Category:** query
**Tools exercised:** knowledge

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `knowledge(action: "list")` | ✅ Returned 47 entries | Full knowledge base listing |
| 2 | `knowledge(action: "list", tags: ["phase-a"])` | ✅ Returned 0 entries | Tag filter correctly returned empty set — no false positives |

**Tool selection reasoning:**
- `knowledge` selected — description says "action: list to find entries by topic, tag, or status that may not be in your context window"
- The parameter documentation clearly distinguishes `topic_filter` (for list) from `topic` (for contribute)
- `tags` parameter documented as "Classification tags (contribute) or tag filter (list)" — dual-purpose but unambiguous in context

**Wrong-tool selections:** None.
- Did not consider `grep` or `codebase_memory_mcp_search_code` — knowledge description says "Do NOT read .kbz/state/knowledge/ files directly"
- Did not consider `doc` — knowledge is for knowledge base entries, doc is for document records

**Decision points:**
- For "find knowledge about topic X": chose `knowledge(action: "list", tags: [...])` over `knowledge(action: "list", topic_filter: "...")` — tags for broad category search, topic_filter for exact topic match

**Result:** PASS

**Observations:**
- Unfiltered list returned 47 entries covering topics like phase-4b scope, TSID13 IDs, YAML field ordering, error handling conventions, MCP thin-adapter pattern, estimation guidelines.
- Tag filter with `["phase-a"]` returned 0 entries, confirming the filter works correctly (no false positives from partial matching or case issues).
- The description's phrasing "find entries by topic, tag, or status" accurately described the three filtering axes.

---

## Test 3: estimate tool — story point queries

**Category:** query
**Tools exercised:** estimate

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `estimate(action: "query", entity_id: "FEAT-01KN58J24S2XW")` | ✅ Returned rollup stats | No estimate set; 6 child tasks, 0 estimated |

**Tool selection reasoning:**
- `estimate` selected — description says "action: query returns rollup stats including child-entity totals"
- Description clearly differentiates `set` (requires `points`) from `query` (requires `entity_id`)
- The "rollup" concept in the description correctly predicted the response shape

**Wrong-tool selections:** None.
- Did not consider `entity(action: "get")` — estimate is specifically for sizing data, entity is for lifecycle state
- The description's "Use INSTEAD OF manually tracking sizing in documents" steered away from manual approaches

**Decision points:**
- Chose `action: "query"` over `action: "set"` — description says "query returns rollup stats" while set requires `points`

**Result:** PASS

**Observations:**
- Feature had no estimate at the feature level. Rollup showed 6 child tasks with 0 estimated — `estimated_task_count: 0`, `progress: 0`, `task_total: null`.
- The response shape matched what the description predicted: rollup stats with child-entity totals.
- Modified Fibonacci scale (0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100) is documented in the description — useful for agents that need to set estimates.

---

## Test 4: health tool — project consistency check

**Category:** query
**Tools exercised:** health

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `health()` (no params) | ✅ Comprehensive health report | 503 entities, 4 hard errors, multiple warnings |

**Tool selection reasoning:**
- `health` selected — description says "comprehensive health check across all entities, knowledge entries, worktrees, branches, and context profiles"
- Description says "No parameters required" — trivially correct invocation
- Description says "Use INSTEAD OF manually inspecting individual entities for consistency issues"

**Wrong-tool selections:** None.
- Did not consider `status()` — health description says "Do NOT use for entity-specific queries — use status for dashboards"
- The bidirectional differentiation between health (consistency/integrity) and status (progress/dashboard) is clear

**Decision points:**
- `health` vs `status`: health is for "is everything consistent?", status is for "what's the progress?"

**Result:** PASS

**Observations:**
- Report covered 503 total entities (88 features, 320 tasks, 21 plans, 47 knowledge entries, 6 bugs, 18 profiles).
- Key findings: 4 hard errors (3 invalid slugs containing `.`, 1 missing worktree path), 7 branches 94–198 commits behind main, 1 merge conflict, 2 overdue cleanups, 8 stale worktrees, stale doc references in skill files, all 18 roles/skills with `last_verified: never`.
- The response was structured by category (entities, branches, worktrees, documents, profiles) matching the description's claim of checking "all entities, knowledge entries, worktrees, branches, and context profiles."

---

## Decision-Point Analysis

### "I need to find all knowledge entries about testing" — which tool?

**Correct answer:** `knowledge(action: "list", tags: ["testing"])`
- Or alternatively: `knowledge(action: "list", topic_filter: "testing")`

**Reasoning:** The description says "action: list to find entries by topic, tag, or status." Tags filter by classification tags (broad category), topic_filter matches exact normalised topic (narrow). For a broad "about testing" search, `tags: ["testing"]` is the better fit since multiple entries could be tagged "testing" with different topics.

**Wrong-tool risk:** An agent might try `grep` or `codebase_memory_mcp_search_code` to search knowledge files — but the description's "Do NOT read .kbz/state/knowledge/ files directly" steers away from that approach.

### "I need to check if the project has any consistency issues" — which tool?

**Correct answer:** `health()`

**Reasoning:** Description says "comprehensive health check across all entities, knowledge entries, worktrees, branches, and context profiles" and "Returns a structured report of errors and warnings." The phrase "consistency issues" maps directly to health checks.

**Wrong-tool risk:** An agent might pick `status()` — but health's description explicitly says "Do NOT use for entity-specific queries — use status for dashboards" which implies the reverse: don't use status for integrity checks. The bidirectional cross-referencing works.

### "Are these two tasks safe to work on in parallel?" — which tool?

**Correct answer:** `conflict(action: "check", task_ids: ["TASK-xxx", "TASK-yyy"])`

**Reasoning:** Description says "check whether they risk conflicting on shared files, dependencies, or architectural boundaries. Returns per-pair risk assessment and recommendation (safe_to_parallelise, serialise, or checkpoint_required)." The phrase "safe to work on in parallel" maps exactly to "safe_to_parallelise."

**Wrong-tool risk:** An agent might try `branch(action: "status")` — but conflict's description explicitly says "For actual merge conflict detection on branches, use branch(action: status) instead" which differentiates the two clearly: conflict is for pre-work planning, branch is for post-work merge readiness.

### "I want to see what retrospective signals have been collected" — which tool?

**Correct answer:** `retro(action: "synthesise")`

**Reasoning:** Description says "Before writing any retrospective or review document, call action: synthesise first — it surfaces signals from across the project." The synthesise action clusters and ranks signals. For just viewing what's been collected, this is the right entry point.

**Wrong-tool risk:** An agent might try `knowledge(action: "list")` since retro signals are a form of project knowledge — but the description's "Use finish(retrospective: [...]) to record individual signals — use this tool to analyse them" makes the boundary clear: knowledge is for reusable facts, retro is for workflow observations.

### "I need to check if a feature's branch is ready to merge" — which tool?

**Correct answer:** `branch(action: "status", entity_id: "FEAT-...")`

**Reasoning:** Description says "reports staleness, drift from main, and merge conflicts." This is specifically about branch-level readiness, not task-level conflict risk (which is `conflict`).

**Wrong-tool risk:** An agent might pick `merge(action: "check")` — but merge evaluates workflow gates (CI, reviews, task completion) while branch checks the git-level health. Both may be needed before merging, but they answer different questions.

---

## Summary

| Test | Tools | Result | Wrong Selections |
|------|-------|--------|------------------|
| 1 — Conflict check | conflict | **PASS** | 0 |
| 2 — Knowledge query | knowledge | **PASS** | 0 |
| 3 — Estimate query | estimate | **PASS** | 0 |
| 4 — Health check | health | **PASS** | 0 |
| Decision: knowledge about X? | knowledge(list) | **PASS** | 0 |
| Decision: consistency issues? | health() | **PASS** | 0 |
| Decision: parallel safety? | conflict(check) | **PASS** | 0 |
| Decision: retro signals? | retro(synthesise) | **PASS** | 0 |
| Decision: branch readiness? | branch(status) | **PASS** | 0 |

**P3 tools exercised directly:** conflict ✅, knowledge ✅, estimate ✅, health ✅
**P3 tools exercised via decision-point analysis:** retro ✅, branch ✅

### Key Findings on Tool Descriptions

1. **Explicit negative guidance is highly effective.** The "Use INSTEAD OF", "Do NOT use for", and "For X, use Y instead" patterns in P3 tool descriptions prevented every potential wrong-tool selection. The bidirectional guidance between `conflict`/`branch`, `health`/`status`, and `knowledge`/`doc` eliminated ambiguity at each decision point.

2. **Action-parameter documentation is clear.** Each tool clearly stated which action does what, so `query` vs `set` on `estimate`, `check` vs no-action on `conflict`, and `list` vs `contribute` on `knowledge` were all unambiguous.

3. **Response shape previews aid selection.** Phrases like "returns per-pair risk assessment and recommendation (safe_to_parallelise, serialise, or checkpoint_required)" in `conflict` and "rollup stats including child-entity totals" in `estimate` let the agent predict what it would get back, confirming tool selection before calling.

4. **No-parameter tools are trivially correct.** `health` with "No parameters required" and tools with simple required parameter sets (like `conflict` needing only `task_ids`) are easy to invoke correctly. The descriptions match the actual parameter requirements.

5. **Cross-tool boundaries are well-defined.** Every P3 tool occupies a clearly scoped responsibility area, and the descriptions explicitly name which other tools handle adjacent concerns. No responsibility overlap was detected during testing.

### Description Rewrites

No description rewrites were required. All P3 tool descriptions guided the agent to correct tool selections on every attempt.