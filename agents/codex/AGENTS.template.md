# Contract-first validation

> Copy and tailor this section in a consuming repository's root `AGENTS.md`
> after its project manifest and initial contract pack exist.

- Use contract modeling only when stable, observable data constraints make it
  useful. If the domain is too uncertain or available evidence cannot represent
  the important behavior, explain that limitation and use normal repository
  tests rather than inventing a schema.
- Choose routine configuration that fits the domain and repository conventions.
  Explain defaults that affect use and when to adjust them; do not ask the user
  to decide every mechanical detail. Ask only about consequential unresolved
  domain choices or ambiguity, and reuse recorded decisions.
- Before relevant edits, use task intent, repository structure, relevant
  examples, shared schemas, and the affected code to discover and inspect
  plausible contracts. Treat names, descriptions, and artifact labels as search
  hints, not authority.
- Explain why selected contracts appear applicable and ask focused questions
  when scope is ambiguous. Record exact contract identifiers, content digests,
  any contract-defined versions, the evidence boundary, and known coverage gaps.
  Do not claim exhaustive impact or dependency coverage from discovery alone.
- Treat each pack's manifest `sources` as its explicit local schema dependency
  list. If the user supplied it, use it as authoritative source selection after
  validation; do not silently augment or rewrite it. Otherwise inspect the
  repository best-effort, propose a workspace-local mapping with rationale and
  uncertainties, ask whether it is good enough to start, apply requested
  adjustments, and persist the confirmed mapping. Do not imply exhaustive
  analysis or that the evaluator fetches packages, resolves versions, or follows
  imports.
- Collaborate with the user on missing change bounds in author mode. Persist the
  agreed change contract with references to reviewed shared definitions.
- Inspect native code, resolution semantics, and tests to propose required,
  optional, native-default, and null behavior in natural technical language.
  Confirm consequential uncertainty unless the user already answered it, and
  record the agreed rationale for reuse. Do not assume nullable means optional.
- Validate raw fragments only against a separately designed partial-input
  contract. Never present partial compatibility as complete effective evidence.
- Strict effective validation must not silently add absent evidence from CUE
  defaults, constants, or derivations. Use JSON or YAML for implementation
  evidence; the evaluator rejects implementation-mode CUE because it cannot yet
  provide the same evidence-presence guarantee. CUE-driven completion and
  nested completion overrides are deferred.
- Treat `compatible-so-far` as incomplete and never as a gate pass.
- After meaningful candidate changes, evaluate actual adapter-derived outputs
  in implementation mode against the pinned contract.
- Repair the implementation or report missing evidence. Never silently weaken,
  replace, or reclassify a reviewed baseline.
- Review contract revisions separately from implementation changes.
- Before finishing, run the repository's required tests and authoritative
  evaluation, then report exact input and contract revisions, checked scope,
  diagnostics, coverage, and anything unavailable or unchecked.
- Keep secrets and unrestricted production data out of model-visible inputs,
  diagnostics, and reports.
- Treat `reviewVerified: false` and `gateEligible: false` literally. Declared
  review metadata and a digest do not establish approval.
- Use the consuming repository's existing review process. Do not introduce a
  separate mandatory approval workflow for these contracts.
