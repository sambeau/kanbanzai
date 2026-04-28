# Kanbanzai Evaluation Suite

This directory contains representative workflow scenarios for evaluating Kanbanzai workflow behaviour with live LLM agents.

## Purpose

Test whether agents correctly follow the intended orchestration patterns: navigating stage gates, recovering from prerequisite failures, cycling through review-rework loops, and coordinating multiple concurrent features.

## Requirements

- Kanbanzai MCP server running (`kanbanzai serve`)
- A compatible LLM agent session (Claude, GPT-4, etc.) connected to the server
- A clean `.kbz/` instance with `kanbanzai init` run

No external dependencies, databases, or network services are required beyond the MCP server itself.

## How to Run

1. Start the kanbanzai server in your project directory:
   ```
   kanbanzai serve
   ```

2. Open an LLM agent session with the MCP server configured.

3. Load a scenario YAML file and direct the agent to fulfill the `name` / `description`, starting from `starting_state`.

4. Observe the tool call sequence and compare it against `expected_pattern.tool_sequence`.

5. Verify all entries in `success_criteria` are satisfied.

## Interpreting Results

| Outcome | Meaning |
|---------|---------|
| **Pass** | All `success_criteria` are satisfied; tool sequence matches `expected_pattern` closely |
| **Partial** | Some criteria met; agent deviated from expected pattern but reached a valid end state |
| **Fail** | Core criteria not met; a required gate was bypassed, workflow stalled, or an incorrect transition was made |

The `expected_pattern` is indicative, not prescriptive — minor reorderings that still satisfy all criteria should be considered a pass.

## Scenario Categories

| Category | Description |
|----------|-------------|
| `happy-path` | No errors; agent navigates all stages and satisfies all gates cleanly |
| `gate-failure-recovery` | Agent encounters a missing prerequisite and correctly recovers |
| `review-rework-loop` | Feature returns from reviewing to developing for rework |
| `multi-feature-orchestration` | Agent juggles two or more features with dependencies or conflicts |
| `edge-case` | Unusual inputs, empty states, or boundary conditions |

## Maintenance Discipline

When workflow changes are made — gates added or removed, tool descriptions changed, stage bindings modified, new document types required — update the affected scenarios in the **same commit**. This keeps the scenario history aligned with the workflow history in git and makes regressions easy to bisect.
