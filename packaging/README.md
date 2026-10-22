# Packaging

The repository supports three ways to run the same `cue-eval-mcp` binary:

- a native Go build for local development and CI;
- a minimal OCI image for stdio MCP or CLI use; and
- a Dev Container Feature assembled from a locally built, checksummed binary.

No release artifacts or container images are published yet. The commands below
build from the checked-out source. The Dev Container Feature does not edit MCP
or agent configuration.

Native and staged-Feature builds require Go 1.25 or newer. Docker or another
compatible builder is required only for the OCI image. The standalone CUE CLI
is optional because the binary embeds the pinned CUE Go module.

## Native binary

```sh
go build -trimpath -o ./bin/cue-eval-mcp ./cmd/cue-eval-mcp
./bin/cue-eval-mcp --help
```

Use `cue-eval-mcp serve` for stdio MCP. The same binary provides `discover`,
`inspect`, and `evaluate` for local use and CI.

## OCI image

Build the image from the repository root:

```sh
docker build -f Containerfile -t cue-eval-mcp:local .
```

The default multi-architecture Go builder image is pinned by version and OCI
index digest. Override `GO_IMAGE` only as an explicit build decision.

When Docker is available, `./packaging/tests/container-smoke.sh` builds the
image, verifies its CLI, initializes MCP over stdio, and checks the advertised
tool list under the same read-only restrictions.

Run the stdio server with stdin attached, no TTY, no network, a read-only root
filesystem, and a read-only workspace mount:

```sh
docker run --rm -i \
  --network none \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --mount type=bind,src="$PWD",dst=/workspace,readonly \
  cue-eval-mcp:local
```

Do not use `-t`: MCP stdio is a framed protocol, not an interactive terminal.
The image exposes no port and runs as UID/GID `65532`.

Run a CLI command by replacing the image's default `serve` argument:

```sh
docker run --rm \
  --network none \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --mount type=bind,src="$PWD",dst=/workspace,readonly \
  cue-eval-mcp:local discover
```

## Dev Container Feature

The Feature source is under [`devcontainer-feature`](devcontainer-feature/).
It installs one binary at `/usr/local/bin/cue-eval-mcp`. It neither registers
the MCP server nor writes agent configuration.

Because there is no published release, stage a binary and checksum into the
Feature bundle before using a local Feature source:

```sh
./packaging/devcontainer-feature/scripts/stage-local-artifact.sh amd64
```

Use `arm64` for an ARM64 target. The staging script performs a static Linux
build from the current checkout and writes a SHA-256 companion file. The
generated binaries and checksums are ignored by Git.

During pre-release development, use the staged Feature source with the standard
Dev Container Feature test tooling. A regular `devcontainer.json` needs a
published OCI Feature reference, which does not exist yet.

The installer also accepts an `artifactPath` plus an exact `artifactSha256`.
That path must already exist inside the Feature installation container. This is
useful for controlled image builds and tests; it is not a reference to a host
workspace path.

A future published Feature can replace local staging with immutable release
artifacts and published checksums. No release URL is assumed here.
