# Kanbanzai 2.0 Migration Audit: Lost Features & Recommendations

| Document | 2.0 Migration Audit — Hardening Features Lost in Tool Redesign |
|----------|----------------------------------------------------------------|
| Status   | Draft                                                          |
| Created  | 2026-03-28T13:43:41Z                                          |
| Author   | sambeau (with AI analysis)                                     |
| Related  | P7 Implementation Retrospective, Hardening Spec §10            |

---

## 1. Summary

During an investigation into why MCP tool safety annotations were missing from the Kanbanzai 2.0 tools, a broader discovery was made: **the entire hardening feature branch (`feature/FEAT-01KMKRQWF0FCH-hardening`) was never merged to `main`**. The Kanbanzai 2.0 tool surface redesign then proceeded from `main`, and the branch is now 506 files diverged. All hardening-phase implementations — working, tested code — are effectively lost.

This report documents what was lost, what was preserved, and recommends corrective actions.

### How this was discovered

The P7 implementation retrospective (§3.10 of the MCP Server Design Issues report) identified "permission friction" — the human overseer must approve too many tool calls, creating a babysitting burden. Investigation revealed that the safety annotations designed to solve this problem were implemented on the hardening branch but never reached `main`.

---

## 2. Root Cause: Unmerged Hardening Branch

### Evidence

```
$ git merge-base --is-ancestor feature/FEAT-01KMKRQWF0FCH-hardening main
# exit code: 1 (NOT an ancestor — never merged)

$ git branch -a | grep harden
+ feature/FEAT-01KMKRQWF0FCH-hardening

$ git diff main..feature/FEAT-01KMKRQWF0FCH-hardening --stat | tail -1
506 files changed, 14794 insertions(+), 115098 deletions(-)
```

The hardening branch contains commit `8ae3b7e` ("feat(FEAT-01KMKRQWF0FCH): doc_record_refresh tool, MCP safety annotations, error message audit (Wave 1)") and subsequent work — none of which exists on `main`.

### Timeline

1. Hardening feature (FEAT-01KMKRQWF0FCH) was specified, planned, and implemented on its feature branch.
2. Task TASK-01KMNA4SKB6H5 (MCP safety annotations) was completed — annotations added to all 85 1.0 tools.
3. Task TASK-01KMNA3GR1AET (doc_record_refresh) was completed — service method, MCP tool, and tests added.
4. The branch was **never merged to `main`**.
5. The Kanbanzai 2.0 redesign proceeded from `main`, replacing all 97 1.0 tools with 20 consolidated 2.0 tools.
6. The 2.0 specification makes zero mention of hardening features — they were not in scope for preservation.
7. Track K (1.0 tool removal) deleted the old tool files, including any that had annotations.

---

## 3. Lost Features

### 3.1 MCP Safety Annotations

**Severity:** High — directly causes the babysitting/permission-friction problem

**Specified in:** Hardening spec §10 (AC-14, AC-15)

**What was built:** All 85 1.0 tools had explicit `readOnlyHint`, `destructiveHint`, and `idempotentHint` annotations, classified into three tiers:

| Tier | Annotations | Policy | Example tools |
|------|------------|--------|---------------|
| 1 — Read-only | `readOnly: true`, `destructive: false`, `idempotent: true` | Auto-approve freely | `get_entity`, `list_entities`, `health_check`, `doc_outline` |
| 2 — Auto-approvable writes | `readOnly: false`, `destructive: false` | Auto-approve (writes only to `.kbz/`) | `create_feature`, `update_status`, `doc_record_approve` |
| 3 — Requires confirmation | `destructive: true` or external systems | Must prompt user | `worktree_create`, `merge_execute`, `pr_create`, `cleanup_execute` |

**Current state on `main`:** Zero 2.0 tools call `mcp.WithToolAnnotation()`, `mcp.WithReadOnlyHintAnnotation()`, `mcp.WithDestructiveHintAnnotation()`, or `mcp.WithIdempotentHintAnnotation()`. All 20 tools are annotation-free.

**Impact:** MCP clients (Claude Code, Cursor, etc.) cannot distinguish read-only tools from destructive ones. The client must either auto-approve everything (unsafe) or prompt for everything (noisy). There is no granularity.

**2.0 complication:** The 2.0 tools are consolidated — a single `entity` tool handles both reads (`get`, `list`) and writes (`create`, `transition`). Since annotations are set at tool registration time (not per-invocation), each consolidated tool must take the most permissive classification across all its actions. This is still useful — the key distinction is Tier 2 (auto-approvable) vs. Tier 3 (requires confirmation). Most 2.0 tools would be Tier 2.

**Library support:** The mcp-go v0.45.0 library fully supports these annotations:
- `mcp.WithToolAnnotation(mcp.ToolAnnotation{...})`
- `mcp.WithReadOnlyHintAnnotation(true)`
- `mcp.WithDestructiveHintAnnotation(false)`
- `mcp.WithIdempotentHintAnnotation(true)`

### 3.2 Annotations Canary Test

**Severity:** High — the safety net for annotation coverage is gone

**What was built:** `annotations_test.go` (181 lines) — a test that instantiated every tool group and asserted all three annotations were non-nil on every registered tool. This would catch new tools being added without annotations.

**Current state on `main`:** File does not exist. No equivalent test exists for 2.0 tools.

### 3.3 `doc_record_refresh` Tool

**Severity:** Medium — creates real, recurring workflow friction

**Specified in:** Hardening spec §9 (AC-11, AC-12, AC-13)

**What was built:**
- `DocumentService.RefreshDocument()` method with `RefreshDocumentInput`/`RefreshDocumentResult` types
- MCP tool handler in `doc_record_tools.go`
- `doc_record_refresh_test.go` (384 lines) covering: hash updated, approved→draft transition, unchanged file returns `changed: false`, record-not-found, file-not-found

**Current state on `main`:** No `RefreshDocument` method on `DocumentService`. The 2.0 `doc` tool has 9 actions (`register`, `approve`, `get`, `content`, `list`, `gaps`, `validate`, `supersede`, `import`) — no `refresh` action.

**Impact:** When a user edits a registered document, the stored content hash becomes stale. The only way to fix this today is to delete the document record and re-register it, losing document intelligence data (classifications, entity references, graph relationships). The hardening branch solved this with a single `refresh` call that updated the hash and optionally transitioned `approved` → `draft`.

This friction was cited in the P7 retrospective as knowledge entry KE-01KMT5T79D9Q1 ("doc approve doesn't patch file Status header") and has been observed in multiple implementation cycles.

### 3.4 Improved Error Messages

**Severity:** Low-Medium

**Specified in:** Hardening spec §5 (AC-01)

**What was built:** Every service-layer error was audited to include a plain-language description plus a concrete action the user should take.

**Examples of what was lost:**

| Context | Current (main) | Hardening branch |
|---------|---------------|------------------|
| Document not found | `"document file not found: %s"` | `"document file not found at %q — ensure the file exists at that path before registering it"` |
| Already registered | `"document already registered: %s"` | `"document %q is already registered. Use doc_record_get to view the existing record"` |
| Wrong status | `"cannot approve document in status %s (must be draft)"` | `"cannot approve a document with status %q — only draft documents can be approved. If the document was previously approved, use doc_record_get to check its current status"` |
| Hash mismatch | `"content hash mismatch: ... (recorded=%s, current=%s)"` | `"document file has changed since it was registered. Run doc_record_refresh to update the stored hash, then retry doc_record_approve"` |

**Note:** The 2.0 tools have their own error formatting at the MCP layer (§7.4 defines structured `code`/`message`/`details`), and some 2.0 tool-level error messages are decent. But the underlying service-layer messages exposed through those tools still use the old, less helpful format.

### 3.5 `doc_supersession_chain` — Unreachable Service Code

**Severity:** Low-Medium

**What exists:** `DocumentService.SupersessionChain()` is a working method with tests — it follows the chain of document supersessions (A superseded by B superseded by C).

**Current state:** No 2.0 MCP tool exposes it. The 1.0 `doc_supersession_chain` tool was removed in Track K. The 2.0 spec §25.2 lists it as "removed" and §15.1 says it's "replaced by `doc`" — but the `doc` tool has no `chain` action.

**Impact:** Users cannot follow the version chain of a superseded document via MCP. The capability exists in the service layer but is dead code from the MCP perspective.

### 3.6 `suggest_links` — Specified as Automatic, Never Built

**Severity:** Low-Medium

**What was specified:** The 2.0 spec §25.3 relocated functionality table says `suggest_links` → "Inside `entity(action: create)` and `doc(action: register)` (automatic)".

**Current state:** The `entity` tool's create handler implements `entityDuplicateAdvisory` (the `check_duplicates` equivalent) but has no link suggestion code. The `doc` tool's register handler has no link suggestion code. `grep` for `suggest|link` in `entity_tool.go` returns zero matches.

**Impact:** There is no way for agents to extract entity references from free text via MCP. The spec explicitly said this should be automatic in entity/doc creation, but it was not implemented.

---

## 4. What Was Preserved

The core functionality of the 2.0 redesign is intact. These features all made it through the migration correctly:

| Feature | 2.0 Tool | Status |
|---------|----------|--------|
| Entity CRUD (create/get/list/update/transition) | `entity` | ✅ Working, with batch mode |
| Document operations (register/approve/get/list/etc.) | `doc` | ✅ 9 of 11 original actions (missing refresh, chain) |
| Dispatch and completion | `next`, `finish` | ✅ Including lenient lifecycle |
| Work queue with conflict check | `next` (queue mode) | ✅ |
| Context assembly | `next` (claim), `handoff` | ✅ Invisible inside both tools |
| Knowledge management (12 → 1 tool) | `knowledge` | ✅ All actions preserved |
| Document intelligence (10 → 1 tool) | `doc_intel` | ✅ Including `guide` action |
| Worktree/merge/PR/branch/cleanup | Individual tools | ✅ |
| Decompose/estimate/conflict | Individual tools | ✅ |
| Incident/checkpoint | Individual tools | ✅ |
| Duplicate detection | `entity(action: create)` | ✅ As `entityDuplicateAdvisory` |
| Side-effect reporting | `WithSideEffects` wrapper | ✅ New in 2.0 |
| Feature group framework | `groups.go` | ✅ New in 2.0 |
| Batch operations | `ExecuteBatch` | ✅ New in 2.0 |

---

## 5. The Portability Problem

The P7 retrospective's corrective actions (§6) recommend adding six new rules to `AGENTS.md`. This is not portable — `AGENTS.md` is project-specific. If other people install the `kanbanzai` MCP server, these rules would not travel with the tool.

### Instruction delivery hierarchy (most reliable → least)

| Tier | Mechanism | Reliability | Portability | Use for |
|------|-----------|-------------|-------------|---------|
| **1** | Built into tool responses | Highest — requires zero agent discipline | Ships with binary | Nudges at key moments (finish warns about missing retro signals, handoff reminds about state commits) |
| **2** | Skills (`.agents/skills/`) | High — auto-activates by context | Ships with `kanbanzai init` | Workflow rules, debugging discipline, sub-agent guidance |
| **3** | `AGENTS.md` | Low — agents may not read or may not follow | Project-specific, not portable | Project-specific overrides only |

### What belongs where

| Retro corrective action | Current target | Should be |
|------------------------|----------------|-----------|
| "Must use MCP tools for state queries" | AGENTS.md | Skill (`kanbanzai-agents`) + `next` claim response nudge |
| "Commit .kbz/state/ before spawning sub-agents" | AGENTS.md | Skill (`kanbanzai-agents` §Sub-Agent Spawning) + `handoff` output nudge |
| "Consult retro + knowledge before writing retrospectives" | AGENTS.md | New skill (`kanbanzai-retrospective`) or section in `kanbanzai-agents` |
| "Include retrospective signals in finish calls" | AGENTS.md | `finish` response nudge (on last task in feature with no signals) |
| "Investigate before explaining" | AGENTS.md | Skill (`kanbanzai-agents` §Debugging) |
| "1–2 tasks max per sub-agent" | AGENTS.md | Skill (`kanbanzai-agents` §Sub-Agent Spawning) |

---

## 6. The Permissions Problem

### Status of MCP auto-approval support

The hardening spec (§10) designed a comprehensive system for MCP client auto-approval. The implementation was completed on the hardening branch. It was lost in the 2.0 migration.

The design is still valid and the spec still applies — it just needs to be re-implemented against the 20 consolidated 2.0 tools rather than the 85 1.0 tools.

### Proposed 2.0 tool classification

Since 2.0 tools are consolidated (multiple actions per tool), each tool takes the most permissive tier across its actions:

**Tier 2 — Auto-approvable** (`readOnly: false`, `destructive: false`):

Most 2.0 tools write only to `.kbz/` and should be auto-approvable:

- `entity` (get/list are read-only, but create/update/transition write to `.kbz/`)
- `doc` (get/list/content are read-only, but register/approve write to `.kbz/`)
- `doc_intel` (classify writes to `.kbz/index/`, all others are read-only)
- `status` (read-only — could be Tier 1, but grouping as Tier 2 is conservative)
- `next` (queue mode is read-only, claim mode writes to `.kbz/`)
- `handoff` (read-only — could be Tier 1)
- `finish` (writes to `.kbz/`)
- `knowledge` (list/get are read-only, contribute/confirm write to `.kbz/`)
- `estimate` (query is read-only, set writes to `.kbz/`)
- `decompose` (writes to `.kbz/`)
- `conflict` (read-only — could be Tier 1)
- `profile` (read-only — could be Tier 1)
- `checkpoint` (writes to `.kbz/`)
- `incident` (writes to `.kbz/`)
- `health` (read-only — could be Tier 1)
- `retro` (reads knowledge, report action writes a file)

**Tier 3 — Requires confirmation** (`destructive: true` or external systems):

- `cleanup` (action: execute — deletes worktree directories and branches)
- `merge` (action: execute — merges branches on Git/GitHub)
- `pr` (action: create/update — interacts with GitHub API)
- `worktree` (action: create/remove — filesystem operations outside `.kbz/`)
- `branch` (action: status — reads Git state; could be Tier 2 if only `status` action)

---

## 7. Recommended Sprint

### Priority 1: Restore MCP safety annotations (High impact, small effort)

- Re-classify 20 tools against the three-tier model (see §6 above)
- Add `mcp.WithToolAnnotation()` to every tool registration in `internal/mcp/`
- Port the canary test: iterate all registered tools, assert all three annotations are non-nil
- The spec already exists (hardening spec §10), the library support exists (mcp-go v0.45.0), and the classification is straightforward

### Priority 2: Add `refresh` action to `doc` tool (Medium impact, small-medium effort)

- Reimplement `DocumentService.RefreshDocument()` (can reference hardening branch commit `8ae3b7e` for the design)
- Wire it as `doc(action: refresh)` alongside the existing 9 actions
- Port relevant tests from `doc_record_refresh_test.go` on the hardening branch

### Priority 3: Tool-embedded nudges (Medium impact, small effort)

- `finish`: soft warning when last task in feature completes without any retrospective signals
- `handoff`: include "commit .kbz/ state before spawning sub-agents" and "record retro signals on completion" in assembled context
- `next` (claim mode): include "use Kanbanzai MCP tools for state queries — do not read .kbz/state/ directly" in response

### Priority 4: Update skills with portable agent guidance (Medium impact, small effort)

- Update `kanbanzai-agents` SKILL: add MCP-tools-for-state-queries rule, investigation-before-explanation discipline, sub-agent scope guidance (1–2 tasks max), commit-state-before-spawning
- Consider new `kanbanzai-retrospective` SKILL with the procedure for writing retrospectives

### Priority 5: Wire remaining dead code (Low-medium impact, tiny effort)

- Add `chain` action to `doc` tool (service method `SupersessionChain()` already exists)
- Assess whether `suggest_links` automatic behaviour is still desired or can be deferred

### Stretch: Slim down AGENTS.md

- Move all Kanbanzai-generic rules out of `AGENTS.md` into skills
- Keep only project-specific content (repo structure, build commands, tracked `.kbz/state/` caveats)
- This makes `AGENTS.md` a template other projects can use as-is

---

## 8. Disposition of the Hardening Branch

The hardening branch (`feature/FEAT-01KMKRQWF0FCH-hardening`) is 506 files diverged from `main` and cannot be meaningfully merged. However, it contains valuable reference implementations:

| Content | Commit | Use |
|---------|--------|-----|
| MCP safety annotations on 1.0 tools | `8ae3b7e` | Reference for tier classification (not code-reusable — 1.0 tools are gone) |
| `annotations_test.go` | `8ae3b7e` | Template for 2.0 canary test structure |
| `RefreshDocument()` service method | `8ae3b7e` | Reference implementation for 2.0 `doc(action: refresh)` |
| `doc_record_refresh_test.go` | `8ae3b7e` | Test cases to port |
| Improved error messages | `8ae3b7e` | Reference for message quality standard |

**Recommendation:** Keep the branch as a reference archive. Do not attempt to merge it. Re-implement the lost features directly on `main` using the branch as a design reference.