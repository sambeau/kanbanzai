| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06T16:19:45Z           |
| Status | approved |
| Author | spec-author                     |

# Specification: Context Budget Recalibration

**Feature:** FEAT-01KQYZZFGBGQK (Context Budget Recalibration)
**Parent Batch:** B1-p51-exec
**Design:** `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md`

## Overview

This specification implements the context budget recalibration described in `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md` (design document for P51). It updates two stale budget constants, makes the context window token limit configurable via `.kbz/local.yaml`, and adds topic-level detail to trimmed metadata so orchestrators can assess what knowledge was dropped during assembly.

## Scope

**In scope:**
- Update `DefaultContextWindowTokens` from 200,000 to 1,000,000
- Make `DefaultContextWindowTokens` configurable via `.kbz/local.yaml`
- Raise `assemblyDefaultBudget` from 30,720 to 65,536 (64KB)
- Add topic-level detail to `trimmed` metadata in `next`/`handoff` responses

**Out of scope:**
- Changing the pipeline's budget warning/refuse ratio logic
- Replacing the byte-budget system with percentage-based allocation (deferred per design Open Question 9)
- P44's `dispatch_task` internal pipeline path (that feature bypasses MCP response caps entirely)

## Related Work

Concepts searched: `DefaultContextWindowTokens`, `assemblyDefaultBudget`, `context budget`, `token budget`, `trimmed metadata`, `MCP response cap`.

Entity IDs searched: P50, P51, FEAT-01KQYZZFGBGQK.

Prior specifications searched: none found in `doc(action: "list", type: "specification", status: "approved")` — this project has no approved specifications yet.

**Attestation:** No directly related prior work was found in the corpus. The design document (P51-design-handoff-pipeline-unification, §1.6.1 "Trimming visibility gap" and Open Questions 8–9) is the sole decision-making artifact for these budget values. The P50 fast-track implementation provides empirical evidence (30,301 bytes used of 30,720 cap, causing invisible knowledge trimming) documented in the design.

## Problem Statement

The context assembly pipeline has two budget constraints that are stale:

1. **`DefaultContextWindowTokens = 200_000`** — this was calibrated against older model context windows. Current models (e.g., Claude Opus 4, GPT-4.1) support 1,000,000+ tokens. The pipeline uses this value for the token budget warning/refuse thresholds (`BudgetWarnRatio = 0.40`, `BudgetRefuseRatio = 0.60`), so a 200K window warns at 80K tokens and refuses at 120K tokens — far more aggressively than needed.

2. **`assemblyDefaultBudget = 30_720`** (30KB) — this is an MCP response size guard, not a context window limit. During P50 fast-track, the orchestrator received `next` responses at 98.6% capacity with `trimmed` metadata listing entry counts but not **topics**. Knowledge entries relevant to the work were silently dropped without the orchestrator knowing which ones.

Additionally, `DefaultContextWindowTokens` is a hardcoded constant with no per-environment override, so operators using models with different context windows must either recompile or accept incorrect budget behavior.

## Functional Requirements

- **FR-001:** The `DefaultContextWindowTokens` constant MUST be 1,000,000. Pipeline token budget warnings MUST fire at 400,000 tokens (40% of 1,000,000) and budget refusal MUST fire at 600,000 tokens (60% of 1,000,000).
- **FR-002:** When `.kbz/local.yaml` contains `context_window_tokens: <N>`, the pipeline MUST use `<N>` as the context window size instead of `DefaultContextWindowTokens`. When the key is absent, the pipeline MUST use `DefaultContextWindowTokens`.
- **FR-003:** The `assemblyDefaultBudget` constant MUST be 65,536 (64KB). All `next` and `handoff` responses MUST use this value as the byte budget cap.
- **FR-004:** When knowledge entries are trimmed from a `next` or `handoff` response, the `trimmed` metadata array MUST include a `topic` field containing the knowledge entry's topic string for each trimmed entry. The existing `scope`, `tier`, and `token_estimate` fields MUST continue to be present.
- **FR-005:** The `Pipeline` struct MUST expose the active context window size (whether from config or default) so that callers and tests can inspect which value is in effect.

## Non-Functional Requirements

- **FR-NF-001:** The configurable `context_window_tokens` value MUST be validated at load time — values below 100,000 MUST be rejected with an error message recommending at least 1,000,000.
- **FR-NF-002:** Changing the `assemblyDefaultBudget` MUST NOT change the semantic content of `next`/`handoff` responses — responses that previously fit within 30,720 bytes MUST produce identical output under the new 65,536-byte cap.
- **FR-NF-003:** The `topic` field on trimmed entries MUST NOT increase response size beyond the existing budget cap — if adding topic strings causes budget overflow, topics MUST be truncated with `…` (U+2026) rather than omitted.

## Constraints

- The `Pipeline` struct's `WindowSize` field already supports override (zero means "use default") — this mechanism MUST be preserved and extended to read from config.
- The budget warning/refuse ratios (`BudgetWarnRatio = 0.40`, `BudgetRefuseRatio = 0.60`) MUST NOT change.
- This specification does NOT cover how P44's `dispatch_task` tool bypasses the MCP response cap — that is specified separately.
- The `assemblyDefaultBudget` change applies to both `next` and `handoff` response paths.

## Acceptance Criteria

- **AC-001 (FR-001):** Given a pipeline with no custom config, when `windowTokens()` is called, then it returns 1,000,000.
- **AC-002 (FR-001):** Given a pipeline with a 1,000,000-token window, when token usage reaches 400,001 tokens, then the pipeline emits a budget warning. When token usage reaches 600,001 tokens, then the pipeline refuses further assembly.
- **AC-003 (FR-002):** Given `.kbz/local.yaml` contains `context_window_tokens: 500000`, when a pipeline is initialized from this config, then `windowTokens()` returns 500,000.
- **AC-004 (FR-002):** Given `.kbz/local.yaml` does NOT contain `context_window_tokens`, when a pipeline is initialized, then `windowTokens()` returns `DefaultContextWindowTokens` (1,000,000).
- **AC-005 (FR-002):** Given `.kbz/local.yaml` contains `context_window_tokens: 50000`, when the pipeline is initialized, then initialization fails with an error message containing "100,000" and "1,000,000".
- **AC-006 (FR-003):** Given the `assemblyDefaultBudget` constant is changed to 65,536, when a `next` or `handoff` response is assembled, then `byte_budget` in the response metadata equals 65536.
- **AC-007 (FR-004):** Given knowledge entries are trimmed during assembly, when the response metadata is inspected, then each entry in the `trimmed` array contains a `topic` field with the knowledge entry's topic string.
- **AC-008 (FR-004):** Given a trimmed knowledge entry with topic "handoff-to-spawn_agent-pattern", when the trimmed metadata is inspected, then `topic` equals "handoff-to-spawn_agent-pattern".
- **AC-009 (FR-NF-001):** Given `.kbz/local.yaml` contains `context_window_tokens: 50000`, when the config is loaded, then an error is returned.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: create Pipeline with no config, assert `windowTokens() == 1_000_000` |
| AC-002 | Test | Unit test: create Pipeline with 1M window, feed 400K+ token usage, assert warning emitted |
| AC-003 | Test | Unit test: load config with `context_window_tokens: 500000`, assert window is 500,000 |
| AC-004 | Test | Unit test: load config without the key, assert window is `DefaultContextWindowTokens` |
| AC-005 | Test | Unit test: load config with 50,000, assert error returned |
| AC-006 | Test | Unit test: assemble response, assert `byte_budget == 65536` in metadata |
| AC-007 | Test | Integration test: force trimming, inspect `trimmed` entries for `topic` field |
| AC-008 | Test | Unit test: verify specific topic value in trimmed metadata |
| AC-009 | Test | Unit test: validate config loading rejects values below 100,000 |
