# Conformance Review: FEAT-01KR3MEYRQ9RG — Runtime Discovery Surfaces

## Scope
- **Batch:** B62 — Discover Runtime Instruction Surfaces
- **Feature:** FEAT-01KR3MEYRQ9RG — Runtime discovery surfaces
- **Spec:** `work/B62-discover-runtime-instruction-surfaces/B62-F1-spec-runtime-discovery-surfaces.md` (approved)
- **Dev-Plan:** `work/B62-discover-runtime-instruction-surfaces/plan/B62-F1-dev-plan.md` (approved)
- **Tasks:** 6 total (6 done, 0 incomplete)
- **Review date:** 2026-05-08
- **Reviewer:** reviewer-conformance

## Feature Status

| Field | Value |
|-------|-------|
| Lifecycle | developing (all tasks done — ready to advance) |
| Spec | approved |
| Dev-Plan | approved |
| Design | approved (P59 plan-level design) |
| Tasks | 6/6 done |
| Verification | passed |

## Acceptance Criteria Traceability

### AC-001 (REQ-001): Seven required wrappers exist
**Verdict:** ✅ PASS

Seven wrapper directories confirmed under `.claude/skills/`:
`orchestrate-development`, `implement-task`, `kanbanzai-getting-started`,
`kanbanzai-workflow`, `write-spec`, `write-design`, `review-code`.

Each contains a `SKILL.md` file. Matches the required set from REQ-001.

### AC-002 (REQ-002, REQ-003): Frontmatter + canonical path per wrapper
**Verdict:** ✅ PASS

Every wrapper has:
- YAML frontmatter with `name` and `description` fields
- A single-line description suitable for Claude skill discovery
- `<!-- canonical: <path> -->` HTML comment pointing to the canonical source
- A title heading, trigger text, and redirect to the canonical SKILL.md

### AC-003 (REQ-004, REQ-NF-005): Regular files, not symlinks
**Verdict:** ✅ PASS

All seven wrappers are regular files (verified via `test ! -L`). No symlinks present.
Satisfies REQ-NF-005 (works on filesystems that don't preserve symlinks).

### AC-004 (REQ-005, REQ-012): Drift check detects stale wrappers
**Verdict:** ✅ PASS

Implementation at `internal/claudeskills/check.go`:
- `CheckAll()` compares on-disk wrapper content against `ExpectedWrappers` spec
- Returns paths of stale/missing wrappers
- Three tests in `internal/claudeskills/check_test.go` all pass:
  - `TestClaudeSkillsWrappers` — validates all on-disk wrappers match expected content
  - `TestClaudeSkillsDriftDetected` — fixture-based: stale wrapper → detected with path
  - `TestClaudeSkillsDriftClean` — correctly-generated wrappers pass check
- Makefile target `claude-skills-check` runs the check via `go test`
- CI-able: `make claude-skills-check` exits non-zero on drift

### AC-005 (REQ-006): OPENAI.md redirect
**Verdict:** ✅ PASS

- File exists at repository root
- 13 lines (≤ 20 limit per REQ-NF-002)
- Contains reference to `AGENTS.md`
- Does not duplicate the full AGENTS instruction corpus
- Includes one-sentence explanation of why it exists (GPT-class host convention)

### AC-006 (REQ-007, REQ-008): Tool descriptions with P59 rule text
**Verdict:** ✅ PASS

Five Kanbanzai MCP tools updated with concise rule text:

| Tool | Rule text | Invariant | Lines added |
|------|-----------|-----------|-------------|
| `next` | "Claim before work begins — do not start implementing without a claimed task. Orchestrators must dispatch via handoff(task_id), not direct spawn_agent" | INV-001 | 2 |
| `handoff` | "Use INSTEAD OF calling spawn_agent directly — this is the only safe dispatch path; direct spawn_agent bypasses context assembly and stage gate enforcement" | INV-001 | 2 |
| `entity` | "Verify entity existence before working on it — do not create worktrees, write files, or open PRs against unregistered entity names" | INV-002 | 2 |
| `worktree` | "One worktree per entity. Do not commit directly to main — all implementation work must be isolated in an entity worktree." | TODO(P59-B2) | 2 |
| `pr` | "pr enforces the branch/title conventions required by stage gate merge checks" | INV-005 | 1 |

Notes:
- `spawn_agent` and `dispatch_task` are not live Kanbanzai MCP tools (confirmed in T4 task spec) — they are excluded by design
- `worktree` tool uses `TODO(P59-B2)` comment for invariant code alignment per the spec's allowance when B59 codes are not yet defined
- `go build ./internal/mcp/...` and `go test ./internal/mcp/...` both pass

### AC-007 (REQ-009): DeepSeek host documentation
**Verdict:** ✅ PASS

Section "### DeepSeek" in `refs/sub-agents.md` covers all four C4 schema fields:

| Field | Content |
|-------|---------|
| Host/platform | DeepSeek API (`api.deepseek.com`), OpenAI/Anthropic-compatible inference |
| AGENTS.md injection | No — stateless inference API with no filesystem awareness |
| Tool-description injection | No (raw API); Unknown for third-party MCP clients (see REQ-010) |
| Manual configuration | 3 items: system prompt, MCP-to-OpenAI bridge, client config (URL/key/model) |

Includes an "Important distinction" note clarifying that injection questions are client-specific, not DeepSeek API properties.

### AC-008 (REQ-010): Unknown behaviour explicitly marked
**Verdict:** ✅ PASS

DeepSeek documentation uses the literal string `Unknown — see REQ-010` for client-specific tool-description injection behaviour. A dedicated "REQ-010 note" section reiterates that third-party client behaviour is not verifiable from DeepSeek API documentation. The feature correctly treats the unverifiable portion as incomplete while documenting what is known.

### AC-009 (REQ-011): Cursor rules shim
**Verdict:** ✅ PASS

- File exists at `.cursor/rules/kanbanzai.mdc`
- Contains redirect-style guidance pointing to `AGENTS.md`
- No duplicated rule prose — all workflow guidance deferred to the canonical corpus
- 9 lines (compact and well within reason)

### AC-010 (REQ-NF-001): 80-line wrapper maximum
**Verdict:** ✅ PASS

| Wrapper | Lines |
|---------|-------|
| orchestrate-development | 14 |
| implement-task | 14 |
| kanbanzai-getting-started | 14 |
| kanbanzai-workflow | 14 |
| write-spec | 14 |
| write-design | 14 |
| review-code | 14 |

All wrappers are 14 lines — well under the 80-line maximum.

### AC-011 (REQ-NF-003): 3-line rule-text maximum per tool
**Verdict:** ✅ PASS

Maximum addition is 2 lines (entity, handoff, next, worktree). `pr` adds 1 line. All within the 3-line budget. `worktree` has an additional `TODO(P59-B2)` comment line, which is a spec-allowed annotation, not rule text.

## Interface Contract Verification

### C1: Wrapper file format
**Verdict:** ✅ PASS

All wrappers conform to the contract defined in the dev-plan:
- YAML frontmatter delimited by `---`
- `name` and `description` fields
- `kanbanzai-generated: true` marker comment
- `canonical: <path>` comment
- Title heading, trigger line, and canonical redirect

### C2: Drift-check input/output contract
**Verdict:** ✅ PASS

`CheckAll(skillsDir string) []string` returns paths of stale/missing wrappers. Empty slice when all current. Non-empty when drift detected. Tests confirm both paths.

### C3: Tool description text budget
**Verdict:** ✅ PASS

All modifications are additive (no content removed). Each tool description grows by at most 2 lines of rule text.

### C4: DeepSeek documentation schema
**Verdict:** ✅ PASS

All four required fields present and populated. Unknown values use the literal per REQ-010.

## Cross-Cutting Checks

### Health check
- No errors or warnings specific to FEAT-01KR3MEYRQ9RG
- Expected warning: "FEAT-01KR3MEYRQ9RG has all 6 child task(s) in terminal state but feature is developing" — this is the pre-review state; advancing to reviewing resolves this
- Expected warning: "branch feature/FEAT-01KR3MEYRQ9RG-runtime-discovery-surfaces is drifting: 87 commits behind main" — expected for an in-development feature; resolved on merge

### Worktree
- Active worktree at `.worktrees/FEAT-01KR3MEYRQ9RG-runtime-discovery-surfaces`
- Branch `feature/FEAT-01KR3MEYRQ9RG-runtime-discovery-surfaces`
- Expected to be merged and cleaned up after review approval

### Generator script
- `scripts/gen-claude-skills.sh` is executable (mode 100755)
- Uses `set -euo pipefail` for safety
- Regenerates all 7 wrappers from canonical metadata in a single run
- Satisfies REQ-NF-004 (canonical-source ownership preserved through regeneration)

## Conformance Gaps

None identified. All 11 acceptance criteria pass with evidence.

## Spec-Gap Observations

None. The specification is complete and all requirements are traceable to implementation.

## Batch Verdict

**PASS** — All 11 acceptance criteria verified with specific evidence. The implementation conforms to the approved specification. All 6 tasks are done, verification passed, and the drift check CI target is operational.

## Evidence

- Feature: `entity(action: "get", id: "FEAT-01KR3MEYRQ9RG")` — developing, 6/6 tasks done
- Tasks: `entity(action: "list", type: "task", parent_feature: "FEAT-01KR3MEYRQ9RG")` — 6 done
- Spec: `doc(id: "FEAT-01KR3MEYRQ9RG/spec-b62-f1-spec-runtime-discovery-surfaces")` — approved
- Dev-plan: `doc(id: "FEAT-01KR3MEYRQ9RG/dev-plan-b62-f1-dev-plan")` — approved
- Wrappers: `.claude/skills/*/SKILL.md` — 7 wrappers, 14 lines each, regular files
- OPENAI.md: 13 lines, redirect to AGENTS.md
- Tool descriptions: 5 tools updated in `internal/mcp/` (diff verified via `git diff 88ea71cb^..88ea71cb`)
- DeepSeek docs: `refs/sub-agents.md` lines 47-87 (worktree version)
- Cursor shim: `.cursor/rules/kanbanzai.mdc` — 9 lines, redirect to AGENTS.md
- Drift check: `internal/claudeskills/check.go` + `check_test.go` — 3 tests PASS
- CI target: `make claude-skills-check` — exits 0
- Health: `health()` — no FEAT-01KR3MEYRQ9RG-specific errors
