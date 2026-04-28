# Interactive Help System Design

| Field | Value |
|-------|-------|
| Author | Sam Phillips |
| Created | 2026-07-19 |

Related:

- `work/design/public-release-documentation.md` (documentation design — this system consumes its output)
- `work/design/fresh-install-experience.md` (init command and onboarding)
- `work/design/skills-system-redesign-v2.md` (roles and skills architecture)
- `work/design/workflow-design-basis.md` §15 (knowledge and memory)

---

## 1. Purpose

This document designs an interactive help system for Kanbanzai: a `help` MCP tool, a `teacher` role, and an associated skill that together provide on-demand, in-context guidance to users learning the system.

The help system complements the written documentation designed in `work/design/public-release-documentation.md`. The documentation is for reading; the help system is for asking. Some users prefer one, some the other, most use both.

This design can be built independently of the documentation, but its content — the help topics — cannot be finalised until the documentation exists. The dependency is: documentation design → help system design (this document) → documentation production → help content population.

---

## 2. Problem Statement

A new Kanbanzai user's first interaction with the system is a chat conversation with an AI agent. That agent has no Kanbanzai-specific knowledge unless it happens to be loaded. The user asks "how do I approve a design?" or "what's a stage gate?" and the agent either guesses, hallucinates, or says it doesn't know.

Three approaches were considered and rejected:

### 2.1 Preloaded Knowledge Entries

Ship `kanbanzai init` with tier-2 knowledge entries covering core concepts. These would be surfaced in every context assembly via `next` and `handoff`.

**Rejected because:** Context assembly loads *all* qualifying knowledge entries into every context packet — there is no relevance filtering. The current `asmLoadKnowledge` function loads all tier-2 entries with confidence ≥ 0.3 whose scope matches `"project"` or the current role. Fifty preloaded system entries would add 3,000–5,000 tokens to every task's context, including implementation tasks where the agent doesn't need to know what a stage gate is. The knowledge system is designed for "operational facts that are always relevant," not "reference material that's sometimes relevant."

### 2.2 Documentation Files in `.kbz/`

Ship documentation as files inside `.kbz/docs/` or `.kbz/help/`, written by `kanbanzai init` and readable by agents via `read_file`.

**Rejected because:** It clutters the user's repository with documents they didn't write. The `.kbz/` directory is for project state, not reference material. Files would need the managed-marker versioning pattern, adding maintenance complexity. And agents would need to know *which* file to read for a given question — requiring either a table of contents mechanism or topic-to-file mapping, which is essentially what the `help` tool provides but with more moving parts.

### 2.3 Knowledge Queries on Demand

No preloaded content. The teacher role queries the knowledge base with `knowledge(action: "list", tags: ["system"])` when the user asks a question.

**Rejected as primary mechanism because:** On a fresh project the knowledge base is empty. The entries would have to be preloaded, which brings back the problems from §2.1. As a supplementary mechanism it could work — but it needs a primary source of content that doesn't pollute the knowledge base.

---

## 3. Design: The `help` Tool

### 3.1 Overview

A new MCP tool, `help`, that serves documentation content embedded in the `kanbanzai` binary. Content is organised into topics. The tool returns the relevant section when asked, or lists available topics when browsed.

The content is compiled into the binary at build time using Go's `embed.FS`, the same mechanism used for embedded skills. It is versioned with the binary: when the user upgrades `kanbanzai`, the help content updates automatically with no `init` re-run needed.

### 3.2 Interface

```
help(topic: "stage-gates")        → returns the help content for that topic
help(topic: "approval")           → returns the help content for that topic
help(list: true)                  → returns all available topics with one-line descriptions
help(search: "how do I approve")  → returns topics matching the search terms
```

Parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `topic` | string | no | Topic identifier. Returns the content for that topic. |
| `list` | boolean | no | When true, returns all available topics with descriptions. |
| `search` | string | no | Free-text search query. Returns matching topics ranked by relevance. |

Exactly one of `topic`, `list`, or `search` must be provided.

### 3.3 Tool Registration

The `help` tool is registered in the `core` group. It should be available in every preset (minimal, orchestration, full) because help is needed most when the user is just getting started — which is when they are most likely to be running a minimal configuration.

This makes it the 23rd MCP tool (or the 9th tool in the core group, alongside `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health`, and `server_info`).

### 3.4 Tool Annotations

```
ReadOnly:     true
Destructive:  false
Idempotent:   true
OpenWorld:    false
```

The tool reads from embedded content and has no side effects.

### 3.5 Return Format

For `topic` queries, the tool returns:

```
{
  "topic": "stage-gates",
  "title": "Stage Gates and Approval",
  "content": "... markdown content ...",
  "related_topics": ["approval", "workflow-stages", "feature-lifecycle"],
  "see_also": "docs/workflow-overview.md §7"
}
```

For `list` queries:

```
{
  "topics": [
    {"topic": "stage-gates", "title": "Stage Gates and Approval", "category": "workflow"},
    {"topic": "knowledge-system", "title": "The Knowledge System", "category": "orchestration"},
    ...
  ],
  "count": 25
}
```

For `search` queries:

```
{
  "query": "how do I approve",
  "matches": [
    {"topic": "approval", "title": "Approving Documents and Features", "category": "workflow", "relevance": "high"},
    {"topic": "stage-gates", "title": "Stage Gates and Approval", "category": "workflow", "relevance": "medium"},
    ...
  ]
}
```

### 3.6 Search Implementation

Search does not need to be sophisticated. A simple approach:

1. Tokenise the query into lowercase words
2. Match against topic identifiers, titles, category names, and a keywords list per topic
3. Rank by number of matching tokens
4. Return the top 5 matches

This is adequate for a corpus of 20–30 topics. If the topic count grows substantially, a more sophisticated approach (TF-IDF, prefix matching) can be added later without changing the interface.

### 3.7 Content Size Budget

Each help topic should be concise — a target of 300–800 tokens. This is enough to explain a concept, show a brief example, and point to the full documentation. It is not enough to replace the documentation.

The total embedded help corpus at 25–30 topics would be roughly 10,000–25,000 tokens of source text. Compressed in the binary, this is negligible (tens of kilobytes).

---

## 4. Help Content Structure

### 4.1 Topic Anatomy

Each topic is a Markdown file in the embedded filesystem with structured front matter:

```
---
topic: stage-gates
title: Stage Gates and Approval
category: workflow
keywords: [approve, approval, gate, transition, lifecycle, advance]
related: [approval, workflow-stages, feature-lifecycle]
see_also: "docs/workflow-overview.md"
---

Stage gates are checkpoints between workflow stages. A feature cannot advance
to the next stage until its gate passes...
```

The front matter provides metadata for listing and search. The body is the content returned by `help(topic: "stage-gates")`.

### 4.2 Categories

Topics are grouped into categories for browsable listing:

| Category | Covers |
|----------|--------|
| `getting-started` | Installation, init, first steps, editor setup |
| `workflow` | Stages, approval, documents, design process, specification |
| `orchestration` | Roles, skills, context assembly, task dispatch, parallel execution |
| `knowledge` | Knowledge entries, confidence, tiers, retrospectives |
| `entities` | Plans, features, tasks, bugs, decisions, lifecycles |
| `git` | Worktrees, branches, merge gates, pull requests |
| `reference` | Tools overview, configuration, schema |

### 4.3 Provisional Topic List

This is an indicative list. The final topics are derived from the documentation once written.

| Topic | Title | Category |
|-------|-------|----------|
| `what-is-kanbanzai` | What Is Kanbanzai? | getting-started |
| `quick-start` | Quick Start Summary | getting-started |
| `editor-setup` | Connecting Your Editor | getting-started |
| `workflow-overview` | The Six-Stage Workflow | workflow |
| `stage-gates` | Stage Gates and Approval | workflow |
| `approval` | Approving Documents and Features | workflow |
| `design-process` | The Design-Led Workflow | workflow |
| `documents` | Document Types and Their Roles | workflow |
| `specifications` | Specifications and What They Mean | workflow |
| `chat-workflow` | Working Through Chat | workflow |
| `roles` | Agent Roles | orchestration |
| `skills` | Skills and Stage Bindings | orchestration |
| `context-assembly` | How Agents Get Context | orchestration |
| `parallel-execution` | Running Agents in Parallel | orchestration |
| `knowledge-entries` | The Knowledge System | knowledge |
| `retrospectives` | Running Retrospectives | knowledge |
| `plans` | Plans | entities |
| `features` | Features and Their Lifecycle | entities |
| `tasks` | Tasks and the Work Queue | entities |
| `bugs` | Bugs and Incidents | entities |
| `decisions` | Decision Records | entities |
| `worktrees` | Worktrees and Isolation | git |
| `merge-gates` | Merge Gates | git |
| `pull-requests` | Pull Request Integration | git |
| `tools-overview` | The MCP Tool Surface | reference |
| `configuration` | Configuration Reference Summary | reference |
| `when-to-use` | When to Use Kanbanzai (and When Not To) | getting-started |

### 4.4 Content Derivation from Documentation

Each help topic is a condensed, self-contained summary derived from a section of the full documentation. The relationship is:

- **The documentation is the source of truth.** Help topics are derived from it, not the other way around.
- **Help topics are summaries, not excerpts.** They are written to stand alone as concise answers, not cut-and-pasted from the docs.
- **Each topic includes a `see_also` pointer** to the relevant documentation section for readers who want the full treatment.
- **Help content is regenerated when documentation changes.** This is a manual editorial step, not an automated pipeline — the condensation requires judgement.

---

## 5. Embedded Content Pipeline

### 5.1 Directory Structure

Help topics live in the source tree alongside the embedded skills:

```
internal/
  kbzhelp/
    help.go              ← embed.FS declaration, topic loader, search
    help_test.go         ← tests
    topics/              ← embedded content
      getting-started/
        what-is-kanbanzai.md
        quick-start.md
        editor-setup.md
        when-to-use.md
      workflow/
        workflow-overview.md
        stage-gates.md
        approval.md
        design-process.md
        documents.md
        specifications.md
        chat-workflow.md
      orchestration/
        roles.md
        skills.md
        context-assembly.md
        parallel-execution.md
      knowledge/
        knowledge-entries.md
        retrospectives.md
      entities/
        plans.md
        features.md
        tasks.md
        bugs.md
        decisions.md
      git/
        worktrees.md
        merge-gates.md
        pull-requests.md
      reference/
        tools-overview.md
        configuration.md
```

### 5.2 Embedding

```
//go:embed topics
var embeddedTopics embed.FS
```

The `kbzhelp` package provides functions to:

1. **Load a topic by ID** — parse the front matter, return structured content
2. **List all topics** — walk the embedded FS, parse front matter, return metadata
3. **Search topics** — tokenise query, match against topic/title/keywords, rank and return

### 5.3 Front Matter Parsing

Topic files use YAML front matter (delimited by `---`). The parser needs to extract: `topic`, `title`, `category`, `keywords`, `related`, `see_also`. The body after the second `---` is the content.

This is a simple parser — not a full YAML library dependency. The front matter fields are flat strings and string arrays. A line-by-line parser similar to the skill `transformSkillContent` pattern is sufficient.

### 5.4 Versioning

Help content is embedded at build time. It is versioned implicitly by the binary version — there is no separate version marker on individual topics. When the binary is updated, the help content updates.

This is simpler than the skill versioning pattern (which needs managed markers and version-aware update logic) because help content is never written to disk and never needs to coexist with user modifications.

---

## 6. The `teacher` Role

### 6.1 Purpose

A context role that shapes the agent's behaviour for interactive teaching and guidance. When a user selects the teacher profile, the agent adopts a patient, explanatory identity optimised for answering questions and guiding exploration.

### 6.2 Role Definition

```
id: teacher
inherits: base
identity: "Patient guide helping a user learn the Kanbanzai workflow system"

vocabulary:
  - "help topic — a concise reference article on a specific Kanbanzai concept"
  - "workflow stage — one of the six stages a feature passes through"
  - "stage gate — an approval checkpoint between stages"
  - "document type — proposal, research, design, specification, or dev plan"
  - "approval — human sign-off that advances work to the next stage"
  - "context assembly — how agents receive scoped information before starting work"
  - "knowledge entry — a persistent operational fact that survives across sessions"

anti_patterns:
  - name: "Assumed Knowledge"
    detect: "Using Kanbanzai-specific terms without explanation on first use"
    because: "The user is learning; unexplained jargon breaks understanding"
    resolve: "Explain each term briefly on first use, or use help(topic) to provide the definition"

  - name: "Information Dump"
    detect: "Responding to a simple question with multiple paragraphs of detail"
    because: "Overwhelms a learner; they asked one thing, not everything"
    resolve: "Answer the specific question concisely, then offer to go deeper"

  - name: "Tool-First Explanation"
    detect: "Explaining a concept by listing MCP tool parameters"
    because: "Concepts should be understood before tool mechanics; parameters are implementation detail"
    resolve: "Explain the concept, show what it looks like in practice, then mention the tool"

  - name: "Doing Instead of Teaching"
    detect: "Performing workflow actions (creating entities, transitioning state) without explaining what is happening and why"
    because: "The user is learning the system, not delegating work; actions without explanation teach nothing"
    resolve: "Explain what you are about to do and why before doing it; after, explain what changed"

tools:
  - help
  - status
  - entity
  - doc
  - knowledge
  - health
  - server_info
  - next
```

### 6.3 Tool Surface

The teacher role has a deliberately limited tool set. It can *read* and *explain* the system state but should not be used for production workflow operations. The key tool is `help` — the teacher's primary source of accurate information.

Tools like `finish`, `handoff`, `decompose`, `merge`, and `worktree` are excluded. The teacher explains how these work; it does not perform them. If the user wants to execute workflow operations, they should switch to an appropriate role (orchestrator, implementer, etc.).

The exception is `entity` and `next` — the teacher can demonstrate entity creation and queue inspection as part of guided learning. These are low-risk, reversible operations useful for teaching.

### 6.4 Shipping

The teacher role is installed by `kanbanzai init` as a managed role file at `.kbz/roles/teacher.yaml`, using the same managed-marker versioning pattern as other init-managed roles. It is available immediately after project initialisation.

---

## 7. The `teach-kanbanzai` Skill

### 7.1 Purpose

A skill that defines the procedure for interactive teaching sessions. It tells the agent *how* to teach — when to use the `help` tool, how to structure explanations, when to show examples, when to link to documentation.

### 7.2 Procedure Summary

1. **Understand the question.** Identify what the user is asking about. If ambiguous, ask a clarifying question.
2. **Look it up.** Use `help(topic)` or `help(search)` to find the relevant content. Do not rely on training data for Kanbanzai-specific answers — the help content is the authoritative source.
3. **Answer concisely.** Give a direct answer to the question. Use the inverted pyramid: the key point first, then supporting detail.
4. **Show, don't tell.** Where possible, demonstrate with a concrete example — a tool call, a chat interaction, a workflow step.
5. **Offer depth.** After answering, offer to go deeper or to explain related concepts. Mention the `see_also` documentation reference for the full treatment.
6. **Stay in scope.** The teacher explains Kanbanzai. It does not perform project work, write code, or make design decisions. If the user asks for something outside teaching scope, suggest the appropriate role.

### 7.3 Stage Binding

The teacher skill does not bind to a standard workflow stage. It is available at any time — it is not part of the design → specify → implement pipeline. It binds to a special `learning` context that exists outside the stage-gate model.

This means the teacher role is selected explicitly by the user (via profile selection in the editor), not invoked automatically by the stage binding system.

### 7.4 Shipping

The skill is embedded in the binary and installed by `kanbanzai init` at `.agents/skills/kanbanzai-teaching/SKILL.md`, following the same pattern as other embedded skills.

---

## 8. Default Profile for New Projects

### 8.1 Rationale

A new user running `kanbanzai init` for the first time is in learning mode. They do not yet know what roles are available or which to select. If the default profile is `base` or `orchestrator`, they get an agent that assumes they know the system.

### 8.2 Proposal

Consider setting the teacher profile as the default for new projects — either by configuring it as the initial active profile, or by having the agent suggest switching to the teacher profile when it detects a new (empty) project with no entities, no documents, and no knowledge entries.

This is a recommendation, not a hard requirement. The exact mechanism depends on how profile selection works in each editor. The important thing is that a new user encounters a helpful, teaching-oriented agent early in their experience.

### 8.3 Decision Deferred

The mechanism for default profile selection is editor-specific and may require changes outside the Kanbanzai codebase. This decision is deferred to implementation. The teacher role and help tool are valuable regardless of whether the default-profile behaviour is implemented.

---

## 9. Implementation Scope

### 9.1 What Is New Code

| Component | Location | Estimate |
|-----------|----------|----------|
| `help` tool | `internal/mcp/help_tool.go` | Small — topic lookup, list, search |
| `kbzhelp` package | `internal/kbzhelp/` | Small — embed.FS, front matter parser, search |
| Tool group registration | `internal/mcp/server.go`, `internal/mcp/groups.go`, `internal/config/config.go` | Trivial — add to core group |
| Teacher role YAML | `internal/kbzinit/roles/teacher.yaml` | Content only |
| Teaching skill | `internal/kbzinit/skills/teaching/SKILL.md` | Content only |
| Help topic files | `internal/kbzhelp/topics/**/*.md` | Content — depends on documentation |

### 9.2 What Is Content

The help topic files are the largest piece of work, and they depend on the documentation being written first. The tool, package, role, and skill can all be built and tested with placeholder content before the final topics exist.

### 9.3 What Is Not In Scope

- **Automated content extraction from docs.** Help topics are manually written summaries, not auto-generated. Automation could be explored later but adds complexity without clear benefit at this scale.
- **Interactive tutorials.** The teacher explains and demonstrates; it does not run scripted multi-step tutorials with state tracking. If interactive tutorials are wanted later, that is a separate feature.
- **Content served from the network.** All help content is embedded in the binary. No network dependency, no fetching from GitHub, no web service.
- **User-contributed help topics.** The help corpus is authoritative and versioned with the binary. Users who want to add project-specific guidance should use the knowledge system.

---

## 10. Relationship to Documentation

| Concern | Owner |
|---------|-------|
| Full documentation (manual, guides, references) | Documentation design (`public-release-documentation.md`) |
| Help topic content | Derived from documentation, lives in `internal/kbzhelp/topics/` |
| `help` tool implementation | This design |
| Teacher role and skill | This design |
| Documentation editorial principles | Documentation design |
| Help content editorial principles | This design (§4), informed by documentation design (§3) |

The help system does not duplicate the documentation. Each help topic is a condensed summary that points to the full documentation for depth. The documentation does not need to know about the help system — it just produces good documentation. The help system references the documentation design and specifies how it consumes the output.

---

## 11. Sequencing

| Step | Depends on | Can parallelise with |
|------|------------|----------------------|
| 1. Help system design (this document) | Documentation design | — |
| 2. `help` tool + `kbzhelp` package | This design approved | Step 3, step 4, documentation production |
| 3. Teacher role + skill | This design approved | Step 2, documentation production |
| 4. Placeholder help topics (for testing) | Step 2 | Step 3, documentation production |
| 5. Final help topic content | Documentation production complete | — |
| 6. Integration testing | Steps 2–5 complete | — |

Steps 2, 3, and 4 can all run in parallel with documentation production. Step 5 is the only step that must wait for the documentation to be finished.

---

## 12. Decisions

### 12.1 Content Is Embedded in the Binary

Help content is compiled into the `kanbanzai` binary via `embed.FS`. It is not stored in `.kbz/`, not shipped as separate files, and not fetched from the network. This gives zero-cost availability (no files to manage, no network dependency), automatic versioning (upgrades update content), and offline operation.

### 12.2 The `help` Tool Is in the Core Group

Help is most needed by new users, who are most likely running a minimal configuration. Placing `help` in the core group — which cannot be disabled — ensures it is always available.

### 12.3 Help Topics Are Manually Authored Summaries

Topics are written by hand, not auto-extracted from documentation. This ensures each topic is self-contained, concise, and conversational — qualities that mechanical extraction does not reliably produce. The cost is a manual editorial step when documentation changes; the benefit is quality.

### 12.4 The Teacher Role Does Not Perform Workflow Actions

The teacher explains the system; it does not operate it. Workflow operations (creating entities, approving documents, merging branches) are performed by the appropriate workflow roles. This prevents the teacher from accidentally modifying project state during a learning session. The exception is low-risk, demonstrative actions like entity creation and queue inspection.

### 12.5 Search Is Simple Token Matching

The search implementation is keyword-based token matching over a small corpus. Full-text search, semantic search, or embedding-based retrieval are not needed for 25–30 topics and would add unnecessary complexity.

### 12.6 No Preloaded Knowledge Entries

The help system does not preload knowledge entries. The knowledge system is for project-specific operational facts, not system documentation. Mixing the two would pollute every context assembly with reference material and blur the distinction between "what this project has learned" and "how Kanbanzai works."

---

## 13. Open Questions

### 13.1 Exact Topic List

The provisional topic list in §4.3 is indicative. The final list depends on the documentation structure and will be determined during the content population step. Topics may be added, removed, or restructured based on what the documentation covers.

### 13.2 Default Profile Mechanism

§8 proposes the teacher as the default profile for new projects but defers the mechanism. This depends on editor-specific profile selection behaviour and may require investigation during implementation.

### 13.3 Help Content Maintenance Process

When the documentation is updated, the corresponding help topics need updating too. There is no automated check for this. A manual review step ("update help topics") should be added to the documentation update process. Whether to add a health check that detects stale help content (by comparing binary version to doc modification dates) is deferred to implementation.

### 13.4 Skill Name in Init Installer

The embedded skills list in `internal/kbzinit/skills.go` uses a fixed `skillNames` slice. Adding `"teaching"` to this list is straightforward but needs to be coordinated with the init command's version-aware update logic.

---

## 14. Summary

The interactive help system adds three components to Kanbanzai:

1. **A `help` MCP tool** — serves embedded documentation topics on demand, with browse and search. Zero token cost on normal operations; content loaded only when explicitly requested.
2. **A `teacher` role** — shapes agent behaviour for patient, explanatory interaction with learners. Limited tool surface focused on reading and explaining, not performing workflow actions.
3. **A `teach-kanbanzai` skill** — defines the procedure for effective teaching: look it up, answer concisely, show by example, offer depth, stay in scope.

Content is embedded in the binary, versioned with releases, and derived from the public documentation. The machinery (tool, role, skill) can be built and tested before the documentation is finished. The content is populated afterwards.