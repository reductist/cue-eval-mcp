# Contract-first agent development kit

Status: **working MVP implemented; review trust and production adapters remain open**.

This document is the central design source. Integration documents own
tool-specific evidence and semantics. Agent documents own agent-specific setup
and workflow templates. Neither should duplicate this design.

## Goal

Package two activities into one consistent development loop:

1. collaborative definition of useful, reviewed contracts; and
2. fast CUE-backed evaluation of actual implementation evidence while a coding
   agent iterates.

Version 1 supplies the minimum reusable agent instructions and CUE tools needed
to bootstrap that loop consistently. It is intended for tasks where practical
contracts can capture meaningful data constraints. If important behavior is too
unknown, unstable, or poorly represented as data, the agent should say that
modeling is not useful yet rather than force a speculative schema.

For each design subproblem, assume a capable agent: provide context,
conventions, reusable contracts, and reliable validation feedback instead of
reimplementing the agent's reasoning inside the server.

The intended user does not need data-modeling expertise. They describe what
should be safe or true in ordinary language. The agent helps turn that into a
reviewable model, explains safeguards in terms of failures prevented, asks
small operational questions, reuses reviewed domain definitions, proposes
bounds, and reads the result back in plain language before the user agrees.

After agreement, the agent implements the change and evaluates real
adapter-derived data, configuration, or output. On failure it repairs the
implementation or asks for missing evidence; it does not silently weaken the
reviewed contract.

Every final report identifies the exact contract and input revision, checked
scope, failures, and behavior that remained unchecked or unavailable.

## Non-goals

- Not merely an MCP wrapper around `cue vet`.
- Not a replacement for unit, integration, parser, compile, syntax, no-op, or
  deployment tests.
- Not a hidden mechanism for a model to revise constraints during coding.
- Not a full dependent type system for every host language.
- Not a proof system for arbitrary implementation behavior.
- Not a general natural-language DSL/compiler in the first release.
- Not a production-data collector or credential holder.
- Not universal code understanding or deterministic impact analysis.
- Not a requirement that every task, language behavior, or unknown domain be
  represented as a contract.
- Not a mandatory workflow engine or elaborate approval system around the
  evaluator.

CUE checks dependent constraints over explicitly modeled data. Adapters and
tests establish how that data relates to the implementation.

## Collaborative modeling workflow

```text
operational goal in natural language
               ↓
structured domain model
(entities, references, cardinality, bounds, associations, invariants)
               ↓
plain-language semantic readback + examples
               ↓
human agreement and separately trusted review evidence
               ↓
CUE definitions + positive and negative fixtures
               ↓
implementation evidence evaluated during iteration and in CI
```

The agent should ask incrementally rather than presenting a large schema as a
fait accompli. Useful questions are operational: “Can one service belong to
more than one site?”, “What failure should an upper bound prevent?”, or “Does
absence mean forbidden or simply unknown?”

For routine configuration, the agent chooses values and structure that fit the
domain, repository conventions, available evidence, and intended validation
loop. Project guidance should offer sensible starting points, explain why they
are useful, and identify conditions that justify changing them. It should not
turn every setup into a rigid universal default or a questionnaire about details
the agent can resolve safely from context.

The agent asks about consequential unresolved domain choices and genuine
ambiguity. It reuses recorded decisions instead of asking again. For evidence
shape, it inspects native code, semantics, and tests, proposes required,
optional, default, and null behavior in natural technical language, and confirms
consequential uncertainty unless the user already supplied the answer. Formal
review follows the consuming repository's existing process; this kit does not
require a separate approval system, and a stored `reviewed` label is not
independently verified authorization. This is agreed workflow guidance, not
configuration automation implemented by the current server.

### Agent-assisted contract relevance

The core product priority is a useful collaborative workflow, not a requirement
to prove every possible impact before contract work can begin. The agent uses
the task intent, repository structure, relevant examples, reviewed shared
schemas, and its understanding of the affected code to propose applicable
contracts. It explains why each contract appears relevant and asks focused
questions when the relationship or requested scope is ambiguous.

Contract discovery is assistance, not an exhaustive impact proof. Literal
identifiers, titles, descriptions, or artifact labels are search hints; they do
not establish applicability or approval by themselves. The agent inspects the
contract contents and records the selected exact contract identifiers, content
digests, any contract-defined versions, evidence boundary, and known coverage
gaps. CUE evaluation of the selected contract and supplied evidence is
deterministic, while the preceding relevance judgment remains reviewable agent
work.

Explicit artifact bindings, native evidence adapters, and structural analysis
such as Tree-sitter may improve coverage over time. They are optional aids, not
prerequisites for useful contract definition and evaluation. The agent must not
claim exhaustive dependency coverage, silently weaken a contract to obtain a
pass, or treat descriptive metadata as trusted review evidence.

These and other impact-analysis features are potential extensions, not version
1 prerequisites. Version 1 remains useful when an agent and user can agree on a
bounded contract, obtain relevant evidence, and repeat the
implement–validate–revise loop alongside normal tests.

### Agreed MVP evidence semantics

The native application or integration adapter resolves its own defaults,
precedence, merging, and derived behavior. Strict effective-configuration
validation checks the resulting evidence. It must not silently add an absent
value from a CUE default, constant, or derivation.

Required, optional, and nullable are separate contract choices. A required
field must be present in effective evidence. An optional field may be absent.
Nullable permits an explicit null value; it does not make absence valid unless
the field is also optional. When only a raw fragment is available, validate it
against a separately designed partial-input contract and keep the result
incomplete. Do not present a compatible fragment as full effective evidence.

The agent inspects the native code, resolution semantics, and tests, proposes
the required/optional/default/null behavior in plain technical language, and
asks only about consequential uncertainty. It records the agreed rationale with
the contract for reuse. Version 1 does not attempt a universal language or
default extractor.

CUE-driven completion and nested completion overrides are deferred. In
implementation mode, the current evaluator enforces the strict rule for JSON
and YAML by rejecting every materialized path absent from the submitted
evidence. CUE subjects remain available in author mode; implementation-mode CUE
is rejected because equivalent evidence-presence guarantees are not implemented.

### Proposed modeling vocabulary

The first version should be a playbook, not a compiler. Suggested mappings:

| User concept | Structured meaning |
|---|---|
| “is one of” | closed enumerated value set |
| “between” | inclusive or exclusive bound, stated explicitly |
| “references” | referenced identity must exist in a registry |
| “hosts” / “is available in” | association between two entity sets |
| “exactly one parent” | cardinality of one, not merely existence |
| “multiple” | clarify “at least two” versus unrestricted many |
| “not all are allowed” | explicit forbidden combinations; never infer the rest |
| “no indirect self-dependency” | explicit graph acyclicity constraint or validator |

An environment/datacenter relationship is often many-to-many, not a hierarchy.
Its reverse view should be derived from one authoritative association rather
than maintained independently. Two reverse views of the same association are
not a dependency cycle.

CUE reference-cycle semantics do not prove that an application graph is
acyclic. If acyclicity matters, model edges explicitly and add an appropriate
bounded constraint or dedicated validator.

## Contract ownership in consuming repositories

Proposed default:

```text
schemas/
  project manifest          # exact name and format remain open
  shared/                   # reviewed reusable definitions and registries
  changes/                  # separately identifiable negotiated contracts
  fixtures/                 # positive, negative, and incomplete examples
```

The project manifest would declare schema roots, exported definitions, fixture
locations, and artifact mappings. Its filename and exact shape are not yet
settled and must be configurable.

The consuming repository owns its contracts. Private registry data stays in
that repository and should be derived from an authoritative inventory when one
exists. This public project contains only generic definitions and synthetic
examples.

Shared definitions and negotiated change contracts have separate identity and
review provenance. A change contract references pinned shared definitions;
the service does not regenerate lists, regular expressions, or policy for each
change.

The MVP uses an explicit `.cue-eval/project.json` manifest. It lists pack roots,
exact contract source files, allowed evidence roots, targets, formats, and
declared review state. The alternative `schemas/` layout above remains a future
design option; no migration is implied. Discovery never creates, initializes,
or moves schema directories.

For version 1, the manifest's `sources` list is the explicit schema dependency
mapping. It is similar in purpose to a fixture or dependency list: the agent may
propose the files, and the user supplies or confirms the mapping. If the user
already supplies the manifest mapping, it is authoritative for source selection
and the agent skips proposal and negotiation. This authority applies to file
selection, not review approval. The evaluator still validates the manifest
syntax, supported fields and formats, workspace path boundary, listed files, and
CUE package compatibility. The agent does not silently add files or rewrite the
user's selection.

When no mapping is supplied, the agent uses ordinary code understanding to
inspect the repository on a best-effort basis, proposes an initial `sources`
list with its rationale and uncertainties, and asks whether it is good enough to
start. The user can give adjustment instructions; the confirmed result is then
persisted in `.cue-eval/project.json` for reuse. This is not exhaustive impact
analysis and requires no dedicated parser. A separate mapping-import or override
interface would be a future feature; today, a user-provided project manifest is
the supported explicit mapping mechanism.

The mapping is not a package manager, fetch mechanism, version resolver, or
transitive dependency scanner. Each source path is resolved relative to the pack
root and must remain inside the workspace. Explicitly listed files may live
outside that root via a workspace-contained relative path; the evaluator adds
them to one CUE build instance, so they must form a compatible CUE package.
Shared definitions do not need to be copied when they can be listed within this
boundary. CUE import statements remain disabled.

## Implemented generic MCP surface

The local stdio server exposes three read-only tools through the official Go
MCP SDK. The CLI invokes the same evaluator.

### `discover_contracts`

Returns a small, searchable metadata index: identifiers, titles, summaries,
declared review state, configured targets, and the exact contract digest. It is
read-only and does not initialize a directory or rewrite a manifest. Results
support agent relevance judgment; they do not prove that every applicable
contract was found.

### `inspect_contract`

Returns the exact configured contract sources, selected target, digest, and
declared review metadata. The MVP does not resolve unlisted transitive imports,
enumerate open domains, or perform context-based narrowing.

### `evaluate`

Evaluates a batch of fixtures or evidence items in the existing modes:

- **author:** `compatible-so-far`, `conflict`, or `error`;
- **implementation:** `valid-concrete`, `invalid-concrete`, or `error`.

Partial compatibility is always incomplete and never gate-eligible. Required
field checks and concrete-value checks are explicit.

MCP transport schemas should use JSON Schema for interoperability, while CUE
remains the richer validation backend. Avoid per-language or per-type tool
proliferation.

Contract source, examples, and full reports may also be exposed as resources,
but resources are not automatically read by every host. Critical verdicts and
diagnostics remain directly accessible from the evaluation result.

## Go/CUE core

One Go core serves MCP, CLI, and CI so semantic behavior does not drift between
interfaces. The MVP implements exact-byte digests, bounded batch/input/source
sizes, structured diagnostics, rooted file access, and import rejection.

- A compiled-value cache is not implemented. Any future cache must key the
  exact contract, dependency contents, evaluator version, and relevant options.
- The MVP bounds batch size, input and contract bytes, source-file count,
  diagnostic count and bytes, and filesystem scope. Evaluation is sequential.
  Context cancellation is checked between items; there is no hard interruption
  of an in-flight CUE call.
- Support batch evaluation and compact, grouped diagnostics for agent loops.
- Permit scoped workspace references only under configured roots, resolve them
  safely, and include exact-byte hashes in results.
- Do not execute arbitrary commands or hold integration credentials. Adapters
  collect and normalize external evidence.
- Pin and verify CUE and MCP SDK versions during implementation. Do not make
  universal claims about context or concurrency limitations without measuring
  the pinned versions.

Each batch item should report:

- mode and semantic outcome;
- input identity/digest and contract revision/digest;
- separately trusted review provenance;
- exact evaluated scope; adapters must represent evidence coverage in their
  contract-defined input because the generic result has no coverage field;
- structured diagnostics with CUE path and source location;
- supplied JSON and YAML paths plus any schema-materialized paths absent from
  that evidence. CUE implementation input is rejected, and the evaluator
  cannot independently distinguish an adapter-derived value from an observed
  native value;
- truncation, unavailable evidence, and unchecked behavior.

A digest proves content identity, not approval. Review provenance must come
from an independently trusted repository or CI boundary.

## Agent and enforcement boundary

Installing an MCP server does not force an agent to invoke it. A consuming
repository's root `AGENTS.md` is the Codex entrypoint for durable workflow
instructions; this repository's [Codex directory](agents/codex/README.md) is
only documentation and a reusable template.

CI or another trusted runner enforces reviewed contracts independently of an
agent. MCP provides low-latency feedback during iteration; it is not the sole
gate.

## What exists today

- Validated CUE request/result definitions in [`.cue-eval/api`](.cue-eval/api).
- Five aligned semantic outcomes and candidate/reviewed trust fields.
- A validated synthetic toy pack and fixtures.
- A validated synthetic [Puppet integration example](integrations/puppet/README.md).
- A working Go evaluator, CLI, and local stdio MCP server with three read-only
  tools.
- Native, minimal OCI, and offline-first Dev Container Feature packaging.
- A tested [Codex workflow and instruction template](agents/codex/README.md).
- Strict effective-evidence enforcement for JSON and YAML implementation input,
  including nested objects and lists.

No review verifier, compiled-value cache, external data adapter, initialization
workflow, or production consuming-repository integration is implemented.
Context-aware relevance ranking, structural code analysis, lifecycle writes,
and cross-pack imports are also not implemented.

## Open decisions

- Project-manifest name, schema, and default root behavior.
- Whether and how the current `.cue-eval` draft maps to consuming `schemas/`.
- Final MCP capability names and JSON Schema transport contracts.
- Review-provenance format and trusted verifier.
- Change-contract identity, lifecycle, and contract-evolution policy.
- Cache boundaries, invalidation tests, and bounded resource limits.
- When graph invariants belong in CUE versus a dedicated validator.
- Whether implementation-mode CUE can provide evidence-presence guarantees
  equivalent to JSON and YAML without conflating constraints with observations.
- Minimum safe inline diagnostic detail and resource/report retention.
