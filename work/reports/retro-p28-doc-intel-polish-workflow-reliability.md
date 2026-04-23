# Retrospective: P28 — Doc-Intel Polish and Workflow Reliability

| Field | Value |
|-------|-------|
| Scope | P28-doc-intel-polish-workflow-reliability |
| Plan | Doc-Intel Polish and Workflow Reliability |
| Date | 2026-04-23 |
| Features | 6 delivered, 6 merged |
| Tasks | 24 / 24 done |
| Author | orchestrator (Claude Sonnet 4.6) |

---

## Summary

P28 addressed four P27 retrospective findings: a plan lifecycle gap (`proposed → active`
shortcut), missing dev-plan auto-registration in `decompose apply`, worktree creation
timeouts under load, and a set of doc-intel guide enrichments. All six features were
delivered and merged in a single session, continuing from a handoff that already had Sprint
0 and Sprint 1A merged and Sprint 1B partially done.

The session went well overall. The parallel dispatch pattern worked, the code quality was
high, and no blocking review findings were raised. There were, however, three significant
friction patterns that recurred and deserve systematic fixes.

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
throughput cost to the additional coordination overhead.

### 2. Context handoff summary

The session began with a structured handoff document (provided by the previous session's
orchestrator) that listed exact task IDs, worktree paths, task statuses, and the critical
infrastructure note about the broken `worktree(action: create)` tool. This made session
startup essentially instant — no git archaeology, no status polling loop, no uncertainty
about where work had stopped. The value of a well-structured handoff is hard to overstate.

### 3. Sub-agents correctly handled ambiguity rather than guessing

The T3 agent for FEAT-01KPVDDYVETV5 discovered that the spec's AC-009/AC-010 ("designing →
active without/with approved design doc still rejected/succeeds") assumed a plan-level
design-document gate that does not exist in the codebase. Rather than inventing a gate or
silently writing vacuous tests, the agent surfaced the ambiguity in its status report. The
resolution — regression-test the current behaviour — was correct. This is the pattern we
want: stop and report, don't guess.

### 4. `write_file(entity_id: ...)` for worktree writes

Once this pattern was adopted (after the T3 heredoc failure), sub-agents used it reliably
for the rest of the session. The Sprint 2 agents all used it correctly from their first
dispatch, producing clean Go source files with no escaping issues.

### 5. The worktree timeout fix fixed itself

FEAT-01KPVDDYZ3182 was implemented correctly and merged during this session. The very
infrastructure problem that required manual workarounds throughout P28 (pre-creating all
worktrees via `git worktree add` in the terminal and manually writing YAML state files) is
now fixed. The next plan will benefit immediately.

---

## What Went Wrong

### 1. Sub-agent context ceiling on 4-task sequential feature chains  *(significant)*

All three Sprint 2 orchestrator agents hit their context window ceiling before completing
their feature chains. This required a full second dispatch wave: first a status-report
prompt to each stopped agent (to understand how far they had got), then resetting three
tasks back to `ready`, then dispatching three fresh agents for the remaining work.

**Root cause:** The `handoff` tool assembles the orchestrator role context, which includes
the full orchestration skill (~8K tokens of procedure, vocabulary, anti-patterns, and
examples). When a single agent is asked to implement a 4-task sequential chain, it
accumulates: the handoff prompt, the dev plan, the spec, large source files read during
investigation, generated Go code, test output, and knowledge entries. A 4-task chain for a
feature touching a ~500-line source file reliably saturates context before task 3.

**What we should change:** For strictly sequential 4-task chains where each task depends on
the previous, do not dispatch a single orchestrator to handle the full chain. Instead,
dispatch one implementer per task from the top-level session, wait for each to complete,
then dispatch the next. This is more round-trips but each sub-agent starts fresh and never
accumulates more than one task's worth of context. The orchestrator role prompt (which is
8K tokens) is also excessive for implementer sub-agents — but this is a product issue
described below.

### 2. Terminal heredoc failure in worktrees — a known issue that bit us again  *(moderate)*

Sprint 1B's initial dispatch instructions said "use `terminal` + heredoc for writing Go
files in the worktree", because that was the guidance in the session handoff. In practice,
the T3 sub-agent's heredoc call failed when the Go code contained embedded double-quoted
strings — a known issue documented in at least three knowledge entries (KE-01KN5CXMBWSXE,
KE-01KN5TFZZB7B9, and the new KE-01KPW39640YG9 from this session).

The correct pattern — `write_file(entity_id: "FEAT-...", path: "...", content: "...")` —
was already known and in the knowledge base, but was not in the dispatch instructions I
sent. I corrected this for all subsequent dispatches, but the initial failure wasted one
sub-agent run.

**What we should change:** The `write_file(entity_id: ...)` pattern for worktree file
creation should be the default instruction in all implementer dispatches, not an optional
note. The heredoc/python3 workarounds are now strictly inferior and should be retired from
task instructions.

### 3. FEAT-01KPVDDYQQS1Y lifecycle gap — orphaned reviewing state  *(moderate)*

Sprint 1A's feature (doc-intel guide enrichment) was merged to main at commit `1ecf036`
during the previous session, but its entity status was left at `reviewing`. No review
report was ever created. This was discovered only at the end of this session when I tried
to close P28 and found one of its six features still mid-lifecycle.

**Root cause:** The previous session's orchestrator merged the feature but skipped the
review report creation step of the close-out sequence. The `merge(action: execute)` tool
does not check for a review report — it only checks that the feature is in `done` status,
which itself requires a registered review report. The sequence that was followed was
apparently: mark all tasks done → transition to `reviewing` → merge directly, bypassing
the report + `done` transition.

Actually, looking more carefully: the merge tool checks `entity_done` as a gate, which
means the previous orchestrator must have used `override: true`. The feature was merged but
the entity never reached `done`.

**What we should change:** The `merge` tool's `entity_done` gate should not be bypassable
by override for features in `reviewing` without a registered review report. A merged-but-
reviewing feature is an invisible lifecycle debt. Additionally, the close-out checklist
should be more prominent in the orchestration skill — the current Phase 6 is detailed but
appears to have been skipped in the previous session.

### 4. `handoff` always assembles orchestrator context for implementer sub-agents  *(moderate)*

When I called `handoff(task_id: "TASK-01KPVFY051XFC", role: "implementer-go")`, the
returned prompt still had the full orchestrator procedure, vocabulary, and anti-patterns
from the orchestrate-development skill. The `role` parameter affected the identity section
("Senior Go engineer") but not the procedure or anti-pattern sections, which remained
orchestrator-oriented.

This means every implementer sub-agent dispatched via `handoff` receives ~8K tokens of
orchestration guidance (dispatch batches, context compaction, multi-feature scoping) that
is entirely irrelevant to the task of writing a function in a Go file. This is a significant
token overhead that contributes directly to the context ceiling problem above.

**What we should change:** The `handoff` tool's context assembly should use the skill
matching the role parameter, not the feature's stage binding skill. An `implementer-go`
dispatch should assemble the `implement-task` skill, not `orchestrate-development`.

---

## Recurring Patterns Across Plans

The following issues have now appeared in three or more consecutive plans (P26, P27, P28):

| Issue | First seen | Status |
|-------|-----------|--------|
| Terminal heredoc failure in worktrees | P25 | Unfixed — recurring |
| Sub-agent context ceiling on sequential chains | P26 | Unfixed — recurring |
| `handoff` assembling wrong skill for implementer role | P26 | Unfixed — recurring |
| Stale MCP binary confusion after merges | P25 | Partially mitigated (server_info tool added) |

The heredoc issue in particular has now generated four separate knowledge entries across
four plans and is documented in at least two skills. It is clearly not going to be fixed
by documentation alone — it needs a product change: either deprecate the heredoc
recommendation entirely in favour of `write_file(entity_id: ...)`, or make the tool emit
a clear error when called from a worktree path.

---

## Personal Experience (Orchestrator Perspective)

This was the first session where I operated as the top-level orchestrator for an entire
plan completion, rather than being handed individual tasks. A few honest observations:

**The context management overhead is real.** Deciding how to split work across sub-agents
without running them into their context ceilings requires constant calibration. I
underestimated how much the orchestrator handoff prompts inflate sub-agent context. Sending
a full-feature orchestrator prompt (~8K tokens) plus a 300-line source file plus generated
tests to a single agent is a context budget problem before the agent writes a single line.
I should have dispatched one-task-at-a-time from the start rather than hoping the agents
could handle full chains.

**The second dispatch wave was wasteful but recoverable.** Three agents hitting the ceiling
simultaneously meant three status-report prompts, three task resets, three fresh dispatches.
That is six extra spawn_agent calls. In terms of wall-clock time this probably added 30–40
minutes. It was avoidable with the one-task-at-a-time dispatch pattern.

**The AC-009/AC-010 spec ambiguity was genuinely interesting.** The spec referenced a gate
(`designing → active` requires approved design doc for plans) that the implementing agent
couldn't find in the codebase. This was a real spec gap, not an implementation error. The
agent's response — surface the ambiguity, adopt the conservative interpretation, regression-
test current behaviour — was exactly right. I would not have caught this by reading the
spec myself. The agent earned its keep here.

**Lifecycle hygiene is easy to defer and hard to notice when deferred.** The FEAT-01KPVDDYQQS1Y
lifecycle gap was invisible until I tried to close P28. If I hadn't tried to close the plan,
it would have remained an orphaned `reviewing` feature indefinitely. This makes me think
the project overview dashboard should flag features that have been in `reviewing` for more
than N days with no review report registered.

**Parallel dispatch across features genuinely works.** Three completely different packages,
three agents, no conflicts. The key precondition — disjoint file scopes — was satisfied by
construction (the plan decomposed features along package boundaries). When that condition
holds, parallel dispatch is essentially free.

---

## Recommendations

| Priority | Change | Owner |
|----------|--------|-------|
| High | Add `write_file(entity_id: ...)` as the primary worktree file-write instruction in `implement-task` SKILL.md; remove heredoc recommendation | skills |
| High | Fix `handoff` tool: when `role: implementer-go` is passed, assemble `implement-task` skill, not `orchestrate-development` | product |
| Medium | Add "features stuck in `reviewing` with no review report" as an attention item in the `status` project dashboard | product |
| Medium | Document the one-task-at-a-time dispatch pattern for sequential 4-task chains in `orchestrate-development` SKILL.md | skills |
| Medium | Add a review report existence check to the `merge` tool's pre-merge gates (non-bypassable for features in `reviewing`) | product |
| Low | Explore whether the orchestrator skill procedure sections can be stripped from `handoff` output when `role: implementer-go` is specified | product |

---

## Metrics

| Metric | Value |
|--------|-------|
| Features delivered | 6 / 6 |
| Tasks completed | 24 / 24 |
| Blocking review findings | 0 |
| Sub-agent dispatches | ~18 (including retry wave) |
| Context ceiling events | 3 (all Sprint 2 T1 orchestrators) |
| Worktree conflicts | 0 |
| Lifecycle orphans cleaned up | 1 (FEAT-01KPVDDYQQS1Y) |
| MCP binary rebuilds | 4 |