# Dev Container Feature

This directory contains an offline-first Dev Container Feature for the
pre-release `cue-eval-mcp` binary.

Its `src/<feature-id>/devcontainer-feature.json` and `install.sh` layout follows
the official [Dev Container Features specification](https://containers.dev/implementors/features/).

## Local preparation

From the repository root, stage the target architecture:

```sh
./packaging/devcontainer-feature/scripts/stage-local-artifact.sh amd64
```

The Feature installer selects `cue-eval-mcp-linux-amd64` or
`cue-eval-mcp-linux-arm64` from `src/cue-eval-mcp/artifacts` and verifies its
SHA-256 companion file before installation.

The staged source tree is intended for the standard Dev Container Feature
development and test tooling. It is not referenced by a fabricated registry
identifier. A normal `devcontainer.json` consumer needs an actual published OCI
Feature reference; publishing is outside this repository's current scope.

Alternatively, set both Feature options:

- `artifactPath`: a binary path already accessible inside the Feature install
  container; and
- `artifactSha256`: the exact lowercase SHA-256 digest of that file.

The installer refuses unsupported architectures, malformed checksums, missing
artifacts, checksum mismatches, and replacement of a different existing binary.
It does not download software, modify agent configuration, or register MCP.

## Checks

```sh
sh -n src/cue-eval-mcp/install.sh
./tests/install-script.sh
```

The standard Dev Container Feature smoke test is in
[`test/cue-eval-mcp/test.sh`](test/cue-eval-mcp/test.sh). Run it through the
Dev Container CLI after staging the target artifact when that CLI and a
container runtime are available.
