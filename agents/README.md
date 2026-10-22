# Agent workflows

Agent directories describe how a particular coding agent could use the shared
core and existing integration contracts. They must link to integration-owned
domain specifications instead of copying tool rules into every agent folder.

This repository provides a working local MCP server but never registers it with
an agent automatically. Current substantive workflow:

- [Codex](codex/README.md) — use of integration contracts in authoring and
  implementation validation loops, plus a reusable
  [`AGENTS.md` template](codex/AGENTS.template.md) for consuming repositories.

Future agent workflows should be added only when their setup or interaction
model materially differs and can be documented accurately.

The shared behavior and MCP surface live in [DESIGN.md](../DESIGN.md).
