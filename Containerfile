# syntax=docker/dockerfile:1.7

ARG GO_IMAGE=golang:1.27.3-bookworm@sha256:5c3e9b43cfd3bb95c7937462e29d87ada57b481a96fa23ad0eb61778d7a00110
FROM ${GO_IMAGE} AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/cue-eval-mcp ./cmd/cue-eval-mcp

FROM scratch
COPY --from=build --chown=65532:65532 /out/cue-eval-mcp /usr/local/bin/cue-eval-mcp

USER 65532:65532
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/cue-eval-mcp"]
CMD ["serve"]
