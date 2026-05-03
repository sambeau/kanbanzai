# Design: Hash-Anchored Edit Tool

**Plan ID:** P42-hash-anchored-edit-tool  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §6.1, §7.3

## Overview

Implement a hash-anchored edit mechanism for Kanbanzai's MCP tool surface. Every line read by an agent is tagged with a short content hash. When the agent submits an edit, it references those hash tags. If the file changed since the last read, the hash won't match and the edit is rejected before corruption.

This is the single highest-leverage feature from the OpenCode ecosystem — both independent evaluations converge on it as the top priority. The claimed improvement from oh-my-openagent is 6.7% → 68.3% on the Grok Code Fast 1 benchmark, just from changing the edit tool. The claim is unverified but the mechanism is sound, and the harness problem (Can Bölük, 2025) is well-documented: most agent failures are edit-tool failures, not reasoning failures.

## Goals and Non-Goals

**Goals:**
- Prevent stale-line edit corruption: if a file changed between read and edit, reject the edit
- Give agents stable, verifiable line identifiers that don't depend on reproducing text
- Work with existing `edit_file` tool or as a new `hash_edit` MCP tool
- Compatible with `codebase-memory-mcp` knowledge graph (different layers, no conflict)
- Zero architectural dependencies — standalone tool enhancement

**Non-Goals:**
- Not a replacement for `edit_file` — can coexist as an alternative tool
- Not a version control mechanism — hashes are ephemeral, per-session
- Not a conflict resolution tool — rejection only, no merge logic
- Not changing how agents read files in general — only when they intend to edit

## Design

### Hashline Format

Inspired by oh-my-openagent's format. Each line returned to the agent is tagged:

```
11#VK| function hello() {
22#XJ|   return "world";
33#MB| }
```

- `11`, `22`, `33` — line numbers
- `#VK`, `#XJ`, `#MB` — 2-char hex content hashes (truncated from a full hash of the line content)
- `|` — separator between hash metadata and content

### Edit Submission

The agent submits edits referencing hash tags instead of reproducing text:

```json
{
  "path": "src/main.go",
  "edits": [
    {
      "hash_ref": "22#XJ",
      "new_text": "  return \"hello, world\";"
    }
  ]
}
```

The tool:
1. Reads the current file at `path`
2. Computes the hash of line 22's current content
3. If it matches `#XJ` → apply the edit
4. If it doesn't match → reject with error: "Line 22 hash mismatch: expected #XJ, got #AB. File may have changed since last read."

### Hash Computation

- Hash function: SHA-256, truncated to 2 hex characters (256 possible values per line)
- Collision probability: ~1/256 chance of false match per line. Acceptable for an edit-time guard (not a security mechanism). False matches mean a stale edit could slip through, but the probability is low and the consequence is a failed fuzzy match fallback, not data corruption.
- Alternative: use a longer hash (4 chars = 65,536 values) if collision rate proves problematic in practice. Start with 2 chars for readability; monitor.

### Read Tool Integration

Two approaches, not mutually exclusive:

**Option A: New `hash_read` tool.** Returns file content with hash-tagged lines. Agent uses this when it intends to edit. Clean separation — `read_file` stays as-is, `hash_read` is explicitly for edit preparation.

**Option B: Enhanced `read_file` with optional hash tagging.** `read_file` gains a `hash_tag: true` parameter. When set, output includes hash-tagged lines. Simpler for agents — one read tool, one parameter.

Recommendation: Start with Option B (enhanced `read_file`). It's simpler to adopt — agents already use `read_file`. If the hash metadata proves distracting for non-edit reads, extract to Option A later.

### Edit Tool Integration

**Option A: New `hash_edit` tool.** Accepts hash-anchored edits only. Clean, unambiguous — the tool name signals the contract.

**Option B: Enhanced `edit_file`.** `edit_file` gains a `hash_validate: true` parameter. When set, edits must reference hash tags. When absent, current fuzzy-match behavior applies.

Recommendation: Start with Option B (enhanced `edit_file`). Single edit tool, parameter-controlled behavior. See Alternatives Considered for discussion.

### Knowledge Graph Compatibility

Hash-anchored edits and `codebase-memory-mcp` operate at different layers — no conflict:

- The graph indexes filesystem state. It doesn't care *how* a file was edited, only *that* it changed.
- Hash-anchored validation ensures edits are correct before they land on disk. The graph indexes clean state, not corrupted edits.
- `detect_changes` becomes more reliable as a result (no garbage-in from corrupted edits).

**Potential synergy:** `get_code_snippet` (the graph's code-reading tool) could return hash-tagged lines, giving agents stable structural understanding AND stable line identifiers in a single call. This is an optimization to explore after the basic tool is working — not a prerequisite.

### Error Modes

| Scenario | Behavior |
|----------|----------|
| Hash matches, edit valid | Apply edit, return success |
| Hash mismatch (file changed) | Reject with error: which line, expected hash, actual hash |
| Line number out of range (file shortened) | Reject with error: line no longer exists |
| File not found | Standard file-not-found error |
| Hash collision (different content, same 2-char hash) | Edit applied to wrong line — acceptable risk at 1/256 per line |
| Agent doesn't use hash refs | Falls back to current fuzzy-match behavior (backward compatible) |

## Alternatives Considered

### New `hash_edit` tool vs. enhanced `edit_file`

**New tool:** Clean separation, unambiguous contract, no backward-compatibility concerns. But: agents need to learn a new tool, two edit paths to maintain.

**Enhanced `edit_file`:** Single tool, parameter-controlled, backward compatible. But: the tool schema grows, and the hash-validate path is implicitly coupled to the fuzzy-match path.

**Decision:** Start with enhanced `edit_file` for adoption simplicity. If the combined tool proves confusing or error-prone, extract `hash_edit` as a separate tool later. The internal implementation should keep the hash-validation logic in a separate package so extraction is cheap.

### 2-char vs. 4-char hashes

2-char: 256 values, 1/256 collision chance. More readable in prompts.
4-char: 65,536 values, 1/65K collision chance. Less readable.

**Decision:** Start with 2-char. The consequence of a collision is a fuzzy-match fallback (which is what we have today), not data corruption. Monitor collision rate in practice; upgrade to 4-char if it causes visible edit failures.

## Dependencies

- None. This is a tool-level enhancement to `read_file` and `edit_file` MCP tools.
- No entity model changes, no stage binding changes, no new roles or skills.
- The only integration point is `codebase-memory-mcp` for the optional `get_code_snippet` hash-tagging synergy — that's deferred until after the core tool works.

## Open Questions

1. Should hash-tagged output be the default for `read_file`, or opt-in via parameter? Default would mean every read is edit-ready but adds visual noise. Opt-in is cleaner but requires agents to know when they intend to edit.
2. Should we benchmark the claim? Set up a SWE-bench-style test comparing edit success rates with and without hash validation on Kanbanzai's actual tool surface before committing to full implementation.
3. How does this interact with `write_file` (full-file overwrite)? `write_file` doesn't need hash validation — it replaces the entire file. But if an agent reads with hashes then uses `write_file`, the hashes are irrelevant. Should we warn?
