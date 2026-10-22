# Puppet contract-first workflow specification

Status: **worked synthetic integration**. The generic evaluator, CLI, and MCP
server are implemented. Puppet data collection remains an external adapter
responsibility; this repository does not connect to PuppetDB or execute Puppet.

The central product and architecture proposal is in [DESIGN.md](../../DESIGN.md).
This document owns only Puppet-specific evidence and validation semantics.

This document is intentionally generic and suitable for a public repository.
All identifiers, data, hostnames, policies, and deployment waves are fictional.

## Purpose

Use shared CUE domain definitions to catch configuration mistakes earlier while
preserving Puppet as the system that performs lookup, interpolation, merging,
catalog compilation, application, and reporting.

The kit should add three things to an existing Puppet workflow:

1. reviewed, reusable vocabulary such as `#DatacenterID` and
   `#EnvironmentID`;
2. change-specific contracts that reference that vocabulary instead of
   recreating strings, lists, and regular expressions;
3. fast, structured feedback for a coding agent, with the same Go/CUE core used
   later by CLI and CI.

CUE complements Puppet linting, unit tests, catalog comparison, no-op runs, and
staged rollout. It does not replace them.

## Design boundaries

- Puppet/Hiera remains authoritative for lookup precedence, interpolation,
  `lookup_options`, and merge behavior. CUE must validate adapter output, not
  simulate Hiera with CUE unification.
- Source Hiera fragments are checked against explicit **partial** schemas.
  Resolved per-node values are checked against **complete** contracts.
- A partial result can only be `compatible-so-far`, `conflict`, or `error`.
  It is never complete and never eligible for a CI gate.
- A concrete implementation result uses the repository's existing outcomes:
  `valid-concrete`, `invalid-concrete`, or `error`.
- A content digest identifies bytes. It does not prove human approval. A
  trusted repository review boundary must establish whether a digest is a
  candidate or reviewed baseline.
- No secrets belong in fixtures, model-visible diagnostics, or MCP results.
- The first implementation should be bounded and local. It must not execute
  arbitrary commands supplied by a model.

## Proposed architecture

```text
Puppet/Hiera repositories and inventory
               |
       Python evidence adapter
       - lookup/compile collection
       - hostname parser
       - catalog normalization
       - provenance and coverage
               |
       versioned JSON evidence
               |
     generic Go + CUE validation core
        /                         \
 local stdio MCP                 CLI
 agent feedback            local/CI invocation
```

### Python evidence adapter

Python owns integration with Puppet-specific data and APIs. It should emit a
versioned, bounded JSON envelope containing:

- adapter and evidence-schema versions;
- baseline and candidate revision identities;
- fact snapshot identity and source;
- requested filters and node scope;
- requested, covered, excluded, and failed nodes;
- source Hiera fragments relevant to the contract;
- effective values resolved by Puppet lookup/compile semantics;
- relevant baseline and candidate catalog resources and edges;
- a structured delta;
- collection and compilation errors.

A display-oriented catalog diff alone is insufficient: a renderer may omit
unchanged resources or details for added resources, while a contract may need
those values to establish preservation or exact output. The adapter therefore
includes the relevant baseline and candidate facts directly, with provenance.

The baseline source remains an open decision. Two candidates are a stored
catalog or a fresh baseline compile using the same facts as the candidate. The
pilot must choose and test one; the specification does not assume either.

Puppet's official [`puppet lookup` documentation](https://help.puppet.com/core/current/content/puppetcore/using_puppet_lookup.htm)
describes lookup explanations, node facts, environments, merge overrides, and
when full compilation is necessary. The
[Catalog API](https://help.puppet.com/core/current/Content/PuppetCore/server/http_api/puppet-api/v4/catalog.htm)
accepts an environment and optional facts and returns a JSON compiled catalog.
Those interfaces inform the adapter; they are not MCP operations.

### Go/CUE core

The generic core accepts bounded contract packs and evidence. It resolves
configured definitions, computes content digests, performs authoring or
implementation validation, and returns the shared structured result schema.
The MCP server and CLI use the same validation logic. Independent review
verification and a Puppet evidence adapter are not implemented.

The current runtime reads packs from the consuming repository's configurable
`.cue-eval/project.json` manifest. A future `schemas/` convention is discussed
in [the central design](../../DESIGN.md), but the MVP does not require or create
that layout. The public examples here do not stand in for a deployment's
registry.

## First pilot: shared Hiera domain vocabulary

The initial contract pack should separate three concerns.

### 1. Authoritative registry data

Data records what currently exists:

- canonical datacenter identifiers;
- canonical long and short names;
- environment identifiers;
- allowed datacenter/environment combinations.

The example models availability as an association. It does not assume that
environments form a hierarchy beneath datacenters. Real deployments may be
many-to-many; maintain one authoritative association and derive reverse views.

Prefer deriving this registry from an existing authoritative inventory when
one exists. Generated registry data should record its source revision and be
reviewed for drift. Do not regenerate equivalent lists independently for every
change contract.

### 2. Schema and policy

Schemas define the shape and relationships of registry records and node data.
Policies define which values or combinations are permitted. Change contracts
reference shared definitions such as `#DatacenterID`, `#EnvironmentID`, and
`#EffectiveNodeData`.

Keep data and policy separate so a registry refresh does not silently rewrite
the meaning of a schema. Both remain digest-addressed and subject to review.

### 3. Hostname parsing

Use one parser implementation, or shared conformance fixtures consumed by all
implementations. The adapter should emit both the original hostname and its
parsed identity. CUE then checks that parsed datacenter/environment values agree
with declared and resolved values.

Do not independently reimplement a production hostname grammar in each CUE
contract. The synthetic example intentionally supplies parser output rather
than claiming a real-world hostname convention.

### Source versus effective validation

`#SourceHieraFragment` permits omitted fields because any one source file may
contribute only part of the eventual value. A successful partial check means
only “no conflict found in this fragment.”

`#EffectiveNodeData` requires all fields needed by the contract and is checked
for concreteness. Effective data must come from Puppet-aware lookup or compile
evidence so interpolation, precedence, and merge behavior have already been
applied.

### Pattern-map closure gotcha

A CUE pattern field constrains labels that match its pattern; it does not, by
itself, prove that other labels are forbidden. Contracts that require an exact
set of keys must use a closed struct or another explicit closure strategy, and
tests must include an unexpected-key fixture. Required fields and concrete
validation are separate concerns: a partial schema can be internally
consistent while still lacking required effective values.

## Synthetic executable example

[`examples/domain.cue`](examples/domain.cue) contains deliberately fictional
registry data and shared schemas. It demonstrates:

- canonical identifiers and names;
- allowed environment/datacenter combinations;
- adapter-produced hostname identity checked against declared values;
- a partial source-fragment schema;
- a complete effective-data contract;
- a closed illustrative settings map.

Run from the repository root:

```sh
# Schema and registry consistency.
cue vet ./integrations/puppet/examples

# Complete positive evidence: must pass.
cue vet -c -d '#EffectiveNodeData' \
  ./integrations/puppet/examples/domain.cue \
  ./integrations/puppet/examples/fixtures/valid.json

# Each must fail for the stated relationship.
cue vet -c -d '#EffectiveNodeData' \
  ./integrations/puppet/examples/domain.cue \
  ./integrations/puppet/examples/fixtures/invalid-combination.json
cue vet -c -d '#EffectiveNodeData' \
  ./integrations/puppet/examples/domain.cue \
  ./integrations/puppet/examples/fixtures/hostname-mismatch.json

# The incomplete source is compatible as a fragment...
cue vet -d '#SourceHieraFragment' \
  ./integrations/puppet/examples/domain.cue \
  ./integrations/puppet/examples/fixtures/incomplete.json

# ...but must fail complete effective-data validation.
cue vet -c -d '#EffectiveNodeData' \
  ./integrations/puppet/examples/domain.cue \
  ./integrations/puppet/examples/fixtures/incomplete.json
```

The example names and allowed combinations are teaching data, not recommended
defaults.

The repository project manifest exposes these definitions as the
`puppet-domain-example` pack. Discover and inspect it with:

```sh
cue-eval-mcp discover --workspace . --query puppet
cue-eval-mcp inspect --workspace . \
  --pack puppet-domain-example --target effective-node-data
```

MCP clients use the equivalent `discover_contracts`, `inspect_contract`, and
`evaluate` tools. File evidence must be under the pack's configured
`inputRoots`. Inline evidence is bounded by the evaluator's input limit.

Author mode accepts a CUE candidate definition and reports only schema
compatibility. A JSON source fragment can instead be checked in implementation
mode against the permissive `source-fragment` target; that validates the
fragment itself and does not establish complete effective-node coverage.
Complete adapter output belongs against `effective-node-data`.

## Secondary worked change: rendered Apache environment values

Consider a fictional Puppet profile with:

- a hash parameter mapping names to values;
- a boolean that gates rendering those values;
- a template that emits Apache configuration;
- illustrative `dev` → `preprod` → `prod` rollout waves.

The hash may be staged while the boolean is off. A change contract should
compose the effective Hiera value, toggle, node identity, baseline rendered
configuration, candidate rendered configuration, and parsed directive evidence.

Proposed invariants:

- **Off:** candidate rendered output is identical to baseline output under an
  equivalent fact snapshot.
- **On:** parsed output contains exactly the requested keys and values, at the
  intended Apache scope, with no missing or extra directives.
- The result is based on actual rendered output parsed by an adapter, not a
  model's statement that it rendered correctly.
- Parser/adaptor tests and `apachectl configtest` complement CUE validation.

This example assumes the template emits Apache `SetEnv`; that assumption must
be confirmed for a real change. `SetEnv` creates Apache internal environment
variables that can be passed to CGI/SSI. It is not the same as process
environment inheritance via `PassEnv`, and configuration-file `SetEnv` values
are too late for `${VAR}` expansion in the same Apache configuration. See the
official [`mod_env` documentation](https://httpd.apache.org/docs/2.4/en/mod/mod_env.html)
and [configuration guide](https://httpd.apache.org/docs/2.4/en/configuring.html).

Wave eligibility should be computed from the reviewed registry and change
contract. A wave snapshot does not prove that a previous wave succeeded over
time; promotion requires trusted CI/deployment evidence defined outside the
snapshot.

## Agent workflows

Agent-specific setup and operating instructions belong outside this integration
specification. The current [Codex workflow](../../agents/codex/README.md)
links back here and does not redefine Puppet contracts or adapter evidence.

## Pilot acceptance criteria

The pilot is acceptable only when automated fixtures establish all of the
following:

1. unknown datacenter or environment identifiers fail;
2. individually valid identifiers in a forbidden combination fail;
3. parsed hostname identity that disagrees with declared identity fails;
4. an incomplete JSON source fragment can pass its explicit partial target but
   fails the complete effective-data target; a CUE candidate definition can be
   `compatible-so-far` in author mode but is never a complete pass;
5. unexpected registry or closed-map keys fail;
6. registry generation drift is detected against shared fixtures or a canonical
   parser;
7. missing revision, fact-snapshot, input provenance, or evidence-schema
   identity produces `error`, not a domain pass;
8. partial node coverage or node collection failures prevent a complete pass;
9. disabled behavior preserves baseline rendered output under equivalent facts;
10. enabled behavior matches actual parsed rendered output exactly, including
    keys, values, scope, and count;
11. wave eligibility uses reviewed registry data, while temporal promotion
    requires separately trusted evidence.

## Adoption plan

1. Review the domain vocabulary and trust boundary using only synthetic data.
2. Connect one approved inventory source to a local adapter and compare its
   generated registry with independently reviewed fixtures.
3. Run source-fragment validation as advisory feedback.
4. Add effective per-node validation and coverage/provenance errors.
5. Pilot one bounded change contract alongside existing tests and no-op checks.
6. Add CLI/CI enforcement only after false-positive and evidence-completeness
   criteria are met.
7. Add the local MCP workflow using the same evaluator already exercised by CI.

## Open decisions

- What system is authoritative for registry data, and how is drift reviewed?
- What evidence establishes a reviewed baseline beyond its digest?
- Is the catalog baseline stored state or a fresh compile with the same facts?
- Which Hiera resolution path supplies effective values, and when is a full
  compile required?
- What hostname parser is canonical, and which cross-language fixtures prevent
  drift?
- What constitutes equivalent facts for baseline/candidate comparison?
- What node coverage threshold is complete, and how are exclusions approved?
- Which catalog resources and edges must always be included even when a display
  diff omits them?
- What Apache directive and scope are intended in the worked change?
- What trusted evidence permits promotion between rollout waves?
- What bounded input, diagnostic, and evaluation limits apply locally and in CI?

## References

- [CUE language specification](https://cuelang.org/docs/reference/spec/)
- [Puppet lookup command](https://help.puppet.com/core/current/content/puppetcore/using_puppet_lookup.htm)
- [Puppet Catalog API](https://help.puppet.com/core/current/Content/PuppetCore/server/http_api/puppet-api/v4/catalog.htm)
- [Apache `mod_env`](https://httpd.apache.org/docs/2.4/en/mod/mod_env.html)
- [Apache configuration files](https://httpd.apache.org/docs/2.4/en/configuring.html)
