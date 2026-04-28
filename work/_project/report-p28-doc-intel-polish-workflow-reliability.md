# Retrospective: P28 — Doc-Intel Polish and Workflow Reliability

| Field | Value |
|-------|-------|
| Scope | P28-doc-intel-polish-workflow-reliability |
| Plan | Doc-Intel Polish and Workflow Reliability |
| Date | 2026-04-23 |
| Features | 6 delivered, 6 merged |
| Tasks | 24 / 24 done |
| Authors | Two orchestrators (Claude Sonnet 4.6) — Sprint setup + Sprint 1; Sprint 1B completion + Sprint 2 |

---

## Summary

P28 addressed four P27 retrospective findings: a plan lifecycle gap (`proposed → active`
shortcut), missing dev-plan auto-registration in `decompose apply`, worktree creation
timeouts under load, and a set of doc-intel guide enrichments. All six features were
delivered and merged across two sessions.

The sessions went well overall. The parallel dispatch pattern worked, the code quality was
high, and no blocking review findings were raised. There were, however, several significant
friction patterns — some recurring from earlier plans, some new — that deserve systematic
fixes. This report combines the direct experience of both orchestrators who worked on this
plan.

---

## What Went Well

### 1. Parallel sub-agent dispatch for independent tasks

Dispatching T3 and T4 of Sprint 1B simultaneously — once file scope boundaries were clear
— produced two passing test files with zero conflicts. The same pattern applied to Sprint 2:
all three features touched entirely different packages (`internal/service/plans.go`,
`internal/service/decompose.go`, `internal/mcp/worktree_tool.go`), making parallel
orchestration straightforward and safe. Three features merged cleanly in the same session
with no rebasing required.

This continues a consistent pattern across P25–P28: when decomposition is clean and file
scopes are disjoint, parallel dispatch is highly effective and there is no meaningful
throughput cost to the additional coordination overhead. Declaring file ownership explicitly
in the dispatch message is the direct cause of zero conflicts across Sprint 0 and Sprint 2.

### 2. Sprint 0 and Sprint 1A completed without intervention

Sprint 0's three markdown edit tasks ran in parallel cleanly — different files, no
conflicts, all committed, kbzinit integration test passed on first run, merged without
issues. Sprint 1A (doc-intel guide enrichment) was fully orchestrated by a single sub-agent
that handled all four tasks through to merge without needing any correction from the
orchestrator. The agent correctly identified the guide handler, implemented three additive
struct fields, wrote benchmarks, and merged cleanly. No rework cycle. This is the ideal
pattern.

### 3. Context handoff summary made session resumption instant

The second session began with a structured handoff document that listed exact task IDs,
worktree paths, task statuses, and the critical infrastructure note about the broken
`worktree(action: create)` tool. This made startup essentially instant — no git
archaeology, no status polling loop, no uncertainty about where work had stopped. The value
of a well-structured handoff is hard to overstate and should be a required close-out
artifact for every session that doesn't fully complete.

### 4. The manual worktree fallback procedure worked perfectly

When `worktree(action: create)` timed out for all six Sprint 2 features, the fallback of
`git worktree add` in a loop plus a Python one-liner to write the six YAML state files
worked cleanly: no data loss, no orphaned git state, everything committed in one commit.
The fallback procedure documented in the Sprint 0 skill update was validated by the
situation it was written to address.

### 5. Session-ID follow-up rescued a stalled agent in one round-trip

When the T2 agent (kanbanzai-getting-started edit) hit its output token limit before
committing, sending a minimal targeted follow-up to the *same* session ID resolved the
situation in one round-trip rather than spawning a fresh agent. The key was keeping the
follow-up extremely specific — essentially providing the exact command to run. A fresh
agent spawn would have needed to re-claim an active task and re-read all context; the
session-ID follow-up skipped all of that. This is a useful recovery pattern worth
documenting.

### 6. Sub-agents correctly handled ambiguity rather than guessing

The T3 agent for FEAT-01KPVDDYVETV5 discovered that the spec's AC-009/AC-010 assumed a
plan-level design-document gate for `designing → active` that does not exist in the
codebase. Rather than inventing a gate or writing vacuous tests, the agent surfaced the
ambiguity in its status report. The resolution — regression-test the current behaviour —
was correct. This is the pattern we want: stop and report, don't guess.

### 7. `write_file(entity_id: ...)` for worktree writes

Once this pattern was adopted (after the initial T3 heredoc failure), sub-agents used it
reliably for the rest of the session. The Sprint 2 agents all used it correctly from their
first dispatch, producing clean Go source files with no escaping issues.

---

## What Went Wrong

### 1. The bug we were fixing blocked us from starting  *(significant)*

FEAT-01KPVDDYZ3182 (worktree creation timeout) was on the P28 backlog as something to fix.
It was also the first thing that blocked the first session. Six parallel
`worktree(action: create)` calls all timed out, and the surge of stuck goroutines froze
the MCP server entirely, requiring a full restart. Even after restart, single worktree
creation calls continued to time out — regardless of how many git worktrees existed.

Critically: removing all 33 stale worktrees did **not** fix the timeout. A single
`worktree(action: create)` call on a repository with zero extra worktrees still timed out.
This is important evidence about the root cause (see Recommendations §1 below). The git
operation itself takes under half a second in the terminal, so the slowness is somewhere in
the server-side handler path — likely the entity service scan or a blocking call under
concurrency — not in git.

The result of this failure was that all six Sprint 2 worktrees had to be created manually
via `git worktree add` in the terminal and registered by writing YAML state files by hand.
This added significant setup overhead and pushed the actual implementation work into the
second session.

### 2. `next` timed out with 447 task files  *(significant)*

After the server restart, even a single `next(id: TASK-...)` call timed out. The likely
cause is that the entity service scans all task YAML files when processing a task claim —
checking dependency states, propagating unblocks, reading dependent task statuses. With 447
task files accumulated from 28 plans, this full-directory scan crosses the MCP client
timeout threshold.

This was not a problem in earlier sprints and it will get worse with each plan added. It is
a different problem from the worktree timeout but shares the same structural root cause:
linear scans over unbounded file sets. The `cleanup` tool — which should prune completed
worktrees — also timed out during this session, almost certainly for the same reason.

This issue is not fixed by P28 and needs its own dedicated fix in a future plan.

### 3. Sub-agent context ceiling on 4-task sequential feature chains  *(significant)*

All three Sprint 2 orchestrator agents hit their context window ceiling before completing
their feature chains. This required a full second dispatch wave: first a status-report
prompt to each stopped agent (to understand how far they had got), then resetting three
tasks back to `ready`, then dispatching three fresh agents for the remaining work — six
extra `spawn_agent` calls, approximately 30–40 minutes of wall-clock time.

**Root cause:** The `handoff` tool assembles the orchestrator role context, which includes
the full orchestration skill (~8K tokens of procedure, vocabulary, anti-patterns, and
examples). When a single agent is asked to implement a 4-task sequential chain, it
accumulates: the handoff prompt, the dev plan, the spec, large source files read during
investigation, generated Go code, test output, and knowledge entries. A 4-task chain for a
feature touching a ~500-line source file reliably saturates context before task 3.

A contributing pattern: agents read files exhaustively *before* claiming the task,
accumulating context before getting to the actual work. Earlier task claiming and minimal
reads would extend useful working range significantly (see Recommendations §4).

**What we should change:** For strictly sequential 4-task chains, dispatch one implementer
per task from the top-level session and wait for completion before dispatching the next.
Each sub-agent starts fresh and never accumulates more than one task's worth of context.

### 4. Setup overhead consumed nearly half of the first session  *(moderate)*

Before any implementation happened in the first session, significant effort was spent on:
committing orphaned state from the previous session, diagnosing the worktree timeout,
removing 33 stale worktrees (including a user confirmation round-trip), creating six git
worktrees manually via terminal, writing six YAML state files by hand, committing those,
restarting the server, and confirming recovery.

This is legitimate remediation work, but the volume of it is a signal that accumulated
technical debt from previous sessions — stale worktrees never cleaned up, orphaned state
files left uncommitted — has real carrying costs. Deferred maintenance compounds: each
stale worktree is a file the entity scanner reads on every operation, and 33 of them
together are enough to push scan times over the timeout threshold.

### 5. Terminal heredoc failure in worktrees — a known issue that bit us again  *(moderate)*

Sprint 1B's initial dispatch instructions said "use `terminal` + heredoc for writing Go
files in the worktree" because that was the guidance in the session handoff. The T3
sub-agent's heredoc call failed when the Go code contained embedded double-quoted strings —
a known issue documented in at least three prior knowledge entries (KE-01KN5CXMBWSXE,
KE-01KN5TFZZB7B9, KE-01KN5TFZZB7B9). The correct pattern (`write_file(entity_id: ...)`)
was in the knowledge base but not in the dispatch instructions, so the failure was
preventable.

This issue has now generated four separate knowledge entries across four plans and is
documented in at least two skill files. Documentation is not working as a mitigation.

### 6. FEAT-01KPVDDYQQS1Y lifecycle gap — orphaned reviewing state  *(moderate)*

Sprint 1A's feature was merged at commit `1ecf036` during the first session, but its
entity status was left at `reviewing` with no review report created. This was invisible
until the second session tried to close P28 and found one of the six features still
mid-lifecycle. The previous orchestrator apparently used `override: true` on the
`entity_done` merge gate to force the merge, bypassing the lifecycle.

### 7. `handoff` always assembles orchestrator context for implementer sub-agents  *(moderate)*

When called with `role: implementer-go`, the `handoff` tool returns a prompt with the full
orchestrator procedure, vocabulary, and anti-patterns (~8K tokens) because the feature's
stage binding maps `developing` to the `orchestrate-development` skill. The role parameter
only affected the identity header. This means every implementer sub-agent dispatched via
`handoff` receives guidance about dispatch batches, context compaction, and multi-feature
scoping that is entirely irrelevant to writing a Go function. This is a direct contributor
to the context ceiling problem.

---

## Recurring Patterns Across Plans

The following issues have now appeared in three or more consecutive plans (P25–P28):

| Issue | First seen | Status |
|-------|-----------|--------|
| Terminal heredoc failure in worktrees | P25 | **Unfixed — 4th recurrence** |
| Sub-agent context ceiling on sequential chains | P26 | Unfixed — recurring |
| `handoff` assembling wrong skill for implementer role | P26 | Unfixed — recurring |
| Stale MCP binary confusion after merges | P25 | Partially mitigated (server_info tool added) |
| Stale worktrees accumulating and degrading scan performance | P25 | **New scale: 33 stale in P28** |
| Orphaned lifecycle state (reviewing with no report) | P26 | Unfixed — recurring |

The heredoc issue has been documented four times and is in at least two skill files. It is
not going to be fixed by documentation alone. It needs a product change.

The stale worktree accumulation is now causing concrete performance failures (both the
worktree create timeout and the `next` timeout are at least partially attributable to
scan-over-files overhead). This needs a structural fix, not just periodic manual cleanup.

---

## Personal Experience

### From the first-session orchestrator

**The infrastructure failure was demoralising but the fallback was good.** Hitting the
timeout on the very first real tool call of the session — worktree creation — and then
watching the server freeze was a rough start. But the manual fallback (git loop + Python
YAML writer) worked exactly as designed, which was satisfying. The fact that the Sprint 0
skill updates had documented this exact fallback procedure, and that we needed it
immediately, was an ironic but fitting validation.

**The 447-task `next` timeout is a more serious concern than the worktree timeout.** The
worktree timeout is a one-time cost per feature (create the worktree, done). The `next`
timeout affects every single task claim for the rest of the project's lifetime. It is going
to get worse. If it is not fixed before P29, task claiming may become unreliable across the
board.

**Stale worktree debt has a real carrying cost.** I spent the first half of the session
paying it down. This is time that should have been implementation time. The lesson isn't
"do better cleanup" — it's that cleanup needs to be automatic and low-cost, not a manual
operation that itself times out.

### From the second-session orchestrator

**The context management overhead is real.** Deciding how to split work across sub-agents
without running them into their context ceilings requires constant calibration. I
underestimated how much the orchestrator handoff prompts inflate sub-agent context. I should
have dispatched one-task-at-a-time from the start rather than hoping the agents could handle
full chains.

**The second dispatch wave was wasteful but recoverable.** Three agents hitting the ceiling
simultaneously meant three status-report prompts, three task resets, three fresh dispatches.
Six extra `spawn_agent` calls, ~30–40 minutes of wall-clock time. Entirely avoidable with
the one-task-at-a-time dispatch pattern.

**The AC-009/AC-010 spec ambiguity was genuinely interesting.** The spec referenced a gate
that doesn't exist in the codebase. The agent's response — surface the ambiguity, adopt the
conservative interpretation, regression-test current behaviour — was exactly right. I would
not have caught this by reading the spec myself.

**Lifecycle hygiene is easy to defer and hard to notice when deferred.** The
FEAT-01KPVDDYQQS1Y gap was invisible until I tried to close P28. If I hadn't tried to
close the plan, it would have remained an orphaned `reviewing` feature indefinitely.

---

## Recommendations

| Priority | Change | Owner |
|----------|--------|-------|
| **Critical** | Properly diagnose the worktree create timeout root cause before shipping FEAT-01KPVDDYZ3182's fix. Removing 33 stale worktrees did not fix it — the bottleneck is likely `entitySvc.Get` or `store.List()` scanning task YAML files in the handler path, not `git worktree list`. Add timing instrumentation to the handler before writing retry logic. | product |
| **Critical** | Fix the `next(id: TASK-...)` timeout for large task file sets. Options: (a) a lookup index keyed by task ID, (b) a short-circuit path that reads only the named task + its declared dependency IDs rather than scanning all 447+ files. | product |
| High | Add `write_file(entity_id: ...)` as the **only** worktree file-write instruction in `implement-task` SKILL.md. Remove all heredoc and python3 recommendations for Go source files. | skills |
| High | Fix `handoff` tool: when `role: implementer-go` is passed, assemble `implement-task` skill context, not `orchestrate-development`. | product |
| High | Add sub-agent instruction: "Call `next(id)` as the **very first tool call**. Read only the files you need for the immediate implementation step. Do not read the full source tree before claiming." | skills |
| Medium | Add lightweight automatic worktree cleanup after merge (post-merge hook or low-cost cleanup path). The `cleanup` tool itself timed out — it cannot be the cleanup mechanism if it has the same scan-over-files problem. | product |
| Medium | Add "features stuck in `reviewing` with no registered review report" as a warning-severity attention item in the `status` project dashboard. | product |
| Medium | Document the one-task-at-a-time dispatch pattern for sequential 4-task chains in `orchestrate-development` SKILL.md as the default, not an exception. | skills |
| Medium | Add a review report existence check to the `merge` tool's pre-merge gates (non-bypassable by override for features in `reviewing`). | product |
| Medium | Add a note to `orchestrate-development` SKILL.md: cap parallel MCP write calls with git side-effects at three. Six parallel git-touching calls can freeze the server. | skills |
| Low | Document the session-ID follow-up recovery pattern in the orchestration skill: when an agent hits its context ceiling mid-task, a targeted minimal follow-up to the same session ID is faster and cheaper than a fresh agent spawn. | skills |

---

## Metrics

| Metric | Value |
|--------|-------|
| Features delivered | 6 / 6 |
| Tasks completed | 24 / 24 |
| Blocking review findings | 0 |
| Sub-agent dispatches | ~24 (including retry wave and setup recovery) |
| Context ceiling events | 4 (T2 in session 1; all three Sprint 2 T1 orchestrators in session 2) |
| MCP server freezes requiring restart | 1 |
| Worktrees created manually via terminal | 6 |
| Stale worktrees removed | 33 |
| Worktree conflicts | 0 |
| Lifecycle orphans cleaned up | 1 (FEAT-01KPVDDYQQS1Y) |
| MCP binary rebuilds | 4 |
| `next` timeout events | 1 (post-restart, 447 task files) |