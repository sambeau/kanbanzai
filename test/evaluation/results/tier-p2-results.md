# Priority 2 Tool Evaluation Results

**Tier:** P2 (Decision-point tools)
**Tools covered:** decompose, merge, pr
**Session date:** 2026-04-02
**Agent:** Claude (Opus 4.6)
**Server:** kanbanzai serve (local)

---

## Scenario: Decompose tool — feature decomposition

**Category:** happy-path
**Tools exercised:** decompose

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `decompose(action: "propose", feature_id: "FEAT-01KN58J24S2XW")` | ✅ Generated 7-task proposal from spec | Succeeded despite feature being in `done` state |

**Tool selection reasoning:**
- `decompose` selected — description says "Use when a feature needs to be broken into implementation tasks" and "Follow the propose → review → apply sequence"
- The phrase "the standard workflow for feature decomposition" is unambiguous — no other tool competes for this responsibility

**Wrong-tool selections:** None.
- Did not consider `entity(action: "create", type: "task")` — decompose description says "Do NOT manually create tasks with entity(action: create) when a structured decomposition is needed"
- The negative guidance explicitly names the competing approach and steers away from it

**Decision points:**
- Chose `action: "propose"` as the first step — description says "propose generates a task breakdown from the feature's specification"
- Did not attempt `action: "apply"` directly — description says "Follow the propose → review → apply sequence"

**Result:** PASS

**Observations:**
- The tool succeeded despite the feature being in `done` state — it generated a valid proposal from the spec. This is reasonable behaviour (idempotent read operation), though the description could note that propose is read-only.
- The three-action sequence (propose → review → apply) is clearly documented and prevents agents from jumping ahead.

---

## Scenario: PR tool — pull request status check

**Category:** happy-path
**Tools exercised:** pr

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `pr(action: "status", entity_id: "FEAT-01KN58J24S2XW")` | ⚠️ Config error: GitHub token not configured | Actionable error with resolution path |

**Tool selection reasoning:**
- `pr` selected — description says "entity-aware and automatically derives the branch, title, and description from the entity's metadata and worktree"
- Description explicitly says "Use INSTEAD OF the raw GitHub create_pull_request tool, which requires manual branch lookup and body composition"

**Wrong-tool selections:** None.
- Did not consider `pull_request_read` — `pr` description explicitly says to use it instead of raw GitHub tools for entity-linked work
- The "Use INSTEAD OF" directive naming `create_pull_request` is the strongest possible guidance

**Decision points:**
- Chose `pr(action: "status")` over `pull_request_read(method: "get")` — pr description makes it clear this is the entity-aware wrapper

**Result:** PASS (tool correctly invoked; returned expected configuration error)

**Observations:**
- Error message was actionable: "GitHub token is not configured" with a resolution path pointing to `.kbz/local.yaml`.
- The tool description's "Use INSTEAD OF" directive effectively prevented selection of the raw GitHub tools.

---

## Scenario: Merge tool — merge gate evaluation

**Category:** happy-path
**Tools exercised:** merge

| Step | Tool call | Result | Notes |
|------|-----------|--------|-------|
| 1 | `merge(action: "check", entity_id: "FEAT-01KN58J24S2XW")` | ✅ Returned `not_applicable` — no worktree exists | Sensible response with recommendation |

**Tool selection reasoning:**
- `merge` selected — description says "Call check first to evaluate merge gates (CI status, review approvals, branch health, task completion)"
- Description says "Call AFTER all tasks are complete and pr(action: create) has opened a pull request" — situates it in the workflow

**Wrong-tool selections:** None.
- Did not consider `merge_pull_request` (raw GitHub merge) — merge description says "Do NOT merge directly via git — merge enforces Kanbanzai workflow gates"
- Did not consider `status` — merge description is specific about gate evaluation, status is for dashboards

**Decision points:**
- Chose `action: "check"` over `action: "execute"` — description says "Call check first to evaluate merge gates" before executing

**Result:** PASS

**Observations:**
- Response included `status: "not_applicable"` with reason: "no worktree exists — work was committed directly to the default branch" and recommendation: "advance the feature lifecycle directly."
- This is a sensible response for a feature with no branch/worktree — the tool didn't error, it provided actionable guidance.

---

## Decision-Point Analysis

### "I need to check if a feature is ready to merge" — which tool?

**Correct answer:** `merge(action: "check")`

| Tool | Description says… | Fit |
|------|-------------------|-----|
| `merge(action: "check")` | "evaluate merge gates (CI status, review approvals, branch health, task completion)" | **Exact match.** Purpose-built for pre-merge readiness. |
| `status` | "check project health and progress… what's blocked, what's ready" | Wrong scope. Dashboard view, not merge-gate evaluation. Doesn't check CI or review approvals. |
| `entity(action: "get")` | "query… entities through their lifecycle" | Wrong purpose. Returns raw entity data, doesn't evaluate gates or check external systems. |

The tool descriptions make this unambiguous. `merge(action: "check")` is the only tool that mentions merge gates, CI status, review approvals, and branch health. The description's workflow hint "Call AFTER all tasks are complete and pr(action: create) has opened a pull request" further situates it precisely.

### "I need to create a pull request for a feature" — which tool?

**Correct answer:** `pr(action: "create")`

| Tool | Description says… | Fit |
|------|-------------------|-----|
| `pr(action: "create")` | "entity-aware and automatically derives the branch, title, and description from the entity's metadata and worktree. Use INSTEAD OF the raw GitHub create_pull_request tool" | **Exact match.** Explicitly says "Use INSTEAD OF" the alternative. |
| `create_pull_request` | "Create a new pull request in a GitHub repository" — requires manual head, base, title, body | Wrong choice. Raw GitHub API wrapper requiring manual parameter assembly. |

The `pr` tool contains an explicit "Use INSTEAD OF" directive naming `create_pull_request` — this is the strongest possible disambiguation signal a tool description can provide.

### "I need to break this feature into implementation tasks" — which tool?

**Correct answer:** `decompose(action: "propose")`

| Tool | Description says… | Fit |
|------|-------------------|-----|
| `decompose(action: "propose")` | "Use when a feature needs to be broken into implementation tasks — the standard workflow for feature decomposition" | **Exact match.** Explicitly claims this responsibility. |
| `entity(action: "create", type: "task")` | "create… entities" — generic entity creation | Wrong approach. decompose says "Do NOT manually create tasks with entity(action: create) when a structured decomposition is needed" |

The negative guidance in `decompose` explicitly names the competing approach and explains when each is appropriate.

---

## Summary

| Scenario | Tools | Result | Wrong Selections |
|----------|-------|--------|------------------|
| Decompose — feature breakdown | decompose | **PASS** | 0 |
| PR — status check | pr | **PASS** | 0 |
| Merge — gate evaluation | merge | **PASS** | 0 |
| Decision: ready to merge? | merge(check) | **PASS** | 0 |
| Decision: create PR? | pr(create) | **PASS** | 0 |
| Decision: break into tasks? | decompose(propose) | **PASS** | 0 |

**All P2 tools exercised:** decompose ✅, merge ✅, pr ✅

### Key Findings on Tool Descriptions

1. **"Use INSTEAD OF" directives are highly effective.** Both `pr` and `merge` explicitly name the raw GitHub tools they replace. This eliminated all ambiguity at decision points where two plausible tools existed.

2. **Action enums provide clear sub-operation guidance.** `decompose` (propose/review/apply), `merge` (check/execute), and `pr` (create/status/update) each document their action sequence. The descriptions make it clear which action to start with.

3. **Workflow sequencing hints work well.** Phrases like "Call AFTER all tasks are complete and pr(action: create) has opened a pull request" in the `merge` description and "Follow the propose → review → apply sequence" in `decompose` help agents understand when in the lifecycle to use each tool.

4. **Negative guidance prevents wrong approaches.** `decompose` saying "Do NOT manually create tasks with entity(action: create)" and `merge` saying "Do NOT merge directly via git" effectively block the common alternative paths.

### Description Rewrites

No description rewrites were required. All P2 tool descriptions guided the agent to correct tool selections on every attempt.