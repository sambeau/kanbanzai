# Phase 3 Scope and Planning

| Document | Phase 3 Scope and Planning |
|----------|----------------------------|
| Status | Draft |
| Created | 2025-01-27 |
| Updated | 2025-01-27 |
| Related | `work/design/workflow-design-basis.md` §16, §22, §24 |
|         | `work/design/machine-context-design.md` §9.5, §9.6, §15.3 |
|         | `work/spec/phase-2b-specification.md` §3.2 |
|         | `work/plan/phase-2-decision-log.md` P2-DEC-001 |

---

## 1. Purpose

This document defines the scope, approved design decisions, and planning structure for Phase 3: Git Integration.

Phase 3 builds on the workflow kernel (Phase 1), document intelligence (Phase 2a), and context management (Phase 2b) to add Git-native collaboration features: worktree isolation for parallel work, branch tracking, merge readiness gates, and the deferred knowledge lifecycle automation features.

---

## 2. Phase 3 Goals

1. **Enable parallel agent work** — Multiple agents can work on different features/bugs simultaneously without interfering with each other's working state.

2. **Track branch health** — The system understands branch age, drift from main, activity, and merge readiness.

3. **Enforce quality gates before merge** — Merge requires verification: tasks complete, specs current, tests exist, branch not stale.

4. **Complete the knowledge feedback loop** — Knowledge entries are tied to code, automatically flagged when stale, pruned when unused, promoted when valuable.

5. **Integrate with GitHub** — Create and update PRs to provide visibility into workflow state.

6. **Self-clean** — Post-merge cleanup of worktrees, branches, and duplicate knowledge.

---

## 3. Scope

### 3.1 In scope for Phase 3

**Git integration (core):**

- Worktree management (create, list, remove, track entity relationships)
- Branch tracking (age, drift, activity, merge readiness)
- Merge readiness checks (gates that verify pre-merge conditions)
- PR integration with GitHub (create, update descriptions/labels)
- Post-merge cleanup (worktrees, branches)

**Knowledge lifecycle automation (deferred from Phase 2b per P2-DEC-001):**

- Git anchoring and automatic staleness detection
- TTL-based automatic pruning
- Automatic promotion triggers (Tier 3 → Tier 2)
- Post-merge knowledge compaction

### 3.2 Deferred beyond Phase 3

- Orchestration or agent delegation (Phase 4)
- Cross-project knowledge sharing
- Embedding-based semantic similarity for deduplication
- Automatic context assembly optimisation
- GitLab, Bitbucket, or other platform support
- Conflict domain analysis tooling
- Vertical slice decomposition tooling

### 3.3 Explicitly excluded

- Replacing GitHub as the canonical workflow store (workflow kernel remains source of truth)
- GitHub Issues sync (workflow entities are canonical; GitHub is a view)
- Automated PR merging without human approval
- Webhook-based real-time sync

---

## 4. Approved Design Decisions

The following Human UX decisions were made during scope planning and are binding for Phase 3 specification and implementation.

### P3-DES-001: Branch Naming Convention

**Decision:** Suggested convention with explicit tracking.

The system tracks branch-to-entity relationships in its own state, not by parsing branch names. Branch names are for human readability.

- **Suggested pattern:** `feature/FEAT-01JX-short-slug` or `bug/BUG-01JX-short-slug`
- **Enforcement:** None — any branch name is accepted
- **Tracking:** Explicit relationship stored in workflow state

### P3-DES-002: Worktree Lifecycle Ownership

**Decision:** System-automatic on task activation.

The system creates a worktree when the first task within a feature/bug transitions to `in_progress`. This is the moment isolation becomes valuable.

- **Trigger:** First task in feature/bug starts
- **Creation:** Automatic by system
- **Visibility:** `kbz worktree list` shows all active worktrees and their entity relationships
- **Destruction:** Automatic cleanup post-merge (see P3-DES-008)

### P3-DES-003: Multiple Worktrees Per Feature

**Decision:** One worktree per feature/bug.

All tasks within a feature share one worktree. Tasks within a feature are typically interdependent; if true parallelism is needed, decompose into separate features.

- **Model:** One-to-one relationship between feature/bug and worktree
- **Escape hatch:** Humans can manually create additional worktrees if truly needed (not a system-supported workflow)

### P3-DES-004: Platform Scope

**Decision:** GitHub-only for Phase 3.

Build for GitHub. Don't pre-design abstractions for platforms not yet implemented. Core Git features (worktrees, branches) are platform-agnostic; only PR integration is GitHub-specific.

- **Phase 3:** GitHub only
- **Future:** Extract abstraction when a second platform becomes a priority

### P3-DES-005: PR Operation Scope

**Decision:** Create and Update.

The system can create PRs for completed features/bugs and keep descriptions/labels in sync with workflow state.

- **Create:** Generate PR with description from feature/bug state
- **Update:** Sync description to reflect task completion, verification status
- **Not included:** Commenting, requesting reviewers (remain human-initiated)

### P3-DES-006: Merge Gate Workflow Model

**Decision:** Enforced by default, human override available.

The system blocks merge if gates fail, but humans can force-merge with explicit override.

- **Default:** Gates are enforced
- **Override:** Explicit flag required (e.g., `--override-gates`)
- **Accountability:** Override requires reason; logged for audit

### P3-DES-007: Merge Gate Failure Behavior

**Decision:** Confirmation required.

When gates fail, merge requires explicit acknowledgment of what failed and why the human is proceeding anyway.

- **Display:** System shows what failed and why
- **Confirm:** Human must acknowledge failures
- **Reason:** Human provides brief reason (logged)

### P3-DES-008: Cleanup Scope and Triggering

**Decision:** Automatic with 7-day grace period.

- **Worktrees:** Auto-cleanup 7 days after merge
- **Branches:** Auto-cleanup 7 days after merge
- **Knowledge entries:** Not auto-deleted; TTL handles Tier 3 naturally; high-use entries surfaced for potential promotion
- **Configuration:** Grace period configurable per-project

---

## 5. Decisions Deferred to Development

### 5.1 Implementation details

These decisions can be made by the developer (AI agents) during specification, planning, or implementation at the appropriate stage.

| Decision | Notes |
|----------|-------|
| Storage model for worktree tracking | New entity type vs. metadata on existing entities vs. separate tracking file |
| Staleness detection timing | On-access vs. background vs. health-check trigger |
| TTL pruning execution model | Background job vs. lazy evaluation on access |
| GitHub authentication mechanism | Token storage and configuration approach |
| Clock handling for TTL | Timezone and drift considerations |
| Compaction triggering mechanism | Post-merge hook vs. explicit invocation vs. health check |

### 5.2 AI Agent UX decisions

These decisions are best made by AI agents during specification or design, as they understand their own capabilities and workflows.

| Decision | Notes |
|----------|-------|
| Git anchor path format | Exact paths vs. globs vs. line ranges — what granularity is useful for retrieval and staleness? |
| Change threshold for staleness | Any modification vs. structural changes only — what sensitivity produces actionable signals? |
| MCP tool design for merge readiness | Request/response shapes that support agent decision-making |
| MCP tool design for worktree operations | Parameters and responses that support agent workflows |
| Compaction conflict resolution UX | How to present conflicting knowledge entries to agents for resolution |

---

## 6. Phase 3 Split Consideration

### Option A: Single Phase 3

Ship all features together. Advantages: simpler planning, one specification document, no artificial boundaries.

### Option B: Phase 3a / 3b Split

**Phase 3a: Knowledge Lifecycle Automation**
- Git anchoring and staleness detection
- TTL-based pruning
- Automatic promotion triggers

**Phase 3b: Git Integration**
- Worktree management
- Branch tracking
- Merge gates
- PR integration
- Post-merge cleanup and compaction

**Advantages:** Knowledge lifecycle features have complete designs from Phase 2b planning; they can proceed to specification immediately. Git integration features are larger and benefit from the Human UX decisions just made.

### Recommendation

**Proceed as single Phase 3** unless scope proves too large during planning.

With the Human UX decisions now made, Git integration has enough design direction to proceed. The knowledge lifecycle features are smaller and can be implemented alongside or interleaved with Git integration features. A split adds coordination overhead without clear benefit.

If implementation reveals that scope is too large for a single phase, split at that point with knowledge lifecycle as the natural first tranche.

---

## 7. Feature Breakdown

### 7.1 Worktree Management

| Capability | Description |
|------------|-------------|
| Create worktree | Create isolated worktree for a feature/bug, linked to entity |
| List worktrees | Show all active worktrees with entity relationships |
| Remove worktree | Clean up worktree (manual or post-merge automatic) |
| Track relationship | Store and query worktree-to-entity mapping |

**MCP tools needed:**
- `worktree_create` — create worktree for entity
- `worktree_list` — list all worktrees with status
- `worktree_remove` — remove worktree
- `worktree_get` — get worktree for entity

**CLI commands needed:**
- `kbz worktree list`
- `kbz worktree create <entity-id>`
- `kbz worktree remove <entity-id>`

### 7.2 Branch Tracking

| Metric | Description |
|--------|-------------|
| Branch age | Time since branch creation |
| Drift from main | Commits behind main; merge conflict potential |
| Recent activity | Time since last commit |
| Merge readiness | Composite status from merge gates |

**MCP tools needed:**
- `branch_status` — get tracking metrics for a branch/entity

**CLI commands needed:**
- `kbz branch status <entity-id>`
- `kbz branch list` — list branches with health indicators

### 7.3 Merge Gates

| Gate | Checks |
|------|--------|
| Tasks complete | All tasks in feature/bug are `done` or `wont_do` |
| Verification exists | Feature/bug has verification criteria and they're met |
| Specs current | Linked specifications are not stale |
| Branch not stale | Recent activity; not excessively drifted |
| Health check clean | No blocking health-check errors |

**MCP tools needed:**
- `merge_readiness_check` — run all gates, return pass/fail with details
- `merge_execute` — perform merge (with override option)

**CLI commands needed:**
- `kbz merge check <entity-id>`
- `kbz merge <entity-id> [--override-gates --reason "..."]`

### 7.4 GitHub PR Integration

| Operation | Description |
|-----------|-------------|
| Create PR | Create PR from feature/bug branch to main |
| Update PR | Sync PR description with current workflow state |
| Check status | Read PR status (CI, reviews, merge conflicts) |

**MCP tools needed:**
- `pr_create` — create PR for entity
- `pr_update` — update PR description/labels from workflow state
- `pr_status` — get PR status (CI, reviews, conflicts)

**CLI commands needed:**
- `kbz pr create <entity-id>`
- `kbz pr update <entity-id>`
- `kbz pr status <entity-id>`

### 7.5 Post-Merge Cleanup

| Target | Timing | Action |
|--------|--------|--------|
| Worktree | 7 days post-merge | Remove |
| Branch | 7 days post-merge | Delete |
| Knowledge | On merge | Surface high-use Tier 3 for promotion review |

**MCP tools needed:**
- `cleanup_check` — identify items pending cleanup
- `cleanup_execute` — perform cleanup (or let automatic process handle)

**CLI commands needed:**
- `kbz cleanup list` — show pending cleanup items
- `kbz cleanup run` — run cleanup now (don't wait for grace period)

### 7.6 Git Anchoring

| Capability | Description |
|------------|-------------|
| Anchor knowledge | Associate knowledge entry with file paths |
| Detect changes | Flag anchored entries when files change |
| Surface staleness | Include staleness status in knowledge retrieval |

**MCP tools needed:**
- Extend `knowledge_contribute` to accept `git_anchors`
- Extend `knowledge_get` / `context_get` to include staleness indicators
- `knowledge_check_staleness` — explicit staleness check

**CLI commands needed:**
- `kbz knowledge check` — check for stale knowledge entries

### 7.7 TTL-Based Pruning

| Tier | Default TTL | Reset on use | Prune condition |
|------|-------------|--------------|-----------------|
| Tier 3 | 30 days | Yes, to 30 days | TTL expires AND use_count < 3 |
| Tier 2 | 90 days | Yes, to 90 days | TTL expires AND confidence < 0.5 |
| Tier 1 | No expiry | N/A | Manual only |

**MCP tools needed:**
- `knowledge_prune` — run pruning (or automatic background process)
- Extend `health_check` to report entries approaching TTL

**CLI commands needed:**
- `kbz knowledge prune [--dry-run]`

### 7.8 Automatic Promotion

| Trigger | Action |
|---------|--------|
| Tier 3 entry reaches `use_count ≥ 5` AND `miss_count = 0` | Promote to Tier 2 |

**MCP tools needed:**
- `knowledge_promote` — manual promotion
- Automatic promotion check on usage report processing

**CLI commands needed:**
- `kbz knowledge promote <entry-id>`

### 7.9 Post-Merge Compaction

| Scenario | Action |
|----------|--------|
| Exact duplicates | Keep higher confidence; transfer usage counts |
| Near-duplicates (Jaccard > 0.65) | Auto-merge if both confidence > 0.5; else flag |
| Contradictions (Jaccard 0.3–0.6) | Mark `disputed`; surface for resolution |

**MCP tools needed:**
- `knowledge_compact` — run compaction
- `knowledge_resolve_conflict` — resolve disputed entries

**CLI commands needed:**
- `kbz knowledge compact [--dry-run]`

---

## 8. Dependencies

| Dependency | Status | Notes |
|------------|--------|-------|
| Entity model (Feature, Bug, Task) | ✅ Ready | Phase 1 |
| KnowledgeEntry with `git_anchors` field | ✅ Ready | Phase 2b (field exists, unused) |
| Health check system | ✅ Ready | Phase 1+2a |
| MCP/CLI infrastructure | ✅ Ready | Pattern established |
| Document intelligence | ✅ Ready | Can analyze specs for staleness |
| GitHub API access | ❓ New | Requires auth configuration |

---

## 9. Risks

### 9.1 Scope size

Phase 3 includes both Git integration features and deferred knowledge lifecycle features. Combined scope may be larger than Phase 2a or 2b.

**Mitigation:** Monitor during implementation planning. Split into 3a/3b if scope proves too large.

### 9.2 GitHub API complexity

PR operations require GitHub API integration, which introduces authentication, rate limiting, and API versioning concerns.

**Mitigation:** Start with minimal viable operations (create, update description). Expand scope based on actual value delivered.

### 9.3 Worktree edge cases

Git worktrees have edge cases: uncommitted changes, detached heads, nested submodules.

**Mitigation:** Start with happy path. Document known limitations. Add edge case handling incrementally based on actual usage.

### 9.4 Compaction correctness

Post-merge compaction involves duplicate detection and automatic merging of knowledge entries. Incorrect merges could corrupt the knowledge base.

**Mitigation:** Conservative defaults (flag rather than auto-merge when uncertain). Dry-run mode for verification. Tier-based restrictions (never auto-modify Tier 1).

---

## 10. Next Steps

1. **Record design decisions** — Add P3-DES-001 through P3-DES-008 to decision log
2. **Write Phase 3 specification** — `work/spec/phase-3-specification.md`
3. **Write implementation plan** — `work/plan/phase-3-implementation-plan.md`
4. **Implement** — Following established patterns from Phase 1, 2a, 2b

---

## 11. Summary

Phase 3 delivers Git-native collaboration features that enable parallel agent work with proper isolation, quality gates before merge, and self-maintaining knowledge.

Eight Human UX design decisions are now approved, covering branch naming, worktree lifecycle, platform scope, PR operations, merge gate behavior, and cleanup policy.

Implementation details and AI Agent UX decisions are deferred to the appropriate development stage.

The phase proceeds as a single unit unless scope proves too large during planning, at which point knowledge lifecycle features form a natural first tranche.