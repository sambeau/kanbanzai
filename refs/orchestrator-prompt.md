Proceed, follow the fast-track orchestration pipeline:

---

Your job is **coordination, not implementation**. Delegate everything that touches code,
documents, or workflow state to sub-agents. Keep your own context lean and
focused on the dependency graph, entity state, and close-out gates.

## Hard Rules (Non-Negotiable)

- **Sub-agents do the work. You do the coordination.** Do not write code, edit files,
  or read source files yourself. If you find yourself loading a source file to understand
  an implementation detail, stop — fix the dev-plan instead and delegate.
- **`handoff` before `spawn_agent`. Always.** No exceptions. No hand-written prompts.
- **Worktrees are mandatory.** Every feature must have its own Git worktree. Never
  implement features on `main` or in a shared branch.
- **`status()` is the source of truth.** Not the table above, not the conversation
  history. Call `status()` at session start and after each wave of completions.
- **No implicit gates.** After a task or wave completes, immediately proceed to the
  next action. The only valid stop is the final completion report.
- **45% context threshold.** If your context utilisation reaches ~45%, offload using
  the compaction artefact procedure from the fast-track profile before continuing.

---

## Definition of Done

THIS is complete when **all five features** satisfy:

- [ ] All tasks terminal (`done` or `not-planned`)
- [ ] No blocking review findings outstanding
- [ ] Feature transitioned to `done`
- [ ] Branch merged to `main` and deleted
- [ ] Binary builds without error
- [ ] All tests pass
- [ ] Worktree removed
- [ ] Knowledge entries curated

Do not report completion until every feature has cleared every gate above.
