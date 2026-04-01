# Agent-Driven Test Scenarios for MCP Tool Descriptions

## Purpose

These scenarios validate that an LLM agent can select the correct Kanbanzai MCP
tools from their descriptions alone. Each scenario presents a natural-language task,
the expected tool-call sequence, and at least one decision point where the agent
must choose between plausible alternatives.

## How to use

1. Present the agent with the full set of MCP tool descriptions (no additional hints).
2. Give it the **Task description** from a scenario.
3. Compare the agent's tool selections against the **Expected tool sequence**.
4. Pay special attention to **Decision points** — these are the moments where a
   vague or misleading description would cause the agent to pick the wrong tool.

A passing scenario means the agent selected the right tools in the right order
at every decision point. Partial credit can be given when the agent self-corrects
after an initial wrong choice.

## Coverage matrix

| Workflow pattern                        | Scenarios |
|-----------------------------------------|-----------|
| Advancing a feature through a lifecycle | 1, 6      |
| Claiming and completing a task          | 2, 7      |
| Decomposing a feature into tasks        | 3         |
| Creating and registering a document     | 4, 6      |
| Querying project status / blocked work  | 5, 8      |

| Priority | Tools                                       | Scenarios covering them        |
|----------|---------------------------------------------|--------------------------------|
| P1       | `entity`                                    | 1, 3, 4, 5, 6                 |
| P1       | `doc`                                       | 4, 6                           |
| P1       | `handoff`                                   | 7                              |
| P1       | `next`                                      | 2, 5, 7                        |
| P1       | `finish`                                    | 2, 7                           |
| P1       | `status`                                    | 1, 5, 6, 8                     |
| P2       | `decompose`                                 | 3                              |
| P2       | `merge`                                     | 6, 8                           |
| P2       | `pr`                                        | 6, 8                           |

---

## Scenarios

### Scenario 1: Advance a feature from specifying to dev-planning

**Workflow pattern:** Advancing a feature through a lifecycle stage
**Priority tools covered:** P1: `status`, `entity`

**Task description:**
Feature FEAT-01ABC has an approved specification. Check whether it's ready to
move forward, and if so, advance it to the dev-planning stage.

**Expected tool sequence:**
1. `status(id: "FEAT-01ABC")` — inspect the feature's current lifecycle state and check whether stage-gate prerequisites (approved spec) are met
2. `entity(action: "transition", id: "FEAT-01ABC", status: "dev-planning")` — advance the feature to the next stage

**Decision point(s):**
- **`status` vs `entity(action: "get")`:** Both can retrieve feature state. The agent must choose `status` because it returns *derived* stage-gate readiness and attention items, while `entity get` returns only raw field data without prerequisite analysis. The tool description for `status` should make clear that it provides "lifecycle status, attention items, and derived state" that raw entity data does not.

---

### Scenario 2: Claim the next available task and complete it

**Workflow pattern:** Claiming and completing a task
**Priority tools covered:** P1: `next`, `finish`

**Task description:**
Check the work queue for ready tasks, claim the highest-priority one, do the
work, then mark it done with a summary of what was accomplished.

**Expected tool sequence:**
1. `next()` — inspect the ready queue (no id) to see available tasks sorted by priority
2. `next(id: "TASK-...")` — claim the top task; returns assembled context (spec sections, knowledge, file paths)
3. *(agent performs the work)*
4. `finish(task_id: "TASK-...", summary: "...")` — record completion and transition to done

**Decision point(s):**
- **`next()` vs `entity(action: "list", type: "task", status: "ready")`:** Both can show ready tasks. The agent must choose `next` because it is described as "the primary way to pick up work" and returns tasks sorted by priority with optional conflict checking. `entity list` returns raw task records without priority ranking or context assembly.
- **`finish` vs `entity(action: "transition", status: "done")`:** Both can move a task to done. The agent must choose `finish` because it also records the completion summary, knowledge contributions, and retrospective signals in one call — `entity transition` only changes status.

---

### Scenario 3: Decompose a feature into implementation tasks

**Workflow pattern:** Decomposing a feature into tasks
**Priority tools covered:** P1: `entity`, P2: `decompose`

**Task description:**
Feature FEAT-02XYZ is in dev-planning and has an approved design. Break it down
into implementation tasks using the standard propose → review → apply workflow.

**Expected tool sequence:**
1. `decompose(action: "propose", feature_id: "FEAT-02XYZ")` — generate a task breakdown proposal from the feature's spec
2. `decompose(action: "review", feature_id: "FEAT-02XYZ", proposal: {...})` — review the proposal for completeness and ordering
3. `decompose(action: "apply", feature_id: "FEAT-02XYZ", proposal: {...})` — create the tasks from the confirmed proposal
4. `entity(action: "list", type: "task", parent: "FEAT-02XYZ")` — verify the created tasks exist and are correctly parented

**Decision point(s):**
- **`decompose(action: "propose")` vs manually calling `entity(action: "create", type: "task")` multiple times:** The agent must choose `decompose` because it generates a structured proposal from the feature specification with dependency analysis, rather than requiring the agent to manually invent and create tasks one by one. The description should make clear that `decompose` is the standard workflow for breaking features into tasks.
- **`entity list` for verification vs `status`:** After applying, `entity list` is correct for confirming the raw task records were created under the parent feature. `status` would be appropriate if the agent needed to check lifecycle readiness, but here we just need to verify task creation.

---

### Scenario 4: Write a spec document and register it against a feature

**Workflow pattern:** Creating and registering a document
**Priority tools covered:** P1: `entity`, `doc`

**Task description:**
Write a specification document for feature FEAT-03DEF, save it to disk, then
register it in the document system so it can be tracked and approved.

**Expected tool sequence:**
1. `entity(action: "get", id: "FEAT-03DEF")` — retrieve the feature's summary and context to inform the spec content
2. *(agent writes the spec file to disk)*
3. `doc(action: "register", path: "work/spec/FEAT-03DEF-spec.md", type: "specification", title: "...", owner: "FEAT-03DEF")` — register the document record so it's tracked in the system

**Decision point(s):**
- **`doc(action: "register")` vs `doc(action: "import")`:** Both can create document records. The agent must choose `register` for a single known document with explicit metadata. `import` is a batch operation that scans a directory and infers document types from path patterns — it's the wrong tool for registering one specific document with a known type and owner.
- **`entity get` vs `status` for initial context:** The agent needs the feature's raw field data (summary, slug, design reference) to write the spec. `entity get` is correct here because we need specific fields, not the synthesised dashboard view that `status` provides.

---

### Scenario 5: Find all blocked work across the project

**Workflow pattern:** Querying project status and finding blocked work
**Priority tools covered:** P1: `status`, `next`, `entity`

**Task description:**
Give me a summary of the project's health: what's blocked, what needs attention,
and what tasks are ready to be picked up.

**Expected tool sequence:**
1. `status()` — get the project-level synthesis dashboard with attention items, blocked entities, and progress metrics
2. `next()` — inspect the ready queue to see which tasks are available for claiming
3. `entity(action: "list", type: "task", status: "blocked")` — drill into specific blocked tasks if the status dashboard flags them

**Decision point(s):**
- **`status()` vs `entity(action: "list")` for the overview:** The agent must start with `status` (no id) because it returns a *synthesised* project overview with derived state — what's blocked, what's ready, attention items — that `entity list` cannot provide. `entity list` returns raw records and requires the agent to manually compute blocked/ready state from dependencies.
- **`next()` vs `entity(action: "list", type: "task", status: "ready")`:** For seeing what's ready to work on, `next` is preferred because it returns tasks sorted by priority, which is what "ready to be picked up" implies. However, `entity list` with a status filter would be reasonable if the agent only needs to count ready tasks without priority context.

---

### Scenario 6: Ship a completed feature — document, advance, PR, merge

**Workflow pattern:** Advancing a feature through a lifecycle stage + Creating and registering a document
**Priority tools covered:** P1: `status`, `entity`, `doc`; P2: `pr`, `merge`

**Task description:**
Feature FEAT-04GHI has all tasks done. Register the final dev plan document,
advance the feature to its next lifecycle stage, open a pull request, check
merge readiness, and merge it.

**Expected tool sequence:**
1. `status(id: "FEAT-04GHI")` — confirm all tasks are complete and check current lifecycle state
2. `doc(action: "register", path: "...", type: "dev-plan", owner: "FEAT-04GHI")` — register the dev plan document
3. `entity(action: "transition", id: "FEAT-04GHI", status: "implementing", advance: true)` — advance the feature through lifecycle stages toward implementing/complete
4. `pr(action: "create", entity_id: "FEAT-04GHI")` — open a GitHub pull request for the feature
5. `merge(action: "check", entity_id: "FEAT-04GHI")` — evaluate merge gate readiness (CI, reviews, etc.)
6. `merge(action: "execute", entity_id: "FEAT-04GHI")` — execute the merge after gates pass

**Decision point(s):**
- **`pr(action: "create")` vs using the GitHub `create_pull_request` tool directly:** The agent must choose `pr` because it is entity-aware — it automatically derives the branch, title, and description from the feature's metadata and worktree. The raw GitHub tool would require the agent to manually look up the branch name, compose a body, and set the correct base.
- **`merge(action: "check")` then `merge(action: "execute")` vs calling `merge(action: "execute")` directly:** The agent should check gates first. The `merge check` action evaluates all merge prerequisites (CI status, review approvals, branch health) and reports blocking issues, while `execute` would fail or require an override if gates aren't met.
- **`doc register` vs `doc approve`:** The agent registers the document first. Approval is a separate action (typically performed by a reviewer), so the agent should not call `approve` in the same flow as `register` unless explicitly told to self-approve.

---

### Scenario 7: Delegate a task to a sub-agent and record completion

**Workflow pattern:** Claiming and completing a task
**Priority tools covered:** P1: `next`, `handoff`, `finish`

**Task description:**
Claim task TASK-05JKL, generate a prompt to delegate it to a sub-agent, and
after the sub-agent finishes, record the task as complete with the files it
modified and a knowledge entry about what was learned.

**Expected tool sequence:**
1. `next(id: "TASK-05JKL")` — claim the task and get assembled context (spec, knowledge, files, role)
2. `handoff(task_id: "TASK-05JKL", role: "backend")` — generate a complete sub-agent prompt with all context assembled
3. *(sub-agent performs the work)*
4. `finish(task_id: "TASK-05JKL", summary: "...", files_modified: [...], knowledge: [...])` — record completion with knowledge contribution

**Decision point(s):**
- **`handoff` vs manually reading the task with `entity get` and composing a prompt:** The agent must choose `handoff` because it assembles spec sections, knowledge entries, file paths, and role conventions into a structured prompt — far more comprehensive than what `entity get` returns. The description should make clear that `handoff` output "is designed to go directly into spawn_agent's message parameter."
- **`next(id: ...)` vs `entity(action: "transition", status: "active")`:** Both can activate a task. `next` is preferred because it claims the task *and* returns full assembled context in one call. `entity transition` only changes the status without providing any context for actually doing the work.

---

### Scenario 8: Triage a stalled feature — diagnose and unblock

**Workflow pattern:** Querying project status and finding blocked work
**Priority tools covered:** P1: `status`, `entity`; P2: `merge`, `pr`

**Task description:**
Feature FEAT-06MNO has been stuck for a while. Figure out what's wrong — check
its status, look at whether the PR has CI failures or review issues, and
determine what needs to happen to unblock it.

**Expected tool sequence:**
1. `status(id: "FEAT-06MNO")` — get the feature's synthesised state: lifecycle position, blocked tasks, attention items
2. `pr(action: "status", entity_id: "FEAT-06MNO")` — check the PR's CI and review status for failures
3. `merge(action: "check", entity_id: "FEAT-06MNO")` — evaluate which merge gates are failing and why
4. `entity(action: "list", type: "task", parent: "FEAT-06MNO")` — list child tasks to see if any are incomplete or blocked

**Decision point(s):**
- **`pr(action: "status")` vs `merge(action: "check")`:** These serve different purposes and the agent needs both. `pr status` reports CI check-run results and reviewer states from GitHub, while `merge check` evaluates Kanbanzai's own merge gates (which may include document prerequisites, task completion, and branch health beyond just CI). An agent that only calls one would miss half the picture.
- **`status` vs `entity get` for initial diagnosis:** The agent must start with `status` because a stalled feature needs the synthesised view — attention items, blocked dependencies, and lifecycle state analysis. `entity get` would only return the raw fields without diagnosing *why* the feature is stuck.