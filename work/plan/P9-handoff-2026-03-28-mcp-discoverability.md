# Session Handoff: MCP Discoverability & Reliability

| Document | Session Handoff — MCP Discoverability & Reliability |
|----------|-----------------------------------------------------|
| Created  | 2026-03-28T14:10:20Z                                |
| From     | Planning session (context exhausted)                 |
| Next     | Specification and task breakdown                     |

---

## Where We Are

The **design document is approved**: `work/design/mcp-discoverability-and-reliability.md`

The next step is **specification → dev-plan → tasks → implementation**.

---

## What Happened This Session

### 1. P7 Retrospective Analysis

Read and analysed `work/reports/p7-implementation-retrospective.md`. Identified five failure modes in agent ↔ MCP server interaction, all rooted in the server not making it easy enough for agents to do the right thing.

### 2. Migration Audit — Hardening Branch Never Merged

**Critical discovery:** The hardening feature branch (`feature/FEAT-01KMKRQWF0FCH-hardening`) was never merged to `main`. The 2.0 tool redesign proceeded from `main`, so all hardening work was lost:

- **MCP safety annotations** (`readOnlyHint`, `destructiveHint`, `idempotentHint`) on all tools — gone
- **`annotations_test.go` canary test** — gone
- **`doc_record_refresh` tool** (fixes stale content hash) — gone
- **Improved error messages** — gone
- **`doc_supersession_chain`** — service method exists but unreachable via MCP

Full audit report committed: `work/reports/kanbanzai-2.0-migration-audit.md`

### 3. MCP Spec Review

Reviewed the MCP 2025-11-25 specification (tools, prompts, resources, authorization). Key findings:

- **Tool annotations** are the spec-blessed mechanism for permission hints / auto-approval. The mcp-go v0.45.0 library supports `ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`, `OpenWorldHint`, and `Title`.
- **Prompts** are user-controlled (slash commands) — interesting for portable instruction delivery but deferred. They don't solve the reliability problem where agents bypass tools they already have.
- **Resources** could expose knowledge/skills as URIs — low priority until clients proactively consume them.

### 4. Design Document

Created and approved `work/design/mcp-discoverability-and-reliability.md` with six features:

| Feature | What | Priority |
|---------|------|----------|
| **A: Tool Annotations** | `readOnlyHint`/`destructiveHint`/`idempotentHint`/`openWorldHint` on all 22 tools + canary test | High |
| **B: Tool Titles** | Human-readable `title` field via `mcp.WithTitleAnnotation()` | High |
| **C: Improved Descriptions** | Rewrite descriptions to guide agents toward MCP tools, away from shell | High |
| **D: Response Nudges** | `finish` warns when last task in feature has no retro signals | Medium |
| **E: `doc` Refresh Action** | New `doc(action: refresh)` — fixes stale hash without delete/re-register | Medium |
| **F: `doc` Chain Action** | Wire existing `SupersessionChain()` as `doc(action: chain)` | Low |

### 5. Decisions Made

- **Nudges are informational, not blocking.** Nudges are a `nudge` string field in the response JSON. They do not set `isError: true` or block operations. Start soft, escalate if ineffective after a full cycle. (Human decision, recorded in design §11.3)
- **Instructions belong in the server, not AGENTS.md.** Generic Kanbanzai workflow rules should be built into tool annotations, descriptions, and responses — not project-specific files that agents may not read.
- **Prompts deferred.** MCP Prompts are interesting for portable instruction delivery but are a user-facing convenience, not a reliability fix. Parked for a future design.

### 6. Open Questions (carry forward)

- **§11.1 (developer decides):** `branch` Tier 1 vs 2? `retro report` auto-composing `synthesise`?
- **§11.2 (agent decides):** Does a retro reminder in `handoff` prompts actually change behaviour? What description phrasing best steers agents toward MCP tools?

### 7. Key Files

| File | What |
|------|------|
| `work/design/mcp-discoverability-and-reliability.md` | **Approved design** — the basis for spec work |
| `work/reports/kanbanzai-2.0-migration-audit.md` | Full audit of what was lost in the 2.0 migration |
| `work/reports/p7-implementation-retrospective.md` | The retrospective that started this analysis |
| `work/spec/hardening.md` §10 | Original annotation spec (1.0 tools) — reference for classification model |
| `feature/FEAT-01KMKRQWF0FCH-hardening` | Unmerged branch with reference implementations (do not merge, use as reference only) |

### 8. What to Do Next

1. Write a specification for the six features (A–F) with testable acceptance criteria.
2. Create a plan entity (P9 or similar).
3. Run `decompose propose` to break into tasks.
4. Implement — Features A–C can be a single commit; D–F are independent.