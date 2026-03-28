# P7 Implementation Retrospective

| Document | P7 Developer Experience — Implementation Retrospective |
|----------|--------------------------------------------------------|
| Status   | Draft                                                  |
| Created  | 2026-03-28T13:12:43Z                                  |
| Plan     | P7-developer-experience                                |
| Author   | sambeau (with AI analysis)                             |

---

## 1. Summary

P7 (Developer Experience Improvements) delivered three features across 12 tasks
in a single implementation session:

| Feature | Tasks | Commits | Scope |
|---------|-------|---------|-------|
| `server-info-tool` (FEAT-01KMT40GZSMHB) | 5/5 | 2 | New `server_info` MCP tool, `buildinfo` + `install` packages, Makefile, post-merge install |
| `human-friendly-id-display` (FEAT-01KMT40KKZZR5) | 5/5 | 3 | Label field, split ID input, `display_id` promotion, slug display, dashboard label column |
| `review-naming-and-folder-conventions` (FEAT-01KMT40P0AGS7) | 2/2 | 1 | File migrations, doc updates (no Go code) |

All 22 test packages pass under `go test -race ./...`. No regressions.

The implementation itself was successful, but the *process* exposed three
significant issues that warrant analysis and corrective action.

---

## 2. What Went Well

**Parallel feature execution.** Three independent features were implemented
concurrently using sub-agents, with no code conflicts between them. The task
decomposition correctly identified independence boundaries.

**Spec quality.** All three specifications had clear acceptance criteria that
mapped directly to implementation tasks. No ambiguity required human
clarification during implementation.

**Test coverage held.** The cross-cutting `display_id` changes touched many
tools (entity, status, next, handoff, finish, side_effects) without breaking
any existing tests. The test suite caught real issues during development.

**Quick recovery.** When workflow state was corrupted (see §3.2), the fix was
straightforward — re-run the MCP tool calls. No code was lost.

---

## 3. What Went Wrong

### 3.1 Agent bypassed MCP tools for state queries

**Observation.** The orchestrating agent opened the session by running raw
shell commands to read `.kbz/state/` YAML files:

```
find .kbz/state/ -name "P7-*" -o -name "FEAT-*" | xargs grep -l "P7"
cat .kbz/state/features/FEAT-01KMT40GZSMHB-server-info-tool.yaml
```

and bash for-loops to extract feature slugs:

```
for f in FEAT-01KMKRQSD1TKK FEAT-01KMRX1F47Z94 ...; do
  slug=$(grep "^slug:" .kbz/state/features/${f}-*.yaml | sed 's/slug: //')
  echo "${f} -> ${slug}"
done
```

instead of using the Kanbanzai MCP tools (`status`, `entity list`, `entity get`,
`doc list`).

When challenged, the agent self-corrected:

> "Works fine! I was being sloppy — falling back to raw file reads out of habit
> instead of using the structured tools."

But then relapsed during context gathering for sub-agents, using more bash
scripts to look up document records and feature slugs.

**Impact.** Low in this instance — the YAML files are well-organized and the
shell commands returned correct results. But this bypasses the value
proposition of the MCP tools: lifecycle-aware context, derived state (what's
blocked, what's ready), and structured output.

**Root cause.** Three contributing factors:

1. **No thinking mode.** The agent was not in extended-thinking mode, which
   reduced upfront planning about *how* to approach the task.
2. **Weak signposting.** AGENTS.md mentions MCP tools but doesn't make their
   use mandatory for state queries. The instruction "follow bootstrap-workflow"
   is too indirect.
3. **Path of least resistance.** Shell commands require no schema knowledge and
   return results faster than learning a tool's parameter format.

**Suggested fix.** Add a hard rule to AGENTS.md:

> **State queries must use Kanbanzai MCP tools.** Do not read `.kbz/state/`
> files directly via `cat`, `grep`, or shell scripts. Use `status`,
> `entity list/get`, `doc list/get`, `next`, and other MCP tools. These
> tools enforce lifecycle rules, compute derived state (what's blocked,
> what's ready), and return structured context that raw YAML does not.

### 3.2 Workflow state destroyed by sub-agent git stash

**Observation.** After spawning sub-agents for implementation, the
orchestrating agent discovered that all three features had reverted from
`developing` to `specifying`, specs from `approved` to `draft`, and tasks
from `ready`/`active` to `queued`.

The agent offered two explanations in-session:

> "The MCP server is using in-memory state that was persisted to disk by the
> sub-agents. The sub-agents didn't actually see the spec approvals because
> they were working off a stale MCP binary."

and:

> "The Kanbanzai state still shows tasks as `ready` because the sub-agents'
> `finish` calls were in their own MCP sessions."

**Both explanations were wrong.**

**What actually happened (verified by forensic analysis):**

1. The main agent approved specs and advanced features via MCP tool calls.
   These changes were written to `.kbz/state/` files on disk by the MCP
   server but were **not committed to git**.

2. A sub-agent (human-friendly-id) followed the AGENTS.md instruction:
   *"Run `git status`. If there are uncommitted changes from previous work,
   commit or stash before starting new work."*

3. The sub-agent ran `git stash`, which saved all uncommitted changes —
   **including the `.kbz/state/` modifications** — and reverted the working
   tree to HEAD.

4. HEAD still contained the old state (`specifying`, `draft`, `queued`).

5. The stash was never popped. It remains as `stash@{0}` with the message
   `"Stash previous work before label/display-id tasks"`.

**Evidence:**

| Location | Feature status | Spec status |
|----------|---------------|-------------|
| `stash@{0}` (the good state) | `developing` | `approved` |
| HEAD commit (the old state) | `specifying` | `draft` |
| Disk after stash (what the agent found) | `specifying` | `draft` |

**Impact.** Medium. Required re-running all approval and transition commands.
No code was lost, but debugging time was wasted and the agent's incorrect
explanations added confusion.

**Root cause.** An architectural tension between three facts:

1. `.kbz/state/` is tracked in git (necessary for this project, which
   develops the tool itself).
2. MCP tool calls modify `.kbz/state/` files without committing them.
3. AGENTS.md instructs agents to stash uncommitted changes before new work.

These three things together create a trap: MCP state changes are silently
destroyed by git operations that agents are *instructed to perform*.

**Suggested fixes (choose one or combine):**

1. **Commit workflow state before spawning sub-agents.** The orchestrating
   agent should run a `workflow(...)` commit for `.kbz/state/` changes
   before any sub-agent is spawned. This is the cleanest fix.

2. **Add an AGENTS.md rule for sub-agent orchestration:**

   > When spawning sub-agents that will perform git operations, first commit
   > any pending `.kbz/state/` changes with a `workflow(...)` commit message.
   > MCP tool calls (approve, transition, next, finish) modify `.kbz/state/`
   > files on disk without committing them. A sub-agent that runs `git stash`
   > or `git checkout` will destroy these changes.

3. **Tell sub-agents to exclude `.kbz/` from git operations.** For example,
   `git stash -- ':!.kbz/'`. Fragile but targeted.

4. **For non-development projects:** add `.kbz/state/` to `.gitignore`. This
   is the standard Kanbanzai configuration and eliminates the issue entirely.
   Not appropriate for this project where state is intentionally tracked.

### 3.3 Confabulated root-cause explanations

**Observation.** When the state corruption was discovered, the agent produced
two confident-sounding but entirely wrong explanations:

- "Stale MCP binary" — sub-agents weren't calling MCP state tools at all
- "MCP session isolation" — irrelevant; all sessions share one server process

The agent never ran `git stash list` or `git stash show` to investigate. It
jumped from symptom to plausible narrative without verifying.

**Impact.** Medium. The wrong explanation didn't prevent recovery (the fix was
to re-run the MCP calls), but it obscured the real cause and could have led to
incorrect preventive measures.

**Root cause.** Under time pressure, the agent pattern-matched to a familiar
failure mode (stale binary — a known issue that P7 itself was designed to fix)
rather than investigating the actual cause. This is a known failure mode of
LLM-based agents: generating plausible explanations from training data rather
than from evidence.

**Suggested fix.** Add a debugging discipline rule to AGENTS.md:

> **When workflow state is unexpected, investigate before explaining.** Run
> `git stash list`, `git log --oneline -5`, and `git diff --stat` before
> forming a hypothesis. Do not offer a root cause explanation until you have
> evidence. "I don't know yet — let me check" is always better than a
> confident wrong answer.

---

## 4. Sub-Agent Effectiveness

Three waves of sub-agents were spawned during P7:

| Wave | Agents | Result |
|------|--------|--------|
| 1 | review-naming (1), server-info buildinfo+install (1), human-friendly-id label+split (1) | review-naming ✅, server-info ✅, human-friendly-id ⚠️ ran out of context |
| 2 | server-info CLI+MCP+merge (1), human-friendly-id all remaining (1) | Both ⚠️ ran out of context, but left significant progress |
| 3 | status dashboard display_id (1), next/handoff/finish display_id (1) | Both ✅ |

**Observations:**

- **Context window exhaustion** was the primary failure mode. Two of three
  Wave 1 agents and both Wave 2 agents ran out of context. Large Go codebases
  with many files to read consume context quickly.

- **Smaller, focused tasks succeed.** Wave 3 agents had narrow scope ("update
  this one file") and completed reliably. Wave 1-2 agents had broad scope
  ("implement these 3-5 tasks") and frequently failed.

- **Incomplete work was still useful.** Even when agents ran out of context,
  their partial work (staged changes, committed code) was recoverable. The
  orchestrating agent could verify and commit what was done.

- **Sub-agents don't call MCP workflow tools.** None of the sub-agents called
  `finish` or other workflow management tools — they correctly limited
  themselves to code implementation. Workflow state management should remain
  with the orchestrating agent.

**Recommendation for future sub-agent use:**

- Limit each sub-agent to 1-2 tasks or a single focused file-editing scope
- Reserve workflow state management (approve, transition, finish) for the
  orchestrating agent
- Always commit `.kbz/state/` changes before spawning sub-agents

---

## 5. Corrective Actions

| # | Action | Priority | Scope |
|---|--------|----------|-------|
| 1 | Add "must use MCP tools for state queries" rule to AGENTS.md | High | AGENTS.md |
| 2 | Add "commit .kbz/state/ before spawning sub-agents" rule to AGENTS.md | High | AGENTS.md |
| 3 | Add "investigate before explaining" debugging discipline to AGENTS.md | Medium | AGENTS.md |
| 4 | Drop orphaned `stash@{0}` (contains superseded state) | Low | Git cleanup |
| 5 | Consider sub-agent scope guidelines (1-2 tasks max per agent) | Medium | AGENTS.md |

---

## 6. Metrics

| Metric | Value |
|--------|-------|
| Features delivered | 3/3 |
| Tasks completed | 12/12 |
| Commits | 6 (code) + 0 (workflow state — see §3.2) |
| Test packages passing | 22/22 (with `-race`) |
| Sub-agents spawned | 7 |
| Sub-agents completed successfully | 4 |
| Sub-agents ran out of context | 3 |
| State corruption incidents | 1 (recovered) |
| Incorrect root-cause explanations | 2 |