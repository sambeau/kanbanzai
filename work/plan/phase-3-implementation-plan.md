# Phase 3 Implementation Plan

| Document | Phase 3 Implementation Plan |
|----------|----------------------------|
| Status | Draft |
| Created | 2025-01-27 |
| Updated | 2025-01-27 |
| Related | `work/spec/phase-3-specification.md` |
|         | `work/plan/phase-3-scope.md` |
|         | `work/plan/phase-3-decision-log.md` |

---

## 1. Overview

This document defines the implementation plan for Phase 3: Git Integration and Knowledge Lifecycle.

Phase 3 delivers two major capability areas:

1. **Git Integration** — Worktree management, branch tracking, merge gates, GitHub PR integration, and post-merge cleanup
2. **Knowledge Lifecycle Automation** — Git anchoring, staleness detection, TTL pruning, automatic promotion, and post-merge compaction

The plan organises work into parallel tracks with clear dependencies, verification points, and risk mitigations.

---

## 2. Implementation Strategy

### 2.1 Two parallel streams

Phase 3 work divides naturally into two independent streams:

| Stream | Tracks | Focus |
|--------|--------|-------|
| **Stream 1: Git Integration** | A, B, C, D, E | Worktrees, branches, merge, PRs, cleanup |
| **Stream 2: Knowledge Lifecycle** | F, G, H, I | Anchoring, TTL, promotion, compaction |

These streams share no code dependencies and can proceed in parallel.

Cross-cutting work (Track J: Health Checks, Track K: Configuration) integrates both streams.

### 2.2 Build order within streams

**Stream 1 (Git Integration):**

```
Track A: Worktree Management (foundation)
    ↓
Track B: Branch Tracking (depends on A)
    ↓
Track C: Merge Gates (depends on A, B)
    ↓
Track D: GitHub PR Integration (depends on A)
    ↓
Track E: Post-Merge Cleanup (depends on A, C)
```

**Stream 2 (Knowledge Lifecycle):**

```
Track F: Git Anchoring (foundation, extends Phase 2b)
    ↓
Track G: TTL Pruning (independent, extends Phase 2b)
    ↓
Track H: Automatic Promotion (independent, extends Phase 2b)
    ↓
Track I: Post-Merge Compaction (can start after F)
```

### 2.3 Parallelism opportunities

| Parallel Group | Tracks | Notes |
|----------------|--------|-------|
| Group 1 | A + F + K | Foundations: worktrees, anchoring, config |
| Group 2 | B + G + H | Branch tracking, TTL, promotion (after Group 1) |
| Group 3 | C + D + I | Merge gates, PRs, compaction (after Group 2) |
| Group 4 | E + J | Cleanup, health checks (after Group 3) |

With two agents, both streams can execute in parallel. With more agents, finer parallelism within groups.

---

## 3. Track Breakdown

### Track A: Worktree Management

**Goal:** Enable creation, tracking, and removal of Git worktrees associated with features and bugs.

| Task | Description | Estimate |
|------|-------------|----------|
| A.1 | Define worktree storage model (`internal/worktree/`) | S |
| A.2 | Implement worktree YAML serialisation with deterministic formatting | S |
| A.3 | Implement worktree store (create, get, list, update, delete) | M |
| A.4 | Implement Git worktree operations (shell out to `git worktree`) | M |
| A.5 | Implement branch name generation from entity | S |
| A.6 | Implement automatic worktree creation on task `in_progress` transition | M |
| A.7 | Implement MCP tools: `worktree_create`, `worktree_list`, `worktree_get`, `worktree_remove` | M |
| A.8 | Implement CLI commands: `kbz worktree list/create/remove/show` | S |
| A.9 | Write tests for worktree operations | M |
| A.10 | Integration test: task start triggers worktree creation | S |

**Dependencies:** None (foundational)

**Verification:**
- Unit tests for storage and serialisation
- Integration tests for Git operations
- Round-trip tests for YAML formatting
- End-to-end test: task transition creates worktree

---

### Track B: Branch Tracking

**Goal:** Compute and expose branch health metrics (age, drift, staleness, conflicts).

| Task | Description | Estimate |
|------|-------------|----------|
| B.1 | Implement Git operations for branch metrics (`internal/git/`) | M |
| B.2 | Implement commits-behind/ahead calculation | S |
| B.3 | Implement merge conflict detection | S |
| B.4 | Implement branch age and last-commit-age calculation | S |
| B.5 | Implement staleness threshold evaluation | S |
| B.6 | Implement MCP tool: `branch_status` | S |
| B.7 | Implement CLI commands: `kbz branch status/list` | S |
| B.8 | Write tests for branch tracking | M |

**Dependencies:** Track A (needs worktree-to-branch mapping)

**Verification:**
- Unit tests for Git metric extraction
- Integration tests against real Git repo
- Tests for threshold evaluation

---

### Track C: Merge Gates

**Goal:** Enforce quality checks before merge with override capability.

| Task | Description | Estimate |
|------|-------------|----------|
| C.1 | Define gate interface and registry (`internal/merge/`) | S |
| C.2 | Implement gate: `tasks_complete` | S |
| C.3 | Implement gate: `verification_exists` | S |
| C.4 | Implement gate: `verification_passed` | S |
| C.5 | Implement gate: `branch_not_stale` | S |
| C.6 | Implement gate: `no_conflicts` | S |
| C.7 | Implement gate: `health_check_clean` | S |
| C.8 | Implement gate aggregation and overall status | S |
| C.9 | Implement override mechanism with reason capture | M |
| C.10 | Implement override audit logging in worktree record | S |
| C.11 | Implement merge execution (shell to `git merge`) | M |
| C.12 | Implement MCP tools: `merge_readiness_check`, `merge_execute` | M |
| C.13 | Implement CLI commands: `kbz merge check`, `kbz merge` | M |
| C.14 | Write tests for gates and override | M |

**Dependencies:** Track A, Track B

**Verification:**
- Unit tests for each gate
- Integration tests for gate aggregation
- Tests for override flow
- End-to-end test: merge with passing/failing gates

---

### Track D: GitHub PR Integration

**Goal:** Create and update GitHub PRs from workflow state.

| Task | Description | Estimate |
|------|-------------|----------|
| D.1 | Implement GitHub API client (`internal/github/`) | M |
| D.2 | Implement token configuration in local.yaml | S |
| D.3 | Implement repository detection from Git remote | S |
| D.4 | Implement PR description template generation | M |
| D.5 | Implement PR creation | M |
| D.6 | Implement PR update (description, labels) | M |
| D.7 | Implement label creation/sync | S |
| D.8 | Implement PR status retrieval (CI, reviews, conflicts) | M |
| D.9 | Store PR URL/number in worktree record | S |
| D.10 | Implement MCP tools: `pr_create`, `pr_update`, `pr_status` | M |
| D.11 | Implement CLI commands: `kbz pr create/update/status` | S |
| D.12 | Write tests (mocked GitHub API) | M |

**Dependencies:** Track A (needs worktree/branch)

**Verification:**
- Unit tests with mocked HTTP responses
- Integration test with real GitHub (optional, manual)
- Error handling tests (no token, rate limit, etc.)

---

### Track E: Post-Merge Cleanup

**Goal:** Automatically clean up worktrees and branches after merge.

| Task | Description | Estimate |
|------|-------------|----------|
| E.1 | Implement merge detection and worktree status transition | S |
| E.2 | Implement cleanup scheduling (`cleanup_after` calculation) | S |
| E.3 | Implement grace period configuration | S |
| E.4 | Implement cleanup execution (worktree removal, branch deletion) | M |
| E.5 | Implement remote branch deletion (optional, configurable) | S |
| E.6 | Implement cleanup listing (pending, scheduled) | S |
| E.7 | Implement MCP tools: `cleanup_list`, `cleanup_execute` | S |
| E.8 | Implement CLI commands: `kbz cleanup list/run` | S |
| E.9 | Implement dry-run mode | S |
| E.10 | Write tests for cleanup lifecycle | M |

**Dependencies:** Track A, Track C (needs merge completion)

**Verification:**
- Unit tests for scheduling logic
- Integration tests for Git cleanup operations
- Tests for grace period handling

---

### Track F: Git Anchoring and Staleness

**Goal:** Tie knowledge entries to files and detect when anchored files change.

| Task | Description | Estimate |
|------|-------------|----------|
| F.1 | Validate `git_anchors` field on knowledge contribution | S |
| F.2 | Implement file modification time lookup from Git | M |
| F.3 | Implement staleness detection logic | M |
| F.4 | Extend `knowledge_get` response with staleness info | S |
| F.5 | Extend `context_get` response with staleness info | S |
| F.6 | Implement MCP tool: `knowledge_check_staleness` | S |
| F.7 | Implement MCP tool: `knowledge_confirm` | S |
| F.8 | Implement CLI command: `kbz knowledge check` | S |
| F.9 | Implement CLI command: `kbz knowledge confirm` | S |
| F.10 | Write tests for staleness detection | M |

**Dependencies:** Phase 2b knowledge infrastructure

**Verification:**
- Unit tests for staleness logic
- Integration tests with real Git history
- Tests for confirm/clear cycle

---

### Track G: TTL-Based Pruning

**Goal:** Automatically retire unused knowledge entries based on TTL expiry.

| Task | Description | Estimate |
|------|-------------|----------|
| G.1 | Implement `ttl_expires_at` computation | S |
| G.2 | Implement TTL reset on use (extend usage report processing) | S |
| G.3 | Implement tier-specific prune conditions | S |
| G.4 | Implement grace period for new entries | S |
| G.5 | Implement pruning execution (transition to `retired`) | S |
| G.6 | Implement MCP tool: `knowledge_prune` | S |
| G.7 | Implement CLI command: `kbz knowledge prune` | S |
| G.8 | Implement dry-run mode | S |
| G.9 | Write tests for TTL logic | M |

**Dependencies:** Phase 2b knowledge infrastructure

**Verification:**
- Unit tests for TTL computation
- Tests for tier-specific rules
- Tests for grace period

---

### Track H: Automatic Promotion

**Goal:** Elevate high-value Tier 3 knowledge to Tier 2 based on usage.

| Task | Description | Estimate |
|------|-------------|----------|
| H.1 | Implement promotion trigger detection | S |
| H.2 | Implement promotion execution (tier change, TTL reset) | S |
| H.3 | Integrate promotion check into usage report processing | S |
| H.4 | Implement MCP tool: `knowledge_promote` | S |
| H.5 | Implement CLI command: `kbz knowledge promote` | S |
| H.6 | Implement promotion configuration (thresholds) | S |
| H.7 | Write tests for promotion logic | S |

**Dependencies:** Phase 2b knowledge infrastructure, Track G (TTL)

**Verification:**
- Unit tests for trigger conditions
- Integration tests for promotion flow
- Tests for Tier 1 promotion rejection

---

### Track I: Post-Merge Compaction

**Goal:** Detect and resolve duplicate/contradictory knowledge after branch merges.

| Task | Description | Estimate |
|------|-------------|----------|
| I.1 | Implement Jaccard similarity calculation | S |
| I.2 | Implement exact duplicate detection | S |
| I.3 | Implement near-duplicate detection | S |
| I.4 | Implement contradiction detection | M |
| I.5 | Implement auto-merge for duplicates | M |
| I.6 | Implement `disputed` status marking | S |
| I.7 | Implement tier-based compaction rules | S |
| I.8 | Implement MCP tool: `knowledge_compact` | M |
| I.9 | Implement MCP tool: `knowledge_resolve_conflict` | S |
| I.10 | Implement CLI commands: `kbz knowledge compact/resolve` | S |
| I.11 | Implement dry-run mode | S |
| I.12 | Write tests for compaction logic | M |

**Dependencies:** Track F (anchoring, for merged_from tracking)

**Verification:**
- Unit tests for similarity calculations
- Tests for each detection scenario
- Tests for auto-merge correctness
- Tests for disputed handling

---

### Track J: Health Check Extensions

**Goal:** Integrate Phase 3 features into the health check system.

| Task | Description | Estimate |
|------|-------------|----------|
| J.1 | Implement health category: `worktree` | S |
| J.2 | Implement health category: `branch` | S |
| J.3 | Implement health category: `knowledge_staleness` | S |
| J.4 | Implement health category: `knowledge_ttl` | S |
| J.5 | Implement health category: `knowledge_conflicts` | S |
| J.6 | Implement health category: `cleanup` | S |
| J.7 | Integrate new categories into health check output | S |
| J.8 | Write tests for health check extensions | M |

**Dependencies:** All other tracks (integrates their status)

**Verification:**
- Unit tests for each category
- Integration test for combined output

---

### Track K: Configuration

**Goal:** Implement configuration schema for Phase 3 features.

| Task | Description | Estimate |
|------|-------------|----------|
| K.1 | Extend config schema for `branch_tracking` | S |
| K.2 | Extend config schema for `cleanup` | S |
| K.3 | Extend config schema for `knowledge` (TTL, promotion) | S |
| K.4 | Implement local.yaml schema for `github` | S |
| K.5 | Implement config validation | S |
| K.6 | Document configuration options | S |
| K.7 | Write tests for config parsing | S |

**Dependencies:** None (can start early)

**Verification:**
- Unit tests for schema validation
- Tests for default values
- Tests for invalid config rejection

---

## 4. Dependency Graph

```
                    ┌─────────────────────────────────────────────┐
                    │              Track K: Config                │
                    │            (can start immediately)          │
                    └─────────────────────────────────────────────┘
                                         │
         ┌───────────────────────────────┼───────────────────────────────┐
         │                               │                               │
         ▼                               ▼                               ▼
┌─────────────────┐             ┌─────────────────┐             ┌─────────────────┐
│  Track A:       │             │  Track F:       │             │  Track G:       │
│  Worktree Mgmt  │             │  Git Anchoring  │             │  TTL Pruning    │
│  (foundation)   │             │  (foundation)   │             │  (independent)  │
└────────┬────────┘             └────────┬────────┘             └────────┬────────┘
         │                               │                               │
         ▼                               │                               ▼
┌─────────────────┐                      │                      ┌─────────────────┐
│  Track B:       │                      │                      │  Track H:       │
│  Branch Track   │                      │                      │  Promotion      │
└────────┬────────┘                      │                      └────────┬────────┘
         │                               │                               │
    ┌────┴────┐                          │                               │
    │         │                          │                               │
    ▼         ▼                          ▼                               │
┌───────┐ ┌───────┐             ┌─────────────────┐                      │
│Track C│ │Track D│             │  Track I:       │                      │
│ Merge │ │  PR   │             │  Compaction     │                      │
│ Gates │ │       │             └────────┬────────┘                      │
└───┬───┘ └───────┘                      │                               │
    │                                    │                               │
    ▼                                    │                               │
┌─────────────────┐                      │                               │
│  Track E:       │                      │                               │
│  Cleanup        │                      │                               │
└────────┬────────┘                      │                               │
         │                               │                               │
         └───────────────────────────────┼───────────────────────────────┘
                                         │
                                         ▼
                              ┌─────────────────────┐
                              │  Track J:           │
                              │  Health Checks      │
                              │  (integrates all)   │
                              └─────────────────────┘
```

---

## 5. Effort Estimates

### 5.1 Size definitions

| Size | Effort | Description |
|------|--------|-------------|
| S | 1-2 hours | Simple, well-defined task |
| M | 2-4 hours | Moderate complexity, some design |
| L | 4-8 hours | Complex, may need iteration |

### 5.2 Track estimates

| Track | Tasks | S | M | L | Est. Total |
|-------|-------|---|---|---|------------|
| A: Worktree Management | 10 | 5 | 5 | 0 | 15-25 hrs |
| B: Branch Tracking | 8 | 6 | 2 | 0 | 10-16 hrs |
| C: Merge Gates | 14 | 10 | 4 | 0 | 18-26 hrs |
| D: GitHub PR Integration | 12 | 5 | 7 | 0 | 19-33 hrs |
| E: Post-Merge Cleanup | 10 | 8 | 2 | 0 | 12-20 hrs |
| F: Git Anchoring | 10 | 7 | 3 | 0 | 13-19 hrs |
| G: TTL Pruning | 9 | 8 | 1 | 0 | 10-14 hrs |
| H: Automatic Promotion | 7 | 7 | 0 | 0 | 7-14 hrs |
| I: Post-Merge Compaction | 12 | 8 | 4 | 0 | 16-24 hrs |
| J: Health Check Extensions | 8 | 7 | 1 | 0 | 9-13 hrs |
| K: Configuration | 7 | 7 | 0 | 0 | 7-14 hrs |
| **Total** | **107** | **78** | **29** | **0** | **136-218 hrs** |

### 5.3 Phase estimate

At 6-8 productive hours per day:

- **Single agent:** 17-36 days
- **Two agents (parallel streams):** 10-20 days
- **Three+ agents:** 8-15 days

This is comparable to Phase 2a+2b combined, which is expected given the scope.

---

## 6. Implementation Order

### 6.1 Recommended sequence (single agent)

| Week | Focus | Tracks |
|------|-------|--------|
| 1 | Foundations | K (config), A (worktree), F (anchoring) |
| 2 | Git operations | B (branch), G (TTL), H (promotion) |
| 3 | Merge and PRs | C (gates), D (PR) |
| 4 | Compaction and cleanup | I (compaction), E (cleanup) |
| 5 | Integration | J (health), polish, documentation |

### 6.2 Parallel execution (two agents)

**Agent 1: Stream 1 (Git Integration)**

| Week | Tracks |
|------|--------|
| 1 | K (config), A (worktree) |
| 2 | B (branch tracking) |
| 3 | C (merge gates), D (PR) |
| 4 | E (cleanup) |
| 5 | J (health - git categories) |

**Agent 2: Stream 2 (Knowledge Lifecycle)**

| Week | Tracks |
|------|--------|
| 1 | F (git anchoring) |
| 2 | G (TTL pruning), H (promotion) |
| 3 | I (compaction) |
| 4 | J (health - knowledge categories) |
| 5 | Integration, polish |

---

## 7. Verification Strategy

### 7.1 Test coverage requirements

| Category | Requirement |
|----------|-------------|
| Unit tests | All business logic functions |
| Integration tests | Git operations, GitHub API (mocked) |
| Round-trip tests | YAML serialisation for worktree records |
| End-to-end tests | Key workflows (task start → worktree, merge → cleanup) |

### 7.2 Verification checkpoints

| Checkpoint | Tracks | Verification |
|------------|--------|--------------|
| CP1: Worktree basics | A | Create/list/remove worktrees; automatic creation on task start |
| CP2: Branch health | A, B | Branch metrics computed correctly; staleness detection |
| CP3: Merge flow | A, B, C | Gates enforced; override works; merge executes |
| CP4: PR integration | A, D | PR create/update with mocked GitHub |
| CP5: Cleanup | A, C, E | Post-merge cleanup scheduled and executed |
| CP6: Knowledge staleness | F | Anchored entries flagged when files change |
| CP7: TTL and promotion | G, H | Entries pruned/promoted correctly |
| CP8: Compaction | I | Duplicates merged; conflicts flagged |
| CP9: Health integration | J | All categories report correctly |
| CP10: Full integration | All | End-to-end workflow from task to merge to cleanup |

### 7.3 Acceptance criteria verification

Each acceptance criterion from the specification (§20) must have at least one test that exercises it. The test file should reference the criterion by number (e.g., `// Verifies §20.1: Worktree is created automatically when first task starts`).

---

## 8. Risk Mitigations

### 8.1 Git operation reliability

**Risk:** Git subprocess calls may fail in unexpected ways (permissions, corrupt repo, unusual configs).

**Mitigation:**
- Capture and log stderr from all Git commands
- Return structured errors with Git output included
- Test against repos with different configurations
- Graceful degradation: worktree creation failure doesn't block task progress

### 8.2 GitHub API complexity

**Risk:** GitHub API has many edge cases (rate limiting, pagination, permissions, enterprise vs. cloud).

**Mitigation:**
- Start with minimal scope (create, update, status)
- Implement rate limit handling with clear user feedback
- Use mocked tests for most coverage; manual integration test for validation
- Clear error messages when GitHub operations fail

### 8.3 Compaction correctness

**Risk:** Auto-merging knowledge entries incorrectly could corrupt the knowledge base.

**Mitigation:**
- Conservative defaults: flag rather than merge when uncertain
- Never auto-modify Tier 1 or Tier 2 entries
- Dry-run mode for verification before execution
- Comprehensive tests for each detection scenario
- Easy rollback: retired entries are not deleted

### 8.4 Scope creep

**Risk:** Phase 3 is large; scope may expand during implementation.

**Mitigation:**
- Specification is the contract; additions require explicit approval
- Track velocity against estimates; flag early if falling behind
- Defer nice-to-haves to Phase 3.1 or later
- Monitor scope during implementation; split if needed

### 8.5 Performance with large repos

**Risk:** Git operations (commits behind, conflict detection) may be slow on large repos.

**Mitigation:**
- Compute on-demand, not background; user initiates when needed
- Consider caching for expensive operations (with TTL)
- Profile and optimise if needed
- Document performance characteristics

---

## 9. Definition of Done

A track is complete when:

1. All tasks are implemented
2. All tests pass (unit, integration, round-trip)
3. MCP tools follow existing patterns and pass validation
4. CLI commands are consistent with existing UX
5. Code follows project style (go fmt, go vet, goimports)
6. No race conditions (go test -race passes)
7. Health check integration works (if applicable)
8. Relevant acceptance criteria have passing tests

Phase 3 is complete when:

1. All tracks are complete
2. All acceptance criteria (§20 in spec) are verified
3. Full integration test passes
4. Documentation is updated (AGENTS.md, README.md if needed)
5. No blocking health check errors in a clean instance

---

## 10. Open Items

### 10.1 Questions to resolve during implementation

| Question | Disposition |
|----------|-------------|
| Worktree path: absolute or relative in Git commands? | Resolve in Track A |
| GitHub: handle enterprise URLs? | Defer to future; document limitation |
| Compaction: run automatically on merge or require manual trigger? | Start with manual; add auto if valuable |
| Cleanup: run on startup, on schedule, or only manual? | Start with manual and startup check |

### 10.2 Potential Phase 3.1 scope

If scope exceeds estimates, these can be deferred:

- Remote branch deletion (E.5)
- PR label creation (D.7) — just sync existing labels
- Automatic compaction triggering
- Health check severity tuning

---

## 11. Summary

Phase 3 implements Git integration and knowledge lifecycle automation through 11 tracks and 107 tasks.

**Key deliverables:**

- Worktree management with automatic creation
- Branch health tracking and staleness detection
- Merge gates with human override
- GitHub PR create/update
- Post-merge cleanup with grace period
- Git anchoring for proactive staleness
- TTL-based knowledge pruning
- Automatic Tier 3 → Tier 2 promotion
- Post-merge knowledge compaction

**Estimated effort:** 136-218 hours (10-20 days with two agents)

**Verification:** 10 checkpoints, acceptance criteria tests, full integration test

**Risks mitigated:** Git reliability, GitHub API complexity, compaction correctness, scope creep

The plan supports parallel execution with clear dependencies, enabling efficient multi-agent implementation.