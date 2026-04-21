# Session Retrospective — 2026-04-20

| Field | Value |
|-------|-------|
| Session scope | P23 review + fixes; P24 plan creation; REC-10/11/12 implementation |
| Period | 2026-04-20 |

---

## What We Did

1. **Reviewed the P23 doc-intel implementation** against all four feature specs, using four parallel specialist sub-agents (conformance × 2, quality, testing).
2. **Fixed all blocking and significant findings** from the review across eight files, added six new test functions, and verified the full test suite (31 packages) passed.
3. **Reconciled a divergent git history** — merged three documentation-only commits from `origin/main` into local `main` which was 90 commits ahead.
4. **Read and summarised the full retrospective knowledge base**, then produced a formal recommendations report (`work/reports/retro-recommendations-2026-04-20.md`) with 12 prioritised recommendations.
5. **Created Plan P24** (Retro recommendations) with five features and five design documents, all in `designing` state.
6. **Implemented REC-10, REC-11, REC-12 directly** — three small documentation additions requiring no plan.

---

## What Went Well

### Parallel specialist review was highly effective

Dispatching four sub-agents simultaneously — conformance, quality, testing, and a second conformance agent covering different features — produced a thorough review with no duplicated effort. The structured output format (per-requirement verdict table + blocking/non-blocking findings) made collation straightforward. The review found two genuine blocking spec violations (B-1: missing first-paragraph fallback in summary mode; B-2: connection pool instead of single connection), plus eight significant non-blocking issues, and correctly distinguished them by severity.

The key enabler: each sub-agent received a precise, bounded scope (specific files + specific spec requirements) rather than a vague "review this". Agents that know exactly what to look for produce structured, comparable output.

### Parallel code fixes with disjoint file sets worked without conflicts

Two sub-agents fixed different file sets simultaneously (`store.go` + `intelligence.go` in one; `assembly.go` + `doc_intel_tool.go` in the other), and three more sub-agents added tests to separate test files in parallel. All five completed cleanly, all builds passed, and no merge conflicts occurred. This mirrors the pattern established in P9/P16 — explicit file ownership is the key, not just intent to avoid overlap.

### Sub-agents used Kanbanzai MCP tools correctly

Sub-agents writing design documents called `doc(action: "register")` directly, committed `.kbz/state/` changes after registration, and verified builds with `go build ./...` — all without being explicitly told the MCP was available. This is a meaningful improvement over earlier sessions where agents bypassed MCP tools in favour of shell commands. The design doc sub-agents that explored code first (using `read_file` and `grep`) then wrote accurate documents — no hallucinated API shapes or wrong file paths.

### No Python was used for file editing

Every file edit in this session used `edit_file` directly. No `python3 -c` workarounds were needed. The reason is significant: all changes were in the main project root, not in worktrees. The well-documented friction around `edit_file` in worktrees (`KE-01KN5CXMBWSXE`) simply did not apply because no worktrees were opened. This confirms the scope of the problem — it is worktree-specific, not general.

### `retro synthesise` + `knowledge list` before writing the report

Following the policy established after P7 (KE-01KMT9J3YKJCB), the recommendations report was written only after consulting both `retro synthesise` and `knowledge list`. The result was a report that captured issues from across multiple plans (P7, P9, P16, P17, P19) rather than just the current session. This cross-plan learning would have been missed if the report had been written from in-session memory alone.

### `edit_file` in the embedded skills landed correctly first try (mostly)

REC-10 and REC-12 additions to the embedded skill files worked on the first attempt. Consistent surrounding context in those files made the edits unambiguous.

---

## What Didn't Go Well

### `edit_file` on `planning/SKILL.md` failed twice before succeeding

When adding the partial-task-completion dependency gotcha (REC-11) to `internal/kbzinit/skills/planning/SKILL.md`, `edit_file` failed to apply the edit twice. The fix was to read more of the file to give the tool a more unique anchor — specifically the exact text of the "Simple version" gotcha directly above the insertion point. Once that context was included, the edit applied cleanly.

This is a recurring `edit_file` friction pattern: the tool needs enough unique surrounding text to locate the insertion point. Short sections with generic language (e.g. short bullet-point gotchas that could appear anywhere in a SKILL.md) are particularly prone to this. The workaround is always the same — read more surrounding context before retrying — but it adds latency and requires recognising the pattern quickly rather than retrying blindly.

### Batch feature entity creation failed on the first call

The initial attempt to create all five P24 features in a single `entity(action: "create", entities: [...])` batch call failed for all five because the batch items used `parent_feature` (the correct field for tasks) instead of `parent` (the correct field for features). Features were then created one at a time. This cost five extra round trips.

The root cause is that the `parent` vs `parent_feature` distinction is not prominent in the tool description — `parent` is described as a "list only" filter parameter, which obscures its use as a creation parameter for features. A clearer description would help, and this is now captured as a potential tool-description improvement.

### Sub-agent commits created a noisy git log

The design document workflow produced interleaved auto-commit and manual commit entries because sub-agents committed their `.kbz/state/` changes (document registration records) in separate commits immediately after writing the document files, while working in parallel. The result was a log with entries like `workflow(FEAT-.../design-...): register design` interspersed between `design(p24): ...` commits. Functionally correct, but harder to read than a clean per-feature commit.

This is a known tradeoff with parallel sub-agents and auto-commit behaviour — each sub-agent produces its own commit sequence. The alternative (having sub-agents stage without committing) introduces the state-loss risk documented in KE-01KMT93WRC43V. The current behaviour is preferable.

### The skill distribution gap was not caught until asked

The three REC-10/11/12 changes were made to the correct files for the Kanbanzai project (AGENTS.md and `.kbz/skills/`) but the equivalent changes to the embedded distributed skills (`internal/kbzinit/skills/`) were only added after a follow-up question from the human. The review of "will these reach other projects?" was not part of the implementation checklist.

This is a structural gap specific to Kanbanzai's own development: changes to documentation that affect agent behaviour need to be evaluated against *both* the local project files *and* the embedded distributed skills. These are entirely separate file trees and the connection between them is not obvious. A note in AGENTS.md about this dual-write requirement for skill changes would prevent the miss.

---

## MCP Server Behaviour — Detailed Observations

### Sub-agents had full MCP access and used it correctly

All sub-agents spawned via `spawn_agent` had access to the full Kanbanzai MCP tool set and used it without issues. `doc(action: "register")` calls succeeded from within sub-agents. `entity` transitions worked. `terminal` commands (including `go build ./...`, `go test ./...`, and `git commit`) worked reliably.

No sub-agent experienced an MCP connectivity failure or fell back to shell-based state queries. This is in contrast to earlier sessions (P7) where agents bypassed MCP tools. The difference appears to be in how tasks were framed — sub-agents that received explicit instructions naming specific MCP tool calls to make used them; sub-agents given open-ended "explore and fix" instructions in earlier sessions defaulted to shell commands.

### No stale binary issues this session

`server_info` was not needed — no mysterious tool failures occurred. This is worth noting because the stale binary problem has affected multiple earlier sessions. The absence of issues here likely reflects that the binary was recently rebuilt and the session did not involve mid-session code compilation.

### State isolation between main agent and sub-agents was not a problem

The session used sub-agents for file editing and committing, not for MCP state transitions (entity transitions, doc approvals). The main agent handled all lifecycle state changes. This clean separation avoided the state-isolation problem documented in KE-01KMT93WRC43V — sub-agents never ran git operations that could have affected uncommitted `.kbz/state/` changes.

---

## File Editing — Detailed Observations

### `edit_file` — reliable for main project root, context-sensitive

`edit_file` was used for every file change in this session. It worked first-try for large files with clear, unique surrounding context and for new file creation (overwrite mode). It required extra context-reading passes for short files with repetitive structure (the planning skill gotchas section).

**No Python was used.** The `python3 -c` workaround documented in KE-01KN5CXMBWSXE was completely absent from this session. This is directly because no worktree operations were performed — all edits were to the main project root, where `edit_file` works as designed.

The friction is real and the knowledge base is correct, but the scope is narrower than general "file editing friction" — it is specifically a worktree isolation issue, not an `edit_file` reliability issue in the main project.

### Overwrite mode for new design documents worked cleanly

Five design documents were written using `edit_file` in `overwrite` mode (writing new files). All succeeded without issue. Sub-agents that wrote design documents read existing design docs first to calibrate style and section structure, producing consistent documents without explicit style guidance in the delegation message.

---

## Patterns to Carry Forward

| Pattern | Observation |
|---------|-------------|
| Parallel specialist review with bounded scope per agent | Effective. Use for any feature with >5 spec requirements. |
| Parallel code fixes with explicit file ownership | Zero conflicts across five sub-agents. Make file ownership explicit in the delegation message. |
| Read more context before retrying a failed `edit_file` | Retrying with identical context produces the same failure. Always read the surrounding section first. |
| Main agent owns MCP state transitions; sub-agents own file edits | Clean separation prevents state-loss risk. Maintain this boundary. |
| Sub-agents need explicit MCP tool call instructions, not open-ended exploration | Agents given specific tool calls to make use them. Agents given "figure it out" prompts may fall back to shell. |
| Dual-write required for skill changes in Kanbanzai's own development | Changes to `.kbz/skills/` and `AGENTS.md` also need corresponding changes in `internal/kbzinit/skills/` to reach other projects. |

---

## Open Questions

1. **Should AGENTS.md include a note about the dual-write requirement for skill changes?** The gap was caught this session only because the human asked. A checklist item in AGENTS.md ("if you changed `.kbz/skills/` or `AGENTS.md`, also check `internal/kbzinit/skills/`") would close this.

2. **Is the `parent` vs `parent_feature` distinction worth a tool description improvement?** The current description of `parent` as "(list only)" obscures its use as a feature-creation field. Low priority but genuine friction.

3. **P24 designs are awaiting human approval.** Five design documents are registered and in `designing` state. Once approved, specs can be written and development can begin.