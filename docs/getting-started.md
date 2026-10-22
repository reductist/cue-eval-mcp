# Getting started: turn a Puppet change into a reusable check

This walkthrough shows the intended interactive workflow through a small,
fictional Puppet change. The conversation, tool calls, and results are
illustrative and are not an actual tool-execution transcript.

The project's overall goal is to turn operational knowledge into a persistent
data specification. A user and an agent work together in natural language to
establish useful domain definitions and value bounds. The agent records the
agreed result as CUE contracts, and this tool gives the agent fast validation
against those contracts during its implementation loop.

The result is compact, reusable guidance for the agent. It can help produce
better-targeted changes without requiring every user request to restate all of
the codebase's operational constraints. This is conceptually similar to
automated CI tests, but the feedback is available earlier while the agent is
still developing the change.

Contracts complement Puppet and rspec-puppet; they do not replace them. They
record the data conditions that must remain true while Puppet performs its
usual Hiera lookup, catalog compilation, application, and reporting.

## Prerequisites

- Build and configure `cue-eval-mcp` using the [Codex guide](../agents/codex/README.md).
- Add an explicit `.cue-eval/project.json` to the Puppet repository. Discovery
  reads that manifest; it does not scan the repository or create contracts.
- Keep the repository's normal Puppet lint, rspec-puppet, catalog, and no-op
  checks available.

For a first local trial, follow the Codex guide, adapt the reusable
[`AGENTS.md` template](../agents/codex/AGENTS.template.md), and configure one
synthetic pack and evidence file. The server will not create or register these
for you.

The fictional repository already has a contract pack named
`puppet-profile-contracts`. Its manifest lists `shared.cue`, which contains a
reusable definition for short, single-line text. Current contract packs compose
shared and change-specific files by listing them as sources in the same CUE
package. This explicit list plays a role similar to a fixture dependency
mapping, but it does not fetch or resolve packages. Paths are relative to the
pack root, must remain inside the workspace, and may identify shared files
outside the root when listed with a workspace-contained relative path. CUE
imports are intentionally rejected by the evaluator.

In this example, the repository already supplies that mapping, so the AI uses it
as the source selection rather than proposing another one. The evaluator still
checks the manifest, files, path boundary, and resulting CUE package. The AI
does not silently add sources or rewrite the mapping. If no mapping existed, it
would inspect the repository best-effort, propose an initial list with its
rationale and uncertainties, and ask whether it was good enough to start.

## 1. Start with the operational change

**User**

> Add a `notice_text` parameter to our existing `profile::notice` class. Read
> it from Hiera, assign it to a local variable, and use it as the content of the
> managed notice file.

**AI**

> Based on the requested parameter, the existing Puppet class, and the nearby
> Hiera examples, I will first check whether the repository already defines a
> reusable contract for this kind of value. Then I will explain why any result
> appears relevant and ask about the limits that affect safe operation before
> changing Puppet code.

Illustrative MCP call:

```json
{
  "name": "discover_contracts",
  "arguments": {
    "query": "puppet",
    "limit": 10
  }
}
```

Illustrative result excerpt:

```json
{
  "packs": [
    {
      "id": "puppet-profile-contracts",
      "title": "Puppet profile contracts",
      "declaredReviewState": "candidate",
      "reviewVerified": false,
      "contractDigest": "sha256:<digest>"
    }
  ]
}
```

The AI inspects the configured sources rather than guessing their contents:

```json
{
  "name": "inspect_contract",
  "arguments": {
    "pack": "puppet-profile-contracts",
    "includeSource": true
  }
}
```

The returned `shared.cue` includes this existing definition:

```cue
package contracts

#SingleLineNotice: string & =~"^[ -~]{1,64}$"
```

In plain language, the value must contain 1–64 printable ASCII characters on
one line. That is an illustrative local policy, not a general recommendation.
The matching pack name and description only made it a useful candidate. Reading
the definition and comparing it with the requested value establishes the
working rationale; it does not prove that discovery found every dependency or
applicable contract. The final report records the selected contract digest,
the evidence checked, and any known coverage gaps.

## 2. Agree on what should be allowed

**AI**

> I found an existing `#SingleLineNotice` definition. Before reusing it:
>
> 1. Should an empty notice be rejected?
> 2. Should line breaks or other control characters be rejected?
> 3. Is 64 characters a useful upper bound for the managed file and its
>    consumers?
> 4. The current class has no parameter default. Should the new value remain
>    required so catalog compilation fails when Hiera has no value, or should
>    Puppet define a native default? I recommend keeping it required so a
>    missing operational setting is visible. Can the resolved value ever be
>    null, or must it always be a string?

These questions connect each constraint to an operational outcome: avoiding an
empty managed file, keeping the result readable, and detecting missing data
instead of silently changing behavior.

**User**

> Reuse that definition. The notice must be present, printable, one line, and at
> most 64 characters. Do not add a Puppet class parameter default; a missing
> Hiera value should fail compilation. The resolved value must not be null.

**AI**

> Understood. The class will require `notice_text`. Hiera will provide
> `profile::notice::notice_text`. The accepted value is one printable line from
> 1 through 64 characters. Puppet supplies no default, so effective evidence
> must contain a non-null string and missing data is an error. I will record that
> rationale with the change contract before implementing it so later work does
> not reopen the same question unless the native behavior or scope changes.

The AI can check that a proposed CUE bound does not conflict with the existing
shared definition:

```json
{
  "name": "evaluate",
  "arguments": {
    "items": [
      {
        "id": "notice-bound-candidate",
        "mode": "author",
        "pack": "puppet-profile-contracts",
        "target": "single-line-notice",
        "format": "cue",
        "expression": "#CandidateNotice",
        "source": {
          "type": "inline",
          "content": "package contracts\n\n#CandidateNotice: string & =~\"^[ -~]{1,64}$\"\n"
        }
      }
    ]
  }
}
```

Illustrative result excerpt:

```json
{
  "items": [
    {
      "id": "notice-bound-candidate",
      "outcome": "compatible-so-far",
      "completeCheck": false,
      "gateEligible": false,
      "diagnostics": []
    }
  ]
}
```

`compatible-so-far` means only that the proposed CUE constraints do not
conflict. It does not mean that the Puppet change works, that evidence is
complete, or that a reviewer approved the contract.

### Why use a contract here?

This notice example could also be covered by an rspec-puppet test. CUE becomes
more useful when several independently owned constraints must remain true at
the same time. Consider a Puppet parameter for a worker count:

1. The shared service definition allows 1 through 64 workers.
2. Production policy requires at least 3 workers.
3. The active change limits the count to no more than 5.

CUE can state those constraints directly:

```cue
workers: int & >=1 & <=64
workers: >=3
workers: <=5
```

Repeated constraints intersect rather than replacing earlier definitions. The
combined field therefore permits only 3, 4, or 5. No concrete worker count has
been supplied yet, but the agent already has a useful definition to guide the
change. If another policy required 8 or more workers, CUE would report the
contradiction before a deployment was needed to expose it.

CUE calls this iterative refinement
[unification](https://cuelang.org/docs/tour/basics/unification/). The same idea
supports reusable domain definitions and change-specific constraints that are
often more awkward to maintain as separate imperative checks.

## 3. Persist the agreed contract

The consuming repository adds a change-specific definition alongside the
shared definition:

```cue
package contracts

#NoticeInput: {
	node: string & =~"^[a-z0-9][a-z0-9.-]{0,252}$"
	hiera: {
		"profile::notice::notice_text": #SingleLineNotice
	}
}
```

The repository also records the native behavior and contract rationale: Puppet
has no parameter default, the effective value is required and non-null, and the
contract checks the value Puppet resolved. A raw Hiera fragment would use a
separate partial-input contract rather than being reported as full effective
evidence.

The MVP does not use CUE to complete effective evidence. For JSON and YAML
implementation input, the evaluator rejects schema-produced defaults,
constants, or derivations that were absent from the submitted evidence. Use CUE
input for author-mode compatibility checks; implementation-mode CUE is rejected
because it cannot yet provide the same evidence-presence guarantee.

The pack's entry in `.cue-eval/project.json` lists both files and exposes the
definitions that the tools may evaluate:

```json
{
  "id": "puppet-profile-contracts",
  "title": "Puppet profile contracts",
  "summary": "Validates effective data for fictional Puppet profiles.",
  "root": "schemas/contracts",
  "review": { "state": "candidate" },
  "sources": ["shared.cue", "notice.cue"],
  "inputRoots": [".cue-eval/evidence"],
  "targets": [
    {
      "name": "single-line-notice",
      "definition": "#SingleLineNotice",
      "artifactKinds": ["text-value"],
      "formats": ["cue"]
    },
    {
      "name": "notice-input",
      "definition": "#NoticeInput",
      "artifactKinds": ["effective-hiera-data"],
      "formats": ["json"]
    }
  ]
}
```

Agreement in the conversation is not verified review provenance. The current
MVP records the manifest's declared state, but `reviewVerified` and
`gateEligible` remain false until an independent review mechanism is added.

## 4. Implement the Puppet and Hiera change

Puppet can use automatic class-parameter lookup when a class is included. The
Hiera key is the fully qualified class and parameter name:

```yaml
profile::notice::notice_text: 'Planned maintenance at 22:00 UTC'
```

The fictional class becomes:

```puppet
class profile::notice (
  String[1, 64] $notice_text,
) {
  $notice_content = "${notice_text}\n"

  file { '/etc/example-notice':
    ensure  => file,
    content => $notice_content,
  }
}
```

Because the parameter has no default, a classified node with no matching value
should fail during catalog compilation. Puppet's type limits the length; the
CUE contract additionally applies the agreed single-line printable-character
policy. Puppet documents class parameters as the preferred lookup interface in
[Methods for data lookup](https://help.puppet.com/core/current/Content/PuppetCore/methods_for_data_lookup.htm).

## 5. Validate effective data, not a model's claim

The MCP server does not run Puppet or connect to PuppetDB. A trusted local
adapter or CI step must collect the value that Puppet actually resolves for a
representative node. For example, `puppet lookup` can use the node and
environment context and can explain where the result came from:

```sh
puppet lookup profile::notice::notice_text \
  --node node01.example.test \
  --environment test \
  --render-as json

puppet lookup profile::notice::notice_text \
  --node node01.example.test \
  --environment test \
  --explain
```

The exact collection command and provenance envelope are repository decisions.
The official [`puppet lookup` reference](https://help.puppet.com/core/current/content/puppetcore/Markdown/lookup.htm)
describes node facts, environments, rendering, compilation, and explanations.

Assume the adapter writes this bounded evidence inside the pack's configured
`inputRoots`:

```json
{
  "node": "node01.example.test",
  "hiera": {
    "profile::notice::notice_text": "Planned maintenance at 22:00 UTC"
  }
}
```

The AI evaluates the file in implementation mode:

```json
{
  "name": "evaluate",
  "arguments": {
    "items": [
      {
        "id": "notice-node01",
        "mode": "implementation",
        "pack": "puppet-profile-contracts",
        "target": "notice-input",
        "format": "json",
        "source": {
          "type": "file",
          "path": ".cue-eval/evidence/notice-node01.json"
        }
      }
    ]
  }
}
```

Illustrative result excerpt:

```json
{
  "items": [
    {
      "id": "notice-node01",
      "outcome": "valid-concrete",
      "completeCheck": true,
      "gateEligible": false,
      "inputDigest": "sha256:<input-digest>",
      "contractDigest": "sha256:<contract-digest>",
      "scope": "puppet-profile-contracts:#NoticeInput",
      "provenance": {
        "subjectSource": ".cue-eval/evidence/notice-node01.json",
        "declaredReviewState": "candidate",
        "reviewVerified": false
      },
      "diagnostics": []
    }
  ]
}
```

This says the supplied effective value is concrete and satisfies the selected
contract. It does not prove that Puppet performed automatic parameter lookup,
that the class used the value, or that the managed file rendered correctly.
Those claims require Puppet-aware evidence and the repository's normal tests.

The same evaluation is available to CI through the CLI:

```sh
cue-eval-mcp evaluate --workspace . \
  --request .cue-eval/requests/notice-node01.json
```

## 6. Finish with the existing Puppet checks

**AI**

> The effective value satisfies the agreed contract. I will now run the
> repository's normal Puppet lint and rspec-puppet commands, verify lookup for
> representative node data, compile or compare the relevant catalog, and use
> the established no-op procedure before rollout. The contract result is
> complementary evidence, not a replacement for those checks.

The final change report should identify the input and contract digests, the
node/environment scope checked, the normal Puppet checks run, and anything that
was not evaluated.

## Executable validation

The snippets in this walkthrough were checked with Puppet 7.20.0 from the
Puppet-maintained
[`puppet/puppet-agent`](https://hub.docker.com/r/puppet/puppet-agent) image,
pinned to digest
`sha256:9223a40ef490ae331b41ff36edaaba3470caf02e984b4bf107c4184b99a32f0a`.
The disposable container had no network, a read-only root filesystem, and only
read-only fixture data plus temporary container-local state.

The checks established that:

- Puppet parser validation accepts the class shown above;
- automatic parameter lookup supplies the Hiera value during catalog
  compilation;
- the compiled `File[/etc/example-notice]` resource contains the supplied value
  followed by the assigned newline;
- missing data and a value longer than 64 characters fail catalog compilation;
- a companion control fixture uses its class-parameter default when Hiera is
  absent and uses the Hiera value when supplied;
- Puppet accepts a multiline value that is within its length bound, while the
  CUE single-line contract rejects the same value; and
- a local Puppet no-op completes without applying the resource.

The repository does not contain an rspec-puppet fixture for this fictional
class, so rspec-puppet was not run.

## Why this remains useful

The reusable value definition is now available to later contracts in the same
manifest-listed pack. A future change can reuse the same agreed meaning of
“single-line notice” instead of inventing another length limit or regular
expression. Operational knowledge becomes a repeatable check rather than a
detail that must be remembered in each review.

Several natural extensions already work with the current evaluator:

- add another required parameter to `#NoticeInput` and validate both effective
  values together;
- express a cross-field rule, such as requiring an expiry value only when the
  notice is enabled; or
- include environment in the evidence and apply different reviewed bounds for
  test and production data.

Other extensions require product development: automatic mapping from changed
files to contract packs, CUE package fetching or import resolution, Puppet
evidence collection, and independently verified review provenance. Explicitly
listing compatible local files in one pack already supports the pilot without
those features. They are not capabilities of the current MCP server.
