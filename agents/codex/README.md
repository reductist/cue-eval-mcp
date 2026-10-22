# Codex workflow

This repository provides a local stdio MCP server and CLI. Codex can use the
server for fast feedback while CI invokes the same evaluator independently.
MCP installation alone does not require an agent to call these tools; the
consuming repository's root `AGENTS.md` should define when evaluation is
required.

## Build and configure

Build the binary from this repository:

```sh
go build -trimpath -o ./bin/cue-eval-mcp ./cmd/cue-eval-mcp
```

Codex supports stdio servers configured in `config.toml`, including
project-scoped `.codex/config.toml` in trusted repositories. Configuration is
an explicit user action; this project does not write it automatically.

```toml
[mcp_servers.cue_eval]
command = "/absolute/path/to/cue-eval-mcp"
args = ["serve", "--workspace", "/absolute/path/to/consuming-repository"]
startup_timeout_sec = 10
tool_timeout_sec = 60
```

The client timeout is a host-side bound. The evaluator checks cancellation
between batch items; it cannot hard-interrupt an in-flight CUE call.

Use absolute paths or another deliberately managed executable location. The
workspace must contain `.cue-eval/project.json` unless `--manifest` selects a
different workspace-relative manifest. See the current
[official Codex MCP documentation](https://developers.openai.com/codex/mcp)
for configuration scope and supported options.

The manifest's pack `sources` are an explicit local schema dependency list.
This resembles a fixture mapping, but the evaluator does not fetch packages,
resolve versions, or discover transitive dependencies. If the user supplies the
manifest mapping, use it as the authoritative source selection and skip mapping
proposal or negotiation. That does not make it verified review authorization.
Validate it normally, but do not silently augment or rewrite it. Paths are
relative to the pack root and must resolve inside the workspace. Listed shared
and change-specific files are compiled as one compatible CUE package, so shared
definitions can be reused without copying. CUE import statements remain
unsupported.

If no mapping exists, inspect the repository with ordinary code understanding,
propose an initial `sources` list with rationale and uncertainties, and ask
whether it is good enough to start. Apply the user's adjustment instructions and
persist the confirmed manifest mapping through normal repository edits; the MCP
tools remain read-only. This is best-effort selection, not exhaustive impact
analysis, and it does not require a parser dependency. There is no separate
mapping override loader in the current server; supplying
`.cue-eval/project.json` is the supported explicit path.

## Agent loop

First decide whether the task has stable, observable data constraints that a
contract can check usefully. If the domain is too uncertain or the important
behavior cannot be represented by available evidence, say so and use the
repository's normal tests instead of inventing a schema.

For routine setup, choose configuration that fits the domain and repository
conventions. Use documented starting points, explain choices that affect how the
contract is used, and state when they should be adjusted. Do not ask the user to
decide every mechanical detail. Ask about consequential domain choices or
ambiguity, and reuse recorded decisions. Contract review follows the consuming
repository's normal process; do not create a separate approval workflow or
treat declared review metadata as verified authorization.

1. Use the task intent, repository structure, relevant examples, shared schemas,
   and affected code to form a relevance hypothesis. Call `discover_contracts`
   with useful search terms, then inspect plausible results. Names,
   descriptions, and artifact labels are hints, not proof of applicability or
   approval. If the user already supplied the pack mapping, use that selection.
   Otherwise, if applicable definitions are not listed, propose the required
   workspace-local source-file mapping, explain uncertainty, and ask whether it
   is good enough to start.
2. Call `inspect_contract` for plausible targets and read their source before
   selecting them. Explain why each selected contract applies and ask focused
   questions when the scope is ambiguous.
3. Record the selected exact contract identifiers, content digests, any
   contract-defined versions, the evidence boundary, and known coverage gaps.
   Discovery assists this judgment; it does not prove that all dependencies or
   applicable contracts were found.
4. Inspect the native code, resolution semantics, and tests. Propose required,
   optional, native-default, and null behavior in natural technical language;
   confirm consequential uncertainty unless the user already specified it.
5. In author mode, collaborate with the user on a CUE candidate definition and
   evaluate it for `compatible-so-far` or `conflict`. Compatibility remains
   incomplete and gate-ineligible. Persist the agreed constraints and rationale
   in the consuming repository instead of hiding them in a prompt.
6. Implement the change and have the integration adapter produce actual,
   bounded evidence.
7. Call `evaluate` in implementation mode. Repair the implementation or request
   missing evidence; do not weaken the contract to obtain a pass.
8. Report exact contract identifiers, versions where defined, contract and input
   digests, checked evidence boundary, diagnostics, known coverage gaps, and
   anything not evaluated.

The reusable [`AGENTS.template.md`](AGENTS.template.md) captures these rules.
It is a template only; Codex does not automatically load it from this server
repository.

## Current trust boundary

The project manifest may declare a pack as `candidate` or `reviewed`, but the
MVP has no independent review-verification mechanism. Results expose that
declared state separately from `reviewVerified`, which is always false.
`gateEligible` is therefore always false even when implementation validation
returns `valid-concrete`. This keeps validation useful without treating a
digest or caller assertion as approval.

File subjects are limited to each pack's configured `inputRoots`. Contract and
subject imports are disabled, paths are workspace-relative, and rooted file
access rejects traversal and escaping symlinks. The server runs no external
commands and holds no integration credentials.

## CLI parity

The same evaluator is available without MCP:

```sh
cue-eval-mcp discover --workspace .
cue-eval-mcp inspect --workspace . --pack toy-plugin --target manifest
cue-eval-mcp evaluate --workspace . --request examples/requests/toy-valid.json
```

`evaluate` writes the structured result before returning a nonzero status for
request errors, `conflict`, `invalid-concrete`, or per-item `error` outcomes.
CI should remain the authoritative enforcement point.

The agreed MVP validates effective configuration after the native application or
adapter resolves its own defaults and derivations. Strict validation must not
silently add absent evidence from CUE defaults, constants, or derivations.
Required, optional, and nullable remain separate: nullable permits an explicit
null but does not permit absence unless the field is optional. Raw fragments use
a separate partial-input contract and never count as full effective evidence.

The evaluator enforces this strict evidence rule for JSON and YAML
implementation input. A schema-produced value absent from the submitted input
returns `invalid-concrete` with a `missing-evidence` diagnostic. CUE input is
supported in author mode but rejected in implementation mode because it cannot
yet provide the same evidence-presence guarantee. CUE-driven completion and
nested completion overrides are deferred.

The current `discover_contracts` tool is a deterministic metadata search, not
context-aware impact analysis. Explicit bindings, native adapters, or structural
analysis can improve future coverage, but they are not required for the agent to
propose and negotiate a useful contract now.

The server does not currently generate this routine configuration or enforce
the collaborative decision process. These are instructions for the agent using
the tools.

The intended version 1 experience is deliberately small: reusable agent
instructions plus discovery, inspection, and evaluation support a repeatable
collaborate–implement–validate–revise loop. Broader code understanding, universal
language semantics, and workflow or approval infrastructure are possible future
extensions, not prerequisites.
