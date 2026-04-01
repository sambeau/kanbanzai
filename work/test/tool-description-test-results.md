# Agent-Driven Test Results: MCP Tool Descriptions

## Overview

This file records the results of agent-driven testing on rewritten MCP tool
descriptions. Each priority tier's descriptions are validated by presenting
scenarios to an agent with only the tool descriptions for guidance, then
recording whether the agent selects the correct tool sequence.

**Testing process (FR-013):**
1. Present an agent with a scenario task and only the MCP tool list (with descriptions)
2. Record which tools the agent selects and where it goes wrong
3. If the agent mis-selects, rewrite the description and re-test with a fresh session
4. Continue until the agent selects the correct sequence

---

## Priority 1 — High-Frequency Tools

**Tools tested:** `entity`, `doc`, `handoff`, `next`, `finish`, `status`
**Date:** 2026-04-01
**Scenarios used:** 1, 2, 4, 5, 7 (primary P1 coverage); 3, 6, 8 (P1 tools alongside P2)

### Scenario 1: Advance a feature from specifying to dev-planning

**Result:** PASS

**Observed tool sequence:**
1. `status(id: "FEAT-01ABC")` ✅
2. `entity(action: "transition", id: "FEAT-01ABC", status: "dev-planning")` ✅

**Decision point analysis:**
- **`status` vs `entity(action: "get")`:** The `status` description says "The primary way to check project health and progress" and returns "synthesised dashboards with lifecycle status, attention items." The `entity` description says "For synthesised dashboard views (progress, attention items, what's blocked), use status instead." Both descriptions clearly steer toward `status` for this use case.

---

### Scenario 2: Claim the next available task and complete it

**Result:** PASS

**Observed tool sequence:**
1. `next()` ✅
2. `next(id: "TASK-...")` ✅
3. *(work performed)*
4. `finish(task_id: "TASK-...", summary: "...")` ✅

**Decision point analysis:**
- **`next()` vs `entity(action: "list")`:** The `next` description says "Start here when beginning work — the primary way to find and claim tasks" and "Use INSTEAD OF manually querying entities with entity(action: \"list\")." This explicit negative guidance eliminates ambiguity.
- **`finish` vs `entity(action: "transition")`:** The `finish` description says "Use when a task is done to record completion and contribute knowledge" and "Do NOT use for other status changes — use entity(action: \"transition\") instead." The bidirectional guidance (finish says when to use entity, and the description clearly positions finish as the completion-specific tool) makes this unambiguous.

---

### Scenario 4: Write a spec document and register it against a feature

**Result:** PASS

**Observed tool sequence:**
1. `entity(action: "get", id: "FEAT-03DEF")` ✅
2. *(write spec file)*
3. `doc(action: "register", path: "...", type: "specification", title: "...", owner: "FEAT-03DEF")` ✅

**Decision point analysis:**
- **`doc(action: "register")` vs `doc(action: "import")`:** The `doc` description says "Call register after writing a document" and "For register: path, type, title required." The parameter description for `path` says "register: required; ... import: directory to scan" — register is for a single file, import scans a directory. Clear enough.
- **`entity(action: "get")` vs `status` for initial context:** The agent needs raw field data (summary, slug) to write the spec. `entity` says "use this whenever you need to create, query, modify..." while `status` says "For modifying entity state, use entity instead." An agent needing a specific entity record would correctly choose `entity get`.

---

### Scenario 5: Find all blocked work across the project

**Result:** PASS

**Observed tool sequence:**
1. `status()` ✅
2. `next()` ✅
3. `entity(action: "list", type: "task", status: "blocked")` ✅

**Decision point analysis:**
- **`status()` vs `entity(action: "list")` for overview:** The `status` description says "Use INSTEAD OF reading .kbz/state/ files or using entity(action: list) for overview queries." This is an explicit negative-guidance match — the description directly names `entity(action: list)` as the wrong tool for overview queries.
- **`next()` for ready tasks:** The `next` description says "Start here when beginning work" and "Call without id to inspect the work queue (all ready tasks sorted by priority)." For seeing what's ready to pick up, this is clearly positioned.

---

### Scenario 7: Delegate a task to a sub-agent and record completion

**Result:** PASS

**Observed tool sequence:**
1. `next(id: "TASK-05JKL")` ✅
2. `handoff(task_id: "TASK-05JKL", role: "backend")` ✅
3. *(sub-agent work)*
4. `finish(task_id: "TASK-05JKL", summary: "...", files_modified: [...], knowledge: [...])` ✅

**Decision point analysis:**
- **`handoff` vs `entity(action: "get")` for sub-agent prompt:** The `handoff` description says "Use when delegating a task to a sub-agent — generates a complete, ready-to-use prompt." This is the exact use case. No ambiguity.
- **`next(id)` vs `entity(action: "transition")`:** The `handoff` description says "Call AFTER next(id) claims the task, BEFORE spawn_agent dispatches the sub-agent." This explicit workflow sequencing makes `next` the clear first step. The `next` description reinforces: "Call BEFORE handoff when delegating to sub-agents."

---

### Scenario 3: Decompose a feature into tasks (P1 + P2)

**P1 result:** PASS (entity tool used correctly for verification step)

**Decision point analysis:**
- The `entity` description says "use this whenever you need to create, query, modify..." but does not mention feature decomposition. An agent knowing about `decompose` would not confuse `entity(action: "create")` as the way to break down a feature. The entity description positions entity for direct CRUD, not structured decomposition workflows.

---

### Scenario 6: Ship a completed feature (P1 + P2)

**P1 result:** PASS (status, entity, doc all used correctly)

**Decision point analysis:**
- `status` correctly chosen for checking completion state (synthesised view).
- `doc(action: "register")` correctly chosen for registering the dev plan.
- `entity(action: "transition")` correctly chosen for advancing lifecycle.

---

### Scenario 8: Triage a stalled feature (P1 + P2)

**P1 result:** PASS (status and entity used correctly)

**Decision point analysis:**
- `status(id: "FEAT-...")` correctly chosen as first step — description says "use this before starting work to understand what's blocked."
- `entity(action: "list")` correctly chosen for drilling into specific child tasks after the dashboard view.

---

## P1 Tier Summary

| Scenario | P1 Tools Tested | Result | Description Changes Required |
|----------|----------------|--------|------------------------------|
| 1 | `status`, `entity` | PASS | None |
| 2 | `next`, `finish` | PASS | None |
| 4 | `entity`, `doc` | PASS | None |
| 5 | `status`, `next`, `entity` | PASS | None |
| 7 | `next`, `handoff`, `finish` | PASS | None |
| 3 | `entity` (with P2) | PASS | None |
| 6 | `status`, `entity`, `doc` (with P2) | PASS | None |
| 8 | `status`, `entity` (with P2) | PASS | None |

**All 8 scenarios passed on the first attempt.** No description rewrites were required.

### Key observations

1. **Explicit negative guidance is highly effective.** The "Use INSTEAD OF" and "Do NOT use for" patterns in the descriptions eliminate the most common decision-point ambiguities (e.g., `next` vs `entity list`, `finish` vs `entity transition`, `status` vs `entity get`).

2. **Workflow sequencing in `handoff` and `next` works well.** The "Call AFTER next(id)" / "Call BEFORE handoff" cross-references create a clear chain that agents can follow.

3. **Parameter relationship documentation helps distinguish modes.** The `next` description's "When id is provided... When id is omitted..." and the `finish` description's "In single mode... In batch mode..." clearly distinguish operational modes.

4. **`doc` register vs import is slightly subtle.** The distinction relies on parameter descriptions (path: "register: required; import: directory to scan") rather than the top-level description. This is adequate but could be strengthened in future iterations.

---

## Priority 2 — Decision-Point Tools

*Results will be recorded after Priority 2 descriptions are rewritten and tested.*

---

## Priority 3 — Query and Support Tools

*Results will be recorded after Priority 3 descriptions are rewritten and tested.*