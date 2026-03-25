# Phase 3 Specification: Git Integration and Knowledge Lifecycle

| Document | Phase 3 Specification |
|----------|----------------------|
| Status | Draft |
| Created | 2025-01-27 |
| Updated | 2025-01-27 |
| Related | `work/plan/phase-3-scope.md` |
|         | `work/plan/phase-3-decision-log.md` |
|         | `work/design/workflow-design-basis.md` §16, §22 |
|         | `work/design/machine-context-design.md` §9.5, §9.6, §13.1 |
|         | `work/spec/phase-2b-specification.md` |

---

## 1. Purpose

This specification defines the requirements for Phase 3 of the Kanbanzai workflow system. Phase 3 delivers Git-native collaboration features that enable parallel agent work with proper isolation, quality gates before merge, and self-maintaining knowledge.

Phase 3 builds on:

- **Phase 1** — workflow kernel with entities, validation, and MCP interface
- **Phase 2a** — document intelligence and entity model evolution
- **Phase 2b** — context profiles, knowledge contribution, and confidence scoring

---

## 2. Goals

1. **Enable parallel agent work** — Multiple agents can work on different features or bugs simultaneously without interfering with each other's working state.

2. **Track branch health** — The system understands branch age, drift from main, activity levels, and merge readiness.

3. **Enforce quality gates** — Merge requires verification that tasks are complete, specs are current, tests exist, and the branch is healthy.

4. **Complete the knowledge feedback loop** — Knowledge entries are tied to code, automatically flagged when stale, pruned when unused, and promoted when valuable.

5. **Integrate with GitHub** — Create and update PRs to provide visibility into workflow state through the existing review surface.

6. **Self-clean** — Post-merge cleanup of worktrees, branches, and duplicate knowledge entries happens automatically.

---

## 3. Scope

### 3.1 In scope for Phase 3

**Git integration:**

- Worktree management (create, list, remove, track entity relationships)
- Branch tracking (age, drift from main, activity, merge readiness)
- Merge readiness checks (gates that verify pre-merge conditions)
- GitHub PR integration (create, update descriptions and labels)
- Post-merge cleanup (worktrees, branches)

**Knowledge lifecycle automation (deferred from Phase 2b):**

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

- Replacing GitHub as the canonical workflow store
- GitHub Issues synchronisation
- Automated PR merging without human approval
- Webhook-based real-time synchronisation

---

## 4. Design Principles

### 4.1 Workflow kernel remains source of truth

GitHub is a coordination layer and review surface, not the canonical workflow store. Entity state lives in `.kbz/state/`. PR descriptions reflect this state; they do not define it.

### 4.2 Isolation is infrastructure, not ceremony

Worktrees are created automatically when work begins. Agents and humans should not need to think about isolation setup — it happens as a consequence of starting work.

### 4.3 Gates have teeth, but humans have override

Merge gates are enforced by default. Quality matters. But humans retain ultimate authority and can override with explicit acknowledgment and justification.

### 4.4 Staleness is proactive, not reactive

Knowledge tied to code should be flagged when that code changes, before an agent retrieves stale information and wastes a task cycle discovering the error.

### 4.5 Cleanup is automatic with grace

Post-merge cleanup happens automatically, but with a grace period. Humans shouldn't need to remember to clean up, but they should have time to recover if something went wrong.

---

## 5. Approved Design Decisions

The following Human UX decisions are binding for this specification. See `work/plan/phase-3-decision-log.md` for full rationale.

| ID | Decision | Summary |
|----|----------|---------|
| P3-DES-001 | Branch naming | Suggested convention; explicit tracking in state |
| P3-DES-002 | Worktree lifecycle | Automatic creation on first task `in_progress` |
| P3-DES-003 | Multiple worktrees | One worktree per feature/bug |
| P3-DES-004 | Platform scope | GitHub-only for Phase 3 |
| P3-DES-005 | PR operations | Create and update (no comments or reviewer requests) |
| P3-DES-006 | Merge gates | Enforced by default; human override available |
| P3-DES-007 | Gate failures | Confirmation required with reason |
| P3-DES-008 | Cleanup | Automatic; 7-day grace period; configurable |

---

## 6. Worktree Management

### 6.1 Purpose

Worktrees provide isolated working directories for parallel development. Each feature or bug gets its own worktree so agents working on different units of work do not interfere with each other's files.

### 6.2 Worktree-entity relationship

Each worktree is associated with exactly one feature or bug entity. Tasks inherit their parent's worktree — they do not have independent worktrees.

The relationship is stored in the worktree tracking record, not inferred from branch names.

### 6.3 Worktree tracking record

Worktree state is stored in `.kbz/state/worktrees/` as YAML files, one per worktree.

**Required fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Worktree ID (format: `WT-{ulid}`) |
| `entity_id` | string | Associated feature or bug ID |
| `branch` | string | Git branch name |
| `path` | string | Filesystem path to worktree (relative to repo root) |
| `status` | enum | `active`, `merged`, `abandoned` |
| `created` | timestamp | Creation time (RFC 3339) |
| `created_by` | string | User or agent that triggered creation |
| `merged_at` | timestamp? | Merge completion time, if merged |
| `cleanup_after` | timestamp? | Scheduled cleanup time, if merged |

**Field order for deterministic YAML:**

```yaml
id: WT-01JX...
entity_id: FEAT-01JX...
branch: feature/FEAT-01JX-user-profiles
path: .worktrees/FEAT-01JX-user-profiles
status: active
created: 2025-01-27T10:00:00Z
created_by: sambeau
merged_at: null
cleanup_after: null
```

### 6.4 Worktree lifecycle

**States:**

| State | Description |
|-------|-------------|
| `active` | Worktree is in use; work is ongoing |
| `merged` | Associated branch has been merged; cleanup scheduled |
| `abandoned` | Worktree was manually abandoned without merge |

**Transitions:**

| From | To | Trigger |
|------|-----|---------|
| (none) | `active` | First task in feature/bug starts |
| `active` | `merged` | Merge to main completed |
| `active` | `abandoned` | Manual abandonment |
| `merged` | (removed) | Grace period expires; cleanup runs |
| `abandoned` | (removed) | Manual cleanup |

### 6.5 Automatic worktree creation

When a task transitions to `in_progress`, the system checks whether the parent feature or bug has an active worktree.

If no active worktree exists:

1. Generate worktree ID (`WT-{ulid}`)
2. Generate branch name following suggested convention: `{type}/{entity-id}-{slug}`
   - `type` is `feature` or `bug`
   - `entity-id` is the feature or bug ID
   - `slug` is derived from the entity title (lowercase, hyphens, max 40 chars)
3. Create Git worktree: `git worktree add {path} -b {branch}`
4. Create worktree tracking record
5. Return worktree information to the caller

If worktree creation fails, the task transition should still succeed, but a warning should be emitted. Worktree creation is valuable but not blocking for task progress.

### 6.6 Worktree path convention

Worktrees are created in `.worktrees/` at the repository root:

```
.worktrees/
  FEAT-01JX-user-profiles/
  BUG-01JY-login-crash/
```

The directory name is `{entity-id}-{slug}` for human readability.

### 6.7 Worktree and gitignore

The `.worktrees/` directory should be added to `.gitignore`. Worktrees contain working copies, not committed content.

---

## 7. Branch Tracking

### 7.1 Purpose

Branch tracking provides visibility into branch health: how old is the branch, how far has it drifted from main, when was it last touched, and is it ready to merge.

### 7.2 Tracked metrics

| Metric | Type | Description |
|--------|------|-------------|
| `branch_age_days` | integer | Days since branch creation |
| `commits_behind_main` | integer | Number of commits on main not in this branch |
| `commits_ahead_of_main` | integer | Number of commits on this branch not in main |
| `last_commit_at` | timestamp | Time of most recent commit on branch |
| `last_commit_age_days` | integer | Days since last commit |
| `has_conflicts` | boolean | Whether branch has merge conflicts with main |
| `merge_readiness` | object | Result of merge gate checks |

### 7.3 Staleness thresholds

Branches are considered stale based on configurable thresholds:

| Condition | Default Threshold | Severity |
|-----------|-------------------|----------|
| No commits in X days | 14 days | warning |
| Behind main by X commits | 50 commits | warning |
| Behind main by X commits | 100 commits | error |
| Has merge conflicts | — | error |

Thresholds are configurable in project config:

```yaml
branch_tracking:
  stale_after_days: 14
  drift_warning_commits: 50
  drift_error_commits: 100
```

### 7.4 Branch status computation

Branch status is computed on-demand, not stored. Each call to `branch_status` or `merge_readiness_check` computes current values from Git.

This ensures accuracy without requiring background synchronisation.

---

## 8. Merge Gates

### 8.1 Purpose

Merge gates verify that work is complete and quality requirements are met before allowing merge. Gates are enforced by default; humans can override with explicit acknowledgment.

### 8.2 Gate definitions

| Gate | Check | Severity |
|------|-------|----------|
| `tasks_complete` | All tasks in feature/bug are `done` or `wont_do` | blocking |
| `verification_exists` | Feature/bug has non-empty `verification` field | blocking |
| `verification_passed` | Feature/bug `verification_status` is `passed` | blocking |
| `branch_not_stale` | Last commit within threshold; drift within threshold | warning |
| `no_conflicts` | Branch has no merge conflicts with main | blocking |
| `health_check_clean` | No blocking health-check errors for this entity | blocking |

### 8.3 Gate severity

- **blocking** — Merge cannot proceed without override
- **warning** — Merge can proceed; warning is displayed

### 8.4 Gate override

When blocking gates fail, merge requires:

1. Explicit override flag (`--override-gates` in CLI, `override: true` in MCP)
2. Reason for override (`--reason "..."` in CLI, `override_reason: "..."` in MCP)
3. Confirmation of failures (CLI prompts; MCP requires explicit acknowledgment)

Override events are logged in the worktree record:

```yaml
overrides:
  - gate: tasks_complete
    reason: "Hotfix - will backfill tests"
    overridden_by: sambeau
    overridden_at: 2025-01-27T15:00:00Z
```

### 8.5 Gate check output

Gate check returns structured output:

```yaml
entity_id: FEAT-01JX...
branch: feature/FEAT-01JX-user-profiles
overall_status: blocked  # passed | warnings | blocked
gates:
  - name: tasks_complete
    status: failed
    severity: blocking
    message: "2 tasks not complete: TASK-01JX.2 (in_progress), TASK-01JX.3 (todo)"
  - name: verification_exists
    status: passed
    severity: blocking
    message: null
  - name: branch_not_stale
    status: warning
    severity: warning
    message: "Branch is 21 commits behind main"
  # ... other gates
```

---

## 9. GitHub PR Integration

### 9.1 Purpose

PR integration creates and updates GitHub pull requests to provide visibility into workflow state through GitHub's review interface.

### 9.2 Platform scope

Phase 3 supports GitHub only (per P3-DES-004). Operations use the GitHub REST API.

### 9.3 Authentication

GitHub authentication is configured in `.kbz/local.yaml` (not committed):

```yaml
github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
```

The token requires `repo` scope for PR operations.

If no token is configured, PR operations return an error with a clear message about how to configure authentication.

### 9.4 PR creation

PR creation generates a pull request from the feature/bug branch to the default branch (typically `main`).

**PR title:** Entity title  
**PR body:** Generated from entity state (see §9.6)  
**PR labels:** Synced from entity state (see §9.7)  

The created PR URL is stored in the worktree record:

```yaml
pr_url: https://github.com/owner/repo/pull/42
pr_number: 42
```

### 9.5 PR update

PR update synchronises the PR description and labels with current entity state. This is called:

- Explicitly via `pr_update` tool/command
- Optionally on entity state changes (future enhancement)

Update is idempotent. If the PR description already matches current state, no API call is made.

### 9.6 PR description template

PR descriptions are generated from entity state:

```markdown
## {Entity Title}

{Entity description}

### Tasks

- [x] TASK-01JX.1: Set up database schema (done)
- [x] TASK-01JX.2: Implement API endpoints (done)
- [ ] TASK-01JX.3: Write integration tests (in_progress)

### Verification

{Verification criteria from entity}

**Status:** {verification_status}

### Workflow

- **Entity:** {entity_id}
- **Created:** {created date}
- **Branch:** {branch name}

---
*This description is managed by Kanbanzai. Manual edits may be overwritten.*
```

### 9.7 PR labels

Labels are synced from entity state:

| Condition | Label |
|-----------|-------|
| Entity is Feature | `feature` |
| Entity is Bug | `bug` |
| All tasks complete | `tasks-complete` |
| Verification passed | `verified` |
| Merge gates pass | `ready-to-merge` |

Labels are created in the repository if they don't exist.

### 9.8 PR status reading

The system can read PR status to inform merge readiness:

- CI check status (passed, failed, pending)
- Review status (approved, changes requested, pending)
- Merge conflict status

This information is included in `merge_readiness_check` output when a PR exists.

---

## 10. Post-Merge Cleanup

### 10.1 Purpose

After merge, worktrees and branches should be cleaned up automatically to prevent accumulation of stale artifacts.

### 10.2 Cleanup scheduling

When a branch is merged:

1. Worktree status transitions to `merged`
2. `merged_at` timestamp is recorded
3. `cleanup_after` is set to `merged_at + grace_period`

Default grace period is 7 days, configurable in project config:

```yaml
cleanup:
  grace_period_days: 7
```

### 10.3 Cleanup execution

Cleanup can be triggered:

- **Automatically:** Background check on kanbanzai startup or periodic health check
- **Manually:** `kbz cleanup run` command

Cleanup performs:

1. Remove Git worktree: `git worktree remove {path}`
2. Delete branch: `git branch -d {branch}` (local) and `git push origin --delete {branch}` (remote, if configured)
3. Remove worktree tracking record

### 10.4 Cleanup dry-run

`kbz cleanup list` shows items pending cleanup without executing.

`kbz cleanup run --dry-run` shows what would be cleaned without executing.

### 10.5 Abandoned worktree cleanup

Worktrees marked `abandoned` are cleaned up immediately on next cleanup run (no grace period).

---

## 11. Git Anchoring

### 11.1 Purpose

Git anchoring ties knowledge entries to specific file paths. When anchored files change, the knowledge entry is flagged as potentially stale, enabling proactive staleness detection.

### 11.2 Anchor format

The `git_anchors` field on KnowledgeEntry contains a list of file paths:

```yaml
git_anchors:
  - internal/api/handler.go
  - internal/api/routes.go
```

**Path format:**

- Paths are relative to repository root
- No glob patterns (exact paths only)
- No line numbers (file-level granularity)

**Rationale for exact paths only:**

Glob patterns add complexity (matching, expansion) without proportional benefit. If knowledge applies to multiple files, list them explicitly. If the list would be too long, the knowledge may be too broad — consider splitting into scoped entries.

Line-number anchoring is fragile (lines shift with edits) and adds significant complexity. File-level anchoring catches most meaningful changes.

### 11.3 Staleness detection

A knowledge entry is flagged as potentially stale when any anchored file has been modified since the entry was last confirmed.

**Detection logic:**

1. For each anchored file, get the last commit that modified it
2. Compare that commit's timestamp to the entry's `last_confirmed` timestamp
3. If any anchored file was modified after `last_confirmed`, mark entry as potentially stale

**Staleness status:**

```yaml
staleness:
  is_stale: true
  stale_reason: "Anchored file modified"
  stale_files:
    - path: internal/api/handler.go
      modified_at: 2025-01-26T14:00:00Z
      commit: abc123
  entry_last_confirmed: 2025-01-20T10:00:00Z
```

### 11.4 Staleness detection timing

Staleness is checked:

1. **On retrieval:** When knowledge entries are retrieved via `context_get` or `knowledge_get`, staleness is computed and included in response
2. **On health check:** `health_check` includes stale knowledge entries in its report
3. **On explicit check:** `knowledge_check_staleness` tool for targeted checking

Staleness is **not** checked via background file watching. The system does not maintain filesystem watchers.

### 11.5 Staleness confirmation

When an agent retrieves a knowledge entry and verifies it's still accurate, it can confirm the entry:

```yaml
# knowledge_confirm request
entry_id: KE-01JX...
```

This updates `last_confirmed` timestamp, clearing the staleness flag until anchored files change again.

When an agent finds the entry is wrong, it reports via usage reporting (Phase 2b mechanism), which degrades confidence and may trigger retirement.

### 11.6 Anchoring on contribution

When contributing knowledge, agents should include relevant anchors:

```yaml
# knowledge_contribute request
topic: api-error-format
scope: backend
content: "API errors return JSON with 'error' and 'code' fields"
tier: 3
git_anchors:
  - internal/api/errors.go
```

Anchors are optional. Entries without anchors cannot be proactively flagged as stale but are still subject to confidence-based lifecycle.

---

## 12. TTL-Based Pruning

### 12.1 Purpose

TTL-based pruning automatically removes knowledge entries that haven't been used within their time-to-live period. This prevents knowledge accumulation without ongoing value.

### 12.2 TTL rules by tier

| Tier | Default TTL | TTL Reset | Prune Condition |
|------|-------------|-----------|-----------------|
| Tier 3 (session) | 30 days | Reset to 30 days on use | TTL expires AND `use_count < 3` |
| Tier 2 (architecture) | 90 days | Reset to 90 days on use | TTL expires AND `confidence < 0.5` |
| Tier 1 (conventions) | No expiry | N/A | Manual retirement only |

### 12.3 TTL fields

KnowledgeEntry includes TTL-related fields (defined in Phase 2b, now actively processed):

```yaml
ttl_days: 30              # Current TTL
ttl_expires_at: 2025-02-26T10:00:00Z  # Computed expiry timestamp
last_used: 2025-01-27T10:00:00Z       # Last usage timestamp
```

`ttl_expires_at` is computed: `last_used + ttl_days`

### 12.4 TTL reset on use

When a knowledge entry is used (reported via usage reporting), its TTL is reset:

1. `last_used` is updated to current time
2. `ttl_days` is reset to tier default (30 for Tier 3, 90 for Tier 2)
3. `ttl_expires_at` is recomputed

This ensures actively-used knowledge persists regardless of age.

### 12.5 Pruning execution

Pruning can be triggered:

- **On health check:** Expired entries are identified and reported
- **Explicitly:** `kbz knowledge prune` command
- **On context assembly:** Expired entries matching prune conditions are excluded

Pruning transitions entries to `retired` status rather than deleting them. Retired entries:

- Are excluded from context assembly
- Remain in storage for audit purposes
- Can be restored if retirement was premature

### 12.6 Grace period for new entries

Entries younger than 7 days are exempt from pruning regardless of TTL. This prevents premature removal of entries that haven't had opportunity to be used.

### 12.7 Pruning dry-run

`kbz knowledge prune --dry-run` shows what would be pruned without executing.

---

## 13. Automatic Promotion

### 13.1 Purpose

Automatic promotion elevates high-value Tier 3 (session) knowledge to Tier 2 (architecture) based on usage patterns. This surfaces entries that have proven useful across multiple tasks.

### 13.2 Promotion trigger

A Tier 3 entry is promoted to Tier 2 when:

- `use_count >= 5` AND
- `miss_count = 0` AND
- `confidence >= 0.7`

### 13.3 Promotion execution

Promotion can be triggered:

- **Automatically:** During usage report processing, entries meeting criteria are promoted
- **Manually:** `kbz knowledge promote {entry-id}` command

Promotion updates:

```yaml
tier: 2                   # Changed from 3
promoted_from: KE-01JX... # Original ID (self-reference for audit)
promoted_at: 2025-01-27T10:00:00Z
ttl_days: 90              # Reset to Tier 2 default
```

### 13.4 Promotion notification

When automatic promotion occurs, the entry is included in health check output:

```yaml
promotions:
  - entry_id: KE-01JX...
    topic: api-pagination-format
    promoted_at: 2025-01-27T10:00:00Z
    reason: "use_count=7, miss_count=0, confidence=0.85"
```

This provides visibility into knowledge that has proven valuable.

---

## 14. Post-Merge Compaction

### 14.1 Purpose

When branches merge, knowledge entries created in parallel branches may duplicate or contradict each other. Post-merge compaction detects and resolves these conflicts.

### 14.2 Compaction scope

Compaction applies to:

- **Tier 3 entries:** Auto-compacted according to rules
- **Tier 2 entries:** Flagged for human review; not auto-modified
- **Tier 1 entries:** Never compacted (conflicts are Git merge conflicts)

### 14.3 Detection rules

After a merge, compare knowledge entries added or modified by the merge against existing entries:

| Scenario | Detection | Action |
|----------|-----------|--------|
| Exact duplicate | Same topic AND same normalised content | Keep higher confidence; transfer usage counts |
| Near-duplicate | Same topic OR Jaccard similarity > 0.65 in same scope | Auto-merge if both confidence > 0.5; else flag |
| Contradiction | Same scope AND topic overlap AND Jaccard 0.3–0.6 | Mark both `disputed`; never auto-resolve |
| Independent | Different topic AND different scope | No action |

### 14.4 Jaccard similarity

Jaccard similarity is computed on normalised word sets:

1. Normalise content: lowercase, remove punctuation, split on whitespace
2. Create word set for each entry
3. Jaccard = |intersection| / |union|

### 14.5 Auto-merge rules

When auto-merging near-duplicates:

1. Keep the entry with higher confidence
2. Add usage counts from discarded entry to kept entry
3. Merge `git_anchors` (union of both lists)
4. Update `merged_from` field with discarded entry ID
5. Retire discarded entry with reason "merged into {kept-id}"

### 14.6 Disputed entries

Entries marked `disputed`:

- Are included in context assembly with annotation
- Appear in health check output
- Require human or coordinator agent resolution

Resolution sets one entry to `confirmed` and the other to `retired` (or merges them).

### 14.7 Compaction triggering

Compaction can be triggered:

- **Post-merge hook:** If configured, runs automatically after `git merge` to main
- **Explicitly:** `kbz knowledge compact` command
- **Via MCP:** `knowledge_compact` tool

### 14.8 Compaction output

```yaml
compaction_result:
  duplicates_merged: 2
  near_duplicates_merged: 1
  conflicts_flagged: 1
  details:
    - action: merged
      kept: KE-01JX...
      discarded: KE-01JY...
      reason: "Exact duplicate"
    - action: disputed
      entries: [KE-01JA..., KE-01JB...]
      reason: "Contradictory content in same scope"
```

---

## 15. Storage Model

### 15.1 Worktree storage

Worktree records are stored in `.kbz/state/worktrees/`:

```
.kbz/state/worktrees/
  WT-01JX....yaml
  WT-01JY....yaml
```

One file per worktree, named by worktree ID.

### 15.2 Branch tracking storage

Branch metrics are computed on-demand from Git, not stored. This avoids synchronisation issues.

The worktree record stores the branch name; metrics are computed when requested.

### 15.3 GitHub configuration storage

GitHub authentication is stored in `.kbz/local.yaml` (gitignored):

```yaml
github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  # Optional: override repository detection
  owner: example-org
  repo: example-repo
```

### 15.4 Cleanup configuration storage

Cleanup settings are stored in `.kbz/config.yaml`:

```yaml
cleanup:
  grace_period_days: 7
  auto_delete_remote_branch: true

branch_tracking:
  stale_after_days: 14
  drift_warning_commits: 50
  drift_error_commits: 100
```

### 15.5 KnowledgeEntry extensions

The KnowledgeEntry schema (Phase 2b) is extended with:

```yaml
# TTL management (fields exist; now actively processed)
ttl_expires_at: 2025-02-26T10:00:00Z  # Computed field

# Promotion tracking
promoted_at: 2025-01-27T10:00:00Z     # When promoted
promotion_reason: "use_count=7, miss_count=0"

# Compaction tracking
merged_from: [KE-01JA..., KE-01JB...]  # Entries merged into this one
```

### 15.6 Deterministic formatting

All YAML files follow Phase 1 deterministic formatting rules:

- Block style for mappings and sequences
- Double-quoted strings only when required
- Defined field order per entity type
- UTF-8, LF line endings, trailing newline

---

## 16. MCP Interface

### 16.1 Worktree operations

#### `worktree_create`

Create a worktree for a feature or bug.

**Request:**

```yaml
entity_id: FEAT-01JX...
branch_name: feature/FEAT-01JX-user-profiles  # Optional; auto-generated if omitted
```

**Response:**

```yaml
worktree:
  id: WT-01JX...
  entity_id: FEAT-01JX...
  branch: feature/FEAT-01JX-user-profiles
  path: .worktrees/FEAT-01JX-user-profiles
  status: active
  created: 2025-01-27T10:00:00Z
```

**Errors:**

- `ENTITY_NOT_FOUND` — entity_id does not exist
- `INVALID_ENTITY_TYPE` — entity is not a feature or bug
- `WORKTREE_EXISTS` — active worktree already exists for this entity
- `GIT_ERROR` — Git worktree creation failed

#### `worktree_list`

List all worktrees.

**Request:**

```yaml
status: active  # Optional filter: active | merged | abandoned | all
entity_id: FEAT-01JX...  # Optional filter by entity
```

**Response:**

```yaml
worktrees:
  - id: WT-01JX...
    entity_id: FEAT-01JX...
    branch: feature/FEAT-01JX-user-profiles
    path: .worktrees/FEAT-01JX-user-profiles
    status: active
    created: 2025-01-27T10:00:00Z
  # ... more worktrees
```

#### `worktree_get`

Get worktree for a specific entity.

**Request:**

```yaml
entity_id: FEAT-01JX...
```

**Response:**

```yaml
worktree:
  id: WT-01JX...
  # ... full worktree record
```

**Errors:**

- `ENTITY_NOT_FOUND` — entity_id does not exist
- `NO_WORKTREE` — no worktree exists for this entity

#### `worktree_remove`

Remove a worktree.

**Request:**

```yaml
entity_id: FEAT-01JX...
force: false  # If true, remove even if uncommitted changes exist
```

**Response:**

```yaml
removed:
  id: WT-01JX...
  path: .worktrees/FEAT-01JX-user-profiles
```

**Errors:**

- `NO_WORKTREE` — no worktree exists for this entity
- `UNCOMMITTED_CHANGES` — worktree has uncommitted changes (unless force=true)

### 16.2 Branch operations

#### `branch_status`

Get branch tracking metrics.

**Request:**

```yaml
entity_id: FEAT-01JX...
```

**Response:**

```yaml
branch: feature/FEAT-01JX-user-profiles
metrics:
  branch_age_days: 5
  commits_behind_main: 12
  commits_ahead_of_main: 8
  last_commit_at: 2025-01-26T14:00:00Z
  last_commit_age_days: 1
  has_conflicts: false
warnings:
  - "Branch is 12 commits behind main"
errors: []
```

### 16.3 Merge operations

#### `merge_readiness_check`

Check merge gates for an entity.

**Request:**

```yaml
entity_id: FEAT-01JX...
```

**Response:**

```yaml
entity_id: FEAT-01JX...
branch: feature/FEAT-01JX-user-profiles
overall_status: blocked  # passed | warnings | blocked
gates:
  - name: tasks_complete
    status: passed
    severity: blocking
    message: null
  - name: verification_passed
    status: failed
    severity: blocking
    message: "verification_status is 'pending', expected 'passed'"
  - name: branch_not_stale
    status: warning
    severity: warning
    message: "Branch is 21 commits behind main"
  # ... other gates
pr_status:  # If PR exists
  url: https://github.com/owner/repo/pull/42
  ci_status: passed
  review_status: approved
  has_conflicts: false
```

#### `merge_execute`

Execute merge to main.

**Request:**

```yaml
entity_id: FEAT-01JX...
override: false
override_reason: null
merge_strategy: squash  # merge | squash | rebase
delete_branch: true     # Delete branch after merge
```

**Response:**

```yaml
merged:
  entity_id: FEAT-01JX...
  branch: feature/FEAT-01JX-user-profiles
  merge_commit: abc123def...
  merged_at: 2025-01-27T15:00:00Z
cleanup_scheduled:
  cleanup_after: 2025-02-03T15:00:00Z
```

**Errors:**

- `GATES_FAILED` — blocking gates failed and override not specified
- `OVERRIDE_REASON_REQUIRED` — override=true but no override_reason
- `MERGE_CONFLICT` — branch has conflicts that must be resolved first
- `NO_WORKTREE` — no worktree/branch exists for this entity

### 16.4 GitHub PR operations

#### `pr_create`

Create a GitHub pull request.

**Request:**

```yaml
entity_id: FEAT-01JX...
draft: false  # Create as draft PR
```

**Response:**

```yaml
pr:
  url: https://github.com/owner/repo/pull/42
  number: 42
  title: "User profile management"
  state: open
  draft: false
```

**Errors:**

- `GITHUB_NOT_CONFIGURED` — no GitHub token configured
- `PR_EXISTS` — PR already exists for this branch
- `NO_WORKTREE` — no worktree/branch exists for this entity

#### `pr_update`

Update PR description and labels from entity state.

**Request:**

```yaml
entity_id: FEAT-01JX...
```

**Response:**

```yaml
pr:
  url: https://github.com/owner/repo/pull/42
  updated: true
  changes:
    - "Updated description"
    - "Added label: tasks-complete"
```

**Errors:**

- `NO_PR` — no PR exists for this entity
- `GITHUB_NOT_CONFIGURED` — no GitHub token configured

#### `pr_status`

Get PR status.

**Request:**

```yaml
entity_id: FEAT-01JX...
```

**Response:**

```yaml
pr:
  url: https://github.com/owner/repo/pull/42
  number: 42
  state: open
  draft: false
  ci_status: passed  # passed | failed | pending | none
  review_status: approved  # approved | changes_requested | pending | none
  reviews:
    - user: reviewer1
      state: approved
    - user: reviewer2
      state: pending
  has_conflicts: false
  mergeable: true
```

### 16.5 Cleanup operations

#### `cleanup_list`

List items pending cleanup.

**Request:**

```yaml
include_pending: true   # Items past grace period
include_scheduled: true # Items within grace period
```

**Response:**

```yaml
pending_cleanup:
  - worktree_id: WT-01JX...
    entity_id: FEAT-01JX...
    merged_at: 2025-01-20T10:00:00Z
    cleanup_after: 2025-01-27T10:00:00Z
    status: ready  # ready | scheduled
scheduled_cleanup:
  - worktree_id: WT-01JY...
    entity_id: FEAT-01JY...
    merged_at: 2025-01-25T10:00:00Z
    cleanup_after: 2025-02-01T10:00:00Z
    status: scheduled
```

#### `cleanup_execute`

Execute cleanup.

**Request:**

```yaml
worktree_id: WT-01JX...  # Optional; if omitted, clean all ready items
dry_run: false
```

**Response:**

```yaml
cleaned:
  - worktree_id: WT-01JX...
    branch: feature/FEAT-01JX-user-profiles
    path: .worktrees/FEAT-01JX-user-profiles
    remote_branch_deleted: true
```

### 16.6 Knowledge lifecycle operations

#### `knowledge_check_staleness`

Check staleness for knowledge entries.

**Request:**

```yaml
entry_id: KE-01JX...  # Optional; if omitted, check all anchored entries
scope: backend        # Optional filter
```

**Response:**

```yaml
stale_entries:
  - entry_id: KE-01JX...
    topic: api-error-format
    staleness:
      is_stale: true
      stale_reason: "Anchored file modified"
      stale_files:
        - path: internal/api/errors.go
          modified_at: 2025-01-26T14:00:00Z
          commit: abc123
      entry_last_confirmed: 2025-01-20T10:00:00Z
```

#### `knowledge_confirm`

Confirm a knowledge entry is still accurate.

**Request:**

```yaml
entry_id: KE-01JX...
```

**Response:**

```yaml
confirmed:
  entry_id: KE-01JX...
  last_confirmed: 2025-01-27T10:00:00Z
  staleness_cleared: true
```

#### `knowledge_prune`

Prune expired knowledge entries.

**Request:**

```yaml
dry_run: false
tier: 3  # Optional filter; if omitted, prune all eligible tiers
```

**Response:**

```yaml
pruned:
  - entry_id: KE-01JX...
    topic: old-api-format
    tier: 3
    reason: "TTL expired, use_count=1"
  - entry_id: KE-01JY...
    topic: deprecated-pattern
    tier: 3
    reason: "TTL expired, use_count=2"
dry_run: false
would_prune: null  # Populated if dry_run=true
```

#### `knowledge_promote`

Promote a knowledge entry to a higher tier.

**Request:**

```yaml
entry_id: KE-01JX...
target_tier: 2  # Optional; defaults to current_tier - 1
```

**Response:**

```yaml
promoted:
  entry_id: KE-01JX...
  from_tier: 3
  to_tier: 2
  promoted_at: 2025-01-27T10:00:00Z
```

**Errors:**

- `ALREADY_PROMOTED` — entry is already at target tier or higher
- `CANNOT_PROMOTE_TO_TIER_1` — Tier 1 promotion requires human review

#### `knowledge_compact`

Run post-merge compaction.

**Request:**

```yaml
dry_run: false
scope: backend  # Optional filter
```

**Response:**

```yaml
compaction_result:
  duplicates_merged: 2
  near_duplicates_merged: 1
  conflicts_flagged: 1
  details:
    - action: merged
      kept: KE-01JX...
      discarded: KE-01JY...
      reason: "Exact duplicate"
    - action: disputed
      entries: [KE-01JA..., KE-01JB...]
      reason: "Contradictory content in same scope"
```

#### `knowledge_resolve_conflict`

Resolve disputed knowledge entries.

**Request:**

```yaml
keep: KE-01JX...
retire: KE-01JY...
merge_content: false  # If true, merge content before retiring
```

**Response:**

```yaml
resolved:
  kept: KE-01JX...
  retired: KE-01JY...
  merged: false
```

### 16.7 Extended context operations

The existing `context_get` operation (Phase 2b) is extended to include staleness information:

**Response extension:**

```yaml
knowledge:
  - entry_id: KE-01JX...
    # ... existing fields
    staleness:
      is_stale: true
      stale_files: [internal/api/errors.go]
```

Stale entries are included in context assembly but marked with staleness information so agents can verify before relying on them.

### 16.8 MCP validation

All MCP operations validate inputs strictly per Phase 1 patterns:

- Unknown fields are rejected
- Missing required fields return clear errors
- Invalid field values return clear errors with expected format

---

## 17. CLI Interface

### 17.1 Worktree commands

```bash
# List worktrees
kbz worktree list [--status=active|merged|abandoned|all]

# Create worktree (usually automatic, but available for manual use)
kbz worktree create <entity-id> [--branch=<name>]

# Remove worktree
kbz worktree remove <entity-id> [--force]

# Show worktree for entity
kbz worktree show <entity-id>
```

### 17.2 Branch commands

```bash
# Show branch status
kbz branch status <entity-id>

# List branches with health indicators
kbz branch list [--stale] [--conflicts]
```

### 17.3 Merge commands

```bash
# Check merge readiness
kbz merge check <entity-id>

# Execute merge
kbz merge <entity-id> [--strategy=squash|merge|rebase]
                      [--override-gates --reason="..."]
                      [--no-delete-branch]
```

### 17.4 PR commands

```bash
# Create PR
kbz pr create <entity-id> [--draft]

# Update PR
kbz pr update <entity-id>

# Show PR status
kbz pr status <entity-id>
```

### 17.5 Cleanup commands

```bash
# List pending cleanup
kbz cleanup list

# Run cleanup
kbz cleanup run [--dry-run]
```

### 17.6 Knowledge lifecycle commands

```bash
# Check staleness
kbz knowledge check [--scope=<scope>]

# Confirm entry
kbz knowledge confirm <entry-id>

# Prune expired entries
kbz knowledge prune [--dry-run] [--tier=<tier>]

# Promote entry
kbz knowledge promote <entry-id> [--to-tier=<tier>]

# Run compaction
kbz knowledge compact [--dry-run] [--scope=<scope>]

# Resolve conflict
kbz knowledge resolve --keep=<entry-id> --retire=<entry-id>
```

---

## 18. Health Check Extensions

### 18.1 New health check categories

Phase 3 adds these health check categories:

| Category | Checks |
|----------|--------|
| `worktree` | Worktree state consistency, orphaned worktrees |
| `branch` | Stale branches, excessive drift, conflicts |
| `knowledge_staleness` | Stale knowledge entries (anchored files changed) |
| `knowledge_ttl` | Entries approaching or past TTL expiry |
| `knowledge_conflicts` | Disputed entries requiring resolution |
| `cleanup` | Items pending cleanup past grace period |

### 18.2 Health check output

```yaml
health:
  status: warnings  # ok | warnings | errors
  categories:
    worktree:
      status: ok
      issues: []
    branch:
      status: warnings
      issues:
        - severity: warning
          entity_id: FEAT-01JX...
          message: "Branch 21 commits behind main"
    knowledge_staleness:
      status: warnings
      issues:
        - severity: warning
          entry_id: KE-01JX...
          message: "Anchored file modified since last confirmation"
    knowledge_ttl:
      status: ok
      issues: []
    knowledge_conflicts:
      status: errors
      issues:
        - severity: error
          entries: [KE-01JA..., KE-01JB...]
          message: "Disputed entries require resolution"
    cleanup:
      status: ok
      issues: []
```

---

## 19. Configuration

### 19.1 Project configuration additions

In `.kbz/config.yaml`:

```yaml
# Branch tracking thresholds
branch_tracking:
  stale_after_days: 14
  drift_warning_commits: 50
  drift_error_commits: 100

# Cleanup settings
cleanup:
  grace_period_days: 7
  auto_delete_remote_branch: true

# Knowledge lifecycle settings
knowledge:
  ttl:
    tier_3_days: 30
    tier_2_days: 90
  promotion:
    min_use_count: 5
    max_miss_count: 0
    min_confidence: 0.7
  pruning:
    grace_period_days: 7
```

### 19.2 Local configuration additions

In `.kbz/local.yaml` (gitignored):

```yaml
# GitHub authentication
github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: example-org    # Optional override
  repo: example-repo    # Optional override
```

---

## 20. Acceptance Criteria

### 20.1 Worktree management

- [ ] Worktree is created automatically when first task in feature/bug starts
- [ ] Worktree creation follows suggested branch naming convention
- [ ] Worktree-entity relationship is tracked in state
- [ ] `worktree_list` returns all worktrees with correct status
- [ ] `worktree_remove` removes worktree and updates state
- [ ] Worktree creation failure does not block task transition

### 20.2 Branch tracking

- [ ] `branch_status` returns accurate metrics from Git
- [ ] Stale branch detection uses configured thresholds
- [ ] Merge conflict detection is accurate
- [ ] Drift calculation (commits behind/ahead) is correct

### 20.3 Merge gates

- [ ] All defined gates are checked by `merge_readiness_check`
- [ ] Gate failures block merge by default
- [ ] Override with reason allows merge despite failures
- [ ] Override events are logged
- [ ] Gate check output is structured and actionable

### 20.4 GitHub PR integration

- [ ] PR creation works with valid GitHub token
- [ ] PR description is generated from entity state
- [ ] PR update syncs description and labels
- [ ] PR status returns CI, review, and conflict status
- [ ] Missing GitHub config returns clear error
- [ ] Labels are created if they don't exist

### 20.5 Post-merge cleanup

- [ ] Merged worktree transitions to `merged` status
- [ ] Cleanup is scheduled for grace period after merge
- [ ] `cleanup_list` shows pending and scheduled items
- [ ] `cleanup_execute` removes worktrees and branches
- [ ] Remote branch deletion works if configured
- [ ] Dry-run mode works correctly

### 20.6 Git anchoring

- [ ] `git_anchors` field accepts list of file paths
- [ ] Staleness detection compares file modification to `last_confirmed`
- [ ] Stale entries are flagged in retrieval responses
- [ ] Stale entries appear in health check
- [ ] `knowledge_confirm` clears staleness flag
- [ ] Entries without anchors are not flagged as stale

### 20.7 TTL-based pruning

- [ ] TTL is computed from `last_used` + `ttl_days`
- [ ] TTL resets on knowledge use
- [ ] Pruning respects tier-specific conditions
- [ ] New entries have grace period before pruning
- [ ] Pruned entries transition to `retired`
- [ ] Dry-run mode works correctly

### 20.8 Automatic promotion

- [ ] Promotion triggers when conditions met (use_count >= 5, miss_count = 0, confidence >= 0.7)
- [ ] Promotion updates tier and TTL
- [ ] Promotion is logged and visible in health check
- [ ] Manual promotion works via tool/command
- [ ] Cannot promote directly to Tier 1

### 20.9 Post-merge compaction

- [ ] Exact duplicates are detected and merged
- [ ] Near-duplicates (Jaccard > 0.65) are handled correctly
- [ ] Contradictions are marked `disputed`
- [ ] Tier 2 entries are flagged, not auto-modified
- [ ] Tier 1 entries are never compacted
- [ ] `knowledge_resolve_conflict` resolves disputes
- [ ] Dry-run mode works correctly

### 20.10 Health checks

- [ ] New health check categories are implemented
- [ ] Health output includes all new categories
- [ ] Severity levels are correct
- [ ] Actionable messages are provided

### 20.11 Configuration

- [ ] All configuration options are documented
- [ ] Defaults are sensible
- [ ] Invalid configuration is rejected with clear errors
- [ ] Local config is gitignored

### 20.12 Deterministic storage

- [ ] Worktree records follow deterministic YAML format
- [ ] Round-trip (read-write-read) produces identical output
- [ ] Field order matches specification

---

## 21. Open Questions Resolved During Specification

### 21.1 Git anchor path format

**Decision:** Exact paths only, relative to repository root. No glob patterns, no line numbers.

**Rationale:** Globs add matching complexity; if knowledge applies to many files, list them or consider whether the knowledge is too broad. Line numbers are fragile as code changes.

### 21.2 Change threshold for staleness

**Decision:** Any modification to an anchored file triggers staleness flag.

**Rationale:** Simpler to implement and understand. False positives (flagging unchanged semantics) are acceptable — agents verify on retrieval. False negatives (missing semantic changes) are worse than false positives.

### 21.3 Staleness detection timing

**Decision:** On-demand (during retrieval and health check), not background watching.

**Rationale:** Background file watching adds complexity and resource usage. On-demand detection is sufficient — staleness is checked when knowledge is retrieved, which is when it matters.

### 21.4 MCP tool design approach

**Decision:** Separate tools for distinct operations (create, list, get, remove) rather than combined tools with mode parameters.

**Rationale:** Clearer semantics, better discoverability, simpler validation. Matches Phase 1 and 2 patterns.

---

## 22. Implementation Notes

### 22.1 Git operations

All Git operations should use the `git` command-line tool via subprocess, not a Git library. This ensures compatibility with any Git configuration and avoids library version issues.

Error handling should capture Git stderr and include it in error responses.

### 22.2 GitHub API

Use GitHub REST API v3. GraphQL (v4) is not required for the operations in scope.

Rate limiting should be handled gracefully — if rate limited, return an error with retry guidance.

### 22.3 Worktree path handling

Worktree paths should be relative to repository root in stored records, but operations should resolve to absolute paths for Git commands.

### 22.4 Concurrent operations

Worktree creation and cleanup operations should use file locking to prevent concurrent modifications to the same worktree.

The lock file pattern from Phase 1 applies.

---

## 23. Summary

Phase 3 delivers:

1. **Worktree management** — Automatic creation on task start, explicit tracking, lifecycle management
2. **Branch tracking** — Health metrics, staleness detection, drift monitoring
3. **Merge gates** — Quality enforcement with human override
4. **GitHub PR integration** — Create and update PRs from workflow state
5. **Post-merge cleanup** — Automatic with grace period
6. **Git anchoring** — Tie knowledge to files, detect staleness proactively
7. **TTL pruning** — Automatic retirement of unused knowledge
8. **Automatic promotion** — Elevate high-value session knowledge
9. **Post-merge compaction** — Detect and resolve duplicate/conflicting knowledge

The specification defines 8 Human UX decisions (binding), resolves AI Agent UX decisions during specification, and defers implementation details to development.

All features include MCP tools, CLI commands, health check integration, and acceptance criteria.