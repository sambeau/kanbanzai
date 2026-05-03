# Workflow UX Gaps — Findings Report

*Identified during trigger and conversational guide research.*

This report documents contradictions, missing connections, and silent failures found in the skill and workflow files. Each entry describes the gap, where it manifests in the skill files, the user impact, and a suggested fix. These are candidates for skill improvements — they are separate from the documentation work that produced this report.

---

## Gap 1: Entity creation timing — contradicting rules between two skills

**Files in conflict:**
- `.kbz/skills/kanbanzai-workflow/SKILL.md` — Emergency Brake section
- `.kbz/skills/kanbanzai-planning/SKILL.md` — When Planning Is Done section

**The contradiction:**

`kanbanzai-workflow` lists as an Emergency Brake condition: *"Entities without an approved design. You are about to create Plan, Feature, or Task entities and no approved design document exists."* — the agent must stop.

`kanbanzai-planning` states: *"Entity creation requires clear scope but does not require a design document to exist yet."*

These rules directly contradict. An agent that reads `kanbanzai-workflow` before `kanbanzai-planning` will refuse to create entities at the end of a successful planning conversation, citing the emergency brake. An agent that reads `kanbanzai-planning` first will create them freely. Behaviour is unpredictable depending on which skill is prioritised.

**User impact:** A user who has just concluded a planning conversation and says "create a Feature entity for this" may find the agent refusing. The error message ("no approved design exists") makes it look like the user has done something wrong, when in fact the planning workflow explicitly permits entity creation before a design exists.

**Suggested fix:** Clarify both files. The `kanbanzai-workflow` emergency brake should specify that it applies to *implementation-phase entity creation* — specifically Task entities, or Feature entities being advanced to `specifying` or beyond — not to Plan and Feature entity creation during the planning phase. Update the emergency brake condition to: *"You are about to begin writing design content (not creating planning entities) without an approved design document."*

---

## Gap 2: Dev-plan approval is a silent non-event

**File:** `.kbz/skills/write-dev-plan/SKILL.md` — Step 7

**The issue:**

Step 7 contains this note: *"Approving a dev-plan does NOT automatically transition the feature to `developing`."* This is correct system behaviour. However:

1. It is buried in an agent-facing skill file and never surfaced to the user
2. It is counterintuitive — every other document approval in the workflow unlocks the next stage and the agent (in practice) advances the feature
3. There is no documented agent behaviour specifying what should happen immediately after dev-plan approval

**User impact:** The user approves the dev-plan and nothing happens. There is no error, no explanation, and no visible next step. The user either re-approves, assumes the system is broken, or does not know to say "advance [feature] to developing." This is one of the most consistently reported points of confusion in user sessions.

**Suggested fix:** Two changes:

1. Add a mandatory post-approval step to `write-dev-plan` Step 7: after recording dev-plan approval, the agent should immediately call `entity(transition)` to advance the feature to `developing` and report: *"Dev-plan approved. [Feature] is now in `developing` status — tasks are ready to be dispatched."* The agent should not wait to be asked.

2. If the auto-advance is not desirable for architectural reasons, add an explicit prompt at the end of Step 7: *"Say 'advance [feature] to developing' when you are ready to begin implementation."*

---

## Gap 3: `write-dev-plan` and `decompose-feature` are separately triggered with no compound default

**File:** `.kbz/stage-bindings.yaml` — `dev-planning` entry

**The issue:**

The stage binding for `dev-planning` lists both `write-dev-plan` and `decompose-feature` as the skills for the stage. But:

- They have separate trigger phrase sets with no overlap
- They produce separate outputs (document vs. task entities)
- Neither trigger set indicates that the other operation is also required

A user who says "write a dev-plan" gets a document with no tasks. A user who says "decompose [feature] into tasks" gets tasks with no document. Neither operation makes it clear that the other should also be done.

**User impact:** User triggers "write a dev-plan", approves it, then tries to start development — and finds the work queue empty because no task entities were ever created. This is a common failure mode that requires the user to check status and then make a second request.

**Suggested fix:**

1. Add a compound trigger phrase to both skills: *"Write a dev-plan and decompose [feature] into tasks"* should be explicitly listed in both trigger sections and documented as the canonical dev-planning invocation.
2. Add a proactive prompt at the end of `write-dev-plan` Step 7: *"The dev-plan document is complete. Should I also decompose [feature] into task entities now?"* — making the two-part nature visible rather than hidden.

---

## Gap 4: No orchestrated specification path for multi-feature plans

**Affected file:** `.kbz/skills/write-spec/SKILL.md`

**The issue:**

`write-spec` operates strictly per-feature. For a Plan with N features, N separate specification requests are required, each followed by a separate approval cycle. There is no "write specifications for all features in this plan" trigger or orchestration pattern.

**User impact:** After a planning conversation that produces a Plan with multiple Features, the user discovers they must spec each feature individually in series with no tooling support. For plans with 5+ features, this is time-consuming and error-prone — easy to lose track of which features have approved specs and which do not.

**Suggested fix:** Introduce an `orchestrate-specification` stage or extend the orchestration system to cover the specifying phase. This would iterate over all features in a plan, write a spec for each (calling `write-spec` per feature), present them for sequential approval, and advance each feature as approvals come in. This mirrors the `orchestrate-development` pattern applied one stage earlier.

At minimum: add a "spec all features in plan" trigger to `write-spec` that makes the iteration explicit, presents each spec one at a time, and stops for approval at each one before proceeding to the next.

---

## Gap 5: Approve-and-advance not consistently connected across stages

**Affected files:** All stage skill files

**The issue:**

In every stage, document approval and feature lifecycle advancement are two separate operations. Approving a document opens a gate but does not walk the feature through it. The agent must call `entity(transition)` separately. This behaviour is not documented in any human-facing file, and agent behaviour across stages is inconsistent:

- Some agents advance the feature after approval as a natural next step
- Others do not, leaving the feature stuck with no visible explanation

**User impact:** After approving any document, the user sometimes sees the feature advance and sometimes nothing happens. The inconsistency is more confusing than a consistent rule would be — users learn the behaviour from one stage and expect it in others, which it then fails to exhibit.

**Suggested fix:** Establish an explicit behavioural rule in each stage skill: *"After recording document approval, immediately transition the feature to the next lifecycle state and report the new state to the user."* Make this explicit in the skill procedures rather than leaving it to agent discretion.

The target message at every approval: *"[Document type] approved. [Feature] is now in `[next state]` status. Next step: [exact phrase to say next]."*

This would also resolve the dev-plan gap (Gap 2) as a special case.

---

## Gap 6: Design document can be written without a Feature entity

**Affected file:** `.kbz/skills/write-design/SKILL.md`

**The issue:**

`write-design` does not verify that a Feature entity exists in `designing` status before drafting the document. A user can say "write a design for [topic]" and receive a registered design document with no Feature entity attached to it. The document floats in the corpus without a lifecycle anchor.

**User impact:** The user writes a design, reviews it, approves it — then discovers the specifying stage cannot proceed because no Feature entity exists in the right state. The approved document may not be properly associated with any feature, requiring manual cleanup to create the entity and link the document retroactively.

**Suggested fix:** Add a prerequisite check to `write-design` Step 1 (Establish Context): verify that a Feature entity exists and is in `designing` status before writing anything. If not, stop and ask: *"No Feature entity found for this work. Should I create one first?"* This mirrors the prerequisite guard in `write-spec` (which correctly checks for an approved design before proceeding).

---

## Summary

| # | Gap | Severity | Type |
|---|---|---|---|
| 1 | Entity creation timing — contradicting rules | High | Spec conflict between two skill files |
| 2 | Dev-plan approval is a silent non-event | High | Missing post-approval feedback and action |
| 3 | `write-dev-plan` / `decompose-feature` separation | Medium | Missing compound default trigger |
| 4 | No orchestrated specification for multi-feature plans | Medium | Missing orchestration pattern |
| 5 | Approve-and-advance inconsistent across stages | High | Missing explicit behavioural rule |
| 6 | Design doc creatable without a Feature entity | Low | Missing prerequisite guard |

Gaps 2 and 5 are likely the highest-ROI fixes — they affect the most common workflow path and produce the most visible user confusion. Gap 1 affects onboarding and planning session reliability. Gaps 3 and 4 affect productivity on plans with multiple features. Gap 6 is low-frequency but produces confusing cleanup work when it occurs.
