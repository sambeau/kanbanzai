# Kanbanzai MCP Server: Design Issues & Feedback Collation

> **Purpose:** Collate field feedback from Kanbanzai 1.0 usage into a clear picture of what's wrong, to inform the design of Kanbanzai 2.0.
>
> **Created:** 2026-03-26T17:15:10Z
> **Status:** Draft
> **Audience:** Design team (human + AI)

---

## 1. Executive Summary

Kanbanzai 1.0 shipped with 97 MCP tools and a comprehensive entity lifecycle model. Field usage revealed a fundamental misalignment: **the API is designed from the system's perspective (entity CRUD operations) but consumed from the agent's perspective (workflow actions)**. The result is that agents bypass Kanbanzai in favour of `grep`, `cat`, and direct YAML editing — because those are faster and more flexible for how agents actually work.

Three independent agent feedback sessions, plus the project manager's own observations, converge on the same core problems:

1. **Too many tools, too few workflows** — 97 tools burning context window budget, most sessions using fewer than 10
2. **No batch operations** — orchestration workflows touch N entities of the same kind, costing N round-trips
3. **Silent side effects** — the system does the right thing (cascade transitions) but doesn't tell the caller what it did
4. **Ceremony over substance** — mandatory parameters (`role`, `dispatched_by`) and strict status prerequisites create friction that agents route around
5. **The wrong abstraction level** — agents think "I'm done with this task" not "call `update_status` on entity type `task` with status `done`"

The Kanbanzai workflow model is sound. The lifecycle enforcement is correct. The YAML-on-disk-in-Git foundation is genuinely good. **The problem is the MCP tool surface** — how the system presents itself to agents.

---

## 2. Who Gave Feedback

| Source | Role | Session Type |
|--------|------|-------------|
| **Project Manager** | Human overseer | Observation across multiple sessions |
| **Feedback 1** | AI agent (orchestrator + implementer) | Implementation: ~10 tasks across 5 features |
| **Feedback 2** | AI agent (orchestrator) | Planning: specs → dev-plans → 31 tasks across 6 features |
| **Feedback 3** | AI agent (doc writer) | Documentation: 5 user-facing docs in parallel |

Each feedback source also provided redesign suggestions, collected separately in §5.

---

## 3. Issues Identified

Issues are grouped by theme. Each issue includes the evidence from feedback and an impact assessment.

### 3.1 Tool Sprawl and Context Window Cost

**The problem:** 97 MCP tools. Every MCP client loads all 97 tool schemas into the context window at connection time — thousands of tokens before the conversation starts. Most sessions use fewer than 10.

**Evidence:**
- Feedback 1: 3 Kanbanzai calls out of an entire multi-task session
- Feedback 3: 17 Kanbanzai calls vs. 36 traditional tool calls (32%/68% split)
- Feedback 2: many tools listed as "never called" with explanations of why
- Project Manager: "burning through tokens at a higher rate than before"

**Impact:** HIGH — This is a tax on every single session, whether or not the tools are used. The tool schemas compete with actual work content for context window space.

**Breakdown of current tool surface:**

| Domain | Approx. Tools | Typical Usage |
|--------|--------------|---------------|
| Entity CRUD (create, get, list, update per type) | ~17 | Low — agents grep YAML instead |
| Document records | ~11 | Low-moderate (submit, approve used; rest unused) |
| Document intelligence | ~9 | Near zero — agents read files directly |
| Knowledge management | ~12 | Near zero — nothing prompts usage |
| Workflow (dispatch, complete, queue, etc.) | ~8 | `work_queue` used; dispatch/complete often bypassed |
| Git integration (worktree, merge, PR, cleanup) | ~12 | Worktree creation used; rest situational |
| Planning (decompose, estimate, conflict) | ~6 | Near zero — manual decomposition preferred |
| Other (checkpoint, incident, config, profile) | ~12 | Rarely used |

### 3.2 No Batch Operations

**The problem:** Orchestration workflows routinely touch N entities of the same kind. Each operation is a separate tool call with full round-trip overhead.

**Evidence:**
- Feedback 2: 5× `doc_record_approve`, 6× `update_status`, 31× `create_task`, 6× `doc_record_submit` — all doing the same operation to multiple entities
- Feedback 3: 5× `update_status` + 5× `complete_task` = 10 calls for lifecycle management on 5 tasks
- Feedback 1: "I frequently need to mark 3–5 tasks done at once after a sub-agent finishes"

**Impact:** HIGH — This is the single largest source of unnecessary tool calls. Batch mutations would cut token usage roughly in half for most orchestration sessions.

### 3.3 Silent Cascade Side Effects

**The problem:** When `doc_record_approve` approves a specification, it auto-transitions the owning feature (e.g., `specifying → dev-planning`). The response says nothing about this. Agents discover it only when subsequent `update_status` calls fail with "self-transition not allowed."

**Evidence:**
- Feedback 2: "Two wasted tool calls and brief confusion" when spec approval silently advanced features
- Feedback 2: "The information the system *acted on* should be in the response it *returns*"

**Impact:** MEDIUM — Wastes calls and causes confusion. Applies everywhere cascades happen: spec approval, worktree creation, merge execution, task completion that unblocks dependencies.

### 3.4 `dispatch_task` Couples Claiming with Context Assembly

**The problem:** `dispatch_task` does two things: (1) transition ready → active, and (2) assemble a context packet for the executing agent. Many agents need #1 but not #2. The context packet is designed for implementation agents; for documentation, research, or triage tasks it's wasted tokens. Agents use `update_status` as a workaround.

**Evidence:**
- Feedback 1: "I never used `dispatch_task` — `dispatched_by` feels ceremonial when I'm the one doing the work"
- Feedback 3: "`dispatch_task` assembles a coding-focused context packet. I chose `update_status` instead."
- Feedback 2: "The context profiles are about Go development, not documentation writing"

**Impact:** MEDIUM — The "proper" tool for claiming work is avoided because it's overloaded with functionality the agent doesn't want. This leads to agents bypassing the dispatch model entirely.

### 3.5 Lifecycle Enforcement Creates Friction for Solo Agents

**The problem:** The lifecycle state machine (ready → active → done) is correct for multi-agent coordination but creates unnecessary ceremony when the orchestrator and implementer are the same agent session.

**Evidence:**
- Feedback 1: "Writing the YAML directly is one operation; using the tools would be dispatch + do work + complete = three calls minimum, with status validation at each step"
- Feedback 1: "`complete_task` requires the task to be in `active` status, but I often skipped the ready → active transition entirely"
- Feedback 1 bypassed tools entirely, editing `.kbz/state/` YAML files directly to mark tasks done

**Impact:** MEDIUM — The lifecycle model is sound, but the tooling makes agents feel the cost of every state transition. Higher-level tools that handle transitions internally would preserve correctness while reducing friction.

### 3.6 No Plan/Feature Dashboard Query

**The problem:** The most common real-world question — "what's the current state of this plan/feature?" — requires chaining 3–4 tool calls and mentally joining the results.

**Evidence:**
- Feedback 1: "A single `grep 'status:' .kbz/state/tasks/TASK-01KM*.yaml` gives me the same information in one call"
- Feedback 2: "To understand the state of P3-kanbanzai-1.0 I called: `get_plan` + `list_entities_filtered` + `doc_record_list` × 2 — four round trips for something I need every time"
- All three feedback sources independently requested a dashboard/overview tool

**Impact:** HIGH — This is the "give me the situation picture" query that happens at the start of every session and repeatedly during orchestration. Its absence forces agents back to grep.

### 3.7 Document Intelligence Tools Unused

**The problem:** The document intelligence layer (9+ tools: `doc_outline`, `doc_classify`, `doc_find_by_concept`, `doc_find_by_entity`, `doc_find_by_role`, `doc_trace`, `doc_impact`, `doc_extraction_guide`, `doc_pending`) saw zero usage across all three feedback sessions.

**Evidence:**
- Feedback 1: "I just read the spec files directly with `read_file`"
- Feedback 2: "Document intelligence tools for extracting entities from documents — I read the specs directly instead"
- Feedback 3: "I used zero of them. I read files directly."

**Why:** Agents either already know which document to read (task summary or human told them), or they need something specific (grep is faster). The intelligence layer adds value for agents that don't know *which* document to read, but that situation rarely arises in practice.

**Impact:** MEDIUM-HIGH — 9+ tools consuming context window space with near-zero usage. The underlying capability may have value, but the current tool-per-operation surface isn't how agents want to access it.

### 3.8 Knowledge System Goes Unused

**The problem:** Knowledge management tools (`knowledge_contribute`, `knowledge_list`, `knowledge_confirm`, `knowledge_flag`, etc.) were never called despite agents learning things during their sessions.

**Evidence:**
- Feedback 2: "Learned things — 'feature status auto-transitions on spec approval', 'Go package named init is invalid' — and contributed none of them"
- Feedback 2: "The tool exists but nothing prompts me to use it at task completion time"
- Feedback 1: "I never used `context_assemble` or `context_report`"

**Why:** Contributing and consuming knowledge are separate steps that agents skip because they're not on the critical path. The system relies on agents voluntarily calling knowledge tools — they don't.

**Impact:** MEDIUM — The knowledge model is well-designed (confidence scoring, TTL, compaction). The problem is the contribution mechanism, not the system. Knowledge contribution should be inline with task completion, not a separate workflow.

### 3.9 `decompose_feature` Bypassed

**The problem:** The purpose-built tool for breaking features into tasks (`decompose_feature`) was ignored by all agents, who decomposed manually instead.

**Evidence:**
- Feedback 2: "I skipped it because (a) I wasn't sure Layer 3 classification had run on the specs, (b) it works one feature at a time — 6 calls for 6 features, (c) after calling it I'd need `decompose_review` then `create_tasks` — the round trip wasn't obviously faster"
- Feedback 1: never mentioned the tool at all

**Why:** Three-step round-trip (`decompose_feature` → `decompose_review` → N × `create_task`) that requires pre-classified documents. Manual decomposition is one mental step.

**Impact:** LOW-MEDIUM — The tool represents significant implementation investment that isn't paying off. The pipeline needs to be collapsed or the entry conditions simplified.

### 3.10 Permission Friction for Human Overseer

**The problem:** The human manager must approve too many individual tool calls, creating a babysitting burden.

**Evidence:**
- Project Manager: "It asks for too many permissions, so I have to baby the process ready to approve even the smallest of actions — despite always taking the time to allow all, I still seem to be permitting actions."

**Impact:** MEDIUM — This is partly an MCP client configuration issue, but the high tool count exacerbates it. Fewer, coarser tools would mean fewer permission prompts.

### 3.11 No Agent Handoff Tool

**The problem:** There is no integration between Kanbanzai's dispatch model and the `spawn_agent` mechanism. Agents manually assemble sub-agent prompts by reading specs, checking file structures, and composing context — 3–5 tool calls plus significant token spend per handoff.

**Evidence:**
- Feedback 1: "If `dispatch_task` could return a pre-formatted prompt for `spawn_agent` — including relevant spec sections, existing code references, and constraints — that would be genuinely useful"
- All three agents composed handoff prompts manually

**Impact:** MEDIUM — This is the gap between Kanbanzai's theoretical dispatch model and how agents actually delegate work.

### 3.12 Missing Document Type for User Documentation

**The problem:** Valid document types are `design, specification, dev-plan, research, report, policy, rca`. User documentation doesn't fit any of these.

**Evidence:**
- Feedback 3: "I registered all 5 docs as type `report` because none of the types map to 'user documentation'"

**Impact:** LOW — Trivial to fix, but indicative of the taxonomy not covering real use cases.

### 3.13 Cross-Feature Task Dependencies Not Modelled

**The problem:** Tasks can depend on other tasks, but not on features. When a task in Feature A is gated on all of Feature B completing, this can only be expressed in prose, not in the entity model.

**Evidence:**
- Feedback 2: "The hard dependency — `init-skill-embedding` cannot start until `skills-content` is complete — is prose in a dev-plan document. The system has no idea about it. The task showed up in the work queue as `ready` immediately."

**Impact:** LOW-MEDIUM — Edge case for most projects, but when it occurs it silently produces incorrect work queue ordering.

---

## 4. What Works Well

Not everything needs fixing. The feedback consistently praised:

| What | Why It Works |
|------|-------------|
| **`work_queue`** | One call, complete picture, auto-promotes eligible tasks. Universally praised as the best tool in the system. |
| **YAML on disk in Git** | Agents can always fall back to direct file operations. Nothing is locked behind the API. Everything is version-controlled. |
| **Lifecycle enforcement** | The state machines are correct. Self-transition errors caught real mistakes. The model is sound — the exposure through tools is the problem. |
| **Document → entity lifecycle coupling** | Approving a spec auto-advances the feature. Elegant and correct. Just needs to surface what it did. |
| **Worktree creation** | `worktree_create` × 5 in parallel was seamless. Isolated branches, clear paths. |
| **The delivery plan document** | A markdown file a human wrote was the single most useful orchestration artefact — more useful than any tool call. Signal that the document-centric approach works. |
| **Knowledge confidence scoring** | Wilson score with use_count/miss_count is self-tuning. The model is right; the contribution UX is wrong. |
| **`health_check`** | Good single diagnostic entry point. |
| **Entity record timestamps** | `.kbz/state/` records had correct timestamps throughout, even when document content didn't. The system's metadata layer is reliable. |

---

## 5. What Agents Would Change — Consolidated Suggestions

All three feedback sources and the project manager's observations point toward the same redesign direction. Here are the recurring themes, consolidated:

### 5.1 Collapse the Tool Surface (97 → ~10–18 tools)

Every feedback source independently arrived at the same conclusion: replace the per-entity-per-operation tool matrix with resource-oriented tools.

**Current:** `create_feature`, `create_task`, `create_bug`, `get_entity`, `list_entities`, `update_status`, `update_entity`, `update_plan`, `update_plan_status`, `list_plans`, `get_plan`, `query_plan_tasks`, `list_entities_filtered`, `list_by_tag`, `list_tags`, `validate_candidate`, `record_decision` ...

**Proposed:** A single `entity` tool with `action` and `type` parameters, plus an `entity_batch` tool for multi-entity operations.

This pattern applies across all domains — documents, knowledge, worktrees, incidents, etc. The projected reduction is from 97 tools to 15–18 resource tools, saving an estimated 60–70% of context window tokens consumed by tool schemas.

### 5.2 Add Batch Mutations

The highest-value single change. Accept arrays where the current API accepts single IDs:

- Approve multiple documents in one call
- Transition multiple entities in one call
- Create multiple tasks in one call
- Complete multiple tasks in one call

### 5.3 Surface Side Effects in Responses

Every mutation that triggers a cascade should return what it changed:

```
/dev/null/example.json#L1-8
{
  "result": { "document": "...", "status": "approved" },
  "side_effects": [
    { "entity": "FEAT-...", "from": "specifying", "to": "dev-planning",
      "reason": "specification approved" }
  ]
}
```

### 5.4 Higher-Level Workflow Tools

Replace the dispatch/complete ceremony with tools that match how agents think:

| Instead of | Provide |
|-----------|---------|
| `work_queue` → `dispatch_task` → work → `complete_task` | `next_task` (claim + context) → work → `finish_task` (complete + knowledge) |
| `get_plan` + `list_entities_filtered` + `doc_record_list` × 2 | `status(plan_id)` — synthesised dashboard |
| Reading specs + checking files + composing prompt | `prepare_handoff(task_id)` — ready-to-use sub-agent prompt |

### 5.5 Decouple Claiming from Context Assembly

Split `dispatch_task` into:
- `claim_task` — just transitions ready → active
- `context_assemble` — standalone, call when wanted
- `dispatch_task` becomes sugar for claim + assemble

### 5.6 Make Knowledge Contribution Inline

Instead of expecting agents to call `knowledge_contribute` separately, accept knowledge entries as a parameter of `finish_task` / `complete_task`. Knowledge contribution happens at the natural moment (task completion) without a separate tool call.

### 5.7 Simplify `decompose_feature` Pipeline

Collapse the three-step `decompose_feature` → `decompose_review` → N × `create_task` into a pipeline that can optionally create tasks directly from the proposal. Make it work without requiring pre-classified documents.

### 5.8 Tool Groups / Lazy Loading

Not all sessions need all tools. Group tools by workflow phase:
- **Core:** entity, queue, health (always loaded)
- **Documents:** document, doc_intel (on demand)
- **Knowledge:** knowledge, profile, context (on demand)
- **Git:** worktree, merge, PR, cleanup (on demand)
- **Planning:** decompose, estimate, conflict (on demand)

This may require MCP protocol evolution, but the system should be designed for it.

### 5.9 Smart Self-Advertising Between Tools

Tools should guide agents toward the next right thing:
- `work_queue` mentions `conflict_domain_check` when parallel tasks exist
- `create_task` prompts `check_duplicates` when similarity might be high
- `finish_task` prompts for knowledge contributions

---

## 6. Root Cause Analysis

The issues above are symptoms. The root causes are:

### 6.1 Database Schema API, Not Workflow API

The API was designed by asking "what operations exist on what entity types?" and creating one tool per answer. The result is a CRUD surface that maps to the storage layer, not to agent workflows.

Agents think in workflows: "I'm done with this task," "what should I work on next," "show me the state of this plan." The tools make them translate these into sequences of database operations.

### 6.2 Designed for Multi-Agent Coordination, Used by Solo Agents

The lifecycle enforcement, role-based context assembly, and formal dispatch/complete protocol are designed for a world where multiple agents work concurrently and need coordination guardrails. In practice, most sessions are a single orchestrator (sometimes spawning sub-agents) who finds the ceremony overhead disproportionate to the coordination benefit.

### 6.3 Tool-Per-Operation in a Token-Budgeted Environment

Each tool registration consumes context window space. The design didn't account for the cost of tool *existence* — only the cost of tool *invocation*. In an LLM context window, unused tools are not free; they compete with work content for attention and budget.

### 6.4 Intelligence Layer Without an On-Ramp

Document intelligence and knowledge management are sophisticated systems with no natural trigger in the agent workflow. Nothing prompts an agent to classify documents, contribute knowledge, or check for duplicates. The capabilities exist but the workflow doesn't lead agents to them.

---

## 7. Summary: What Needs to Happen

The Kanbanzai workflow model — plans, features, tasks, lifecycle state machines, document-driven stage gates, YAML-on-disk-in-Git — is sound and validated by usage. The problem is entirely in **how the MCP server presents this model to agents**.

The 2.0 design needs to answer:

1. **Who are Kanbanzai's users?** Agents of varying capability, working solo or in teams, on tasks ranging from implementation to documentation to research. The API must serve all of these without imposing the overhead of the most complex case on every session.

2. **What do they actually need?** A small number of high-level workflow tools (claim work, finish work, show status, hand off to sub-agent) backed by a comprehensive but unobtrusive entity model. Not 97 CRUD endpoints.

3. **What's the right abstraction?** Workflow actions, not entity operations. "I'm done with this task" rather than "update status on entity type task with ID X to status done." The lifecycle enforcement still happens — it's just inside the tool, not the agent's responsibility to manage.

4. **How do we respect the context budget?** Fewer tools with richer responses. Batch operations. Lazy loading of optional tool groups. Every tool schema that sits in the context window must earn its space by being used.

These questions — and particularly the first one, *who is this for and what do they need* — should drive the 2.0 design before any tool schemas are written.