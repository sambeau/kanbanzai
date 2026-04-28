# Research: P28 Issues Investigation

| Field   | Value |
|---------|-------|
| Date    | 2026-04-23 |
| Author  | Claude Sonnet 4.6 |
| Scope   | P28 retrospective issues |
| Status  | Draft |

---

## 1. Executive Summary

Five of the seven issues identified in the P28 retrospective share root causes in two
distinct architectural layers — they are not independent bugs. The first cluster
(`next` timeout, `worktree create` timeout, `cleanup` timeout) is caused by O(n)
full-directory scans in the entity service's `List()` and `Get()` paths, even though the
architectural design explicitly called for a SQLite cache to serve fast lookups. The cache
exists in the codebase but is bypassed on every read. The second cluster (context ceiling,
`handoff` wrong-skill assembly, heredoc recurrence) is caused by the `handoff` tool
assembling orchestrator-level skill content for implementer sub-agents, inflating every
dispatched prompt by approximately 8K tokens unnecessarily. Two remaining issues (lifecycle
orphan, setup overhead) are downstream consequences of these same root causes rather than
independent problems. The recommended next step is to commission two focused plans — one
addressing the state-persistence read path, one addressing the agent prompt system — and to
proceed directly to design without a further research spike.

---

## 2. Issue Clusters

### Cluster A: State-Persistence O(n) Scan

- **Issues:**
  - Issue 2: `next` timed out with 447 task files
  - Issue 1 (partial): `worktree(action: create)` timed out even with zero stale worktrees
  - Issue 4 (partial): Setup overhead — `cleanup` tool timed out during stale worktree removal
- **Shared root cause:** Every read operation on the entity service — `List()`, `Get()`
  when called without a slug, and `WorkQueue()` — performs a full `filepath.Glob` scan of
  the entity directory and reads every YAML file from disk. A SQLite cache exists and is
  populated on writes, but is never consulted on reads. At 447 task files, the combined
  I/O in a single MCP tool call exceeds the client timeout budget.
- **Evidence:**
  - `internal/service/entities.go:527–558`: `List()` uses `filepath.Glob` + reads every
    matched file. No cache consultation.
  - `internal/service/entities.go:444–488`: `ResolvePrefix()` — called by `Get()` when
    slug is absent — does a second `filepath.Glob` over all files to match a filename
    prefix.
  - `internal/mcp/next_tool.go:200–204`: `nextClaimMode` calls
    `entitySvc.Get("task", taskID, "")` with an empty slug, triggering the
    `ResolvePrefix` scan over 447 task files just to resolve the slug for one task.
  - `internal/mcp/next_tool.go:468–485`: `nextFindTopReadyTask` calls
    `entitySvc.List("task")` — a second full scan in the same claim call when the input
    ID is a feature ID rather than a task ID.
  - `internal/service/dependency_hook.go:68`: `evaluateDependents` calls
    `h.entitySvc.List("task")` — a full scan triggered on every task status transition
    to a terminal state, meaning every `finish` or `transition` call also pays the O(n)
    cost.
  - `internal/service/queue.go:43–53`: `WorkQueue()` calls `s.List("task")` as its first
    operation — a full scan on every `next()` queue-mode call.
  - `internal/mcp/cleanup_tool.go:70`: `cleanupListAction` calls `store.List()` — full
    worktree scan.
  - `internal/mcp/cleanup_tool.go:158`: `cleanupExecuteAction` calls `store.List()` when
    no specific `worktree_id` is given.
  - `internal/mcp/worktree_tool.go:116` (comment): The code comment at the
    `store.GetByEntityID` call explicitly notes it was added to fix "O(n) scan that caused
    timeouts with 34+ worktrees" — confirming the scan problem is known and was partially
    mitigated at the worktree store level, but not at the entity service level.
  - `internal/service/entities.go:135–143`: The `SetCache` comment says "lookups *may* use
    the cache for acceleration" — but neither `List()` nor `Get()` contain any `s.cache`
    reference. The cache is write-only from the read path's perspective.
  - File counts measured: 447 task YAML files, 122 feature YAML files, 45 worktree YAML
    files, 1091 total state files across all entity types.
  - Design decision (workflow-design-basis.md §7.1): The workflow design basis document
    explicitly calls for a SQLite cache for "fast querying, indexing, dependency analysis"
    alongside the flat-file canonical store. The architectural intent for fast reads exists
    and is unimplemented on the read path.
- **Recurrence:** The worktree-level scan issue was partially mitigated in P27
  (`GetByEntityID` early-termination). The underlying `List()` / `ResolvePrefix()` pattern
  in the entity service was not addressed. The `next` timeout is new at P28 scale (447
  tasks). The structural pattern has accumulated across at least P25–P28.

---

### Cluster B: Agent Prompt Inflation

- **Issues:**
  - Issue 7: `handoff` always assembles orchestrator context for implementer sub-agents
  - Issue 3: Sub-agent context ceiling on 4-task sequential feature chains
  - Issue 5: Terminal heredoc failure in worktrees (4th recurrence)
- **Shared root cause:** The `handoff` tool's pipeline assembly (3.0 path) selects skill
  content exclusively from the feature's stage binding (`state.Binding.Skills[0]`),
  ignoring the `role` parameter when choosing the skill. When a feature is in `developing`
  state, the stage binding maps to `orchestrate-development` skill regardless of whether
  the caller passes `role: implementer-go`. The `role` parameter only affects the identity
  header section via `stepResolveRole`; the procedural content is always the orchestrator
  skill. The `implement-task` skill is never loaded by the pipeline for sub-agent dispatch.
  Additionally, `implement-task/SKILL.md` still recommends heredoc as the primary Go
  file-write method, so even if the skill were correctly loaded, it would propagate the
  wrong instruction into every sub-agent prompt.
- **Evidence:**
  - `internal/context/pipeline.go:391–415` (`stepLoadSkill`): skill name is always taken
    from `state.Binding.Skills[0]`; `state.Input.Role` (the caller's role parameter) is
    not referenced in this function at all.
  - `internal/context/pipeline.go:353–388` (`stepResolveRole`): the `role` parameter IS
    used here to select the role identity file, but role identity (who you are) is a
    separate section from skill content (what you do). Role identity being correct does not
    make the skill content correct.
  - `.kbz/stage-bindings.yaml` (developing entry): `skills: [orchestrate-development]`
    is the binding's primary skill. `sub_agents.skills: [implement-task]` exists in the
    YAML under `sub_agents` but no pipeline step reads this field.
  - `.kbz/skills/implement-task/SKILL.md:92–95`: Still recommends heredoc as "primary"
    for Go source files ("Use a heredoc for Go source files. `GOEOF` is the standard
    delimiter").
  - `.kbz/skills/implement-task/SKILL.md:137`: Checklist still reads "use `terminal` +
    heredoc for Go files, `python3 -c` for Markdown/YAML, NOT `edit_file`".
  - Retro recurring patterns table: heredoc failure has appeared in P25, P26, P27, P28 —
    four consecutive plans without the skill file being corrected.
- **Recurrence:** `handoff` wrong-skill assembly first observed P26, unfixed through P28
  (3 consecutive plans). Heredoc failure first observed P25, unfixed through P28 (4
  consecutive plans). Context ceiling first observed P26, contributing cause unfixed
  through P28.

---

### Cluster C: Lifecycle Gate Hygiene (isolated)

- **Issues:**
  - Issue 6: FEAT-01KPVDDYQQS1Y orphaned in `reviewing` state with no review report
- **Shared root cause:** The `merge` tool's pre-merge gate does not enforce review report
  existence for features in `reviewing` status, and `override: true` can bypass the
  lifecycle check entirely. The `status` dashboard does not surface features stuck in
  `reviewing` without a registered review report as a warning-severity attention item,
  making the orphan invisible until plan close.
- **Evidence:** Retro §what went wrong issue 6: "The previous orchestrator apparently used
  `override: true` on the `entity_done` merge gate to force the merge, bypassing the
  lifecycle." No code-level fix has been applied to the merge gate or status dashboard. The
  pattern was also observed in P26.
- **Recurrence:** Lifecycle orphan pattern observed in P26 and P28 (2 plans). Lower
  recurrence and narrower blast radius than Clusters A and B.

---

## 3. Architectural Assessment

### Layer: State Persistence (flat-file store + entity service read path)

- **What it is:** The canonical entity store is a directory of YAML files, one per entity,
  under `.kbz/state/{type}/`. The entity service provides `List()`, `Get()`,
  `ResolvePrefix()`, and `WorkQueue()` as the primary read API for all MCP tools.
- **Current design:** All read operations perform a `filepath.Glob` against the entity
  directory and deserialise every matching YAML file on every call. `Get()` without a
  slug calls `ResolvePrefix()`, which is also a full scan of filenames. A SQLite cache
  (`cache.Cache`) is populated on every write — `cacheUpsertFromResult` is called at the
  end of `CreateFeature`, `CreateTask`, `CreateBug`, `CreateDecision`, `UpdateStatus`, and
  `UpdateEntity` — but the cache is never consulted in `List()`, `Get()`, or
  `ResolvePrefix()`. The design basis document (§7.1) explicitly specified the cache for
  "fast querying, indexing, dependency analysis"; the read path has not been wired to it.
- **Scaling property:** O(n) where n = number of entity files of the given type. At P28,
  n = 447 for tasks. A single `next(id: TASK-xxx)` call in claim mode can trigger up to
  two separate full-directory scans: one in `ResolvePrefix` inside `Get`, and one in
  `evaluateDependents` inside the dependency hook when the task is transitioned to active.
  A `next()` queue-mode call triggers at least one full scan in `WorkQueue`. The `cleanup`
  tool triggers one or two scans of the worktree store per call.
- **Breaking point:** Already broken at P28. A single `next` call with 447 task files
  timed out post-server-restart. The MCP client timeout is a fixed external constraint;
  the task file count grows monotonically with each plan at a rate of approximately 15–25
  tasks per plan. Without a structural fix, every plan after P28 increases the probability
  of timeout on the most frequently called tool in the system.
- **Prior decisions:** Workflow Design Basis §7.1 specifies a SQLite cache for fast
  querying and dependency analysis alongside the flat-file canonical store. §7.5 specifies
  one file per entity. The architectural intent is that the flat-file store is the
  canonical source of truth while the cache accelerates reads. The current implementation
  has only half of this design in place.

---

### Layer: Agent Prompt System (pipeline skill assembly)

- **What it is:** The `handoff` tool's 3.0 pipeline assembles a Markdown prompt for
  sub-agent dispatch. The pipeline runs a sequence of steps: resolve stage, lookup
  binding, apply inclusion strategy, resolve role, load skill, surface knowledge, assemble
  sections, resolve tool hint, check token budget.
- **Current design:** `stepResolveRole` (pipeline.go:353) respects the caller's `role`
  parameter for identity selection. `stepLoadSkill` (pipeline.go:391) always selects
  `state.Binding.Skills[0]` from the stage binding, ignoring `role`. The `developing`
  stage binding maps to `orchestrate-development` skill. The `sub_agents.skills:
  [implement-task]` field in the binding is present in the YAML but no pipeline step reads
  it. As a result, `handoff(task_id: T, role: implementer-go)` returns a prompt that is
  identity-correct (the identity header says "you are implementer-go") but skill-wrong
  (the procedural sections contain `orchestrate-development` content: dispatch batches,
  context compaction, multi-feature scoping — entirely irrelevant to writing a Go
  function).
- **Scaling property:** The inflation is a fixed per-dispatch cost (~8K tokens for the
  orchestrate-development skill content), but it compounds multiplicatively with sequential
  task chains: a 4-task chain has 4x the accumulated inflation overhead, reliably hitting
  the context ceiling before task 3. This does not worsen with project scale per se, but
  the mismatch grows more expensive per feature as features grow longer or more sequential.
- **Breaking point:** Already causing concrete failures. P28 recorded 4 context ceiling
  events (the T2 agent in session 1 and all three Sprint 2 orchestrator agents in session
  2). P26 and P27 also recorded context ceiling events from the same cause. Each ceiling
  event costs approximately 6 extra `spawn_agent` calls and 30–40 minutes of wall-clock
  time.
- **Prior decisions:** No design decision was found in the knowledge base or document
  records that specifies the current `stepLoadSkill` behaviour as intentional. The stage
  bindings YAML has a `sub_agents.skills` field that implies design intent to load
  different skills for sub-agent versus orchestrator dispatch, but no rationale document
  was found. This is a research gap (see §6, Gap 3).

---

## 4. Connection Assessment

**Are the issues small and unconnected? NO.**

Five of the seven P28 issues share root causes in two identifiable architectural layers.
The `next` timeout, `worktree create` timeout, and `cleanup` timeout are not three separate
bugs — they are three manifestations of the same O(n) full-directory scan in the entity
service read path. The architectural evidence is unambiguous: `List()` at line 527 reads
every YAML file on every call, `ResolvePrefix()` at line 444 scans every filename on every
`Get()` call without a slug, and the SQLite cache designed to prevent this is wired only
into the write path. Similarly, the context ceiling, wrong-skill assembly, and heredoc
recurrence are not three separate issues — they form a prompt-inflation cluster in which
the `handoff` tool's `stepLoadSkill` (pipeline.go:391) ignores the role parameter, loads
orchestrator skill content for implementer agents, and `implement-task/SKILL.md` compounds
the problem by still recommending heredoc as the primary write method. The remaining two
issues (lifecycle orphan, setup overhead) are downstream consequences of the persistence
scan problem (the cleanup tool's own timeout prevents automatic maintenance, allowing stale
state to accumulate) and the merge gate gap (override bypasses the review report check),
not independent problems requiring separate diagnosis.

---

## 5. Plan Structure Recommendation

Two focused plans are recommended, with a third optional lower-priority plan. Their file
scopes are non-overlapping, their fix strategies are independent, and each delivers
immediate user-visible value without the other.

### Plan A — State Store Read Path Performance

- **Scope:** Wire the existing SQLite cache into the `List()`, `Get()`, and
  `ResolvePrefix()` read paths in `internal/service/entities.go`. Ensure the cache is
  warm at server startup (verify or add `RebuildCache()` call in the startup path). Address
  all call sites that trigger full-directory scans: `next_tool.go` (claim mode),
  `worktree_tool.go` (create), `cleanup_tool.go` (list + execute), `dependency_hook.go`
  (evaluateDependents), `queue.go` (WorkQueue). Explicitly excludes: prompt system changes,
  skill files, MCP tool descriptions, lifecycle gate changes.
- **Dependencies:** None. Can start immediately.
- **Rationale:** Pure Go refactoring within `internal/service/`. The cache already exists
  and is populated on writes; connecting it to reads is a targeted change with
  well-understood scope. This is the highest-urgency fix: the `next` timeout will worsen
  monotonically with every plan added after P28 — the tool is called on every single task
  claim and queue inspection.

### Plan B — Handoff Skill Assembly and Prompt Hygiene

- **Scope:** Fix `internal/context/pipeline.go`'s `stepLoadSkill` to select the
  sub-agent skill (from `sub_agents.skills` in the stage binding) when the caller passes
  a sub-agent role that matches a role listed under `sub_agents.roles`. Update
  `.kbz/skills/implement-task/SKILL.md` to replace heredoc with `write_file(entity_id:
  ...)` as the sole recommended Go file-write method and update the checklist at L137
  accordingly. Update `.kbz/stage-bindings.yaml` if the sub-agent skill selection logic
  requires a structural change to how bindings encode role-to-skill mappings. Explicitly
  excludes: state store changes, merge gate logic, dashboard changes.
- **Dependencies:** None. Can be designed in parallel with Plan A; ships independently.
- **Rationale:** The fix touches `internal/context/pipeline.go` and `.kbz/skills/` — a
  disjoint write set from Plan A. The skill file fix (heredoc to `write_file`) is a small
  change that would immediately stop the 4th-recurrence failure. The pipeline fix requires
  a design decision on how the binding expresses role-to-skill mapping for sub-agents (see
  §6, Gap 3), which should be resolved in the design document for this plan.

### Plan C — Lifecycle Gate Hardening (lower priority)

- **Scope:** Add a review report existence check to the `merge` tool's pre-merge gate for
  features in `reviewing` status, non-bypassable by `override: true` for this specific
  check. Add "features stuck in `reviewing` with no registered review report" as a
  warning-severity attention item in the `status` project dashboard. Explicitly excludes:
  state store and prompt changes.
- **Dependencies:** None. Can be designed and shipped after Plans A and B, or in parallel.
- **Rationale:** This is a correctness issue but not a throughput issue. Two-plan
  recurrence and narrow blast radius justify lower priority. The fix scope is confined to
  `internal/mcp/merge_tool.go` (or equivalent gate logic) and the `status` dashboard
  formatter.

---

## 6. Research Gaps

### Gap 1: MCP client timeout budget

- **Unknown:** What is the MCP client timeout threshold in the kanbanzai server? Is it
  configurable per-tool, per-server, or fixed by the MCP library? The retro describes a
  timeout but does not state the threshold in seconds.
- **Why it matters:** The threshold determines whether the fix needs to reduce scan time
  below a fixed ceiling or whether the timeout can be raised as a short-term stopgap while
  the structural cache fix lands.
- **How to answer it:** Inspect `github.com/mark3labs/mcp-go` server configuration and
  the kanbanzai server startup code in `cmd/` for timeout parameters. A one-off
  measurement of `next` call wall-clock latency with 447 task files on a warm server would
  bound the problem concretely.

### Gap 2: Cache completeness at server startup

- **Unknown:** Is the SQLite cache rebuilt from the YAML files at server startup, or does
  it start empty? If it starts empty, wiring reads through the cache would return cache
  misses for all entities until a write event triggers `cacheUpsertFromResult`.
- **Why it matters:** A cache-first read strategy is only correct if the cache is
  guaranteed to be warm. If the cache is cold on startup, the very scenario that caused
  the P28 timeout — a `next` call immediately after a server restart — would be unchanged
  even after the fix.
- **How to answer it:** Read the server startup sequence in `cmd/serve/` (or equivalent)
  to find where `RebuildCache()` is or is not called. If absent, the Plan A design must
  include a cache warm-up step at startup. This is a 10-minute code read, not a research
  spike.

### Gap 3: Pipeline sub-agent skill selection — design intent

- **Unknown:** Was the `sub_agents.skills: [implement-task]` field in
  `.kbz/stage-bindings.yaml` intended to be consumed by the pipeline, or is it
  documentation-only metadata? Is there a prior design document that specifies how the
  pipeline should behave when the caller's `role` matches a sub-agent role rather than the
  binding's primary orchestrator role?
- **Why it matters:** Two fix strategies have different complexity and different
  backward-compatibility profiles: (a) add logic to `stepLoadSkill` that checks whether
  the caller's role is listed under `sub_agents.roles` and if so loads the corresponding
  `sub_agents.skills` entry; (b) require callers to pass the skill name explicitly as a
  `handoff` parameter. Strategy (a) is automatic but requires the binding schema to
  reliably encode sub-agent role/skill pairs. Strategy (b) is explicit but requires all
  existing orchestration skills to be updated. The choice should be recorded as a design
  decision before implementation begins.
- **How to answer it:** Review P26 and P27 retrospectives for any proposed fix approaches.
  Check whether a design document was commissioned at that time. No spike needed — this is
  a design decision to be made in the Plan B design document.

### Gap 4: Token size of orchestrate-development vs implement-task skill

- **Unknown:** What is the actual token count of `orchestrate-development` skill content
  as loaded by the pipeline? The retro estimates ~8K tokens but this is unverified.
- **Why it matters:** Quantifying the inflation determines how much context headroom the
  skill-selection fix would recover for implementer agents, and whether the context ceiling
  problem is fully resolved by the fix or whether additional token-budgeting changes in the
  pipeline are also required.
- **How to answer it:** Run `wc -w .kbz/skills/orchestrate-development/SKILL.md` and
  `wc -w .kbz/skills/implement-task/SKILL.md`. Apply a ~1.3 words-per-token ratio for a
  rough estimate. This is a one-command measurement.

---

## 7. Recommendation on Further Research

**Verdict: Partial**

**Cluster A (state persistence)** and **Cluster B (prompt inflation / heredoc)** can
proceed to design immediately. The root causes are confirmed by direct code reading: the
cache is wired only into writes, `stepLoadSkill` ignores `role`, and `implement-task`
SKILL.md still recommends heredoc. The file-scope boundaries for both plans are clear and
non-overlapping. The only pre-design measurement that would improve Plan A's design is
confirming cache completeness at startup (Gap 2) — this is a 10-minute code read, not a
research spike, and can be the first task of Plan A's dev plan rather than a blocker to
writing the design document.

**Cluster C (lifecycle gate)** can also proceed to design without further research; the
scope is narrow and the fix strategy is unambiguous.

The four gaps identified are inputs to the design documents, not blockers to writing them.
Gap 1 (MCP timeout budget) and Gap 2 (cache warmth) should be answered in the first
implementation task of Plan A. Gap 3 (pipeline sub-agent skill intent) is a design
decision to be recorded in the Plan B design document. Gap 4 (token size) is a
one-command measurement that can be done during design or implementation.

---

## 8. Code Evidence Appendix

### A1 — `List()` performs full-directory scan with no cache consultation

```kanbanzai/internal/service/entities.go#L527-558
func (s *EntityService) List(entityType string) ([]ListResult, error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	if entityType == "" {
		return nil, fmt.Errorf("entity type is required")
	}

	dir := filepath.Join(s.root, entityDirectory(entityType))
	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list %s entities: %w", entityType, err)
	}

	sort.Strings(entries)

	results := make([]ListResult, 0, len(entries))
	for _, entry := range entries {
		record, err := s.loadRecordFromPath(entityType, entry)
		if err != nil {
			return nil, err
		}
		results = append(results, ListResult{
			Type:  record.Type,
			ID:    record.ID,
			Slug:  record.Slug,
			Path:  entry,
			State: record.Fields,
		})
	}
	return results, nil
}
```

### A2 — `ResolvePrefix()` also performs a full-directory scan

```kanbanzai/internal/service/entities.go#L444-488
func (s *EntityService) ResolvePrefix(entityType, prefix string) (resolvedID, resolvedSlug string, err error) {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	prefix = strings.ToUpper(strings.TrimSpace(prefix))
	prefix = id.StripBreakHyphens(prefix)

	dir := filepath.Join(s.root, entityDirectory(entityType))
	entries, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	// ... iterates all entries, matching by filename prefix ...
	for _, entry := range entries {
		base := filepath.Base(entry)
		baseName := strings.TrimSuffix(base, ".yaml")
		fileID, fileSlug, err := parseRecordIdentity(entityType, baseName)
		if err != nil {
			continue
		}
		normalizedID := strings.ToUpper(fileID)
		if strings.HasPrefix(normalizedID, prefix) {
			matches = append(matches, match{id: fileID, slug: fileSlug})
		}
	}
	// ...
}
```

### A3 — `Get()` falls through to `ResolvePrefix()` when slug is absent

```kanbanzai/internal/service/entities.go#L490-525
func (s *EntityService) Get(entityType, entityID, slug string) (GetResult, error) {
	// ...
	if slug == "" {
		resolvedID, resolvedSlug, err := s.ResolvePrefix(entityType, entityID)
		if err != nil {
			return GetResult{}, err
		}
		entityID = resolvedID
		slug = resolvedSlug
	}
	// ...
}
```

### A4 — `nextClaimMode` calls `Get` with empty slug, triggering `ResolvePrefix` scan

```kanbanzai/internal/mcp/next_tool.go#L200-204
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return nil, fmt.Errorf("Cannot claim task %s: task not found.\n\nTo resolve:\n  Verify the task ID with entity(action: \"list\", type: \"task\") or inspect the queue with next()", taskID)
	}
```

### A5 — `dependency_hook.go` calls `List("task")` on every terminal-state transition

```kanbanzai/internal/service/dependency_hook.go#L68-75
func (h *DependencyUnblockingHook) evaluateDependents(completedTaskID string) ([]UnblockedTask, error) {
	allTasks, err := h.entitySvc.List("task")
	if err != nil {
		return nil, err
	}
	// Index all task statuses for fast lookup during dependency checks.
	taskStatuses := make(map[string]string, len(allTasks))
```

### A6 — `WorkQueue()` calls `List("task")` as its first operation

```kanbanzai/internal/service/queue.go#L43-53
func (s *EntityService) WorkQueue(input WorkQueueInput) (WorkQueueResult, error) {
	var result WorkQueueResult

	// Load all tasks
	allTasks, err := s.List(string(model.EntityKindTask))
	if err != nil {
		return result, fmt.Errorf("list tasks: %w", err)
	}
	// ...
}
```

### A7 — Cache is populated on writes but never consulted on reads

```kanbanzai/internal/service/entities.go#L135-143
// SetCache attaches an optional local derived cache.
// When set, mutations update the cache best-effort, and lookups
// may use the cache for acceleration. All operations fall back
func (s *EntityService) SetCache(c *cache.Cache) {
	s.cache = c
}
```

The comment says "lookups *may* use the cache for acceleration" but `List()` and `Get()`
contain no `s.cache` reference. The SQLite cache is write-only from the read path's
perspective.

### A8 — `stepLoadSkill` ignores the `role` parameter

```kanbanzai/internal/context/pipeline.go#L391-415
func (p *Pipeline) stepLoadSkill(state *PipelineState) error {
	skillName := ""
	if len(state.Binding.Skills) > 0 {
		skillName = state.Binding.Skills[0]
	}
	if skillName == "" {
		return pipelineError(6, "skill-loading",
			fmt.Sprintf("no skill specified for stage %q", state.Stage),
			"add a skill to the binding for this stage in stage-bindings.yaml")
	}
	// NOTE: state.Input.Role is never referenced in this function.
	// The caller's role parameter does not influence skill selection.
	sk, err := p.Skills.Load(skillName)
	// ...
}
```

`state.Input.Role` (the `implementer-go` value passed by the orchestrator) is visible in
the pipeline state but is not read here. The skill is selected solely by the binding.

### A9 — `developing` stage binding maps to `orchestrate-development`; sub-agent skill exists but is not loaded

```kanbanzai/.kbz/stage-bindings.yaml#L64-79
  developing:
    description: "Implementing tasks from the dev plan"
    orchestration: orchestrator-workers
    roles: [orchestrator]
    skills: [orchestrate-development]       # always loaded by stepLoadSkill
    human_gate: false
    ...
    sub_agents:
      roles: [implementer]
      skills: [implement-task]              # present in YAML; never loaded by the pipeline
      topology: parallel
```

### A10 — `implement-task/SKILL.md` still recommends heredoc as the primary method

```kanbanzai/.kbz/skills/implement-task/SKILL.md#L92-106
### Go source files — use heredoc (primary)

Use a heredoc for Go source files. `GOEOF` is the standard delimiter for Go
source; any unique uppercase string may substitute:

...

> **Delimiter collision warning:** If the file content contains a line that is
> exactly `GOEOF`, the heredoc will silently truncate at that line. Fix: choose
> a different delimiter (e.g. `GOEOF2`, `ENDOFFILE`).
```

And at the checklist (SKILL.md:137):

```kanbanzai/.kbz/skills/implement-task/SKILL.md#L137-137
- [ ] Confirmed whether this task runs inside a worktree — if yes, use `terminal` + heredoc for Go files, `python3 -c` for Markdown/YAML, NOT `edit_file`
```

### A11 — File counts (measured 2026-04-23)

```/dev/null/terminal-output.txt#L1-5
Total state files:   1091
Task YAML files:      447
Feature YAML files:   122
Worktree YAML files:   45
```

### A12 — Architectural design intent: cache for fast reads (not yet wired)

From `work/design/workflow-design-basis.md §7.1`:

```kanbanzai/work/design/workflow-design-basis.md#L1-1
### 7.1 Git-Native State With Local Cache

A local SQLite cache should be used for:
- fast querying
- indexing
- dependency analysis
- health checks
- search support
```

The design basis explicitly specifies that the cache, not the flat-file scan, should serve
read queries. The current implementation inverts this: flat-file scans serve all reads, the
cache is populated on writes and ignored on reads.