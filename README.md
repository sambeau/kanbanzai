# Kanbanzai

## Introduction

Kanbanzai is a workflow system (and tool) for small teams (or single developers) to work with AI agents to create large software projects. It is an MCP Server that provides tools to your AI Agents to manage your project.

Kanbanzai manages the process of creating software using a ***design-first***, ***spec-driven*** development process. It is part ***project management*** tool (it manages, documents, plans, features, tasks etc), part ***orchestration*** tool (it manages parallel teams of agents, roles, skills, sub-agents etc.)

You play the role of product/design manager, the AI Agent plays the role of the rest of the team: senior designer, development manager, developers, testers, etc. Your main role is to create designs that get turned into clear specs; the AI's role is to take the specs and accurately turn them into software.

You communicate with it using chat and markdown documents: proposals, design, specification, implementation plan, review etc. These documents act as the project's main source of persistent memory for the team. The AI dev manager also tracks the process of all plans, features and tasks using the tools in the MCP server. Like Jira or any other project management software there are approval steps (‘gates’) and state management (designing, waiting for approval, … done).

However, you don’t have to deal with the tool: you are the manager. The AI deals with the tool as they are the dev team. You simply write proposals, discuss them, collaborate on creating a design and once you are happy, the AI Agent takes over creating a spec, a development plan and then orchestrates its development using a team of sub-agents. As the work progresses through the system, the AI project manager uses the tool to keep the project state up-to-date.

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
go install github.com/sambeau/kanbanzai/cmd/kanbanzai@latest

# Initialise in your project root
cd your-project
kanbanzai init

# Verify
kanbanzai health
```

`kanbanzai init` creates the `.kbz/` directory, installs default roles and skills, and generates editor configuration. Open the project in your editor — the MCP server should connect automatically.

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
