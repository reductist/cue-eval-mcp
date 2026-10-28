# cue-eval-mcp

`cue-eval-mcp` provides contract-first CUE validation for coding agents and CI.
When a task can be usefully modeled, people and agents can define reusable
domain schemas and change constraints, then evaluate actual data or generated
artifacts against those contracts during implementation.

Start with the fictional [Puppet getting-started walkthrough](docs/getting-started.md)
to see the intended user-and-agent workflow from initial request through
concrete validation.

The current version includes:

- a Go evaluator shared by the CLI and MCP server;
- read-only `discover_contracts`, `inspect_contract`, and `evaluate` MCP tools;
- separate author and implementation modes with structured diagnostics;
- strict JSON/YAML implementation evidence that cannot be completed by schema
  defaults, constants, or derivations;
- bounded, workspace-scoped file access and exact input/contract digests;
- synthetic plugin and Puppet examples; and
- native, OCI container, and Dev Container Feature packaging.

Build and exercise the local CLI:

```sh
go build -o ./bin/cue-eval-mcp ./cmd/cue-eval-mcp
./bin/cue-eval-mcp discover
./bin/cue-eval-mcp inspect --pack toy-plugin --target manifest
./bin/cue-eval-mcp evaluate --request examples/requests/toy-valid.json
```

Building requires Go 1.25 or newer. The CUE CLI is optional; the evaluator
embeds the pinned CUE Go module.

Start the local stdio MCP server with:

```sh
./bin/cue-eval-mcp serve --workspace "$PWD"
```

Contract review is fail-closed in this first version. Repository manifests may
declare review state, but no independent review verifier exists yet. Validation
therefore remains useful while every result stays ineligible for an authoritative
gate.

See [DESIGN.md](DESIGN.md) for architecture and limitations,
[the Codex guide](agents/codex/README.md) for agent setup,
[the Puppet integration](integrations/puppet/README.md) for a worked example,
[packaging](packaging/README.md) for container and Feature usage, and
[CONTRIBUTING.md](CONTRIBUTING.md) for the preferred branch and review workflow.
