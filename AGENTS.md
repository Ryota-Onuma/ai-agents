# Agents Guide

A concise, practical reference for designing, configuring, and operating agents in this repository. Use it as a starting point and adapt to the project's needs.

---

## TL;DR

- Define the agent's purpose, inputs, outputs, and success metrics before writing code
- Keep prompts short, explicit, and testable; prefer few-shot over long narratives
- Give agents the minimum toolset needed and add more only when required
- Log everything (inputs, decisions, tool calls, outputs) for observability and evaluation
- Write contract tests around prompts, tools, and plans; simulate edge cases
- Guardrails first: rate limits, allow-lists, sandboxing, cost budgets, and kill switches

---

## What is an Agent?

An agent is an autonomous or semi-autonomous system that:

- Maintains context (memory/state)
- Plans actions based on goals and environment feedback
- Uses tools (APIs, code, files, browsers) to act in the world
- Produces outputs aligned with explicit policies and constraints

Core components:

- Objective and policies (what to do; what not to do)
- Reasoning and planning (decompose, prioritize, and schedule)
- Tool use (invoke capabilities with safe contracts)
- Memory (short-term scratchpad + optional long-term store)
- Observation and feedback (loop until done or budget exhaustion)

---

## Recommended Repository Layout

This is a recommended structure. Adjust as needed.

```
agents/
  <agent-name>/
    prompt/
      system.md
      examples.md
    config/
      agent.yaml
    src/
      planner.py|ts
      executor.py|ts
      tools/...
    tests/
      test_agent.py|ts
memory/
  <agent-name>/
      vectorstore/...
      episodes/...
telemetry/
  logs/
  traces/
configs/
  models.yaml
  tools.yaml
scripts/
  evals/
```

---

## Design a New Agent

1) Purpose and Contract
- Goal statement: single-sentence description
- Inputs and outputs: schemas or examples
- Success criteria: acceptance tests and metrics
- Operating constraints: time, cost, safety, data scope

2) Capabilities and Tools
- Enumerate necessary tools only; define clear input/output contracts
- Add guardrails: allow-lists, argument validation, rate limits, timeouts

3) Reasoning and Planning
- Choose planner style: reactive (one-shot), iterative (reflect/act), or hierarchical
- Keep loops bounded: max steps, max cost, max duration

4) Memory Strategy
- Short-term: scratchpad/state passed per step
- Long-term (optional): vector store, file cache, or database
- Retrieval policy: what to store, when to retrieve, eviction

5) Prompts and Policies
- Provide a terse, testable system prompt
- Add few-shot examples for edge cases
- Encode policies: redlines, PII handling, refusal conditions

6) Observability and Safety
- Log inputs, plans, tool calls, outputs, costs
- Add circuit breakers, kill switch, and budget enforcement

7) Tests and Evaluation
- Unit tests for tools and planners
- Golden tests for prompts with fixtures
- E2E evals with scenarios and assertions

---

## Minimal Agent Config (Example)

```yaml
# configs/agents/researcher.yaml
name: researcher
model: gpt-4o-mini
system: |
  You are a focused research assistant.
  - Cite sources.
  - Prefer primary documentation.
  - Output concise bullet points and links.
policies:
  max_steps: 6
  max_seconds: 120
  max_cost_usd: 0.20
  allow_network: true
  disallow_file_write: true
memory:
  short_term: scratchpad
  long_term:
    kind: vector
    path: memory/researcher/vectorstore
    top_k: 6
planner:
  kind: iterative
  reflect_every_n_steps: 2
  stop_conditions:
    - goal_satisfied
    - budget_exhausted
    - no_progress
tools:
  - id: web_search
    type: http
    endpoint: https://api.example.com/search
    timeout_ms: 10000
    rate_limit_rps: 2
    allowlist_domains:
      - docs.python.org
      - developer.mozilla.org
  - id: code_reader
    type: filesystem_read
    roots:
      - src
      - packages
logging:
  level: info
  persist:
    traces: telemetry/traces
    logs: telemetry/logs
```

---

## Prompt Files (Suggested)

- `prompt/system.md`: role, policies, output contract
- `prompt/examples.md`: few-shot examples for success and failure cases

Example `prompt/system.md`:

```md
You are the Researcher agent.
- Always answer with concise bullet points.
- Cite sources with markdown links.
- If you cannot verify a claim, say so explicitly.
- Do not execute any code.
```

---

## Planner Patterns

- Reactive: single step from instruction to output (fast, cheap)
- Iterative: think/act/observe loops with bounded steps (balanced)
- Hierarchical: top-level planner delegates to sub-agents (complex tasks)

Guidelines:
- Separate planning from acting; make state explicit
- Encode stop conditions clearly
- Prefer deterministic steps where possible

---

## Tooling Guidelines

- Small, composable tools with typed inputs/outputs
- Validate arguments and sanitize outputs
- Timeouts, retries with backoff, circuit breakers
- Strict allow-lists for network and filesystem
- Never return raw secrets from tools

---

## Memory Guidelines

- Store only what improves future steps; avoid dumping full transcripts
- Normalize entries: who, what, when, why useful
- Add TTLs and size caps; periodically prune
- Log retrieval hits/misses for evaluation

---

## Testing and Evaluation

- Unit tests: tool contracts, planner branches, error handling
- Golden tests: prompt outputs against fixtures (allow minor diffs)
- E2E scenarios: realistic tasks with budgets and assertions
- Regression suite: add failures as tests to prevent recurrence

Example test skeleton:

```python
# agents/researcher/tests/test_researcher.py
import pytest

from agents.researcher.src.planner import plan_step


def test_plan_stops_on_budget():
    state = {"steps": 5, "max_steps": 5}
    decision = plan_step(state)
    assert decision["type"] == "stop"
```

---

## Safety and Governance

- Redlines: disallowed actions/content (PII, credentials, self-modification)
- Budgets: time/cost/step limits; enforce at runtime
- Approvals: manual check for high-risk tools or actions
- Audit logs: immutable records of actions and rationale

---

## Operational Runbook

- Version agents and prompts; keep changelogs
- Telemetry dashboards: latency, cost, success rate, tool error rate
- Incident response: kill switch, rollback, disable tools
- Rollouts: canary new prompts/configs before full release

---

## Adding a New Agent (Checklist)

- [ ] Create `agents/<name>/prompt/system.md` and `examples.md`
- [ ] Create `agents/<name>/config/agent.yaml`
- [ ] Implement planner and executor in `agents/<name>/src/`
- [ ] Implement tools with validation and allow-lists
- [ ] Add tests in `agents/<name>/tests/`
- [ ] Wire up telemetry (logs/traces) and budgets
- [ ] Add E2E eval scenarios
- [ ] Document the agent here under Catalog

---

## Catalog (Fill Me In)

Document existing agents here as they are added.

- Name: <agent-name>
  - Purpose: <what it does>
  - Inputs: <schemas/examples>
  - Outputs: <schemas/examples>
  - Tools: <ids>
  - Policies: <limits, guardrails>
  - Owner: <team/person>

---

## Troubleshooting

- Agent loops without progress: lower max steps; add explicit stop conditions; tighten success criteria
- Tool flakiness: add retries with backoff; improve argument validation; increase timeouts
- Hallucinations: shorten prompts; add references; strengthen refusal policy; retrieve memory first
- Cost overruns: lower model, reduce steps, trim context, cache tool results
- Non-determinism: reduce temperature; add more explicit examples and checks

---

## Changelog

Maintain notable changes to agent behaviors and configs here.

- 2025-08-08: Initial guide created.
