#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
feature_root="$(cd -- "${script_dir}/.." && pwd)"
repository_root="$(cd -- "${feature_root}/../.." && pwd)"
architecture="${1:-}"

if [[ -z "${architecture}" ]]; then
  case "$(go env GOARCH)" in
    amd64 | arm64)
      architecture="$(go env GOARCH)"
      ;;
    *)
      echo "usage: $0 <amd64|arm64>" >&2
      exit 2
      ;;
  esac
fi

case "${architecture}" in
  amd64 | arm64) ;;
  *)
    echo "usage: $0 <amd64|arm64>" >&2
    exit 2
    ;;
esac

artifact_dir="${feature_root}/src/cue-eval-mcp/artifacts"
artifact="${artifact_dir}/cue-eval-mcp-linux-${architecture}"
mkdir -p -- "${artifact_dir}"

(
  cd -- "${repository_root}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${architecture}" \
    go build -trimpath -o "${artifact}" ./cmd/cue-eval-mcp
)

(
  cd -- "${artifact_dir}"
  sha256sum -- "$(basename -- "${artifact}")" > "$(basename -- "${artifact}").sha256"
)

echo "staged ${artifact}"
