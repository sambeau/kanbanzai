| Field  | Value |
|--------|-------|
| Date   | 2026-04-22T00:00:00Z |
| Status | Draft |
| Author | GPT-5.4 |

## Research Question

Which state-backend direction is the better fit for Kanbanzai at this stage:

1. evolving the current Git-native model with structured transition logs and coarser workflow commits, or
2. adding a centralized database-backed state server as an alternative canonical backend?

This report is intended to inform architectural direction, not immediate implementation. The decision it supports is whether Kanbanzai should continue optimizing the Git-native model as the primary path, begin designing for dual-mode support, or plan a larger product shift toward centralized state.

This memo compares the two competing design documents directly:

- `work/design/transition-history-storage.md` — Git-native evolution via per-entity JSONL transition logs and coarser workflow commits
- `work/design/centralized-state-server.md` — centralized database-backed canonical state as an alternative backend

It should be read alongside those two designs rather than as a standalone replacement for them.

## Scope and Methodology

**In scope:**
- comparison of the two newly drafted design directions
- implications for Git-native fidelity, operational complexity, concurrency, auditability, migration cost, and product positioning
- whether both approaches can coexist in one product
- what confidence level is justified by the available evidence

**Out of scope:**
- detailed database schema design
- implementation plan task breakdown
- benchmarking of storage engines
- security model for a hosted multi-tenant service

**Methodology:**
- primary-source review of the two draft design documents:
  - `work/design/transition-history-storage.md`
  - `work/design/centralized-state-server.md`
- review of existing project design constraints around Git-native storage and viewer architecture:
  - `work/design/kanbanzai-1.0.md`
  - `work/design/public-schema-interface.md`
- synthesis against the project's stated identity as a Git-native workflow system and the known pain point of noisy workflow commits

**Primary sources reviewed:**
- `work/design/transition-history-storage.md`
- `work/design/centralized-state-server.md`
- `work/design/git-commit-policy.md`
- `work/design/kanbanzai-1.0.md`
- `work/design/public-schema-interface.md`
- `work/design/agent-interaction-protocol.md`

## Findings

### Finding 1: The Git-native evolution path is the lower-risk response to the current problem

The immediate problem under discussion is noisy Git history caused by using commits as the primary transition log. The Git-native evolution design addresses that problem directly by separating semantic transition history from commit history while preserving the current authority model.

This is a narrower and more proportional response than introducing a centralized state server. It changes storage shape and commit granularity, but it does not require a new deployment model, a new operational surface, or a redefinition of what Git means in Kanbanzai.

Source:
- `work/design/transition-history-storage.md` — primary, current draft; see Overview, Goals and Non-Goals, and Dependencies
- `work/design/git-commit-policy.md` — primary, current design policy; see the sections on commit history supporting review and diagnosis, and workflow-state-only commits

Evidence grade: primary / current

Confidence: high

### Finding 2: A centralized backend solves a broader class of problems than the current one, but at materially higher cost

The centralized-state design addresses not only commit noise, but also real-time coordination, stronger concurrency control, richer queries, and team-shared canonical state. Those are real benefits, but they are broader than the problem that triggered this investigation.

This means centralized state is not merely an alternative implementation of transition history. It is a larger architectural move with consequences for deployment, operations, service reachability, and product identity.

Source:
- `work/design/centralized-state-server.md` — primary, current draft; see Overview, Goals and Non-Goals, Recommended approach, and Operational model in centralized mode

Evidence grade: primary / current

Confidence: high

### Finding 3: The current product identity strongly favors keeping Git-native mode as a first-class option

Existing design documents describe Kanbanzai as a Git-native workflow system and explicitly rely on Git as the transport for shared state and viewer synchronization. This is not an incidental implementation detail; it is part of the product's conceptual model.

A centralized canonical backend is therefore compatible with Kanbanzai only if it is introduced as an explicit expansion of the product model, not as a silent substitution. Retiring Git-native mode would be a product repositioning, not a storage refactor.

Source:
- `work/design/kanbanzai-1.0.md` — primary, historical design basis; see the discussion of Git as transport and the viewer as a separate product
- `work/design/public-schema-interface.md` — primary, current design note; see the viewer assumptions about committed state visibility and independent clones
- `work/design/centralized-state-server.md` — primary, current draft; see Repository relationship in centralized mode and Dependencies

Evidence grade: primary / mixed recency

Confidence: high

### Finding 4: Supporting both backends is plausible only with strict authority boundaries

The centralized-state design correctly identifies that dual support is feasible only if each project has exactly one canonical backend at a time. This is consistent with the Git-native design's emphasis on clear authority and with the broader project discipline around canonical records.

A permanent dual-write model would create ambiguity, drift risk, and difficult repair semantics. A backend abstraction with per-project selection is plausible; simultaneous canonical file and database state is not.

Source:
- `work/design/centralized-state-server.md` — primary, current draft; see Could both possibilities be supported?, What transformation would be required?, and Decisions
- `work/design/agent-interaction-protocol.md` — primary, current design principle on canonical records; see the sections on canonical records and using the workflow system for canonical changes
- `work/design/transition-history-storage.md` — primary, current draft; see Authority and consistency and Dependencies

Evidence grade: primary / current

Confidence: high

### Finding 5: The Git-native evolution path preserves portability and inspectability advantages that a centralized backend weakens

The Git-native model keeps state transparent in repository files and allows read-only consumers to synchronize through Git alone. These are meaningful strengths, especially for small teams and low-ops adoption. Offline operation is not treated as a decision factor here because Kanbanzai is specifically for agentic workflows, and agentic development already assumes online access to AI systems.

A centralized backend weakens all three:

- inspectability shifts from files to service/database tooling
- portability now depends more on service availability and backup/export discipline
- service reachability becomes a more explicit operational dependency

These trade-offs may be acceptable for some teams, but they are genuine losses relative to the current model.

Source:
- `work/design/kanbanzai-1.0.md` — primary, historical design basis; see the discussion of Git-native transport and viewer separation
- `work/design/public-schema-interface.md` — primary, current design note; see the sections on committed-state visibility and viewer freshness
- `work/design/centralized-state-server.md` — primary, current draft; see Repository relationship in centralized mode, Operational model in centralized mode, and Failure modes and handling
- `work/design/transition-history-storage.md` — primary, current draft; see Recommended approach, Query model, and Failure modes and handling

Evidence grade: primary / mixed recency

Confidence: high

### Finding 6: The centralized backend path is strategically valuable if Kanbanzai wants to serve larger teams, but it is not required to solve the current issue

The centralized design is best understood as a strategic expansion path. It becomes attractive when the target environment includes:

- multiple humans and agents coordinating in near real time
- stronger consistency requirements across users
- richer operational dashboards and analytics
- willingness to run shared infrastructure

None of those conditions are necessary to solve the current commit-noise problem. They are conditions under which Kanbanzai may outgrow a purely Git-native model.

Source:
- `work/design/centralized-state-server.md` — primary, current draft; see Problem and Motivation, Recommended approach, and What transformation would be required?
- `work/design/transition-history-storage.md` — primary, current draft; see Problem and Motivation, Recommended approach, and Migration strategy

Evidence grade: primary / current

Confidence: medium-high

## Trade-Off Analysis

| Criterion | Git-native evolution (`work/design/transition-history-storage.md`) | Centralized backend (`work/design/centralized-state-server.md`) |
|-----------|--------------------------------------------------------------------|----------------------------------------------------------------|
| **Solves current commit-noise problem directly** | Strong | Indirect / over-broad |
| **Preserves Git-native identity** | Strong | Weak unless dual-mode |
| **Operational complexity** | Low | High |
| **Human inspectability** | Strong | Medium–weak |
| **Real-time shared coordination** | Weak–medium | Strong |
| **Concurrency control** | Medium | Strong |
| **Cross-entity query power** | Medium initially, stronger with derived index | Strong |
| **Migration cost** | Low–medium | High |
| **Risk of architectural drift** | Low | High |
| **Fit for small teams / solo use** | Strong | Weak–medium |
| **Fit for larger coordinated teams** | Medium | Strong |
| **Viewer model compatibility** | Strong | Requires redesign |
| **Product positioning continuity** | Strong | Weak unless explicitly repositioned |

## Recommendations

### Recommendation 1: Treat Git-native evolution as the primary near-term direction

- **Recommendation:** Prioritize the Git-native evolution path represented by `transition-history-storage.md` as the main response to the current problem.
- **Confidence:** high
- **Based on:** Findings 1, 3, and 5
- **Conditions:** Applies if the immediate goal is to reduce Git noise while preserving Kanbanzai's current product identity and operating model.

This is the most proportional response to the problem that triggered the investigation. It solves the specific pain without forcing a broader architectural commitment. For the concrete design, see `work/design/transition-history-storage.md`.

### Recommendation 2: Treat centralized state as a strategic expansion path, not the default answer to commit noise

- **Recommendation:** Keep the centralized backend design under consideration, but frame it as a strategic option for larger-team deployments rather than the immediate default architecture.
- **Confidence:** high
- **Based on:** Findings 2 and 6
- **Conditions:** Applies unless there is already a clear product decision to target centrally coordinated team deployments as the primary use case.

This keeps the option alive without conflating two different decisions: fixing commit history versus redefining the product topology. For the concrete alternative, see `work/design/centralized-state-server.md`.

### Recommendation 3: If centralized mode is pursued, require one canonical backend per project

- **Recommendation:** Adopt the rule that a project may support either file-backed or database-backed canonical state, but not both simultaneously in steady state.
- **Confidence:** high
- **Based on:** Finding 4
- **Conditions:** Applies to any future dual-mode architecture.

This is the key guardrail that makes dual-mode support plausible rather than chaotic. Both design documents now assume this boundary explicitly.

### Recommendation 4: If centralized mode remains strategically interesting, invest next in backend-neutral service boundaries

- **Recommendation:** The next exploratory engineering step for centralized mode should be persistence-layer decoupling and service-boundary clarification, not immediate database implementation.
- **Confidence:** medium-high
- **Based on:** Findings 2, 4, and 6
- **Conditions:** Applies if the team wants to preserve the option of centralized mode without committing to it immediately.

This creates optionality. It improves the architecture even if Kanbanzai remains Git-native for the foreseeable future. It is also the clearest shared prerequisite between `work/design/transition-history-storage.md` and `work/design/centralized-state-server.md`.

### Recommendation 5: Do not retire the Git-native model without an explicit product-positioning decision

- **Recommendation:** Any move to make centralized state canonical by default should be treated as a product strategy decision and documented as such.
- **Confidence:** high
- **Based on:** Findings 3 and 5
- **Conditions:** Applies if future work begins to assume a server-backed default.

The evidence does not support treating this as a mere implementation detail. It changes what Kanbanzai is. That distinction is reflected directly in the competing design documents: one preserves the current Git-native model, while the other proposes an explicit expansion beyond it.

## Related Documents

- `work/design/transition-history-storage.md` — Git-native evolution path recommended for the near term
- `work/design/centralized-state-server.md` — centralized backend alternative kept as a strategic option
- `work/design/git-commit-policy.md` — commit-history quality constraints that motivated the investigation
- `work/design/kanbanzai-1.0.md` — Git-native product framing
- `work/design/public-schema-interface.md` — viewer assumptions tied to committed state visibility

## Limitations

- This report compares draft design documents, not implemented systems.
- No prototype or benchmark was performed, so performance and operational claims are architectural judgments rather than measured results.
- The `now` tool failed during drafting, so the date in the header was set manually to today's UTC date.
- The analysis is grounded in current project identity and documented design principles; if product goals change materially, the recommendations may change as well.
