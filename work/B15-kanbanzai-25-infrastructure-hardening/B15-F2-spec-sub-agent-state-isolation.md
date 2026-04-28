# Specification: Sub-Agent State Isolation

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Updated  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPG44T5B (sub-agent-state-isolation)                     |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §4         |

---

## 1. Purpose

This specification defines requirements for protecting Kanbanzai workflow state
from destruction by sub-agent git operations. The `handoff` tool must commit any
pending `.kbz/state/` changes before assembling a sub-agent prompt, ensuring
that state written by orchestrator MCP tool calls is persisted before a
sub-agent can disrupt the working tree.

---

## 2. Goals

1. Workflow state written by orchestrator MCP tool calls survives subsequent
   sub-agent git operations (stash, checkout, reset).
2. The safeguard is automatic — it requires no orchestrator action beyond
   calling `handoff` as normal.
3. The safeguard is best-effort — a commit failure does not block the handoff.
4. Only `.kbz/state/` files are included in the protective commit; unrelated
   working tree changes are not touched.
5. No empty commits are created when there are no pending state changes.

---

## 3. Scope

### 3.1 In Scope

- `internal/mcp/handoff_tool.go` — pre-dispatch state commit logic.
- Git commit logic that stages and commits only files under `.kbz/state/`.
- Logging behaviour when the pre-dispatch commit fails.
- Unit and integration tests covering the above.

### 3.2 Out of Scope

- Preventing sub-agents from running git operations (requires sandboxing beyond
  current architecture).
- Committing non-state changes (code, documents, index, cache).
- Modifying the `next` tool — it claims work for the current agent, not a
  sub-agent.
- Skill-level anti-pattern documentation (deferred to V3.0).
- Widening the commit scope to other `.kbz/` directories (open question deferred
  to implementation).

---

## 4. Requirements

### 4.1 Pre-Dispatch State Commit

**REQ-01.** When `handoff` is called, it MUST check for uncommitted changes
under `.kbz/state/` before assembling the sub-agent prompt.

**REQ-02.** If uncommitted changes exist under `.kbz/state/`, `handoff` MUST
create a git commit containing those changes before proceeding with context
assembly.

**REQ-03.** The commit MUST include only files under `.kbz/state/`. Files
outside `.kbz/state/` MUST NOT be staged or committed.

**REQ-04.** The commit message MUST be:
`chore(kbz): persist workflow state before sub-agent dispatch`

**REQ-05.** If no uncommitted changes exist under `.kbz/state/`, no commit
MUST be created. Empty commits are prohibited.

### 4.2 Failure Handling

**REQ-06.** If the pre-dispatch commit fails for any reason (e.g., git lock
contention, repository not initialised), `handoff` MUST log a warning
describing the failure.

**REQ-07.** A pre-dispatch commit failure MUST NOT prevent `handoff` from
completing context assembly and returning its result. The safeguard is
non-blocking.

### 4.3 No Behavioural Change to Context Assembly

**REQ-08.** The pre-dispatch commit MUST occur before context assembly begins.
The content and format of the assembled sub-agent prompt MUST be identical
to what it would be without this change.

---

## 5. Acceptance Criteria

**AC-07.** When `handoff` is called and `.kbz/state/` has uncommitted changes,
a commit is created before context assembly.

**AC-08.** The commit includes only files under `.kbz/state/`, not other
working tree changes.

**AC-09.** The commit message is exactly:
`chore(kbz): persist workflow state before sub-agent dispatch`

**AC-10.** If no `.kbz/state/` changes exist, no commit is created.

**AC-11.** If the commit fails, `handoff` logs a warning and proceeds without
error.

---

## 6. Dependencies and Assumptions

- The working directory is a git repository. If it is not, the pre-commit check
  must degrade gracefully (REQ-06 applies).
- The git user identity is configured in the environment. If it is not, the
  commit will fail and REQ-06/REQ-07 apply.
- `.kbz/state/` is the canonical location for all MCP-managed workflow state.
  Files written by MCP tool calls during orchestration are stored there and
  nowhere else.
- This fix addresses tool-level safeguarding only. The corresponding skill-level
  anti-pattern guidance ("State Destruction via Git Operations") is out of scope
  for this feature and is deferred to V3.0.

---

## 7. Verification

| Criteria     | Method                                                               |
|--------------|----------------------------------------------------------------------|
| AC-07        | Unit test: call `handoff` with dirty `.kbz/state/`; assert commit created |
| AC-08        | Unit test: assert staged files are limited to `.kbz/state/**`        |
| AC-09        | Unit test: assert commit message equals expected string              |
| AC-10        | Unit test: call `handoff` with clean state; assert no new commit     |
| AC-11        | Unit test: simulate commit failure; assert warning logged, handoff returns |
| REQ-08       | Integration test: assembled prompt is unchanged from pre-fix baseline |

All tests must pass under `go test ./...` and `go test -race ./...` with no
regressions to existing handoff behaviour.