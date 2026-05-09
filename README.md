# Kanbanzai

## Introduction

Kanbanzai is a workflow system (and tool) for small teams (or single developers) to work with AI agents to create large software projects. It is an MCP Server that provides tools to your AI Agents to manage your project.

Kanbanzai manages the process of creating software using a ***design-first***, ***spec-driven*** development process. It is part ***project management*** tool (it manages, documents, plans, features, tasks etc), part ***orchestration*** tool (it manages parallel teams of agents, roles, skills, sub-agents etc.)

You play the role of product/design manager, the AI Agent plays the role of the rest of the team: senior designer, development manager, developers, testers, etc. Your main role is to create designs that get turned into clear specs; the AI's role is to take the specs and accurately turn them into software.

You communicate with it using chat and markdown documents: proposals, design, specification, implementation plan, review etc. These documents act as the project's main source of persistent memory for the team. The AI dev manager also tracks the process of all plans, features and tasks using the tools in the MCP server. Like Jira or any other project management software there are approval steps ('gates') and state management (designing, waiting for approval, … done).

However, you don't have to deal with the tool: you are the manager. The AI deals with the tool as they are the dev team. You simply write proposals, discuss them, collaborate on creating a design and once you are happy, the AI Agent takes over creating a spec, a development plan and then orchestrates its development using a team of sub-agents. As the work progresses through the system, the AI project manager uses the tool to keep the project state up-to-date.

The orchestration, roles and skills are built upon the latest academic research, first introduced to me by JD Forsythe in his incredibly useful [10 CLAUDE CODE PRINCIPLES: What the Research Actually Says](https://jdforsythe.github.io/10-principles/) research.

## What is Kanbanzai?

Kanbanzai is a workflow system that gives AI agents structure — so they can pick up where the last session left off, coordinate without stepping on each other, and hand back to you when they need a decision.

You write designs and make decisions. Agents handle implementation: decomposing work, claiming tasks, reviewing output, and tracking what they learnt. Everything lives in plain YAML files committed to your Git repository — nothing hidden in a database.

## What problems does it solve?

**If you work with AI agents on code**, you have probably hit these:

- **Context vanishes between sessions.** An agent does good work, then the next session starts from scratch. Nobody remembers what was decided, what was tried, or what failed.
- **Parallel agents collide.** Two agents edit the same file. One wins; the other's work disappears. You find out later.
- **Knowledge is rediscovered, not retained.** Every session re-learns the same conventions, the same architectural constraints, the same "don't touch that module" warnings.
- **You become a full-time project manager.** Instead of designing, you spend your time tracking what is done, what is blocked, and what to do next.

**If you care about design quality and specification-led development**, you have probably hit these:

- **Implementation outruns design.** Agents start coding before anyone has agreed on the approach. Rework follows.
- **There is no approval gate.** Work proceeds without explicit sign-off. You review after the fact, when changing course is expensive.
- **Process is invisible.** You cannot see which features are designed, which are specified, and which are in progress — not without manually checking files.

Kanbanzai solves both sets of problems with a stage-gate workflow delivered as MCP (Model Context Protocol) tools that agents call directly.

## What does using it look like?

A typical feature flows like this:

> **You:** "Here is a proposal for an authentication system. Please give me some feedback."
>
> **Agent:** Provides feedback, offers questions.
> 
> **You:** Answer the questions in the document or directly in the chat. "I have answered your questions. Please create a design document."
>
> **Agent:** Creates the feature, writes a design document, and asks you to review it.
>
> **You:** Read the design. "Approved — but use OAuth, not custom tokens."
>
> **Agent:** Revises, writes the specification, decomposes it into tasks, and starts implementing. Two sub-agents work in parallel on separate worktrees. A third reviews their output.
>
> **You:** Get a checkpoint: "The OAuth library doesn't support refresh tokens out of the box. Custom wrapper or different library?" You decide. Work resumes.
>
> **Agent:** All tasks pass review. A pull request is ready for your final sign-off.

You make the design calls and approve at gates. Agents handle the mechanics between gates.

## How the workflow fits together

Every feature progresses through a stage-gate lifecycle. Gates enforce prerequisites — approved documents, completed tasks, or registered reports — before work advances to the next stage.

<img width="505" height="533" alt="Kanbanzai Workflow Diagram" align="center" src="https://github.com/user-attachments/assets/ea1937f4-ee0f-4c2c-87de-0ab655bcc86c" />

Agents cannot skip stages. A feature in **designing** cannot move to **developing** without an approved specification. This is deliberate — catching errors at design time costs less than catching them during implementation.

## Key capabilities

**Workflow and lifecycle**
- Stage-gate progression with document-based approval at each gate
- Plans group features; features decompose into dependency-aware tasks
- Backward transitions when rework is needed — nothing is a one-way door except done

**Agent coordination**
- Work queue sorted by estimate and age — agents claim the next ready task
- Conflict analysis flags file overlap before parallel tasks start
- Role and skill profiles shape agent behaviour per workflow stage
- Context assembly packs relevant spec sections, knowledge, and conventions into each task handoff

**Knowledge persistence**
- Knowledge entries survive across sessions and earn confidence as agents confirm them
- Retrospective signals capture process observations — what worked, what caused friction
- Synthesis clusters signals into actionable themes

**Git integration**
- Isolated worktrees for parallel feature development
- Merge gates check task completion, verification, conflicts, and branch health before merging
- Creates pull requests and manages branch lifecycles

**Human control**
- Checkpoints pause work and wait for your decision — agents do not guess
- Document approval is always a human action
- Everything is plain YAML in Git — you can read, diff, and review it in a pull request

## When to use it — and when not to

**Kanbanzai earns its keep when:**
- Features regularly take more than one session to implement
- Multiple AI agents work on the same codebase
- Design decisions need to persist and be reviewable
- You care about specification-led quality — catching errors at design time rather than during implementation

**Think twice when:**
- You are building a weekend project or prototype — the process overhead exceeds the work itself
- Every feature is simple and self-contained, done in a single session with no coordination needed
- You prefer to work without structured process

**The honest cost.** Kanbanzai adds process, time, and token overhead. On a small project, that overhead exceeds the savings. On a larger project — multiple concurrent features, complex architecture, long-running work — the overhead is repaid through reduced rework, persistent knowledge, and coordinated parallel execution. The crossover point is roughly when features span multiple sessions or when you have more than one active work stream.

**The investment.** Expect to spend one to two hours learning the workflow before it feels natural. The first feature will feel slow. The fifth will not.

## Quickstart

**Requirements:** Go 1.25+, Git, and an MCP-capable editor (Zed recommended)

```sh
# Install
go install github.com/sambeau/kanbanzai/cmd/kbz@latest

# Initialise in your project root
cd your-project
kbz init

# Verify
kbz health
```

`kbz init` creates the `.kbz/` directory, installs default roles and skills, and generates editor configuration. Open the project in your editor — the MCP server should connect automatically.

For GitHub integration, editor profiles, and a guided walkthrough of your first feature, see the [Getting Started guide](docs/getting-started.md).

## What gets stored

Kanbanzai stores all state in a `.kbz/` directory at your project root, committed to Git alongside your code:

```
.kbz/
├── config.yaml            ← project settings
├── local.yaml             ← machine-local settings (not committed)
├── stage-bindings.yaml    ← maps workflow stages to roles and skills
├── roles/                 ← role definitions (identity, vocabulary, constraints)
├── skills/                ← skill procedures (checklists, evaluation criteria)
├── state/
│   ├── plans/             ← plans
│   ├── features/          ← features
│   ├── tasks/             ← tasks
│   ├── bugs/              ← bugs
│   ├── checkpoints/       ← human decision points
│   ├── decisions/         ← architectural decisions
│   ├── documents/         ← document metadata
│   ├── knowledge/         ← knowledge entries
│   └── worktrees/         ← worktree records
└── index/                 ← document intelligence index (derived)
```

Every file is plain YAML. You can read it, search it with grep, and review it in a pull request like any other file in your repository.

## Stages and roles

This table maps every workflow stage to its bound roles, skills, and gate type.
It is generated from `.kbz/stage-bindings.yaml` — the single source of truth.

<!-- registry-gen:begin:roles-and-skills source=.kbz/stage-bindings.yaml -->
> **Generated** — canonical source: `.kbz/stage-bindings.yaml`. Do not hand-edit this section; run `make registry-sync` to update.

| Stage | Description | Roles | Skills | Gate | Doc Type |
|-------|-------------|-------|--------|------|----------|
| designing | Creating or revising a design document | `architect` | [write-design](.kbz/skills/write-design/SKILL.md) | auto | design |
| specifying | Writing a formal specification with acceptance criteria | `spec-author` | [write-spec](.kbz/skills/write-spec/SKILL.md) | human | specification |
| dev-planning | Breaking a spec into an implementation plan and tasks | `architect` | [write-dev-plan](.kbz/skills/write-dev-plan/SKILL.md), [decompose-feature](.kbz/skills/decompose-feature/SKILL.md) | human | dev-plan |
| developing | Implementing tasks from the dev plan | `orchestrator` | [orchestrate-development](.kbz/skills/orchestrate-development/SKILL.md) | auto | — |
| reviewing | Evaluating implementation against the specification | `orchestrator` | [orchestrate-review](.kbz/skills/orchestrate-review/SKILL.md) | human | report |
| merging | Merging the feature branch into main after review approval | `orchestrator` | [orchestrate-review](.kbz/skills/orchestrate-review/SKILL.md) | auto | — |
| verifying | Delegating close-out verification of the Definition of Done to a clean-context verifier sub-agent | `orchestrator` | [orchestrate-review](.kbz/skills/orchestrate-review/SKILL.md) | auto | — |
| batch-reviewing | Reviewing a completed batch for aggregate delivery | `reviewer-conformance` | [review-plan](.kbz/skills/review-plan/SKILL.md) | human | report |
| researching | Producing a research report or analysis | `researcher` | [write-research](.kbz/skills/write-research/SKILL.md) | auto | research |
| documenting | Updating project documentation for currency | `documenter` | [update-docs](.kbz/skills/update-docs/SKILL.md) | auto | — |
| doc-publishing | Running a document through the five-stage editorial pipeline | `doc-pipeline-orchestrator` | [orchestrate-doc-pipeline](.kbz/skills/orchestrate-doc-pipeline/SKILL.md) | auto | — |
| retro-fixing | Implementing a fix for a retrospective theme | — | — | auto | — |
<!-- registry-gen:end:roles-and-skills -->

### Role index

Each role defines identity, vocabulary, and anti-patterns for a specific workflow
stage. Roles can inherit from a parent role (e.g. `reviewer-security` inherits from
`reviewer`).

<!-- registry-gen:begin:role-index source=.kbz/roles/*.yaml -->
> **Generated** — canonical source: `.kbz/roles/*.yaml`. Do not hand-edit this section; run `make registry-sync` to update.

| Role | Identity | Inherits | Source |
|------|----------|----------|--------|
| `architect` | Senior software architect | `base` | `.kbz/roles/architect.yaml` |
| `base` | Software development agent | — | `.kbz/roles/base.yaml` |
| `doc-checker` | Technical fact-checker | `base` | `.kbz/roles/doc-checker.yaml` |
| `doc-copyeditor` | Senior copy editor | `base` | `.kbz/roles/doc-copyeditor.yaml` |
| `doc-editor` | Developmental editor | `base` | `.kbz/roles/doc-editor.yaml` |
| `doc-pipeline-orchestrator` | AI content editor | `base` | `.kbz/roles/doc-pipeline-orchestrator.yaml` |
| `doc-stylist` | AI prose editor | `base` | `.kbz/roles/doc-stylist.yaml` |
| `documenter` | Senior technical writer | `base` | `.kbz/roles/documenter.yaml` |
| `implementer` | Senior software engineer | `base` | `.kbz/roles/implementer.yaml` |
| `implementer-go` | Senior Go engineer | `implementer` | `.kbz/roles/implementer-go.yaml` |
| `orchestrator` | Senior engineering manager coordinating an agent team | `base` | `.kbz/roles/orchestrator.yaml` |
| `plan-validator` | Senior implementation plan auditor. Verify that dev-plans are complete, well-decomposed, and fully traceable to their parent specification. Do not evaluate whether task ordering is *optimal* — only whether it is *valid* and *complete*. | `base` | `.kbz/roles/plan-validator.yaml` |
| `researcher` | Senior technical analyst | `base` | `.kbz/roles/researcher.yaml` |
| `review-gate-validator` | Senior review quality auditor. Verify that a completed review is thorough, evidence-backed, and suitable for auto-approval. Do not re-review the code — audit the review process itself. | `reviewer` | `.kbz/roles/review-gate-validator.yaml` |
| `reviewer` | Senior code reviewer | `base` | `.kbz/roles/reviewer.yaml` |
| `reviewer-conformance` | Senior requirements verification engineer | `reviewer` | `.kbz/roles/reviewer-conformance.yaml` |
| `reviewer-quality` | Senior software quality engineer | `reviewer` | `.kbz/roles/reviewer-quality.yaml` |
| `reviewer-security` | Senior application security engineer | `reviewer` | `.kbz/roles/reviewer-security.yaml` |
| `reviewer-testing` | Senior test engineer | `reviewer` | `.kbz/roles/reviewer-testing.yaml` |
| `spec-author` | Senior requirements engineer | `base` | `.kbz/roles/spec-author.yaml` |
| `spec-validator` | Senior requirements quality auditor. Verify that specifications are complete, testable, and traceable to their parent design. Do not evaluate whether requirements are *correct* — only whether they are *well-formed* and *complete*. | `base` | `.kbz/roles/spec-validator.yaml` |
| `verifier` | Methodical close-out auditor | `base` | `.kbz/roles/verifier.yaml` |
<!-- registry-gen:end:role-index -->

## Further reading

| Document | What it covers |
|----------|---------------|
| [Getting Started](docs/getting-started.md) | Install, configure, and deliver your first feature end to end |
| [User Guide](docs/user-guide.md) | Conceptual overview — how all the pieces fit together |
| [Workflow Overview](docs/workflow-overview.md) | The stage-gate methodology in depth |
| [Orchestration and Knowledge](docs/orchestration-and-knowledge.md) | Agent coordination, context assembly, and the knowledge system |
| [Retrospectives](docs/retrospectives.md) | Recording process signals and synthesising actionable themes |
| [MCP Tool Reference](docs/mcp-tool-reference.md) | All 22 MCP tools with parameters, returns, and examples |
| [Schema Reference](docs/schema-reference.md) | Entity types, field definitions, and lifecycle state machines |
| [Configuration Reference](docs/configuration-reference.md) | Every configuration option with defaults and examples |
| [Viewer Agents Guide](docs/kanbanzai-guide-for-viewer-agents.md) | Building read-only integrations against `.kbz/` state |
| [AGENTS.md](AGENTS.md) | Instructions for AI agents working on this project |

## Licence

MIT
