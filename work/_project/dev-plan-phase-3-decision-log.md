# Phase 3 Decision Log

| Document | Phase 3 Decision Log |
|----------|----------------------|
| Status | Active |
| Created | 2025-01-27 |
| Updated | 2025-01-27 |
| Related | `work/plan/phase-3-scope.md` |
|         | `work/design/workflow-design-basis.md` §16, §22 |
|         | `work/design/machine-context-design.md` §9.5, §9.6 |

---

## Decision Register

| ID | Topic | Status | Date |
|----|-------|--------|------|
| P3-DES-001 | Branch naming convention | accepted | 2025-01-27 |
| P3-DES-002 | Worktree lifecycle ownership | accepted | 2025-01-27 |
| P3-DES-003 | Multiple worktrees per feature | accepted | 2025-01-27 |
| P3-DES-004 | Platform scope | accepted | 2025-01-27 |
| P3-DES-005 | PR operation scope | accepted | 2025-01-27 |
| P3-DES-006 | Merge gate workflow model | accepted | 2025-01-27 |
| P3-DES-007 | Merge gate failure behavior | accepted | 2025-01-27 |
| P3-DES-008 | Cleanup scope and triggering | accepted | 2025-01-27 |

---

## `P3-DES-001: Branch naming convention`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, git-integration, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §16.4

### Decision

The system uses **suggested convention with explicit tracking**.

Branch-to-entity relationships are tracked in workflow state, not inferred from branch names. Branch names are for human readability only.

**Suggested pattern:** `feature/FEAT-01JX-short-slug` or `bug/BUG-01JX-short-slug`

**Enforcement:** None — any branch name is accepted by the system.

**Tracking:** The relationship between branch and entity is stored explicitly in workflow state.

### Rationale

Explicit tracking decouples human conventions from system function. Teams with existing branch naming conventions can adopt kanbanzai without changing their practices. New teams get sensible guidance.

Parsing branch names to infer relationships is fragile and limits flexibility. Explicit tracking is more robust and supports any naming scheme.

### Alternatives Considered

- **Strict enforcement.** Rejected — conflicts with existing team conventions; reduces adoption.
- **No convention.** Rejected — loses the "glance at branch name, know what it's for" benefit for humans.

### Consequences

- System must store branch-to-entity mapping explicitly (implementation decision: where and how).
- Suggested naming convention documented in user-facing docs.
- Tooling (e.g., `kbz worktree create`) can auto-generate branch names following the convention.

### Follow-up Needed

- Define storage model for branch-to-entity relationship (implementation detail).

---

## `P3-DES-002: Worktree lifecycle ownership`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, git-integration, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §16.1

### Decision

The system uses **automatic creation on task activation**.

A worktree is created automatically when the first task within a feature or bug transitions to `in_progress`. This is the moment isolation becomes valuable.

**Trigger:** First task in feature/bug starts (`in_progress` transition).

**Creation:** Automatic by system.

**Visibility:** `kbz worktree list` shows all active worktrees and their entity relationships.

**Destruction:** Automatic cleanup post-merge per P3-DES-008.

### Rationale

Automatic creation removes friction — humans and agents don't need to remember to create worktrees before starting work. The trigger (task starting) is a meaningful workflow event, not arbitrary.

This follows the design principle: humans own intent (starting work on a task), agents and system own execution (creating isolation infrastructure).

Creating on feature creation would be too early (features may sit in backlog without work). Creating on explicit human request adds friction.

### Alternatives Considered

- **Human-initiated.** Rejected — adds friction; humans must remember to create worktrees.
- **System-automatic on feature creation.** Rejected — creates unnecessary worktrees for dormant features.
- **Agent-initiated.** Rejected — less predictable; humans may be surprised by worktrees appearing.

### Consequences

- System must detect task `in_progress` transitions and check if parent feature/bug has a worktree.
- If no worktree exists, system creates one automatically.
- Worktree creation failure should block the transition or warn clearly.

### Follow-up Needed

- Define behavior if worktree creation fails (e.g., disk space, git errors).
- Define behavior for tasks without a parent feature/bug.

---

## `P3-DES-003: Multiple worktrees per feature`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, git-integration, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §16.1

### Decision

The system supports **one worktree per feature or bug**.

All tasks within a feature share one worktree. This is a one-to-one relationship.

**Model:** One worktree per feature/bug entity.

**Parallelism:** Agents working on different features/bugs work in different worktrees. Agents working on tasks within the same feature share a worktree.

**Escape hatch:** Humans can manually create additional worktrees outside the system if truly needed (not a supported workflow).

### Rationale

Tasks within a feature are typically interdependent — they're building toward the same outcome. Full isolation between tasks would complicate merging and increase coordination overhead.

If true parallelism is needed between pieces of work, decompose them into separate features. This keeps the model simple and forces explicit thinking about work boundaries.

### Alternatives Considered

- **One worktree per task.** Rejected — creates many worktrees; complicates merging task branches into feature branch; tasks often have hidden dependencies.
- **Hybrid (one by default, more on request).** Rejected — adds model complexity without clear benefit.

### Consequences

- Worktree-to-entity relationship is one-to-one with features and bugs.
- Tasks do not have their own worktrees; they inherit from their parent.
- System does not support multiple active worktrees for the same feature/bug.

### Follow-up Needed

- None.

---

## `P3-DES-004: Platform scope`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, git-integration, product-scope
- Related:
  - `work/design/workflow-design-basis.md` §22

### Decision

Phase 3 is **GitHub-only**.

PR integration features are built for GitHub. No abstraction layer for other platforms is pre-designed.

**Phase 3:** GitHub support only.

**Core Git features:** Worktrees, branches, and local merge operations are platform-agnostic and work without any platform.

**Future:** When a second platform (GitLab, Bitbucket) becomes a priority, extract an abstraction based on the working GitHub implementation.

### Rationale

GitHub is dominant in the target user base. Building an abstraction layer before implementing a second platform risks over-engineering — the abstraction may not fit other platforms well.

The "don't build beyond current phase" principle applies. Core Git functionality works everywhere; only PR integration is platform-specific, providing a natural seam for future extraction.

### Alternatives Considered

- **Abstract interface, GitHub first.** Rejected for Phase 3 — premature abstraction without a second implementation to validate it.
- **Generic Git only.** Rejected — loses high-value PR integration features.

### Consequences

- Phase 3 specification defines GitHub-specific operations.
- Teams on GitLab or Bitbucket can use worktree and branch features but not PR integration.
- Future phases may add platform abstraction and additional platform support.

### Follow-up Needed

- None for Phase 3.

---

## `P3-DES-005: PR operation scope`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, github-integration, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §22

### Decision

PR operations include **create and update**.

**Create:** System can create a PR from a feature/bug branch to main, with description generated from workflow state.

**Update:** System can update PR description and labels to reflect current workflow state (tasks completed, verification status).

**Not included:** Commenting on PRs and requesting reviewers remain human-initiated actions.

### Rationale

Creating PRs removes significant toil — it's a mechanical step that can be automated. Updating descriptions to reflect workflow state makes the PR a live dashboard of feature progress.

Commenting and requesting reviewers are social actions where human judgment matters. Automated comments may feel spammy. Reviewer assignment involves team dynamics the system shouldn't presume to understand.

### Alternatives Considered

- **Read-only.** Rejected — humans must still manually create PRs, which is significant toil.
- **Full (create, update, comment, request reviewers).** Rejected — commenting and reviewer assignment are social actions better left to humans.

### Consequences

- MCP tools and CLI commands for PR create and update.
- PR description template generated from feature/bug state.
- Labels synced from workflow state (e.g., `verification: passed`).
- No automated PR comments or reviewer requests.

### Follow-up Needed

- Define PR description template format.
- Define which labels are synced from workflow state.

---

## `P3-DES-006: Merge gate workflow model`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, merge-gates, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §16.6

### Decision

Merge gates are **enforced by default with human override available**.

**Default behavior:** If merge gates fail, the merge is blocked.

**Override:** Humans can force-merge with an explicit override flag.

**Accountability:** Override requires a reason, which is logged for audit.

**Example:** `kbz merge FEAT-01J... --override-gates --reason "hotfix, will backfill tests"`

### Rationale

This follows the "strict core, forgiving interface" principle. The system has an opinion (gates matter for quality), but humans retain ultimate control.

Requiring an explicit override flag and reason creates accountability. Teams can review overrides to understand patterns and improve their gates.

### Alternatives Considered

- **Advisory only.** Rejected — gates become "checkbox nobody checks" without enforcement.
- **Strictly enforced (no override).** Rejected — can block urgent fixes if gates are misconfigured or too strict.
- **Configurable per-project.** Rejected — adds complexity; decision fatigue for teams.

### Consequences

- Merge command checks gates before proceeding.
- Gate failure blocks merge by default.
- `--override-gates` flag available with required `--reason`.
- Override events logged for audit.

### Follow-up Needed

- Define audit log format and location.
- Define which gates are overridable vs. hard blocks (if any).

---

## `P3-DES-007: Merge gate failure behavior`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, merge-gates, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §16.6
  - P3-DES-006

### Decision

When merge gates fail, **confirmation is required**.

**Display:** System shows what failed and why.

**Confirm:** Human must explicitly acknowledge the failures before proceeding.

**Reason:** Human provides a brief reason for proceeding despite failures.

This pairs with P3-DES-006 (enforced with override) to create a coherent model: gates are enforced, but humans can override with explicit acknowledgment and justification.

### Rationale

Explicit acknowledgment creates accountability without creating complete blocks. The human must see what failed, understand it, and decide to proceed anyway — not just click through a dialog.

This is more protective than "warning only" (which is easily ignored) but less restrictive than "hard block" (which can strand urgent work).

### Alternatives Considered

- **Hard block (no merge possible).** Rejected — frustrating if gates have bugs or are overly strict; no escape hatch.
- **Warning only (merge proceeds, warning logged).** Rejected — warnings are easily ignored; "alert fatigue" risk.

### Consequences

- Gate failure output includes structured information about what failed.
- Merge with failures requires interactive confirmation (CLI) or explicit override parameter (MCP).
- Reason is captured and logged.

### Follow-up Needed

- Define gate failure output format.
- Define MCP tool behavior (how to represent confirmation in non-interactive context).

---

## `P3-DES-008: Cleanup scope and triggering`

- Status: accepted
- Date: 2025-01-27
- Scope: phase-3, cleanup, human-ux
- Related:
  - `work/design/workflow-design-basis.md` §16.5

### Decision

Cleanup is **automatic with a 7-day grace period**.

**Worktrees:** Auto-cleanup 7 days after merge.

**Branches:** Auto-cleanup 7 days after merge.

**Knowledge entries:** Not auto-deleted. TTL handles Tier 3 entries naturally. High-use Tier 3 entries from the merged branch are surfaced for potential promotion to Tier 2.

**Configuration:** Grace period is configurable per-project.

### Rationale

Automatic cleanup prevents accumulation of stale worktrees and branches. The 7-day grace period provides a recovery window if the merge had issues or the human needs to revisit.

Knowledge entries are handled differently because they have value beyond the branch lifecycle. TTL-based retention (defined in Phase 2b) handles natural expiry. Entries that proved valuable during the branch's lifetime deserve consideration for promotion, not automatic deletion.

### Alternatives Considered

- **Manual only.** Rejected — cleanup is often forgotten; stale artifacts accumulate.
- **Prompted (suggest cleanup, wait for confirmation).** Rejected — adds friction to every merge.
- **Immediate automatic.** Rejected — no recovery window if merge had issues.

### Consequences

- System tracks merge timestamp for cleanup scheduling.
- Background process or on-demand check identifies items past grace period.
- Grace period default is 7 days; configurable in project config.
- Post-merge, system identifies high-use Tier 3 knowledge entries for promotion review.

### Follow-up Needed

- Define configuration schema for grace period.
- Define how promotion candidates are surfaced to users.

---

## Summary

Phase 3 has eight accepted design decisions covering Human UX concerns:

| ID | Topic | Key Choice |
|----|-------|------------|
| P3-DES-001 | Branch naming | Suggested convention, explicit tracking |
| P3-DES-002 | Worktree lifecycle | Automatic on task activation |
| P3-DES-003 | Multiple worktrees | One per feature/bug |
| P3-DES-004 | Platform scope | GitHub-only for Phase 3 |
| P3-DES-005 | PR operations | Create and update |
| P3-DES-006 | Merge gate model | Enforced with override |
| P3-DES-007 | Gate failure behavior | Confirmation required |
| P3-DES-008 | Cleanup | Automatic with 7-day grace |

Implementation details and AI Agent UX decisions are deferred to specification and development phases.