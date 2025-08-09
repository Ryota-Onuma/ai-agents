# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is an AI agents repository focused on designing, configuring, and operating autonomous/semi-autonomous systems. The repository currently contains documentation and guidelines for agent development but no implementation code yet.

## Repository Structure

- `AGENTS.md` - Comprehensive guide for agent design, configuration, and operation
- Repository follows a recommended structure for organizing agents, memory, telemetry, and configurations

## Key Concepts from AGENTS.md

- **Agent Definition**: Autonomous systems that maintain context, plan actions, use tools, and produce constrained outputs
- **Core Components**: Objective/policies, reasoning/planning, tool use, memory, observation/feedback
- **Architecture Patterns**: Reactive (one-shot), iterative (think/act/observe), hierarchical (delegated sub-agents)

## Recommended Development Approach

When working with agents in this repository:

1. **Design First**: Define purpose, inputs/outputs, success metrics, and constraints before coding
2. **Minimal Toolsets**: Start with essential tools only, add more as needed
3. **Safety First**: Implement guardrails (rate limits, allow-lists, budgets, kill switches)
4. **Observability**: Log all inputs, decisions, tool calls, and outputs
5. **Testing**: Write contract tests for prompts, tools, and plans

## Agent Development Structure

Follow the recommended layout from AGENTS.md:
- `agents/<name>/prompt/` - System prompts and examples
- `agents/<name>/config/` - Agent configuration (YAML)
- `agents/<name>/src/` - Implementation (planner, executor, tools)
- `agents/<name>/tests/` - Unit and integration tests
- `memory/<name>/` - Vector stores and episode storage
- `telemetry/` - Logs and traces
- `configs/` - Model and tool configurations

## Development Guidelines

- Keep prompts short, explicit, and testable
- Prefer few-shot examples over long narratives
- Implement proper memory strategies (short-term scratchpad, optional long-term storage)
- Use bounded loops with max steps, cost, and duration limits
- Version agents and prompts with changelogs
- Include safety redlines for PII, credentials, and self-modification

## Testing Strategy

- Unit tests for tool contracts and planner logic
- Golden tests for prompt outputs against fixtures
- E2E evaluation scenarios with realistic tasks and assertions
- Regression suite for documented failures